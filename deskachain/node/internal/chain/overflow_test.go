package chain

import (
	"errors"
	"math"
	"testing"

	"indochain/internal/arith"
	"indochain/internal/config"
	"indochain/internal/types"
)

func TestValidateNextBlockRejectsHeightOverflow(t *testing.T) {
	profile := config.Localnet()
	tip := types.Block{Height: math.MaxUint64, Hash: "tip"}
	block := types.Block{Height: 0}

	err := ValidateNextBlockWithNetwork(
		block,
		tip,
		[]types.Block{tip},
		profile.Difficulty,
		profile.Consensus,
		profile,
	)
	if err == nil {
		t.Fatal("expected block height overflow")
	}
	if !errors.Is(err, arith.ErrOverflow) && err.Error() != "block height overflow" {
		t.Fatalf("expected height overflow, got %v", err)
	}
}
