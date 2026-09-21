package cpuminer

import (
	"context"
	"sync/atomic"
	"testing"

	"indochain/internal/config"
	"indochain/internal/types"
	"indochain/internal/wallet"
)

func TestFirstNoncesPartitionByThread(t *testing.T) {
	got := FirstNonces(4)
	want := []uint64{0, 1, 2, 3}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("nonce %d = %d want %d", i, got[i], want[i])
		}
	}
}

func TestMineStopsWorkersAfterBlockFound(t *testing.T) {
	miner, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	block := types.NewBlock(1, "prev", miner.Address, 1, []types.Transaction{
		types.NewCoinbaseTransaction(miner.Address, config.InitialBlockReward, 1),
	})
	var hashes atomic.Uint64
	result, err := Mine(context.Background(), block, 4, &hashes)
	if err != nil {
		t.Fatal(err)
	}
	if result.Block.Hash == "" || result.Thread < 0 || result.Thread > 3 || hashes.Load() == 0 {
		t.Fatalf("unexpected mining result: %#v hashes=%d", result, hashes.Load())
	}
}
