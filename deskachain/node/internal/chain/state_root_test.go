package chain

import (
	"testing"

	"indochain/internal/config"
	"indochain/internal/state"
	"indochain/internal/types"
	"indochain/internal/wallet"
)

func TestCanonicalBlockStateRootValidatesFromPreBlockState(t *testing.T) {
	profile := config.Localnet()
	profile.BlockVersion = types.BlockVersionCanonical
	profile.TxVersion = types.TxVersionCanonical
	profile.Difficulty.InitialDifficulty = 1
	profile.Difficulty.MinDifficulty = 1
	profile.Difficulty.MaxDifficulty = 1

	w, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	genesis := GenesisBlockForNetwork(profile)
	coinbase := types.NewCoinbaseTransactionWithVersion(w.Address, config.InitialBlockReward, 1, profile.TxVersion)
	if err := coinbase.RefreshIDForChainID(profile.ChainID); err != nil {
		t.Fatal(err)
	}
	block := types.NewBlockWithVersion(1, genesis.Hash, w.Address, 1, []types.Transaction{coinbase}, profile.BlockVersion)
	root, err := state.RootForBlock([]types.Block{genesis}, block, profile.Consensus, profile)
	if err != nil {
		t.Fatal(err)
	}
	block.StateRoot = root
	block = Mine(block)

	if err := ValidateBlockWithNetwork(block, []types.Block{genesis}, profile); err != nil {
		t.Fatalf("valid canonical state root rejected: %v", err)
	}

	bad := block
	bad.StateRoot = "bad-state-root"
	bad.Hash = bad.CalculateHash()
	if err := ValidateBlockWithNetwork(bad, []types.Block{genesis}, profile); err == nil {
		t.Fatal("invalid canonical state root accepted")
	}
}
