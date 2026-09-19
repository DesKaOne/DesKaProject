package mempool

import (
	"math/big"
	"sort"

	"deskachain/internal/config"
	"deskachain/internal/fees"
	"deskachain/internal/types"
)

// SelectWithNonces deterministically selects transactions while preserving
// per-sender nonce order. nextNonce contains the next executable nonce for
// each sender according to the current chain state. A sender's transaction
// with a nonce above that value is held back until its predecessor is present.
func SelectWithNonces(txs []types.Transaction, profile config.NetworkConfig, maxGas uint64, nextNonce map[string]uint64) ([]types.Transaction, error) {
	type candidate struct {
		tx  types.Transaction
		gas uint64
	}
	bySender := make(map[string][]candidate)
	for _, tx := range txs {
		if tx.Coinbase {
			continue
		}
		if err := types.ValidateTransactionVersion(tx.ProtocolVersion(), profile.TxVersion); err != nil {
			return nil, err
		}
		if err := fees.Validate(tx, profile); err != nil {
			return nil, err
		}
		if tx.ProtocolVersion() == types.TxVersionAsset {
			if err := tx.ValidateAssetEnvelope(); err != nil {
				return nil, err
			}
			if err := tx.ValidateFeePayerAuthorization(profile); err != nil {
				return nil, err
			}
		}
		gas, _, err := fees.GasUsed(tx, profile)
		if err != nil {
			return nil, err
		}
		bySender[tx.From] = append(bySender[tx.From], candidate{tx: tx, gas: gas})
	}

	for sender, list := range bySender {
		sort.Slice(list, func(i, j int) bool {
			if list[i].tx.Nonce != list[j].tx.Nonce {
				return list[i].tx.Nonce < list[j].tx.Nonce
			}
			return list[i].tx.ID < list[j].tx.ID
		})
		start, ok := nextNonce[sender]
		if !ok {
			start = list[0].tx.Nonce
		}
		ready := make([]candidate, 0, len(list))
		expected := start
		for _, c := range list {
			if c.tx.Nonce < expected {
				continue
			}
			if c.tx.Nonce != expected {
				break
			}
			ready = append(ready, c)
			if expected == ^uint64(0) {
				break
			}
			expected++
		}
		bySender[sender] = ready
	}

	// Only the head transaction for each sender competes for block space.
	// After selecting one, the next nonce from that sender becomes eligible.
	out := make([]types.Transaction, 0, len(txs))
	var used uint64
	for {
		bestSender := ""
		best := candidate{}
		found := false
		for sender, list := range bySender {
			if len(list) == 0 {
				continue
			}
			c := list[0]
			if maxGas > 0 && (c.gas > maxGas || used > maxGas-c.gas) {
				continue
			}
			if !found || higherPriority(c, best) || (samePriority(c, best) && (sender < best.tx.From || (sender == best.tx.From && c.tx.ID < best.tx.ID))) {
				bestSender, best, found = sender, c, true
			}
		}
		if !found {
			break
		}
		out = append(out, best.tx)
		used += best.gas
		bySender[bestSender] = bySender[bestSender][1:]
	}
	return out, nil
}

func higherPriority(a, b struct{ tx types.Transaction; gas uint64 }) bool {
	left := new(big.Int).SetUint64(a.tx.Fee)
	left.Mul(left, new(big.Int).SetUint64(b.gas))
	right := new(big.Int).SetUint64(b.tx.Fee)
	right.Mul(right, new(big.Int).SetUint64(a.gas))
	if cmp := left.Cmp(right); cmp != 0 {
		return cmp > 0
	}
	if a.tx.Fee != b.tx.Fee {
		return a.tx.Fee > b.tx.Fee
	}
	if a.gas != b.gas {
		return a.gas < b.gas
	}
	return a.tx.ID < b.tx.ID
}

func samePriority(a, b struct{ tx types.Transaction; gas uint64 }) bool {
	return a.tx.Fee == b.tx.Fee && a.gas == b.gas && new(big.Int).SetUint64(a.tx.Fee).Cmp(new(big.Int).SetUint64(b.tx.Fee)) == 0
}

