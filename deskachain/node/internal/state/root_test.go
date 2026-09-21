package state

import (
	"testing"

	"indochain/internal/config"
	"indochain/internal/ledger"
	"indochain/internal/types"
	"indochain/internal/wallet"
)

func TestStateRootDeterministicAcrossMapInsertionOrder(t *testing.T) {
	profile := config.Localnet()
	params := profile.Consensus
	wa, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	wb, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}

	first := ledger.NewMatureWithProfile(params, profile)
	if err := first.ApplyCoinbaseAtHeight(types.NewCoinbaseTransaction(wa.Address, 10, 1), 1); err != nil {
		t.Fatal(err)
	}
	if err := first.ApplyCoinbaseAtHeight(types.NewCoinbaseTransaction(wb.Address, 20, 2), 2); err != nil {
		t.Fatal(err)
	}
	rootA, err := RootForLedger(first)
	if err != nil {
		t.Fatal(err)
	}
	second := ledger.NewMatureWithProfile(params, profile)
	if err := second.ApplyCoinbaseAtHeight(types.NewCoinbaseTransaction(wb.Address, 20, 2), 2); err != nil {
		t.Fatal(err)
	}
	if err := second.ApplyCoinbaseAtHeight(types.NewCoinbaseTransaction(wa.Address, 10, 1), 1); err != nil {
		t.Fatal(err)
	}
	rootB, err := RootForLedger(second)
	if err != nil {
		t.Fatal(err)
	}
	if rootA != rootB {
		t.Fatalf("same state produced different roots: %s != %s", rootA, rootB)
	}
}

func TestStateRootChangesWhenAccountStateChanges(t *testing.T) {
	profile := config.Localnet()
	w, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	params := profile.Consensus
	first := ledger.NewMatureWithProfile(params, profile)
	second := ledger.NewMatureWithProfile(params, profile)
	if err := first.ApplyCoinbaseAtHeight(types.NewCoinbaseTransaction(w.Address, 10, 1), 1); err != nil {
		t.Fatal(err)
	}
	if err := second.ApplyCoinbaseAtHeight(types.NewCoinbaseTransaction(w.Address, 11, 1), 1); err != nil {
		t.Fatal(err)
	}
	rootA, err := RootForLedger(first)
	if err != nil {
		t.Fatal(err)
	}
	rootB, err := RootForLedger(second)
	if err != nil {
		t.Fatal(err)
	}
	if rootA == rootB {
		t.Fatal("different account state produced identical state roots")
	}
}

func TestStateRootRejectsNilLedger(t *testing.T) {
	if _, err := RootForLedger(nil); err == nil {
		t.Fatal("expected nil ledger error")
	}
}

func TestRootForBlockMatchesRootForBlocks(t *testing.T) {
	profile := config.Localnet()
	block := types.NewBlockWithVersion(1, "prev", "", 1, nil, types.BlockVersionLegacy)
	rootA, err := RootForBlock(nil, block, profile.Consensus, profile)
	if err != nil {
		t.Fatal(err)
	}
	rootB, err := RootForBlocks([]types.Block{block}, profile.Consensus, profile)
	if err != nil {
		t.Fatal(err)
	}
	if rootA != rootB {
		t.Fatalf("root helper mismatch: %s != %s", rootA, rootB)
	}
}
