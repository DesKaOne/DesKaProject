package chain

import (
	"encoding/json"
	"errors"
	"fmt"

	"indochain/internal/config"
	"indochain/internal/fees"
	"indochain/internal/types"
)

func ValidateTransactionSize(tx types.Transaction, consensus config.ConsensusParams) error {
	maxTxBytes := consensus.MaxTxBytes
	if maxTxBytes == 0 {
		maxTxBytes = config.DefaultMaxTxBytes
	}
	raw, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to encode transaction: %w", err)
	}
	if uint64(len(raw)) > maxTxBytes {
		return fmt.Errorf("transaction exceeds max size: got %d bytes want <= %d", len(raw), maxTxBytes)
	}
	return nil
}

func ValidateBlockResources(block types.Block, consensus config.ConsensusParams) error {
	return validateBlockResources(block, consensus, config.Localnet())
}

func ValidateBlockResourcesWithProfile(block types.Block, profile config.NetworkConfig) error {
	return validateBlockResources(block, profile.Consensus, profile)
}

func validateBlockResources(block types.Block, consensus config.ConsensusParams, profile config.NetworkConfig) error {
	maxTxCount := consensus.MaxTxCount
	if maxTxCount == 0 {
		maxTxCount = config.DefaultMaxTxCount
	}
	if uint64(len(block.Transactions)) > maxTxCount {
		return fmt.Errorf("block exceeds max transaction count: got %d want <= %d", len(block.Transactions), maxTxCount)
	}

	var totalGas uint64
	for i, tx := range block.Transactions {
		if err := ValidateTransactionSize(tx, consensus); err != nil {
			return fmt.Errorf("tx %d %w", i, err)
		}
		gas, _, err := fees.GasUsed(tx, profile)
		if err != nil {
			return fmt.Errorf("tx %d gas validation failed: %w", i, err)
		}
		if ^uint64(0)-totalGas < gas {
			return errors.New("block gas calculation overflow")
		}
		totalGas += gas
	}
	if maxGas := consensus.MaxGasPerBlock; maxGas > 0 && totalGas > maxGas {
		return fmt.Errorf("block exceeds max gas: got %d want <= %d", totalGas, maxGas)
	}

	maxBlockBytes := consensus.MaxBlockBytes
	if maxBlockBytes == 0 {
		maxBlockBytes = config.DefaultMaxBlockBytes
	}
	raw, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to encode block: %w", err)
	}
	if uint64(len(raw)) > maxBlockBytes {
		return fmt.Errorf("block exceeds max size: got %d bytes want <= %d", len(raw), maxBlockBytes)
	}
	return nil
}
