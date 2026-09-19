package fees

import (
	"errors"
	"fmt"
	"math"

	"deskachain/internal/config"
	"deskachain/internal/types"
)

// Quote is a deterministic fee estimate for a transaction envelope.
type Quote struct {
	GasUnits     uint64 `json:"gas_units"`
	TxBytes      uint64 `json:"tx_bytes"`
	MinFee       uint64 `json:"min_fee"`
	RequestedFee uint64 `json:"requested_fee"`
	FeeAssetID   string `json:"fee_asset_id"`
	Sufficient   bool   `json:"sufficient"`
}

var (
	ErrFeeTooLow = errors.New("transaction fee below policy minimum")
	ErrGasLimit  = errors.New("transaction exceeds gas limit")
)

func GasUsed(tx types.Transaction, profile config.NetworkConfig) (uint64, uint64, error) {
	if tx.Coinbase {
		return 0, 0, nil
	}
	params := profile.Fee
	if !params.Enabled {
		return 0, 0, nil
	}
	base, err := baseGas(tx, params)
	if err != nil {
		return 0, 0, err
	}
	raw, err := tx.CanonicalSigningBytesWithChainID(profile.ChainID)
	if err != nil {
		return 0, 0, fmt.Errorf("fee estimation encoding failed: %w", err)
	}
	bytesPerGas := params.BytesPerGas
	if bytesPerGas == 0 {
		bytesPerGas = 32
	}
	size := uint64(len(raw))
	sizeGas := size / bytesPerGas
	if size%bytesPerGas != 0 {
		sizeGas++
	}
	if base > math.MaxUint64-sizeGas {
		return 0, 0, errors.New("gas calculation overflow")
	}
	gas := base + sizeGas
	if params.MaxGasPerTx > 0 && gas > params.MaxGasPerTx {
		return 0, size, fmt.Errorf("%w: got %d want <= %d", ErrGasLimit, gas, params.MaxGasPerTx)
	}
	return gas, size, nil
}

func MinimumFee(tx types.Transaction, profile config.NetworkConfig) (uint64, error) {
	if tx.Coinbase || tx.ProtocolVersion() < types.TxVersionAsset || !profile.Fee.Enabled {
		return 0, nil
	}
	gas, _, err := GasUsed(tx, profile)
	if err != nil {
		return 0, err
	}
	price := profile.Fee.MinGasPrice
	if price != 0 && gas > math.MaxUint64/price {
		return 0, errors.New("minimum fee overflow")
	}
	fee := gas * price
	if fee < profile.Fee.MinFee {
		fee = profile.Fee.MinFee
	}
	return fee, nil
}

func Estimate(tx types.Transaction, profile config.NetworkConfig) (Quote, error) {
	gas, size, err := GasUsed(tx, profile)
	if err != nil {
		return Quote{}, err
	}
	minFee, err := MinimumFee(tx, profile)
	if err != nil {
		return Quote{}, err
	}
	return Quote{
		GasUnits:     gas,
		TxBytes:      size,
		MinFee:       minFee,
		RequestedFee: tx.Fee,
		FeeAssetID:   config.FeeAssetID,
		Sufficient:   tx.Fee >= minFee,
	}, nil
}

func Validate(tx types.Transaction, profile config.NetworkConfig) error {
	if tx.Coinbase || tx.ProtocolVersion() < types.TxVersionAsset || !profile.Fee.Enabled {
		return nil
	}
	if _, _, err := GasUsed(tx, profile); err != nil {
		return err
	}
	minFee, err := MinimumFee(tx, profile)
	if err != nil {
		return err
	}
	if tx.Fee < minFee {
		return fmt.Errorf("%w: got %d want >= %d %s", ErrFeeTooLow, tx.Fee, minFee, config.FeeAssetID)
	}
	return nil
}

func baseGas(tx types.Transaction, params config.FeeParams) (uint64, error) {
	switch tx.TxType() {
	case types.TxTypeTransfer:
		if tx.EffectiveAssetID() == config.NativeAssetID {
			return params.BaseGasTransfer, nil
		}
		return params.BaseGasAssetTransfer, nil
	case types.TxTypeStakeLock:
		return params.BaseGasStakeLock, nil
	case types.TxTypeStakeUnlock:
		return params.BaseGasStakeUnlock, nil
	case types.TxTypeAssetCreate:
		return params.BaseGasAssetCreate, nil
	case types.TxTypeAssetMint:
		return params.BaseGasAssetMint, nil
	case types.TxTypeAssetBurn:
		return params.BaseGasAssetBurn, nil
	default:
		return 0, fmt.Errorf("unsupported fee transaction type: %s", tx.TxType())
	}
}
