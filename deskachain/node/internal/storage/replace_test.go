package storage

import (
	"strings"
	"testing"

	"deskachain/internal/types"
)

func TestReplaceFromHeightAtomicallyReplacesBranchAndTip(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	blocks := []types.Block{
		{Height: 0, Hash: "genesis"},
		{Height: 1, PreviousHash: "genesis", Hash: "old-1"},
		{Height: 2, PreviousHash: "old-1", Hash: "old-2"},
		{Height: 3, PreviousHash: "old-2", Hash: "old-3"},
	}
	for _, block := range blocks {
		if err := store.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	replacement := []types.Block{
		{Height: 2, PreviousHash: "old-1", Hash: "new-2"},
		{Height: 3, PreviousHash: "new-2", Hash: "new-3"},
		{Height: 4, PreviousHash: "new-3", Hash: "new-4"},
	}
	if err := store.ReplaceFromHeight(2, replacement); err != nil {
		t.Fatal(err)
	}

	got, err := store.Blocks()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("block count = %d, want 5", len(got))
	}
	if got[1].Hash != "old-1" || got[2].Hash != "new-2" || got[3].Hash != "new-3" || got[4].Hash != "new-4" {
		t.Fatalf("unexpected branch replacement: %#v", got)
	}

	tip, err := store.Tip()
	if err != nil {
		t.Fatal(err)
	}
	if tip.Height != 4 || tip.Hash != "new-4" {
		t.Fatalf("unexpected tip: %#v", tip)
	}
}

func TestReplaceFromHeightRejectsNonContiguousBranchWithoutChangingChain(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	original := []types.Block{
		{Height: 0, Hash: "genesis"},
		{Height: 1, PreviousHash: "genesis", Hash: "old-1"},
		{Height: 2, PreviousHash: "old-1", Hash: "old-2"},
	}
	for _, block := range original {
		if err := store.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
	}

	err = store.ReplaceFromHeight(2, []types.Block{
		{Height: 2, PreviousHash: "old-1", Hash: "new-2"},
		{Height: 4, PreviousHash: "new-2", Hash: "new-4"},
	})
	if err == nil || !strings.Contains(err.Error(), "not contiguous") {
		t.Fatalf("expected non-contiguous branch rejection, got %v", err)
	}

	block, err := store.GetBlockByHeight(2)
	if err != nil {
		t.Fatal(err)
	}
	if block.Hash != "old-2" {
		t.Fatalf("old block was changed after rejected replacement: %#v", block)
	}
	tip, err := store.Tip()
	if err != nil {
		t.Fatal(err)
	}
	if tip.Height != 2 || tip.Hash != "old-2" {
		t.Fatalf("tip changed after rejected replacement: %#v", tip)
	}
}

func TestReplaceFromHeightRejectsHeightOverflow(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	err = store.ReplaceFromHeight(^uint64(0), []types.Block{
		{Height: ^uint64(0), Hash: "max"},
		{Height: 0, Hash: "wrapped"},
	})
	if err == nil || !strings.Contains(err.Error(), "height overflow") {
		t.Fatalf("expected replacement height overflow, got %v", err)
	}
}
