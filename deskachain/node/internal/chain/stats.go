package chain

import (
	"indochain/internal/config"
	"indochain/internal/ledger"
	"indochain/internal/types"
)

type ChainStats struct {
	Blocks               int
	CoinbaseBlocks       int
	TotalTransactions    int
	CoinbaseTransactions int
	NormalTransactions   int
	TotalSupply          uint64
	CirculatingSupply    uint64
	CumulativeWork       uint64
}

func CalculateChainStats(blocks []types.Block) ChainStats {
	return CalculateChainStatsWithConsensus(blocks, config.Localnet().Consensus)
}

func CalculateChainStatsWithConsensus(blocks []types.Block, consensus config.ConsensusParams) ChainStats {
	return calculateChainStats(blocks, consensus, config.Localnet())
}

func CalculateChainStatsWithProfile(blocks []types.Block, profile config.NetworkConfig) ChainStats {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	return calculateChainStats(blocks, profile.Consensus, profile)
}

func calculateChainStats(blocks []types.Block, consensus config.ConsensusParams, profile config.NetworkConfig) ChainStats {
	stats := ChainStats{
		Blocks:            len(blocks),
		TotalSupply:       ledger.TotalSupply(blocks),
		CirculatingSupply: ledger.CirculatingSupplyWithProfile(blocks, consensus, profile),
		CumulativeWork:    CalculateCumulativeWork(blocks),
	}
	for _, block := range blocks {
		stats.TotalTransactions += len(block.Transactions)
		if block.Height == 0 {
			for _, tx := range block.Transactions {
				if tx.Coinbase {
					stats.CoinbaseTransactions++
				} else {
					stats.NormalTransactions++
				}
			}
			continue
		}
		hasCoinbase := false
		for _, tx := range block.Transactions {
			if tx.Coinbase {
				stats.CoinbaseTransactions++
				hasCoinbase = true
			} else {
				stats.NormalTransactions++
			}
		}
		if hasCoinbase {
			stats.CoinbaseBlocks++
		}
	}
	return stats
}
