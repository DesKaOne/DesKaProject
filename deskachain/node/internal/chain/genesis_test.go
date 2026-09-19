package chain

import (
	"testing"

	"deskachain/internal/config"
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
	const want = "6e1b3fed63a01109cb665c45c76796b7272b9fec08efcf057b6134a6dc71c22c"
	if got := GenesisBlockForNetwork(config.Localnet()).Hash; got != want {
		t.Fatalf("localnet genesis hash changed: got %s want %s", got, want)
	}
}

func TestTestnetGenesisStable(t *testing.T) {
	const want = "db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4"
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
