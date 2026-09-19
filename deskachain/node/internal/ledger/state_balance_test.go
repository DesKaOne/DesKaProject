package ledger

import (
	"testing"

	"deskachain/internal/staking"
	"deskachain/internal/types"
)

func TestBalanceDetailsFromStateMatchesStateAccounting(t *testing.T) {
	account := StateAccount{
		Address:   "DKC-alice",
		Confirmed: 100,
		Mature:    100,
		Nonce:     7,
	}
	stakes := []staking.Record{
		{
			StakeID:      "stake-active",
			OwnerAddress: "DKC-alice",
			Amount:       20,
			Status:       staking.StatusActive,
		},
		{
			StakeID:       "stake-unlocking",
			OwnerAddress:  "DKC-alice",
			Amount:        10,
			Status:        staking.StatusUnlocking,
			ReleaseHeight: 20,
		},
		{
			StakeID:      "stake-released",
			OwnerAddress: "DKC-alice",
			Amount:       5,
			Status:       staking.StatusReleased,
		},
	}
	pending := []types.Transaction{
		{From: "DKC-alice", To: "DKC-bob", Amount: 11, Fee: 2, Type: types.TxTypeTransfer},
		{From: "DKC-bob", To: "DKC-alice", Amount: 7, Type: types.TxTypeTransfer},
		{From: "DKC-alice", To: "DKC-alice", Amount: 15, Type: types.TxTypeStakeLock},
		{From: "COINBASE", To: "DKC-alice", Amount: 999, Coinbase: true},
	}

	details := BalanceDetailsFromState(
		"DKC-alice",
		account,
		stakes,
		pending,
		100,
		12,
	)

	if details.Confirmed != 100 || details.Mature != 100 {
		t.Fatalf("unexpected balances: %#v", details)
	}
	if details.PendingOutgoing != 13 || details.PendingIncoming != 7 {
		t.Fatalf("unexpected pending balances: %#v", details)
	}
	if details.ActiveStake != 20 || details.UnlockingStake != 10 || details.ReleasedStake != 5 {
		t.Fatalf("unexpected stake balances: %#v", details)
	}
	// Spendable = mature - active/unlocking stake - pending stake lock - pending outgoing.
	if details.Spendable != 47 {
		t.Fatalf("spendable = %d, want 47", details.Spendable)
	}
	if details.CoinbaseMaturity != 100 || details.CurrentHeight != 12 {
		t.Fatalf("unexpected metadata: %#v", details)
	}
}

func TestBalanceDetailsFromStateNeverUnderflowsSpendable(t *testing.T) {
	details := BalanceDetailsFromState(
		"DKC-alice",
		StateAccount{Address: "DKC-alice", Mature: 10},
		[]staking.Record{
			{StakeID: "stake", OwnerAddress: "DKC-alice", Amount: 10, Status: staking.StatusActive},
		},
		[]types.Transaction{
			{From: "DKC-alice", To: "DKC-bob", Amount: 100, Fee: 1, Type: types.TxTypeTransfer},
		},
		100,
		1,
	)
	if details.Spendable != 0 {
		t.Fatalf("spendable = %d, want 0", details.Spendable)
	}
}
