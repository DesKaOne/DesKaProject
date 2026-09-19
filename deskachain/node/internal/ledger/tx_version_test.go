package ledger

import (
	"strings"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestCanonicalTransactionVersionWaitsForNetworkActivation(t *testing.T) {
	profile := config.Localnet()
	profile.TxVersion = types.TxVersionLegacy
	from, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	to, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	tx := types.Transaction{
		Version: types.TxVersionCanonical,
		From:    from.Address,
		To:      to.Address,
		Amount:  1,
		Nonce:   1,
	}
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}

	l := New()
	err = l.ValidateTransaction(tx)
	if err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("expected inactive version error from ledger, got %v", err)
	}

	mature := NewMatureWithProfile(profile.Consensus, profile)
	err = mature.ValidateTransaction(tx)
	if err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("expected inactive version error from mature ledger, got %v", err)
	}
}
