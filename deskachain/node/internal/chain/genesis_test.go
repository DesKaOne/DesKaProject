package chain

import (
	"testing"

	"indochain/internal/config"
)

func TestGenesisHashDeterministic(t *testing.T) {
	first := GenesisBlock()
	second := GenesisBlock()
	if first.Hash == "" {
		t.Fatal("genesis hash is empty")
	}
	if first.Hash != second.Hash {
		t.Fatalf("genesis hash changed: %s != %s", first.Hash, second.Hash)
	}
}

func TestLocalnetGenesisUnchanged(t *testing.T) {
	const want = "63eaea2c6ad50193c31f956584b74a084f9113461b7e03a866a153a7b8cd88a3"
	if got := GenesisBlockForNetwork(config.Localnet()).Hash; got != want {
		t.Fatalf("localnet genesis hash changed: got %s want %s", got, want)
	}
}

func TestTestnetGenesisStable(t *testing.T) {
	const want = "f037ce35443c149714590cde076e99e15f2ed183637a2331e0e2fdc226b16709"
	first := GenesisBlockForNetwork(config.Testnet())
	second := GenesisBlockForNetwork(config.Testnet())
	if first.Hash == "" {
		t.Fatal("testnet genesis hash is empty")
	}
	if first.Hash != want {
		t.Fatalf("testnet genesis hash changed: got %s want %s", first.Hash, want)
	}
	if first.Hash != second.Hash {
		t.Fatalf("testnet genesis hash changed: %s != %s", first.Hash, second.Hash)
	}
	if first.Hash == GenesisBlockForNetwork(config.Localnet()).Hash {
		t.Fatalf("testnet genesis must differ from localnet: %s", first.Hash)
	}
}

func TestMainnetGenesisMatchesFrozenHash(t *testing.T) {
	profile := config.Mainnet()
	genesis := GenesisBlockForNetwork(profile)
	if genesis.Hash != config.MainnetGenesisHash {
		t.Fatalf("mainnet genesis hash = %s, want %s", genesis.Hash, config.MainnetGenesisHash)
	}
	if profile.GenesisHash != genesis.Hash {
		t.Fatalf("mainnet profile genesis hash = %s, generated genesis = %s", profile.GenesisHash, genesis.Hash)
	}
}

func TestGenesisHashForNetworkUsesFrozenMainnetIdentity(t *testing.T) {
	profile := config.Mainnet()
	if got := GenesisHashForNetwork(profile); got != config.MainnetGenesisHash {
		t.Fatalf("mainnet canonical genesis hash = %s, want %s", got, config.MainnetGenesisHash)
	}
	profile.GenesisHash = "tampered"
	if got := GenesisHashForNetwork(profile); got != config.MainnetGenesisHash {
		t.Fatalf("mainnet canonical genesis must ignore injected genesis hash: got %s want %s", got, config.MainnetGenesisHash)
	}
}

func TestGenesisHashForNetworkMatchesGeneratedNonMainnetGenesis(t *testing.T) {
	for _, profile := range []config.NetworkConfig{config.Localnet(), config.Testnet()} {
		want := GenesisBlockForNetwork(profile).Hash
		if got := GenesisHashForNetwork(profile); got != want {
			t.Fatalf("%s canonical genesis hash = %s, want %s", profile.Name, got, want)
		}
	}
}
