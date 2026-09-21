package chain

import (
	"testing"

	"indochain/internal/config"
	"indochain/internal/types"
)

func TestCalculateChainStatsGenesis(t *testing.T) {
	stats := CalculateChainStats([]types.Block{GenesisBlock()})
	if stats.Blocks != 1 || stats.TotalTransactions != 0 || stats.CoinbaseTransactions != 0 || stats.NormalTransactions != 0 || stats.CoinbaseBlocks != 0 {
		t.Fatalf("unexpected genesis stats: %#v", stats)
	}
}

func TestCalculateChainStatsCoinbaseAndNormal(t *testing.T) {
	genesis := GenesisBlock()
	coinbase := types.NewCoinbaseTransaction("iND10000000000000000000000000000000000000000", config.InitialBlockReward, 1)
	normal := types.Transaction{ID: "normal", From: "from", To: "to", Amount: 1}
	blocks := []types.Block{
		genesis,
		{Height: 1, Hash: "a", PreviousHash: genesis.Hash, Difficulty: config.InitialDifficulty, Transactions: []types.Transaction{coinbase}},
		{Height: 2, Hash: "b", PreviousHash: "a", Difficulty: config.InitialDifficulty, Transactions: []types.Transaction{types.NewCoinbaseTransaction("iND10000000000000000000000000000000000000000", config.InitialBlockReward, 2), normal}},
	}
	stats := CalculateChainStats(blocks)
	if stats.Blocks != 3 || stats.CoinbaseBlocks != 2 || stats.TotalTransactions != 3 || stats.CoinbaseTransactions != 2 || stats.NormalTransactions != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}
