package mempool

import (
	"testing"

	"indochain/internal/types"
	"indochain/internal/wallet"
)

func TestSelectWithNoncesPreservesSenderNonceOrder(t *testing.T) {
	p := admissionProfile()
	w, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}

	tx1 := signedTransfer(t, p, w, 1, 1)
	tx2 := signedTransfer(t, p, w, 10, 2)
	selected, err := SelectWithNonces([]types.Transaction{tx2, tx1}, p, 0, map[string]uint64{w.Address: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 || selected[0].Nonce != 1 || selected[1].Nonce != 2 {
		t.Fatalf("nonce order not preserved: %+v", selected)
	}
}

func TestSelectWithNoncesHoldsNonceGap(t *testing.T) {
	p := admissionProfile()
	w, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	tx2 := signedTransfer(t, p, w, 10, 2)
	selected, err := SelectWithNonces([]types.Transaction{tx2}, p, 0, map[string]uint64{w.Address: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 0 {
		t.Fatalf("nonce gap transaction selected: %+v", selected)
	}
}

func TestSelectWithNoncesAllowsIndependentSenders(t *testing.T) {
	p := admissionProfile()
	w1, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	w2, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	tx1 := signedTransfer(t, p, w1, 1, 1)
	tx2 := signedTransfer(t, p, w2, 10, 1)
	selected, err := SelectWithNonces([]types.Transaction{tx1, tx2}, p, 0, map[string]uint64{w1.Address: 1, w2.Address: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 || selected[0].ID != tx2.ID {
		t.Fatalf("unexpected sender ordering: %+v", selected)
	}
}
