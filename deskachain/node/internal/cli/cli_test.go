package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"deskachain/internal/amount"
	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/crypto"
	"deskachain/internal/mempool"
	"deskachain/internal/mining"
	"deskachain/internal/nodestate"
	"deskachain/internal/p2p"
	"deskachain/internal/rpc"
	"deskachain/internal/storage"
	"deskachain/internal/state"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestCustomDataDirInitCreatesGenesisOnce(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "chain.db")); err != nil {
		t.Fatal(err)
	}
	first := out.String()
	if !strings.Contains(first, "chain initialized") {
		t.Fatalf("unexpected first init output: %s", first)
	}
	out.Reset()
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	second := out.String()
	if !strings.Contains(second, "chain already initialized") || !strings.Contains(second, "height: 0") {
		t.Fatalf("unexpected second init output: %s", second)
	}
}

func TestVersionCommandPrintsBuildMetadata(t *testing.T) {
	var out bytes.Buffer
	app := New(&out)
	if err := app.Run([]string{"version"}); err != nil {
		t.Fatal(err)
	}
	output := out.String()
	for _, want := range []string{"DesKaChain", "version:", "commit:", "built:", "go:", "os/arch:", "networks: localnet,testnet", "mainnet: not available"} {
		assertOutputContains(t, output, want)
	}
}

func TestHelpCommandPrintsUsage(t *testing.T) {
	var out bytes.Buffer
	app := New(&out)
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "usage: deskachain")
	assertOutputContains(t, out.String(), "version")
}

func TestInitWritesNetworkMetadataLocalnet(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"--network", "localnet", "init"}); err != nil {
		t.Fatal(err)
	}
	meta, ok, err := config.ReadNetworkMetadata(config.NewPaths(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || meta.Network != config.Localnet().Name || meta.ChainID != config.Localnet().ChainID || meta.NetworkID != config.Localnet().NetworkID {
		t.Fatalf("unexpected localnet metadata ok=%t meta=%+v", ok, meta)
	}
}

func TestInitWritesNetworkMetadataTestnet(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"--network", "testnet", "init"}); err != nil {
		t.Fatal(err)
	}
	meta, ok, err := config.ReadNetworkMetadata(config.NewPaths(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || meta.Network != config.Testnet().Name || meta.ChainID != config.Testnet().ChainID || meta.NetworkID != config.Testnet().NetworkID {
		t.Fatalf("unexpected testnet metadata ok=%t meta=%+v", ok, meta)
	}
	if meta.GenesisHash != chain.GenesisBlockForNetwork(config.Testnet()).Hash {
		t.Fatalf("unexpected testnet genesis metadata: %+v", meta)
	}
}

func TestWalletNewUsesDatadirTestnetProfile(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"--network", "testnet", "init"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "new"}); err != nil {
		t.Fatal(err)
	}
	address := strings.TrimSpace(out.String())
	if err := crypto.ValidateAddressForNetwork(address, config.Testnet()); err != nil {
		t.Fatalf("wallet address is not testnet-valid: %v address=%s", err, address)
	}
	if err := crypto.ValidateAddressForNetwork(address, config.Localnet()); err == nil {
		t.Fatalf("testnet wallet unexpectedly validates on localnet: %s", address)
	}
}

func TestWalletNewNetworkMismatchRejected(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"--network", "testnet", "init"}); err != nil {
		t.Fatal(err)
	}
	err := app.Run([]string{"--network", "localnet", "wallet", "new"})
	if err == nil || !strings.Contains(err.Error(), "datadir initialized for testnet") {
		t.Fatalf("expected datadir network mismatch, got %v", err)
	}
}

func TestWalletNewLocalnetBackwardCompatible(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "new"}); err != nil {
		t.Fatal(err)
	}
	address := strings.TrimSpace(out.String())
	if err := crypto.ValidateAddressForNetwork(address, config.Localnet()); err != nil {
		t.Fatalf("wallet address is not localnet-valid: %v address=%s", err, address)
	}
}

func TestStartUsesDatadirNetworkMetadata(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"--network", "testnet", "init"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"network", "info"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "network name: testnet")
	assertOutputContains(t, out.String(), "network id: idr-testnet-1")
	assertOutputContains(t, out.String(), "chain id: 777101")
}

func TestMainnetProfileRejectsTamperedGenesis(t *testing.T) {
	profile := config.Mainnet()
	profile.GenesisHash = "tampered"
	if err := config.ValidateNetworkProfile(profile); err == nil || !strings.Contains(err.Error(), "mainnet genesis hash is not frozen") {
		t.Fatalf("expected frozen mainnet genesis rejection, got %v", err)
	}
}

func TestMainnetCLIMetadataGenesisMismatchRejected(t *testing.T) {
	dir := t.TempDir()
	profile := config.Mainnet()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := config.WriteNetworkMetadata(config.NewPaths(dir), profile, "tampered"); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"--network", "mainnet", "network", "info"}); err == nil || !strings.Contains(err.Error(), "datadir genesis mismatch for mainnet") {
		t.Fatalf("expected mainnet metadata genesis mismatch rejection, got %v", err)
	}
}

func TestMainnetCLIRequiresExplicitSelection(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"--network", "mainnet", "network", "info"}); err != nil {
		t.Fatalf("explicit mainnet selection should pass profile validation: %v", err)
	}
	if !strings.Contains(out.String(), "network name: mainnet") || !strings.Contains(out.String(), "genesis hash: "+config.MainnetGenesisHash) {
		t.Fatalf("unexpected mainnet network info: %s", out.String())
	}
}

func TestMainnetOperationalCommandsRemainGated(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"--network", "mainnet", "init"},
		{"--network", "mainnet", "wallet", "new"},
		{"--network", "mainnet", "mine", "--address", "a", "--blocks", "1"},
		{"--network", "mainnet", "node", "start"},
		{"--network", "mainnet", "service", "register", "--address", "a", "--endpoint", "https://example.invalid"},
		{"--network", "mainnet", "stake", "lock", "--address", "a", "--amount", "1"},
	} {
		app := New(io.Discard).WithDataDir(dir)
		if err := app.Run(args); err == nil || !strings.Contains(err.Error(), "mainnet is not operational: launch gate is closed") {
			t.Fatalf("expected mainnet launch gate for %v, got %v", args, err)
		}
	}
}


func TestDatadirNetworkMismatchRejected(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"--network", "localnet", "init"}); err != nil {
		t.Fatal(err)
	}
	err := app.Run([]string{"--network", "testnet", "node", "start", "--rpc", ":0", "--p2p", ":0"})
	if err == nil || !strings.Contains(err.Error(), "datadir initialized for localnet, cannot start as testnet") {
		t.Fatalf("expected network mismatch, got %v", err)
	}
}

func TestMainnetStandaloneRPCIsGated(t *testing.T) {
	app := New(io.Discard).WithDataDir(t.TempDir())
	err := app.Run([]string{"--network", "mainnet", "rpc", "--addr", ":0"})
	if err == nil || !strings.Contains(err.Error(), "mainnet is not operational: launch gate is closed") {
		t.Fatalf("expected standalone mainnet RPC launch gate, got %v", err)
	}
}

func TestMainnetOperationalCommandsRemainGatedWhenOverridesAreSet(t *testing.T) {
	t.Setenv("IDR_ALLOW_ISOLATED_MINING", "true")
	t.Setenv("IDR_ALLOW_ISOLATED_WRITES", "true")
	app := New(io.Discard).WithDataDir(t.TempDir())
	err := app.Run([]string{"--network", "mainnet", "node", "start"})
	if err == nil || !strings.Contains(err.Error(), "mainnet is not operational: launch gate is closed") {
		t.Fatalf("expected mainnet launch gate to take precedence, got %v", err)
	}
}

func TestPublicRPCUnsafeWalletAdminBindRejected(t *testing.T) {
	if err := validatePublicRPCMode(true, ":8811", true, false); err == nil || !strings.Contains(err.Error(), "wallet RPC cannot be enabled") {
		t.Fatalf("expected public wallet RPC rejection, got %v", err)
	}
	if err := validatePublicRPCMode(true, "0.0.0.0:8811", false, true); err == nil || !strings.Contains(err.Error(), "admin RPC cannot be enabled") {
		t.Fatalf("expected public admin RPC rejection, got %v", err)
	}
	if err := validatePublicRPCMode(true, "127.0.0.1:8811", true, true); err != nil {
		t.Fatalf("localhost trusted public RPC mode should not be rejected by bind guard: %v", err)
	}
	if err := validatePublicRPCMode(false, ":8811", true, true); err != nil {
		t.Fatalf("local development mode should not be rejected: %v", err)
	}
}

func TestNetworkMismatchRejectedBothDirections(t *testing.T) {
	var out bytes.Buffer
	localDir := t.TempDir()
	localApp := New(&out).WithDataDir(localDir)
	if err := localApp.Run([]string{"--network", "localnet", "init"}); err != nil {
		t.Fatal(err)
	}
	if err := localApp.Run([]string{"--network", "testnet", "chain", "info"}); err == nil || !strings.Contains(err.Error(), "datadir initialized for localnet, cannot start as testnet") {
		t.Fatalf("expected localnet-as-testnet mismatch, got %v", err)
	}

	testnetDir := t.TempDir()
	testnetApp := New(&out).WithDataDir(testnetDir)
	if err := testnetApp.Run([]string{"--network", "testnet", "init"}); err != nil {
		t.Fatal(err)
	}
	if err := testnetApp.Run([]string{"--network", "localnet", "chain", "info"}); err == nil || !strings.Contains(err.Error(), "datadir initialized for testnet, cannot start as localnet") {
		t.Fatalf("expected testnet-as-localnet mismatch, got %v", err)
	}
}

func TestBootnodeAddedToPeerStore(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	store := p2p.NewPeerStore(paths.Peers)
	boot := "http://127.0.0.1:9611"
	if _, err := addStartupPeers(store, startupPeerInputs{Bootnodes: []string{boot, boot}}); err != nil {
		t.Fatal(err)
	}
	peers, err := store.LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 || peers[0].URL != boot || !strings.Contains(peers[0].Source, "bootnode") {
		t.Fatalf("unexpected bootnode peers: %#v", peers)
	}
	if _, err := addStartupPeers(store, startupPeerInputs{Bootnodes: []string{"not-a-url"}}); err == nil || !strings.Contains(err.Error(), "peer url") {
		t.Fatalf("expected invalid bootnode error, got %v", err)
	}
}

func TestSeedPeersParseNormalizeDedupeAndFile(t *testing.T) {
	seedFile := filepath.Join(t.TempDir(), "testnet-seeds.txt")
	if err := os.WriteFile(seedFile, []byte(`
# comment
http://127.0.0.1:9811/

http://127.0.0.1:9812 # inline comment
`), 0644); err != nil {
		t.Fatal(err)
	}
	seeds, err := collectSeedPeers(config.NetworkConfig{SeedPeers: []string{"http://127.0.0.1:9810/"}}, "http://127.0.0.1:9811", "http://127.0.0.1:9812/,http://127.0.0.1:9813", seedFile)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"http://127.0.0.1:9810", "http://127.0.0.1:9811", "http://127.0.0.1:9812", "http://127.0.0.1:9813"}
	if strings.Join(seeds, ",") != strings.Join(want, ",") {
		t.Fatalf("seeds = %#v want %#v", seeds, want)
	}
	var repeated multiStringFlag
	if err := repeated.Set("http://127.0.0.1:9814/"); err != nil {
		t.Fatal(err)
	}
	if err := repeated.Set("http://127.0.0.1:9815/"); err != nil {
		t.Fatal(err)
	}
	seeds, err = collectSeedPeers(config.NetworkConfig{}, repeated.String(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(seeds, ",") != "http://127.0.0.1:9814,http://127.0.0.1:9815" {
		t.Fatalf("repeated seed peers = %#v", seeds)
	}
}

func TestUpstreamPeersParseNormalizeDedupeAndFile(t *testing.T) {
	upstreamFile := filepath.Join(t.TempDir(), "upstreams.txt")
	if err := os.WriteFile(upstreamFile, []byte("http://127.0.0.1:9911/\nhttp://127.0.0.1:9912 # inline\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var repeated multiStringFlag
	if err := repeated.Set("http://127.0.0.1:9910/"); err != nil {
		t.Fatal(err)
	}
	if err := repeated.Set("http://127.0.0.1:9911/"); err != nil {
		t.Fatal(err)
	}
	peers, err := collectUpstreamPeers(repeated.String(), "http://127.0.0.1:9912/,http://127.0.0.1:9913", upstreamFile)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"http://127.0.0.1:9910", "http://127.0.0.1:9911", "http://127.0.0.1:9912", "http://127.0.0.1:9913"}
	if strings.Join(peers, ",") != strings.Join(want, ",") {
		t.Fatalf("upstream peers = %#v want %#v", peers, want)
	}
}

func TestBootstrapPeersContinuesAfterFailedSeed(t *testing.T) {
	profile := config.Testnet()
	local := initCLIProfileNode(t, profile)
	peer := initCLIProfileNode(t, profile)
	mux := http.NewServeMux()
	p2p.NewServerWithAdvertiseAndProfile(peer, ":0", "", profile).Register(mux)
	onlineSeed := httptest.NewServer(mux)
	defer onlineSeed.Close()
	offlineSeed := httptest.NewServer(http.NotFoundHandler())
	offlineURL := offlineSeed.URL
	offlineSeed.Close()

	var out bytes.Buffer
	app := App{out: &out, paths: local, profile: profile}
	app.bootstrapPeers([]string{offlineURL, onlineSeed.URL}, "", nil, config.DefaultMaxReorgDepth(profile), nil)

	peers, err := p2p.NewPeerStore(local.Peers).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	foundOffline, foundOnline := false, false
	for _, peer := range peers {
		switch peer.URL {
		case offlineURL:
			foundOffline = peer.Status == p2p.PeerStatusOffline && peer.FailureCount > 0
		case onlineSeed.URL:
			foundOnline = peer.Status == p2p.PeerStatusActive && peer.NetworkID == profile.NetworkID && peer.SuccessCount > 0
		}
	}
	if !foundOffline || !foundOnline {
		t.Fatalf("bootstrap did not preserve failed seed and continue to online seed: %+v", peers)
	}
}

func TestSeedFileInvalidURLRejected(t *testing.T) {
	seedFile := filepath.Join(t.TempDir(), "bad-seeds.txt")
	if err := os.WriteFile(seedFile, []byte("not-a-url\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := parseSeedFile(seedFile)
	if err == nil || !strings.Contains(err.Error(), "invalid seed peer") {
		t.Fatalf("expected invalid seed peer error, got %v", err)
	}
}

func TestNodeConfigAppliesSeedPeersAndFile(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var rpcAddr, p2pAddr, advertise, seedPeersText, seedFile, cors string
	var upstreamPeersText, upstreamFile string
	var publicRPC, enableWallet, enableMiner, enableAdmin, enableService bool
	var rateLimit, maxPeers, minMiningPeers, minWritePeers int
	var allowIsolatedMining, allowIsolatedWrites bool
	maxReorgDepth := uint64(64)
	cfg := nodeConfigFile{}
	cfg.P2P.SeedPeers = []string{"http://127.0.0.1:9811", "http://127.0.0.1:9812"}
	cfg.P2P.SeedFile = "./seeds.txt"
	cfg.P2P.UpstreamPeers = []string{"http://127.0.0.1:9911", "http://127.0.0.1:9912"}
	cfg.P2P.UpstreamFile = "./upstreams.txt"
	cfg.P2P.MinMiningPeers = 2
	allowMine := true
	cfg.P2P.AllowIsolatedMine = &allowMine
	cfg.P2P.MinWritePeers = 3
	allowWrite := true
	cfg.P2P.AllowIsolatedWrite = &allowWrite

	applyNodeConfig(fs, cfg, &rpcAddr, &p2pAddr, &advertise, &seedPeersText, &seedFile, &upstreamPeersText, &upstreamFile, &publicRPC, &enableWallet, &enableMiner, &enableAdmin, &enableService, &cors, &rateLimit, &maxPeers, &maxReorgDepth, &minMiningPeers, &allowIsolatedMining, &minWritePeers, &allowIsolatedWrites)

	if seedPeersText != "http://127.0.0.1:9811,http://127.0.0.1:9812" {
		t.Fatalf("unexpected config seed peers: %q", seedPeersText)
	}
	if seedFile != "./seeds.txt" {
		t.Fatalf("unexpected config seed file: %q", seedFile)
	}
	if upstreamPeersText != "http://127.0.0.1:9911,http://127.0.0.1:9912" {
		t.Fatalf("unexpected config upstream peers: %q", upstreamPeersText)
	}
	if upstreamFile != "./upstreams.txt" {
		t.Fatalf("unexpected config upstream file: %q", upstreamFile)
	}
	if minMiningPeers != 2 || !allowIsolatedMining || minWritePeers != 3 || !allowIsolatedWrites {
		t.Fatalf("unexpected isolated guard config: minMining=%d allowMining=%t minWrite=%d allowWrites=%t", minMiningPeers, allowIsolatedMining, minWritePeers, allowIsolatedWrites)
	}
}

func TestSeedPeersPersistSourceAndSkipSelf(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	store := p2p.NewPeerStore(paths.Peers)
	summary, err := addStartupPeers(store, startupPeerInputs{
		SeedPeers: []string{"http://127.0.0.1:9811/", "http://127.0.0.1:9812"},
		SelfURL:   "http://127.0.0.1:9811",
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.SeedsAdded != 1 || summary.SkippedSelf != 1 {
		t.Fatalf("unexpected seed summary: %#v", summary)
	}
	peers, err := store.LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 || peers[0].URL != "http://127.0.0.1:9812" || peers[0].Source != "seed" {
		t.Fatalf("unexpected seed peers: %#v", peers)
	}
}

func fundedTestProfile() config.NetworkConfig {
	profile := config.Localnet()
	profile.Economic.BlockSubsidy = config.InitialBlockReward
	profile.Economic.FeeOnlyBlocks = false
	return profile
}

func TestMiningNBlocksIncreasesHeightAndBalance(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir).WithProfile(fundedTestProfile())
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "new"}); err != nil {
		t.Fatal(err)
	}
	addr := strings.TrimSpace(out.String())
	out.Reset()
	if err := app.Run([]string{"mine", "--address", addr, "--blocks", "2"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "mining complete") {
		t.Fatalf("missing mining summary: %s", out.String())
	}
	out.Reset()
	if err := app.Run([]string{"balance", addr}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "confirmed balance: 100 IDR")
	assertOutputContains(t, out.String(), "mature balance: 0 IDR")
	assertOutputContains(t, out.String(), "spendable balance: 0 IDR")
	out.Reset()
	if err := app.Run([]string{"chain", "info"}); err != nil {
		t.Fatal(err)
	}
	info := out.String()
	for _, want := range []string{"height: 2", "blocks: 3", "coinbase blocks: 2", "total transactions: 2", "coinbase transactions: 2", "normal transactions: 0", "total supply: 100 IDR"} {
		if !strings.Contains(info, want) {
			t.Fatalf("chain info missing %q:\n%s", want, info)
		}
	}
}

func TestServiceRewardSimulationDoesNotMutateConsensus(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "new"}); err != nil {
		t.Fatal(err)
	}
	addr := strings.TrimSpace(out.String())
	out.Reset()
	if err := app.Run([]string{"chain", "info"}); err != nil {
		t.Fatal(err)
	}
	beforeInfo := out.String()
	out.Reset()
	if err := app.Run([]string{"balance", addr}); err != nil {
		t.Fatal(err)
	}
	beforeBalance := out.String()

	out.Reset()
	if err := app.Run([]string{"service", "register", "--address", addr, "--endpoint", "http://127.0.0.1:9501"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"service", "heartbeat", "--address", addr, "--endpoint", "http://127.0.0.1:9501"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"service", "challenge", "create", "--address", addr}); err != nil {
		t.Fatal(err)
	}
	challengeID := extractLineValue(out.String(), "challenge id: ")
	out.Reset()
	if err := app.Run([]string{"service", "challenge", "submit", "--challenge-id", challengeID, "--latency-ms", "50", "--bytes-up", "10000000", "--bytes-down", "50000000", "--success", "true"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"service", "rewards", "--address", addr}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "not spendable IDR") {
		t.Fatalf("missing simulation note: %s", out.String())
	}

	out.Reset()
	if err := app.Run([]string{"chain", "info"}); err != nil {
		t.Fatal(err)
	}
	afterInfo := out.String()
	if extractLineValue(afterInfo, "total supply: ") != extractLineValue(beforeInfo, "total supply: ") {
		t.Fatalf("service rewards changed total supply\nbefore=%s\nafter=%s", beforeInfo, afterInfo)
	}
	out.Reset()
	if err := app.Run([]string{"balance", addr}); err != nil {
		t.Fatal(err)
	}
	if out.String() != beforeBalance {
		t.Fatalf("service rewards changed balance\nbefore=%s\nafter=%s", beforeBalance, out.String())
	}
	out.Reset()
	if err := app.Run([]string{"chain", "validate"}); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteServiceRegisterCommand(t *testing.T) {
	addr := mustTestWallet(t).Address
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/service/register" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["address"] != addr {
			t.Fatalf("unexpected address: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "service_node_id": "svc_test", "owner_address": addr, "status": "registered"})
	}))
	defer server.Close()
	var out bytes.Buffer
	if err := New(&out).Run([]string{"--rpc-url", server.URL, "service", "register", "--address", addr, "--endpoint", "http://127.0.0.1:9501"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "service node registered")
	assertOutputContains(t, out.String(), "service node id: svc_test")
}

func TestMineRejectsInvalidChecksumAddress(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	addr := createTestWallet(t, app, &out)
	bad := flipLastBase58(addr)
	out.Reset()
	if err := app.Run([]string{"mine", "--address", bad, "--blocks", "1"}); err == nil || !strings.Contains(err.Error(), "invalid address") {
		t.Fatalf("expected invalid address error, got output=%q err=%v", out.String(), err)
	}
}

func TestChainInfoStatsGenesisCoinbaseAndNormalTx(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir).WithProfile(fundedTestProfile())
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"chain", "info"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"height: 0", "blocks: 1", "total transactions: 0", "coinbase transactions: 0", "normal transactions: 0"} {
		assertOutputContains(t, out.String(), want)
	}
	miner := createTestWallet(t, app, &out)
	receiver := createTestWallet(t, app, &out)
	out.Reset()
	if err := app.Run([]string{"mine", "--address", miner, "--blocks", "11"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"chain", "info"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"height: 11", "blocks: 12", "coinbase blocks: 11", "total transactions: 11", "coinbase transactions: 11", "normal transactions: 0", "circulating supply: 50 IDR"} {
		assertOutputContains(t, out.String(), want)
	}
	out.Reset()
	if err := app.Run([]string{"send", "--from", miner, "--to", receiver, "--amount", "10"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"mempool", "list", "--detail"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "pending tx count: 1")
	assertOutputContains(t, out.String(), "txid:")
	assertOutputContains(t, out.String(), "status: pending")
	out.Reset()
	if err := app.Run([]string{"mine", "--address", miner, "--blocks", "1"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"chain", "info"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"height: 12", "total transactions: 13", "coinbase transactions: 12", "normal transactions: 1"} {
		assertOutputContains(t, out.String(), want)
	}
}

func TestChainValidatePassesAndDetectsTampering(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "new"}); err != nil {
		t.Fatal(err)
	}
	addr := strings.TrimSpace(out.String())
	if err := app.Run([]string{"mine", "--address", addr, "--blocks", "1"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"chain", "validate"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "chain valid") {
		t.Fatalf("unexpected validate output: %s", out.String())
	}
	bc, closeFn, err := app.openChain()
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := bc.Blocks()
	closeFn()
	if err != nil {
		t.Fatal(err)
	}
	blocks[1].PreviousHash = "bad"
	if _, err := chain.ValidateChainWithNetwork(blocks, config.Localnet()); err == nil || !strings.Contains(err.Error(), "previous_hash mismatch") {
		t.Fatalf("expected tamper error, got %v", err)
	}
}

func TestWalletListAndExportSafety(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"wallet", "new"}); err != nil {
		t.Fatal(err)
	}
	addr := strings.TrimSpace(out.String())
	out.Reset()
	if err := app.Run([]string{"wallet", "list"}); err != nil {
		t.Fatal(err)
	}
	list := out.String()
	if !strings.Contains(list, "address="+addr) || strings.Contains(list, "private") {
		t.Fatalf("unsafe wallet list output: %s", list)
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "export", "--address", addr}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Refusing to print private key") {
		t.Fatalf("expected export refusal, got: %s", out.String())
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "export", "--address", addr, "--show-private-key"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "private_key:") {
		t.Fatalf("expected private key export, got: %s", out.String())
	}
}

func TestWalletNewInspectExportImportUsesIDRBase58Secp256k1(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	addr := createTestWallet(t, app, &out)
	if !strings.HasPrefix(addr, "IDR") {
		t.Fatalf("wallet address = %s", addr)
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "inspect", "--address", addr}); err != nil {
		t.Fatal(err)
	}
	inspect := out.String()
	for _, want := range []string{"format: base58check", "network: localnet", "key curve: secp256k1"} {
		if !strings.Contains(inspect, want) {
			t.Fatalf("inspect missing %q:\n%s", want, inspect)
		}
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "export", "--address", addr, "--show-private-key"}); err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(?m)^private_key: ([0-9a-f]{64})$`).FindStringSubmatch(out.String())
	if len(match) != 2 {
		t.Fatalf("private key not exported as 64-char hex:\n%s", out.String())
	}
	importDir := t.TempDir()
	importApp := New(&out).WithDataDir(importDir)
	out.Reset()
	if err := importApp.Run([]string{"wallet", "import", "--private-key", match[1]}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "address: "+addr) {
		t.Fatalf("import did not reproduce address %s:\n%s", addr, out.String())
	}
}

func TestSendMempoolNonceAndPendingOverspend(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir).WithProfile(fundedTestProfile())
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "new"}); err != nil {
		t.Fatal(err)
	}
	from := strings.TrimSpace(out.String())
	out.Reset()
	if err := app.Run([]string{"wallet", "new"}); err != nil {
		t.Fatal(err)
	}
	to := strings.TrimSpace(out.String())
	if err := app.Run([]string{"mine", "--address", from, "--blocks", "11"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"send", "--from", from, "--to", to, "--amount", "30"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "nonce: 1") {
		t.Fatalf("first send did not use nonce 1: %s", out.String())
	}
	out.Reset()
	if err := app.Run([]string{"send", "--from", from, "--to", to, "--amount", "20"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "nonce: 2") {
		t.Fatalf("second send did not use nonce 2: %s", out.String())
	}
	if err := app.Run([]string{"send", "--from", from, "--to", to, "--amount", "0.00000001"}); err == nil {
		t.Fatal("expected pending overspend to fail")
	}
	out.Reset()
	if err := app.Run([]string{"mempool", "list"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "pending tx count: 2") {
		t.Fatalf("unexpected mempool list: %s", out.String())
	}
}

func TestEndToEndTransferMempoolMiningAndTxGet(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir).WithProfile(fundedTestProfile())
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	walletA := createTestWallet(t, app, &out)
	walletB := createTestWallet(t, app, &out)
	out.Reset()
	if err := app.Run([]string{"mine", "--address", walletA, "--blocks", "11"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContainsCommand(t, app, &out, []string{"balance", walletA}, "confirmed balance: 550 IDR")
	assertOutputContainsCommand(t, app, &out, []string{"balance", walletA}, "spendable balance: 50 IDR")
	out.Reset()
	if err := app.Run([]string{"send", "--from", walletA, "--to", walletB, "--amount", "10"}); err != nil {
		t.Fatal(err)
	}
	txID := extractTxID(t, out.String())
	assertOutputContains(t, out.String(), "status: pending")
	out.Reset()
	if err := app.Run([]string{"mempool", "list"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "pending tx count: 1")
	assertOutputContains(t, out.String(), "amount=10 IDR")
	out.Reset()
	if err := app.Run([]string{"tx", "get", txID}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "status: pending")
	out.Reset()
	if err := app.Run([]string{"wallet", "inspect", "--address", walletA}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "pending outgoing tx: 1")
	assertOutputContains(t, out.String(), "pending outgoing amount: 10 IDR")
	out.Reset()
	if err := app.Run([]string{"wallet", "inspect", "--address", walletB}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "pending incoming tx: 1")
	assertOutputContains(t, out.String(), "pending incoming amount: 10 IDR")
	out.Reset()
	if err := app.Run([]string{"mine", "--address", walletA, "--blocks", "1"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "txs=2")
	out.Reset()
	if err := app.Run([]string{"mempool", "list"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "pending tx count: 0")
	assertOutputContainsCommand(t, app, &out, []string{"balance", walletA}, "confirmed balance: 590 IDR")
	assertOutputContainsCommand(t, app, &out, []string{"balance", walletB}, "mature balance: 10 IDR")
	out.Reset()
	if err := app.Run([]string{"chain", "info"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "total supply: 600 IDR")
	out.Reset()
	if err := app.Run([]string{"chain", "validate"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "chain valid")
	out.Reset()
	if err := app.Run([]string{"tx", "get", txID}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "status: confirmed")
	assertOutputContains(t, out.String(), "block_height: 12")
}

func TestStakeCommandsAndBalanceOutput(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir).WithProfile(fundedTestProfile())
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	walletA := createTestWallet(t, app, &out)
	assertOutputContainsCommand(t, app, &out, []string{"mine", "--address", walletA, "--blocks", "12"}, "mining complete")
	assertOutputContainsCommand(t, app, &out, []string{"stake", "info"}, "staking enabled: true")
	assertOutputContains(t, out.String(), "min stake amount: 10 IDR")
	assertOutputContains(t, out.String(), "min service stake: 100 IDR")
	assertOutputContains(t, out.String(), "unbonding period: 10")

	out.Reset()
	if err := app.Run([]string{"stake", "lock", "--address", walletA, "--amount", "10"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "stake lock tx created")
	assertOutputContains(t, out.String(), "tx id:")
	assertOutputContains(t, out.String(), "stake id:")
	assertOutputContains(t, out.String(), "status: pending")
	stakeID := extractLineValue(out.String(), "stake id: ")
	assertOutputContainsCommand(t, app, &out, []string{"mine", "--address", walletA, "--blocks", "1"}, "txs=2")
	assertOutputContainsCommand(t, app, &out, []string{"stake", "list", "--address", walletA}, "status: active")
	assertOutputContains(t, out.String(), "stake id: "+stakeID)
	assertOutputContains(t, out.String(), "lock height:")
	assertOutputContainsCommand(t, app, &out, []string{"balance", walletA}, "active stake: 10 IDR")
	assertOutputContains(t, out.String(), "unlocking stake: 0 IDR")
	assertOutputContains(t, out.String(), "released stake: 0 IDR")
	assertOutputContains(t, out.String(), "pending stake lock: 0 IDR")
	assertOutputContains(t, out.String(), "spendable balance:")

	out.Reset()
	if err := app.Run([]string{"stake", "unlock", "--address", walletA, "--stake-id", stakeID}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "stake unlock tx created")
	assertOutputContains(t, out.String(), "release height:")
	assertOutputContains(t, out.String(), "status: pending")
}

func TestSendFailureCasesAndMempoolClear(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir).WithProfile(fundedTestProfile())
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	from := createTestWallet(t, app, &out)
	to := createTestWallet(t, app, &out)
	if err := app.Run([]string{"send", "--from", from, "--to", to, "--amount", "1"}); err == nil || !strings.Contains(err.Error(), "send failed: insufficient mature balance") {
		t.Fatalf("expected insufficient balance, got %v", err)
	}
	if err := app.Run([]string{"send", "--from", from, "--to", "bad", "--amount", "1"}); err == nil || !strings.Contains(err.Error(), "send failed: invalid recipient address") {
		t.Fatalf("expected invalid recipient, got %v", err)
	}
	missing := "idr10000000000000000000000000000000000000000"
	if err := app.Run([]string{"send", "--from", missing, "--to", to, "--amount", "1"}); err == nil || !strings.Contains(err.Error(), "send failed: local wallet not found for sender") {
		t.Fatalf("expected missing sender wallet, got %v", err)
	}
	out.Reset()
	if err := app.Run([]string{"mine", "--address", from, "--blocks", "11"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"send", "--from", from, "--to", to, "--amount", "10"}); err != nil {
		t.Fatal(err)
	}
	txID := extractTxID(t, out.String())
	out.Reset()
	if err := app.Run([]string{"tx", "get", txID}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "status: pending")
	out.Reset()
	if err := app.Run([]string{"mempool", "clear", "--yes"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"mempool", "list"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "pending tx count: 0")
}

func TestAddressValidateCommand(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	addr := createTestWallet(t, app, &out)
	out.Reset()
	if err := app.Run([]string{"address", "validate", addr}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "address valid")
	out.Reset()
	if err := app.Run([]string{"address", "validate", "bad"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "address invalid")
}

func TestLockRejectsWriteAllowsRead(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p2p.CreateLock(config.NewPaths(dir).Lock, ":8331", ":9331"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p2p.RemoveLock(config.NewPaths(dir).Lock) }()
	out.Reset()
	if err := app.Run([]string{"chain", "info"}); err != nil {
		t.Fatalf("read command blocked by lock: %v", err)
	}
	if err := app.Run([]string{"mine", "--address", "idr10000000000000000000000000000000000000000", "--blocks", "1"}); err == nil || !strings.Contains(err.Error(), "datadir is locked by running node") {
		t.Fatalf("expected lock error, got %v", err)
	}
	if err := app.Run([]string{"wallet", "new"}); err == nil || !strings.Contains(err.Error(), "--rpc-url http://127.0.0.1:8331 wallet new") {
		t.Fatalf("expected supported remote wallet new hint, got %v", err)
	}
	out.Reset()
	if err := app.Run([]string{"node", "status"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "locked: true")
}

func TestRemoteCLIMineSendAndMempool(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	var setupOut bytes.Buffer
	local := New(&setupOut).WithDataDir(dir).WithProfile(fundedTestProfile())
	if err := local.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	walletA := createTestWallet(t, local, &setupOut)
	walletB := createTestWallet(t, local, &setupOut)
	mux := http.NewServeMux()
	rpc.RegisterHandlers(mux, paths, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0", Profile: fundedTestProfile()})
	server := httptest.NewServer(mux)
	defer server.Close()
	var out bytes.Buffer
	remote := New(&out).WithProfile(fundedTestProfile())
	if err := remote.Run([]string{"--rpc-url", server.URL, "chain", "info"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "height: 0")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", server.URL, "mine", "--address", walletA, "--blocks", "11"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "mining complete")
	assertOutputContains(t, out.String(), "block broadcast: success=0 failed=0")
	p2pMux := http.NewServeMux()
	p2p.NewServer(paths).Register(p2pMux)
	p2pServer := httptest.NewServer(p2pMux)
	defer p2pServer.Close()
	status, err := remoteGetMap(p2pServer.URL, "/p2p/status")
	if err != nil {
		t.Fatal(err)
	}
	if status["height"].(float64) != 11 {
		t.Fatalf("p2p status height = %v", status["height"])
	}
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", server.URL, "send", "--from", walletA, "--to", walletB, "--amount", "10"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "status: pending")
	assertOutputContains(t, out.String(), "tx broadcast: success=0 failed=0")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", server.URL, "mempool", "list"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "pending tx count: 1")
	assertOutputContains(t, out.String(), "amount=10 IDR")
}

func TestRemoteWalletNewPersistsAndCLIPrintsAddressOnly(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	mux := http.NewServeMux()
	rpc.RegisterHandlers(mux, paths, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0"})
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Post(server.URL+"/wallet/new", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var info map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	addr, _ := info["address"].(string)
	if !strings.HasPrefix(addr, "IDR") {
		t.Fatalf("rpc wallet address = %q", addr)
	}
	if info["format"] != "base58check" || info["network"] != "localnet" || info["key_curve"] != "secp256k1" {
		t.Fatalf("unexpected wallet metadata: %#v", info)
	}
	if err := crypto.ValidateAddress(addr); err != nil {
		t.Fatal(err)
	}
	wallets, err := wallet.NewStore(paths.Wallets).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(wallets) != 1 || wallets[0].Address != addr {
		t.Fatalf("wallet not persisted: %#v", wallets)
	}

	var out bytes.Buffer
	remote := New(&out)
	if err := remote.Run([]string{"--rpc-url", server.URL, "wallet", "new"}); err != nil {
		t.Fatal(err)
	}
	cliAddr := strings.TrimSpace(out.String())
	if !strings.HasPrefix(cliAddr, "IDR") || strings.Contains(out.String(), "private") {
		t.Fatalf("unexpected remote wallet new output: %q", out.String())
	}
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", server.URL, "wallet", "list"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "addresses:")
	assertOutputContains(t, out.String(), addr)
	assertOutputContains(t, out.String(), cliAddr)
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", server.URL, "wallet", "inspect", cliAddr}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"address: " + cliAddr, "format: base58check", "network: localnet", "key curve: secp256k1"} {
		assertOutputContains(t, out.String(), want)
	}
}

func TestRPCChainInfoRuntimeStatsAndBodyErrors(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	var setupOut bytes.Buffer
	local := New(&setupOut).WithDataDir(paths.DataDir).WithProfile(fundedTestProfile())
	if err := local.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	miner := createTestWallet(t, local, &setupOut)
	receiver := createTestWallet(t, local, &setupOut)
	if err := local.Run([]string{"mine", "--address", miner, "--blocks", "11"}); err != nil {
		t.Fatal(err)
	}
	if err := local.Run([]string{"send", "--from", miner, "--to", receiver, "--amount", "10"}); err != nil {
		t.Fatal(err)
	}
	if err := local.Run([]string{"mine", "--address", miner, "--blocks", "1"}); err != nil {
		t.Fatal(err)
	}
	state, err := nodestate.New(paths)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	rpc.RegisterHandlers(mux, paths, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0", State: state, Profile: fundedTestProfile()})
	server := httptest.NewServer(mux)
	defer server.Close()
	info, err := remoteGetMap(server.URL, "/chain/info")
	if err != nil {
		t.Fatal(err)
	}
	if info["total_transactions"].(float64) != 13 || info["coinbase_transactions"].(float64) != 12 || info["normal_transactions"].(float64) != 1 {
		t.Fatalf("unexpected runtime chain info stats: %#v", info)
	}
	if info["network"] != "localnet" || info["chain_id"].(float64) != 777001 || info["protocol_version"].(float64) == 0 || info["rpc_api_version"] == "" {
		t.Fatalf("unexpected protocol metadata: %#v", info)
	}
	resp, err := http.Post(server.URL+"/mine", "application/json", strings.NewReader(`{"address":"`+flipLastBase58(miner)+`","blocks":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("large body status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
	resp, err = http.Post(server.URL+"/send", "application/json", strings.NewReader("{bad"))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid json status = %d", resp.StatusCode)
	}
	large := strings.NewReader(`{"from":"` + strings.Repeat("x", 1024*1024+1) + `"}`)
	resp, err = http.Post(server.URL+"/send", "application/json", large)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("large body status = %d", resp.StatusCode)
	}
}

func TestRemotePeerCommands(t *testing.T) {
	nodeA := config.NewPaths(t.TempDir())
	nodeB := config.NewPaths(t.TempDir())
	var out bytes.Buffer
	appA := New(&out).WithDataDir(nodeA.DataDir)
	if err := appA.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	miner := createTestWallet(t, appA, &out)
	if err := appA.Run([]string{"mine", "--address", miner, "--blocks", "2"}); err != nil {
		t.Fatal(err)
	}
	p2pMux := http.NewServeMux()
	p2p.NewServer(nodeA).Register(p2pMux)
	p2pServer := httptest.NewServer(p2pMux)
	defer p2pServer.Close()
	rpcMux := http.NewServeMux()
	rpc.RegisterHandlers(rpcMux, nodeB, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0", P2PAdvertise: "http://127.0.0.1:9332"})
	rpcServer := httptest.NewServer(rpcMux)
	defer rpcServer.Close()
	remote := New(&out)
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "peer", "check", p2pServer.URL}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "peer ok")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "peer", "connect", p2pServer.URL}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "peer connected")
	assertOutputContains(t, out.String(), "introduced: true")
	introduced, err := p2p.NewPeerStore(nodeA.Peers).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(introduced) != 1 || introduced[0].URL != "http://127.0.0.1:9332" {
		t.Fatalf("remote did not learn introduced peer: %#v", introduced)
	}
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "peer", "add", p2pServer.URL}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "peer added")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "peer", "status"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "checked peers: 1")
	assertOutputContains(t, out.String(), "active: 1")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "peer", "sync"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "imported blocks: 2")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "peer", "sync", "--peer", p2pServer.URL}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "imported blocks: 0")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "peer", "sync", p2pServer.URL}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "imported blocks: 0")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "peer", "list", "--source"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "source=manual")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "peer", "clear", "--yes"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "peers cleared")
	peers, err := p2p.NewPeerStore(nodeB.Peers).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 0 {
		t.Fatalf("remote peer clear left peers: %#v", peers)
	}
}

func TestDevResetClearsPeersAndInspect(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	if err := p2p.NewPeerStore(paths.Peers).AddWithSource("http://127.0.0.1:9332", "manual"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"dev", "reset", "--yes"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "dev data reset complete")
	out.Reset()
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	peers, err := p2p.NewPeerStore(paths.Peers).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 0 {
		t.Fatalf("reset did not clear peers: %#v", peers)
	}
	out.Reset()
	if err := app.Run([]string{"dev", "inspect"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "peers.json: missing")
	assertOutputContains(t, out.String(), "peers: 0")
}

func TestPeerClearKeepsChainAndWallet(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	createTestWallet(t, app, &out)
	if err := p2p.NewPeerStore(paths.Peers).AddWithSource("http://127.0.0.1:9332", "manual"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	err := app.Run([]string{"peer", "clear"})
	if err == nil || !strings.Contains(err.Error(), "refusing to clear peers without --yes") {
		t.Fatalf("expected --yes refusal, got %v", err)
	}
	out.Reset()
	if err := app.Run([]string{"peer", "clear", "--yes"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "peers cleared")
	peers, err := p2p.NewPeerStore(paths.Peers).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 0 {
		t.Fatalf("peer clear left peers: %#v", peers)
	}
	if _, err := os.Stat(paths.DB); err != nil {
		t.Fatalf("chain db removed by peer clear: %v", err)
	}
	out.Reset()
	if err := app.Run([]string{"wallet", "list"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "address=")
}

func TestPeerHealthSeedsAndListJSON(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"--network", "testnet", "init"}); err != nil {
		t.Fatal(err)
	}
	store := p2p.NewPeerStore(config.NewPaths(dir).Peers)
	if err := store.Upsert(p2p.PeerMetadata{URL: "http://127.0.0.1:10311", Source: "seed", Status: p2p.PeerStatusActive, Score: 5, LastHeight: 9}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"peer", "health"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "known peers: 1")
	assertOutputContains(t, out.String(), "active peers: 1")
	assertOutputContains(t, out.String(), "seed peers: 1")
	out.Reset()
	if err := app.Run([]string{"peer", "seeds"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "http://127.0.0.1:10311")
	out.Reset()
	if err := app.Run([]string{"peer", "list", "--json"}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(out.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["peer_count"].(float64) != 1 {
		t.Fatalf("unexpected peer list json: %#v", body)
	}
}

func TestMiningCommandsLocalAndRemote(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"--network", "testnet", "init"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"mining", "status"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "current difficulty:")
	assertOutputContains(t, out.String(), "projected retarget direction:")
	out.Reset()
	if err := app.Run([]string{"mining", "difficulty"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "blocks until retarget:")
	out.Reset()
	if err := app.Run([]string{"mining", "blocks", "--limit", "1"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "recent blocks:")

	paths := config.NewPaths(dir)
	mux := http.NewServeMux()
	rpc.RegisterHandlers(mux, paths, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0", PublicRPC: true})
	server := httptest.NewServer(mux)
	defer server.Close()
	out.Reset()
	remote := New(&out).WithDataDir(t.TempDir())
	if err := remote.Run([]string{"--rpc-url", server.URL, "mining", "status"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "current difficulty:")
}

func TestNodeConfigMaxReorgDepthAndFlagOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node.json")
	if err := os.WriteFile(path, []byte(`{"p2p":{"max_reorg_depth":128}}`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadNodeConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	fs := flag.NewFlagSet("node start", flag.ContinueOnError)
	rpcAddr := ""
	p2pAddr := ""
	advertise := ""
	seedPeers := ""
	seedFile := ""
	upstreamPeers := ""
	upstreamFile := ""
	publicRPC := false
	enableWallet := true
	enableMiner := true
	enableAdmin := true
	enableService := true
	corsOrigins := ""
	rateLimit := 0
	maxPeers := 0
	maxReorgDepth := uint64(64)
	minMiningPeers := 0
	allowIsolatedMining := false
	minWritePeers := 0
	allowIsolatedWrites := false
	applyNodeConfig(fs, cfg, &rpcAddr, &p2pAddr, &advertise, &seedPeers, &seedFile, &upstreamPeers, &upstreamFile, &publicRPC, &enableWallet, &enableMiner, &enableAdmin, &enableService, &corsOrigins, &rateLimit, &maxPeers, &maxReorgDepth, &minMiningPeers, &allowIsolatedMining, &minWritePeers, &allowIsolatedWrites)
	if maxReorgDepth != 128 {
		t.Fatalf("config max reorg depth = %d, want 128", maxReorgDepth)
	}

	fs = flag.NewFlagSet("node start", flag.ContinueOnError)
	_ = fs.Uint64("max-reorg-depth", 64, "")
	if err := fs.Parse([]string{"--max-reorg-depth", "26"}); err != nil {
		t.Fatal(err)
	}
	maxReorgDepth = 26
	applyNodeConfig(fs, cfg, &rpcAddr, &p2pAddr, &advertise, &seedPeers, &seedFile, &upstreamPeers, &upstreamFile, &publicRPC, &enableWallet, &enableMiner, &enableAdmin, &enableService, &corsOrigins, &rateLimit, &maxPeers, &maxReorgDepth, &minMiningPeers, &allowIsolatedMining, &minWritePeers, &allowIsolatedWrites)
	if maxReorgDepth != 26 {
		t.Fatalf("flag max reorg depth should win, got %d", maxReorgDepth)
	}
}

func TestPeerListSourceAndStartupSourceSummary(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := p2p.NewPeerStore(paths.Peers).AddWithSource("http://127.0.0.1:9332", "flag"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"peer", "list", "--source"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "source=flag")
	if got := peerSourceSummary(false, false); got != "" {
		t.Fatalf("empty peer source = %q", got)
	}
	if got := peerSourceSummary(true, false); got != "peers.json" {
		t.Fatalf("file peer source = %q", got)
	}
	if got := peerSourceSummary(false, true); got != "flag" {
		t.Fatalf("flag peer source = %q", got)
	}
	if got := peerSourceSummary(true, true); got != "peers.json + flag" {
		t.Fatalf("merged peer source = %q", got)
	}
}

func TestChainInfoShowsTipAndNextDifficulty(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := app.Run([]string{"chain", "info"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "height: 0")
	assertOutputContains(t, out.String(), "difficulty: 4")
	assertOutputContains(t, out.String(), "tip difficulty: 0")
	assertOutputContains(t, out.String(), "next difficulty: 4")
	assertOutputContains(t, out.String(), "target block time: 10s")
	assertOutputContains(t, out.String(), "retarget window: 10")
	assertOutputContains(t, out.String(), "blocks until retarget: 10")
	out.Reset()
	if err := app.Run([]string{"chain", "difficulty"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"network: localnet", "chain id: 777001", "height: 0", "tip difficulty: 0", "next difficulty: 4", "min difficulty: 1", "max difficulty: 8"} {
		assertOutputContains(t, out.String(), want)
	}
	miner := createTestWallet(t, app, &out)
	out.Reset()
	if err := app.Run([]string{"mine", "--address", miner, "--blocks", "1"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "difficulty=4")
}

func TestRemoteChainDifficulty(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	var setupOut bytes.Buffer
	local := New(&setupOut).WithDataDir(paths.DataDir)
	if err := local.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	rpc.RegisterHandlers(mux, paths, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0"})
	server := httptest.NewServer(mux)
	defer server.Close()
	var out bytes.Buffer
	remote := New(&out)
	if err := remote.Run([]string{"--rpc-url", server.URL, "chain", "difficulty"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"network: localnet", "chain id: 777001", "height: 0", "tip difficulty: 0", "next difficulty: 4", "target block time: 10s"} {
		assertOutputContains(t, out.String(), want)
	}
}

func TestRemoteP2PPingAndDebug(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	var setupOut bytes.Buffer
	local := New(&setupOut).WithDataDir(paths.DataDir)
	if err := local.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	p2pMux := http.NewServeMux()
	p2p.NewServerWithAdvertise(paths, ":9331", "http://127.0.0.1:9331").Register(p2pMux)
	p2pServer := httptest.NewServer(p2pMux)
	defer p2pServer.Close()
	if err := p2p.NewPeerStore(paths.Peers).Add(p2pServer.URL); err != nil {
		t.Fatal(err)
	}
	state, err := nodestate.New(paths)
	if err != nil {
		t.Fatal(err)
	}
	rpcMux := http.NewServeMux()
	rpc.RegisterHandlers(rpcMux, paths, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0", State: state})
	rpcServer := httptest.NewServer(rpcMux)
	defer rpcServer.Close()
	var out bytes.Buffer
	remote := New(&out)
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "p2p", "ping", p2pServer.URL}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "result: ok")
	assertOutputContains(t, out.String(), "status: ok")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "p2p", "debug"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "node id:")
	assertOutputContains(t, out.String(), "peers: 1")
}

func TestRemoteChainLocatorAndForkCheck(t *testing.T) {
	nodeA := config.NewPaths(t.TempDir())
	nodeB := config.NewPaths(t.TempDir())
	var setupOut bytes.Buffer
	appA := New(&setupOut).WithDataDir(nodeA.DataDir)
	if err := appA.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	minerA := createTestWallet(t, appA, &setupOut)
	if err := appA.Run([]string{"mine", "--address", minerA, "--blocks", "1"}); err != nil {
		t.Fatal(err)
	}
	appB := New(&setupOut).WithDataDir(nodeB.DataDir)
	if err := appB.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	minerB := createTestWallet(t, appB, &setupOut)
	if err := appB.Run([]string{"mine", "--address", minerB, "--blocks", "1"}); err != nil {
		t.Fatal(err)
	}
	p2pMuxA := http.NewServeMux()
	p2p.NewServer(nodeA).Register(p2pMuxA)
	p2pServerA := httptest.NewServer(p2pMuxA)
	defer p2pServerA.Close()
	p2pMuxB := http.NewServeMux()
	p2p.NewServer(nodeB).Register(p2pMuxB)
	p2pServerB := httptest.NewServer(p2pMuxB)
	defer p2pServerB.Close()
	rpcMux := http.NewServeMux()
	rpc.RegisterHandlers(rpcMux, nodeA, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0"})
	rpcServer := httptest.NewServer(rpcMux)
	defer rpcServer.Close()
	var out bytes.Buffer
	remote := New(&out)
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "chain", "locator"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "locator:")
	assertOutputContains(t, out.String(), "height=1")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "fork", "check", "--peer", p2pServerA.URL}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "no fork detected")
	assertOutputContains(t, out.String(), "nodes in sync")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "fork", "check", "--peer", p2pServerB.URL}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "fork detected")
	assertOutputContains(t, out.String(), "common ancestor height: 0")
	assertOutputContains(t, out.String(), "reorg supported: false")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "chain", "common-ancestor", "--peer", p2pServerA.URL}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "common ancestor found")
	assertOutputContains(t, out.String(), "height: 1")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "chain", "common-ancestor", "--peer", p2pServerA.URL, "--debug"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "local locator count:")
	assertOutputContains(t, out.String(), "peer: "+p2pServerA.URL)
	assertOutputContains(t, out.String(), "common ancestor found")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "chain", "common-ancestor", "--peer", p2pServerB.URL}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "common ancestor found")
	assertOutputContains(t, out.String(), "height: 0")
	info, err := remotePostMap(rpcServer.URL, "/chain/common-ancestor", map[string]string{"peer": p2pServerA.URL})
	if err != nil {
		t.Fatal(err)
	}
	if found, _ := info["found"].(bool); !found || info["height"].(float64) != 1 {
		t.Fatalf("expected RPC common ancestor at tip: %#v", info)
	}
	info, err = remotePostMap(rpcServer.URL, "/fork/check", map[string]string{"peer": p2pServerB.URL})
	if err != nil {
		t.Fatal(err)
	}
	if fork, _ := info["fork_detected"].(bool); !fork {
		t.Fatalf("expected RPC fork_detected=true: %#v", info)
	}
	info, err = remotePostMap(rpcServer.URL, "/fork/check", map[string]string{"peer": p2pServerA.URL})
	if err != nil {
		t.Fatal(err)
	}
	if inSync, _ := info["in_sync"].(bool); !inSync {
		t.Fatalf("expected RPC in_sync=true: %#v", info)
	}
}

func TestDevForkSimInspectAndRemoteSyncRejection(t *testing.T) {
	dirA := filepath.Join(t.TempDir(), "forkA")
	dirB := filepath.Join(t.TempDir(), "forkB")
	var out bytes.Buffer
	app := New(&out)
	if err := app.Run([]string{"dev", "fork-sim", "--datadir-a", dirA, "--datadir-b", dirB, "--blocks-a", "3", "--blocks-b", "3"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "fork simulation complete")
	assertOutputContains(t, out.String(), "common ancestor expected: height 0")
	pathsA := config.NewPaths(dirA)
	pathsB := config.NewPaths(dirB)
	locA, err := p2p.LocalLocator(pathsA)
	if err != nil {
		t.Fatal(err)
	}
	locB, err := p2p.LocalLocator(pathsB)
	if err != nil {
		t.Fatal(err)
	}
	if locA.Height != 3 || locB.Height != 3 || locA.TipHash == locB.TipHash {
		t.Fatalf("expected forked equal-height chains: A=%#v B=%#v", locA, locB)
	}
	if locA.Locator[len(locA.Locator)-1].Hash != locB.Locator[len(locB.Locator)-1].Hash {
		t.Fatal("genesis hashes differ")
	}
	if locA.Locator[2].Hash == locB.Locator[2].Hash {
		t.Fatal("height 1 hashes should differ")
	}
	out.Reset()
	if err := New(&out).WithDataDir(dirA).Run([]string{"fork", "inspect", "--other-datadir", dirB}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "fork detected")
	assertOutputContains(t, out.String(), "common ancestor height: 0")
	assertOutputContains(t, out.String(), "reorg supported: false")
	rpcMux := http.NewServeMux()
	rpc.RegisterHandlers(rpcMux, pathsA, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0"})
	rpcServer := httptest.NewServer(rpcMux)
	defer rpcServer.Close()
	info, err := remotePostMap(rpcServer.URL, "/fork/inspect-datadir", map[string]string{"other_datadir": dirB})
	if err != nil {
		t.Fatal(err)
	}
	if fork, _ := info["fork_detected"].(bool); !fork || numberFromAny(info["common_ancestor_height"]) != 0 {
		t.Fatalf("unexpected inspect-datadir response: %#v", info)
	}
	p2pMuxB := http.NewServeMux()
	p2p.NewServer(pathsB).Register(p2pMuxB)
	p2pServerB := httptest.NewServer(p2pMuxB)
	defer p2pServerB.Close()
	if err := p2p.NewPeerStore(pathsA.Peers).AddWithSource(p2pServerB.URL, "manual"); err != nil {
		t.Fatal(err)
	}
	before := locA
	info, err = remotePostMap(rpcServer.URL, "/peers/sync", map[string]any{"include_bad": true})
	if err == nil {
		t.Fatal("expected fork sync to fail")
	}
	if info["error"] != "fork detected" || numberFromAny(info["common_ancestor_height"]) != 0 || info["reorg_supported"].(bool) {
		t.Fatalf("unexpected peer sync fork response: %#v", info)
	}
	after, err := p2p.LocalLocator(pathsA)
	if err != nil {
		t.Fatal(err)
	}
	if after.Height != before.Height || after.TipHash != before.TipHash {
		t.Fatalf("failed fork sync changed local tip: before=%#v after=%#v", before, after)
	}
	out.Reset()
	remote := New(&out)
	err = remote.Run([]string{"--rpc-url", rpcServer.URL, "peer", "sync", "--include-bad"})
	if err == nil {
		t.Fatal("expected remote peer sync to return fork error")
	}
	assertOutputContains(t, out.String(), "sync failed: fork detected")
	assertOutputContains(t, out.String(), "common ancestor height: 0")
	assertOutputContains(t, out.String(), "decision: fork_tie_same_work")
}

func TestRemoteMineStatusAlreadyRunningAndDebugLocks(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	var setupOut bytes.Buffer
	local := New(&setupOut).WithDataDir(paths.DataDir)
	if err := local.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	miner := createTestWallet(t, local, &setupOut)
	state, err := nodestate.New(paths)
	if err != nil {
		t.Fatal(err)
	}
	miningService := mining.NewService()
	if _, err := miningService.Start(miner, 2); err != nil {
		t.Fatal(err)
	}
	rpcMux := http.NewServeMux()
	rpc.RegisterHandlers(rpcMux, paths, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0", State: state, Mining: miningService})
	rpcServer := httptest.NewServer(rpcMux)
	defer rpcServer.Close()
	var out bytes.Buffer
	remote := New(&out)
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "mine", "status"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "mining status: running")
	assertOutputContains(t, out.String(), "mined blocks: 0/2")
	out.Reset()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "debug", "locks"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "mining running: true")
	out.Reset()
	err = remote.Run([]string{"--rpc-url", rpcServer.URL, "mine", "--address", miner, "--blocks", "1", "--timeout", "5s"})
	if err == nil || !strings.Contains(err.Error(), "mining already running") {
		t.Fatalf("expected already running error, got %v", err)
	}
}

func TestRemoteMineCompletesWithSlowPeerBroadcastFailure(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	var setupOut bytes.Buffer
	local := New(&setupOut).WithDataDir(paths.DataDir)
	if err := local.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	miner := createTestWallet(t, local, &setupOut)
	slowMux := http.NewServeMux()
	slowMux.HandleFunc("POST /p2p/block", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(3500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	})
	slowPeer := httptest.NewServer(slowMux)
	defer slowPeer.Close()
	if err := p2p.NewPeerStore(paths.Peers).Upsert(p2p.PeerMetadata{URL: slowPeer.URL, Status: p2p.PeerStatusActive}); err != nil {
		t.Fatal(err)
	}
	state, err := nodestate.New(paths)
	if err != nil {
		t.Fatal(err)
	}
	rpcMux := http.NewServeMux()
	rpc.RegisterHandlers(rpcMux, paths, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0", State: state, Mining: mining.NewService()})
	rpcServer := httptest.NewServer(rpcMux)
	defer rpcServer.Close()
	var out bytes.Buffer
	remote := New(&out)
	start := time.Now()
	if err := remote.Run([]string{"--rpc-url", rpcServer.URL, "mine", "--address", miner, "--blocks", "1", "--timeout", "10s"}); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 8*time.Second {
		t.Fatal("remote mine took too long with slow peer")
	}
	assertOutputContains(t, out.String(), "mining complete")
	assertOutputContains(t, out.String(), "block broadcast: success=0 failed=1")
	info, err := remoteGetMap(rpcServer.URL, "/chain/info")
	if err != nil {
		t.Fatal(err)
	}
	if info["height"].(float64) != 1 {
		t.Fatalf("height = %v, want 1", info["height"])
	}
}

func TestFaucetCLIInfo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /faucet/info", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enabled":true,"network":"testnet","network_id":"idr-testnet-1","chain_id":777101,"faucet_address":"IDRFAUCET","amount":"100","max_per_address":"1000","min_interval_seconds":60,"mempool_pending":0,"note":"testnet faucet only; testnet IDR has no monetary value"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	var out bytes.Buffer
	app := New(&out)
	if err := app.Run([]string{"--rpc-url", server.URL, "faucet", "info"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "enabled: true")
	assertOutputContains(t, out.String(), "network: testnet")
	assertOutputContains(t, out.String(), "faucet address: IDRFAUCET")
	assertOutputContains(t, out.String(), "amount: 100 IDR")
}

func TestFaucetCLIRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /faucet/request", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["address"] != "IDRRECIPIENT" {
			t.Fatalf("unexpected request: %#v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tx_id":"abc123","from":"IDRFAUCET","to":"IDRRECIPIENT","amount":"100","status":"pending","note":"mine a block to confirm faucet transaction"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	var out bytes.Buffer
	app := New(&out)
	if err := app.Run([]string{"--rpc-url", server.URL, "faucet", "request", "--address", "IDRRECIPIENT"}); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), "faucet tx created")
	assertOutputContains(t, out.String(), "tx id: abc123")
	assertOutputContains(t, out.String(), "amount: 100 IDR")
	assertOutputContains(t, out.String(), "status: pending")
}

func TestCLI_FaucetStakeServiceE2E(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	var setupOut bytes.Buffer
	local := New(&setupOut).WithDataDir(paths.DataDir)
	if err := local.Run([]string{"--network", "testnet", "init"}); err != nil {
		t.Fatal(err)
	}
	faucetWallet, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	ownerWallet, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	walletStore := wallet.NewStore(paths.Wallets)
	if err := walletStore.Add(faucetWallet); err != nil {
		t.Fatal(err)
	}
	if err := walletStore.Add(ownerWallet); err != nil {
		t.Fatal(err)
	}
	cliFundFaucetFixture(t, paths, faucetWallet.Address, 125)

	mux := http.NewServeMux()
	rpc.RegisterHandlers(mux, paths, rpc.NodeInfo{
		RPCListen:           ":0",
		P2PListen:           ":0",
		Profile:             config.Testnet(),
		EnableFaucetRPC:     true,
		EnableFaucetRPCSet:  true,
		FaucetAddress:       faucetWallet.Address,
		FaucetAmount:        1000 * config.UnitsPerCoin,
		FaucetMaxPerAddress: 2000 * config.UnitsPerCoin,
		FaucetMinInterval:   time.Second,
		AllowIsolatedWrites: true,
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var out bytes.Buffer
	remote := New(&out)
	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "faucet", "info"}, "amount: 1000 IDR")
	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "faucet", "request", "--address", ownerWallet.Address}, "faucet tx created")
	assertOutputContains(t, out.String(), "amount: 1000 IDR")
	cliMinePendingFixture(t, paths, faucetWallet.Address)
	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "balance", ownerWallet.Address}, "confirmed balance: 1000")
	assertOutputContains(t, out.String(), "spendable balance: 1000")

	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "stake", "lock", "--address", ownerWallet.Address, "--amount", "1000"}, "stake lock tx created")
	assertOutputContains(t, out.String(), "amount: 1000 IDR")
	cliMinePendingFixture(t, paths, faucetWallet.Address)
	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "stake", "list", "--address", ownerWallet.Address}, "status: active")
	assertOutputContains(t, out.String(), "amount: 1000 IDR")
	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "balance", ownerWallet.Address}, "active stake: 1000")
	assertOutputContains(t, out.String(), "spendable balance: 0")

	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "service", "register", "--address", ownerWallet.Address, "--endpoint", "http://127.0.0.1:9971"}, "service node registered")
	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "service", "heartbeat", "--address", ownerWallet.Address, "--endpoint", "http://127.0.0.1:9971"}, "service score:")
	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "service", "challenge", "create", "--address", ownerWallet.Address}, "service challenge created")
	challengeID := extractLineValue(out.String(), "challenge id: ")
	if challengeID == "" {
		t.Fatalf("challenge id not found:\n%s", out.String())
	}
	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "service", "challenge", "submit", "--challenge-id", challengeID, "--latency-ms", "50", "--bytes-up", "100000000", "--bytes-down", "100000000", "--success", "true"}, "service challenge submitted")
	assertOutputContainsCommand(t, remote, &out, []string{"--rpc-url", server.URL, "service", "score", "--address", ownerWallet.Address}, "required stake: 1000 IDR")
	assertOutputContains(t, out.String(), "active stake: 1000 IDR")
	assertOutputContains(t, out.String(), "stake eligible: true")
	assertOutputContains(t, out.String(), "collateral status: eligible")
	assertOutputContains(t, out.String(), "eligible simulated points:")
}

func TestPeerSyncTrackerSkipThrottleAndRecovery(t *testing.T) {
	tracker := newPeerSyncTracker()
	peer := "http://127.0.0.1:9332"
	if !tracker.tryStart(peer) {
		t.Fatal("first sync should start")
	}
	if tracker.tryStart(peer) {
		t.Fatal("second sync should be skipped while running")
	}
	tracker.done(peer)
	if !tracker.tryStart(peer) {
		t.Fatal("sync should start after done")
	}
	tracker.done(peer)
	now := time.Unix(100, 0)
	if !tracker.shouldLogError(peer, "timeout", now) {
		t.Fatal("first error should log")
	}
	if tracker.shouldLogError(peer, "timeout", now.Add(30*time.Second)) {
		t.Fatal("repeated error should be throttled")
	}
	if !tracker.shouldLogError(peer, "timeout", now.Add(61*time.Second)) {
		t.Fatal("error after throttle window should log")
	}
	if !tracker.markRecovered(peer) {
		t.Fatal("recovery should be reported after failure")
	}
	if tracker.markRecovered(peer) {
		t.Fatal("second recovery should not be reported")
	}
}

func TestRemoteMainnetCLIIsBlocked(t *testing.T) {
	var out bytes.Buffer
	app := New(&out)
	app = app.WithProfile(config.Mainnet())
	err := app.Run([]string{"--network", "mainnet", "--rpc-url", "http://127.0.0.1:8332", "chain", "info"})
	if err == nil {
		t.Fatal("expected remote mainnet mode to be blocked")
	}
	assertOutputContains(t, err.Error(), "mainnet is not operational")
	assertOutputContains(t, err.Error(), "remote RPC mode is blocked")
}

func TestLockedPeerSyncSuggestsRemoteCommand(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	var out bytes.Buffer
	app := New(&out).WithDataDir(dir)
	if err := app.Run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p2p.CreateLock(paths.Lock, ":8332", ":9332"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p2p.RemoveLock(paths.Lock) }()
	err := app.Run([]string{"peer", "sync"})
	if err == nil {
		t.Fatal("expected lock error")
	}
	msg := err.Error()
	assertOutputContains(t, msg, "datadir is locked by running node")
	assertOutputContains(t, msg, "go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync")
}

func TestAmountFormattingCases(t *testing.T) {
	cases := map[uint64]string{
		config.UnitsPerCoin:       "1",
		150 * config.UnitsPerCoin: "150",
		1:                         "0.00000001",
	}
	for input, want := range cases {
		if got := amount.Format(input); got != want {
			t.Fatalf("Format(%d) = %q, want %q", input, got, want)
		}
	}
}

func createTestWallet(t *testing.T, app App, out *bytes.Buffer) string {
	t.Helper()
	out.Reset()
	if err := app.Run([]string{"wallet", "new"}); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(out.String())
}

func extractTxID(t *testing.T, output string) string {
	t.Helper()
	match := regexp.MustCompile(`(?m)^id: ([0-9a-f]+)$`).FindStringSubmatch(output)
	if len(match) != 2 {
		t.Fatalf("tx id not found in output:\n%s", output)
	}
	return match[1]
}

func flipLastBase58(addr string) string {
	if strings.HasSuffix(addr, "1") {
		return addr[:len(addr)-1] + "2"
	}
	return addr[:len(addr)-1] + "1"
}

func assertOutput(t *testing.T, app App, out *bytes.Buffer, args []string, want string) {
	t.Helper()
	out.Reset()
	if err := app.Run(args); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != want {
		t.Fatalf("%v output = %q, want %q", args, got, want)
	}
}

func assertOutputContainsCommand(t *testing.T, app App, out *bytes.Buffer, args []string, want string) {
	t.Helper()
	out.Reset()
	if err := app.Run(args); err != nil {
		t.Fatal(err)
	}
	assertOutputContains(t, out.String(), want)
}

func assertOutputContains(t *testing.T, output, want string) {
	t.Helper()
	if !strings.Contains(output, want) {
		t.Fatalf("output missing %q:\n%s", want, output)
	}
}

func mustTestWallet(t *testing.T) wallet.Wallet {
	t.Helper()
	w, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func extractLineValue(output, prefix string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func cliFundFaucetFixture(t *testing.T, paths config.Paths, address string, blocksCount int) {
	t.Helper()
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	blocks, err := chain.New(store).Blocks()
	if err != nil {
		t.Fatal(err)
	}
	prior := append([]types.Block(nil), blocks...)
	for i := 0; i < blocksCount; i++ {
		height := prior[len(prior)-1].Height + 1
		cb := types.NewCoinbaseTransaction(address, config.InitialBlockReward, height)
		difficulty := chain.CalculateNextDifficultyWithParams(prior, config.Testnet().Difficulty)
		block := types.NewBlock(height, prior[len(prior)-1].Hash, address, difficulty, []types.Transaction{cb})
		block.Timestamp = prior[len(prior)-1].Timestamp + config.Testnet().Difficulty.TargetBlockTimeSeconds
		block.Hash = block.CalculateHash()
		if err := store.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		prior = append(prior, block)
	}
	snapshot, err := state.SnapshotForBlocks(prior, config.Testnet().Consensus, config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}
}

func cliMinePendingFixture(t *testing.T, paths config.Paths, miner string) {
	t.Helper()
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	blocks, err := chain.New(store).Blocks()
	if err != nil {
		t.Fatal(err)
	}
	pending, err := mempool.New(paths.Mempool).Load()
	if err != nil {
		t.Fatal(err)
	}
	height := blocks[len(blocks)-1].Height + 1
	txs := append([]types.Transaction{types.NewCoinbaseTransaction(miner, config.InitialBlockReward, height)}, pending...)
	block := types.NewBlock(height, blocks[len(blocks)-1].Hash, miner, chain.CalculateNextDifficultyWithParams(blocks, config.Testnet().Difficulty), txs)
	block.Timestamp = blocks[len(blocks)-1].Timestamp + config.Testnet().Difficulty.TargetBlockTimeSeconds
	block.Hash = block.CalculateHash()
	if err := store.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	prior := append([]types.Block(nil), blocks...)
	prior = append(prior, block)
	snapshot, err := state.SnapshotForBlocks(prior, config.Testnet().Consensus, config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}
	ids := map[string]struct{}{}
	for _, tx := range pending {
		ids[tx.ID] = struct{}{}
	}
	if err := mempool.New(paths.Mempool).RemoveIDs(ids); err != nil {
		t.Fatal(err)
	}
}

func initCLIProfileNode(t *testing.T, profile config.NetworkConfig) config.Paths {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.New(store).InitWithProfile(profile); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	_ = store.Close()
	if err := config.WriteNetworkMetadata(paths, profile, chain.GenesisBlockForNetwork(profile).Hash); err != nil {
		t.Fatal(err)
	}
	return paths
}
