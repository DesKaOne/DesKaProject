package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"indochain/internal/serviceagent"
	"indochain/internal/version"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-version" || args[0] == "version") {
		fmt.Fprint(out, version.String("indoservice"))
		return nil
	}
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
		printServiceHelp(out)
		return nil
	}
	if len(args) > 0 && args[0] == "status" {
		return runStatus(args[1:], out)
	}
	opts := serviceagent.DefaultOptions()
	opts.Platform = runtime.GOOS + "/" + runtime.GOARCH
	fs := flag.NewFlagSet("indoservice", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.RPCURL, "rpc-url", "", "IndoChain node RPC URL")
	fs.StringVar(&opts.Address, "address", "", "service owner iND address")
	fs.StringVar(&opts.Endpoint, "endpoint", "", "optional advertised endpoint")
	fs.StringVar(&opts.StatePath, "state", opts.StatePath, "agent state JSON path")
	fs.DurationVar(&opts.HeartbeatInterval, "heartbeat-interval", opts.HeartbeatInterval, "heartbeat interval")
	fs.DurationVar(&opts.ChallengeInterval, "challenge-interval", opts.ChallengeInterval, "challenge interval")
	fs.DurationVar(&opts.ScoreInterval, "score-interval", opts.ScoreInterval, "score interval")
	fs.DurationVar(&opts.RetryInterval, "retry-interval", opts.RetryInterval, "retry interval")
	fs.BoolVar(&opts.Once, "once", false, "run one register/heartbeat/challenge cycle and exit")
	fs.BoolVar(&opts.SafeMode, "safe-mode", true, "safe simulation mode")
	fs.Int64Var(&opts.MaxBytesPerChallenge, "max-bytes-per-challenge", opts.MaxBytesPerChallenge, "maximum simulated bytes per challenge")
	fs.StringVar(&opts.ClientVersion, "client-version", opts.ClientVersion, "client version")
	fs.StringVar(&opts.Platform, "platform", opts.Platform, "platform metadata")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if opts.RPCURL == "" {
		return errors.New("rpc-url is required")
	}
	if opts.Address == "" {
		return errors.New("address is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	agent := serviceagent.Agent{
		Options: opts,
		Client:  serviceagent.NewClient(opts.RPCURL),
		Out:     out,
	}
	return agent.Run(ctx)
}

func printServiceHelp(out io.Writer) {
	fmt.Fprintln(out, "IndoChain Service Agent")
	fmt.Fprintln(out, "usage: indoservice --rpc-url <url> --address <IND_ADDR> [--endpoint <url>] [--once]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "commands:")
	fmt.Fprintln(out, "  status")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "options:")
	fmt.Fprintln(out, "  --version")
	fmt.Fprintln(out, "  --help")
	fmt.Fprintln(out, "  --rpc-url <url>")
	fmt.Fprintln(out, "  --address <IND_ADDR>")
	fmt.Fprintln(out, "  --endpoint <url>")
	fmt.Fprintln(out, "mainnet: not available")
}

func runStatus(args []string, out io.Writer) error {
	path := "./indoservice-state.json"
	fs := flag.NewFlagSet("indoservice status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&path, "state", path, "agent state JSON path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	state, err := serviceagent.LoadState(path)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "address: %s\n", state.Address)
	fmt.Fprintf(out, "service node id: %s\n", state.ServiceNodeID)
	fmt.Fprintf(out, "endpoint: %s\n", state.Endpoint)
	fmt.Fprintf(out, "rpc: %s\n", state.RPCURL)
	fmt.Fprintf(out, "last heartbeat: %d\n", state.LastHeartbeatAt)
	fmt.Fprintf(out, "last score: %d\n", state.LastScore)
	fmt.Fprintf(out, "simulated points: %d\n", state.LastSimulatedPoints)
	fmt.Fprintf(out, "successful challenges: %d\n", state.SuccessfulChallenges)
	fmt.Fprintf(out, "failed challenges: %d\n", state.FailedChallenges)
	fmt.Fprintf(out, "total simulated bytes up: %d\n", state.TotalSimulatedBytesUp)
	fmt.Fprintf(out, "total simulated bytes down: %d\n", state.TotalSimulatedBytesDown)
	return nil
}
