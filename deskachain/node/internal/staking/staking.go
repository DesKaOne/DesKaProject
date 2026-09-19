package staking

import (
	"errors"
	"fmt"

	"deskachain/internal/arith"\n\t"deskachain/internal/config"
	"deskachain/internal/types"
)

const (
	StatusActive    = "active"
	StatusUnlocking = "unlocking"
	StatusReleased  = "released"
)

type Record struct {
	StakeID       string `json:"stake_id"`
	OwnerAddress  string `json:"owner_address"`
	Amount        uint64 `json:"amount"`
	LockTxID      string `json:"lock_tx_id"`
	LockHeight    uint64 `json:"lock_height"`
	UnlockTxID    string `json:"unlock_tx_id,omitempty"`
	UnlockHeight  uint64 `json:"unlock_height,omitempty"`
	ReleaseHeight uint64 `json:"release_height,omitempty"`
	Status        string `json:"status"`
}

type State struct {
	params  config.StakingParams
	records map[string]Record
}

type Summary struct {
	TotalActiveStake    uint64
	TotalUnlockingStake uint64
	TotalReleasedStake  uint64
	ActiveStakeCount    int
}

func NewState(params config.StakingParams) *State {
	return &State{params: params, records: make(map[string]Record)}
}

func Replay(blocks []types.Block, params config.StakingParams) (*State, error) {
	state := NewState(params)
	for _, block := range blocks {
		if err := state.ApplyBlock(block); err != nil {
			return nil, err
		}
	}
	if len(blocks) > 0 {
		state.RefreshReleased(blocks[len(blocks)-1].Height)
	}
	return state, nil
}

func (s *State) ApplyBlock(block types.Block) error {
	s.RefreshReleased(block.Height)
	for _, tx := range block.Transactions {
		switch tx.TxType() {
		case types.TxTypeStakeLock:
			if err := s.ApplyLock(tx, block.Height); err != nil {
				return err
			}
		case types.TxTypeStakeUnlock:
			if err := s.ApplyUnlock(tx, block.Height); err != nil {
				return err
			}
		}
	}
	s.RefreshReleased(block.Height)
	return nil
}

func (s *State) ApplyLock(tx types.Transaction, height uint64) error {
	if !s.params.Enabled {
		return errors.New("staking disabled")
	}
	if tx.Amount < s.params.MinStakeAmount {
		return errors.New("invalid stake lock: amount below minimum")
	}
	stakeID := tx.StakeID
	if stakeID == "" {
		stakeID = tx.ID
	}
	if _, ok := s.records[stakeID]; ok {
		return errors.New("invalid stake lock: duplicate stake id")
	}
	if s.ActiveCount(tx.From) >= s.params.MaxActiveStakesPerAddress && s.params.MaxActiveStakesPerAddress > 0 {
		return errors.New("invalid stake lock: max active stakes reached")
	}
	s.records[stakeID] = Record{
		StakeID:      stakeID,
		OwnerAddress: tx.From,
		Amount:       tx.Amount,
		LockTxID:     tx.ID,
		LockHeight:   height,
		Status:       StatusActive,
	}
	return nil
}

func (s *State) ApplyUnlock(tx types.Transaction, height uint64) error {
	if !s.params.Enabled {
		return errors.New("staking disabled")
	}
	record, ok := s.records[tx.StakeID]
	if !ok {
		return errors.New("invalid stake unlock: stake not found")
	}
	if record.OwnerAddress != tx.From {
		return errors.New("invalid stake unlock: owner mismatch")
	}
	if record.Status != StatusActive {
		return fmt.Errorf("invalid stake unlock: stake %s", record.Status)
	}
	record.Status = StatusUnlocking
	record.UnlockTxID = tx.ID
	record.UnlockHeight = height
	releaseHeight, err := arith.Add(height, s.params.UnbondingPeriodBlocks)\n\tif err != nil { return errors.New("invalid stake unlock: release height overflow") }\n\trecord.ReleaseHeight = releaseHeight
	s.records[tx.StakeID] = record
	return nil
}

func (s *State) ApplyRecord(record Record) {
	s.records[record.StakeID] = record
}

func (s *State) RefreshReleased(currentHeight uint64) {
	for id, record := range s.records {
		if record.Status == StatusUnlocking && currentHeight >= record.ReleaseHeight {
			record.Status = StatusReleased
			s.records[id] = record
		}
	}
}

func (s *State) Records(currentHeight uint64) []Record {
	s.RefreshReleased(currentHeight)
	out := make([]Record, 0, len(s.records))
	for _, record := range s.records {
		out = append(out, record)
	}
	return out
}

func (s *State) Find(id string, currentHeight uint64) (Record, bool) {
	s.RefreshReleased(currentHeight)
	record, ok := s.records[id]
	return record, ok
}

func (s *State) AddressSummary(address string, currentHeight uint64) (active, unlocking, released uint64) {
	s.RefreshReleased(currentHeight)
	for _, record := range s.records {
		if record.OwnerAddress != address {
			continue
		}
		switch record.Status {
		case StatusActive:
			active = arith.AddCap(active, record.Amount)
		case StatusUnlocking:
			unlocking = arith.AddCap(unlocking, record.Amount)
		case StatusReleased:
			released = arith.AddCap(released, record.Amount)
		}
	}
	return active, unlocking, released
}

func (s *State) ActiveCount(address string) int {
	count := 0
	for _, record := range s.records {
		if record.OwnerAddress == address && record.Status == StatusActive {
			count++
		}
	}
	return count
}

func (s *State) Summary(currentHeight uint64) Summary {
	s.RefreshReleased(currentHeight)
	var summary Summary
	for _, record := range s.records {
		switch record.Status {
		case StatusActive:
			summary.TotalActiveStake = arith.AddCap(summary.TotalActiveStake, record.Amount)
			summary.ActiveStakeCount++
		case StatusUnlocking:
			summary.TotalUnlockingStake = arith.AddCap(summary.TotalUnlockingStake, record.Amount)
		case StatusReleased:
			summary.TotalReleasedStake = arith.AddCap(summary.TotalReleasedStake, record.Amount)
		}
	}
	return summary
}

func PendingStakeLock(txs []types.Transaction, address string) uint64 {
	var total uint64
	for _, tx := range txs {
		if tx.TxType() == types.TxTypeStakeLock && tx.From == address {
			total = arith.AddCap(total, tx.Amount)
		}
	}
	return total
}

func PendingUnlockIDs(txs []types.Transaction, address string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, tx := range txs {
		if tx.TxType() == types.TxTypeStakeUnlock && tx.From == address {
			out[tx.StakeID] = struct{}{}
		}
	}
	return out
}
