package chain

import (
	"fmt"
	"strings"

	"deskachain/internal/config"
	"deskachain/internal/types"
)

func GenesisBlock() types.Block {
	return GenesisBlockForNetwork(config.Localnet())
}

func GenesisBlockForNetwork(profile config.NetworkConfig) types.Block {
	marker := config.GenesisMessage
	if profile.Name != "" && profile.Name != "localnet" {
		marker = fmt.Sprintf("%s | network=%s | network_id=%s | chain_id=%d", config.GenesisMessage, profile.Name, profile.NetworkID, profile.ChainID)
	}
	block := types.Block{
		Height:        0,
		PreviousHash:  strings.Repeat("0", 64),
		Timestamp:     config.GenesisTimestamp,
		Difficulty:    0,
		MinerAddress:  "",
		Transactions:  nil,
		GenesisMarker: marker,
	}
	block.MerkleRoot = types.CalculateMerkleRoot(nil)
	block.Hash = block.CalculateHash()
	return block
}

// GenesisHashForNetwork returns the launch-sensitive genesis identity for a
// network. Mainnet uses the frozen hash from the canonical profile instead of
// relying on a regenerated value at each call site.
func GenesisHashForNetwork(profile config.NetworkConfig) string {
	genesis := GenesisBlockForNetwork(profile)
	if profile.Name == "mainnet" {
		if genesis.Hash != config.MainnetGenesisHash {
			return config.MainnetGenesisHash
		}
		return genesis.Hash
	}
	return genesis.Hash
}
