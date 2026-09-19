package chain

import (
	"context"
	"errors"
	"strings"

	"deskachain/internal/config"
	"deskachain/internal/crypto"
	"deskachain/internal/types"
)

type MineOptions struct {
	MaxNonce   uint64
	Verbose    bool
	OnProgress func(nonce uint64, hash string)
}

var ErrMaxNonceReached = errors.New("max nonce reached")

func Mine(block types.Block) types.Block {
	mined, _ := MineWithContext(context.Background(), block, MineOptions{})
	return mined
}

func MineWithContext(ctx context.Context, block types.Block, opts MineOptions) (types.Block, error) {
	target := strings.Repeat("0", int(block.Difficulty))
	for {
		if block.Nonce%10000 == 0 {
			select {
			case <-ctx.Done():
				return types.Block{}, errors.New("mining cancelled")
			default:
			}
		}
		if opts.MaxNonce > 0 && block.Nonce > opts.MaxNonce {
			return types.Block{}, ErrMaxNonceReached
		}
		hash := crypto.DoubleSHA256Hex(block.HeaderBytesWithNonce(block.Nonce))
		if opts.Verbose && opts.OnProgress != nil && block.Nonce > 0 && block.Nonce%100000 == 0 {
			opts.OnProgress(block.Nonce, hash)
		}
		if strings.HasPrefix(hash, target) {
			block.Hash = hash
			return block, nil
		}
		block.Nonce++
	}
}

func ValidateProofOfWork(block types.Block) bool {
	return strings.HasPrefix(block.Hash, strings.Repeat("0", int(block.Difficulty))) &&
		block.Hash == block.CalculateHash()
}

func CalculateNextDifficulty(blocks []types.Block) uint32 {
	return CalculateNextDifficultyWithParams(blocks, config.Localnet().Difficulty)
}

func CalculateNextDifficultyWithParams(blocks []types.Block, params config.DifficultyParams) uint32 {
	params = normalizeDifficultyParams(params)
	if len(blocks) == 0 {
		return params.InitialDifficulty
	}
	tip := blocks[len(blocks)-1]
	if tip.Height == 0 || tip.Difficulty == 0 {
		return params.InitialDifficulty
	}
	if tip.Height < params.RetargetWindow {
		return clampDifficulty(tip.Difficulty, params)
	}
	if params.RetargetWindow == 0 || tip.Height%params.RetargetWindow != 0 {
		return clampDifficulty(tip.Difficulty, params)
	}
	firstHeight := tip.Height - params.RetargetWindow + 1
	var first types.Block
	found := false
	for _, block := range blocks {
		if block.Height == firstHeight {
			first = block
			found = true
			break
		}
	}
	if !found {
		return clampDifficulty(tip.Difficulty, params)
	}
	actualTimespan := tip.Timestamp - first.Timestamp
	expectedTimespan := params.TargetBlockTimeSeconds * int64(params.RetargetWindow-1)
	next := tip.Difficulty
	if actualTimespan <= 0 {
		next++
	} else if actualTimespan < expectedTimespan/2 {
		next++
	} else if actualTimespan > expectedTimespan*2 && next > 0 {
		next--
	}
	return clampDifficulty(next, params)
}

func defaultDifficulty() uint32 {
	return config.Localnet().Difficulty.InitialDifficulty
}

func normalizeDifficultyParams(params config.DifficultyParams) config.DifficultyParams {
	if params.InitialDifficulty == 0 {
		params.InitialDifficulty = config.InitialDifficulty
	}
	if params.MinDifficulty == 0 {
		params.MinDifficulty = 1
	}
	if params.MaxDifficulty == 0 {
		params.MaxDifficulty = params.InitialDifficulty
	}
	if params.MaxDifficulty < params.MinDifficulty {
		params.MaxDifficulty = params.MinDifficulty
	}
	if params.TargetBlockTimeSeconds <= 0 {
		params.TargetBlockTimeSeconds = config.BlockTimeTargetSecs
	}
	if params.RetargetWindow == 0 {
		params.RetargetWindow = 10
	}
	if params.MaxFutureDriftSeconds <= 0 {
		params.MaxFutureDriftSeconds = 900
	}
	return params
}

func clampDifficulty(difficulty uint32, params config.DifficultyParams) uint32 {
	if difficulty < params.MinDifficulty {
		return params.MinDifficulty
	}
	if difficulty > params.MaxDifficulty {
		return params.MaxDifficulty
	}
	return difficulty
}

func BlocksUntilRetarget(blocks []types.Block, params config.DifficultyParams) uint64 {
	params = normalizeDifficultyParams(params)
	if len(blocks) == 0 {
		return params.RetargetWindow
	}
	height := blocks[len(blocks)-1].Height
	if height == 0 {
		return params.RetargetWindow
	}
	remaining := params.RetargetWindow - (height % params.RetargetWindow)
	if remaining == params.RetargetWindow {
		return 0
	}
	return remaining
}
