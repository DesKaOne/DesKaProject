package chain

import (
	"errors"
	"testing"

	"deskachain/internal/config"
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
