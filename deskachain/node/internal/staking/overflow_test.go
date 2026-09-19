package staking

import (
	"math"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/types"
)

func TestStakeUnlockRejectsReleaseHeightOverflowWithoutMutation(t *testing.T) {
	state := NewState(config.Localnet().Consensus.Staking)
	state.ApplyRecord(Record{
		StakeID:      "stake-overflow",
		OwnerAddress: stakeOwner,
		Amount:       10 * config.UnitsPerCoin,
		LockTxID:     "lock-tx",
		LockHeight:   1,
		Status:       StatusActive,
	})

	tx := types.NewStakeUnlockTransaction(stakeOwner, "stake-overflow", 1)
	err := state.ApplyUnlock(tx, math.MaxUint64)
	if err == nil || err.Error() != "invalid stake unlock: release height overflow" {
		t.Fatalf("expected release height overflow, got %v", err)
	}

	record, ok := state.Find("stake-overflow", math.MaxUint64)
	if !ok {
		t.Fatal("stake record disappeared after failed unlock")
	}
	if record.Status != StatusActive || record.UnlockTxID != "" || record.UnlockHeight != 0 || record.ReleaseHeight != 0 {
		t.Fatalf("stake record mutated after failed unlock: %+v", record)
	}
}
