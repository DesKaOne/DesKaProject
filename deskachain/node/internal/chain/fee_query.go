package chain

import (
	"errors"
	"fmt"

	"deskachain/internal/asset"
	"deskachain/internal/config"
	"deskachain/internal/fees"
	"deskachain/internal/ledger"
	"deskachain/internal/state"
	"deskachain/internal/storage"
	"deskachain/internal/types"
)

type FeePoolResult struct {
	AssetID string `json:"asset_id"`
	Amount  uint64 `json:"amount"`
	Symbol  string `json:"symbol"`
	Pool    string `json:"pool"`
}

func EstimateFee(tx types.Transaction, profile config.NetworkConfig) (fees.Quote, error) {
	return fees.Estimate(tx, profile)
}

func FeePoolBalanceWithProfile(bc *Blockchain, profile config.NetworkConfig) (FeePoolResult, error) {
	if bc == nil {
		return FeePoolResult{}, errors.New("blockchain is required")
	}
	if profile.Name == "" {
		profile = config.Localnet()
	}
	result := FeePoolResult{
		AssetID: asset.NativeAssetID,
		Symbol:  asset.NativeSymbol,
		Pool:    asset.FeeCollectorAddress,
	}
	tip, err := bc.Tip()
	if err != nil {
		return FeePoolResult{}, err
	}
	if q, ok := bc.store.(storage.AssetStateQueryStore); ok {
		version, height, root, metaErr := q.GetStateMetadata()
		current := metaErr == nil && version == state.SnapshotVersion && height == tip.Height
		if current && tip.ProtocolVersion() == types.BlockVersionCanonical {
			current = tip.StateRoot != "" && tip.StateRoot == root
		}
		if current {
			amount, _, err := q.GetStateAssetBalance(asset.FeeCollectorAddress, asset.NativeAssetID)
			if err != nil {
				return FeePoolResult{}, err
			}
			result.Amount = amount
			return result, nil
		}
	}
	blocks, err := bc.Blocks()
	if err != nil {
		return FeePoolResult{}, err
	}
	l, err := ledger.ReplayMatureWithProfile(blocks, profile.Consensus, profile)
	if err != nil {
		return FeePoolResult{}, fmt.Errorf("fee pool replay failed: %w", err)
	}
	result.Amount = l.FeePoolBalance()
	return result, nil
}
