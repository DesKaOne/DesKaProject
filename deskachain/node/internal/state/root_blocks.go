package state

import (
	"sort"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/types"
	"deskachain/internal/staking"
)

func RootForBlocks(blocks []types.Block, params config.ConsensusParams, profile config.NetworkConfig) (string, error) {
	l, err := ledger.ReplayMatureWithProfile(blocks, params, profile)
	if err != nil {
		return "", err
	}
	return RootForLedger(l)
}

func RootForBlock(parent []types.Block, block types.Block, params config.ConsensusParams, profile config.NetworkConfig) (string, error) {
	all := make([]types.Block, 0, len(parent)+1)
	all = append(all, parent...)
	all = append(all, block)
	return RootForBlocks(all, params, profile)
}

// StableDigestInputs returns the sorted state collections used by the root.
// It is intentionally exported for future snapshot/index implementations.
func StableDigestInputs(accounts []ledger.StateAccount, stakes []staking.Record) ([]ledger.StateAccount, []staking.Record) {
	accounts = append([]ledger.StateAccount(nil), accounts...)
	stakes = append([]staking.Record(nil), stakes...)
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].Address < accounts[j].Address })
	sort.Slice(stakes, func(i, j int) bool {
		if stakes[i].StakeID != stakes[j].StakeID {
			return stakes[i].StakeID < stakes[j].StakeID
		}
		return stakes[i].OwnerAddress < stakes[j].OwnerAddress
	})
	return accounts, stakes
}
