package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/btcsuite/btcutil"
	"github.com/lightninglabs/llm"
	"github.com/lightninglabs/llm/clmrpc"
	"github.com/lightninglabs/protobuf-hex-display/jsonpb"
	"github.com/lightninglabs/protobuf-hex-display/proto"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
)

type invalidUsageError struct {
	ctx     *cli.Context
	command string
}

func (e *invalidUsageError) Error() string {
	return fmt.Sprintf("invalid usage of command %s", e.command)
}

func printJSON(resp interface{}) {
	b, err := json.Marshal(resp)
	if err != nil {
		fatal(err)
	}

	var out bytes.Buffer
	_ = json.Indent(&out, b, "", "\t")
	out.WriteString("\n")
	_, _ = out.WriteTo(os.Stdout)
}

func printRespJSON(resp proto.Message) { // nolint
	jsonMarshaler := &jsonpb.Marshaler{
		EmitDefaults: true,
		OrigName:     true,
		Indent:       "\t", // Matches indentation of printJSON.
	}

	jsonStr, err := jsonMarshaler.MarshalToString(resp)
	if err != nil {
		fmt.Println("unable to decode response: ", err)
		return
	}

	fmt.Println(jsonStr)
}

func fatal(err error) {
	var e *invalidUsageError
	if errors.As(err, &e) {
		_ = cli.ShowCommandHelp(e.ctx, e.command)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "[llm] %v\n", err)
	}
	os.Exit(1)
}

func main() {
	app := cli.NewApp()

	app.Version = llm.Version()
	app.Name = "llm"
	app.Usage = "control plane for your llmd"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "rpcserver",
			Value: "localhost:12010",
			Usage: "llmd daemon address host:port",
		},
	}
	app.Commands = append(app.Commands, accountsCommands...)
	app.Commands = append(app.Commands, ordersCommands...)
	app.Commands = append(app.Commands, auctionCommands...)

	err := app.Run(os.Args)
	if err != nil {
		fatal(err)
	}
}

func getClient(ctx *cli.Context) (clmrpc.TraderClient, func(),
	error) {

	rpcServer := ctx.GlobalString("rpcserver")
	conn, err := getClientConn(rpcServer)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close() }

	traderClient := clmrpc.NewTraderClient(conn)
	return traderClient, cleanup, nil
}

func parseAmt(text string) (btcutil.Amount, error) {
	amtInt64, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amt value: %v", err)
	}
	return btcutil.Amount(amtInt64), nil
}

func getClientConn(address string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to RPC server: %v",
			err)
	}

	return conn, nil
}

func parseStr(ctx *cli.Context, argIdx int, flag, cmd string) (string, error) {
	var str string
	switch {
	case ctx.IsSet(flag):
		str = ctx.String(flag)
	case ctx.Args().Get(argIdx) != "":
		str = ctx.Args().Get(argIdx)
	default:
		return "", &invalidUsageError{ctx, cmd}
	}
	return str, nil
}

func parseHexStr(ctx *cli.Context, argIdx int, flag, cmd string) ([]byte, error) {
	hexStr, err := parseStr(ctx, argIdx, flag, cmd)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(hexStr)
}

func parseUint64(ctx *cli.Context, argIdx int, flag, cmd string) (uint64, error) {
	str, err := parseStr(ctx, argIdx, flag, cmd)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(str, 10, 64)
}