package storage

import (
	"errors"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/state"
	"deskachain/internal/types"
)

func emptySnapshot(t *testing.T) state.Snapshot {
	t.Helper()
	l := ledger.NewMatureWithProfile(config.ConsensusParams{}, config.Localnet())
	snapshot, err := state.SnapshotForLedger(l)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestStateStorePersistsMetadataAndIndexes(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snapshot := emptySnapshot(t)
	snapshot.Accounts = append(snapshot.Accounts, ledger.StateAccount{
		Address:   "DKC-test",
		Confirmed: 10,
		Mature:    10,
	})
	snapshot.StateRoot, err = state.RootForCollections(snapshot.Accounts, snapshot.Stakes)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if got.StateRoot != snapshot.StateRoot || len(got.Accounts) != 1 {
		t.Fatalf("unexpected restored snapshot: %#v", got)
	}
	if got.Accounts[0].Address != "DKC-test" || got.Accounts[0].Confirmed != 10 {
		t.Fatalf("unexpected account index contents: %#v", got.Accounts)
	}
}

func TestStateStoreMissingStateIsExplicit(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, err = store.LoadState()
	if !errors.Is(err, ErrStateNotInitialized) {
		t.Fatalf("expected ErrStateNotInitialized, got %v", err)
	}
}

func TestSaveBlockAndStatePersistsBothDatasets(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snapshot := emptySnapshot(t)
	block := types.Block{Height: 0, Hash: "genesis"}
	if err := store.SaveBlockAndState(block, snapshot); err != nil {
		t.Fatal(err)
	}

	has, err := store.HasChain()
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("chain was not persisted")
	}
	got, err := store.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if got.Height != 0 || got.StateRoot != snapshot.StateRoot {
		t.Fatalf("unexpected persisted state: %#v", got)
	}
}
