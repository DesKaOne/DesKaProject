package state

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

	"deskachain/internal/arith"

	"deskachain/internal/asset"
	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/staking"
	"deskachain/internal/types"
)

const SnapshotVersion uint8 = 3

var ErrInvalidSnapshot = errors.New("invalid state snapshot")

// Snapshot is the persisted deterministic state at one canonical chain tip.
type Snapshot struct {
	Version       uint8                  `json:"version"`
	Height        uint64                 `json:"height"`
	StateRoot     string                 `json:"state_root"`
	Accounts      []ledger.StateAccount  `json:"accounts"`
	Stakes        []staking.Record       `json:"stakes"`
	Coinbases     []ledger.StateCoinbase `json:"coinbases"`
	Assets        []asset.Definition     `json:"assets,omitempty"`
	AssetBalances []asset.BalanceEntry   `json:"asset_balances,omitempty"`
}

func SnapshotForLedger(l *ledger.MatureLedger) (Snapshot, error) {
	if l == nil {
		return Snapshot{}, ErrNilLedger
	}
	accounts, stakes := StableDigestInputs(l.StateAccounts(), l.StateStakes())
	snapshot := Snapshot{
		Version:        SnapshotVersion,
		Height:         l.Height(),
		Accounts:       accounts,
		Stakes:         stakes,
		Coinbases:      l.StateCoinbases(),
		Assets:         l.StateAssetDefinitions(),
		AssetBalances:  l.StateAssetBalances(),
	}
	var root string
	var err error
	if len(snapshot.Assets) > 0 {
		root, err = RootForCollectionsWithAssets(accounts, stakes, snapshot.Assets, snapshot.AssetBalances)
	} else {
		root, err = RootForCollections(accounts, stakes)
	}
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
	if err := snapshot.Validate(); err != nil {
		return Snapshot{}, err
	}
	expectedHeight, err := arith.Add(snapshot.Height, 1)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: state height overflow", ErrInvalidSnapshot)
	}
	if block.Height != expectedHeight {
		return Snapshot{}, fmt.Errorf("%w: block height %d does not follow state height %d", ErrInvalidSnapshot, block.Height, snapshot.Height)
	}
	l, err := ledger.NewMatureFromStateWithAssets(
		params,
		profile,
		snapshot.Height,
		snapshot.Accounts,
		snapshot.Stakes,
		snapshot.Coinbases,
		snapshot.Assets,
		snapshot.AssetBalances,
	)
	if err != nil {
		return Snapshot{}, err
	}
	if err := l.ApplyBlock(block); err != nil {
		return Snapshot{}, err
	}
	return SnapshotForLedger(l)
}

func Equivalent(a, b Snapshot) bool {
	if a.Version != b.Version || a.Height != b.Height || a.StateRoot != b.StateRoot {
		return false
	}

	aAccounts, aStakes := StableDigestInputs(a.Accounts, a.Stakes)
	aAssets := append([]asset.Definition(nil), a.Assets...)
	bAssets := append([]asset.Definition(nil), b.Assets...)
	aBalances := append([]asset.BalanceEntry(nil), a.AssetBalances...)
	bBalances := append([]asset.BalanceEntry(nil), b.AssetBalances...)
	sort.Slice(aAssets, func(i, j int) bool { return aAssets[i].ID < aAssets[j].ID })
	sort.Slice(bAssets, func(i, j int) bool { return bAssets[i].ID < bAssets[j].ID })
	sort.Slice(aBalances, func(i, j int) bool {
		if aBalances[i].Address != aBalances[j].Address { return aBalances[i].Address < aBalances[j].Address }
		return aBalances[i].AssetID < aBalances[j].AssetID
	})
	sort.Slice(bBalances, func(i, j int) bool {
		if bBalances[i].Address != bBalances[j].Address { return bBalances[i].Address < bBalances[j].Address }
		return bBalances[i].AssetID < bBalances[j].AssetID
	})
	bAccounts, bStakes := StableDigestInputs(b.Accounts, b.Stakes)
	a.Coinbases = append([]ledger.StateCoinbase(nil), a.Coinbases...)
	b.Coinbases = append([]ledger.StateCoinbase(nil), b.Coinbases...)
	sort.Slice(a.Coinbases, func(i, j int) bool {
		if a.Coinbases[i].Height != a.Coinbases[j].Height {
			return a.Coinbases[i].Height < a.Coinbases[j].Height
		}
		if a.Coinbases[i].Address != a.Coinbases[j].Address {
			return a.Coinbases[i].Address < a.Coinbases[j].Address
		}
		return a.Coinbases[i].Amount < a.Coinbases[j].Amount
	})
	sort.Slice(b.Coinbases, func(i, j int) bool {
		if b.Coinbases[i].Height != b.Coinbases[j].Height {
			return b.Coinbases[i].Height < b.Coinbases[j].Height
		}
		if b.Coinbases[i].Address != b.Coinbases[j].Address {
			return b.Coinbases[i].Address < b.Coinbases[j].Address
		}
		return b.Coinbases[i].Amount < b.Coinbases[j].Amount
	})
	a.Accounts, a.Stakes = aAccounts, aStakes
	b.Accounts, b.Stakes = bAccounts, bStakes
	a.Assets, a.AssetBalances = aAssets, aBalances
	b.Assets, b.AssetBalances = bAssets, bBalances
	return reflect.DeepEqual(a, b)
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
	coinbases := append([]ledger.StateCoinbase(nil), s.Coinbases...)
	sort.Slice(coinbases, func(i, j int) bool {
		if coinbases[i].Height != coinbases[j].Height {
			return coinbases[i].Height < coinbases[j].Height
		}
		if coinbases[i].Address != coinbases[j].Address {
			return coinbases[i].Address < coinbases[j].Address
		}
		return coinbases[i].Amount < coinbases[j].Amount
	})
	for i := 1; i < len(coinbases); i++ {
		if coinbases[i-1].Height == coinbases[i].Height &&
			coinbases[i-1].Address == coinbases[i].Address {
			return fmt.Errorf("%w: duplicate pending coinbase %d/%q", ErrInvalidSnapshot, coinbases[i].Height, coinbases[i].Address)
		}
	}
	for _, coinbase := range coinbases {
		if coinbase.Address == "" || coinbase.Amount == 0 {
			return fmt.Errorf("%w: invalid pending coinbase", ErrInvalidSnapshot)
		}
		if coinbase.Height > s.Height {
			return fmt.Errorf("%w: pending coinbase height %d exceeds snapshot height %d", ErrInvalidSnapshot, coinbase.Height, s.Height)
		}
	}
	nativeSeen := false
	for _, def := range s.Assets {
		if asset.IsNative(def.ID) {
			if nativeSeen || def.ID != asset.NativeAssetID || def.Symbol != asset.NativeSymbol || def.Decimals != asset.NativeDecimals {
				return fmt.Errorf("%w: invalid native IDR definition", ErrInvalidSnapshot)
			}
			nativeSeen = true
			continue
		}
		if err := asset.ValidateDefinition(def); err != nil {
			return fmt.Errorf("%w: invalid asset %q: %v", ErrInvalidSnapshot, def.ID, err)
		}
	}
	if len(s.Assets) > 0 && !nativeSeen {
		return fmt.Errorf("%w: missing native IDR definition", ErrInvalidSnapshot)
	}
	assets := append([]asset.Definition(nil), s.Assets...)
	balances := append([]asset.BalanceEntry(nil), s.AssetBalances...)
	sort.Slice(assets, func(i, j int) bool { return assets[i].ID < assets[j].ID })
	sort.Slice(balances, func(i, j int) bool {
		if balances[i].Address != balances[j].Address { return balances[i].Address < balances[j].Address }
		return balances[i].AssetID < balances[j].AssetID
	})
	for i := 1; i < len(assets); i++ {
		if assets[i-1].ID == assets[i].ID { return fmt.Errorf("%w: duplicate asset %q", ErrInvalidSnapshot, assets[i].ID) }
	}
	known := make(map[string]struct{}, len(assets))
	for _, def := range assets { known[def.ID] = struct{}{} }
	for i := 1; i < len(balances); i++ {
		if balances[i-1].Address == balances[i].Address && balances[i-1].AssetID == balances[i].AssetID {
			return fmt.Errorf("%w: duplicate asset balance %s/%s", ErrInvalidSnapshot, balances[i].Address, balances[i].AssetID)
		}
	}
	for _, entry := range balances {
		if entry.Address == "" || entry.AssetID == "" || entry.Amount == 0 { return fmt.Errorf("%w: invalid asset balance", ErrInvalidSnapshot) }
		if _, ok := known[entry.AssetID]; !ok { return fmt.Errorf("%w: unknown asset %q", ErrInvalidSnapshot, entry.AssetID) }
	}
	root, err := RootForCollections(accounts, stakes)
	if len(assets) > 0 {
		root, err = RootForCollectionsWithAssets(accounts, stakes, assets, balances)
	}
	if err != nil { return err }
	if root != s.StateRoot { return fmt.Errorf("%w: state root mismatch", ErrInvalidSnapshot) }
	if len(assets) > 0 {
		idrs := make(map[string]uint64)
		for _, entry := range balances {
			if asset.IsNative(entry.AssetID) { idrs[entry.Address] = entry.Amount }
		}
		for _, account := range accounts {
			if got := idrs[account.Address]; got != account.Confirmed {
				return fmt.Errorf("%w: native IDR/account mismatch for %q", ErrInvalidSnapshot, account.Address)
			}
		}
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
