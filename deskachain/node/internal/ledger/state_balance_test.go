package ledger

import (
	"testing"

	"indochain/internal/asset"
	"indochain/internal/staking"
	"indochain/internal/types"
)

func TestBalanceDetailsFromStateMatchesStateAccounting(t *testing.T) {
	account := StateAccount{
		Address:   "iND-alice",
		Confirmed: 100,
		Mature:    100,
		Nonce:     7,
	}
	stakes := []staking.Record{
		{
			StakeID:      "stake-active",
			OwnerAddress: "iND-alice",
			Amount:       20,
			Status:       staking.StatusActive,
		},
		{
			StakeID:       "stake-unlocking",
			OwnerAddress:  "iND-alice",
			Amount:        10,
			Status:        staking.StatusUnlocking,
			ReleaseHeight: 20,
		},
		{
			StakeID:      "stake-released",
			OwnerAddress: "iND-alice",
			Amount:       5,
			Status:       staking.StatusReleased,
		},
	}
	pending := []types.Transaction{
		{From: "iND-alice", To: "iND-bob", Amount: 11, Fee: 2, Type: types.TxTypeTransfer},
		{From: "iND-bob", To: "iND-alice", Amount: 7, Type: types.TxTypeTransfer},
		{From: "iND-alice", To: "iND-alice", Amount: 15, Type: types.TxTypeStakeLock},
		{From: "COINBASE", To: "iND-alice", Amount: 999, Coinbase: true},
	}

	details := BalanceDetailsFromState(
		"iND-alice",
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
	if details.Spendable != 42 {
		t.Fatalf("spendable = %d, want 42", details.Spendable)
	}
	if details.CoinbaseMaturity != 100 || details.CurrentHeight != 12 {
		t.Fatalf("unexpected metadata: %#v", details)
	}
}

func TestBalanceDetailsFromStateNeverUnderflowsSpendable(t *testing.T) {
	details := BalanceDetailsFromState(
		"iND-alice",
		StateAccount{Address: "iND-alice", Mature: 10},
		[]staking.Record{
			{StakeID: "stake", OwnerAddress: "iND-alice", Amount: 10, Status: staking.StatusActive},
		},
		[]types.Transaction{
			{From: "iND-alice", To: "iND-bob", Amount: 100, Fee: 1, Type: types.TxTypeTransfer},
		},
		100,
		1,
	)
	if details.Spendable != 0 {
		t.Fatalf("spendable = %d, want 0", details.Spendable)
	}
}

func TestBalanceDetailsFromStateSeparatesTokenAmountFromINDFee(t *testing.T) {
	pending := []types.Transaction{
		{
			Version:  types.TxVersionAsset,
			Type:     types.TxTypeTransfer,
			From:     "iND-alice",
			To:       "iND-bob",
			AssetID:  "asset:usd",
			Amount:   1000,
			Fee:      3,
			FeePayer: "iND-alice",
		},
		{
			Version:  types.TxVersionAsset,
			Type:     types.TxTypeTransfer,
			From:     "iND-carol",
			To:       "iND-dave",
			AssetID:  asset.NativeAssetID,
			Amount:   20,
			Fee:      4,
			FeePayer: "iND-sponsor",
		},
	}
	details := BalanceDetailsFromState(
		"iND-alice",
		StateAccount{Address: "iND-alice", Confirmed: 100, Mature: 100},
		nil,
		pending,
		0,
		1,
	)
	if details.PendingOutgoing != 3 || details.PendingIncoming != 0 {
		t.Fatalf("alice pending balances = %#v, want outgoing=3 incoming=0", details)
	}
	details = BalanceDetailsFromState(
		"iND-sponsor",
		StateAccount{Address: "iND-sponsor", Confirmed: 10, Mature: 10},
		nil,
		pending,
		0,
		1,
	)
	if details.PendingOutgoing != 4 {
		t.Fatalf("sponsor pending outgoing=%d want 4", details.PendingOutgoing)
	}
}
