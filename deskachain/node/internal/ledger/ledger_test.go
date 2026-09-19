package ledger

import (
	"errors"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestLedgerRejectsInsufficientBalance(t *testing.T) {
	from, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	to, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	tx := types.NewUnsignedTransaction(from.Address, to.Address, 1, 0, 1)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	if err := New().ValidateTransaction(tx); err == nil {
		t.Fatal("expected insufficient balance error")
	}
}

func TestMatureLedgerCoinbaseMaturityBalances(t *testing.T) {
	miner := newLedgerWallet(t)
	params := config.Localnet().Consensus

	blocks := maturityBlocks(miner.Address, 3)
	details, err := BalanceDetailsFor(miner.Address, blocks, nil, params)
	if err != nil {
		t.Fatal(err)
	}
	assertBalanceDetails(t, details, 150*config.UnitsPerCoin, 0, 150*config.UnitsPerCoin, 0)
	if got := CirculatingSupply(blocks, params); got != 0 {
		t.Fatalf("circulating supply height 3 = %d", got)
	}

	blocks = maturityBlocks(miner.Address, 11)
	details, err = BalanceDetailsFor(miner.Address, blocks, nil, params)
	if err != nil {
		t.Fatal(err)
	}
	assertBalanceDetails(t, details, 550*config.UnitsPerCoin, 50*config.UnitsPerCoin, 500*config.UnitsPerCoin, 50*config.UnitsPerCoin)
	if got := CirculatingSupply(blocks, params); got != 50*config.UnitsPerCoin {
		t.Fatalf("circulating supply height 11 = %d", got)
	}

	blocks = maturityBlocks(miner.Address, 15)
	details, err = BalanceDetailsFor(miner.Address, blocks, nil, params)
	if err != nil {
		t.Fatal(err)
	}
	assertBalanceDetails(t, details, 750*config.UnitsPerCoin, 250*config.UnitsPerCoin, 500*config.UnitsPerCoin, 250*config.UnitsPerCoin)
	if got := CirculatingSupply(blocks, params); got != 250*config.UnitsPerCoin {
		t.Fatalf("circulating supply height 15 = %d", got)
	}
}

func TestMatureLedgerRejectsImmatureSpendAndTracksPending(t *testing.T) {
	miner := newLedgerWallet(t)
	receiver := newLedgerWallet(t)
	params := config.Localnet().Consensus

	immature, err := ReplayMature(maturityBlocks(miner.Address, 3), params)
	if err != nil {
		t.Fatal(err)
	}
	tx := signedLedgerTx(t, miner, receiver.Address, 10*config.UnitsPerCoin, immature.Nonce(miner.Address)+1)
	if err := immature.ApplyTransaction(tx); !errors.Is(err, ErrImmatureBalance) {
		t.Fatalf("expected immature spend rejection, got %v", err)
	}
	if immature.Nonce(miner.Address) != 0 {
		t.Fatalf("nonce changed after failed immature spend")
	}

	mature, err := ReplayMature(maturityBlocks(miner.Address, 11), params)
	if err != nil {
		t.Fatal(err)
	}
	tx = signedLedgerTx(t, miner, receiver.Address, 10*config.UnitsPerCoin, mature.Nonce(miner.Address)+1)
	details := mature.BalanceDetails(miner.Address, []types.Transaction{tx}, 11)
	if details.PendingOutgoing != 10*config.UnitsPerCoin || details.Spendable != 40*config.UnitsPerCoin {
		t.Fatalf("unexpected pending details: %#v", details)
	}
	if err := mature.ApplyTransaction(tx); err != nil {
		t.Fatal(err)
	}
	receiverDetails := mature.BalanceDetails(receiver.Address, nil, 11)
	if receiverDetails.Mature != 10*config.UnitsPerCoin || receiverDetails.Spendable != 10*config.UnitsPerCoin {
		t.Fatalf("receiver normal tx not mature immediately: %#v", receiverDetails)
	}
}

func TestMatureLedgerStakingLockUnlockAndSpendable(t *testing.T) {
	miner := newLedgerWallet(t)
	receiver := newLedgerWallet(t)
	params := config.Localnet().Consensus
	blocks := maturityBlocks(miner.Address, 11)
	l, err := ReplayMature(blocks, params)
	if err != nil {
		t.Fatal(err)
	}
	lock := signedStakeLockTx(t, miner, 10*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	pendingDetails := l.BalanceDetails(miner.Address, []types.Transaction{lock}, 11)
	if pendingDetails.PendingStakeLock != 10*config.UnitsPerCoin || pendingDetails.Spendable != 40*config.UnitsPerCoin {
		t.Fatalf("pending stake lock not reflected: %#v", pendingDetails)
	}
	if err := l.ApplyTransactionAtHeight(lock, 12); err != nil {
		t.Fatal(err)
	}
	details := l.BalanceDetails(miner.Address, nil, 12)
	if details.ActiveStake != 10*config.UnitsPerCoin || details.Spendable != 40*config.UnitsPerCoin {
		t.Fatalf("active stake not reflected: %#v", details)
	}
	tooMuch := signedLedgerTx(t, miner, receiver.Address, 45*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ValidateTransaction(tooMuch); err == nil || err.Error() != "invalid transaction: spends locked stake" {
		t.Fatalf("expected locked stake rejection, got %v", err)
	}
	unlock := signedStakeUnlockTx(t, miner, lock.StakeID, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(unlock, 13); err != nil {
		t.Fatal(err)
	}
	details = l.BalanceDetails(miner.Address, nil, 13)
	if details.UnlockingStake != 10*config.UnitsPerCoin || details.Spendable != 40*config.UnitsPerCoin {
		t.Fatalf("unlocking stake not reflected: %#v", details)
	}
	// Mine/replay enough empty blocks to cross release height.
	blocks = append(blocks, types.Block{Height: 12, Transactions: []types.Transaction{lock}})
	blocks = append(blocks, types.Block{Height: 13, Transactions: []types.Transaction{unlock}})
	for h := uint64(14); h <= 23; h++ {
		blocks = append(blocks, types.Block{Height: h})
	}
	released, err := BalanceDetailsFor(miner.Address, blocks, nil, params)
	if err != nil {
		t.Fatal(err)
	}
	if released.ReleasedStake != 10*config.UnitsPerCoin || released.Spendable != 550*config.UnitsPerCoin {
		t.Fatalf("released stake should be spendable: %#v", released)
	}
}

func TestMatureLedgerRejectsInvalidStakeOperations(t *testing.T) {
	miner := newLedgerWallet(t)
	other := newLedgerWallet(t)
	params := config.Localnet().Consensus
	l, err := ReplayMature(maturityBlocks(miner.Address, 11), params)
	if err != nil {
		t.Fatal(err)
	}
	belowMin := signedStakeLockTx(t, miner, config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ValidateTransaction(belowMin); err == nil || err.Error() != "invalid stake lock: amount below minimum" {
		t.Fatalf("expected below minimum rejection, got %v", err)
	}
	lock := signedStakeLockTx(t, miner, 10*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(lock, 12); err != nil {
		t.Fatal(err)
	}
	badUnlock := signedStakeUnlockTx(t, other, lock.StakeID, 1)
	if err := l.ValidateTransaction(badUnlock); err == nil || err.Error() != "invalid stake unlock: owner mismatch" {
		t.Fatalf("expected owner mismatch, got %v", err)
	}
	missing := signedStakeUnlockTx(t, miner, "missing", l.Nonce(miner.Address)+1)
	if err := l.ValidateTransaction(missing); err == nil || err.Error() != "invalid stake unlock: stake not found" {
		t.Fatalf("expected missing stake, got %v", err)
	}
}

func TestStakeLockImmatureCoinbaseRejected(t *testing.T) {
	miner := newLedgerWallet(t)
	params := config.Localnet().Consensus
	l, err := ReplayMature(maturityBlocks(miner.Address, 3), params)
	if err != nil {
		t.Fatal(err)
	}
	lock := signedStakeLockTx(t, miner, 10*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	err = l.ValidateTransaction(lock)
	if err == nil || err.Error() != "invalid stake lock: insufficient mature spendable balance" {
		t.Fatalf("expected immature/spendable rejection, got %v", err)
	}
}

func TestStakeLockAfterMaturityAccepted(t *testing.T) {
	miner := newLedgerWallet(t)
	params := config.Localnet().Consensus
	l, err := ReplayMature(maturityBlocks(miner.Address, 11), params)
	if err != nil {
		t.Fatal(err)
	}
	lock := signedStakeLockTx(t, miner, 10*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(lock, 12); err != nil {
		t.Fatal(err)
	}
	details := l.BalanceDetails(miner.Address, nil, 12)
	if details.ActiveStake != 10*config.UnitsPerCoin {
		t.Fatalf("active stake = %d", details.ActiveStake)
	}
}

func TestActiveStakeReducesSpendable(t *testing.T) {
	miner := newLedgerWallet(t)
	params := config.Localnet().Consensus
	l, err := ReplayMature(maturityBlocks(miner.Address, 12), params)
	if err != nil {
		t.Fatal(err)
	}
	lock := signedStakeLockTx(t, miner, 40*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(lock, 13); err != nil {
		t.Fatal(err)
	}
	details := l.BalanceDetails(miner.Address, nil, 13)
	if details.Mature != 100*config.UnitsPerCoin || details.ActiveStake != 40*config.UnitsPerCoin || details.Spendable != 60*config.UnitsPerCoin {
		t.Fatalf("unexpected stake details: %#v", details)
	}
}

func TestUnlockingStakeReducesSpendable(t *testing.T) {
	miner := newLedgerWallet(t)
	params := config.Localnet().Consensus
	l, err := ReplayMature(maturityBlocks(miner.Address, 12), params)
	if err != nil {
		t.Fatal(err)
	}
	lock := signedStakeLockTx(t, miner, 40*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(lock, 13); err != nil {
		t.Fatal(err)
	}
	unlock := signedStakeUnlockTx(t, miner, lock.StakeID, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(unlock, 14); err != nil {
		t.Fatal(err)
	}
	details := l.BalanceDetails(miner.Address, nil, 14)
	if details.UnlockingStake != 40*config.UnitsPerCoin || details.Spendable != 60*config.UnitsPerCoin {
		t.Fatalf("unlocking stake should remain locked: %#v", details)
	}
}

func TestReleasedStakeRestoresSpendable(t *testing.T) {
	miner := newLedgerWallet(t)
	params := config.Localnet().Consensus
	blocks := maturityBlocks(miner.Address, 12)
	l, err := ReplayMature(blocks, params)
	if err != nil {
		t.Fatal(err)
	}
	lock := signedStakeLockTx(t, miner, 40*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(lock, 13); err != nil {
		t.Fatal(err)
	}
	unlock := signedStakeUnlockTx(t, miner, lock.StakeID, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(unlock, 14); err != nil {
		t.Fatal(err)
	}
	blocks = append(blocks, types.Block{Height: 13, Transactions: []types.Transaction{lock}})
	blocks = append(blocks, types.Block{Height: 14, Transactions: []types.Transaction{unlock}})
	for h := uint64(15); h <= 24; h++ {
		blocks = append(blocks, types.Block{Height: h})
	}
	details, err := BalanceDetailsFor(miner.Address, blocks, nil, params)
	if err != nil {
		t.Fatal(err)
	}
	if details.ReleasedStake != 40*config.UnitsPerCoin || details.Spendable != 600*config.UnitsPerCoin {
		t.Fatalf("released stake should restore spendable: %#v", details)
	}
}

func TestPendingStakeLockReducesSpendable(t *testing.T) {
	miner := newLedgerWallet(t)
	params := config.Localnet().Consensus
	l, err := ReplayMature(maturityBlocks(miner.Address, 12), params)
	if err != nil {
		t.Fatal(err)
	}
	lock := signedStakeLockTx(t, miner, 70*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	details := l.BalanceDetails(miner.Address, []types.Transaction{lock}, 12)
	if details.PendingStakeLock != 70*config.UnitsPerCoin || details.Spendable != 30*config.UnitsPerCoin {
		t.Fatalf("pending stake lock should reduce spendable to 30: %#v", details)
	}
}

func TestPendingOutgoingAndStakeLockBothReduceSpendable(t *testing.T) {
	miner := newLedgerWallet(t)
	receiver := newLedgerWallet(t)
	params := config.Localnet().Consensus
	l, err := ReplayMature(maturityBlocks(miner.Address, 13), params)
	if err != nil {
		t.Fatal(err)
	}
	lock := signedStakeLockTx(t, miner, 20*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(lock, 14); err != nil {
		t.Fatal(err)
	}
	pending := signedLedgerTx(t, miner, receiver.Address, 30*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	details := l.BalanceDetails(miner.Address, []types.Transaction{pending}, 14)
	if details.ActiveStake != 20*config.UnitsPerCoin || details.PendingOutgoing != 30*config.UnitsPerCoin || details.Spendable != 100*config.UnitsPerCoin {
		t.Fatalf("active stake and pending outgoing should both reduce spendable: %#v", details)
	}
}

func TestSendCannotSpendActiveStake(t *testing.T) {
	miner := newLedgerWallet(t)
	receiver := newLedgerWallet(t)
	params := config.Localnet().Consensus
	l, err := ReplayMature(maturityBlocks(miner.Address, 12), params)
	if err != nil {
		t.Fatal(err)
	}
	lock := signedStakeLockTx(t, miner, 90*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(lock, 13); err != nil {
		t.Fatal(err)
	}
	tx := signedLedgerTx(t, miner, receiver.Address, 11*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ValidateTransaction(tx); err == nil || err.Error() != "invalid transaction: spends locked stake" {
		t.Fatalf("expected locked stake rejection, got %v", err)
	}
}

func TestSendAtExactSpendableSucceeds(t *testing.T) {
	miner := newLedgerWallet(t)
	receiver := newLedgerWallet(t)
	params := config.Localnet().Consensus
	l, err := ReplayMature(maturityBlocks(miner.Address, 13), params)
	if err != nil {
		t.Fatal(err)
	}
	lock := signedStakeLockTx(t, miner, 55*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(lock, 14); err != nil {
		t.Fatal(err)
	}
	unlock := signedStakeUnlockTx(t, miner, lock.StakeID, l.Nonce(miner.Address)+1)
	if err := l.ApplyTransactionAtHeight(unlock, 15); err != nil {
		t.Fatal(err)
	}
	tx := signedLedgerTx(t, miner, receiver.Address, 95*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	if err := l.ValidateTransaction(tx); err != nil {
		t.Fatalf("exact spendable tx rejected: %v", err)
	}
}

func TestLedgerAcceptsCoinbaseReward(t *testing.T) {
	miner, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	block := types.Block{
		Height:       1,
		MinerAddress: miner.Address,
		Transactions: []types.Transaction{
			types.NewCoinbaseTransaction(miner.Address, config.InitialBlockReward, 1),
		},
	}
	l := New()
	if err := l.ApplyBlock(block); err != nil {
		t.Fatal(err)
	}
	if got := l.Balance(miner.Address); got != config.InitialBlockReward {
		t.Fatalf("balance = %d, want %d", got, config.InitialBlockReward)
	}
}

func TestLedgerSecp256k1TransactionSignatureValidation(t *testing.T) {
	from, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	to, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	l := New()
	if err := l.ApplyCoinbase(types.NewCoinbaseTransaction(from.Address, config.InitialBlockReward, 1)); err != nil {
		t.Fatal(err)
	}
	tx := types.NewUnsignedTransaction(from.Address, to.Address, 10, 0, 1)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	if err := l.ValidateTransaction(tx); err != nil {
		t.Fatalf("valid tx rejected: %v", err)
	}
	tampered := tx
	tampered.Amount = 11
	tampered.RefreshID()
	if err := l.ValidateTransaction(tampered); err == nil {
		t.Fatal("tampered signed tx accepted")
	}
	other, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	mismatch := tx
	mismatch.From = other.Address
	mismatch.RefreshID()
	if err := l.ValidateTransaction(mismatch); err == nil {
		t.Fatal("address/public key mismatch accepted")
	}
}

func maturityBlocks(miner string, count uint64) []types.Block {
	blocks := []types.Block{{Height: 0}}
	for height := uint64(1); height <= count; height++ {
		blocks = append(blocks, types.Block{
			Height:       height,
			MinerAddress: miner,
			Transactions: []types.Transaction{
				types.NewCoinbaseTransaction(miner, config.InitialBlockReward, height),
			},
		})
	}
	return blocks
}

func newLedgerWallet(t *testing.T) wallet.Wallet {
	t.Helper()
	w, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func signedLedgerTx(t *testing.T, from wallet.Wallet, to string, value uint64, nonce uint64) types.Transaction {
	t.Helper()
	tx := types.NewUnsignedTransaction(from.Address, to, value, 0, nonce)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	return tx
}

func signedStakeLockTx(t *testing.T, from wallet.Wallet, value uint64, nonce uint64) types.Transaction {
	t.Helper()
	tx := types.NewStakeLockTransaction(from.Address, value, nonce)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	tx.StakeID = tx.ID
	return tx
}

func signedStakeUnlockTx(t *testing.T, from wallet.Wallet, stakeID string, nonce uint64) types.Transaction {
	t.Helper()
	tx := types.NewStakeUnlockTransaction(from.Address, stakeID, nonce)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	return tx
}

func assertBalanceDetails(t *testing.T, details BalanceDetails, confirmed, mature, immature, spendable uint64) {
	t.Helper()
	if details.Confirmed != confirmed || details.Mature != mature || details.Immature != immature || details.Spendable != spendable {
		t.Fatalf("unexpected balance details: %#v", details)
	}
}
