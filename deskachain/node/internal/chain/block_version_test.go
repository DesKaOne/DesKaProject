package chain

import (
	"strings"
	"testing"

	"indochain/internal/config"
	"indochain/internal/types"
)

func TestBlockVersionActivationGate(t *testing.T) {
	if err := types.ValidateBlockVersion(types.BlockVersionCanonical, config.Localnet().BlockVersion); err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("expected inactive canonical block version, got %v", err)
	}
	if err := types.ValidateBlockVersion(types.BlockVersionCanonical, types.BlockVersionCanonical); err != nil {
		t.Fatal(err)
	}
	if err := types.ValidateBlockVersion(99, types.BlockVersionCanonical); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported block version, got %v", err)
	}
}
