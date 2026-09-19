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

func TestMatureLedgerRejectsCoinbaseBalanceOverflow(t *testing.T) {
	miner := mustMatureTestWallet(t)
	l := NewMatureWithProfile(config.Localnet().Consensus, config.Localnet())
	l.accounts[miner.Address] = MatureAccount{Confirmed: math.MaxUint64}

	err := l.ApplyCoinbaseAtHeight(types.NewCoinbaseTransaction(miner.Address, 1, 1), 1)
	if !errors.Is(err, arith.ErrOverflow) {
		t.Fatalf("expected mature coinbase overflow, got %v", err)
	}
	if got := l.accounts[miner.Address].Confirmed; got != math.MaxUint64 {
		t.Fatalf("coinbase overflow changed confirmed balance to %d", got)
	}
}

func TestMatureLedgerRejectsTransactionCostOverflow(t *testing.T) {
	from := mustMatureTestWallet(t)
	to := mustMatureTestWallet(t)
	l := NewMatureWithProfile(config.Localnet().Consensus, config.Localnet())
	l.accounts[from.Address] = MatureAccount{Confirmed: math.MaxUint64, Mature: math.MaxUint64}

	tx := types.NewUnsignedTransaction(from.Address, to.Address, math.MaxUint64, 1, 1)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}

	err := l.ValidateTransaction(tx)
	if !errors.Is(err, arith.ErrOverflow) {
		t.Fatalf("expected mature transaction cost overflow, got %v", err)
	}
}

func TestMatureLedgerRejectsRecipientBalanceOverflowWithoutStateMutation(t *testing.T) {
	from := mustMatureTestWallet(t)
	to := mustMatureTestWallet(t)
	l := NewMatureWithProfile(config.Localnet().Consensus, config.Localnet())
	l.accounts[from.Address] = MatureAccount{Confirmed: 1, Mature: 1, Nonce: 0}
	l.accounts[to.Address] = MatureAccount{Confirmed: math.MaxUint64, Mature: math.MaxUint64}

	tx := types.NewUnsignedTransaction(from.Address, to.Address, 1, 0, 1)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}

	err := l.ApplyTransactionAtHeight(tx, 1)
	if !errors.Is(err, arith.ErrOverflow) {
		t.Fatalf("expected mature recipient overflow, got %v", err)
	}
	if got := l.accounts[from.Address]; got.Confirmed != 1 || got.Mature != 1 || got.Nonce != 0 {
		t.Fatalf("sender state mutated after failed transfer: %+v", got)
	}
	if got := l.accounts[to.Address]; got.Confirmed != math.MaxUint64 || got.Mature != math.MaxUint64 {
		t.Fatalf("recipient state mutated after failed transfer: %+v", got)
	}
}

func TestMatureLedgerRejectsNonceOverflow(t *testing.T) {
	from := mustMatureTestWallet(t)
	to := mustMatureTestWallet(t)
	l := NewMatureWithProfile(config.Localnet().Consensus, config.Localnet())
	l.accounts[from.Address] = MatureAccount{Confirmed: config.InitialBlockReward, Mature: config.InitialBlockReward, Nonce: math.MaxUint64}

	tx := types.NewUnsignedTransaction(from.Address, to.Address, 1, 0, 0)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}

	err := l.ValidateTransaction(tx)
	if !strings.Contains(err.Error(), "account nonce overflow") {
		t.Fatalf("expected mature nonce overflow, got %v", err)
	}
}

func mustMatureTestWallet(t *testing.T) wallet.Wallet {
	t.Helper()
	w, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	return w
}
