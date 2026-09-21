package state

import (
	"testing"

	"indochain/internal/config"
	"indochain/internal/types"
	"indochain/internal/wallet"
)

func TestSnapshotAfterBlockPreservesCoinbaseMaturityState(t *testing.T) {
	profile := config.Localnet()
	params := profile.Consensus
	params.CoinbaseMaturity = 2

	w, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	genesis := types.NewBlockWithVersion(0, "", w.Address, 1, []types.Transaction{
		types.NewCoinbaseTransaction(w.Address, 10, 0),
	}, types.BlockVersionLegacy)

	block1 := types.NewBlockWithVersion(1, genesis.Hash, w.Address, 1, []types.Transaction{
		types.NewCoinbaseTransaction(w.Address, 10, 1),
	}, types.BlockVersionLegacy)

	block2 := types.NewBlockWithVersion(2, block1.Hash, w.Address, 1, []types.Transaction{
		types.NewCoinbaseTransaction(w.Address, 10, 2),
	}, types.BlockVersionLegacy)

	base, err := SnapshotForBlocks([]types.Block{genesis}, params, profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(base.Coinbases) != 1 || base.Coinbases[0].Height != 0 {
		t.Fatalf("unexpected pending coinbases at height 0: %#v", base.Coinbases)
	}

	incremental, err := SnapshotAfterBlock(base, block1, params, profile)
	if err != nil {
		t.Fatal(err)
	}
	full, err := SnapshotForBlocks([]types.Block{genesis, block1}, params, profile)
	if err != nil {
		t.Fatal(err)
	}
	if incremental.StateRoot != full.StateRoot || incremental.Height != full.Height {
		t.Fatalf("incremental snapshot diverges: incremental=%#v full=%#v", incremental, full)
	}
	if len(incremental.Coinbases) != 2 {
		t.Fatalf("pending coinbases = %d, want 2", len(incremental.Coinbases))
	}

	incremental2, err := SnapshotAfterBlock(incremental, block2, params, profile)
	if err != nil {
		t.Fatal(err)
	}
	full2, err := SnapshotForBlocks([]types.Block{genesis, block1, block2}, params, profile)
	if err != nil {
		t.Fatal(err)
	}
	if incremental2.StateRoot != full2.StateRoot || incremental2.Height != full2.Height {
		t.Fatalf("second incremental snapshot diverges: incremental=%#v full=%#v", incremental2, full2)
	}
	if len(incremental2.Coinbases) != 2 {
		t.Fatalf("pending coinbases at height 2 = %d, want 2", len(incremental2.Coinbases))
	}
}

func TestSnapshotAfterBlockRejectsNonSequentialHeight(t *testing.T) {
	profile := config.Localnet()
	snapshot, err := SnapshotForBlocks(nil, profile.Consensus, profile)
	if err != nil {
		t.Fatal(err)
	}
	block := types.Block{Height: 2}
	if _, err := SnapshotAfterBlock(snapshot, block, profile.Consensus, profile); err == nil {
		t.Fatal("non-sequential block height accepted")
	}
}
