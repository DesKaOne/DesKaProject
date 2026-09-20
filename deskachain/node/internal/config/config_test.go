package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNetworkProfileRejectsNameMismatch(t *testing.T) {
	profile := Localnet()
	profile.NetworkName = "testnet"
	if err := ValidateNetworkProfile(profile); err == nil || !strings.Contains(err.Error(), "network name mismatch") {
		t.Fatalf("expected network name mismatch rejection, got %v", err)
	}
}

func TestNetworkByName(t *testing.T) {
	net, err := NetworkByName("localnet")
	if err != nil {
		t.Fatal(err)
	}
	if net.NetworkID != "idr-local-1" || net.ChainID != 777001 {
		t.Fatalf("unexpected localnet: %#v", net)
	}
	if _, err := NetworkByName("badnet"); err == nil {
		t.Fatal("expected unknown network error")
	}
}

func TestLocalnetStakingParams(t *testing.T) {
	params := Localnet().Consensus.Staking
	if !params.Enabled {
		t.Fatal("localnet staking should be enabled")
	}
	if params.MinStakeAmount != 10*UnitsPerCoin {
		t.Fatalf("min stake = %d", params.MinStakeAmount)
	}
	if params.MinServiceStake != 100*UnitsPerCoin {
		t.Fatalf("min service stake = %d", params.MinServiceStake)
	}
	if params.UnbondingPeriodBlocks != 10 {
		t.Fatalf("unbonding = %d", params.UnbondingPeriodBlocks)
	}
	if params.MaxActiveStakesPerAddress != 10 {
		t.Fatalf("max active stakes = %d", params.MaxActiveStakesPerAddress)
	}
}

func TestNetworkProfilesDifferent(t *testing.T) {
	local := Localnet()
	testnet := Testnet()
	if local.Name == testnet.Name {
		t.Fatalf("network names should differ: %s", local.Name)
	}
	if local.NetworkID == testnet.NetworkID {
		t.Fatalf("network ids should differ: %s", local.NetworkID)
	}
	if local.ChainID == testnet.ChainID {
		t.Fatalf("chain ids should differ: %d", local.ChainID)
	}
	if local.Consensus.Staking.MinServiceStake == testnet.Consensus.Staking.MinServiceStake {
		t.Fatalf("staking service stakes should differ")
	}
}

func TestMaxReorgDepthDefaultsAndEnv(t *testing.T) {
	if got := DefaultMaxReorgDepth(Localnet()); got != 64 {
		t.Fatalf("localnet max reorg depth = %d, want 64", got)
	}
	if got := DefaultMaxReorgDepth(Testnet()); got != 128 {
		t.Fatalf("testnet max reorg depth = %d, want 128", got)
	}
	t.Setenv("IDR_MAX_REORG_DEPTH", "26")
	got, err := MaxReorgDepthFromEnv(Testnet())
	if err != nil {
		t.Fatal(err)
	}
	if got != 26 {
		t.Fatalf("env max reorg depth = %d, want 26", got)
	}
	t.Setenv("IDR_MAX_REORG_DEPTH", "bad")
	if _, err := MaxReorgDepthFromEnv(Testnet()); err == nil {
		t.Fatal("expected invalid env max reorg depth")
	}
}

func TestIsolatedMiningAndWriteDefaultsAndEnv(t *testing.T) {
	local := Localnet()
	if local.MinMiningPeers != 0 || !local.AllowIsolatedMining || local.MinWritePeers != 0 || !local.AllowIsolatedWrites {
		t.Fatalf("unexpected localnet isolation defaults: %#v", local)
	}
	testnet := Testnet()
	if testnet.MinMiningPeers != 1 || testnet.AllowIsolatedMining || testnet.MinWritePeers != 1 || testnet.AllowIsolatedWrites {
		t.Fatalf("unexpected testnet isolation defaults: %#v", testnet)
	}
	t.Setenv("IDR_MIN_MINING_PEERS", "2")
	t.Setenv("IDR_ALLOW_ISOLATED_MINING", "true")
	t.Setenv("IDR_MIN_WRITE_PEERS", "3")
	t.Setenv("IDR_ALLOW_ISOLATED_WRITES", "true")
	if got, err := MinMiningPeersFromEnv(testnet); err != nil || got != 2 {
		t.Fatalf("min mining peers env = %d, %v", got, err)
	}
	if got, err := AllowIsolatedMiningFromEnv(testnet); err != nil || !got {
		t.Fatalf("allow isolated mining env = %t, %v", got, err)
	}
	if got, err := MinWritePeersFromEnv(testnet); err != nil || got != 3 {
		t.Fatalf("min write peers env = %d, %v", got, err)
	}
	if got, err := AllowIsolatedWritesFromEnv(testnet); err != nil || !got {
		t.Fatalf("allow isolated writes env = %t, %v", got, err)
	}
	t.Setenv("IDR_ALLOW_ISOLATED_MINING", "maybe")
	if _, err := AllowIsolatedMiningFromEnv(testnet); err == nil {
		t.Fatal("expected invalid isolated mining env")
	}
}

func TestNetworkProfilesHaveSeedPeerField(t *testing.T) {
	for _, profile := range []NetworkConfig{Localnet(), Testnet(), Mainnet()} {
		if profile.SeedPeers == nil {
			continue
		}
		for _, seed := range profile.SeedPeers {
			if strings.TrimSpace(seed) == "" {
				t.Fatalf("%s has empty seed peer: %#v", profile.Name, profile.SeedPeers)
			}
		}
	}
}

func TestLocalnetHasNoDefaultPublicSeeds(t *testing.T) {
	if len(Localnet().SeedPeers) != 0 {
		t.Fatalf("localnet should not have default public seeds: %#v", Localnet().SeedPeers)
	}
}

func TestTestnetSeedPeersDoNotChangeGenesisOrNetworkID(t *testing.T) {
	testnet := Testnet()
	if testnet.NetworkID != "idr-testnet-1" || testnet.ChainID != 777101 || testnet.GenesisHash != "" {
		t.Fatalf("unexpected testnet identity after seed support: %#v", testnet)
	}
	withoutSeeds := Testnet()
	withoutSeeds.SeedPeers = nil
	if testnet.NetworkID != withoutSeeds.NetworkID || testnet.ChainID != withoutSeeds.ChainID || testnet.AddressVersion != withoutSeeds.AddressVersion {
		t.Fatalf("seed peers changed network identity: with=%#v without=%#v", testnet, withoutSeeds)
	}
}

func TestTestnetGenesisCandidateProfileStable(t *testing.T) {
	testnet := Testnet()
	if testnet.Name != "testnet" {
		t.Fatalf("network name changed: %s", testnet.Name)
	}
	if testnet.NetworkID != "idr-testnet-1" {
		t.Fatalf("network id changed: %s", testnet.NetworkID)
	}
	if testnet.ChainID != 777101 {
		t.Fatalf("chain id changed: %d", testnet.ChainID)
	}
	if testnet.Difficulty.TargetBlockTimeSeconds != 30 {
		t.Fatalf("target block time changed: %d", testnet.Difficulty.TargetBlockTimeSeconds)
	}
	if testnet.Difficulty.RetargetWindow != 30 {
		t.Fatalf("retarget window changed: %d", testnet.Difficulty.RetargetWindow)
	}
	if testnet.Difficulty.MinDifficulty != 1 || testnet.Difficulty.MaxDifficulty != 12 {
		t.Fatalf("difficulty bounds changed: %#v", testnet.Difficulty)
	}
	if testnet.Consensus.CoinbaseMaturity != 100 {
		t.Fatalf("coinbase maturity changed: %d", testnet.Consensus.CoinbaseMaturity)
	}
	if testnet.Consensus.Staking.MinStakeAmount != 100*UnitsPerCoin {
		t.Fatalf("min stake amount changed: %d", testnet.Consensus.Staking.MinStakeAmount)
	}
	if testnet.Consensus.Staking.MinServiceStake != 1000*UnitsPerCoin {
		t.Fatalf("min service stake changed: %d", testnet.Consensus.Staking.MinServiceStake)
	}
	if testnet.Consensus.Staking.UnbondingPeriodBlocks != 100 {
		t.Fatalf("unbonding period changed: %d", testnet.Consensus.Staking.UnbondingPeriodBlocks)
	}
}

func TestPathsStayUnderDataDir(t *testing.T) {
	dir := filepath.Join("tmp", "node1")
	paths := NewPaths(dir)
	for name, path := range map[string]string{
		"chain db": paths.ChainDBPath(),
		"wallets":  paths.WalletPath(),
		"mempool":  paths.MempoolPath(),
		"peers":    paths.PeersPath(),
		"lock":     paths.LockPath(),
		"node id":  paths.NodeIDPath(),
	} {
		rel, err := filepath.Rel(paths.DataDirPath(), path)
		if err != nil {
			t.Fatalf("%s relative path: %v", name, err)
		}
		if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			t.Fatalf("%s path escaped datadir: %s", name, path)
		}
	}
}


func TestMainnetProfileRequiresExplicitNetworkSelection(t *testing.T) {
	if local := Localnet(); local.Name == "mainnet" {
		t.Fatal("localnet profile must remain separate from mainnet")
	}
	if testnet := Testnet(); testnet.Name == "mainnet" {
		t.Fatal("testnet profile must remain separate from mainnet")
	}
}

func TestNetworkByNameMainnetIsExplicit(t *testing.T) {
	mainnet, err := NetworkByName("mainnet")
	if err != nil {
		t.Fatal(err)
	}
	if mainnet.Name != "mainnet" || mainnet.ChainID != 777000 {
		t.Fatalf("unexpected mainnet selection: %#v", mainnet)
	}
	for _, name := range []string{"", "localnet", "testnet"} {
		profile, err := NetworkByName(name)
		if err != nil {
			t.Fatal(err)
		}
		if profile.Name == "mainnet" {
			t.Fatalf("non-mainnet selector %q returned mainnet", name)
		}
	}
}

func TestMainnetDoesNotAdvertiseAsAvailable(t *testing.T) {
	profile := Mainnet()
	if profile.Name != "mainnet" {
		t.Fatalf("unexpected mainnet profile: %q", profile.Name)
	}
	if profile.GenesisHash == "" || profile.GenesisHash != MainnetGenesisHash {
		t.Fatalf("mainnet genesis identity is not frozen: %q", profile.GenesisHash)
	}
}

func TestMainnetLaunchCriticalProfile(t *testing.T) {
	profile := Mainnet()
	checks := map[string]bool{
		"network_id": profile.NetworkID == "idr-main-1",
		"chain_id": profile.ChainID == 777000,
		"legacy_addresses_disabled": !profile.LegacyAddressAllowed,
		"authenticated_p2p": profile.RequireAuthenticatedNode,
		"isolated_mining_disabled": !profile.AllowIsolatedMining,
		"isolated_writes_disabled": !profile.AllowIsolatedWrites,
		"token_transfers": profile.Asset.TokenTransfersEnabled,
		"issued_tokens": profile.Asset.UserIssuedTokensEnabled,
		"paymaster": profile.Asset.PaymasterEnabled,
		"fee_enabled": profile.Fee.Enabled,
		"zero_subsidy": profile.Economic.BlockSubsidy == 0,
		"fee_only": profile.Economic.FeeOnlyBlocks,
	}
	for name, ok := range checks {
		if !ok {
			t.Fatalf("mainnet launch profile invariant failed: %s", name)
		}
	}
	if profile.GenesisHash != MainnetGenesisHash {
		t.Fatalf("mainnet genesis hash = %q, want %q", profile.GenesisHash, MainnetGenesisHash)
	}
}

func TestBuiltInNetworkProfilesSatisfyV3Freeze(t *testing.T) {
	for _, profile := range []NetworkConfig{Localnet(), Testnet(), Mainnet()} {
		if err := ValidateNetworkProfile(profile); err != nil {
			t.Fatalf("%s profile failed validation: %v", profile.Name, err)
		}
		if got, err := NetworkByName(profile.Name); err != nil {
			t.Fatalf("%s profile lookup failed: %v", profile.Name, err)
		} else if got.NetworkID != profile.NetworkID || got.ChainID != profile.ChainID {
			t.Fatalf("%s profile identity changed during lookup", profile.Name)
		}
	}
}

func TestV3NetworkProfileRejectsEconomicDrift(t *testing.T) {
	profile := Localnet()
	profile.Economic.BlockSubsidy = 1
	if err := ValidateNetworkProfile(profile); err == nil {
		t.Fatal("expected non-zero block subsidy to be rejected")
	}

	profile = Localnet()
	profile.Asset.FeeAssetID = "USDT"
	if err := ValidateNetworkProfile(profile); err == nil {
		t.Fatal("expected non-IDR fee asset to be rejected")
	}

	profile = Localnet()
	profile.TxVersion = 4
	if err := ValidateNetworkProfile(profile); err == nil {
		t.Fatal("expected unsupported tx version to be rejected")
	}
}

func TestMainnetGenesisHashIsFrozen(t *testing.T) {
	profile := Mainnet()
	if profile.GenesisHash != MainnetGenesisHash {
		t.Fatalf("mainnet genesis hash = %q, want %q", profile.GenesisHash, MainnetGenesisHash)
	}
	if MainnetGenesisHash == "" {
		t.Fatal("mainnet genesis hash must not be empty")
	}
	profile.GenesisHash = "deadbeef"
	if err := ValidateNetworkProfile(profile); err == nil {
		t.Fatal("expected modified mainnet genesis hash to be rejected")
	}
}

func TestProductionProfilesRequireAuthenticatedP2P(t *testing.T) {
	if Localnet().RequireAuthenticatedNode {
		t.Fatal("localnet should remain permissive for standalone development")
	}
	for _, profile := range []NetworkConfig{Testnet(), Mainnet()} {
		if !profile.RequireAuthenticatedNode {
			t.Fatalf("%s must require authenticated P2P", profile.Name)
		}
		insecure := profile
		insecure.RequireAuthenticatedNode = false
		if err := ValidateNetworkProfile(insecure); err == nil {
			t.Fatalf("%s accepted unauthenticated P2P profile", profile.Name)
		}
	}
}
