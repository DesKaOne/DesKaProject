package chain

import (
	"indochain/internal/asset"
	"indochain/internal/config"
	"indochain/internal/ledger"
	"indochain/internal/state"
	"indochain/internal/storage"
	"indochain/internal/types"
)

func (bc *Blockchain) AssetBalanceWithProfile(address, assetID string, profile config.NetworkConfig) (uint64, error) {
	tip, err := bc.Tip()
	if err != nil {
		return 0, err
	}
	if profile.Name == "" {
		profile = config.Localnet()
	}
	if q, ok := bc.store.(storage.AssetStateQueryStore); ok {
		version, height, root, metaErr := q.GetStateMetadata()
		current := metaErr == nil && version == state.SnapshotVersion && height == tip.Height
		if current && tip.ProtocolVersion() == types.BlockVersionCanonical {
			current = tip.StateRoot != "" && tip.StateRoot == root
		}
		if current {
			amount, _, err := q.GetStateAssetBalance(address, assetID)
			return amount, err
		}
	}
	blocks, err := bc.Blocks()
	if err != nil {
		return 0, err
	}
	l, err := ledger.ReplayMatureWithProfile(blocks, profile.Consensus, profile)
	if err != nil {
		return 0, err
	}
	return l.AssetBalance(address, assetID), nil
}

func (bc *Blockchain) AssetDefinitionWithProfile(assetID string, profile config.NetworkConfig) (asset.Definition, bool, error) {
	tip, err := bc.Tip()
	if err != nil {
		return asset.Definition{}, false, err
	}
	if profile.Name == "" {
		profile = config.Localnet()
	}
	if q, ok := bc.store.(storage.AssetStateQueryStore); ok {
		version, height, root, metaErr := q.GetStateMetadata()
		current := metaErr == nil && version == state.SnapshotVersion && height == tip.Height
		if current && tip.ProtocolVersion() == types.BlockVersionCanonical {
			current = tip.StateRoot != "" && tip.StateRoot == root
		}
		if current {
			def, found, err := q.GetStateAsset(assetID)
			if err != nil {
				return asset.Definition{}, false, err
			}
			if found {
				return def, true, nil
			}
			if asset.IsNative(assetID) {
				return asset.Definition{
					ID:       asset.NativeAssetID,
					Name:     asset.NativeSymbol,
					Symbol:   asset.NativeSymbol,
					Decimals: asset.NativeDecimals,
					Kind:     asset.KindFungible,
					Issuer:   "protocol",
					Status:   asset.StatusActive,
				}, true, nil
			}
			return asset.Definition{}, false, nil
		}
	}
	blocks, err := bc.Blocks()
	if err != nil {
		return asset.Definition{}, false, err
	}
	l, err := ledger.ReplayMatureWithProfile(blocks, profile.Consensus, profile)
	if err != nil {
		return asset.Definition{}, false, err
	}
	def, ok := l.AssetDefinition(assetID)
	return def, ok, nil
}

func (bc *Blockchain) AssetBalancesForAddressWithProfile(address string, profile config.NetworkConfig) ([]asset.BalanceEntry, error) {
	tip, err := bc.Tip()
	if err != nil {
		return nil, err
	}
	if profile.Name == "" {
		profile = config.Localnet()
	}
	if q, ok := bc.store.(storage.AssetStateQueryStore); ok {
		version, height, root, metaErr := q.GetStateMetadata()
		current := metaErr == nil && version == state.SnapshotVersion && height == tip.Height
		if current && tip.ProtocolVersion() == types.BlockVersionCanonical {
			current = tip.StateRoot != "" && tip.StateRoot == root
		}
		if current {
			return q.GetStateAssetBalancesForAddress(address)
		}
	}
	blocks, err := bc.Blocks()
	if err != nil {
		return nil, err
	}
	l, err := ledger.ReplayMatureWithProfile(blocks, profile.Consensus, profile)
	if err != nil {
		return nil, err
	}
	entries := make([]asset.BalanceEntry, 0)
	for _, entry := range l.AssetState().Balances() {
		if entry.Address == address {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}
