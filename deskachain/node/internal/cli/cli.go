package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"deskachain/internal/amount"
	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/crypto"
	"deskachain/internal/ledger"
	"deskachain/internal/mempool"
	"deskachain/internal/mining"
	"deskachain/internal/nodestate"
	"deskachain/internal/p2p"
	"deskachain/internal/rpc"
	"deskachain/internal/servicenode"
	"deskachain/internal/staking"
	"deskachain/internal/storage"
	"deskachain/internal/types"
	"deskachain/internal/version"
	"deskachain/internal/wallet"
)

type App struct {
	out        io.Writer
	paths      config.Paths
	profile    config.NetworkConfig
	rpcURL     string
	ignoreLock bool
	chainMu    *sync.Mutex
}

type multiStringFlag []string

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func New(out io.Writer) App {
	return App{out: out, paths: config.NewPaths(config.DefaultDataDir), profile: config.Localnet()}
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, "DesKaChain")
	fmt.Fprintln(out, "usage: deskachain [--datadir DIR] [--network localnet|testnet] <command> [args]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "commands:")
	fmt.Fprintln(out, "  version")
	fmt.Fprintln(out, "  init")
	fmt.Fprintln(out, "  node start")
	fmt.Fprintln(out, "  chain info|validate|difficulty")
	fmt.Fprintln(out, "  mining status|difficulty|blocks")
	fmt.Fprintln(out, "  wallet new|list")
	fmt.Fprintln(out, "  peer list|sync|check")
	fmt.Fprintln(out, "  upstream list|status|push|push-all")
	fmt.Fprintln(out, "  service register|score")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "mainnet: not available")
}

func (a App) WithDataDir(datadir string) App {
	a.paths = config.NewPaths(datadir)
	return a
}

func (a App) WithProfile(profile config.NetworkConfig) App {
	a.profile = profile
	return a
}

func (a App) Run(args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		printHelp(a.out)
		return nil
	}
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-version") {
		fmt.Fprint(a.out, version.String("DesKaChain"))
		return nil
	}
	fs := flag.NewFlagSet("deskachain", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	datadir := fs.String("datadir", a.paths.DataDir, "data directory")
	rpcURL := fs.String("rpc-url", "", "remote RPC URL")
	ignoreLock := fs.Bool("ignore-lock", false, "ignore datadir runtime lock")
	networkName := fs.String("network", "localnet", "network name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	networkFlagProvided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "network" {
			networkFlagProvided = true
		}
	})
	selectedNetwork, err := config.NetworkByName(*networkName)
	if err != nil {
		return err
	}
	network := selectedNetwork
	if network.Name == "mainnet" && !networkFlagProvided {
		return errors.New("mainnet requires explicit --network mainnet selection")
	}
	a.paths = config.NewPaths(*datadir)
	args = fs.Args()
	if len(args) == 0 {
		return errors.New("command is required")
	}
	if args[0] == "version" {
		fmt.Fprint(a.out, version.String("DesKaChain"))
		return nil
	}
	if args[0] == "help" {
		printHelp(a.out)
		return nil
	}

	// Mainnet is a launch-sensitive network: a caller must explicitly select it,
	// and a test-only injected profile must never be able to masquerade as mainnet.
	if network.Name == "mainnet" {
		if !networkFlagProvided {
			return errors.New("mainnet requires explicit --network mainnet selection")
		}
		if err := config.ValidateNetworkProfile(network); err != nil {
			return err
		}
	}

	skipMetadataCheck := len(args) >= 2 && args[0] == "dev" && args[1] == "reset"
	if !skipMetadataCheck {
		if metadata, ok, err := config.ReadNetworkMetadata(a.paths); err != nil {
			return err
		} else if ok {
			metadataNetwork, err := config.NetworkByName(metadata.Network)
			if err != nil {
				return err
			}
			if metadata.NetworkID != metadataNetwork.NetworkID || metadata.ChainID != metadataNetwork.ChainID {
				return fmt.Errorf("datadir network metadata mismatch: %s chain_id=%d network_id=%s", metadata.Network, metadata.ChainID, metadata.NetworkID)
			}
			if networkFlagProvided {
				if network.Name != metadataNetwork.Name || network.NetworkID != metadata.NetworkID || network.ChainID != metadataNetwork.ChainID {
					return fmt.Errorf("datadir initialized for %s, cannot start as %s", metadata.Network, network.Name)
				}
			} else {
				network = metadataNetwork
			}
		}
	}

	// Preserve injected economics for non-mainnet integration tests only. Mainnet
	// always uses the canonical built-in profile above.
	if !networkFlagProvided && network.Name != "mainnet" && a.profile.Name == network.Name {
		network = a.profile
	}
	if network.Name != "mainnet" {
		network.GenesisHash = chain.GenesisBlockForNetwork(network).Hash
	}
	a.profile = network
	a.rpcURL = strings.TrimRight(*rpcURL, "/")
	a.ignoreLock = *ignoreLock
	if a.rpcURL != "" {
		return a.runRemote(args)
	}
	if err := a.checkLock(args); err != nil {
		return err
	}
	switch args[0] {
	case "version":
		fmt.Fprint(a.out, version.String("DesKaChain"))
		return nil
	case "network":
		return a.network(args[1:])
	case "init":
		return a.init()
	case "dev":
		return a.dev(args[1:])
	case "node":
		return a.node(args[1:])
	case "peer":
		return a.peer(args[1:])
	case "upstream":
		return a.upstream(args[1:])
	case "p2p":
		return a.p2p(args[1:])
	case "fork":
		return a.fork(args[1:])
	case "reorg":
		return a.reorg(args[1:])
	case "debug":
		return a.debug(args[1:])
	case "wallet":
		return a.wallet(args[1:])
	case "address":
		return a.address(args[1:])
	case "balance":
		return a.balance(args[1:])
	case "send":
		return a.send(args[1:])
	case "faucet":
		return a.faucet(args[1:])
	case "tx":
		return a.tx(args[1:])
	case "mempool":
		return a.mempool(args[1:])
	case "mine":
		return a.mine(args[1:])
	case "mining":
		return a.mining(args[1:])
	case "service":
		return a.service(args[1:])
	case "stake":
		return a.stake(args[1:])
	case "chain":
		return a.chain(args[1:])
	case "rpc":
		return a.rpc(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}