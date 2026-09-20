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
		{"--network", "mainnet", "service", "list"},
		{"--network", "mainnet", "service", "register", "--address", "a", "--endpoint", "https://example.invalid"},
		{"--network", "mainnet", "stake", "info"},
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