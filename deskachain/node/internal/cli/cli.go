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

func (a App) network(args []string) error {
	if len(args) != 1 || args[0] != "info" {
		return errors.New("usage: network info")
	}
	net := a.profile
	if net.Name != "mainnet" {
		net.GenesisHash = chain.GenesisBlockForNetwork(net).Hash
	}

	fmt.Fprintf(a.out, "network name: %s\n", net.NetworkName)
	fmt.Fprintf(a.out, "network id: %s\n", net.NetworkID)
	fmt.Fprintf(a.out, "chain id: %d\n", net.ChainID)
	fmt.Fprintf(a.out, "protocol version: %d\n", net.ProtocolVersion)
	fmt.Fprintf(a.out, "rpc api version: %s\n", net.RPCAPIVersion)
	fmt.Fprintf(a.out, "p2p protocol version: %s\n", net.P2PProtocolVersion)
	fmt.Fprintf(a.out, "block version: %d\n", net.BlockVersion)
	fmt.Fprintf(a.out, "tx version: %d\n", net.TxVersion)
	fmt.Fprintf(a.out, "min protocol version: %d\n", net.MinProtocolVersion)
	fmt.Fprintf(a.out, "genesis hash: %s\n", net.GenesisHash)
	return nil
}

func (a App) init() error {
	bc, closeFn, err := a.openChain()
	if err != nil {
		return err
	}
	defer closeFn()
	has, err := bc.HasChain()
	if err != nil {
		return err
	}
	if !has {
		if err := bc.InitWithProfile(a.profile); err != nil {
			return err
		}
		tip, err := bc.Tip()
		if err != nil {
			return err
		}
		if err := config.WriteNetworkMetadata(a.paths, a.profile, tip.Hash); err != nil {
			return err
		}
		fmt.Fprintln(a.out, "chain initialized")
		fmt.Fprintf(a.out, "genesis hash: %s\n", tip.Hash)
		fmt.Fprintf(a.out, "network: %s\n", a.profile.Name)
		fmt.Fprintf(a.out, "chain id: %d\n", a.profile.ChainID)
		fmt.Fprintf(a.out, "datadir: %s\n", a.paths.DataDir)
		return nil
	}
	tip, err := bc.Tip()
	if err != nil {
		return err
	}
	if err := config.EnsureNetworkMatches(a.paths, a.profile); err != nil {
		return err
	}
	if _, ok, err := config.ReadNetworkMetadata(a.paths); err != nil {
		return err
	} else if !ok && a.profile.Name == "localnet" {
		if err := config.WriteNetworkMetadata(a.paths, a.profile, tip.Hash); err != nil {
			return err
		}
	}
	fmt.Fprintln(a.out, "chain already initialized")
	fmt.Fprintf(a.out, "height: %d\n", tip.Height)
	fmt.Fprintf(a.out, "tip hash: %s\n", tip.Hash)
	fmt.Fprintf(a.out, "network: %s\n", a.profile.Name)
	fmt.Fprintf(a.out, "chain id: %d\n", a.profile.ChainID)
	fmt.Fprintf(a.out, "datadir: %s\n", a.paths.DataDir)
	return nil
}

func (a App) dev(args []string) error {
	if len(args) == 0 {
		return errors.New("dev command is required")
	}
	switch args[0] {
	case "reset":
		fs := flag.NewFlagSet("dev reset", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		yes := fs.Bool("yes", false, "confirm reset")
		force := fs.Bool("force", false, "force reset locked datadir")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if _, locked := p2p.IsLocked(a.paths.Lock); locked && !*force {
			return a.lockedError(args)
		}
		if !*yes {
			fmt.Fprintf(a.out, "This will delete local DesKaChain dev data under %s.\n", a.paths.DataDir)
			fmt.Fprintln(a.out, "Re-run with --yes to confirm.")
			return nil
		}
		if err := safeDataDir(a.paths.DataDir); err != nil {
			return err
		}
		if _, err := os.Stat(a.paths.DataDir); errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(a.out, "datadir does not exist, nothing to reset")
			return nil
		} else if err != nil {
			return err
		}
		if err := os.RemoveAll(a.paths.DataDir); err != nil {
			return err
		}
		fmt.Fprintf(a.out, "deleted %s\n", a.paths.DataDir)
		fmt.Fprintln(a.out, "dev data reset complete")
		return nil
	case "inspect":
		return a.devInspect()
	case "fork-sim":
		return a.devForkSim(args[1:])
	case "reorg-tx-sim":
		return a.devReorgTxSim(args[1:])
	default:
		return fmt.Errorf("unknown dev command %q", args[0])
	}
}

func (a App) devForkSim(args []string) error {
	fs := flag.NewFlagSet("dev fork-sim", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	datadirA := fs.String("datadir-a", "", "chain A datadir")
	datadirB := fs.String("datadir-b", "", "chain B datadir")
	blocksA := fs.Uint64("blocks-a", 3, "blocks to mine on chain A")
	blocksB := fs.Uint64("blocks-b", 3, "blocks to mine on chain B")
	minerA := fs.String("miner-a", "", "optional miner A address")
	minerB := fs.String("miner-b", "", "optional miner B address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *datadirA == "" || *datadirB == "" {
		return errors.New("usage: dev fork-sim --datadir-a <path> --datadir-b <path> --blocks-a <n> --blocks-b <n>")
	}
	summaryA, err := runForkSimChain(*datadirA, *minerA, *blocksA)
	if err != nil {
		return err
	}
	summaryB, err := runForkSimChain(*datadirB, *minerB, *blocksB)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.out, "fork simulation complete")
	printForkSimSummary(a.out, "chain A", summaryA)
	printForkSimSummary(a.out, "chain B", summaryB)
	fmt.Fprintln(a.out, "common ancestor expected: height 0")
	if summaryA.GenesisHash != summaryB.GenesisHash {
		return errors.New("fork simulation failed: genesis hashes differ")
	}
	if *blocksA > 0 && *blocksB > 0 && summaryA.Height1Hash == summaryB.Height1Hash {
		return errors.New("fork simulation failed: height 1 hashes are equal")
	}
	return nil
}

func (a App) devReorgTxSim(args []string) error {
	fs := flag.NewFlagSet("dev reorg-tx-sim", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	datadirA := fs.String("datadir-a", "", "local chain datadir")
	datadirB := fs.String("datadir-b", "", "peer chain datadir")
	scenario := fs.String("scenario", "requeue-valid", "scenario name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *datadirA == "" || *datadirB == "" {
		return errors.New("usage: dev reorg-tx-sim --datadir-a <path> --datadir-b <path> --scenario <name>")
	}
	if *scenario == "invalid-after-reorg" {
		*scenario = "conflict-double-spend"
	}
	summary, err := runReorgTxSim(*datadirA, *datadirB, *scenario)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.out, "reorg tx simulation complete")
	fmt.Fprintf(a.out, "scenario: %s\n", *scenario)
	fmt.Fprintf(a.out, "chain A height: %d\n", summary.HeightA)
	fmt.Fprintf(a.out, "chain B height: %d\n", summary.HeightB)
	fmt.Fprintf(a.out, "miner common: %s\n", summary.MinerCommon)
	fmt.Fprintf(a.out, "user1: %s\n", summary.User1)
	fmt.Fprintf(a.out, "user2: %s\n", summary.User2)
	fmt.Fprintf(a.out, "orphan tx: %s\n", summary.OrphanTxID)
	if summary.PeerTxID != "" {
		fmt.Fprintf(a.out, "peer tx: %s\n", summary.PeerTxID)
	}
	fmt.Fprintln(a.out, "start both nodes and run reorg preview/apply from chain A to chain B")
	return nil
}

type reorgTxSimSummary struct {
	HeightA     uint64
	HeightB     uint64
	MinerCommon string
	MinerB      string
	User1       string
	User2       string
	OrphanTxID  string
	PeerTxID    string
}

func runReorgTxSim(datadirA, datadirB, scenario string) (reorgTxSimSummary, error) {
	pathsA := config.NewPaths(datadirA)
	pathsB := config.NewPaths(datadirB)
	if err := safeDataDir(pathsA.DataDir); err != nil {
		return reorgTxSimSummary{}, err
	}
	if err := safeDataDir(pathsB.DataDir); err != nil {
		return reorgTxSimSummary{}, err
	}
	_ = os.RemoveAll(pathsA.DataDir)
	_ = os.RemoveAll(pathsB.DataDir)
	var out bytes.Buffer
	appA := New(&out).WithDataDir(pathsA.DataDir)
	if err := appA.Run([]string{"init"}); err != nil {
		return reorgTxSimSummary{}, err
	}
	storeA := wallet.NewStore(pathsA.Wallets)
	minerCommon, _ := wallet.New()
	minerB, _ := wallet.New()
	user1, _ := wallet.New()
	user2, _ := wallet.New()
	for _, w := range []wallet.Wallet{minerCommon, minerB, user1, user2} {
		if err := storeA.Add(w); err != nil {
			return reorgTxSimSummary{}, err
		}
	}
	if err := appA.Run([]string{"mine", "--address", minerCommon.Address, "--blocks", "3"}); err != nil {
		return reorgTxSimSummary{}, err
	}
	if err := copyDir(pathsA.DataDir, pathsB.DataDir); err != nil {
		return reorgTxSimSummary{}, err
	}
	appB := New(&out).WithDataDir(pathsB.DataDir)
	orphanTx, err := appA.createPendingTransaction(minerCommon.Address, user1.Address, "10")
	if err != nil {
		return reorgTxSimSummary{}, err
	}
	if scenario == "conflict-double-spend" {
		orphanTx, err = appA.createPendingTransaction(minerCommon.Address, user1.Address, "100")
		if err != nil {
			return reorgTxSimSummary{}, err
		}
	}
	if err := mempool.New(pathsA.Mempool).Add(orphanTx); err != nil {
		return reorgTxSimSummary{}, err
	}
	if err := appA.Run([]string{"mine", "--address", minerCommon.Address, "--blocks", "1"}); err != nil {
		return reorgTxSimSummary{}, err
	}
	var peerTx types.Transaction
	switch scenario {
	case "requeue-valid":
	case "confirmed-on-peer", "mempool-removes-confirmed":
		peerTx = orphanTx
		_ = mempool.New(pathsB.Mempool).Add(peerTx)
	case "conflict-double-spend":
		peerTx, err = appB.createPendingTransaction(minerCommon.Address, user2.Address, "120")
		if err != nil {
			return reorgTxSimSummary{}, err
		}
		if err := mempool.New(pathsB.Mempool).Add(peerTx); err != nil {
			return reorgTxSimSummary{}, err
		}
	default:
		return reorgTxSimSummary{}, fmt.Errorf("unknown scenario %q", scenario)
	}
	if err := appB.Run([]string{"mine", "--address", minerB.Address, "--blocks", "3"}); err != nil {
		return reorgTxSimSummary{}, err
	}
	bcA, closeA, err := appA.openInitializedChain()
	if err != nil {
		return reorgTxSimSummary{}, err
	}
	defer closeA()
	bcB, closeB, err := appB.openInitializedChain()
	if err != nil {
		return reorgTxSimSummary{}, err
	}
	defer closeB()
	tipA, _ := bcA.Tip()
	tipB, _ := bcB.Tip()
	return reorgTxSimSummary{HeightA: tipA.Height, HeightB: tipB.Height, MinerCommon: minerCommon.Address, MinerB: minerB.Address, User1: user1.Address, User2: user2.Address, OrphanTxID: orphanTx.ID, PeerTxID: peerTx.ID}, nil
}

func copyDir(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func (a App) devInspect() error {
	peers, peerErr := p2p.NewPeerStore(a.paths.Peers).LoadMetadata()
	wallets, walletErr := wallet.NewStore(a.paths.Wallets).Load()
	height := uint64(0)
	if fileExists(a.paths.DB) {
		bc, closeFn, err := a.openChain()
		if err == nil {
			if tip, tipErr := bc.Tip(); tipErr == nil {
				height = tip.Height
			}
			closeFn()
		}
	}
	fmt.Fprintf(a.out, "datadir: %s\n", a.paths.DataDir)
	fmt.Fprintf(a.out, "chain db: %s\n", fileState(a.paths.DB))
	fmt.Fprintf(a.out, "wallets: %s\n", fileState(a.paths.Wallets))
	fmt.Fprintf(a.out, "peers.json: %s\n", fileState(a.paths.Peers))
	fmt.Fprintf(a.out, "node_id: %s\n", fileState(a.paths.NodeID))
	fmt.Fprintf(a.out, "node.lock: %s\n", fileState(a.paths.Lock))
	fmt.Fprintf(a.out, "mempool: %s\n", fileState(a.paths.Mempool))
	fmt.Fprintf(a.out, "height: %d\n", height)
	if peerErr != nil {
		fmt.Fprintf(a.out, "peers error: %s\n", peerErr)
		fmt.Fprintln(a.out, "peers: 0")
	} else {
		fmt.Fprintf(a.out, "peers: %d\n", len(peers))
	}
	if walletErr != nil {
		fmt.Fprintf(a.out, "wallets error: %s\n", walletErr)
		fmt.Fprintln(a.out, "wallet count: 0")
	} else {
		fmt.Fprintf(a.out, "wallet count: %d\n", len(wallets))
	}
	return nil
}

type forkSimSummary struct {
	DataDir     string
	Height      uint64
	TipHash     string
	Miner       string
	GenesisHash string
	Height1Hash string
}

func runForkSimChain(datadir, miner string, blocks uint64) (forkSimSummary, error) {
	paths := config.NewPaths(datadir)
	if err := safeDataDir(paths.DataDir); err != nil {
		return forkSimSummary{}, err
	}
	if err := os.RemoveAll(paths.DataDir); err != nil {
		return forkSimSummary{}, err
	}
	var out bytes.Buffer
	app := New(&out).WithDataDir(paths.DataDir)
	if err := app.Run([]string{"init"}); err != nil {
		return forkSimSummary{}, err
	}
	if miner == "" {
		w, err := wallet.New()
		if err != nil {
			return forkSimSummary{}, err
		}
		if err := wallet.NewStore(paths.Wallets).Add(w); err != nil {
			return forkSimSummary{}, err
		}
		miner = w.Address
	} else if err := crypto.ValidateAddressForNetwork(miner, config.Localnet()); err != nil {
		return forkSimSummary{}, err
	}
	if blocks > 0 {
		out.Reset()
		if err := app.Run([]string{"mine", "--address", miner, "--blocks", fmt.Sprint(blocks)}); err != nil {
			return forkSimSummary{}, err
		}
	}
	bc, closeFn, err := app.openInitializedChain()
	if err != nil {
		return forkSimSummary{}, err
	}
	defer closeFn()
	all, err := bc.Blocks()
	if err != nil {
		return forkSimSummary{}, err
	}
	tip := all[len(all)-1]
	summary := forkSimSummary{
		DataDir:     paths.DataDir,
		Height:      tip.Height,
		TipHash:     tip.Hash,
		Miner:       miner,
		GenesisHash: all[0].Hash,
	}
	if len(all) > 1 {
		summary.Height1Hash = all[1].Hash
	}
	return summary, nil
}

func printForkSimSummary(out io.Writer, label string, summary forkSimSummary) {
	fmt.Fprintf(out, "%s:\n", label)
	fmt.Fprintf(out, "datadir: %s\n", summary.DataDir)
	fmt.Fprintf(out, "height: %d\n", summary.Height)
	fmt.Fprintf(out, "tip: %s\n", summary.TipHash)
	fmt.Fprintf(out, "miner: %s\n", summary.Miner)
}

func (a App) wallet(args []string) error {
	if len(args) == 0 {
		return errors.New("wallet command is required")
	}
	store := wallet.NewStore(a.paths.Wallets)
	switch args[0] {
	case "new":
		w, err := wallet.NewWithProfile(a.profile)
		if err != nil {
			return err
		}
		if err := store.Add(w); err != nil {
			return err
		}
		fmt.Fprintln(a.out, w.Address)
		return nil
	case "list":
		wallets, err := store.Load()
		if err != nil {
			return err
		}
		for _, w := range wallets {
			format := "base58check"
			if crypto.IsLegacyDevAddress(w.Address) {
				format = "legacy-dev"
			}
			fmt.Fprintf(a.out, "address=%s format=%s network=%s\n", w.Address, format, a.profile.Name)
		}
		return nil
	case "export":
		fs := flag.NewFlagSet("wallet export", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "wallet address")
		show := fs.Bool("show-private-key", false, "print private key")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if !*show {
			fmt.Fprintln(a.out, "Refusing to print private key without --show-private-key.")
			fmt.Fprintln(a.out, "Phase 1 wallet storage is for development only.")
			return nil
		}
		w, ok, err := store.Find(*address)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("wallet not found locally")
		}
		fmt.Fprintf(a.out, "address: %s\n", w.Address)
		fmt.Fprintf(a.out, "format: %s\n", addressFormat(w.Address))
		fmt.Fprintf(a.out, "network: %s\n", a.profile.Name)
		fmt.Fprintln(a.out, "key curve: secp256k1")
		if crypto.IsLegacyDevAddress(w.Address) {
			fmt.Fprintln(a.out, "warning: legacy idr1 dev address; localnet compatibility only")
		}
		fmt.Fprintf(a.out, "private_key: %s\n", w.PrivateKeyHex)
		return nil
	case "import":
		fs := flag.NewFlagSet("wallet import", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		privateKey := fs.String("private-key", "", "raw 32-byte private key hex")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		w, err := wallet.FromPrivateKeyHex(*privateKey, a.profile)
		if err != nil {
			return err
		}
		if err := store.Add(w); err != nil {
			return err
		}
		fmt.Fprintf(a.out, "address: %s\n", w.Address)
		fmt.Fprintln(a.out, "format: base58check")
		fmt.Fprintf(a.out, "network: %s\n", a.profile.Name)
		fmt.Fprintln(a.out, "key curve: secp256k1")
		return nil
	case "inspect":
		fs := flag.NewFlagSet("wallet inspect", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "wallet address")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *address == "" && fs.NArg() == 1 {
			*address = fs.Arg(0)
		}
		return a.printAddressInspection(*address)
	default:
		return fmt.Errorf("unknown wallet command %q", args[0])
	}
}

func (a App) address(args []string) error {
	if len(args) != 2 || args[0] != "validate" {
		return errors.New("usage: address validate <address>")
	}
	if err := crypto.ValidateAddressForNetwork(args[1], a.profile); err != nil {
		fmt.Fprintln(a.out, "address invalid")
		return nil
	}
	fmt.Fprintln(a.out, "address valid")
	return nil
}

func (a App) balance(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: balance <address>")
	}
	if err := crypto.ValidateAddressForNetwork(args[0], a.profile); err != nil {
		return err
	}
	bc, closeFn, err := a.openInitializedChain()
	if err != nil {
		return err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return err
	}
	pending, err := mempool.New(a.paths.Mempool).Load()
	if err != nil {
		return err
	}
	details, err := ledger.BalanceDetailsForWithProfile(args[0], blocks, pending, consensusParams(a.profile), a.profile)
	if err != nil {
		return err
	}
	printBalanceDetails(a.out, details)
	return nil
}

func (a App) send(args []string) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	from := fs.String("from", "", "sender address")
	to := fs.String("to", "", "recipient address")
	amountText := fs.String("amount", "", "amount")
	if err := fs.Parse(args); err != nil {
		return err
	}
	tx, err := a.createPendingTransaction(*from, *to, *amountText)
	if err != nil {
		return fmt.Errorf("send failed: %w", err)
	}
	if err := mempool.New(a.paths.Mempool).Add(tx); err != nil {
		return err
	}
	a.broadcastTx(tx)
	fmt.Fprintln(a.out, "tx created")
	fmt.Fprintf(a.out, "id: %s\n", tx.ID)
	fmt.Fprintf(a.out, "from: %s\n", tx.From)
	fmt.Fprintf(a.out, "to: %s\n", tx.To)
	fmt.Fprintf(a.out, "amount: %s %s\n", amount.Format(tx.Amount), config.Ticker)
	fmt.Fprintf(a.out, "fee: %s %s\n", amount.Format(tx.Fee), config.Ticker)
	fmt.Fprintf(a.out, "nonce: %d\n", tx.Nonce)
	fmt.Fprintln(a.out, "status: pending")
	return nil
}

func (a App) faucet(args []string) error {
	if len(args) == 0 {
		return errors.New("faucet command is required")
	}
	switch args[0] {
	case "info", "request":
		return errors.New("faucet commands require --rpc-url")
	default:
		return fmt.Errorf("unknown faucet command %q", args[0])
	}
}

func (a App) mempool(args []string) error {
	if len(args) == 0 {
		return errors.New("mempool command is required")
	}
	mp := mempool.New(a.paths.Mempool)
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("mempool list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		detail := fs.Bool("detail", false, "print detailed transactions")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		txs, err := mp.Load()
		if err != nil {
			return err
		}
		fmt.Fprintf(a.out, "pending tx count: %d\n", len(txs))
		for _, tx := range txs {
			if *detail {
				fmt.Fprintf(a.out, "txid: %s\nfrom: %s\nto: %s\namount: %s %s\nfee: %s %s\nnonce: %d\ntimestamp: %d\nstatus: pending\n", tx.ID, tx.From, tx.To, amount.Format(tx.Amount), config.Ticker, amount.Format(tx.Fee), config.Ticker, tx.Nonce, tx.Timestamp)
				continue
			}
			fmt.Fprintf(a.out, "id=%s from=%s to=%s amount=%s %s fee=%s %s nonce=%d\n", tx.ID, tx.From, tx.To, amount.Format(tx.Amount), config.Ticker, amount.Format(tx.Fee), config.Ticker, tx.Nonce)
		}
		return nil
	case "clear":
		fs := flag.NewFlagSet("mempool clear", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		yes := fs.Bool("yes", false, "confirm clear")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if !*yes {
			return errors.New("refusing to clear mempool without --yes")
		}
		if err := mp.Clear(); err != nil {
			return err
		}
		fmt.Fprintln(a.out, "mempool cleared")
		return nil
	default:
		return fmt.Errorf("unknown mempool command %q", args[0])
	}
}

func (a App) tx(args []string) error {
	if len(args) != 2 || args[0] != "get" {
		return errors.New("usage: tx get <txid>")
	}
	result, ok, err := a.findTransaction(args[1])
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(a.out, "tx not found")
		return nil
	}
	printTransaction(a.out, result)
	return nil
}

func (a App) mine(args []string) error {
	if len(args) > 0 && args[0] == "status" {
		fmt.Fprintln(a.out, "mining status: idle")
		return nil
	}
	fs := flag.NewFlagSet("mine", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	address := fs.String("address", "", "miner address")
	count := fs.Uint64("blocks", 1, "block count")
	timeout := fs.Duration("timeout", 0, "mining timeout")
	maxNonce := fs.Uint64("max-nonce", 0, "maximum nonce, 0 means unlimited")
	mineVerbose := fs.Bool("mine-verbose", false, "print mining progress")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := crypto.ValidateAddressForNetwork(*address, a.profile); err != nil {
		return err
	}
	ctx := context.Background()
	cancel := func() {}
	if *timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, *timeout)
	}
	defer cancel()
	bc, closeFn, err := a.openInitializedChain()
	if err != nil {
		return err
	}
	defer closeFn()
	mp := mempool.New(a.paths.Mempool)
	for i := uint64(0); i < *count; i++ {
		pending, err := mp.Load()
		if err != nil {
			return err
		}
		block, err := bc.MineBlockWithContextAndNetwork(ctx, *address, pending, chain.MineOptions{
			MaxNonce: *maxNonce,
			Verbose:  *mineVerbose,
			OnProgress: func(nonce uint64, hash string) {
				fmt.Fprintf(a.out, "mining progress nonce=%d hash_prefix=%s\n", nonce, firstN(hash, 8))
			},
		}, a.profile)
		if err != nil {
			return err
		}
		if err := bc.AddBlockWithNetwork(block, a.profile); err != nil {
			return err
		}
		included := make(map[string]struct{})
		for _, tx := range block.Transactions {
			if tx.Coinbase {
				continue
			}
			included[tx.ID] = struct{}{}
		}
		if err := mp.RemoveIDs(included); err != nil {
			return err
		}
		a.broadcastBlock(block)
		reward := uint64(0)
		if len(block.Transactions) > 0 && block.Transactions[0].Coinbase {
			reward = block.Transactions[0].Amount
		}
		fmt.Fprintf(a.out, "mined block height=%d hash=%s txs=%d reward=%s %s difficulty=%d nonce=%d\n", block.Height, block.Hash, len(block.Transactions), amount.Format(reward), config.Ticker, block.Difficulty, block.Nonce)
	}
	blocks, err := bc.Blocks()
	if err != nil {
		return err
	}
	tip, err := bc.Tip()
	if err != nil {
		return err
	}
	fmt.Fprintln(a.out, "mining complete")
	fmt.Fprintf(a.out, "mined blocks: %d\n", *count)
	fmt.Fprintf(a.out, "new height: %d\n", tip.Height)
	pending, _ := mp.Load()
	details, err := ledger.BalanceDetailsForWithProfile(*address, blocks, pending, consensusParams(a.profile), a.profile)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.out, "miner balance: %s %s\n", amount.Format(details.Confirmed), config.Ticker)
	fmt.Fprintf(a.out, "miner mature balance: %s %s\n", amount.Format(details.Mature), config.Ticker)
	fmt.Fprintf(a.out, "miner immature balance: %s %s\n", amount.Format(details.Immature), config.Ticker)
	fmt.Fprintf(a.out, "miner spendable balance: %s %s\n", amount.Format(details.Spendable), config.Ticker)
	return nil
}

func (a App) service(args []string) error {
	if len(args) == 0 {
		return errors.New("service command is required")
	}
	store := servicenode.NewStore(a.paths, a.profile)
	switch args[0] {
	case "register":
		fs := flag.NewFlagSet("service register", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		endpoint := fs.String("endpoint", "", "advertised endpoint")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		node, err := store.Register(*address, *endpoint, localServiceMetadata())
		if err != nil {
			return err
		}
		printServiceRegistered(a.out, map[string]any{"service_node_id": node.ServiceNodeID, "owner_address": node.OwnerAddress, "status": node.Status})
		return nil
	case "heartbeat":
		fs := flag.NewFlagSet("service heartbeat", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		endpoint := fs.String("endpoint", "", "advertised endpoint")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		score, node, err := store.Heartbeat(*address, *endpoint, localServiceMetadata())
		if err != nil {
			return err
		}
		printServiceHeartbeat(a.out, map[string]any{
			"status":            node.Status,
			"uptime_score":      float64(score.UptimeScore),
			"latency_score":     float64(score.LatencyScore),
			"bandwidth_score":   float64(score.BandwidthScore),
			"reliability_score": float64(score.ReliabilityScore),
			"abuse_penalty":     float64(score.AbusePenalty),
			"service_score":     float64(score.ServiceScore),
			"note":              score.Note,
		})
		return nil
	case "challenge":
		return a.serviceChallenge(args[1:], store)
	case "score":
		fs := flag.NewFlagSet("service score", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		score, err := store.Score(*address)
		if err != nil {
			return err
		}
		printServiceScore(a.out, scoreMap(score))
		return nil
	case "rewards":
		fs := flag.NewFlagSet("service rewards", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		rewards, err := store.Rewards(*address)
		if err != nil {
			return err
		}
		score, _ := store.Score(*address)
		printServiceRewards(a.out, *address, rewards, score.Note)
		return nil
	case "list":
		nodes, err := store.LoadNodes()
		if err != nil {
			return err
		}
		printServiceList(a.out, nodes)
		return nil
	default:
		return fmt.Errorf("unknown service command %q", args[0])
	}
}

func (a App) serviceChallenge(args []string, store servicenode.Store) error {
	if len(args) == 0 {
		return errors.New("usage: service challenge <create|submit>")
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("service challenge create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		challenge, err := store.CreateChallenge(*address)
		if err != nil {
			return err
		}
		printServiceChallenge(a.out, map[string]any{
			"challenge_id": challenge.ChallengeID,
			"address":      challenge.Address,
			"expires_at":   float64(challenge.ExpiresAt),
			"status":       challenge.Status,
		})
		return nil
	case "submit":
		fs := flag.NewFlagSet("service challenge submit", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		challengeID := fs.String("challenge-id", "", "challenge id")
		latencyMS := fs.Int64("latency-ms", 0, "latency in milliseconds")
		bytesUp := fs.Int64("bytes-up", 0, "uploaded bytes")
		bytesDown := fs.Int64("bytes-down", 0, "downloaded bytes")
		success := fs.Bool("success", true, "challenge success")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		score, challenge, err := store.SubmitChallenge(*challengeID, *latencyMS, *bytesUp, *bytesDown, *success)
		if err != nil {
			return err
		}
		printServiceChallengeSubmit(a.out, map[string]any{
			"challenge_id":  challenge.ChallengeID,
			"status":        challenge.Status,
			"service_score": float64(score.ServiceScore),
			"note":          score.Note,
		})
		return nil
	default:
		return fmt.Errorf("unknown service challenge command %q", args[0])
	}
}

func (a App) stake(args []string) error {
	if len(args) == 0 {
		return errors.New("stake command is required")
	}
	switch args[0] {
	case "info":
		return a.stakeInfo()
	case "list":
		fs := flag.NewFlagSet("stake list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return a.stakeList(*address)
	case "status":
		fs := flag.NewFlagSet("stake status", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		stakeID := fs.String("stake-id", "", "stake id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return a.stakeStatus(*stakeID)
	case "lock":
		fs := flag.NewFlagSet("stake lock", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		amountText := fs.String("amount", "", "amount")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		tx, err := a.createStakeLockTx(*address, *amountText)
		if err != nil {
			return err
		}
		if err := mempool.New(a.paths.Mempool).Add(tx); err != nil {
			return err
		}
		a.broadcastTx(tx)
		fmt.Fprintln(a.out, "stake lock tx created")
		fmt.Fprintf(a.out, "tx id: %s\n", tx.ID)
		fmt.Fprintf(a.out, "stake id: %s\n", tx.StakeID)
		fmt.Fprintf(a.out, "address: %s\n", tx.From)
		fmt.Fprintf(a.out, "amount: %s %s\n", amount.Format(tx.Amount), config.Ticker)
		fmt.Fprintln(a.out, "status: pending")
		fmt.Fprintln(a.out, "note: stake becomes active after tx is mined")
		return nil
	case "unlock":
		fs := flag.NewFlagSet("stake unlock", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		stakeID := fs.String("stake-id", "", "stake id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		tx, releaseHeight, err := a.createStakeUnlockTx(*address, *stakeID)
		if err != nil {
			return err
		}
		if err := mempool.New(a.paths.Mempool).Add(tx); err != nil {
			return err
		}
		a.broadcastTx(tx)
		fmt.Fprintln(a.out, "stake unlock tx created")
		fmt.Fprintf(a.out, "tx id: %s\n", tx.ID)
		fmt.Fprintf(a.out, "stake id: %s\n", tx.StakeID)
		fmt.Fprintf(a.out, "release height: %d\n", releaseHeight)
		fmt.Fprintln(a.out, "status: pending")
		return nil
	default:
		return fmt.Errorf("unknown stake command %q", args[0])
	}
}

func (a App) stakeInfo() error {
	blocks, err := a.loadBlocks()
	if err != nil {
		return err
	}
	params := consensusParams(a.profile).Staking
	state, err := staking.Replay(blocks, params)
	if err != nil {
		return err
	}
	summary := state.Summary(currentHeight(blocks))
	fmt.Fprintf(a.out, "staking enabled: %t\n", params.Enabled)
	fmt.Fprintf(a.out, "min stake amount: %s %s\n", amount.Format(params.MinStakeAmount), config.Ticker)
	fmt.Fprintf(a.out, "min service stake: %s %s\n", amount.Format(params.MinServiceStake), config.Ticker)
	fmt.Fprintf(a.out, "unbonding period: %d blocks\n", params.UnbondingPeriodBlocks)
	fmt.Fprintf(a.out, "total active stake: %s %s\n", amount.Format(summary.TotalActiveStake), config.Ticker)
	fmt.Fprintf(a.out, "total unlocking stake: %s %s\n", amount.Format(summary.TotalUnlockingStake), config.Ticker)
	fmt.Fprintf(a.out, "active stake records: %d\n", summary.ActiveStakeCount)
	return nil
}

func (a App) stakeList(address string) error {
	blocks, err := a.loadBlocks()
	if err != nil {
		return err
	}
	state, err := staking.Replay(blocks, consensusParams(a.profile).Staking)
	if err != nil {
		return err
	}
	records := state.Records(currentHeight(blocks))
	count := 0
	for _, record := range records {
		if address != "" && record.OwnerAddress != address {
			continue
		}
		count++
		fmt.Fprintf(a.out, "stake id: %s\nowner: %s\namount: %s %s\nstatus: %s\nlock height: %d\nunlock height: %d\nrelease height: %d\n", record.StakeID, record.OwnerAddress, amount.Format(record.Amount), config.Ticker, record.Status, record.LockHeight, record.UnlockHeight, record.ReleaseHeight)
	}
	fmt.Fprintf(a.out, "stake records: %d\n", count)
	return nil
}

func (a App) stakeStatus(stakeID string) error {
	if stakeID == "" {
		return errors.New("stake-id is required")
	}
	blocks, err := a.loadBlocks()
	if err != nil {
		return err
	}
	state, err := staking.Replay(blocks, consensusParams(a.profile).Staking)
	if err != nil {
		return err
	}
	record, ok := state.Find(stakeID, currentHeight(blocks))
	if !ok {
		return errors.New("stake not found")
	}
	fmt.Fprintf(a.out, "stake id: %s\nowner: %s\namount: %s %s\nstatus: %s\nlock height: %d\nunlock height: %d\nrelease height: %d\n", record.StakeID, record.OwnerAddress, amount.Format(record.Amount), config.Ticker, record.Status, record.LockHeight, record.UnlockHeight, record.ReleaseHeight)
	return nil
}

func (a App) chain(args []string) error {
	if len(args) == 0 {
		return errors.New("chain command is required")
	}
	bc, closeFn, err := a.openInitializedChain()
	if err != nil {
		return err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return err
	}
	switch args[0] {
	case "info":
		tip := blocks[len(blocks)-1]
		pending, err := mempool.New(a.paths.Mempool).Load()
		if err != nil {
			return err
		}
		stats := chain.CalculateChainStatsWithProfile(blocks, a.profile)
		params := difficultyParams(a.profile)
		nextDifficulty := chain.CalculateNextDifficultyWithParams(blocks, params)
		fmt.Fprintf(a.out, "height: %d\n", tip.Height)
		fmt.Fprintf(a.out, "tip hash: %s\n", tip.Hash)
		fmt.Fprintf(a.out, "difficulty: %d\n", nextDifficulty)
		fmt.Fprintf(a.out, "tip difficulty: %d\n", tip.Difficulty)
		fmt.Fprintf(a.out, "next difficulty: %d\n", nextDifficulty)
		fmt.Fprintf(a.out, "target block time: %ds\n", params.TargetBlockTimeSeconds)
		fmt.Fprintf(a.out, "retarget window: %d\n", params.RetargetWindow)
		fmt.Fprintf(a.out, "min difficulty: %d\n", params.MinDifficulty)
		fmt.Fprintf(a.out, "max difficulty: %d\n", params.MaxDifficulty)
		fmt.Fprintf(a.out, "blocks until retarget: %d\n", chain.BlocksUntilRetarget(blocks, params))
		fmt.Fprintf(a.out, "coinbase maturity: %d\n", consensusParams(a.profile).CoinbaseMaturity)
		fmt.Fprintf(a.out, "total supply: %s %s\n", amount.Format(stats.TotalSupply), config.Ticker)
		fmt.Fprintf(a.out, "cumulative work: %d\n", stats.CumulativeWork)
		fmt.Fprintf(a.out, "pending tx count: %d\n", len(pending))
		fmt.Fprintf(a.out, "blocks: %d\n", stats.Blocks)
		fmt.Fprintf(a.out, "coinbase blocks: %d\n", stats.CoinbaseBlocks)
		fmt.Fprintf(a.out, "total transactions: %d\n", stats.TotalTransactions)
		fmt.Fprintf(a.out, "coinbase transactions: %d\n", stats.CoinbaseTransactions)
		fmt.Fprintf(a.out, "normal transactions: %d\n", stats.NormalTransactions)
		fmt.Fprintf(a.out, "circulating supply: %s %s\n", amount.Format(stats.CirculatingSupply), config.Ticker)
		stakeState, _ := staking.Replay(blocks, consensusParams(a.profile).Staking)
		if stakeState != nil {
			summary := stakeState.Summary(currentHeight(blocks))
			fmt.Fprintf(a.out, "staking enabled: %t\n", consensusParams(a.profile).Staking.Enabled)
			fmt.Fprintf(a.out, "min stake amount: %s %s\n", amount.Format(consensusParams(a.profile).Staking.MinStakeAmount), config.Ticker)
			fmt.Fprintf(a.out, "min service stake: %s %s\n", amount.Format(consensusParams(a.profile).Staking.MinServiceStake), config.Ticker)
			fmt.Fprintf(a.out, "unbonding period: %d\n", consensusParams(a.profile).Staking.UnbondingPeriodBlocks)
			fmt.Fprintf(a.out, "total active stake: %s %s\n", amount.Format(summary.TotalActiveStake), config.Ticker)
			fmt.Fprintf(a.out, "total unlocking stake: %s %s\n", amount.Format(summary.TotalUnlockingStake), config.Ticker)
			fmt.Fprintf(a.out, "active stake count: %d\n", summary.ActiveStakeCount)
		}
		fmt.Fprintf(a.out, "network: %s\n", a.profile.Name)
		fmt.Fprintf(a.out, "network id: %s\n", a.profile.NetworkID)
		fmt.Fprintf(a.out, "chain id: %d\n", a.profile.ChainID)
		fmt.Fprintf(a.out, "protocol version: %d\n", a.profile.ProtocolVersion)
		fmt.Fprintf(a.out, "rpc api version: %s\n", a.profile.RPCAPIVersion)
		fmt.Fprintf(a.out, "datadir: %s\n", a.paths.DataDir)
		return nil
	case "difficulty":
		printChainDifficulty(a.out, chainDifficultyInfo(blocks, a.profile))
		return nil
	case "print":
		for _, block := range blocks {
			fmt.Fprintf(a.out, "height=%d hash=%s previous=%s txs=%d difficulty=%d\n", block.Height, block.Hash, block.PreviousHash, len(block.Transactions), block.Difficulty)
		}
		return nil
	case "validate":
		result, err := chain.ValidateChainWithNetwork(blocks, a.profile)
		if err != nil {
			return fmt.Errorf("chain invalid: %w", err)
		}
		fmt.Fprintln(a.out, "chain valid")
		fmt.Fprintf(a.out, "height: %d\n", result.Height)
		fmt.Fprintf(a.out, "blocks: %d\n", result.Blocks)
		fmt.Fprintf(a.out, "total supply: %s %s\n", amount.Format(result.TotalSupply), config.Ticker)
		return nil
	case "locator":
		printLocator(a.out, p2p.LocatorResponse{
			Height:  blocks[len(blocks)-1].Height,
			TipHash: blocks[len(blocks)-1].Hash,
			Locator: chain.BuildBlockLocator(blocks),
		})
		return nil
	case "common-ancestor":
		fs := flag.NewFlagSet("chain common-ancestor", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		peer := fs.String("peer", "", "peer P2P URL")
		debug := fs.Bool("debug", false, "show locator sent to peer")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *peer == "" {
			return errors.New("usage: chain common-ancestor --peer <p2p-url>")
		}
		locator := p2p.LocatorResponse{
			Height:  blocks[len(blocks)-1].Height,
			TipHash: blocks[len(blocks)-1].Hash,
			Locator: chain.BuildBlockLocator(blocks),
		}
		if *debug {
			printCommonAncestorDebug(a.out, locator, *peer)
		}
		resp, err := p2p.NewClient().CommonAncestor(*peer, p2p.CommonAncestorRequest{Locator: locator.Locator})
		if err != nil {
			return err
		}
		printCommonAncestor(a.out, resp)
		return nil
	default:
		return fmt.Errorf("unknown chain command %q", args[0])
	}
}

func (a App) mining(args []string) error {
	if len(args) == 0 {
		return errors.New("mining command is required")
	}
	switch args[0] {
	case "status", "difficulty":
		blocks, err := a.loadBlocks()
		if err != nil {
			return err
		}
		pending, _ := mempool.New(a.paths.Mempool).Load()
		peers, _ := p2p.NewPeerStore(a.paths.Peers).LoadMetadata()
		printMiningStatus(a.out, miningObservationInfo(blocks, a.profile, len(pending), len(peers)))
		return nil
	case "blocks":
		fs := flag.NewFlagSet("mining blocks", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		limit := fs.Int("limit", 30, "recent block limit")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		blocks, err := a.loadBlocks()
		if err != nil {
			return err
		}
		pending, _ := mempool.New(a.paths.Mempool).Load()
		peers, _ := p2p.NewPeerStore(a.paths.Peers).LoadMetadata()
		info := miningObservationInfo(blocks, a.profile, len(pending), len(peers))
		info["blocks"] = recentMiningBlockViews(blocks, *limit)
		printMiningBlocks(a.out, info)
		return nil
	default:
		return fmt.Errorf("unknown mining command %q", args[0])
	}
}

func (a App) fork(args []string) error {
	if len(args) == 0 {
		return errors.New("fork command is required")
	}
	switch args[0] {
	case "check":
		fs := flag.NewFlagSet("fork check", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		peer := fs.String("peer", "", "peer P2P URL")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *peer == "" {
			return errors.New("usage: fork check --peer <p2p-url>")
		}
		result, err := p2p.CheckForkWithProfile(a.paths, *peer, a.profile)
		if err != nil {
			return err
		}
		printForkCheck(a.out, result)
		return nil
	case "inspect":
		fs := flag.NewFlagSet("fork inspect", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		other := fs.String("other-datadir", "", "other datadir")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *other == "" {
			return errors.New("usage: fork inspect --other-datadir <path>")
		}
		result, err := p2p.InspectDatadirFork(a.paths, config.NewPaths(*other))
		if err != nil {
			return err
		}
		printForkInspect(a.out, result)
		return nil
	default:
		return fmt.Errorf("unknown fork command %q", args[0])
	}
}

func (a App) debug(args []string) error {
	if len(args) == 1 && args[0] == "locks" {
		fmt.Fprintln(a.out, "debug locks is only available with --rpc-url")
		return nil
	}
	return errors.New("usage: debug locks")
}

func (a App) rpc(args []string) error {
	fs := flag.NewFlagSet("rpc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addr := fs.String("addr", ":8332", "listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fmt.Fprintf(a.out, "rpc listening on %s\n", *addr)
	return rpc.ListenAndServe(*addr, a.paths)
}

func (a App) node(args []string) error {
	if len(args) == 0 {
		return errors.New("node command is required")
	}
	if args[0] == "status" {
		return a.nodeStatus()
	}
	if args[0] == "compare" {
		return a.nodeCompare(args[1:])
	}
	if args[0] != "start" {
		return errors.New("usage: node start")
	}
	fs := flag.NewFlagSet("node start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rpcAddr := fs.String("rpc", fmt.Sprintf(":%d", a.profile.DefaultRPCPort), "rpc listen address")
	p2pAddr := fs.String("p2p", fmt.Sprintf(":%d", a.profile.DefaultP2PPort), "p2p listen address")
	p2pAdvertise := fs.String("advertise-p2p", "", "p2p advertised URL")
	peerText := fs.String("peers", "", "comma-separated peer URLs")
	bootnode := fs.String("bootnode", "", "bootstrap peer URL")
	bootnodesText := fs.String("bootnodes", "", "comma-separated bootstrap peer URLs")
	var seedPeer multiStringFlag
	fs.Var(&seedPeer, "seed-peer", "seed peer URL; may be repeated")
	seedPeersText := fs.String("seed-peers", os.Getenv("IDR_SEED_PEERS"), "comma-separated seed peer URLs")
	seedFile := fs.String("seed-file", "", "seed peer file, one URL per line")
	var upstreamPeer multiStringFlag
	fs.Var(&upstreamPeer, "upstream-peer", "upstream peer URL; may be repeated")
	upstreamPeersText := fs.String("upstream-peers", os.Getenv("IDR_UPSTREAM_PEERS"), "comma-separated upstream peer URLs")
	upstreamFile := fs.String("upstream-file", "", "upstream peer file, one URL per line")
	configPath := fs.String("config", "", "optional JSON config file")
	publicRPC := fs.Bool("public-rpc", false, "enable public RPC safety mode")
	enableWalletRPC := fs.Bool("enable-wallet-rpc", true, "enable wallet management RPC")
	enableMinerRPC := fs.Bool("enable-miner-rpc", true, "enable miner RPC")
	enableAdminRPC := fs.Bool("enable-admin-rpc", true, "enable admin/debug RPC")
	enableServiceRPC := fs.Bool("enable-service-rpc", true, "enable service-node simulation RPC")
	enableFaucetRPC := fs.Bool("enable-faucet-rpc", false, "enable dev/testnet faucet RPC")
	faucetAddress := fs.String("faucet-address", "", "faucet source address")
	faucetAmount := fs.String("faucet-amount", "100", "faucet amount per request")
	faucetMinInterval := fs.Duration("faucet-min-interval", time.Minute, "minimum interval per faucet recipient")
	faucetMaxPerAddress := fs.String("faucet-max-per-address", "1000", "maximum faucet amount per address per day")
	corsOrigins := fs.String("cors-origins", os.Getenv("IDR_CORS_ORIGINS"), "comma-separated allowed CORS origins")
	rateLimitPerMinute := 300
	maxPeers := a.profile.MaxPeers
	defaultMaxReorgDepth, err := config.MaxReorgDepthFromEnv(a.profile)
	if err != nil {
		return err
	}
	maxReorgDepth := fs.Uint64("max-reorg-depth", defaultMaxReorgDepth, "maximum automatic reorg depth")
	defaultMinMiningPeers, err := config.MinMiningPeersFromEnv(a.profile)
	if err != nil {
		return err
	}
	defaultAllowIsolatedMining, err := config.AllowIsolatedMiningFromEnv(a.profile)
	if err != nil {
		return err
	}
	minMiningPeers := fs.Int("min-mining-peers", defaultMinMiningPeers, "minimum active peers before miner templates are enabled")
	allowIsolatedMining := fs.Bool("allow-isolated-mining", defaultAllowIsolatedMining, "allow miner templates while isolated")
	defaultMinWritePeers, err := config.MinWritePeersFromEnv(a.profile)
	if err != nil {
		return err
	}
	defaultAllowIsolatedWrites, err := config.AllowIsolatedWritesFromEnv(a.profile)
	if err != nil {
		return err
	}
	minWritePeers := fs.Int("min-write-peers", defaultMinWritePeers, "minimum active peers before testnet write RPCs are enabled")
	allowIsolatedWrites := fs.Bool("allow-isolated-writes", defaultAllowIsolatedWrites, "allow testnet write RPCs while isolated")
	_ = fs.String("mine-address", "", "optional miner address")
	_ = fs.Bool("auto-mine", false, "auto mine")
	verbose := fs.Bool("verbose", false, "verbose sync logging")
	syncInterval := fs.Duration("sync-interval", 5*time.Second, "sync interval")
	peerTTL := fs.Duration("peer-ttl", p2p.DefaultPeerTTL, "remove non-seed peers not seen within this TTL")
	peerMaintenanceInterval := fs.Duration("peer-maintenance-interval", 30*time.Second, "peer discovery and bootstrap maintenance interval")
	maxPeersPerIP := fs.Int("max-peers-per-ip", p2p.DefaultMaxPeersPerIP, "maximum stored peers per IP/subnet")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *configPath != "" {
		cfg, err := loadNodeConfig(*configPath)
		if err != nil {
			return err
		}
		applyNodeConfig(fs, cfg, rpcAddr, p2pAddr, p2pAdvertise, seedPeersText, seedFile, upstreamPeersText, upstreamFile, publicRPC, enableWalletRPC, enableMinerRPC, enableAdminRPC, enableServiceRPC, corsOrigins, &rateLimitPerMinute, &maxPeers, maxReorgDepth, minMiningPeers, allowIsolatedMining, minWritePeers, allowIsolatedWrites)
	}
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })
	if *publicRPC {
		if !visited["enable-wallet-rpc"] {
			*enableWalletRPC = false
		}
		if !visited["enable-miner-rpc"] {
			*enableMinerRPC = false
		}
		if !visited["enable-admin-rpc"] {
			*enableAdminRPC = false
		}
		if !visited["enable-service-rpc"] {
			*enableServiceRPC = false
		}
	}
	if err := validatePublicRPCMode(*publicRPC, *rpcAddr, *enableWalletRPC, *enableAdminRPC); err != nil {
		return err
	}
	faucetAmountUnits, err := amount.Parse(*faucetAmount)
	if err != nil {
		return fmt.Errorf("invalid faucet amount: %w", err)
	}
	faucetMaxPerAddressUnits, err := amount.Parse(*faucetMaxPerAddress)
	if err != nil {
		return fmt.Errorf("invalid faucet max per address: %w", err)
	}
	if *enableFaucetRPC {
		if strings.TrimSpace(*faucetAddress) == "" {
			return errors.New("faucet address is required when faucet RPC is enabled")
		}
		if err := crypto.ValidateAddressForNetwork(*faucetAddress, a.profile); err != nil {
			return fmt.Errorf("invalid faucet address: %w", err)
		}
		if faucetAmountUnits == 0 {
			return errors.New("faucet amount must be greater than 0")
		}
		if faucetMaxPerAddressUnits > 0 && faucetAmountUnits > faucetMaxPerAddressUnits {
			return errors.New("faucet amount exceeds faucet max per address")
		}
	}
	bc, closeFn, err := a.openInitializedChain()
	if err != nil {
		return err
	}
	if err := config.EnsureNetworkMatches(a.paths, a.profile); err != nil {
		closeFn()
		return err
	}
	tip, err := bc.Tip()
	closeFn()
	if err != nil {
		return err
	}
	state, err := nodestate.NewWithProfile(a.paths, a.profile)
	if err != nil {
		return err
	}
	miningService := mining.NewService()
	chainMutationMu := &sync.Mutex{}
	a.chainMu = chainMutationMu
	flagPeers := parsePeers(*peerText)
	bootnodes := parsePeers(strings.Join([]string{*bootnode, *bootnodesText}, ","))
	seedPeers, err := collectSeedPeers(a.profile, strings.Join(seedPeer, ","), *seedPeersText, *seedFile)
	if err != nil {
		return err
	}
	upstreamPeers, err := collectUpstreamPeers(strings.Join(upstreamPeer, ","), *upstreamPeersText, *upstreamFile)
	if err != nil {
		return err
	}
	store := p2p.NewPeerStore(a.paths.Peers)
	filePeers, err := store.LoadMetadata()
	if err != nil {
		return err
	}
	advertise := *p2pAdvertise
	if advertise == "" {
		advertise = defaultAdvertiseP2P(*p2pAddr)
	}
	startupPeers, err := addStartupPeers(store, startupPeerInputs{FlagPeers: flagPeers, Bootnodes: bootnodes, SeedPeers: seedPeers, SelfURL: advertise})
	if err != nil {
		return err
	}
	peers, err := store.Load()
	if err != nil {
		return err
	}
	peerSource := peerSourceSummary(len(filePeers) > 0, len(flagPeers)+len(bootnodes)+len(seedPeers) > 0)
	lock, err := p2p.CreateLock(a.paths.Lock, *rpcAddr, *p2pAddr, advertise)
	if err != nil {
		return fmt.Errorf("datadir is locked by running node: %w", err)
	}
	defer func() {
		_ = state.Refresh(a.paths)
		_ = p2p.RemoveLock(a.paths.Lock)
		log.Printf("node stopped rpc=%s p2p=%s", lock.RPC, lock.P2P)
	}()
	serviceStore := servicenode.NewStore(a.paths, a.profile)
	serviceNodes, _ := serviceStore.LoadNodes()
	log.Printf("node started network=%s network_id=%s chain_id=%d genesis=%s protocol=%d datadir=%s rpc=%s p2p=%s advertise=%s bootnodes=%d seed_peers=%d upstream_peers=%d skipped_self=%d public_rpc=%t wallet_rpc=%t miner_rpc=%t admin_rpc=%t service_rpc=%t faucet_rpc=%t max_peers=%d max_reorg_depth=%d min_mining_peers=%d allow_isolated_mining=%t height=%d tip=%s pid=%d", a.profile.Name, a.profile.NetworkID, a.profile.ChainID, chain.GenesisBlockForNetwork(a.profile).Hash, a.profile.ProtocolVersion, a.paths.DataDir, lock.RPC, lock.P2P, lock.P2PAdvertise, startupPeers.BootnodesAdded, startupPeers.SeedsAdded, len(upstreamPeers), startupPeers.SkippedSelf, *publicRPC, *enableWalletRPC, *enableMinerRPC, *enableAdminRPC, *enableServiceRPC, *enableFaucetRPC, maxPeers, *maxReorgDepth, *minMiningPeers, *allowIsolatedMining, tip.Height, tip.Hash, lock.PID)
	if *enableMinerRPC && a.profile.Name == "testnet" && !*allowIsolatedMining && *minMiningPeers > 0 {
		log.Printf("warning: miner RPC enabled; mining templates require at least %d active peer or reachable upstream peer on testnet", *minMiningPeers)
	}
	log.Printf("peer store: %s", a.paths.Peers)
	log.Printf("service store: %s", a.paths.ServiceNodes)
	log.Printf("service nodes: %d", len(serviceNodes))
	p2pServer := p2p.NewHTTPServerWithProfileAndMutationMutex(*p2pAddr, a.paths, state, a.profile, chainMutationMu, advertise)
	go func() {
		if err := p2pServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("p2p server stopped: %v", err)
		}
	}()
	rpcInfo := rpc.NodeInfo{
		RPCListen:                *rpcAddr,
		P2PListen:                *p2pAddr,
		P2PAdvertise:             advertise,
		PeerCount:                len(peers),
		State:                    state,
		Mining:                   miningService,
		PublicRPC:                *publicRPC,
		EnableWalletRPC:          *enableWalletRPC,
		EnableWalletRPCSet:       true,
		EnableMinerRPC:           *enableMinerRPC,
		EnableMinerRPCSet:        true,
		EnableAdminRPC:           *enableAdminRPC,
		EnableAdminRPCSet:        true,
		EnableServiceRPC:         *enableServiceRPC,
		EnableServiceRPCSet:      true,
		EnableFaucetRPC:          *enableFaucetRPC,
		EnableFaucetRPCSet:       true,
		FaucetAddress:            strings.TrimSpace(*faucetAddress),
		FaucetAmount:             faucetAmountUnits,
		FaucetMinInterval:        *faucetMinInterval,
		FaucetMaxPerAddress:      faucetMaxPerAddressUnits,
		CORSOrigins:              parseCSV(*corsOrigins),
		RateLimitPerMinute:       rateLimitPerMinute,
		MinerRateLimitPerMinute:  120,
		WalletRateLimitPerMinute: 30,
		MaxPeers:                 maxPeers,
		MaxReorgDepth:            *maxReorgDepth,
		UpstreamPeers:            upstreamPeers,
		MinMiningPeers:           *minMiningPeers,
		AllowIsolatedMining:      *allowIsolatedMining,
		MinWritePeers:            *minWritePeers,
		AllowIsolatedWrites:      *allowIsolatedWrites,
		StartedAt:                time.Now(),
		Profile:                  a.profile,
		ChainMutationMu:          chainMutationMu,
	}
	rpcServer := rpc.NewHTTPServer(*rpcAddr, a.paths, rpcInfo)
	go func() {
		if err := rpcServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("rpc server stopped: %v", err)
		}
	}()
	go a.syncLoop(*syncInterval, *verbose, state, *maxReorgDepth, upstreamPeers)
	go a.bootstrapPeers(seedPeers, advertise, state, *maxReorgDepth, upstreamPeers)
	maintenanceCtx, maintenanceCancel := context.WithCancel(context.Background())
	defer maintenanceCancel()
	go p2p.RunMaintenanceWorker(maintenanceCtx, a.paths, a.profile, p2p.MaintenanceOptions{
		SelfURL:       advertise,
		TTL:           *peerTTL,
		Interval:      *peerMaintenanceInterval,
		Limit:         p2p.DefaultMaxDiscoveredPeers,
		MaxPeers:      maxPeers,
		MaxPeersPerIP: *maxPeersPerIP,
	})
	fmt.Fprintln(a.out, "node started")
	fmt.Fprintf(a.out, "network: %s\n", a.profile.Name)
	fmt.Fprintf(a.out, "network id: %s\n", a.profile.NetworkID)
	fmt.Fprintf(a.out, "chain id: %d\n", a.profile.ChainID)
	fmt.Fprintf(a.out, "genesis hash: %s\n", chain.GenesisBlockForNetwork(a.profile).Hash)
	fmt.Fprintf(a.out, "datadir: %s\n", a.paths.DataDir)
	fmt.Fprintf(a.out, "rpc: %s\n", *rpcAddr)
	fmt.Fprintf(a.out, "p2p: %s\n", *p2pAddr)
	fmt.Fprintf(a.out, "p2p advertise: %s\n", advertise)
	fmt.Fprintf(a.out, "peer store: %s\n", a.paths.Peers)
	fmt.Fprintf(a.out, "peers: %d\n", len(peers))
	fmt.Fprintf(a.out, "bootnodes: %d\n", startupPeers.BootnodesAdded)
	fmt.Fprintf(a.out, "seed peers: %d\n", startupPeers.SeedsAdded)
	fmt.Fprintf(a.out, "upstream peers: %d\n", len(upstreamPeers))
	if len(upstreamPeers) > 0 {
		fmt.Fprintln(a.out, "upstream peer source: flag")
		for _, peer := range upstreamPeers {
			fmt.Fprintf(a.out, "upstream peer: %s\n", peer)
		}
	}
	fmt.Fprintf(a.out, "skipped self peers: %d\n", startupPeers.SkippedSelf)
	fmt.Fprintf(a.out, "max peers: %d\n", maxPeers)
	fmt.Fprintf(a.out, "max peers per ip: %d\n", *maxPeersPerIP)
	fmt.Fprintf(a.out, "peer ttl: %s\n", peerTTL.String())
	fmt.Fprintf(a.out, "peer maintenance interval: %s\n", peerMaintenanceInterval.String())
	fmt.Fprintf(a.out, "max reorg depth: %d\n", *maxReorgDepth)
	fmt.Fprintf(a.out, "min mining peers: %d\n", *minMiningPeers)
	fmt.Fprintf(a.out, "allow isolated mining: %t\n", *allowIsolatedMining)
	fmt.Fprintf(a.out, "min write peers: %d\n", *minWritePeers)
	fmt.Fprintf(a.out, "allow isolated writes: %t\n", *allowIsolatedWrites)
	fmt.Fprintf(a.out, "public rpc: %t\n", *publicRPC)
	fmt.Fprintf(a.out, "wallet rpc: %t\n", *enableWalletRPC)
	fmt.Fprintf(a.out, "miner rpc: %t\n", *enableMinerRPC)
	fmt.Fprintf(a.out, "admin rpc: %t\n", *enableAdminRPC)
	fmt.Fprintf(a.out, "service rpc: %t\n", *enableServiceRPC)
	fmt.Fprintf(a.out, "faucet rpc: %t\n", *enableFaucetRPC)
	fmt.Fprintf(a.out, "service store: %s\n", a.paths.ServiceNodes)
	fmt.Fprintf(a.out, "service nodes: %d\n", len(serviceNodes))
	if peerSource != "" {
		fmt.Fprintf(a.out, "peer source: %s\n", peerSource)
	}
	fmt.Fprintf(a.out, "height: %d\n", tip.Height)
	fmt.Fprintf(a.out, "tip: %s\n", tip.Hash)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Printf("shutdown started")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := rpcServer.Shutdown(ctx); err != nil {
		log.Printf("rpc server shutdown error: %v", err)
	} else {
		log.Printf("rpc server stopped")
	}
	if err := p2pServer.Shutdown(ctx); err != nil {
		log.Printf("p2p server shutdown error: %v", err)
	} else {
		log.Printf("p2p server stopped")
	}
	log.Printf("mempool saved")
	log.Printf("datadir lock released")
	log.Printf("shutdown complete")
	return nil
}

func (a App) peer(args []string) error {
	if len(args) == 0 {
		return errors.New("peer command is required")
	}
	store := p2p.NewPeerStore(a.paths.Peers)
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("peer list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		showSource := fs.Bool("source", false, "show peer source")
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		peers, err := store.LoadMetadata()
		if err != nil {
			return err
		}
		if *jsonOut {
			return json.NewEncoder(a.out).Encode(map[string]any{"peers": peers, "peer_count": len(peers)})
		}
		fmt.Fprintf(a.out, "peers: %d\n", len(peers))
		for _, peer := range peers {
			if *showSource {
				fmt.Fprintf(a.out, "url=%s source=%s status=%s score=%d last_error=%q\n", peer.URL, peer.Source, peer.Status, peer.Score, peer.LastError)
				continue
			}
			fmt.Fprintf(a.out, "url=%s node_id=%s network_id=%s chain_id=%d genesis_hash=%s protocol_version=%d height=%d tip_hash=%s status=%s score=%d source=%s last_seen=%s last_status_check=%s latency_ms=%d reason=%q last_error=%q\n",
				peer.URL, peer.NodeID, peer.NetworkID, peer.ChainID, peer.GenesisHash, peer.ProtocolVersion, peer.LastHeight, peer.LastTipHash, peer.Status, peer.Score, peer.Source, peer.LastSeenAt, peer.LastStatusCheckAt, peer.LastLatencyMS, peer.LastScoreReason, peer.LastError)
		}
		return nil
	case "add":
		if len(args) != 2 {
			return errors.New("usage: peer add <url>")
		}
		hs, err := p2p.CheckPeerWithProfile(a.paths, args[1], a.profile)
		if err != nil {
			return err
		}
		meta := p2p.MetadataFromHandshake(args[1], hs, 5)
		meta.Source = "manual"
		if err := store.Upsert(meta); err != nil {
			return err
		}
		_ = store.AdjustPeerScore(args[1], 0, "peer add")
		log.Printf("peer added %s", args[1])
		fmt.Fprintf(a.out, "peer added: %s\n", args[1])
		return nil
	case "connect":
		if len(args) != 2 {
			return errors.New("usage: peer connect <url>")
		}
		hs, err := p2p.CheckPeerWithProfile(a.paths, args[1], a.profile)
		if err != nil {
			return err
		}
		meta := p2p.MetadataFromHandshake(args[1], hs, 5)
		meta.Source = "peer connect"
		if err := store.Upsert(meta); err != nil {
			return err
		}
		_ = store.AdjustPeerScore(args[1], 0, "peer connect")
		introduced := false
		if info, locked := p2p.IsLocked(a.paths.Lock); locked && info.P2PAdvertise != "" {
			nodeID, _ := p2p.LoadOrCreateNodeID(a.paths.NodeID)
			net := a.profile
			_, err := p2p.NewClient().IntroducePeer(args[1], p2p.PeerIntroduction{
				URL:       info.P2PAdvertise,
				NodeID:    nodeID,
				NetworkID: net.NetworkID,
				ChainID:   net.ChainID,
			})
			introduced = err == nil
		}
		fmt.Fprintf(a.out, "peer connected: %s\n", args[1])
		fmt.Fprintf(a.out, "introduced: %t\n", introduced)
		return nil
	case "remove":
		if len(args) != 2 {
			return errors.New("usage: peer remove <url>")
		}
		if err := store.Remove(args[1]); err != nil {
			return err
		}
		log.Printf("peer removed %s", args[1])
		fmt.Fprintf(a.out, "peer removed: %s\n", args[1])
		return nil
	case "clear":
		fs := flag.NewFlagSet("peer clear", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		yes := fs.Bool("yes", false, "confirm clear")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if !*yes {
			return errors.New("refusing to clear peers without --yes")
		}
		if err := store.Clear(); err != nil {
			return err
		}
		fmt.Fprintln(a.out, "peers cleared")
		return nil
	case "sync":
		fs := flag.NewFlagSet("peer sync", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		includeBad := fs.Bool("include-bad", false, "include bad peers")
		allowReorg := fs.Bool("allow-reorg", false, "allow explicit safe reorg")
		defaultMaxDepth, err := config.MaxReorgDepthFromEnv(a.profile)
		if err != nil {
			return err
		}
		maxDepth := fs.Uint64("max-reorg-depth", defaultMaxDepth, "max reorg depth")
		yes := fs.Bool("yes", false, "confirm reorg")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		directPeer := ""
		if fs.NArg() == 1 {
			directPeer = fs.Arg(0)
		} else if fs.NArg() > 1 {
			return errors.New("usage: peer sync [peer-url]")
		}
		if directPeer != "" {
			if err := p2p.SyncFromPeerWithProfileAndMaxDepth(a.paths, directPeer, a.out, a.profile, *maxDepth); err != nil {
				if *allowReorg {
					if !*yes {
						return errors.New("refusing to apply reorg without --yes")
					}
					res, reorgErr := p2p.ApplyReorgWithProfile(a.paths, directPeer, *maxDepth, true, a.profile)
					if reorgErr != nil {
						return reorgErr
					}
					printReorgResult(a.out, res)
					return nil
				}
				return err
			}
			return nil
		}
		meta, err := store.LoadMetadata()
		if err != nil {
			return err
		}
		if len(meta) == 0 {
			fmt.Fprintln(a.out, "peers: 0")
			return nil
		}
		for _, peer := range meta {
			if peer.Status == p2p.PeerStatusBad && !*includeBad {
				continue
			}
			if err := p2p.SyncFromPeerWithProfileAndMaxDepth(a.paths, peer.URL, a.out, a.profile, *maxDepth); err != nil {
				if *allowReorg {
					if !*yes {
						return errors.New("refusing to apply reorg without --yes")
					}
					res, reorgErr := p2p.ApplyReorgWithProfile(a.paths, peer.URL, *maxDepth, true, a.profile)
					if reorgErr != nil {
						return reorgErr
					}
					printReorgResult(a.out, res)
					continue
				}
				return err
			}
		}
		return nil
	case "check":
		if len(args) != 2 {
			return errors.New("usage: peer check <url>")
		}
		hs, err := p2p.CheckPeerWithProfile(a.paths, args[1], a.profile)
		if err != nil {
			return err
		}
		fmt.Fprintln(a.out, "peer ok")
		fmt.Fprintf(a.out, "url: %s\n", args[1])
		fmt.Fprintf(a.out, "node id: %s\n", hs.NodeID)
		fmt.Fprintf(a.out, "network id: %s\n", hs.NetworkID)
		fmt.Fprintf(a.out, "chain id: %d\n", hs.ChainID)
		fmt.Fprintf(a.out, "height: %d\n", hs.Height)
		fmt.Fprintf(a.out, "tip hash: %s\n", hs.TipHash)
		return nil
	case "status":
		meta, err := store.LoadMetadata()
		if err != nil {
			return err
		}
		active, bad := 0, 0
		for _, peer := range meta {
			_, err := p2p.CheckPeerWithProfile(a.paths, peer.URL, a.profile)
			if err != nil {
				bad++
			} else {
				active++
			}
		}
		meta, _ = store.LoadMetadata()
		fmt.Fprintf(a.out, "checked peers: %d\n", len(meta))
		fmt.Fprintf(a.out, "active: %d\n", active)
		unknown := len(meta) - active - bad
		if unknown < 0 {
			unknown = 0
		}
		fmt.Fprintf(a.out, "bad: %d\n", bad)
		fmt.Fprintf(a.out, "unknown: %d\n", unknown)
		for _, peer := range meta {
			fmt.Fprintf(a.out, "url=%s status=%s height=%d tip_hash=%s score=%d source=%s last_seen=%s last_status_check=%s latency_ms=%d reason=%q", peer.URL, peer.Status, peer.LastHeight, peer.LastTipHash, peer.Score, peer.Source, peer.LastSeenAt, peer.LastStatusCheckAt, peer.LastLatencyMS, peer.LastScoreReason)
			if peer.LastError != "" {
				fmt.Fprintf(a.out, " error=%q", peer.LastError)
			}
			fmt.Fprintln(a.out)
		}
		return nil
	case "health":
		meta, err := store.LoadMetadata()
		if err != nil {
			return err
		}
		active, seedCount, bestHeight := 0, 0, uint64(0)
		for _, peer := range meta {
			if peer.Status == p2p.PeerStatusActive {
				active++
			}
			if strings.Contains(peer.Source, "seed") {
				seedCount++
			}
			if peer.LastHeight > bestHeight {
				bestHeight = peer.LastHeight
			}
		}
		tip := types.Block{}
		if bc, closeFn, err := a.openInitializedChain(); err == nil {
			tip, _ = bc.Tip()
			closeFn()
		}
		fmt.Fprintf(a.out, "network id: %s\n", a.profile.NetworkID)
		fmt.Fprintf(a.out, "chain id: %d\n", a.profile.ChainID)
		fmt.Fprintf(a.out, "genesis hash: %s\n", chain.GenesisBlockForNetwork(a.profile).Hash)
		fmt.Fprintf(a.out, "local height: %d\n", tip.Height)
		fmt.Fprintf(a.out, "local tip: %s\n", tip.Hash)
		fmt.Fprintf(a.out, "known peers: %d\n", len(meta))
		fmt.Fprintf(a.out, "active peers: %d\n", active)
		fmt.Fprintf(a.out, "seed peers: %d\n", seedCount)
		fmt.Fprintf(a.out, "best peer height: %d\n", bestHeight)
		return nil
	case "seeds":
		meta, err := store.LoadMetadata()
		if err != nil {
			return err
		}
		count := 0
		for _, peer := range meta {
			if strings.Contains(peer.Source, "seed") || strings.Contains(peer.Source, "profile") || strings.Contains(peer.Source, "file") || strings.Contains(peer.Source, "cli") {
				count++
				fmt.Fprintf(a.out, "url=%s source=%s status=%s score=%d height=%d\n", peer.URL, peer.Source, peer.Status, peer.Score, peer.LastHeight)
			}
		}
		fmt.Fprintf(a.out, "seed peers: %d\n", count)
		return nil
	case "discover":
		fs := flag.NewFlagSet("peer discover", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		limit := fs.Int("limit", p2p.DefaultMaxDiscoveredPeers, "max discovered peers per peer")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		targets := []string{}
		if fs.NArg() == 1 {
			targets = append(targets, fs.Arg(0))
		} else if fs.NArg() > 1 {
			return errors.New("usage: peer discover [peer-url]")
		} else {
			meta, err := store.LoadMetadata()
			if err != nil {
				return err
			}
			for _, peer := range p2p.SelectPeers(meta, false, 8) {
				targets = append(targets, peer.URL)
			}
		}
		if len(targets) == 0 {
			fmt.Fprintln(a.out, "discovered peers: 0")
			return nil
		}
		total := 0
		self := ""
		if info, locked := p2p.IsLocked(a.paths.Lock); locked {
			self = info.P2PAdvertise
		}
		for _, target := range targets {
			result, err := p2p.DiscoverFromPeer(a.paths, target, a.profile, self, *limit)
			if err != nil {
				fmt.Fprintf(a.out, "peer=%s discovered=0 error=%q\n", target, err.Error())
				continue
			}
			total += result.Added
			fmt.Fprintf(a.out, "peer=%s checked=%d added=%d skipped=%d rejected=%d\n", result.Peer, result.Checked, result.Added, result.Skipped, result.Rejected)
			for _, discovered := range result.Discovered {
				fmt.Fprintf(a.out, "discovered: %s\n", discovered)
			}
		}
		fmt.Fprintf(a.out, "discovered peers: %d\n", total)
		return nil
	default:
		return fmt.Errorf("unknown peer command %q", args[0])
	}
}

func (a App) upstream(args []string) error {
	if len(args) == 0 {
		return errors.New("upstream command is required")
	}
	var upstreamPeer multiStringFlag
	fs := flag.NewFlagSet("upstream "+args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Var(&upstreamPeer, "upstream-peer", "upstream peer URL; may be repeated")
	upstreamPeersText := fs.String("upstream-peers", os.Getenv("IDR_UPSTREAM_PEERS"), "comma-separated upstream peer URLs")
	upstreamFile := fs.String("upstream-file", "", "upstream peer file")
	defaultMaxDepth, err := config.MaxReorgDepthFromEnv(a.profile)
	if err != nil {
		return err
	}
	maxDepth := fs.Uint64("max-reorg-depth", defaultMaxDepth, "max reorg depth")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	upstreams, err := collectUpstreamPeers(strings.Join(upstreamPeer, ","), *upstreamPeersText, *upstreamFile)
	if err != nil {
		return err
	}
	switch args[0] {
	case "list", "status":
		fmt.Fprintf(a.out, "upstream peers: %d\n", len(upstreams))
		for _, peer := range upstreams {
			fmt.Fprintf(a.out, "upstream peer: %s\n", peer)
		}
		return nil
	case "push":
		peer := ""
		if fs.NArg() == 1 {
			peer = fs.Arg(0)
		} else if len(upstreams) == 1 {
			peer = upstreams[0]
		}
		if peer == "" {
			return errors.New("usage: upstream push <peer-url>")
		}
		result, err := p2p.BackfillToPeer(a.paths, peer, a.profile, *maxDepth)
		printUpstreamResult(a.out, result)
		return err
	case "push-all":
		results := p2p.BackfillToPeers(a.paths, upstreams, a.profile, *maxDepth)
		for _, result := range results {
			printUpstreamResult(a.out, result)
		}
		return nil
	default:
		return fmt.Errorf("unknown upstream command %q", args[0])
	}
}

func (a App) nodeStatus() error {
	info, locked := p2p.IsLocked(a.paths.Lock)
	bc, closeFn, err := a.openInitializedChain()
	if err != nil {
		return err
	}
	defer closeFn()
	tip, err := bc.Tip()
	if err != nil {
		return err
	}
	pending, _ := mempool.New(a.paths.Mempool).Load()
	peers, _ := p2p.NewPeerStore(a.paths.Peers).Load()
	fmt.Fprintf(a.out, "datadir: %s\n", a.paths.DataDir)
	fmt.Fprintf(a.out, "rpc: %s\n", info.RPC)
	fmt.Fprintf(a.out, "p2p: %s\n", info.P2P)
	fmt.Fprintf(a.out, "p2p advertise: %s\n", info.P2PAdvertise)
	fmt.Fprintf(a.out, "height: %d\n", tip.Height)
	fmt.Fprintf(a.out, "tip hash: %s\n", tip.Hash)
	fmt.Fprintf(a.out, "peers: %d\n", len(peers))
	fmt.Fprintf(a.out, "mempool pending: %d\n", len(pending))
	fmt.Fprintf(a.out, "locked: %t\n", locked)
	return nil
}

func (a App) nodeCompare(args []string) error {
	fs := flag.NewFlagSet("node compare", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	peer := fs.String("peer", "", "peer RPC URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *peer == "" {
		return errors.New("usage: node compare --peer <rpc-url>")
	}
	local, err := a.localChainInfoMap()
	if err != nil {
		return err
	}
	peerInfo, err := remoteGetMap(*peer, "/chain/info")
	if err != nil {
		return err
	}
	printCompare(a.out, local, peerInfo)
	return nil
}

func (a App) localChainInfoMap() (map[string]any, error) {
	bc, closeFn, err := a.openInitializedChain()
	if err != nil {
		return nil, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return nil, err
	}
	tip := blocks[len(blocks)-1]
	pending, _ := mempool.New(a.paths.Mempool).Load()
	nextDifficulty := chain.CalculateNextDifficultyWithParams(blocks, difficultyParams(a.profile))
	return map[string]any{
		"network_id":         a.profile.NetworkID,
		"chain_id":           float64(a.profile.ChainID),
		"genesis_hash":       chain.GenesisBlockForNetwork(a.profile).Hash,
		"protocol_version":   float64(a.profile.ProtocolVersion),
		"height":             float64(tip.Height),
		"tip_hash":           tip.Hash,
		"difficulty":         float64(nextDifficulty),
		"tip_difficulty":     float64(tip.Difficulty),
		"next_difficulty":    float64(nextDifficulty),
		"total_supply":       amount.Format(ledger.TotalSupply(blocks)) + " " + config.Ticker,
		"circulating_supply": amount.Format(ledger.CirculatingSupplyWithProfile(blocks, consensusParams(a.profile), a.profile)) + " " + config.Ticker,
		"coinbase_maturity":  float64(consensusParams(a.profile).CoinbaseMaturity),
		"cumulative_work":    float64(chain.CalculateCumulativeWork(blocks)),
		"pending_tx_count":   float64(len(pending)),
	}, nil
}

func (a App) createPendingTransaction(from, to, amountText string) (types.Transaction, error) {
	if err := crypto.ValidateAddressForNetwork(from, a.profile); err != nil {
		return types.Transaction{}, fmt.Errorf("invalid sender address: %w", err)
	}
	if err := crypto.ValidateAddressForNetwork(to, a.profile); err != nil {
		return types.Transaction{}, fmt.Errorf("invalid recipient address: %w", err)
	}
	if from == to {
		return types.Transaction{}, errors.New("sender and recipient must differ")
	}
	txAmount, err := amount.Parse(amountText)
	if err != nil {
		return types.Transaction{}, err
	}
	bc, closeFn, err := a.openInitializedChain()
	if err != nil {
		return types.Transaction{}, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return types.Transaction{}, err
	}
	walletStore := wallet.NewStore(a.paths.Wallets)
	w, ok, err := walletStore.Find(from)
	if err != nil {
		return types.Transaction{}, err
	}
	if !ok {
		return types.Transaction{}, errors.New("local wallet not found for sender")
	}
	pending, err := mempool.New(a.paths.Mempool).Load()
	if err != nil {
		return types.Transaction{}, err
	}
	matureLedger, err := ledger.ReplayMatureWithProfile(blocks, consensusParams(a.profile), a.profile)
	if err != nil {
		return types.Transaction{}, err
	}
	details := matureLedger.BalanceDetails(from, pending, blocks[len(blocks)-1].Height)
	if details.Spendable < txAmount {
		return types.Transaction{}, fmt.Errorf("insufficient mature balance: spendable %s %s, required %s %s, active stake %s %s, unlocking stake %s %s", amount.Format(details.Spendable), config.Ticker, amount.Format(txAmount), config.Ticker, amount.Format(details.ActiveStake), config.Ticker, amount.Format(details.UnlockingStake), config.Ticker)
	}
	tx := types.NewUnsignedTransaction(from, to, txAmount, 0, matureLedger.Nonce(from)+pendingFromCount(pending, from)+1)
	if err := w.SignTransaction(&tx); err != nil {
		return types.Transaction{}, err
	}
	return tx, nil
}

func (a App) createStakeLockTx(address, amountText string) (types.Transaction, error) {
	if err := crypto.ValidateAddressForNetwork(address, a.profile); err != nil {
		return types.Transaction{}, err
	}
	txAmount, err := amount.Parse(amountText)
	if err != nil {
		return types.Transaction{}, err
	}
	params := consensusParams(a.profile)
	if !params.Staking.Enabled {
		return types.Transaction{}, errors.New("staking disabled")
	}
	if txAmount < params.Staking.MinStakeAmount {
		return types.Transaction{}, errors.New("invalid stake lock: amount below minimum")
	}
	blocks, err := a.loadBlocks()
	if err != nil {
		return types.Transaction{}, err
	}
	w, ok, err := wallet.NewStore(a.paths.Wallets).Find(address)
	if err != nil {
		return types.Transaction{}, err
	}
	if !ok {
		return types.Transaction{}, errors.New("local wallet not found for address")
	}
	pending, err := mempool.New(a.paths.Mempool).Load()
	if err != nil {
		return types.Transaction{}, err
	}
	matureLedger, err := ledger.ReplayMatureWithProfile(blocks, params, a.profile)
	if err != nil {
		return types.Transaction{}, err
	}
	details := matureLedger.BalanceDetails(address, pending, currentHeight(blocks))
	if details.Spendable < txAmount {
		return types.Transaction{}, errors.New("invalid stake lock: insufficient mature spendable balance")
	}
	tx := types.NewStakeLockTransaction(address, txAmount, matureLedger.Nonce(address)+pendingFromCount(pending, address)+1)
	if err := w.SignTransaction(&tx); err != nil {
		return types.Transaction{}, err
	}
	tx.StakeID = tx.ID
	return tx, nil
}

func (a App) createStakeUnlockTx(address, stakeID string) (types.Transaction, uint64, error) {
	if err := crypto.ValidateAddressForNetwork(address, a.profile); err != nil {
		return types.Transaction{}, 0, err
	}
	if stakeID == "" {
		return types.Transaction{}, 0, errors.New("stake-id is required")
	}
	params := consensusParams(a.profile)
	blocks, err := a.loadBlocks()
	if err != nil {
		return types.Transaction{}, 0, err
	}
	state, err := staking.Replay(blocks, params.Staking)
	if err != nil {
		return types.Transaction{}, 0, err
	}
	record, ok := state.Find(stakeID, currentHeight(blocks))
	if !ok {
		return types.Transaction{}, 0, errors.New("invalid stake unlock: stake not found")
	}
	if record.OwnerAddress != address {
		return types.Transaction{}, 0, errors.New("invalid stake unlock: owner mismatch")
	}
	if record.Status != staking.StatusActive {
		return types.Transaction{}, 0, fmt.Errorf("invalid stake unlock: stake %s", record.Status)
	}
	w, ok, err := wallet.NewStore(a.paths.Wallets).Find(address)
	if err != nil {
		return types.Transaction{}, 0, err
	}
	if !ok {
		return types.Transaction{}, 0, errors.New("local wallet not found for address")
	}
	pending, err := mempool.New(a.paths.Mempool).Load()
	if err != nil {
		return types.Transaction{}, 0, err
	}
	if _, exists := staking.PendingUnlockIDs(pending, address)[stakeID]; exists {
		return types.Transaction{}, 0, errors.New("invalid stake unlock: already pending")
	}
	matureLedger, err := ledger.ReplayMatureWithProfile(blocks, params, a.profile)
	if err != nil {
		return types.Transaction{}, 0, err
	}
	tx := types.NewStakeUnlockTransaction(address, stakeID, matureLedger.Nonce(address)+pendingFromCount(pending, address)+1)
	if err := w.SignTransaction(&tx); err != nil {
		return types.Transaction{}, 0, err
	}
	return tx, currentHeight(blocks) + 1 + params.Staking.UnbondingPeriodBlocks, nil
}

func (a App) loadBlocks() ([]types.Block, error) {
	bc, closeFn, err := a.openInitializedChain()
	if err != nil {
		return nil, err
	}
	defer closeFn()
	return bc.Blocks()
}

func currentHeight(blocks []types.Block) uint64 {
	if len(blocks) == 0 {
		return 0
	}
	return blocks[len(blocks)-1].Height
}

func pendingFromCount(pending []types.Transaction, address string) uint64 {
	var count uint64
	for _, tx := range pending {
		if !tx.Coinbase && tx.From == address {
			count++
		}
	}
	return count
}

type txLookupResult struct {
	tx          types.Transaction
	status      string
	blockHeight uint64
}

func (a App) findTransaction(txID string) (txLookupResult, bool, error) {
	bc, closeFn, err := a.openInitializedChain()
	if err != nil {
		return txLookupResult{}, false, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return txLookupResult{}, false, err
	}
	for _, block := range blocks {
		for _, tx := range block.Transactions {
			if tx.ID == txID {
				return txLookupResult{tx: tx, status: "confirmed", blockHeight: block.Height}, true, nil
			}
		}
	}
	pending, err := mempool.New(a.paths.Mempool).Load()
	if err != nil {
		return txLookupResult{}, false, err
	}
	for _, tx := range pending {
		if tx.ID == txID {
			return txLookupResult{tx: tx, status: "pending"}, true, nil
		}
	}
	return txLookupResult{}, false, nil
}

func printTransaction(out io.Writer, result txLookupResult) {
	tx := result.tx
	fmt.Fprintf(out, "id: %s\n", tx.ID)
	fmt.Fprintf(out, "status: %s\n", result.status)
	if result.status == "confirmed" {
		fmt.Fprintf(out, "block_height: %d\n", result.blockHeight)
	}
	fmt.Fprintf(out, "from: %s\n", tx.From)
	fmt.Fprintf(out, "to: %s\n", tx.To)
	fmt.Fprintf(out, "amount: %s %s\n", amount.Format(tx.Amount), config.Ticker)
	fmt.Fprintf(out, "fee: %s %s\n", amount.Format(tx.Fee), config.Ticker)
	fmt.Fprintf(out, "nonce: %d\n", tx.Nonce)
	fmt.Fprintf(out, "coinbase: %t\n", tx.Coinbase)
}

func (a App) printAddressInspection(address string) error {
	info, err := a.inspectAddress(address)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.out, "address: %s\n", info.address)
	fmt.Fprintf(a.out, "format: %s\n", addressFormat(info.address))
	fmt.Fprintf(a.out, "network: %s\n", a.profile.Name)
	fmt.Fprintln(a.out, "key curve: secp256k1")
	if crypto.IsLegacyDevAddress(info.address) {
		fmt.Fprintln(a.out, "warning: legacy idr1 dev address; localnet compatibility only")
	}
	fmt.Fprintf(a.out, "exists: %t\n", info.exists)
	fmt.Fprintf(a.out, "confirmed balance: %s %s\n", amount.Format(info.confirmedBalance), config.Ticker)
	fmt.Fprintf(a.out, "mature balance: %s %s\n", amount.Format(info.matureBalance), config.Ticker)
	fmt.Fprintf(a.out, "immature balance: %s %s\n", amount.Format(info.immatureBalance), config.Ticker)
	fmt.Fprintf(a.out, "spendable balance: %s %s\n", amount.Format(info.spendableBalance), config.Ticker)
	fmt.Fprintf(a.out, "confirmed nonce: %d\n", info.confirmedNonce)
	fmt.Fprintf(a.out, "pending outgoing tx: %d\n", info.pendingOutgoingCount)
	fmt.Fprintf(a.out, "pending outgoing amount: %s %s\n", amount.Format(info.pendingOutgoingAmount), config.Ticker)
	fmt.Fprintf(a.out, "pending incoming tx: %d\n", info.pendingIncomingCount)
	fmt.Fprintf(a.out, "pending incoming amount: %s %s\n", amount.Format(info.pendingIncomingAmount), config.Ticker)
	return nil
}

type addressInspection struct {
	address               string
	exists                bool
	confirmedBalance      uint64
	matureBalance         uint64
	immatureBalance       uint64
	spendableBalance      uint64
	confirmedNonce        uint64
	pendingOutgoingCount  uint64
	pendingOutgoingAmount uint64
	pendingIncomingCount  uint64
	pendingIncomingAmount uint64
}

func (a App) inspectAddress(address string) (addressInspection, error) {
	if err := crypto.ValidateAddressForNetwork(address, a.profile); err != nil {
		return addressInspection{}, err
	}
	bc, closeFn, err := a.openInitializedChain()
	if err != nil {
		return addressInspection{}, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return addressInspection{}, err
	}
	l, err := bc.Ledger()
	if err != nil {
		return addressInspection{}, err
	}
	_, exists, err := wallet.NewStore(a.paths.Wallets).Find(address)
	if err != nil {
		return addressInspection{}, err
	}
	pending, err := mempool.New(a.paths.Mempool).Load()
	if err != nil {
		return addressInspection{}, err
	}
	details, err := ledger.BalanceDetailsForWithProfile(address, blocks, pending, consensusParams(a.profile), a.profile)
	if err != nil {
		return addressInspection{}, err
	}
	info := addressInspection{
		address:          address,
		exists:           exists,
		confirmedBalance: details.Confirmed,
		matureBalance:    details.Mature,
		immatureBalance:  details.Immature,
		spendableBalance: details.Spendable,
		confirmedNonce:   l.Nonce(address),
	}
	for _, tx := range pending {
		if tx.Coinbase {
			continue
		}
		if tx.From == address {
			info.pendingOutgoingCount++
			info.pendingOutgoingAmount += tx.Amount + tx.Fee
		}
		if tx.To == address {
			info.pendingIncomingCount++
			info.pendingIncomingAmount += tx.Amount
		}
	}
	return info, nil
}

func addressFormat(addr string) string {
	if crypto.IsLegacyDevAddress(addr) {
		return "legacy-dev"
	}
	return "base58check"
}

func (a App) openChain() (*chain.Blockchain, func(), error) {
	store, err := storage.OpenBolt(a.paths.DB)
	if err != nil {
		return nil, nil, err
	}
	return chain.New(store), func() { _ = store.Close() }, nil
}

func (a App) knownPeers() []string {
	peers, err := p2p.NewPeerStore(a.paths.Peers).Load()
	if err != nil {
		log.Printf("load peers failed: %v", err)
		return nil
	}
	return peers
}

func (a App) knownPeerMetadata() []p2p.PeerMetadata {
	peers, err := p2p.NewPeerStore(a.paths.Peers).LoadMetadata()
	if err != nil {
		log.Printf("load peers failed: %v", err)
		return nil
	}
	return peers
}

func (a App) broadcastTx(tx types.Transaction) {
	p2p.BroadcastTxToPeersWithProfile(a.paths, a.profile, a.knownPeerMetadata(), tx)
}

func (a App) broadcastBlock(block types.Block) {
	p2p.BroadcastBlockToPeersWithProfile(a.paths, a.profile, a.knownPeerMetadata(), block)
}

func (a App) lockChainMutation() func() {
	if a.chainMu == nil {
		return func() {}
	}
	a.chainMu.Lock()
	return a.chainMu.Unlock
}

func (a App) syncLoop(interval time.Duration, verbose bool, state *nodestate.Store, maxReorgDepth uint64, upstreamPeers []string) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	tracker := newPeerSyncTracker()
	for {
		time.Sleep(interval + time.Duration(rand.Intn(1000))*time.Millisecond)
		for _, peer := range a.knownPeerMetadata() {
			if peer.Status == p2p.PeerStatusBad {
				continue
			}
			if !tracker.tryStart(peer.URL) {
				if verbose {
					log.Printf("sync skipped peer=%s reason=already_running", peer.URL)
				}
				continue
			}
			go func(peerURL string) {
				defer tracker.done(peerURL)
				unlock := a.lockChainMutation()
				defer unlock()
				if err := p2p.SyncFromPeerWithProfileAndMaxDepth(a.paths, peerURL, nil, a.profile, maxReorgDepth); err != nil {
					if tracker.shouldLogError(peerURL, err.Error(), time.Now()) {
						log.Printf("sync failed peer=%s error=%v", peerURL, err)
					}
					return
				}
				if state != nil {
					if err := state.Refresh(a.paths); err != nil {
						log.Printf("runtime state refresh failed after sync peer=%s error=%v", peerURL, err)
					}
				}
				if tracker.markRecovered(peerURL) {
					log.Printf("peer recovered url=%s", peerURL)
				} else if verbose {
					log.Printf("sync checked peer=%s", peerURL)
				}
				a.scheduleUpstreamBackfill(upstreamPeers, maxReorgDepth)
			}(peer.URL)
		}
	}
}

func (a App) bootstrapPeers(seedPeers []string, selfURL string, state *nodestate.Store, maxReorgDepth uint64, upstreamPeers []string) {
	if len(seedPeers) == 0 {
		return
	}
	time.Sleep(500 * time.Millisecond)
	attempted, accepted, failed, discovered := 0, 0, 0, 0
	for _, seed := range seedPeers {
		attempted++
		normalized, err := p2p.NormalizePeerURL(seed)
		if err != nil {
			failed++
			log.Printf("seed rejected seed=%s reason=%q", normalized, err.Error())
			continue
		}
		if selfURL != "" {
			if self, err := p2p.NormalizePeerURL(selfURL); err == nil && self == normalized {
				log.Printf("seed skipped seed=%s reason=self", normalized)
				continue
			}
		}
		hs, err := p2p.CheckPeerWithProfile(a.paths, normalized, a.profile)
		if err != nil {
			failed++
			log.Printf("seed rejected seed=%s reason=%q", normalized, err.Error())
			continue
		}
		accepted++
		log.Printf("seed accepted seed=%s height=%d tip=%s", normalized, hs.Height, hs.TipHash)
		unlock := a.lockChainMutation()
		err = p2p.SyncFromPeerWithProfileAndMaxDepth(a.paths, normalized, nil, a.profile, maxReorgDepth)
		unlock()
		if err != nil {
			log.Printf("seed sync checked seed=%s result=%q", normalized, err.Error())
		} else {
			log.Printf("seed sync accepted seed=%s", normalized)
			if state != nil {
				if err := state.Refresh(a.paths); err != nil {
					log.Printf("runtime state refresh failed after seed sync seed=%s error=%v", normalized, err)
				}
			}
			a.scheduleUpstreamBackfill(upstreamPeers, maxReorgDepth)
		}
		result, err := p2p.DiscoverFromPeer(a.paths, normalized, a.profile, selfURL, p2p.DefaultMaxDiscoveredPeers)
		if err != nil {
			log.Printf("seed discovery failed seed=%s reason=%q", normalized, err.Error())
			continue
		}
		discovered += result.Added
		log.Printf("seed discovery seed=%s checked=%d accepted=%d rejected=%d skipped=%d", normalized, result.Checked, result.Added, result.Rejected, result.Skipped)
	}
	log.Printf("peer bootstrap summary attempted=%d accepted=%d failed=%d discovered=%d", attempted, accepted, failed, discovered)
}

func (a App) scheduleUpstreamBackfill(upstreamPeers []string, maxReorgDepth uint64) {
	if len(upstreamPeers) == 0 {
		return
	}
	peers := append([]string(nil), upstreamPeers...)
	go p2p.BackfillToPeers(a.paths, peers, a.profile, maxReorgDepth)
}

type peerSyncTracker struct {
	mu       sync.Mutex
	running  map[string]struct{}
	lastLogs map[string]time.Time
	failed   map[string]struct{}
}

func newPeerSyncTracker() *peerSyncTracker {
	return &peerSyncTracker{
		running:  make(map[string]struct{}),
		lastLogs: make(map[string]time.Time),
		failed:   make(map[string]struct{}),
	}
}

func (t *peerSyncTracker) tryStart(peer string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.running[peer]; ok {
		return false
	}
	t.running[peer] = struct{}{}
	return true
}

func (t *peerSyncTracker) done(peer string) {
	t.mu.Lock()
	delete(t.running, peer)
	t.mu.Unlock()
}

func (t *peerSyncTracker) shouldLogError(peer, _ string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failed[peer] = struct{}{}
	last, ok := t.lastLogs[peer]
	if !ok || now.Sub(last) >= time.Minute {
		t.lastLogs[peer] = now
		return true
	}
	return false
}

func (t *peerSyncTracker) markRecovered(peer string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.failed[peer]; !ok {
		return false
	}
	delete(t.failed, peer)
	delete(t.lastLogs, peer)
	return true
}

func (a App) checkLock(args []string) error {
	if a.ignoreLock || len(args) == 0 {
		return nil
	}
	if len(args) > 1 && args[0] == "dev" && args[1] == "reset" {
		return nil
	}
	if isReadOnlyCommand(args) {
		return nil
	}
	if _, locked := p2p.IsLocked(a.paths.Lock); locked {
		return a.lockedError(args)
	}
	return nil
}

func isReadOnlyCommand(args []string) bool {
	switch args[0] {
	case "chain":
		return len(args) > 1 && (args[1] == "info" || args[1] == "print" || args[1] == "validate")
	case "mining":
		return len(args) > 1 && (args[1] == "status" || args[1] == "difficulty" || args[1] == "blocks")
	case "dev":
		return len(args) > 1 && args[1] == "inspect"
	case "balance":
		return true
	case "wallet":
		return len(args) > 1 && args[1] == "list"
	case "peer":
		return len(args) > 1 && args[1] == "list"
	case "node":
		return len(args) > 1 && (args[1] == "status" || args[1] == "compare")
	case "p2p":
		return len(args) > 1 && (args[1] == "ping" || args[1] == "debug")
	case "debug":
		return len(args) > 1 && args[1] == "locks"
	case "tx", "address":
		return true
	case "stake":
		return len(args) > 1 && (args[1] == "info" || args[1] == "list" || args[1] == "status")
	default:
		return false
	}
}

func (a App) lockedError(args []string) error {
	info, _ := p2p.ReadLock(a.paths.Lock)
	rpcURL := normalizeRPCURL(info.RPC)
	return fmt.Errorf("datadir is locked by running node\ndatadir: %s\nrpc: %s\np2p: %s\nuse:\ngo run ./node/cmd/deskachain --rpc-url %s %s", a.paths.DataDir, info.RPC, info.P2P, rpcURL, suggestedRemoteCommand(args))
}

func (a App) runRemote(args []string) error {
	switch args[0] {
	case "dev":
		if len(args) > 1 && args[1] == "inspect" {
			return errors.New("remote mode not supported for dev inspect")
		}
	case "chain":
		if len(args) > 1 && args[1] == "info" {
			info, err := remoteGetMap(a.rpcURL, "/chain/info")
			if err != nil {
				return err
			}
			printRemoteChainInfo(a.out, info)
			return nil
		}
		if len(args) > 1 && args[1] == "validate" {
			info, err := remoteGetMap(a.rpcURL, "/chain/validate")
			if err != nil {
				return err
			}
			if valid, _ := info["valid"].(bool); valid {
				fmt.Fprintln(a.out, "chain valid")
				fmt.Fprintf(a.out, "height: %.0f\n", info["height"])
				fmt.Fprintf(a.out, "blocks: %.0f\n", info["blocks"])
				fmt.Fprintf(a.out, "total supply: %s\n", info["total_supply"])
				return nil
			}
			return fmt.Errorf("chain invalid: %v", info["error"])
		}
		if len(args) > 1 && args[1] == "difficulty" {
			info, err := remoteGetMap(a.rpcURL, "/chain/difficulty")
			if err != nil {
				return err
			}
			printRemoteChainDifficulty(a.out, info)
			return nil
		}
		if len(args) > 1 && args[1] == "locator" {
			info, err := remoteGetMap(a.rpcURL, "/chain/locator")
			if err != nil {
				return err
			}
			printLocator(a.out, locatorFromMap(info))
			return nil
		}
		if len(args) > 1 && args[1] == "common-ancestor" {
			return a.remoteCommonAncestor(args[2:])
		}
	case "mining":
		if len(args) > 1 && (args[1] == "status" || args[1] == "difficulty") {
			info, err := remoteGetMap(a.rpcURL, "/mining/"+args[1])
			if err != nil {
				return err
			}
			printMiningStatus(a.out, info)
			return nil
		}
		if len(args) > 1 && args[1] == "blocks" {
			fs := flag.NewFlagSet("mining blocks", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			limit := fs.Int("limit", 30, "recent block limit")
			if err := fs.Parse(args[2:]); err != nil {
				return err
			}
			info, err := remoteGetMap(a.rpcURL, fmt.Sprintf("/mining/blocks?limit=%d", *limit))
			if err != nil {
				return err
			}
			printMiningBlocks(a.out, info)
			return nil
		}
	case "balance":
		if len(args) != 2 {
			return errors.New("usage: balance <address>")
		}
		info, err := remoteGetMap(a.rpcURL, "/balance/"+args[1])
		if err != nil {
			return err
		}
		printRemoteBalanceDetails(a.out, info)
		return nil
	case "mempool":
		if len(args) > 1 && args[1] == "list" {
			endpoint := "/mempool"
			if len(args) > 2 && args[2] == "--detail" {
				endpoint = "/mempool?detail=true"
			}
			info, err := remoteGetMap(a.rpcURL, endpoint)
			if err != nil {
				return err
			}
			printRemoteMempool(a.out, info)
			return nil
		}
	case "send":
		return a.remoteSend(args[1:])
	case "faucet":
		return a.remoteFaucet(args[1:])
	case "mine":
		return a.remoteMine(args[1:])
	case "peer":
		return a.remotePeer(args[1:])
	case "p2p":
		return a.remoteP2P(args[1:])
	case "fork":
		return a.remoteFork(args[1:])
	case "reorg":
		return a.remoteReorg(args[1:])
	case "debug":
		if len(args) > 1 && args[1] == "locks" {
			info, err := remoteGetMap(a.rpcURL, "/debug/locks")
			if err != nil {
				return err
			}
			printRemoteDebugLocks(a.out, info)
			return nil
		}
	case "service":
		return a.remoteService(args[1:])
	case "stake":
		return a.remoteStake(args[1:])
	case "tx":
		if len(args) == 3 && args[1] == "get" {
			info, err := remoteGetMap(a.rpcURL, "/tx/"+args[2])
			if err != nil {
				return err
			}
			printRemoteTx(a.out, info)
			return nil
		}
	case "address":
		if len(args) == 3 && args[1] == "validate" {
			info, err := remoteGetMap(a.rpcURL, "/address/"+args[2])
			if err != nil {
				return err
			}
			if valid, _ := info["valid"].(bool); valid {
				fmt.Fprintln(a.out, "address valid")
			} else {
				fmt.Fprintln(a.out, "address invalid")
			}
			return nil
		}
	case "wallet":
		if len(args) > 1 && args[1] == "new" {
			info, err := remotePostMap(a.rpcURL, "/wallet/new", nil)
			if err != nil {
				return err
			}
			fmt.Fprintln(a.out, info["address"])
			return nil
		}
		if len(args) > 1 && args[1] == "list" {
			info, err := remoteGetMap(a.rpcURL, "/wallets")
			if err != nil {
				return err
			}
			printRemoteWalletList(a.out, info)
			return nil
		}
		if len(args) > 1 && args[1] == "inspect" {
			fs := flag.NewFlagSet("wallet inspect", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			addr := fs.String("address", "", "wallet address")
			if err := fs.Parse(args[2:]); err != nil {
				return err
			}
			if *addr == "" && fs.NArg() == 1 {
				*addr = fs.Arg(0)
			}
			info, err := remoteGetMap(a.rpcURL, "/address/"+*addr)
			if err != nil {
				return err
			}
			printRemoteAddressInspect(a.out, info)
			return nil
		}
	case "node":
		if len(args) > 1 && args[1] == "status" {
			info, err := remoteGetMap(a.rpcURL, "/node/status")
			if err != nil {
				return err
			}
			printRemoteNodeStatus(a.out, info)
			return nil
		}
		if len(args) > 1 && args[1] == "compare" {
			return a.remoteCompare(args[2:])
		}
	}
	return errors.New("remote mode not supported for this command yet")
}

func (a App) remoteSend(args []string) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	from := fs.String("from", "", "sender")
	to := fs.String("to", "", "recipient")
	amt := fs.String("amount", "", "amount")
	if err := fs.Parse(args); err != nil {
		return err
	}
	info, err := remotePostMap(a.rpcURL, "/send", map[string]string{"from": *from, "to": *to, "amount": *amt})
	if err != nil {
		return err
	}
	fmt.Fprintln(a.out, "tx created")
	fmt.Fprintf(a.out, "id: %s\n", info["id"])
	fmt.Fprintf(a.out, "from: %s\n", info["from"])
	fmt.Fprintf(a.out, "to: %s\n", info["to"])
	fmt.Fprintf(a.out, "amount: %s\n", info["amount"])
	fmt.Fprintf(a.out, "fee: %s\n", info["fee"])
	fmt.Fprintf(a.out, "nonce: %.0f\n", info["nonce"])
	printRemoteBroadcastSummary(a.out, "tx broadcast", info["broadcast"])
	fmt.Fprintln(a.out, "status: pending")
	return nil
}

func (a App) remoteFaucet(args []string) error {
	if len(args) == 0 {
		return errors.New("faucet command is required")
	}
	switch args[0] {
	case "info":
		info, err := remoteGetMap(a.rpcURL, "/faucet/info")
		if err != nil {
			return err
		}
		printRemoteFaucetInfo(a.out, info)
		return nil
	case "request":
		fs := flag.NewFlagSet("faucet request", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "recipient address")
		amountText := fs.String("amount", "", "optional faucet amount")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		body := map[string]string{"address": *address}
		if *amountText != "" {
			body["amount"] = *amountText
		}
		info, err := remotePostMap(a.rpcURL, "/faucet/request", body)
		if err != nil {
			return err
		}
		printRemoteFaucetRequest(a.out, info)
		return nil
	default:
		return fmt.Errorf("unknown faucet command %q", args[0])
	}
}

func (a App) remoteMine(args []string) error {
	if len(args) > 0 && args[0] == "status" {
		info, err := remoteGetMap(a.rpcURL, "/mine/status")
		if err != nil {
			return err
		}
		printRemoteMineStatus(a.out, info)
		return nil
	}
	fs := flag.NewFlagSet("mine", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addr := fs.String("address", "", "miner")
	blocks := fs.Uint64("blocks", 1, "blocks")
	timeout := fs.Duration("timeout", 5*time.Minute, "request timeout")
	maxNonce := fs.Uint64("max-nonce", 0, "maximum nonce, 0 means unlimited")
	mineVerbose := fs.Bool("mine-verbose", false, "print mining progress in node logs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	info, err := remotePostMapWithTimeout(a.rpcURL, "/mine", map[string]any{"address": *addr, "blocks": *blocks, "max_nonce": *maxNonce, "mine_verbose": *mineVerbose}, *timeout)
	if err != nil {
		return err
	}
	for _, item := range info["blocks"].([]any) {
		block := item.(map[string]any)
		fmt.Fprintf(a.out, "mined block height=%.0f hash=%s txs=%.0f reward=%s difficulty=%.0f nonce=%.0f\n", block["height"], block["hash"], block["txs"], block["reward"], block["difficulty"], block["nonce"])
	}
	fmt.Fprintln(a.out, "mining complete")
	fmt.Fprintf(a.out, "mined blocks: %.0f\n", info["mined_blocks"])
	fmt.Fprintf(a.out, "new height: %.0f\n", info["new_height"])
	fmt.Fprintf(a.out, "miner balance: %s\n", info["miner_balance"])
	if value, ok := info["miner_mature_balance"]; ok {
		fmt.Fprintf(a.out, "miner mature balance: %s\n", value)
	}
	if value, ok := info["miner_immature_balance"]; ok {
		fmt.Fprintf(a.out, "miner immature balance: %s\n", value)
	}
	if value, ok := info["miner_spendable_balance"]; ok {
		fmt.Fprintf(a.out, "miner spendable balance: %s\n", value)
	}
	printRemoteBroadcastSummary(a.out, "block broadcast", info["broadcast"])
	return nil
}

func (a App) remoteService(args []string) error {
	if len(args) == 0 {
		return errors.New("service command is required")
	}
	switch args[0] {
	case "register":
		fs := flag.NewFlagSet("service register", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		endpoint := fs.String("endpoint", "", "advertised endpoint")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		info, err := remotePostMap(a.rpcURL, "/service/register", map[string]any{"address": *address, "endpoint": *endpoint, "client_version": "dev", "platform": runtimePlatform()})
		if err != nil {
			return err
		}
		printServiceRegistered(a.out, info)
		return nil
	case "heartbeat":
		fs := flag.NewFlagSet("service heartbeat", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		endpoint := fs.String("endpoint", "", "advertised endpoint")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		info, err := remotePostMap(a.rpcURL, "/service/heartbeat", map[string]any{"address": *address, "endpoint": *endpoint, "client_version": "dev", "platform": runtimePlatform()})
		if err != nil {
			return err
		}
		printServiceHeartbeat(a.out, info)
		return nil
	case "challenge":
		if len(args) < 2 {
			return errors.New("usage: service challenge <create|submit>")
		}
		switch args[1] {
		case "create":
			fs := flag.NewFlagSet("service challenge create", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			address := fs.String("address", "", "owner address")
			if err := fs.Parse(args[2:]); err != nil {
				return err
			}
			info, err := remotePostMap(a.rpcURL, "/service/challenge/create", map[string]any{"address": *address})
			if err != nil {
				return err
			}
			printServiceChallenge(a.out, info)
			return nil
		case "submit":
			fs := flag.NewFlagSet("service challenge submit", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			challengeID := fs.String("challenge-id", "", "challenge id")
			latencyMS := fs.Int64("latency-ms", 0, "latency in milliseconds")
			bytesUp := fs.Int64("bytes-up", 0, "uploaded bytes")
			bytesDown := fs.Int64("bytes-down", 0, "downloaded bytes")
			success := fs.Bool("success", true, "challenge success")
			if err := fs.Parse(args[2:]); err != nil {
				return err
			}
			info, err := remotePostMap(a.rpcURL, "/service/challenge/submit", map[string]any{"challenge_id": *challengeID, "latency_ms": *latencyMS, "bytes_up": *bytesUp, "bytes_down": *bytesDown, "success": *success})
			if err != nil {
				return err
			}
			printServiceChallengeSubmit(a.out, info)
			return nil
		default:
			return fmt.Errorf("unknown service challenge command %q", args[1])
		}
	case "score":
		fs := flag.NewFlagSet("service score", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		info, err := remoteGetMap(a.rpcURL, "/service/score?address="+url.QueryEscape(*address))
		if err != nil {
			return err
		}
		if score, ok := info["score"].(map[string]any); ok {
			printServiceScore(a.out, score)
			return nil
		}
		return errors.New("invalid service score response")
	case "rewards":
		fs := flag.NewFlagSet("service rewards", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		info, err := remoteGetMap(a.rpcURL, "/service/rewards?address="+url.QueryEscape(*address))
		if err != nil {
			return err
		}
		printServiceRewardsMap(a.out, info)
		return nil
	case "list":
		info, err := remoteGetMap(a.rpcURL, "/service/list")
		if err != nil {
			return err
		}
		printServiceListMap(a.out, info)
		return nil
	default:
		return fmt.Errorf("unknown service command %q", args[0])
	}
}

func (a App) remoteStake(args []string) error {
	if len(args) == 0 {
		return errors.New("stake command is required")
	}
	switch args[0] {
	case "info":
		info, err := remoteGetMap(a.rpcURL, "/stake/info")
		if err != nil {
			return err
		}
		printStakeInfoMap(a.out, info)
		return nil
	case "list":
		fs := flag.NewFlagSet("stake list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		path := "/stake/list"
		if *address != "" {
			path += "?address=" + url.QueryEscape(*address)
		}
		info, err := remoteGetMap(a.rpcURL, path)
		if err != nil {
			return err
		}
		printStakeListMap(a.out, info)
		return nil
	case "status":
		fs := flag.NewFlagSet("stake status", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		stakeID := fs.String("stake-id", "", "stake id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		info, err := remoteGetMap(a.rpcURL, "/stake/status?id="+url.QueryEscape(*stakeID))
		if err != nil {
			return err
		}
		printStakeRecordMap(a.out, info)
		return nil
	case "lock":
		fs := flag.NewFlagSet("stake lock", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		amountText := fs.String("amount", "", "amount")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		info, err := remotePostMap(a.rpcURL, "/stake/lock", map[string]string{"address": *address, "amount": *amountText})
		if err != nil {
			return err
		}
		printStakeLockMap(a.out, info)
		return nil
	case "unlock":
		fs := flag.NewFlagSet("stake unlock", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		address := fs.String("address", "", "owner address")
		stakeID := fs.String("stake-id", "", "stake id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		info, err := remotePostMap(a.rpcURL, "/stake/unlock", map[string]string{"address": *address, "stake_id": *stakeID})
		if err != nil {
			return err
		}
		printStakeUnlockMap(a.out, info)
		return nil
	default:
		return fmt.Errorf("unknown stake command %q", args[0])
	}
}

func (a App) remotePeer(args []string) error {
	if len(args) == 0 {
		return errors.New("peer command is required")
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("peer list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		showSource := fs.Bool("source", false, "show peer source")
		jsonOut := fs.Bool("json", false, "print JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		info, err := remoteGetMap(a.rpcURL, "/peers")
		if err != nil {
			return err
		}
		if *jsonOut {
			return json.NewEncoder(a.out).Encode(info)
		}
		peers := info["peers"].([]any)
		fmt.Fprintf(a.out, "peers: %d\n", len(peers))
		for _, peer := range peers {
			if meta, ok := peer.(map[string]any); ok {
				if *showSource {
					fmt.Fprintf(a.out, "url=%s source=%s status=%s score=%.0f\n", meta["url"], meta["source"], meta["status"], meta["score"])
					continue
				}
				fmt.Fprintf(a.out, "url=%s node_id=%s height=%.0f status=%s score=%.0f last_seen=%s\n", meta["url"], meta["node_id"], meta["last_height"], meta["status"], meta["score"], meta["last_seen_at"])
			} else {
				fmt.Fprintln(a.out, peer)
			}
		}
		return nil
	case "check":
		if len(args) != 2 {
			return errors.New("usage: peer check <url>")
		}
		info, err := remotePostMap(a.rpcURL, "/peers/check", map[string]string{"url": args[1]})
		if err != nil {
			return err
		}
		if ok, _ := info["ok"].(bool); !ok {
			return fmt.Errorf("%s", info["error"])
		}
		printRemotePeerCheck(a.out, info)
		return nil
	case "add":
		if len(args) != 2 {
			return errors.New("usage: peer add <url>")
		}
		_, err := remotePostMap(a.rpcURL, "/peers", map[string]string{"url": args[1]})
		if err == nil {
			fmt.Fprintf(a.out, "peer added: %s\n", args[1])
		}
		return err
	case "connect":
		if len(args) != 2 {
			return errors.New("usage: peer connect <url>")
		}
		info, err := remotePostMap(a.rpcURL, "/peers/connect", map[string]string{"url": args[1]})
		if err == nil {
			fmt.Fprintf(a.out, "peer connected: %s\n", args[1])
			fmt.Fprintf(a.out, "introduced: %t\n", info["introduced"])
		}
		return err
	case "remove":
		if len(args) != 2 {
			return errors.New("usage: peer remove <url>")
		}
		_, err := remoteDoMap(http.MethodDelete, a.rpcURL, "/peers", map[string]string{"url": args[1]})
		if err == nil {
			fmt.Fprintf(a.out, "peer removed: %s\n", args[1])
		}
		return err
	case "clear":
		fs := flag.NewFlagSet("peer clear", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		yes := fs.Bool("yes", false, "confirm clear")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if !*yes {
			return errors.New("refusing to clear peers without --yes")
		}
		_, err := remotePostMap(a.rpcURL, "/peers/clear", map[string]bool{"yes": true})
		if err == nil {
			fmt.Fprintln(a.out, "peers cleared")
		}
		return err
	case "sync":
		fs := flag.NewFlagSet("peer sync", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		peer := fs.String("peer", "", "peer")
		includeBad := fs.Bool("include-bad", false, "include bad peers")
		allowReorg := fs.Bool("allow-reorg", false, "allow explicit safe reorg")
		defaultMaxDepth, err := config.MaxReorgDepthFromEnv(a.profile)
		if err != nil {
			return err
		}
		maxDepth := fs.Uint64("max-reorg-depth", defaultMaxDepth, "max reorg depth")
		yes := fs.Bool("yes", false, "confirm reorg")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *peer == "" {
			if fs.NArg() == 1 {
				*peer = fs.Arg(0)
			} else if fs.NArg() > 1 {
				return errors.New("usage: peer sync [peer-url]")
			}
		}
		info, err := remotePostMap(a.rpcURL, "/peers/sync", map[string]any{"peer": *peer, "include_bad": *includeBad, "allow_reorg": *allowReorg, "max_reorg_depth": *maxDepth, "yes": *yes})
		if err != nil {
			if info != nil {
				printRemotePeerSyncFailure(a.out, info)
			}
			return err
		}
		printRemotePeerSync(a.out, info)
		return nil
	case "status":
		info, err := remoteGetMap(a.rpcURL, "/peer/status")
		if err != nil {
			return err
		}
		printRemotePeerStatus(a.out, info)
		return nil
	case "health":
		info, err := remoteGetMap(a.rpcURL, "/peer/health")
		if err != nil {
			return err
		}
		printRemotePeerHealth(a.out, info)
		return nil
	case "seeds":
		info, err := remoteGetMap(a.rpcURL, "/peer/seeds")
		if err != nil {
			return err
		}
		fmt.Fprintf(a.out, "seed peers: %.0f\n", numberFromAny(info["seed_count"]))
		if seeds, ok := info["seeds"].([]any); ok {
			for _, seed := range seeds {
				if meta, ok := seed.(map[string]any); ok {
					fmt.Fprintf(a.out, "url=%s source=%s status=%s score=%.0f height=%.0f\n", meta["url"], meta["source"], meta["status"], numberFromAny(meta["score"]), numberFromAny(meta["last_height"]))
				}
			}
		}
		return nil
	case "discover":
		fs := flag.NewFlagSet("peer discover", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		limit := fs.Int("limit", p2p.DefaultMaxDiscoveredPeers, "max discovered peers per peer")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		peer := ""
		if fs.NArg() == 1 {
			peer = fs.Arg(0)
		} else if fs.NArg() > 1 {
			return errors.New("usage: peer discover [peer-url]")
		}
		info, err := remotePostMap(a.rpcURL, "/peers/discover", map[string]any{"peer": peer, "limit": *limit})
		if err != nil {
			return err
		}
		fmt.Fprintf(a.out, "discovered peers: %.0f\n", numberFromAny(info["discovered"]))
		if results, ok := info["results"].([]any); ok {
			for _, result := range results {
				if meta, ok := result.(map[string]any); ok {
					fmt.Fprintf(a.out, "peer=%s checked=%.0f added=%.0f skipped=%.0f rejected=%.0f\n", meta["peer"], numberFromAny(meta["checked"]), numberFromAny(meta["added"]), numberFromAny(meta["skipped"]), numberFromAny(meta["rejected"]))
				}
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown peer command %q", args[0])
	}
}

func (a App) p2p(args []string) error {
	if len(args) == 0 {
		return errors.New("p2p command is required")
	}
	switch args[0] {
	case "test-broadcast":
		return a.localP2PTestBroadcast(args[1:])
	case "ping":
		if len(args) != 2 {
			return errors.New("usage: p2p ping <p2p-url>")
		}
		result, _ := p2p.Ping(args[1])
		printP2PPing(a.out, result)
		if !result.OK {
			return errors.New("p2p ping failed")
		}
		return nil
	default:
		return fmt.Errorf("unknown p2p command %q", args[0])
	}
}

func (a App) localP2PTestBroadcast(args []string) error {
	return errors.New("p2p test-broadcast local mode requires --rpc-url in Phase 2.3")
}

func (a App) remoteP2P(args []string) error {
	if len(args) == 0 {
		return errors.New("p2p command is required")
	}
	switch args[0] {
	case "test-broadcast":
		return a.remoteP2PTestBroadcast(args[1:])
	case "debug":
		info, err := remoteGetMap(a.rpcURL, "/debug/p2p")
		if err != nil {
			return err
		}
		printRemoteP2PDebug(a.out, info)
		return nil
	case "ping":
		if len(args) != 2 {
			return errors.New("usage: p2p ping <p2p-url>")
		}
		info, err := remotePostMap(a.rpcURL, "/debug/p2p/ping", map[string]string{"url": args[1]})
		if info != nil {
			printP2PPing(a.out, p2pPingResultFromMap(info))
		}
		return err
	default:
		return fmt.Errorf("unknown p2p command %q", args[0])
	}
}

func (a App) remoteP2PTestBroadcast(args []string) error {
	fs := flag.NewFlagSet("p2p test-broadcast", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	from := fs.String("from", "", "sender address")
	to := fs.String("to", "", "recipient address")
	amt := fs.String("amount", "", "amount")
	miner := fs.String("miner", "", "miner address")
	peerRPC := fs.String("peer-rpc", "", "peer RPC URL")
	wait := fs.Duration("wait", 2*time.Second, "wait duration")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "p2p broadcast test started")
	if _, err := remoteGetMap(a.rpcURL, "/node/status"); err != nil {
		return err
	}
	if _, err := remoteGetMap(a.rpcURL, "/peers"); err != nil {
		return err
	}
	sendInfo, err := remotePostMap(a.rpcURL, "/send", map[string]string{"from": *from, "to": *to, "amount": *amt})
	if err != nil {
		return err
	}
	fmt.Fprintf(a.out, "tx created: %s\n", sendInfo["id"])
	localMempool, err := remoteGetMap(a.rpcURL, "/mempool")
	if err != nil {
		return err
	}
	fmt.Fprintf(a.out, "local mempool: %.0f\n", localMempool["pending_tx_count"])
	printRemoteBroadcastSummary(a.out, "tx broadcast", sendInfo["broadcast"])
	time.Sleep(*wait)
	if *peerRPC != "" {
		if peerMempool, err := remoteGetMap(*peerRPC, "/mempool"); err == nil {
			fmt.Fprintf(a.out, "peer mempool: %.0f\n", peerMempool["pending_tx_count"])
		}
	}
	mineInfo, err := remotePostMap(a.rpcURL, "/mine", map[string]any{"address": *miner, "blocks": uint64(1)})
	if err != nil {
		return err
	}
	blocks, _ := mineInfo["blocks"].([]any)
	if len(blocks) > 0 {
		block, _ := blocks[0].(map[string]any)
		fmt.Fprintf(a.out, "mined block: height=%.0f hash=%s txs=%.0f\n", block["height"], block["hash"], block["txs"])
	}
	printRemoteBroadcastSummary(a.out, "block broadcast", mineInfo["broadcast"])
	time.Sleep(*wait)
	if *peerRPC != "" {
		if balanceInfo, err := remoteGetMap(*peerRPC, "/balance/"+*to); err == nil {
			fmt.Fprintf(a.out, "peer balance: %s %s\n", balanceInfo["balance"], balanceInfo["ticker"])
		}
		local, err := remoteGetMap(a.rpcURL, "/chain/info")
		if err != nil {
			return err
		}
		peer, err := remoteGetMap(*peerRPC, "/chain/info")
		if err != nil {
			return err
		}
		fmt.Fprint(a.out, "compare: ")
		printCompare(a.out, local, peer)
	}
	fmt.Fprintln(a.out, "p2p broadcast test passed")
	return nil
}

func (a App) remoteCompare(args []string) error {
	fs := flag.NewFlagSet("node compare", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	peer := fs.String("peer", "", "peer RPC URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *peer == "" {
		return errors.New("usage: node compare --peer <rpc-url>")
	}
	local, err := remoteGetMap(a.rpcURL, "/chain/info")
	if err != nil {
		return err
	}
	peerInfo, err := remoteGetMap(*peer, "/chain/info")
	if err != nil {
		return err
	}
	printCompare(a.out, local, peerInfo)
	return nil
}

func (a App) remoteCommonAncestor(args []string) error {
	fs := flag.NewFlagSet("chain common-ancestor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	peer := fs.String("peer", "", "peer P2P URL")
	debug := fs.Bool("debug", false, "show locator sent to peer")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *peer == "" {
		return errors.New("usage: chain common-ancestor --peer <p2p-url>")
	}
	if *debug {
		locatorInfo, err := remoteGetMap(a.rpcURL, "/chain/locator")
		if err != nil {
			return err
		}
		printCommonAncestorDebug(a.out, locatorFromMap(locatorInfo), *peer)
	}
	result, err := remotePostMap(a.rpcURL, "/chain/common-ancestor", map[string]string{"peer": *peer})
	if err != nil {
		return err
	}
	printCommonAncestor(a.out, commonAncestorFromMap(result))
	return nil
}

func (a App) remoteFork(args []string) error {
	if len(args) == 0 || args[0] != "check" {
		return errors.New("usage: fork check --peer <p2p-url>")
	}
	fs := flag.NewFlagSet("fork check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	peer := fs.String("peer", "", "peer P2P URL")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *peer == "" {
		return errors.New("usage: fork check --peer <p2p-url>")
	}
	result, err := remotePostMap(a.rpcURL, "/fork/check", map[string]string{"peer": *peer})
	if err != nil {
		return err
	}
	printForkCheck(a.out, forkCheckFromMap(result))
	return nil
}

func (a App) reorg(args []string) error {
	if len(args) == 0 {
		return errors.New("reorg command is required")
	}
	fs := flag.NewFlagSet("reorg "+args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	peer := fs.String("peer", "", "peer P2P URL")
	defaultMaxDepth, err := config.MaxReorgDepthFromEnv(a.profile)
	if err != nil {
		return err
	}
	maxDepth := fs.Uint64("max-depth", defaultMaxDepth, "max reorg depth")
	yes := fs.Bool("yes", false, "confirm reorg apply")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *peer == "" {
		return errors.New("usage: reorg " + args[0] + " --peer <p2p-url>")
	}
	switch args[0] {
	case "preview":
		plan, _, err := p2p.BuildReorgPlanWithProfile(a.paths, *peer, *maxDepth, a.profile)
		if err != nil {
			return err
		}
		printReorgPlan(a.out, plan)
		return nil
	case "apply":
		res, err := p2p.ApplyReorgWithProfile(a.paths, *peer, *maxDepth, *yes, a.profile)
		if err != nil {
			if res.Error != "" {
				fmt.Fprintln(a.out, res.Error)
			}
			return err
		}
		printReorgResult(a.out, res)
		return nil
	default:
		return fmt.Errorf("unknown reorg command %q", args[0])
	}
}

func (a App) remoteReorg(args []string) error {
	if len(args) == 0 {
		return errors.New("reorg command is required")
	}
	fs := flag.NewFlagSet("reorg "+args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	peer := fs.String("peer", "", "peer P2P URL")
	defaultMaxDepth, err := config.MaxReorgDepthFromEnv(a.profile)
	if err != nil {
		return err
	}
	maxDepth := fs.Uint64("max-depth", defaultMaxDepth, "max reorg depth")
	yes := fs.Bool("yes", false, "confirm")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *peer == "" {
		return errors.New("usage: reorg " + args[0] + " --peer <p2p-url>")
	}
	body := map[string]any{"peer": *peer, "max_depth": *maxDepth}
	switch args[0] {
	case "preview":
		info, err := remotePostMap(a.rpcURL, "/reorg/preview", body)
		if err != nil {
			return err
		}
		printReorgMap(a.out, info)
		return nil
	case "apply":
		body["yes"] = *yes
		info, err := remotePostMap(a.rpcURL, "/reorg/apply", body)
		if err != nil {
			if info != nil && info["error"] != nil {
				fmt.Fprintln(a.out, info["error"])
			}
			return err
		}
		printReorgApplyMap(a.out, info)
		return nil
	default:
		return fmt.Errorf("unknown reorg command %q", args[0])
	}
}

func parsePeers(value string) []string {
	if value == "" {
		return nil
	}
	return parseCSV(value)
}

func parseCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func validatePublicRPCMode(publicRPC bool, rpcAddr string, walletRPC bool, adminRPC bool) error {
	if !publicRPC || !isPublicRPCBind(rpcAddr) {
		return nil
	}
	switch {
	case walletRPC && adminRPC:
		return errors.New("unsafe public RPC configuration: wallet and admin RPC cannot be enabled on a public bind")
	case walletRPC:
		return errors.New("unsafe public RPC configuration: wallet RPC cannot be enabled on a public bind")
	case adminRPC:
		return errors.New("unsafe public RPC configuration: admin RPC cannot be enabled on a public bind")
	default:
		return nil
	}
}

func isPublicRPCBind(addr string) bool {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return false
	}
	if strings.HasPrefix(addr, ":") {
		return true
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.Trim(host, "[]")
	if host == "" || host == "0.0.0.0" || host == "::" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsUnspecified()
}

func peerSourceSummary(hasFile, hasFlag bool) string {
	switch {
	case hasFile && hasFlag:
		return "peers.json + flag"
	case hasFile:
		return "peers.json"
	case hasFlag:
		return "flag"
	default:
		return ""
	}
}

type startupPeerInputs struct {
	FlagPeers []string
	Bootnodes []string
	SeedPeers []string
	SelfURL   string
}

type startupPeerSummary struct {
	FlagPeersAdded int
	BootnodesAdded int
	SeedsAdded     int
	SkippedSelf    int
}

func addStartupPeers(store p2p.PeerStore, inputs startupPeerInputs) (startupPeerSummary, error) {
	var summary startupPeerSummary
	self := ""
	if inputs.SelfURL != "" {
		normalized, err := p2p.NormalizePeerURL(inputs.SelfURL)
		if err == nil {
			self = normalized
		}
	}
	add := func(peer, source string) error {
		normalized, err := p2p.NormalizePeerURL(peer)
		if err != nil {
			return err
		}
		if self != "" && normalized == self {
			summary.SkippedSelf++
			return nil
		}
		if err := store.AddWithSource(normalized, source); err != nil {
			return err
		}
		switch source {
		case "flag":
			summary.FlagPeersAdded++
		case "bootnode":
			summary.BootnodesAdded++
		case "seed":
			summary.SeedsAdded++
		}
		return nil
	}
	for _, peer := range inputs.FlagPeers {
		if err := add(peer, "flag"); err != nil {
			return summary, err
		}
	}
	for _, peer := range inputs.Bootnodes {
		if err := add(peer, "bootnode"); err != nil {
			return summary, err
		}
	}
	for _, peer := range inputs.SeedPeers {
		if err := add(peer, "seed"); err != nil {
			return summary, err
		}
	}
	return summary, nil
}

func collectSeedPeers(profile config.NetworkConfig, seedPeer, seedPeersText, seedFile string) ([]string, error) {
	var raw []string
	raw = append(raw, profile.SeedPeers...)
	raw = append(raw, parsePeers(seedPeer)...)
	raw = append(raw, parsePeers(seedPeersText)...)
	if seedFile != "" {
		fileSeeds, err := parseSeedFile(seedFile)
		if err != nil {
			return nil, err
		}
		raw = append(raw, fileSeeds...)
	}
	return normalizePeerList(raw)
}

func collectUpstreamPeers(upstreamPeer, upstreamPeersText, upstreamFile string) ([]string, error) {
	var raw []string
	raw = append(raw, parsePeers(upstreamPeer)...)
	raw = append(raw, parsePeers(upstreamPeersText)...)
	if upstreamFile != "" {
		filePeers, err := parseSeedFile(upstreamFile)
		if err != nil {
			return nil, err
		}
		raw = append(raw, filePeers...)
	}
	return normalizePeerList(raw)
}

func parseSeedFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var seeds []string
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}
		if _, err := p2p.NormalizePeerURL(line); err != nil {
			return nil, fmt.Errorf("invalid seed peer %s:%d: %w", path, lineNo, err)
		}
		seeds = append(seeds, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return normalizePeerList(seeds)
}

func normalizePeerList(peers []string) ([]string, error) {
	out := make([]string, 0, len(peers))
	seen := map[string]struct{}{}
	for _, peer := range peers {
		normalized, err := p2p.NormalizePeerURL(peer)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}

type nodeConfigFile struct {
	Network string `json:"network"`
	RPC     struct {
		Addr               string   `json:"addr"`
		Public             *bool    `json:"public"`
		EnableWallet       *bool    `json:"enable_wallet"`
		EnableMiner        *bool    `json:"enable_miner"`
		EnableAdmin        *bool    `json:"enable_admin"`
		EnableService      *bool    `json:"enable_service"`
		RateLimitPerMinute int      `json:"rate_limit_per_minute"`
		CORSOrigins        []string `json:"cors_origins"`
	} `json:"rpc"`
	P2P struct {
		Addr               string   `json:"addr"`
		Advertise          string   `json:"advertise"`
		SeedPeers          []string `json:"seed_peers"`
		SeedFile           string   `json:"seed_file"`
		UpstreamPeers      []string `json:"upstream_peers"`
		UpstreamFile       string   `json:"upstream_file"`
		MaxPeers           int      `json:"max_peers"`
		MaxReorgDepth      uint64   `json:"max_reorg_depth"`
		MinMiningPeers     int      `json:"min_mining_peers"`
		AllowIsolatedMine  *bool    `json:"allow_isolated_mining"`
		MinWritePeers      int      `json:"min_write_peers"`
		AllowIsolatedWrite *bool    `json:"allow_isolated_writes"`
		AllowPrivatePeers  bool     `json:"allow_private_peers"`
	} `json:"p2p"`
}

func loadNodeConfig(path string) (nodeConfigFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nodeConfigFile{}, err
	}
	var cfg nodeConfigFile
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nodeConfigFile{}, fmt.Errorf("invalid config file: %w", err)
	}
	return cfg, nil
}

func applyNodeConfig(fs *flag.FlagSet, cfg nodeConfigFile, rpcAddr, p2pAddr, advertise, seedPeersText, seedFile, upstreamPeersText, upstreamFile *string, publicRPC, enableWallet, enableMiner, enableAdmin, enableService *bool, corsOrigins *string, rateLimitPerMinute, maxPeers *int, maxReorgDepth *uint64, minMiningPeers *int, allowIsolatedMining *bool, minWritePeers *int, allowIsolatedWrites *bool) {
	visited := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })
	if cfg.RPC.Addr != "" && !visited["rpc"] {
		*rpcAddr = cfg.RPC.Addr
	}
	if cfg.P2P.Addr != "" && !visited["p2p"] {
		*p2pAddr = cfg.P2P.Addr
	}
	if cfg.P2P.Advertise != "" && !visited["advertise-p2p"] {
		*advertise = cfg.P2P.Advertise
	}
	if len(cfg.P2P.SeedPeers) > 0 && !visited["seed-peers"] {
		*seedPeersText = strings.Join(cfg.P2P.SeedPeers, ",")
	}
	if cfg.P2P.SeedFile != "" && !visited["seed-file"] {
		*seedFile = cfg.P2P.SeedFile
	}
	if len(cfg.P2P.UpstreamPeers) > 0 && !visited["upstream-peers"] {
		*upstreamPeersText = strings.Join(cfg.P2P.UpstreamPeers, ",")
	}
	if cfg.P2P.UpstreamFile != "" && !visited["upstream-file"] {
		*upstreamFile = cfg.P2P.UpstreamFile
	}
	if cfg.RPC.Public != nil && !visited["public-rpc"] {
		*publicRPC = *cfg.RPC.Public
	}
	if cfg.RPC.EnableWallet != nil && !visited["enable-wallet-rpc"] {
		*enableWallet = *cfg.RPC.EnableWallet
	}
	if cfg.RPC.EnableMiner != nil && !visited["enable-miner-rpc"] {
		*enableMiner = *cfg.RPC.EnableMiner
	}
	if cfg.RPC.EnableAdmin != nil && !visited["enable-admin-rpc"] {
		*enableAdmin = *cfg.RPC.EnableAdmin
	}
	if cfg.RPC.EnableService != nil && !visited["enable-service-rpc"] {
		*enableService = *cfg.RPC.EnableService
	}
	if len(cfg.RPC.CORSOrigins) > 0 && !visited["cors-origins"] {
		*corsOrigins = strings.Join(cfg.RPC.CORSOrigins, ",")
	}
	if cfg.RPC.RateLimitPerMinute > 0 {
		*rateLimitPerMinute = cfg.RPC.RateLimitPerMinute
	}
	if cfg.P2P.MaxPeers > 0 {
		*maxPeers = cfg.P2P.MaxPeers
	}
	if cfg.P2P.MaxReorgDepth > 0 && !visited["max-reorg-depth"] {
		*maxReorgDepth = cfg.P2P.MaxReorgDepth
	}
	if cfg.P2P.MinMiningPeers > 0 && !visited["min-mining-peers"] {
		*minMiningPeers = cfg.P2P.MinMiningPeers
	}
	if cfg.P2P.AllowIsolatedMine != nil && !visited["allow-isolated-mining"] {
		*allowIsolatedMining = *cfg.P2P.AllowIsolatedMine
	}
	if cfg.P2P.MinWritePeers > 0 && !visited["min-write-peers"] {
		*minWritePeers = cfg.P2P.MinWritePeers
	}
	if cfg.P2P.AllowIsolatedWrite != nil && !visited["allow-isolated-writes"] {
		*allowIsolatedWrites = *cfg.P2P.AllowIsolatedWrite
	}
}

func remoteGetMap(base, path string) (map[string]any, error) {
	return remoteDoMap(http.MethodGet, base, path, nil)
}

func remotePostMap(base, path string, body any) (map[string]any, error) {
	return remoteDoMap(http.MethodPost, base, path, body)
}

func remotePostMapWithTimeout(base, path string, body any, timeout time.Duration) (map[string]any, error) {
	return remoteDoMapWithTimeout(http.MethodPost, base, path, body, timeout)
}

func remoteDoMap(method, base, path string, body any) (map[string]any, error) {
	return remoteDoMapWithTimeout(method, base, path, body, 0)
}

func remoteDoMapWithTimeout(method, base, path string, body any, timeout time.Duration) (map[string]any, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, strings.TrimRight(base, "/")+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := http.DefaultClient
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		if msg, ok := out["error"].(string); ok {
			return out, errors.New(msg)
		}
		return out, fmt.Errorf("remote returned %s", resp.Status)
	}
	return out, nil
}

func printReorgPlan(out io.Writer, plan p2p.ReorgPlan) {
	fmt.Fprintln(out, "reorg preview")
	if plan.LocalTip != plan.PeerTip {
		fmt.Fprintln(out, "fork detected")
	}
	fmt.Fprintf(out, "common ancestor height: %d\n", plan.CommonAncestorHeight)
	fmt.Fprintf(out, "local height: %d\n", plan.LocalHeight)
	fmt.Fprintf(out, "peer height: %d\n", plan.PeerHeight)
	fmt.Fprintf(out, "disconnect blocks: %d\n", len(plan.DisconnectBlocks))
	fmt.Fprintf(out, "connect blocks: %d\n", len(plan.ConnectBlocks))
	fmt.Fprintf(out, "reorg depth: %d\n", plan.ReorgDepth)
	fmt.Fprintf(out, "max reorg depth: %d\n", plan.MaxReorgDepth)
	fmt.Fprintf(out, "local work: %d\n", plan.LocalWork)
	fmt.Fprintf(out, "peer work: %d\n", plan.PeerWork)
	fmt.Fprintf(out, "allowed: %t\n", plan.Allowed)
	fmt.Fprintf(out, "decision: %s\n", plan.Decision)
	fmt.Fprintf(out, "reason: %s\n", plan.Reason)
}
func printReorgMap(out io.Writer, info map[string]any) {
	fmt.Fprintln(out, "reorg preview")
	fmt.Fprintln(out, "fork detected")
	fmt.Fprintf(out, "common ancestor height: %.0f\n", numberFromAny(info["common_ancestor_height"]))
	fmt.Fprintf(out, "local height: %.0f\n", numberFromAny(info["local_height"]))
	fmt.Fprintf(out, "peer height: %.0f\n", numberFromAny(info["peer_height"]))
	fmt.Fprintf(out, "disconnect blocks: %.0f\n", numberFromAny(info["disconnect_blocks"]))
	fmt.Fprintf(out, "connect blocks: %.0f\n", numberFromAny(info["connect_blocks"]))
	fmt.Fprintf(out, "reorg depth: %.0f\n", numberFromAny(info["reorg_depth"]))
	fmt.Fprintf(out, "max reorg depth: %.0f\n", numberFromAny(info["max_reorg_depth"]))
	fmt.Fprintf(out, "local work: %.0f\n", numberFromAny(info["local_work"]))
	fmt.Fprintf(out, "peer work: %.0f\n", numberFromAny(info["peer_work"]))
	fmt.Fprintf(out, "allowed: %t\n", info["allowed"])
	if decision, ok := info["decision"].(string); ok && decision != "" {
		fmt.Fprintf(out, "decision: %s\n", decision)
	}
	fmt.Fprintf(out, "reason: %s\n", info["reason"])
}
func printReorgResult(out io.Writer, res p2p.ReorgResult) {
	fmt.Fprintln(out, "reorg applied")
	fmt.Fprintf(out, "common ancestor height: %d\n", res.Plan.CommonAncestorHeight)
	fmt.Fprintf(out, "old height: %d\n", res.OldHeight)
	fmt.Fprintf(out, "new height: %d\n", res.NewHeight)
	fmt.Fprintf(out, "disconnected blocks: %d\n", res.DisconnectedBlocks)
	fmt.Fprintf(out, "connected blocks: %d\n", res.ConnectedBlocks)
	fmt.Fprintf(out, "old tip: %s\n", res.OldTip)
	fmt.Fprintf(out, "new tip: %s\n", res.NewTip)
	fmt.Fprintf(out, "requeued transactions: %d\n", res.RequeuedTransactions)
	fmt.Fprintf(out, "dropped transactions: %d\n", res.DroppedTransactions)
	fmt.Fprintf(out, "dropped confirmed transactions: %d\n", res.DroppedConfirmedTransactions)
	fmt.Fprintf(out, "dropped invalid transactions: %d\n", res.DroppedInvalidTransactions)
	fmt.Fprintf(out, "mempool count: %d\n", res.MempoolCount)
	fmt.Fprintf(out, "chain valid: %t\n", res.ChainValid)
}
func printReorgApplyMap(out io.Writer, info map[string]any) {
	fmt.Fprintln(out, "reorg applied")
	fmt.Fprintf(out, "old height: %.0f\n", numberFromAny(info["old_height"]))
	fmt.Fprintf(out, "new height: %.0f\n", numberFromAny(info["new_height"]))
	fmt.Fprintf(out, "disconnected blocks: %.0f\n", numberFromAny(info["disconnected_blocks"]))
	fmt.Fprintf(out, "connected blocks: %.0f\n", numberFromAny(info["connected_blocks"]))
	fmt.Fprintf(out, "old tip: %s\n", info["old_tip"])
	fmt.Fprintf(out, "new tip: %s\n", info["new_tip"])
	fmt.Fprintf(out, "requeued transactions: %.0f\n", numberFromAny(info["requeued_transactions"]))
	fmt.Fprintf(out, "dropped transactions: %.0f\n", numberFromAny(info["dropped_transactions"]))
	fmt.Fprintf(out, "dropped confirmed transactions: %.0f\n", numberFromAny(info["dropped_confirmed_transactions"]))
	fmt.Fprintf(out, "dropped invalid transactions: %.0f\n", numberFromAny(info["dropped_invalid_transactions"]))
	fmt.Fprintf(out, "mempool count: %.0f\n", numberFromAny(info["mempool_count"]))
	fmt.Fprintf(out, "chain valid: %t\n", info["chain_valid"])
}

func printRemoteChainInfo(out io.Writer, info map[string]any) {
	if network, ok := info["network"]; ok {
		fmt.Fprintf(out, "network: %s\n", network)
	}
	if chainID, ok := info["chain_id"]; ok {
		fmt.Fprintf(out, "chain id: %.0f\n", numberFromAny(chainID))
	}
	fmt.Fprintf(out, "height: %.0f\n", info["height"])
	fmt.Fprintf(out, "tip hash: %s\n", info["tip_hash"])
	fmt.Fprintf(out, "difficulty: %.0f\n", info["difficulty"])
	fmt.Fprintf(out, "tip difficulty: %.0f\n", info["tip_difficulty"])
	fmt.Fprintf(out, "next difficulty: %.0f\n", info["next_difficulty"])
	if target, ok := info["target_block_time_seconds"]; ok {
		fmt.Fprintf(out, "target block time: %.0fs\n", numberFromAny(target))
	}
	if window, ok := info["retarget_window"]; ok {
		fmt.Fprintf(out, "retarget window: %.0f\n", numberFromAny(window))
	}
	if minDifficulty, ok := info["min_difficulty"]; ok {
		fmt.Fprintf(out, "min difficulty: %.0f\n", numberFromAny(minDifficulty))
	}
	if maxDifficulty, ok := info["max_difficulty"]; ok {
		fmt.Fprintf(out, "max difficulty: %.0f\n", numberFromAny(maxDifficulty))
	}
	if blocksUntil, ok := info["blocks_until_retarget"]; ok {
		fmt.Fprintf(out, "blocks until retarget: %.0f\n", numberFromAny(blocksUntil))
	}
	if maturity, ok := info["coinbase_maturity"]; ok {
		fmt.Fprintf(out, "coinbase maturity: %.0f\n", numberFromAny(maturity))
	}
	fmt.Fprintf(out, "total supply: %s\n", info["total_supply"])
	fmt.Fprintf(out, "cumulative work: %.0f\n", info["cumulative_work"])
	fmt.Fprintf(out, "pending tx count: %.0f\n", info["pending_tx_count"])
	fmt.Fprintf(out, "blocks: %.0f\n", info["blocks"])
	fmt.Fprintf(out, "coinbase blocks: %.0f\n", info["coinbase_blocks"])
	fmt.Fprintf(out, "total transactions: %.0f\n", info["total_transactions"])
	fmt.Fprintf(out, "coinbase transactions: %.0f\n", numberFromAny(info["coinbase_transactions"]))
	fmt.Fprintf(out, "normal transactions: %.0f\n", numberFromAny(info["normal_transactions"]))
	fmt.Fprintf(out, "circulating supply: %s\n", info["circulating_supply"])
	for _, field := range []string{"staking_enabled", "min_stake_amount", "min_service_stake", "unbonding_period", "total_active_stake", "total_unlocking_stake", "active_stake_count"} {
		if value, ok := info[field]; ok {
			fmt.Fprintf(out, "%s: %v\n", strings.ReplaceAll(field, "_", " "), value)
		}
	}
	fmt.Fprintf(out, "datadir: %s\n", info["datadir"])
}

func printRemoteBalanceDetails(out io.Writer, info map[string]any) {
	if balance, ok := info["balance"]; ok {
		if ticker, ok := info["ticker"]; ok {
			fmt.Fprintf(out, "%s %s\n", balance, ticker)
		} else {
			fmt.Fprintf(out, "%s\n", balance)
		}
	}
	if address, ok := info["address"]; ok {
		fmt.Fprintf(out, "address: %s\n", address)
	}
	for _, field := range []string{"confirmed_balance", "mature_balance", "immature_balance", "active_stake", "unlocking_stake", "released_stake", "pending_stake_lock", "spendable_balance", "pending_outgoing", "pending_incoming"} {
		if value, ok := info[field]; ok {
			fmt.Fprintf(out, "%s: %s\n", strings.ReplaceAll(field, "_", " "), value)
		}
	}
	if maturity, ok := info["coinbase_maturity"]; ok {
		fmt.Fprintf(out, "coinbase maturity: %.0f\n", numberFromAny(maturity))
	}
	if height, ok := info["current_height"]; ok {
		fmt.Fprintf(out, "current height: %.0f\n", numberFromAny(height))
	}
}

func printRemoteFaucetInfo(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "enabled: %t\n", boolFromAny(info["enabled"]))
	fmt.Fprintf(out, "network: %s\n", info["network"])
	fmt.Fprintf(out, "network id: %s\n", info["network_id"])
	fmt.Fprintf(out, "chain id: %.0f\n", numberFromAny(info["chain_id"]))
	fmt.Fprintf(out, "faucet address: %s\n", info["faucet_address"])
	fmt.Fprintf(out, "amount: %s %s\n", info["amount"], config.Ticker)
	fmt.Fprintf(out, "max per address: %s %s\n", info["max_per_address"], config.Ticker)
	fmt.Fprintf(out, "min interval: %.0fs\n", numberFromAny(info["min_interval_seconds"]))
	fmt.Fprintf(out, "mempool pending: %.0f\n", numberFromAny(info["mempool_pending"]))
	fmt.Fprintf(out, "note: %s\n", info["note"])
}

func printRemoteFaucetRequest(out io.Writer, info map[string]any) {
	fmt.Fprintln(out, "faucet tx created")
	fmt.Fprintf(out, "tx id: %s\n", info["tx_id"])
	fmt.Fprintf(out, "from: %s\n", info["from"])
	fmt.Fprintf(out, "to: %s\n", info["to"])
	fmt.Fprintf(out, "amount: %s %s\n", info["amount"], config.Ticker)
	fmt.Fprintf(out, "status: %s\n", info["status"])
	fmt.Fprintf(out, "note: %s\n", info["note"])
}

func printChainDifficulty(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "network: %s\n", info["network"])
	fmt.Fprintf(out, "chain id: %.0f\n", numberFromAny(info["chain_id"]))
	fmt.Fprintf(out, "height: %.0f\n", numberFromAny(info["height"]))
	fmt.Fprintf(out, "tip difficulty: %.0f\n", numberFromAny(info["tip_difficulty"]))
	fmt.Fprintf(out, "next difficulty: %.0f\n", numberFromAny(info["next_difficulty"]))
	fmt.Fprintf(out, "target block time: %.0fs\n", numberFromAny(info["target_block_time_seconds"]))
	fmt.Fprintf(out, "retarget window: %.0f\n", numberFromAny(info["retarget_window"]))
	fmt.Fprintf(out, "min difficulty: %.0f\n", numberFromAny(info["min_difficulty"]))
	fmt.Fprintf(out, "max difficulty: %.0f\n", numberFromAny(info["max_difficulty"]))
	fmt.Fprintf(out, "blocks until retarget: %.0f\n", numberFromAny(info["blocks_until_retarget"]))
	fmt.Fprintf(out, "cumulative work: %.0f\n", numberFromAny(info["cumulative_work"]))
}

func printRemoteChainDifficulty(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "network: %s\n", info["network"])
	fmt.Fprintf(out, "chain id: %.0f\n", numberFromAny(info["chain_id"]))
	fmt.Fprintf(out, "height: %.0f\n", numberFromAny(info["height"]))
	fmt.Fprintf(out, "tip difficulty: %.0f\n", numberFromAny(info["tip_difficulty"]))
	fmt.Fprintf(out, "next difficulty: %.0f\n", numberFromAny(info["next_difficulty"]))
	fmt.Fprintf(out, "target block time: %.0fs\n", numberFromAny(info["target_block_time_seconds"]))
	fmt.Fprintf(out, "retarget window: %.0f\n", numberFromAny(info["retarget_window"]))
	fmt.Fprintf(out, "min difficulty: %.0f\n", numberFromAny(info["min_difficulty"]))
	fmt.Fprintf(out, "max difficulty: %.0f\n", numberFromAny(info["max_difficulty"]))
	fmt.Fprintf(out, "blocks until retarget: %.0f\n", numberFromAny(info["blocks_until_retarget"]))
	fmt.Fprintf(out, "cumulative work: %.0f\n", numberFromAny(info["cumulative_work"]))
}

func chainDifficultyInfo(blocks []types.Block, profile config.NetworkConfig) map[string]any {
	params := difficultyParams(profile)
	tip := blocks[len(blocks)-1]
	return map[string]any{
		"network":                   profile.Name,
		"chain_id":                  profile.ChainID,
		"height":                    tip.Height,
		"tip_difficulty":            tip.Difficulty,
		"next_difficulty":           chain.CalculateNextDifficultyWithParams(blocks, params),
		"target_block_time_seconds": params.TargetBlockTimeSeconds,
		"retarget_window":           params.RetargetWindow,
		"min_difficulty":            params.MinDifficulty,
		"max_difficulty":            params.MaxDifficulty,
		"blocks_until_retarget":     chain.BlocksUntilRetarget(blocks, params),
		"cumulative_work":           chain.CalculateCumulativeWork(blocks),
	}
}

func miningObservationInfo(blocks []types.Block, profile config.NetworkConfig, pendingCount, peerCount int) map[string]any {
	params := difficultyParams(profile)
	tip := blocks[len(blocks)-1]
	nextDifficulty := chain.CalculateNextDifficultyWithParams(blocks, params)
	intervals := cliRecentBlockIntervals(blocks, int(params.RetargetWindow))
	avg, minInterval, maxInterval := cliIntervalStats(intervals)
	lastBlockAge := int64(0)
	if tip.Height > 0 {
		lastBlockAge = time.Now().Unix() - tip.Timestamp
		if lastBlockAge < 0 {
			lastBlockAge = 0
		}
	}
	stats := chain.CalculateChainStatsWithProfile(blocks, profile)
	return map[string]any{
		"network":                        profile.Name,
		"network_id":                     profile.NetworkID,
		"chain_id":                       profile.ChainID,
		"height":                         tip.Height,
		"tip_hash":                       tip.Hash,
		"current_difficulty":             tip.Difficulty,
		"tip_difficulty":                 tip.Difficulty,
		"next_difficulty":                nextDifficulty,
		"target_block_time_seconds":      params.TargetBlockTimeSeconds,
		"retarget_window":                params.RetargetWindow,
		"blocks_until_retarget":          chain.BlocksUntilRetarget(blocks, params),
		"last_block_time":                tip.Timestamp,
		"last_block_age_seconds":         lastBlockAge,
		"average_interval_seconds":       avg,
		"min_interval_seconds":           minInterval,
		"max_interval_seconds":           maxInterval,
		"projected_retarget_direction":   projectedRetargetDirection(tip.Difficulty, nextDifficulty, avg, params.TargetBlockTimeSeconds),
		"recent_block_intervals_seconds": intervals,
		"pending_tx_count":               pendingCount,
		"peer_count":                     peerCount,
		"total_supply":                   amount.Format(stats.TotalSupply) + " " + config.Ticker,
		"coinbase_maturity":              consensusParams(profile).CoinbaseMaturity,
		"note":                           "difficulty observation is informational only; testnet IDR has no monetary value",
	}
}

func printMiningStatus(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "network: %s\n", info["network"])
	fmt.Fprintf(out, "network id: %s\n", info["network_id"])
	fmt.Fprintf(out, "chain id: %.0f\n", numberFromAny(info["chain_id"]))
	fmt.Fprintf(out, "height: %.0f\n", numberFromAny(info["height"]))
	fmt.Fprintf(out, "tip hash: %s\n", info["tip_hash"])
	fmt.Fprintf(out, "current difficulty: %.0f\n", numberFromAny(firstPresent(info, "current_difficulty", "tip_difficulty")))
	fmt.Fprintf(out, "next difficulty: %.0f\n", numberFromAny(info["next_difficulty"]))
	fmt.Fprintf(out, "target block time: %.0fs\n", numberFromAny(info["target_block_time_seconds"]))
	fmt.Fprintf(out, "retarget window: %.0f\n", numberFromAny(info["retarget_window"]))
	fmt.Fprintf(out, "blocks until retarget: %.0f\n", numberFromAny(info["blocks_until_retarget"]))
	fmt.Fprintf(out, "last block age: %.0fs\n", numberFromAny(info["last_block_age_seconds"]))
	fmt.Fprintf(out, "average interval: %.2fs\n", numberFromAny(info["average_interval_seconds"]))
	fmt.Fprintf(out, "min interval: %.0fs\n", numberFromAny(info["min_interval_seconds"]))
	fmt.Fprintf(out, "max interval: %.0fs\n", numberFromAny(info["max_interval_seconds"]))
	fmt.Fprintf(out, "projected retarget direction: %s\n", info["projected_retarget_direction"])
	fmt.Fprintf(out, "coinbase maturity: %.0f\n", numberFromAny(info["coinbase_maturity"]))
	fmt.Fprintf(out, "pending tx count: %.0f\n", numberFromAny(info["pending_tx_count"]))
	fmt.Fprintf(out, "peer count: %.0f\n", numberFromAny(info["peer_count"]))
	if note, _ := info["note"].(string); note != "" {
		fmt.Fprintf(out, "note: %s\n", note)
	}
}

func printMiningBlocks(out io.Writer, info map[string]any) {
	printMiningStatus(out, info)
	fmt.Fprintln(out, "recent blocks:")
	for _, block := range anySlice(info["blocks"]) {
		entry, ok := block.(map[string]any)
		if !ok {
			continue
		}
		fmt.Fprintf(out, "height=%.0f hash=%s interval=%.0fs difficulty=%.0f txs=%.0f miner=%s\n",
			numberFromAny(entry["height"]),
			entry["hash"],
			numberFromAny(entry["interval_seconds"]),
			numberFromAny(entry["difficulty"]),
			numberFromAny(entry["tx_count"]),
			entry["miner_address"],
		)
	}
}

func recentMiningBlockViews(blocks []types.Block, limit int) []map[string]any {
	if limit <= 0 {
		limit = 30
	}
	start := len(blocks) - limit
	if start < 0 {
		start = 0
	}
	out := make([]map[string]any, 0, len(blocks)-start)
	for i := len(blocks) - 1; i >= start; i-- {
		block := blocks[i]
		interval := int64(0)
		if i > 0 {
			interval = block.Timestamp - blocks[i-1].Timestamp
			if interval < 0 {
				interval = 0
			}
		}
		out = append(out, map[string]any{
			"height":           block.Height,
			"hash":             block.Hash,
			"interval_seconds": interval,
			"difficulty":       block.Difficulty,
			"tx_count":         len(block.Transactions),
			"miner_address":    block.MinerAddress,
		})
	}
	return out
}

func cliRecentBlockIntervals(blocks []types.Block, limit int) []int64 {
	if limit <= 0 {
		limit = 30
	}
	if len(blocks) < 2 {
		return nil
	}
	start := len(blocks) - limit
	if start < 1 {
		start = 1
	}
	out := make([]int64, 0, len(blocks)-start)
	for i := start; i < len(blocks); i++ {
		interval := blocks[i].Timestamp - blocks[i-1].Timestamp
		if interval < 0 {
			interval = 0
		}
		out = append(out, interval)
	}
	return out
}

func cliIntervalStats(intervals []int64) (float64, int64, int64) {
	if len(intervals) == 0 {
		return 0, 0, 0
	}
	minInterval, maxInterval := intervals[0], intervals[0]
	var total int64
	for _, interval := range intervals {
		total += interval
		if interval < minInterval {
			minInterval = interval
		}
		if interval > maxInterval {
			maxInterval = interval
		}
	}
	return float64(total) / float64(len(intervals)), minInterval, maxInterval
}

func projectedRetargetDirection(current, next uint32, avg float64, target int64) string {
	if next > current {
		return "up"
	}
	if next < current {
		return "down"
	}
	if avg <= 0 || target <= 0 {
		return "unchanged"
	}
	if avg < float64(target) {
		return "up"
	}
	if avg > float64(target) {
		return "down"
	}
	return "unchanged"
}

func firstPresent(info map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := info[key]; ok {
			return value
		}
	}
	return nil
}

func anySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func difficultyParams(profile config.NetworkConfig) config.DifficultyParams {
	if profile.Difficulty.InitialDifficulty == 0 {
		return config.Localnet().Difficulty
	}
	return profile.Difficulty
}

func consensusParams(profile config.NetworkConfig) config.ConsensusParams {
	if profile.Consensus.CoinbaseMaturity == 0 {
		return config.Localnet().Consensus
	}
	return profile.Consensus
}

func printBalanceDetails(out io.Writer, details ledger.BalanceDetails) {
	fmt.Fprintf(out, "%s %s\n", amount.Format(details.Confirmed), config.Ticker)
	fmt.Fprintf(out, "address: %s\n", details.Address)
	fmt.Fprintf(out, "confirmed balance: %s %s\n", amount.Format(details.Confirmed), config.Ticker)
	fmt.Fprintf(out, "mature balance: %s %s\n", amount.Format(details.Mature), config.Ticker)
	fmt.Fprintf(out, "immature balance: %s %s\n", amount.Format(details.Immature), config.Ticker)
	fmt.Fprintf(out, "active stake: %s %s\n", amount.Format(details.ActiveStake), config.Ticker)
	fmt.Fprintf(out, "unlocking stake: %s %s\n", amount.Format(details.UnlockingStake), config.Ticker)
	fmt.Fprintf(out, "released stake: %s %s\n", amount.Format(details.ReleasedStake), config.Ticker)
	fmt.Fprintf(out, "pending stake lock: %s %s\n", amount.Format(details.PendingStakeLock), config.Ticker)
	fmt.Fprintf(out, "spendable balance: %s %s\n", amount.Format(details.Spendable), config.Ticker)
	fmt.Fprintf(out, "pending outgoing: %s %s\n", amount.Format(details.PendingOutgoing), config.Ticker)
	fmt.Fprintf(out, "pending incoming: %s %s\n", amount.Format(details.PendingIncoming), config.Ticker)
	fmt.Fprintf(out, "coinbase maturity: %d\n", details.CoinbaseMaturity)
	fmt.Fprintf(out, "current height: %d\n", details.CurrentHeight)
}

func printLocator(out io.Writer, locator p2p.LocatorResponse) {
	fmt.Fprintf(out, "height: %d\n", locator.Height)
	fmt.Fprintf(out, "tip hash: %s\n", locator.TipHash)
	fmt.Fprintln(out, "locator:")
	for _, entry := range locator.Locator {
		fmt.Fprintf(out, "height=%d hash=%s\n", entry.Height, entry.Hash)
	}
}

func printCommonAncestor(out io.Writer, ancestor p2p.CommonAncestorResponse) {
	if !ancestor.Found {
		fmt.Fprintln(out, "common ancestor not found")
		if ancestor.Error != "" {
			fmt.Fprintf(out, "error: %s\n", ancestor.Error)
		}
		return
	}
	fmt.Fprintln(out, "common ancestor found")
	fmt.Fprintf(out, "height: %d\n", ancestor.Height)
	fmt.Fprintf(out, "hash: %s\n", ancestor.Hash)
}

func printCommonAncestorDebug(out io.Writer, locator p2p.LocatorResponse, peer string) {
	fmt.Fprintf(out, "local locator count: %d\n", len(locator.Locator))
	fmt.Fprintln(out, "local locator:")
	for _, entry := range locator.Locator {
		fmt.Fprintf(out, "height=%d hash=%s\n", entry.Height, entry.Hash)
	}
	fmt.Fprintf(out, "peer: %s\n", peer)
}

func printForkCheck(out io.Writer, result p2p.ForkCheckResult) {
	if result.ForkDetected {
		fmt.Fprintln(out, "fork detected")
		if result.PeerHeight > result.LocalHeight {
			fmt.Fprintln(out, "peer is ahead but does not extend local tip")
		}
	} else {
		fmt.Fprintln(out, "no fork detected")
		switch result.Status {
		case "in_sync":
			fmt.Fprintln(out, "nodes in sync")
		case "peer_ahead":
			fmt.Fprintln(out, "peer is ahead")
			fmt.Fprintf(out, "missing blocks: %d\n", result.PeerAheadBlocks)
		case "local_ahead":
			fmt.Fprintln(out, "local is ahead")
			fmt.Fprintf(out, "peer missing blocks: %d\n", result.LocalAheadBlocks)
		}
	}
	fmt.Fprintf(out, "local height: %d\n", result.LocalHeight)
	fmt.Fprintf(out, "peer height: %d\n", result.PeerHeight)
	if result.CommonAncestorFound {
		fmt.Fprintf(out, "common ancestor height: %d\n", result.CommonAncestorHeight)
		fmt.Fprintf(out, "common ancestor hash: %s\n", result.CommonAncestorHash)
	}
	fmt.Fprintf(out, "local ahead blocks: %d\n", result.LocalAheadBlocks)
	fmt.Fprintf(out, "peer ahead blocks: %d\n", result.PeerAheadBlocks)
	fmt.Fprintf(out, "reorg supported: %t\n", result.ReorgSupported)
	if result.ForkDetected {
		fmt.Fprintln(out, "automatic reorg: disabled")
	}
}

func printForkInspect(out io.Writer, result p2p.ForkInspectResult) {
	if result.ForkDetected {
		fmt.Fprintln(out, "fork detected")
	} else {
		fmt.Fprintln(out, "no fork detected")
		if result.InSync {
			fmt.Fprintln(out, "chains in sync")
		}
	}
	fmt.Fprintf(out, "local height: %d\n", result.LocalHeight)
	fmt.Fprintf(out, "other height: %d\n", result.OtherHeight)
	if result.CommonAncestorFound {
		fmt.Fprintf(out, "common ancestor height: %d\n", result.CommonAncestorHeight)
		fmt.Fprintf(out, "common ancestor hash: %s\n", result.CommonAncestorHash)
	}
	fmt.Fprintf(out, "local ahead blocks: %d\n", result.LocalAheadBlocks)
	fmt.Fprintf(out, "other ahead blocks: %d\n", result.OtherAheadBlocks)
	fmt.Fprintf(out, "reorg supported: %t\n", result.ReorgSupported)
}

func printRemoteMempool(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "pending tx count: %.0f\n", info["pending_tx_count"])
	txs, _ := info["transactions"].([]any)
	for _, item := range txs {
		tx := item.(map[string]any)
		if _, detailed := tx["timestamp"]; detailed {
			fmt.Fprintf(out, "txid: %s\nfrom: %s\nto: %s\namount: %s\nfee: %s\nnonce: %.0f\ntimestamp: %.0f\nstatus: %s\n", tx["txid"], tx["from"], tx["to"], remoteAmountText(tx["amount"]), remoteAmountText(tx["fee"]), tx["nonce"], numberFromAny(tx["timestamp"]), tx["status"])
		} else {
			fmt.Fprintf(out, "id=%s from=%s to=%s amount=%s fee=%s nonce=%.0f\n", tx["id"], tx["from"], tx["to"], remoteAmountText(tx["amount"]), remoteAmountText(tx["fee"]), tx["nonce"])
		}
	}
}

func remoteAmountText(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	if number, ok := value.(float64); ok {
		return amount.Format(uint64(number)) + " " + config.Ticker
	}
	return fmt.Sprint(value)
}

func printRemoteTx(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "id: %s\n", info["id"])
	fmt.Fprintf(out, "status: %s\n", info["status"])
	if info["status"] == "confirmed" {
		fmt.Fprintf(out, "block_height: %.0f\n", info["block_height"])
	}
	fmt.Fprintf(out, "from: %s\n", info["from"])
	fmt.Fprintf(out, "to: %s\n", info["to"])
	fmt.Fprintf(out, "amount: %s\n", info["amount"])
	fmt.Fprintf(out, "fee: %s\n", info["fee"])
	fmt.Fprintf(out, "nonce: %.0f\n", info["nonce"])
	fmt.Fprintf(out, "coinbase: %t\n", info["coinbase"])
}

func printRemoteAddressInspect(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "address: %s\n", info["address"])
	if format, ok := info["format"]; ok {
		fmt.Fprintf(out, "format: %s\n", format)
	}
	if network, ok := info["network"]; ok {
		fmt.Fprintf(out, "network: %s\n", network)
	}
	if curve, ok := info["key_curve"]; ok {
		fmt.Fprintf(out, "key curve: %s\n", curve)
	}
	fmt.Fprintln(out, "exists: false")
	fmt.Fprintf(out, "confirmed balance: %s\n", info["confirmed_balance"])
	if value, ok := info["mature_balance"]; ok {
		fmt.Fprintf(out, "mature balance: %s\n", value)
	}
	if value, ok := info["immature_balance"]; ok {
		fmt.Fprintf(out, "immature balance: %s\n", value)
	}
	if value, ok := info["spendable_balance"]; ok {
		fmt.Fprintf(out, "spendable balance: %s\n", value)
	}
	fmt.Fprintf(out, "confirmed nonce: %.0f\n", info["confirmed_nonce"])
	fmt.Fprintf(out, "pending outgoing tx: %.0f\n", info["pending_outgoing_count"])
	fmt.Fprintf(out, "pending outgoing amount: %s\n", info["pending_outgoing_amount"])
	fmt.Fprintf(out, "pending incoming tx: %.0f\n", info["pending_incoming_count"])
	fmt.Fprintf(out, "pending incoming amount: %s\n", info["pending_incoming_amount"])
}

func printRemoteWalletList(out io.Writer, info map[string]any) {
	fmt.Fprintln(out, "addresses:")
	wallets, _ := info["wallets"].([]any)
	for _, item := range wallets {
		walletInfo, _ := item.(map[string]any)
		if walletInfo == nil {
			continue
		}
		fmt.Fprintf(out, "* %s\n", walletInfo["address"])
	}
}

func printRemoteNodeStatus(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "datadir: %s\n", info["datadir"])
	fmt.Fprintf(out, "rpc: %s\n", info["rpc_listen"])
	fmt.Fprintf(out, "p2p: %s\n", info["p2p_listen"])
	fmt.Fprintf(out, "p2p advertise: %s\n", info["p2p_advertise"])
	fmt.Fprintf(out, "height: %.0f\n", info["height"])
	fmt.Fprintf(out, "tip hash: %s\n", info["tip_hash"])
	fmt.Fprintf(out, "peers: %.0f\n", info["peers"])
	fmt.Fprintf(out, "mempool pending: %.0f\n", info["mempool_pending"])
	fmt.Fprintf(out, "locked: %t\n", info["locked"])
}

func printRemoteMineStatus(out io.Writer, info map[string]any) {
	status := fmt.Sprint(info["status"])
	fmt.Fprintf(out, "mining status: %s\n", status)
	if status == "idle" {
		return
	}
	fmt.Fprintf(out, "job id: %s\n", info["job_id"])
	if requested, ok := info["requested_blocks"].(float64); ok {
		fmt.Fprintf(out, "mined blocks: %.0f/%.0f\n", info["mined_blocks"], requested)
	} else {
		fmt.Fprintf(out, "mined blocks: %v\n", info["mined_blocks"])
	}
	fmt.Fprintf(out, "last height: %.0f\n", info["last_height"])
	fmt.Fprintf(out, "last hash: %s\n", info["last_hash"])
	if errText := fmt.Sprint(info["error"]); errText != "" && errText != "<nil>" {
		fmt.Fprintf(out, "error: %s\n", errText)
	}
}

func printRemoteDebugLocks(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "mining running: %t\n", info["mining_running"])
	fmt.Fprintf(out, "mining job id: %s\n", info["mining_job_id"])
	fmt.Fprintf(out, "height: %.0f\n", info["height"])
	fmt.Fprintf(out, "tip hash: %s\n", info["tip_hash"])
	fmt.Fprintf(out, "mempool count: %.0f\n", info["mempool_count"])
	fmt.Fprintf(out, "peer count: %.0f\n", info["peer_count"])
}

func printServiceRegistered(out io.Writer, info map[string]any) {
	fmt.Fprintln(out, "service node registered")
	fmt.Fprintf(out, "service node id: %s\n", info["service_node_id"])
	fmt.Fprintf(out, "owner address: %s\n", info["owner_address"])
	fmt.Fprintf(out, "status: %s\n", info["status"])
}

func printServiceHeartbeat(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "status: %s\n", info["status"])
	fmt.Fprintf(out, "uptime score: %.0f\n", numberFromAny(info["uptime_score"]))
	fmt.Fprintf(out, "latency score: %.0f\n", numberFromAny(info["latency_score"]))
	fmt.Fprintf(out, "bandwidth score: %.0f\n", numberFromAny(info["bandwidth_score"]))
	fmt.Fprintf(out, "reliability score: %.0f\n", numberFromAny(info["reliability_score"]))
	fmt.Fprintf(out, "abuse penalty: %.0f\n", numberFromAny(info["abuse_penalty"]))
	fmt.Fprintf(out, "service score: %.0f\n", numberFromAny(info["service_score"]))
	if note, ok := info["note"]; ok {
		fmt.Fprintf(out, "note: %s\n", note)
	}
}

func printServiceChallenge(out io.Writer, info map[string]any) {
	fmt.Fprintln(out, "service challenge created")
	fmt.Fprintf(out, "challenge id: %s\n", info["challenge_id"])
	fmt.Fprintf(out, "address: %s\n", info["address"])
	fmt.Fprintf(out, "status: %s\n", info["status"])
	fmt.Fprintf(out, "expires at: %.0f\n", numberFromAny(info["expires_at"]))
}

func printServiceChallengeSubmit(out io.Writer, info map[string]any) {
	fmt.Fprintln(out, "service challenge submitted")
	fmt.Fprintf(out, "challenge id: %s\n", info["challenge_id"])
	fmt.Fprintf(out, "status: %s\n", info["status"])
	fmt.Fprintf(out, "service score: %.0f\n", numberFromAny(info["service_score"]))
	if note, ok := info["note"]; ok {
		fmt.Fprintf(out, "note: %s\n", note)
	}
}

func printServiceScore(out io.Writer, score map[string]any) {
	fmt.Fprintf(out, "address: %s\n", score["address"])
	fmt.Fprintf(out, "uptime score: %.0f\n", numberFromAny(score["uptime_score"]))
	fmt.Fprintf(out, "latency score: %.0f\n", numberFromAny(score["latency_score"]))
	fmt.Fprintf(out, "bandwidth score: %.0f\n", numberFromAny(score["bandwidth_score"]))
	fmt.Fprintf(out, "reliability score: %.0f\n", numberFromAny(score["reliability_score"]))
	fmt.Fprintf(out, "abuse penalty: %.0f\n", numberFromAny(score["abuse_penalty"]))
	fmt.Fprintf(out, "service score: %.0f\n", numberFromAny(score["service_score"]))
	fmt.Fprintf(out, "simulated points: %.0f\n", numberFromAny(score["simulated_points"]))
	if _, ok := score["required_stake"]; ok {
		fmt.Fprintf(out, "required stake: %s %s\n", amount.Format(uint64(numberFromAny(score["required_stake"]))), config.Ticker)
		fmt.Fprintf(out, "active stake: %s %s\n", amount.Format(uint64(numberFromAny(score["active_stake"]))), config.Ticker)
		fmt.Fprintf(out, "stake eligible: %t\n", boolFromMap(score, "stake_eligible"))
		fmt.Fprintf(out, "collateral status: %s\n", score["collateral_status"])
		fmt.Fprintf(out, "eligible simulated points: %.0f\n", numberFromAny(score["eligible_simulated_points"]))
	}
	if note, ok := score["eligibility_note"]; ok && fmt.Sprint(note) != "" {
		fmt.Fprintf(out, "eligibility note: %s\n", note)
	}
	if note, ok := score["note"]; ok {
		fmt.Fprintf(out, "note: %s\n", note)
	}
}

func printServiceRewards(out io.Writer, address string, rewards []servicenode.Reward, note string) {
	fmt.Fprintf(out, "address: %s\n", address)
	if len(rewards) == 0 {
		fmt.Fprintln(out, "service rewards: 0")
	}
	for _, reward := range rewards {
		fmt.Fprintf(out, "epoch: %s service score: %d simulated points: %d reason: %s\n", reward.Epoch, reward.ServiceScore, reward.SimulatedPoints, reward.Reason)
	}
	fmt.Fprintf(out, "note: %s\n", note)
}

func printServiceRewardsMap(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "address: %s\n", info["address"])
	items, _ := info["rewards"].([]any)
	if len(items) == 0 {
		fmt.Fprintln(out, "service rewards: 0")
	}
	for _, item := range items {
		reward, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fmt.Fprintf(out, "epoch: %s service score: %.0f simulated points: %.0f reason: %s\n", reward["epoch"], numberFromAny(reward["service_score"]), numberFromAny(reward["simulated_points"]), reward["reason"])
	}
	if note, ok := info["note"]; ok {
		fmt.Fprintf(out, "note: %s\n", note)
	}
}

func printServiceList(out io.Writer, nodes []servicenode.Node) {
	fmt.Fprintf(out, "service nodes: %d\n", len(nodes))
	for _, node := range nodes {
		fmt.Fprintf(out, "id=%s address=%s status=%s score_penalty=%d endpoint=%s\n", node.ServiceNodeID, node.OwnerAddress, node.Status, node.AbusePenalty, node.AdvertisedEndpoint)
	}
}

func printServiceListMap(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "service nodes: %.0f\n", numberFromAny(info["count"]))
	items, _ := info["nodes"].([]any)
	for _, item := range items {
		node, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fmt.Fprintf(out, "id=%s address=%s status=%s score_penalty=%.0f endpoint=%s\n", node["service_node_id"], node["owner_address"], node["status"], numberFromAny(node["abuse_penalty"]), node["advertised_endpoint"])
	}
}

func printStakeInfoMap(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "staking enabled: %t\n", boolFromMap(info, "staking_enabled"))
	fmt.Fprintf(out, "min stake amount: %s %s\n", info["min_stake_amount"], config.Ticker)
	fmt.Fprintf(out, "min service stake: %s %s\n", info["min_service_stake"], config.Ticker)
	fmt.Fprintf(out, "unbonding period: %.0f blocks\n", numberFromAny(info["unbonding_period"]))
	fmt.Fprintf(out, "total active stake: %s %s\n", info["total_active_stake"], config.Ticker)
	fmt.Fprintf(out, "total unlocking stake: %s %s\n", info["total_unlocking_stake"], config.Ticker)
	fmt.Fprintf(out, "active stake records: %.0f\n", numberFromAny(info["active_stake_count"]))
}

func printStakeListMap(out io.Writer, info map[string]any) {
	items, _ := info["stakes"].([]any)
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		printStakeRecordMap(out, record)
	}
	fmt.Fprintf(out, "stake records: %.0f\n", numberFromAny(info["count"]))
}

func printStakeRecordMap(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "stake id: %s\n", info["stake_id"])
	fmt.Fprintf(out, "owner: %s\n", info["owner_address"])
	fmt.Fprintf(out, "amount: %s %s\n", info["amount"], config.Ticker)
	fmt.Fprintf(out, "status: %s\n", info["status"])
	fmt.Fprintf(out, "lock height: %.0f\n", numberFromAny(info["lock_height"]))
	fmt.Fprintf(out, "unlock height: %.0f\n", numberFromAny(info["unlock_height"]))
	fmt.Fprintf(out, "release height: %.0f\n", numberFromAny(info["release_height"]))
}

func printStakeLockMap(out io.Writer, info map[string]any) {
	fmt.Fprintln(out, "stake lock tx created")
	fmt.Fprintf(out, "tx id: %s\n", info["tx_id"])
	fmt.Fprintf(out, "stake id: %s\n", info["stake_id"])
	fmt.Fprintf(out, "address: %s\n", info["address"])
	fmt.Fprintf(out, "amount: %s %s\n", info["amount"], config.Ticker)
	fmt.Fprintln(out, "status: pending")
	fmt.Fprintln(out, "note: stake becomes active after tx is mined")
}

func printStakeUnlockMap(out io.Writer, info map[string]any) {
	fmt.Fprintln(out, "stake unlock tx created")
	fmt.Fprintf(out, "tx id: %s\n", info["tx_id"])
	fmt.Fprintf(out, "stake id: %s\n", info["stake_id"])
	fmt.Fprintf(out, "release height: %.0f\n", numberFromAny(info["release_height"]))
	fmt.Fprintln(out, "status: pending")
}

func printRemotePeerCheck(out io.Writer, info map[string]any) {
	fmt.Fprintln(out, "peer ok")
	fmt.Fprintf(out, "url: %s\n", info["url"])
	fmt.Fprintf(out, "node id: %s\n", info["node_id"])
	fmt.Fprintf(out, "network id: %s\n", info["network_id"])
	fmt.Fprintf(out, "chain id: %.0f\n", info["chain_id"])
	fmt.Fprintf(out, "height: %.0f\n", info["height"])
	fmt.Fprintf(out, "tip hash: %s\n", info["tip_hash"])
}

func printRemotePeerStatus(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "checked peers: %.0f\n", info["checked_peers"])
	fmt.Fprintf(out, "active: %.0f\n", info["active"])
	fmt.Fprintf(out, "bad: %.0f\n", info["bad"])
	fmt.Fprintf(out, "unknown: %.0f\n", info["unknown"])
	peers, _ := info["peers"].([]any)
	for _, item := range peers {
		peer, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fmt.Fprintf(out, "url=%s status=%s height=%.0f score=%.0f source=%s last_seen=%s last_status_check=%s latency_ms=%.0f reason=%q",
			peer["url"], peer["status"], peer["last_height"], peer["score"], peer["source"], peer["last_seen_at"], peer["last_status_check_at"], numberFromAny(peer["last_latency_ms"]), peer["reason"])
		if errText, ok := peer["last_error"].(string); ok && errText != "" {
			fmt.Fprintf(out, " error=%q", errText)
		}
		fmt.Fprintln(out)
	}
}

func printRemotePeerHealth(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "network id: %s\n", info["network_id"])
	fmt.Fprintf(out, "chain id: %.0f\n", numberFromAny(info["chain_id"]))
	fmt.Fprintf(out, "genesis hash: %s\n", info["genesis_hash"])
	fmt.Fprintf(out, "local height: %.0f\n", numberFromAny(info["local_height"]))
	fmt.Fprintf(out, "local tip: %s\n", info["local_tip"])
	fmt.Fprintf(out, "known peers: %.0f\n", numberFromAny(info["known_peer_count"]))
	fmt.Fprintf(out, "active peers: %.0f\n", numberFromAny(info["active_peer_count"]))
	fmt.Fprintf(out, "seed peers: %.0f\n", numberFromAny(info["seed_count"]))
	fmt.Fprintf(out, "best peer height: %.0f\n", numberFromAny(info["best_peer_height"]))
}

func printRemoteBroadcastSummary(out io.Writer, label string, value any) {
	summary, ok := value.(map[string]any)
	if !ok {
		return
	}
	fmt.Fprintf(out, "%s: success=%.0f failed=%.0f\n", label, summary["success"], summary["failed"])
	results, _ := summary["results"].([]any)
	for _, item := range results {
		result, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fmt.Fprintf(out, "  peer=%s ok=%t message=%s\n", result["peer"], result["ok"], result["message"])
	}
}

func printRemotePeerSync(out io.Writer, info map[string]any) {
	imported := info["imported_blocks"].(float64)
	before := info["local_height_before"].(float64)
	after := info["local_height_after"].(float64)
	if imported == 0 {
		fmt.Fprintln(out, "sync complete")
		fmt.Fprintln(out, "local chain already up to date")
		fmt.Fprintf(out, "height: %.0f\n", after)
		fmt.Fprintln(out, "imported blocks: 0")
		return
	}
	fmt.Fprintln(out, "sync started")
	fmt.Fprintf(out, "peers checked: %.0f\n", info["peers_checked"])
	fmt.Fprintf(out, "imported blocks: %.0f\n", imported)
	fmt.Fprintf(out, "old height: %.0f\n", before)
	fmt.Fprintf(out, "new height: %.0f\n", after)
	fmt.Fprintln(out, "sync complete")
}

func printRemotePeerSyncFailure(out io.Writer, info map[string]any) {
	if fmt.Sprint(info["error"]) == "fork detected" {
		fmt.Fprintln(out, "sync failed: fork detected")
		fmt.Fprintf(out, "local height: %.0f\n", numberFromAny(info["local_height_before"]))
		fmt.Fprintf(out, "peer height: %.0f\n", numberFromAny(info["peer_height"]))
		fmt.Fprintf(out, "common ancestor height: %.0f\n", numberFromAny(info["common_ancestor_height"]))
		if hash, ok := info["common_ancestor_hash"].(string); ok && hash != "" {
			fmt.Fprintf(out, "common ancestor hash: %s\n", hash)
		}
		if localWork, ok := info["local_cumulative_work"]; ok {
			fmt.Fprintf(out, "local cumulative work: %.0f\n", numberFromAny(localWork))
		}
		if peerWork, ok := info["peer_cumulative_work"]; ok {
			fmt.Fprintf(out, "peer cumulative work: %.0f\n", numberFromAny(peerWork))
		}
		if depth, ok := info["reorg_depth"]; ok {
			fmt.Fprintf(out, "reorg depth: %.0f\n", numberFromAny(depth))
		}
		if maxDepth, ok := info["max_reorg_depth"]; ok {
			fmt.Fprintf(out, "max reorg depth: %.0f\n", numberFromAny(maxDepth))
		}
		if decision, ok := info["decision"].(string); ok && decision != "" {
			fmt.Fprintf(out, "decision: %s\n", decision)
		}
		if reason, ok := info["reason"].(string); ok && reason != "" {
			fmt.Fprintf(out, "reason: %s\n", reason)
		}
		if _, ok := info["decision"]; !ok {
			fmt.Fprintln(out, "automatic reorg: disabled")
		}
		return
	}
	if msg, ok := info["message"].(string); ok && msg != "" {
		fmt.Fprintln(out, msg)
		return
	}
	if errText, ok := info["error"].(string); ok {
		fmt.Fprintln(out, errText)
	}
}

func printUpstreamResult(out io.Writer, result p2p.UpstreamBackfillResult) {
	fmt.Fprintf(out, "peer: %s\n", result.Peer)
	fmt.Fprintf(out, "ok: %t\n", result.OK)
	fmt.Fprintf(out, "decision: %s\n", result.Decision)
	fmt.Fprintf(out, "local height: %d\n", result.LocalHeight)
	fmt.Fprintf(out, "peer height: %d\n", result.PeerHeight)
	fmt.Fprintf(out, "local tip: %s\n", result.LocalTip)
	fmt.Fprintf(out, "peer tip: %s\n", result.PeerTip)
	fmt.Fprintf(out, "local work: %d\n", result.LocalWork)
	fmt.Fprintf(out, "peer work: %d\n", result.PeerWork)
	if result.CommonAncestorHash != "" {
		fmt.Fprintf(out, "common ancestor height: %d\n", result.CommonAncestorHeight)
		fmt.Fprintf(out, "common ancestor hash: %s\n", result.CommonAncestorHash)
	}
	if result.Pushed > 0 || result.Accepted > 0 {
		fmt.Fprintf(out, "pushed: %d\n", result.Pushed)
		fmt.Fprintf(out, "accepted: %d\n", result.Accepted)
	}
	if result.FailedHeight > 0 {
		fmt.Fprintf(out, "failed height: %d\n", result.FailedHeight)
	}
	if result.Reason != "" {
		fmt.Fprintf(out, "reason: %s\n", result.Reason)
	}
}

func printRemoteP2PDebug(out io.Writer, info map[string]any) {
	fmt.Fprintf(out, "node id: %s\n", info["node_id"])
	fmt.Fprintf(out, "height: %.0f\n", info["height"])
	fmt.Fprintf(out, "tip hash: %s\n", info["tip_hash"])
	peers, _ := info["peers"].([]any)
	fmt.Fprintf(out, "peers: %d\n", len(peers))
	for _, item := range peers {
		peer, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fmt.Fprintf(out, "url=%s status=%s score=%.0f latency=%.0fms last_error=%q\n", peer["url"], peer["status"], peer["score"], peer["last_latency_ms"], fmt.Sprint(peer["last_error"]))
	}
}

func printP2PPing(out io.Writer, result p2p.PingResult) {
	fmt.Fprintf(out, "p2p ping: %s\n", result.URL)
	printPingLine(out, "health", result.HealthLatencyMS, result.HealthError, "")
	statusExtra := ""
	if result.Height != 0 || result.TipHash != "" {
		statusExtra = fmt.Sprintf(" height=%d tip=%s", result.Height, result.TipHash)
	}
	printPingLine(out, "status", result.StatusLatencyMS, result.StatusError, statusExtra)
	handshakeExtra := ""
	if result.NodeID != "" {
		handshakeExtra = fmt.Sprintf(" node_id=%s", result.NodeID)
	}
	printPingLine(out, "handshake", result.HandshakeLatencyMS, result.HandshakeError, handshakeExtra)
	if result.OK {
		fmt.Fprintln(out, "result: ok")
		return
	}
	fmt.Fprintln(out, "result: failed")
	if result.Error != "" {
		fmt.Fprintf(out, "error: %s\n", result.Error)
	}
}

func printPingLine(out io.Writer, label string, latency int64, errText string, extra string) {
	if errText != "" {
		fmt.Fprintf(out, "%s: failed latency=%dms error=%q\n", label, latency, errText)
		return
	}
	fmt.Fprintf(out, "%s: ok latency=%dms%s\n", label, latency, extra)
}

func p2pPingResultFromMap(info map[string]any) p2p.PingResult {
	if info == nil {
		return p2p.PingResult{}
	}
	return p2p.PingResult{
		OK:                 boolFromMap(info, "ok"),
		URL:                stringFromMap(info, "url"),
		HealthLatencyMS:    int64FromMap(info, "health_latency_ms"),
		StatusLatencyMS:    int64FromMap(info, "status_latency_ms"),
		HandshakeLatencyMS: int64FromMap(info, "handshake_latency_ms"),
		Height:             uint64(int64FromMap(info, "height")),
		TipHash:            stringFromMap(info, "tip_hash"),
		NodeID:             stringFromMap(info, "node_id"),
		Error:              stringFromMap(info, "error"),
		HealthError:        stringFromMap(info, "health_error"),
		StatusError:        stringFromMap(info, "status_error"),
		HandshakeError:     stringFromMap(info, "handshake_error"),
	}
}

func boolFromMap(info map[string]any, key string) bool {
	value, _ := info[key].(bool)
	return value
}

func stringFromMap(info map[string]any, key string) string {
	value, _ := info[key].(string)
	return value
}

func localServiceMetadata() servicenode.Metadata {
	return servicenode.Metadata{ClientVersion: "dev", Platform: runtimePlatform(), UserAgent: "deskachain-cli"}
}

func runtimePlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

func scoreMap(score servicenode.Score) map[string]any {
	return map[string]any{
		"address":                   score.Address,
		"uptime_score":              float64(score.UptimeScore),
		"latency_score":             float64(score.LatencyScore),
		"bandwidth_score":           float64(score.BandwidthScore),
		"reliability_score":         float64(score.ReliabilityScore),
		"abuse_penalty":             float64(score.AbusePenalty),
		"service_score":             float64(score.ServiceScore),
		"simulated_points":          float64(score.SimulatedPoints),
		"eligible_simulated_points": float64(score.EligibleSimulatedPoints),
		"required_stake":            float64(score.RequiredStake),
		"active_stake":              float64(score.ActiveStake),
		"stake_eligible":            score.StakeEligible,
		"collateral_status":         score.CollateralStatus,
		"eligibility_note":          score.EligibilityNote,
		"note":                      score.Note,
	}
}

func int64FromMap(info map[string]any, key string) int64 {
	switch value := info[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

func firstN(value string, n int) string {
	if len(value) <= n {
		return value
	}
	return value[:n]
}

func printCompare(out io.Writer, local, peer map[string]any) {
	same := fmt.Sprint(local["network_id"]) == fmt.Sprint(peer["network_id"]) &&
		fmt.Sprint(local["chain_id"]) == fmt.Sprint(peer["chain_id"]) &&
		fmt.Sprint(local["genesis_hash"]) == fmt.Sprint(peer["genesis_hash"]) &&
		fmt.Sprint(local["protocol_version"]) == fmt.Sprint(peer["protocol_version"]) &&
		fmt.Sprint(local["height"]) == fmt.Sprint(peer["height"]) &&
		fmt.Sprint(local["tip_hash"]) == fmt.Sprint(peer["tip_hash"]) &&
		fmt.Sprint(local["total_supply"]) == fmt.Sprint(peer["total_supply"]) &&
		fmt.Sprint(local["pending_tx_count"]) == fmt.Sprint(peer["pending_tx_count"])
	if same {
		fmt.Fprintln(out, "nodes in sync")
		fmt.Fprintf(out, "network id: %s\n", local["network_id"])
		fmt.Fprintf(out, "chain id: %.0f\n", local["chain_id"])
		fmt.Fprintf(out, "height: %.0f\n", local["height"])
		fmt.Fprintf(out, "tip hash: %s\n", local["tip_hash"])
		fmt.Fprintf(out, "local work: %.0f\n", local["cumulative_work"])
		fmt.Fprintf(out, "peer work: %.0f\n", peer["cumulative_work"])
		return
	}
	fmt.Fprintln(out, "nodes differ")
	fmt.Fprintf(out, "network id: local=%s peer=%s\n", local["network_id"], peer["network_id"])
	fmt.Fprintf(out, "chain id: local=%.0f peer=%.0f\n", local["chain_id"], peer["chain_id"])
	fmt.Fprintf(out, "local height: %.0f\n", local["height"])
	fmt.Fprintf(out, "peer height: %.0f\n", peer["height"])
	fmt.Fprintf(out, "local tip: %s\n", local["tip_hash"])
	fmt.Fprintf(out, "peer tip: %s\n", peer["tip_hash"])
	fmt.Fprintf(out, "local work: %.0f\n", local["cumulative_work"])
	fmt.Fprintf(out, "peer work: %.0f\n", peer["cumulative_work"])
}

func locatorFromMap(info map[string]any) p2p.LocatorResponse {
	resp := p2p.LocatorResponse{
		Height:  uint64(numberFromAny(info["height"])),
		TipHash: fmt.Sprint(info["tip_hash"]),
	}
	items, _ := info["locator"].([]any)
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		resp.Locator = append(resp.Locator, chain.BlockLocatorEntry{
			Height: uint64(numberFromAny(entry["height"])),
			Hash:   fmt.Sprint(entry["hash"]),
		})
	}
	return resp
}

func forkCheckFromMap(info map[string]any) p2p.ForkCheckResult {
	return p2p.ForkCheckResult{
		ForkDetected:         boolFromAny(info["fork_detected"]),
		InSync:               boolFromAny(info["in_sync"]),
		LocalHeight:          uint64(numberFromAny(info["local_height"])),
		LocalTip:             fmt.Sprint(info["local_tip"]),
		PeerHeight:           uint64(numberFromAny(info["peer_height"])),
		PeerTip:              fmt.Sprint(info["peer_tip"]),
		CommonAncestorFound:  boolFromAny(info["common_ancestor_found"]),
		CommonAncestorHeight: uint64(numberFromAny(info["common_ancestor_height"])),
		CommonAncestorHash:   fmt.Sprint(info["common_ancestor_hash"]),
		LocalAheadBlocks:     uint64(numberFromAny(info["local_ahead_blocks"])),
		PeerAheadBlocks:      uint64(numberFromAny(info["peer_ahead_blocks"])),
		ReorgSupported:       boolFromAny(info["reorg_supported"]),
		Status:               fmt.Sprint(info["status"]),
		Error:                fmt.Sprint(info["error"]),
	}
}

func commonAncestorFromForkMap(info map[string]any) p2p.CommonAncestorResponse {
	if !boolFromAny(info["common_ancestor_found"]) {
		return p2p.CommonAncestorResponse{Found: false, Error: "no common ancestor found"}
	}
	return p2p.CommonAncestorResponse{
		Found:  true,
		Height: uint64(numberFromAny(info["common_ancestor_height"])),
		Hash:   fmt.Sprint(info["common_ancestor_hash"]),
	}
}

func commonAncestorFromMap(info map[string]any) p2p.CommonAncestorResponse {
	resp := p2p.CommonAncestorResponse{
		Found: boolFromAny(info["found"]),
		Error: fmt.Sprint(info["error"]),
	}
	if resp.Found {
		resp.Height = uint64(numberFromAny(info["height"]))
		resp.Hash = fmt.Sprint(info["hash"])
	}
	return resp
}

func numberFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint16:
		return float64(v)
	case uint:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func boolFromAny(value any) bool {
	v, _ := value.(bool)
	return v
}

func normalizeRPCURL(value string) string {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if strings.HasPrefix(value, ":") {
		return "http://127.0.0.1" + value
	}
	return "http://" + value
}

func defaultAdvertiseP2P(value string) string {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if strings.HasPrefix(value, ":") {
		return "http://127.0.0.1" + value
	}
	return "http://" + value
}

func suggestedRemoteCommand(args []string) string {
	if len(args) == 0 {
		return "node status"
	}
	switch args[0] {
	case "mine":
		if len(args) > 1 && args[1] == "status" {
			return "mine status"
		}
		return "mine --address <address> --blocks <n>"
	case "send":
		return "send --from <from> --to <to> --amount <amount>"
	case "peer":
		if len(args) > 1 {
			switch args[1] {
			case "sync":
				return "peer sync"
			case "check":
				return "peer check <url>"
			case "add":
				return "peer add <url>"
			case "remove":
				return "peer remove <url>"
			case "status":
				return "peer status"
			case "connect":
				return "peer connect <url>"
			}
		}
	case "p2p":
		return strings.Join(args, " ")
	case "debug":
		return strings.Join(args, " ")
	case "mempool":
		if len(args) > 1 && args[1] == "clear" {
			return "mempool clear --yes"
		}
	}
	return strings.Join(args, " ")
}

func (a App) openInitializedChain() (*chain.Blockchain, func(), error) {
	bc, closeFn, err := a.openChain()
	if err != nil {
		return nil, nil, err
	}
	has, err := bc.HasChain()
	if err != nil {
		closeFn()
		return nil, nil, err
	}
	if !has {
		closeFn()
		return nil, nil, errors.New("chain is not initialized")
	}
	if err := config.EnsureNetworkMatches(a.paths, a.profile); err != nil {
		closeFn()
		return nil, nil, err
	}
	return bc, closeFn, nil
}

func safeDataDir(datadir string) error {
	if datadir == "" {
		return errors.New("unsafe datadir: empty path")
	}
	clean := filepath.Clean(datadir)
	if clean == "." || clean == ".." {
		return fmt.Errorf("unsafe datadir: %s", datadir)
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return err
	}
	volume := filepath.VolumeName(abs)
	root := volume + string(os.PathSeparator)
	if volume == "" {
		root = string(os.PathSeparator)
	}
	if abs == root {
		return fmt.Errorf("unsafe datadir: %s", datadir)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileState(path string) string {
	if fileExists(path) {
		return "exists"
	}
	return "missing"
}

func Main() {
	app := New(os.Stdout)
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
