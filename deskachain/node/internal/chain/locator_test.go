package chain

import (
	"testing"

	"deskachain/internal/types"
)

func TestBuildBlockLocatorGenesis(t *testing.T) {
	genesis := GenesisBlock()
	locator := BuildBlockLocator([]types.Block{genesis})
	if len(locator) != 1 || locator[0].Height != 0 || locator[0].Hash != genesis.Hash {
		t.Fatalf("unexpected genesis locator: %#v", locator)
	}
}

func TestBuildBlockLocatorHeightTen(t *testing.T) {
	blocks := fakeBlocks(10)
	locator := BuildBlockLocator(blocks)
	wantHeights := []uint64{10, 9, 8, 6, 2, 0}
	if len(locator) != len(wantHeights) {
		t.Fatalf("locator len = %d, want %d: %#v", len(locator), len(wantHeights), locator)
	}
	seen := map[uint64]struct{}{}
	for i, want := range wantHeights {
		if locator[i].Height != want {
			t.Fatalf("locator[%d].height = %d, want %d: %#v", i, locator[i].Height, want, locator)
		}
		if _, ok := seen[locator[i].Height]; ok {
			t.Fatalf("duplicate locator height: %#v", locator)
		}
		seen[locator[i].Height] = struct{}{}
		if i > 0 && locator[i].Height >= locator[i-1].Height {
			t.Fatalf("locator not descending: %#v", locator)
		}
	}
}

func TestFindCommonAncestorAndCumulativeWork(t *testing.T) {
	a := fakeBlocks(3)
	b := fakeBlocks(3)
	ancestor, ok := FindCommonAncestor(a, BuildBlockLocator(b))
	if !ok || ancestor.Height != 3 {
		t.Fatalf("same chain ancestor = %#v found=%t", ancestor, ok)
	}
	b[1].Hash = "fork-1"
	b[2].Hash = "fork-2"
	b[3].Hash = "fork-3"
	ancestor, ok = FindCommonAncestor(a, BuildBlockLocator(b))
	if !ok || ancestor.Height != 0 {
		t.Fatalf("fork ancestor = %#v found=%t", ancestor, ok)
	}
	if _, ok := FindCommonAncestor(a, []BlockLocatorEntry{{Height: 0, Hash: "other-genesis"}}); ok {
		t.Fatal("expected no common ancestor")
	}
	if CalculateCumulativeWork(a) <= CalculateCumulativeWork(a[:1]) {
		t.Fatalf("cumulative work did not increase")
	}
}

func fakeBlocks(height uint64) []types.Block {
	blocks := make([]types.Block, 0, height+1)
	for i := uint64(0); i <= height; i++ {
		prev := ""
		if i > 0 {
			prev = blocks[i-1].Hash
		}
		blocks = append(blocks, types.Block{Height: i, Hash: fakeHash(i), PreviousHash: prev, Difficulty: uint32(i)})
	}
	return blocks
}

func fakeHash(height uint64) string {
	return string(rune('a' + height))
}
