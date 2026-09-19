package mempool

import (
	"fmt"
	"math/big"
	"sort"

	"deskachain/internal/config"
	"deskachain/internal/fees"
	"deskachain/internal/types"
)

// AdmissionPolicy applies deterministic consensus fee/resource rules to
// transactions entering the mempool.
type AdmissionPolicy struct {
	Profile config.NetworkConfig
	MaxTxs  uint64
	MaxGas  uint64
}

// Admit validates a transaction against the active fee policy and mempool
// capacity. It deliberately does not replace an existing nonce; replacement
// policy is reserved for a later protocol decision.
func ValidateForRevalidation(tx types.Transaction, policy AdmissionPolicy) error {
	if tx.Coinbase {
		return fmt.Errorf("coinbase transactions are not accepted by the mempool")
	}
	if err := types.ValidateTransactionVersion(tx.ProtocolVersion(), policy.Profile.TxVersion); err != nil {
		return err
	}
	if tx.ID == "" {
		return fmt.Errorf("transaction id is required")
	}
	if tx.From == "" {
		return fmt.Errorf("transaction sender is required")
	}
	if err := fees.Validate(tx, policy.Profile); err != nil {
		return err
	}
	if tx.ProtocolVersion() == types.TxVersionAsset {
		if err := tx.ValidateAssetEnvelope(); err != nil {
			return err
		}
		if err := tx.ValidateFeePayerAuthorization(policy.Profile); err != nil {
			return err
		}
	}
	if _, _, err := fees.GasUsed(tx, policy.Profile); err != nil {
		return err
	}
	return nil
}

func (m Mempool) Admit(tx types.Transaction, policy AdmissionPolicy) error {
	if err := ValidateForRevalidation(tx, policy); err != nil {
		return err
	}
	txs, err := m.Load()
	if err != nil {
		return err
	}
	for _, existing := range txs {
		if existing.ID == tx.ID {
			return ErrDuplicateTx
		}
		if existing.From == tx.From && existing.Nonce == tx.Nonce {
			return fmt.Errorf("transaction nonce already pending: %s/%d", tx.From, tx.Nonce)
		}
	}
	if policy.MaxTxs > 0 && uint64(len(txs)) >= policy.MaxTxs {
		return fmt.Errorf("mempool transaction limit reached: got %d want < %d", len(txs), policy.MaxTxs)
	}
	gas, _, err := fees.GasUsed(tx, policy.Profile)
	if err != nil {
		return err
	}
	var totalGas uint64
	for _, existing := range txs {
		existingGas, _, gasErr := fees.GasUsed(existing, policy.Profile)
		if gasErr != nil {
			return gasErr
		}
		if ^uint64(0)-totalGas < existingGas {
			return fmt.Errorf("mempool gas total overflow")
		}
		totalGas += existingGas
	}
	if ^uint64(0)-totalGas < gas {
		return fmt.Errorf("mempool gas total overflow")
	}
	totalGas += gas
	if policy.MaxGas > 0 && totalGas > policy.MaxGas {
		return fmt.Errorf("mempool gas limit exceeded: got %d want <= %d", totalGas, policy.MaxGas)
	}
	return m.Add(tx)
}

// Select returns a deterministic subset that fits the configured gas limit.
// Higher fee-per-gas transactions are selected first; ties use higher fee,
// lower gas, then transaction ID.
func Select(txs []types.Transaction, profile config.NetworkConfig, maxGas uint64) ([]types.Transaction, error) {
	type candidate struct {
		tx  types.Transaction
		gas uint64
	}
	candidates := make([]candidate, 0, len(txs))
	for _, tx := range txs {
		if tx.Coinbase {
			continue
		}
		if err := ValidateForRevalidation(tx, AdmissionPolicy{Profile: profile}); err != nil {
			return nil, err
		}
		gas, _, err := fees.GasUsed(tx, profile)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate{tx: tx, gas: gas})
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := new(big.Int).SetUint64(candidates[i].tx.Fee)
		left.Mul(left, new(big.Int).SetUint64(candidates[j].gas))
		right := new(big.Int).SetUint64(candidates[j].tx.Fee)
		right.Mul(right, new(big.Int).SetUint64(candidates[i].gas))
		if cmp := left.Cmp(right); cmp != 0 {
			return cmp > 0
		}
		if candidates[i].tx.Fee != candidates[j].tx.Fee {
			return candidates[i].tx.Fee > candidates[j].tx.Fee
		}
		if candidates[i].gas != candidates[j].gas {
			return candidates[i].gas < candidates[j].gas
		}
		return candidates[i].tx.ID < candidates[j].tx.ID
	})
	out := make([]types.Transaction, 0, len(candidates))
	var used uint64
	for _, c := range candidates {
		if maxGas > 0 && (c.gas > maxGas || used > maxGas-c.gas) {
			continue
		}
		used += c.gas
		out = append(out, c.tx)
	}
	return out, nil
}
