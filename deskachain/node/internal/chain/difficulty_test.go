package chain

import (
	"math"
	"strings"
	"testing"
	"time"

	"indochain/internal/config"
	"indochain/internal/types"
)

func TestCalculateNextDifficultyInitialAndBeforeRetarget(t *testing.T) {
	params := config.Localnet().Difficulty
	blocks := []types.Block{GenesisBlock()}
	if got := CalculateNextDifficultyWithParams(blocks, params); got != params.InitialDifficulty {
		t.Fatalf("initial difficulty = %d, want %d", got, params.InitialDifficulty)
	}
	for height := uint64(1); height < params.RetargetWindow; height++ {
		blocks = append(blocks, syntheticBlock(height, 100+int64(height)*params.TargetBlockTimeSeconds, params.InitialDifficulty))
		if got := CalculateNextDifficultyWithParams(blocks, params); got != params.InitialDifficulty {
			t.Fatalf("height %d next difficulty = %d", height, got)
		}
	}
}

func TestCalculateNextDifficultyRetargetFastSlowNormal(t *testing.T) {
	params := config.Localnet().Difficulty
	fast := syntheticDifficultyWindow(params, params.InitialDifficulty, 1)
	if got := CalculateNextDifficultyWithParams(fast, params); got != params.InitialDifficulty+1 {
		t.Fatalf("fast next difficulty = %d", got)
	}
	slow := syntheticDifficultyWindow(params, params.InitialDifficulty, 60)
	if got := CalculateNextDifficultyWithParams(slow, params); got != params.InitialDifficulty-1 {
		t.Fatalf("slow next difficulty = %d", got)
	}
	normal := syntheticDifficultyWindow(params, params.InitialDifficulty, params.TargetBlockTimeSeconds)
	if got := CalculateNextDifficultyWithParams(normal, params); got != params.InitialDifficulty {
		t.Fatalf("normal next difficulty = %d", got)
	}
}

func TestCalculateNextDifficultyClamp(t *testing.T) {
	params := config.Localnet().Difficulty
	params.MinDifficulty = 1
	params.MaxDifficulty = 8
	slow := syntheticDifficultyWindow(params, params.MinDifficulty, 60)
	if got := CalculateNextDifficultyWithParams(slow, params); got != params.MinDifficulty {
		t.Fatalf("min clamp difficulty = %d", got)
	}
	fast := syntheticDifficultyWindow(params, params.MaxDifficulty, 1)
	if got := CalculateNextDifficultyWithParams(fast, params); got != params.MaxDifficulty {
		t.Fatalf("max clamp difficulty = %d", got)
	}
}

func TestValidateChainCatchesInvalidDifficultyAndPoW(t *testing.T) {
	params := config.Localnet().Difficulty
	genesis := GenesisBlock()
	wrongDifficulty := validShapeBlock(genesis, 1, params.InitialDifficulty-1)
	if _, err := ValidateChainWithParams([]types.Block{genesis, wrongDifficulty}, params); err == nil || !strings.Contains(err.Error(), "invalid difficulty at height 1 expected 4 got 3") {
		t.Fatalf("expected invalid difficulty, got %v", err)
	}
	badPoW := validShapeBlock(genesis, 1, params.InitialDifficulty)
	if strings.HasPrefix(badPoW.Hash, "0000") {
		t.Skip("synthetic hash unexpectedly satisfies PoW")
	}
	if _, err := ValidateChainWithParams([]types.Block{genesis, badPoW}, params); err == nil || !strings.Contains(err.Error(), "invalid proof of work") {
		t.Fatalf("expected invalid pow, got %v", err)
	}
}

func TestValidateChainCatchesInvalidTimestamps(t *testing.T) {
	params := config.Localnet().Difficulty
	genesis := GenesisBlock()
	beforeParent := validShapeBlock(genesis, 1, params.InitialDifficulty)
	beforeParent.Timestamp = genesis.Timestamp - 1
	beforeParent.Hash = beforeParent.CalculateHash()
	if _, err := ValidateChainWithParams([]types.Block{genesis, beforeParent}, params); err == nil || !strings.Contains(err.Error(), "invalid block timestamp: before parent") {
		t.Fatalf("expected before parent timestamp error, got %v", err)
	}
	future := validShapeBlock(genesis, 1, params.InitialDifficulty)
	future.Timestamp = time.Now().Unix() + params.MaxFutureDriftSeconds + 1
	future.Hash = future.CalculateHash()
	if _, err := ValidateChainWithParams([]types.Block{genesis, future}, params); err == nil || !strings.Contains(err.Error(), "invalid block timestamp: too far in future") {
		t.Fatalf("expected future timestamp error, got %v", err)
	}
}

func TestCalculateBlockWorkAndCumulativeWork(t *testing.T) {
	cases := map[uint32]uint64{
		0: 1,
		1: 16,
		2: 256,
		4: 65536,
		8: 4294967296,
	}
	for difficulty, want := range cases {
		if got := CalculateBlockWork(difficulty); got != want {
			t.Fatalf("work(%d) = %d, want %d", difficulty, got, want)
		}
	}
	if got := CalculateBlockWork(16); got != math.MaxUint64 {
		t.Fatalf("overflow work = %d", got)
	}
	blocks := []types.Block{
		{Height: 0, Difficulty: 0},
		{Height: 1, Difficulty: 4},
		{Height: 2, Difficulty: 4},
		{Height: 3, Difficulty: 4},
	}
	if got, want := CalculateCumulativeWork(blocks), uint64(196609); got != want {
		t.Fatalf("cumulative work = %d, want %d", got, want)
	}
}

func TestValidateChainPassesAfterRetargetAndWorkAccumulates(t *testing.T) {
	params := config.Localnet().Difficulty
	params.InitialDifficulty = 1
	params.MinDifficulty = 1
	params.MaxDifficulty = 3
	params.RetargetWindow = 2
	params.TargetBlockTimeSeconds = 30
	blocks := []types.Block{GenesisBlock()}
	for height := uint64(1); height <= 4; height++ {
		difficulty := CalculateNextDifficultyWithParams(blocks, params)
		block := validShapeBlock(blocks[len(blocks)-1], height, difficulty)
		block.Timestamp = blocks[len(blocks)-1].Timestamp + 1
		block = Mine(block)
		blocks = append(blocks, block)
	}
	if blocks[3].Difficulty <= blocks[1].Difficulty {
		t.Fatalf("expected fast retarget to increase difficulty: h1=%d h3=%d", blocks[1].Difficulty, blocks[3].Difficulty)
	}
	if _, err := ValidateChainWithParams(blocks, params); err != nil {
		t.Fatal(err)
	}
	if work := CalculateCumulativeWork(blocks); work <= uint64(len(blocks)) {
		t.Fatalf("cumulative work did not increase meaningfully: %d", work)
	}
}

func syntheticDifficultyWindow(params config.DifficultyParams, difficulty uint32, interval int64) []types.Block {
	blocks := []types.Block{GenesisBlock()}
	start := int64(1000)
	for height := uint64(1); height <= params.RetargetWindow; height++ {
		blocks = append(blocks, syntheticBlock(height, start+int64(height)*interval, difficulty))
	}
	return blocks
}

func syntheticBlock(height uint64, timestamp int64, difficulty uint32) types.Block {
	return types.Block{Height: height, Timestamp: timestamp, Difficulty: difficulty}
}

func validShapeBlock(parent types.Block, height uint64, difficulty uint32) types.Block {
	tx := types.NewCoinbaseTransaction("iND10000000000000000000000000000000000000000", 0, height)
	block := types.NewBlock(height, parent.Hash, tx.To, difficulty, []types.Transaction{tx})
	block.Timestamp = parent.Timestamp + 1
	block.Hash = block.CalculateHash()
	return block
}
