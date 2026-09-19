package ledger

import (
	"strings"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/types"
)

func TestCanonicalTransactionVersionWaitsForNetworkActivation(t *testing.T) {
	profile := config.Localnet()
	tx := types.Transaction{
		Version: types.TxVersionCanonical,
		From:    "from",
		To:      "to",
		Amount:  1,
	}

	l := New()
	err := l.ValidateTransaction(tx)
	if err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("expected inactive version error from ledger, got %v", err)
	}

	mature := NewMatureWithProfile(profile.Consensus, profile)
	err = mature.ValidateTransaction(tx)
	if err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("expected inactive version error from mature ledger, got %v", err)
	}
}
