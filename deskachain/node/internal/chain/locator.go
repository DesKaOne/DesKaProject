package chain

import (
	"math"

	"indochain/internal/types"
)

type BlockLocatorEntry struct {
	Height uint64 `json:"height"`
	Hash   string `json:"hash"`
}

func BuildBlockLocator(blocks []types.Block) []BlockLocatorEntry {
	if len(blocks) == 0 {
		return nil
	}
	byHeight := make(map[uint64]string, len(blocks))
	for _, block := range blocks {
		byHeight[block.Height] = block.Hash
	}
	tip := blocks[len(blocks)-1]
	seen := make(map[uint64]struct{})
	locator := make([]BlockLocatorEntry, 0)
	step := uint64(2)
	linearSteps := 2
	for height := tip.Height; ; {
		if hash, ok := byHeight[height]; ok {
			if _, duplicate := seen[height]; !duplicate {
				locator = append(locator, BlockLocatorEntry{Height: height, Hash: hash})
				seen[height] = struct{}{}
			}
		}
		if height == 0 {
			break
		}
		if linearSteps > 0 {
			height--
			linearSteps--
			continue
		}
		if height <= step {
			height = 0
		} else {
			height -= step
		}
		step *= 2
	}
	if _, ok := seen[0]; !ok {
		if hash, ok := byHeight[0]; ok {
			locator = append(locator, BlockLocatorEntry{Height: 0, Hash: hash})
		}
	}
	return locator
}

func FindCommonAncestor(blocks []types.Block, locator []BlockLocatorEntry) (BlockLocatorEntry, bool) {
	byHeight := make(map[uint64]string, len(blocks))
	for _, block := range blocks {
		byHeight[block.Height] = block.Hash
	}
	for _, entry := range locator {
		if hash, ok := byHeight[entry.Height]; ok && hash == entry.Hash {
			return entry, true
		}
	}
	return BlockLocatorEntry{}, false
}

func CalculateBlockWork(difficulty uint32) uint64 {
	if difficulty == 0 {
		return 1
	}
	if difficulty >= 16 {
		return math.MaxUint64
	}
	var work uint64 = 1
	for i := uint32(0); i < difficulty; i++ {
		if work > math.MaxUint64/16 {
			return math.MaxUint64
		}
		work *= 16
	}
	return work
}

func CalculateCumulativeWork(blocks []types.Block) uint64 {
	var work uint64
	for _, block := range blocks {
		blockWork := CalculateBlockWork(block.Difficulty)
		if math.MaxUint64-work < blockWork {
			return math.MaxUint64
		}
		work += blockWork
	}
	return work
}
