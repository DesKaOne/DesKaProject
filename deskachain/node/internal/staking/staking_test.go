package staking

import (
	"strings"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/types"
)

const (
	stakeOwner = "IDR1111111111111111111111111111111111"
	otherOwner = "IDR2222222222222222222222222222222222"
)

func TestLocalnetStakingParams(t *testing.T) {
	params := config.Localnet().Consensus.Staking
	if !params.Enabled {
		t.Fatal("localnet staking should be enabled")
	}
	if params.MinStakeAmount != 10*config.UnitsPerCoin {
		t.Fatalf("min stake = %d", params.MinStakeAmount)
	}
	if params.MinServiceStake != 100*config.UnitsPerCoin {
		t.Fatalf("min service stake = %d", params.MinServiceStake)
	}
	if params.UnbondingPeriodBlocks != 10 {
		t.Fatalf("unbonding period = %d", params.UnbondingPeriodBlocks)
	}
	if params.MaxActiveStakesPerAddress != 10 {
		t.Fatalf("max active stakes = %d", params.MaxActiveStakesPerAddress)
	}
}

func TestStakeRecordStatusLifecycle(t *testing.T) {
	state := NewState(testParams())
	lock := stakeLockTx("stake1", stakeOwner, 10*config.UnitsPerCoin, 1)
	if err := state.ApplyLock(lock, 5); err != nil {
		t.Fatal(err)
	}
	record, ok := state.Find(lock.StakeID, 5)
	if !ok || record.Status != StatusActive || record.LockHeight != 5 {
		t.Fatalf("unexpected active record: ok=%t record=%+v", ok, record)
	}
	unlock := stakeUnlockTx("unlock1", stakeOwner, lock.StakeID, 2)
	if err := state.ApplyUnlock(unlock, 9); err != nil {
		t.Fatal(err)
	}
	record, _ = state.Find(lock.StakeID, 18)
	if record.Status != StatusUnlocking || record.ReleaseHeight != 19 {
		t.Fatalf("unexpected unlocking record before release: %+v", record)
	}
	record, _ = state.Find(lock.StakeID, 19)
	if record.Status != StatusReleased {
		t.Fatalf("stake should be released at release height: %+v", record)
	}
}

func TestLockedAmountByAddress(t *testing.T) {
	state := NewState(testParams())
	mustApplyLock(t, state, "active", stakeOwner, 11*config.UnitsPerCoin, 1)
	mustApplyLock(t, state, "unlocking", stakeOwner, 12*config.UnitsPerCoin, 2)
	if err := state.ApplyUnlock(stakeUnlockTx("unlocking_tx", stakeOwner, "unlocking", 3), 7); err != nil {
		t.Fatal(err)
	}
	mustApplyLock(t, state, "released", stakeOwner, 13*config.UnitsPerCoin, 4)
	if err := state.ApplyUnlock(stakeUnlockTx("released_tx", stakeOwner, "released", 5), 1); err != nil {
		t.Fatal(err)
	}
	active, unlocking, released := state.AddressSummary(stakeOwner, 11)
	if active != 11*config.UnitsPerCoin || unlocking != 12*config.UnitsPerCoin || released != 13*config.UnitsPerCoin {
		t.Fatalf("unexpected summary active=%d unlocking=%d released=%d", active, unlocking, released)
	}
}

func TestTotalActiveAndUnlockingStake(t *testing.T) {
	state := NewState(testParams())
	mustApplyLock(t, state, "active", stakeOwner, 10*config.UnitsPerCoin, 1)
	mustApplyLock(t, state, "unlocking", stakeOwner, 20*config.UnitsPerCoin, 2)
	if err := state.ApplyUnlock(stakeUnlockTx("unlocking_tx", stakeOwner, "unlocking", 3), 9); err != nil {
		t.Fatal(err)
	}
	mustApplyLock(t, state, "released", stakeOwner, 30*config.UnitsPerCoin, 4)
	if err := state.ApplyUnlock(stakeUnlockTx("released_tx", stakeOwner, "released", 5), 1); err != nil {
		t.Fatal(err)
	}
	summary := state.Summary(11)
	if summary.TotalActiveStake != 10*config.UnitsPerCoin || summary.TotalUnlockingStake != 20*config.UnitsPerCoin || summary.TotalReleasedStake != 30*config.UnitsPerCoin || summary.ActiveStakeCount != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestStakeBelowMinimumRejected(t *testing.T) {
	err := NewState(testParams()).ApplyLock(stakeLockTx("stake1", stakeOwner, config.UnitsPerCoin, 1), 1)
	if err == nil || !strings.Contains(err.Error(), "amount below minimum") {
		t.Fatalf("expected below-minimum rejection, got %v", err)
	}
}

func TestDuplicateStakeIDRejected(t *testing.T) {
	state := NewState(testParams())
	if err := state.ApplyLock(stakeLockTx("stake1", stakeOwner, 10*config.UnitsPerCoin, 1), 1); err != nil {
		t.Fatal(err)
	}
	err := state.ApplyLock(stakeLockTx("stake1", stakeOwner, 10*config.UnitsPerCoin, 2), 2)
	if err == nil || !strings.Contains(err.Error(), "duplicate stake id") {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
}

func TestUnlockNonExistingStakeRejected(t *testing.T) {
	err := NewState(testParams()).ApplyUnlock(stakeUnlockTx("unlock1", stakeOwner, "missing", 1), 1)
	if err == nil || !strings.Contains(err.Error(), "stake not found") {
		t.Fatalf("expected missing stake rejection, got %v", err)
	}
}

func TestUnlockOwnerMismatchRejected(t *testing.T) {
	state := NewState(testParams())
	mustApplyLock(t, state, "stake1", stakeOwner, 10*config.UnitsPerCoin, 1)
	err := state.ApplyUnlock(stakeUnlockTx("unlock1", otherOwner, "stake1", 1), 2)
	if err == nil || !strings.Contains(err.Error(), "owner mismatch") {
		t.Fatalf("expected owner mismatch, got %v", err)
	}
}

func TestUnlockAlreadyUnlockingRejected(t *testing.T) {
	state := NewState(testParams())
	mustApplyLock(t, state, "stake1", stakeOwner, 10*config.UnitsPerCoin, 1)
	if err := state.ApplyUnlock(stakeUnlockTx("unlock1", stakeOwner, "stake1", 2), 2); err != nil {
		t.Fatal(err)
	}
	err := state.ApplyUnlock(stakeUnlockTx("unlock2", stakeOwner, "stake1", 3), 3)
	if err == nil || !strings.Contains(err.Error(), "stake unlocking") {
		t.Fatalf("expected already unlocking rejection, got %v", err)
	}
}

func TestReleasedStakeNotLocked(t *testing.T) {
	state := NewState(testParams())
	mustApplyLock(t, state, "stake1", stakeOwner, 10*config.UnitsPerCoin, 1)
	if err := state.ApplyUnlock(stakeUnlockTx("unlock1", stakeOwner, "stake1", 2), 2); err != nil {
		t.Fatal(err)
	}
	active, unlocking, released := state.AddressSummary(stakeOwner, 12)
	if active != 0 || unlocking != 0 || released != 10*config.UnitsPerCoin {
		t.Fatalf("released stake should not be locked active=%d unlocking=%d released=%d", active, unlocking, released)
	}
}

func testParams() config.StakingParams {
	return config.Localnet().Consensus.Staking
}

func mustApplyLock(t *testing.T, state *State, id, owner string, value, nonce uint64) {
	t.Helper()
	if err := state.ApplyLock(stakeLockTx(id, owner, value, nonce), nonce); err != nil {
		t.Fatal(err)
	}
}

func stakeLockTx(id, owner string, value, nonce uint64) types.Transaction {
	tx := types.NewStakeLockTransaction(owner, value, nonce)
	tx.ID = id
	tx.StakeID = id
	return tx
}

func stakeUnlockTx(id, owner, stakeID string, nonce uint64) types.Transaction {
	tx := types.NewStakeUnlockTransaction(owner, stakeID, nonce)
	tx.ID = id
	return tx
}
