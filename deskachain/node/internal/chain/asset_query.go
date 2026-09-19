package chain

import (
	"deskachain/internal/asset"
	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/state"
	"deskachain/internal/storage"
	"deskachain/internal/types"
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
			return q.GetStateAsset(assetID)
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

