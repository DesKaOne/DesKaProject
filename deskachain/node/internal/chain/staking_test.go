package chain

import (
	"strings"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/staking"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestChainValidateStakeLockBlock(t *testing.T) {
	miner := chainTestWallet(t)
	blocks := minedCoinbaseBlocks(t, miner, 11)
	l, err := ledger.ReplayMature(blocks, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	lock := chainStakeLockTx(t, miner, 10*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	blocks = append(blocks, minedNextBlock(t, blocks, miner.Address, []types.Transaction{lock}))
	if _, err := ValidateChainWithNetwork(blocks, stakingTestProfile()); err != nil {
		t.Fatal(err)
	}
	state, err := staking.Replay(blocks, config.Localnet().Consensus.Staking)
	if err != nil {
		t.Fatal(err)
	}
	record, ok := state.Find(lock.StakeID, blocks[len(blocks)-1].Height)
	if !ok || record.Status != staking.StatusActive {
		t.Fatalf("stake should be active after valid block: ok=%t record=%+v", ok, record)
	}
}

func TestChainValidateRejectStakeLockOverSpendable(t *testing.T) {
	miner := chainTestWallet(t)
	blocks := minedCoinbaseBlocks(t, miner, 11)
	l, err := ledger.ReplayMature(blocks, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	lock := chainStakeLockTx(t, miner, 60*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	blocks = append(blocks, minedNextBlock(t, blocks, miner.Address, []types.Transaction{lock}))
	if _, err := ValidateChainWithNetwork(blocks, stakingTestProfile()); err == nil || !strings.Contains(err.Error(), "insufficient mature spendable balance") {
		t.Fatalf("expected stake over-spend rejection, got %v", err)
	}
}

func TestChainValidateRejectStakeLockImmature(t *testing.T) {
	miner := chainTestWallet(t)
	blocks := minedCoinbaseBlocks(t, miner, 3)
	l, err := ledger.ReplayMature(blocks, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	lock := chainStakeLockTx(t, miner, 10*config.UnitsPerCoin, l.Nonce(miner.Address)+1)
	blocks = append(blocks, minedNextBlock(t, blocks, miner.Address, []types.Transaction{lock}))
	if _, err := ValidateChainWithNetwork(blocks, stakingTestProfile()); err == nil || !strings.Contains(err.Error(), "insufficient mature spendable balance") {
		t.Fatalf("expected immature stake rejection, got %v", err)
	}
}

func TestChainValidateRejectStakeUnlockMissingStake(t *testing.T) {
	miner := chainTestWallet(t)
	blocks := minedCoinbaseBlocks(t, miner, 11)
	l, err := ledger.ReplayMature(blocks, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	unlock := chainStakeUnlockTx(t, miner, "missing", l.Nonce(miner.Address)+1)
	blocks = append(blocks, minedNextBlock(t, blocks, miner.Address, []types.Transaction{unlock}))
	if _, err := ValidateChainWithNetwork(blocks, stakingTestProfile()); err == nil || !strings.Contains(err.Error(), "stake not found") {
		t.Fatalf("expected missing stake rejection, got %v", err)
	}
}

func TestChainValidateRejectStakeUnlockOwnerMismatch(t *testing.T) {
	miner := chainTestWallet(t)
	other := chainTestWallet(t)
	blocks := chainWithActiveStake(t, miner, 10*config.UnitsPerCoin)
	l, err := ledger.ReplayMature(blocks, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	state, err := staking.Replay(blocks, config.Localnet().Consensus.Staking)
	if err != nil {
		t.Fatal(err)
	}
	record := state.Records(blocks[len(blocks)-1].Height)[0]
	unlock := chainStakeUnlockTx(t, other, record.StakeID, l.Nonce(other.Address)+1)
	blocks = append(blocks, minedNextBlock(t, blocks, miner.Address, []types.Transaction{unlock}))
	if _, err := ValidateChainWithNetwork(blocks, stakingTestProfile()); err == nil || !strings.Contains(err.Error(), "owner mismatch") {
		t.Fatalf("expected owner mismatch rejection, got %v", err)
	}
}

func TestChainValidateRejectTransferSpendingLockedStake(t *testing.T) {
	miner := chainTestWallet(t)
	receiver := chainTestWallet(t)
	blocks := chainWithActiveStake(t, miner, 90*config.UnitsPerCoin)
	l, err := ledger.ReplayMature(blocks, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	tx := types.NewUnsignedTransaction(miner.Address, receiver.Address, 61*config.UnitsPerCoin, 0, l.Nonce(miner.Address)+1)
	if err := miner.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	blocks = append(blocks, minedNextBlock(t, blocks, miner.Address, []types.Transaction{tx}))
	if _, err := ValidateChainWithNetwork(blocks, stakingTestProfile()); err == nil || !strings.Contains(err.Error(), "spends locked stake") {
		t.Fatalf("expected locked stake spend rejection, got %v", err)
	}
}

func TestStakeDoesNotChangeTotalSupply(t *testing.T) {
	miner := chainTestWallet(t)
	blocks := chainWithActiveStake(t, miner, 10*config.UnitsPerCoin)
	before := ledger.TotalSupply(blocks)
	l, err := ledger.ReplayMature(blocks, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	state, err := staking.Replay(blocks, config.Localnet().Consensus.Staking)
	if err != nil {
		t.Fatal(err)
	}
	record := state.Records(blocks[len(blocks)-1].Height)[0]
	unlock := chainStakeUnlockTx(t, miner, record.StakeID, l.Nonce(miner.Address)+1)
	blocks = append(blocks, minedNextBlock(t, blocks, miner.Address, []types.Transaction{unlock}))
	for i := 0; i < int(config.Localnet().Consensus.Staking.UnbondingPeriodBlocks); i++ {
		blocks = append(blocks, minedNextBlock(t, blocks, miner.Address, nil))
	}
	if _, err := ValidateChainWithNetwork(blocks, stakingTestProfile()); err != nil {
		t.Fatal(err)
	}
	if after := ledger.TotalSupply(blocks); after != before+uint64(1+config.Localnet().Consensus.Staking.UnbondingPeriodBlocks)*config.InitialBlockReward {
		t.Fatalf("stake changed supply unexpectedly: before=%d after=%d", before, after)
	}
}

func chainWithActiveStake(t *testing.T, miner wallet.Wallet, value uint64) []types.Block {
	t.Helper()
	blocks := minedCoinbaseBlocks(t, miner, 12)
	l, err := ledger.ReplayMature(blocks, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	lock := chainStakeLockTx(t, miner, value, l.Nonce(miner.Address)+1)
	return append(blocks, minedNextBlock(t, blocks, miner.Address, []types.Transaction{lock}))
}

func minedCoinbaseBlocks(t *testing.T, miner wallet.Wallet, count uint64) []types.Block {
	t.Helper()
	blocks := []types.Block{GenesisBlock()}
	for i := uint64(0); i < count; i++ {
		blocks = append(blocks, minedNextBlock(t, blocks, miner.Address, nil))
	}
	return blocks
}

func minedNextBlock(t *testing.T, prior []types.Block, miner string, txs []types.Transaction) types.Block {
	t.Helper()
	tip := prior[len(prior)-1]
	height := tip.Height + 1
	allTxs := append([]types.Transaction{types.NewCoinbaseTransaction(miner, config.InitialBlockReward, height)}, txs...)
	block := types.NewBlock(height, tip.Hash, miner, CalculateNextDifficultyWithParams(prior, stakingTestProfile().Difficulty), allTxs)
	return Mine(block)
}

func chainStakeLockTx(t *testing.T, from wallet.Wallet, value uint64, nonce uint64) types.Transaction {
	t.Helper()
	tx := types.NewStakeLockTransaction(from.Address, value, nonce)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	tx.StakeID = tx.ID
	return tx
}

func chainStakeUnlockTx(t *testing.T, from wallet.Wallet, stakeID string, nonce uint64) types.Transaction {
	t.Helper()
	tx := types.NewStakeUnlockTransaction(from.Address, stakeID, nonce)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	return tx
}

func chainTestWallet(t *testing.T) wallet.Wallet {
	t.Helper()
	w, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func stakingTestProfile() config.NetworkConfig {
	profile := config.Localnet()
	profile.Difficulty.InitialDifficulty = 1
	profile.Difficulty.MinDifficulty = 1
	profile.Difficulty.MaxDifficulty = 1
	profile.Difficulty.RetargetWindow = 1000
	return profile
}
