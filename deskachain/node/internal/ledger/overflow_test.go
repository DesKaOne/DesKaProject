package ledger

import (
	"errors"
	"math"
	"strings"
	"testing"

	"deskachain/internal/arith"
	"deskachain/internal/config"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestLedgerRejectsCoinbaseBalanceOverflow(t *testing.T) {
	miner := mustTestWallet(t)
	l := New()
	l.accounts[miner.Address] = Account{Balance: math.MaxUint64}

	err := l.ApplyCoinbase(types.NewCoinbaseTransaction(miner.Address, 1, 1))
	if !errors.Is(err, arith.ErrOverflow) {
		t.Fatalf("expected coinbase overflow, got %v", err)
	}
	if got := l.accounts[miner.Address].Balance; got != math.MaxUint64 {
		t.Fatalf("coinbase overflow changed balance to %d", got)
	}
}

func TestLedgerRejectsTransactionCostOverflow(t *testing.T) {
	from := mustTestWallet(t)
	to := mustTestWallet(t)
	l := New()
	l.accounts[from.Address] = Account{Balance: math.MaxUint64}

	tx := types.NewUnsignedTransaction(from.Address, to.Address, math.MaxUint64, 1, 1)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}

	err := l.ValidateTransaction(tx)
	if !errors.Is(err, arith.ErrOverflow) {
		t.Fatalf("expected transaction cost overflow, got %v", err)
	}
}

func TestLedgerRejectsRecipientBalanceOverflowWithoutStateMutation(t *testing.T) {
	from := mustTestWallet(t)
	to := mustTestWallet(t)
	l := New()
	l.accounts[from.Address] = Account{Balance: 1, Nonce: 0}
	l.accounts[to.Address] = Account{Balance: math.MaxUint64}

	tx := types.NewUnsignedTransaction(from.Address, to.Address, 1, 0, 1)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}

	err := l.ApplyTransaction(tx)
	if !errors.Is(err, arith.ErrOverflow) {
		t.Fatalf("expected recipient balance overflow, got %v", err)
	}
	if got := l.accounts[from.Address]; got.Balance != 1 || got.Nonce != 0 {
		t.Fatalf("sender state mutated after failed transfer: %+v", got)
	}
	if got := l.accounts[to.Address].Balance; got != math.MaxUint64 {
		t.Fatalf("recipient balance mutated after failed transfer: %d", got)
	}
}

func TestLedgerRejectsNonceOverflow(t *testing.T) {
	from := mustTestWallet(t)
	to := mustTestWallet(t)
	l := New()
	l.accounts[from.Address] = Account{Balance: config.InitialBlockReward, Nonce: math.MaxUint64}

	tx := types.NewUnsignedTransaction(from.Address, to.Address, 1, 0, 0)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}

	err := l.ValidateTransaction(tx)
	if !errors.Is(err, arith.ErrOverflow) && !strings.Contains(err.Error(), "account nonce overflow") {
		t.Fatalf("expected nonce overflow, got %v", err)
	}
}

func mustTestWallet(t *testing.T) wallet.Wallet {
	t.Helper()
	w, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestLedgerRejectsFutureTransactionVersion(t *testing.T) {
	from := mustTestWallet(t)
	to := mustTestWallet(t)
	l := New()
	tx := types.NewUnsignedTransaction(from.Address, to.Address, 1, 0, 1)
	tx.Version = types.MaxSupportedTxVersion + 1
	err := l.ValidateTransaction(tx)
	if err == nil || !strings.Contains(err.Error(), "unsupported transaction version") {
		t.Fatalf("expected future transaction version rejection, got %v", err)
	}
}
