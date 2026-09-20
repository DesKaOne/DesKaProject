package chain

import (
	"errors"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/staking"
	"deskachain/internal/state"
	"deskachain/internal/types"
)

var errReplayMustNotRun = errors.New("block replay must not run")

type queryOnlyStore struct {
	tip     types.Block
	account ledger.StateAccount
	stakes  []staking.Record
}

func (s *queryOnlyStore) Init() error                                   { return nil }
func (s *queryOnlyStore) Close() error                                  { return nil }
func (s *queryOnlyStore) HasChain() (bool, error)                       { return true, nil }
func (s *queryOnlyStore) SaveBlock(types.Block) error                   { return nil }
func (s *queryOnlyStore) Blocks() ([]types.Block, error)                { return nil, errReplayMustNotRun }
func (s *queryOnlyStore) Tip() (types.Block, error)                     { return s.tip, nil }
func (s *queryOnlyStore) GetBlockByHeight(uint64) (types.Block, error)  { return s.tip, nil }
func (s *queryOnlyStore) DeleteBlockByHeight(uint64) error              { return nil }
func (s *queryOnlyStore) ReplaceFromHeight(uint64, []types.Block) error { return nil }
func (s *queryOnlyStore) SetTip(uint64, string) error                   { return nil }
func (s *queryOnlyStore) GetHeight() (uint64, error)                    { return s.tip.Height, nil }
func (s *queryOnlyStore) GetTip() (uint64, string, error)               { return s.tip.Height, s.tip.Hash, nil }

func (s *queryOnlyStore) SaveState(state.Snapshot) error     { return nil }
func (s *queryOnlyStore) LoadState() (state.Snapshot, error) { return state.Snapshot{}, nil }
func (s *queryOnlyStore) DeleteState() error                 { return nil }
func (s *queryOnlyStore) GetStateMetadata() (uint8, uint64, string, error) {
	return state.SnapshotVersion, s.tip.Height, "legacy-state-root", nil
}
func (s *queryOnlyStore) GetStateAccount(string) (ledger.StateAccount, bool, error) {
	return s.account, true, nil
}
func (s *queryOnlyStore) GetStateStake(string) (staking.Record, bool, error) {
	return staking.Record{}, false, nil
}
func (s *queryOnlyStore) GetStateStakesForAddress(string) ([]staking.Record, error) {
	return s.stakes, nil
}

func (s *queryOnlyStore) ValidateStateIndexes() error { return nil }

func TestBalanceDetailsPrefersCurrentStateIndexOverReplay(t *testing.T) {
	store := &queryOnlyStore{
		tip: types.Block{Height: 5, Hash: "tip", StateRoot: "legacy-state-root"},
		account: ledger.StateAccount{
			Address:   "IDR-alice",
			Confirmed: 100,
			Mature:    80,
			Nonce:     9,
		},
		stakes: []staking.Record{
			{
				StakeID:      "stake-1",
				OwnerAddress: "IDR-alice",
				Amount:       20,
				Status:       staking.StatusActive,
			},
		},
	}
	bc := New(store)

	got, err := bc.BalanceDetailsForWithProfile("IDR-alice", nil, config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	if got.Confirmed != 100 || got.Mature != 80 || got.ActiveStake != 20 || got.Spendable != 60 {
		t.Fatalf("unexpected state-backed balance: %#v", got)
	}
}

func TestAccountNoncePrefersCurrentStateIndexOverReplay(t *testing.T) {
	store := &queryOnlyStore{
		tip:     types.Block{Height: 5, Hash: "tip", StateRoot: "legacy-state-root"},
		account: ledger.StateAccount{Address: "IDR-alice", Nonce: 11},
	}
	bc := New(store)

	got, err := bc.AccountNonceWithProfile("IDR-alice", config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	if got != 11 {
		t.Fatalf("nonce = %d, want 11", got)
	}
}
