package chain

import (
	"errors"
	"strings"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/state"
	"deskachain/internal/storage"
)

func TestInitRebuildsMissingStateDB(t *testing.T) {
	profile := config.Localnet()
	store, err := storage.OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	bc := New(store)
	if err := bc.InitWithProfile(profile); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteState(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadState(); !errors.Is(err, storage.ErrStateNotInitialized) {
		t.Fatalf("expected missing state after delete, got %v", err)
	}

	if err := bc.InitWithProfile(profile); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	tip, err := store.Tip()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Height != tip.Height {
		t.Fatalf("state height = %d, tip height = %d", snapshot.Height, tip.Height)
	}
	if snapshot.StateRoot == "" {
		t.Fatal("rebuilt state root is empty")
	}
}

func TestValidateStateWithNetworkDetectsTamperedState(t *testing.T) {
	profile := config.Localnet()
	store, err := storage.OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	bc := New(store)
	if err := bc.InitWithProfile(profile); err != nil {
		t.Fatal(err)
	}

	result, err := bc.ValidateStateWithNetwork(profile)
	if err != nil {
		t.Fatalf("fresh state validation failed: %v", err)
	}
	if !result.Valid {
		t.Fatalf("fresh state reported invalid: %#v", result)
	}

	snapshot, err := store.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Accounts = append(snapshot.Accounts, ledger.StateAccount{
		Address:   "DKC-tampered",
		Confirmed: 1,
		Mature:    1,
	})
	snapshot.StateRoot, err = state.RootForCollections(snapshot.Accounts, snapshot.Stakes)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}

	result, err = bc.ValidateStateWithNetwork(profile)
	if err == nil || !strings.Contains(err.Error(), "does not match canonical chain") {
		t.Fatalf("expected state mismatch, got result=%#v err=%v", result, err)
	}
	if result.Valid {
		t.Fatalf("tampered state reported valid: %#v", result)
	}
}

