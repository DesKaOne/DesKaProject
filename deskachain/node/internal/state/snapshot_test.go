package state

import (
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
)

func TestSnapshotForLedgerRoundTripValidation(t *testing.T) {
	l := ledger.NewMatureWithProfile(config.ConsensusParams{}, config.Localnet())
	snapshot, err := SnapshotForLedger(l)
	if err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Validate(); err != nil {
		t.Fatalf("snapshot validation failed: %v", err)
	}
	if snapshot.Version != SnapshotVersion {
		t.Fatalf("snapshot version = %d, want %d", snapshot.Version, SnapshotVersion)
	}
	if snapshot.Height != 0 {
		t.Fatalf("snapshot height = %d, want 0", snapshot.Height)
	}
}

func TestSnapshotRootIsIndependentOfInputOrder(t *testing.T) {
	accountsA := []ledger.StateAccount{
		{Address: "b", Confirmed: 2},
		{Address: "a", Confirmed: 1},
	}
	accountsB := []ledger.StateAccount{
		{Address: "a", Confirmed: 1},
		{Address: "b", Confirmed: 2},
	}
	rootA, err := RootForCollections(accountsA, nil)
	if err != nil {
		t.Fatal(err)
	}
	rootB, err := RootForCollections(accountsB, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rootA != rootB {
		t.Fatalf("roots differ for equivalent unordered inputs: %s != %s", rootA, rootB)
	}
}

func TestSnapshotRejectsTamperedRoot(t *testing.T) {
	snapshot := Snapshot{
		Version:   SnapshotVersion,
		Height:    0,
		Accounts:  []ledger.StateAccount{{Address: "a", Confirmed: 1}},
		StateRoot: "tampered",
	}
	if err := snapshot.Validate(); err == nil {
		t.Fatal("tampered snapshot accepted")
	}
}

func TestSnapshotEquivalentIncludesPendingCoinbaseState(t *testing.T) {
	a := Snapshot{
		Version:   SnapshotVersion,
		Height:    3,
		StateRoot: "root",
		Coinbases: []ledger.StateCoinbase{{Address: "IDR-a", Amount: 1, Height: 2}},
	}
	b := Snapshot{
		Version:   SnapshotVersion,
		Height:    3,
		StateRoot: "root",
		Coinbases: []ledger.StateCoinbase{{Address: "IDR-a", Amount: 2, Height: 2}},
	}
	if Equivalent(a, b) {
		t.Fatal("snapshots with different pending coinbase state considered equivalent")
	}
}

func TestSnapshotValidateRejectsFuturePendingCoinbase(t *testing.T) {
	snapshot := Snapshot{
		Version:   SnapshotVersion,
		Height:    3,
		StateRoot: "root",
		Coinbases: []ledger.StateCoinbase{{Address: "IDR-a", Amount: 1, Height: 4}},
	}
	// Root is populated from empty consensus state so validation reaches the
	// structural pending-coinbase checks below.
	snapshot.StateRoot, _ = RootForCollections(nil, nil)
	if err := snapshot.Validate(); err == nil {
		t.Fatal("future pending coinbase accepted")
	}
}
