package state

import (
	"errors"
	"deskachain/internal/arith"
	"fmt"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/staking"
	"deskachain/internal/types"
)

const SnapshotVersion uint8 = 2

var ErrInvalidSnapshot = errors.New("invalid state snapshot")

// Snapshot is the persisted deterministic state at one canonical chain tip.
type Snapshot struct {
	Version   uint8                 `json:"version"`
	Height    uint64                `json:"height"`
	StateRoot string                `json:"state_root"`
	Accounts  []ledger.StateAccount `json:"accounts"`
	Stakes    []staking.Record      `json:"stakes"`
	Coinbases []ledger.StateCoinbase `json:"coinbases"`
}

func SnapshotForLedger(l *ledger.MatureLedger) (Snapshot, error) {
	if l == nil {
		return Snapshot{}, ErrNilLedger
	}
	accounts, stakes := StableDigestInputs(l.StateAccounts(), l.StateStakes())
	snapshot := Snapshot{
		Version:  SnapshotVersion,
		Height:   l.Height(),
		Accounts: accounts,
		Stakes:    stakes,
	}
	root, err := RootForCollections(accounts, stakes)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.StateRoot = root
	return snapshot, nil
}

func SnapshotForBlocks(blocks []types.Block, params config.ConsensusParams, profile config.NetworkConfig) (Snapshot, error) {
	l, err := ledger.ReplayMatureWithProfile(blocks, params, profile)
	if err != nil {
		return Snapshot{}, err
	}
	return SnapshotForLedger(l)
}

func SnapshotAfterBlock(snapshot Snapshot, block types.Block, params config.ConsensusParams, profile config.NetworkConfig) (Snapshot, error) {
	expectedHeight, err := arith.Add(snapshot.Height, 1)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: state height overflow", ErrInvalidSnapshot)
	}
	if block.Height != expectedHeight {
		return Snapshot{}, fmt.Errorf("%w: block height %d does not follow state height %d", ErrInvalidSnapshot, block.Height, snapshot.Height)
	}
	l := ledger.NewMatureFromState(
		params,
		profile,
		snapshot.Height,
		snapshot.Accounts,
		snapshot.Stakes,
		snapshot.Coinbases,
	)
	if err := l.ApplyBlock(block); err != nil {
		return Snapshot{}, err
	}
	return SnapshotForLedger(l)
}

func (s Snapshot) Validate() error {
	if s.Version != SnapshotVersion {
		return fmt.Errorf("%w: unsupported version %d", ErrInvalidSnapshot, s.Version)
	}
	if s.StateRoot == "" {
		return fmt.Errorf("%w: empty state root", ErrInvalidSnapshot)
	}
	accounts, stakes := StableDigestInputs(s.Accounts, s.Stakes)
	for i := 1; i < len(accounts); i++ {
		if accounts[i-1].Address == accounts[i].Address {
			return fmt.Errorf("%w: duplicate account %q", ErrInvalidSnapshot, accounts[i].Address)
		}
	}
	for i := 1; i < len(stakes); i++ {
		if stakes[i-1].StakeID == stakes[i].StakeID {
			return fmt.Errorf("%w: duplicate stake %q", ErrInvalidSnapshot, stakes[i].StakeID)
		}
	}
	root, err := RootForCollections(accounts, stakes)
	if err != nil {
		return err
	}
	if root != s.StateRoot {
		return fmt.Errorf("%w: state root mismatch", ErrInvalidSnapshot)
	}
	return nil
}

func (s Snapshot) Account(address string) (ledger.StateAccount, bool) {
	for _, account := range s.Accounts {
		if account.Address == address {
			return account, true
		}
	}
	return ledger.StateAccount{}, false
}

func (s Snapshot) Stake(stakeID string) (staking.Record, bool) {
	for _, record := range s.Stakes {
		if record.StakeID == stakeID {
			return record, true
		}
	}
	return staking.Record{}, false
}
