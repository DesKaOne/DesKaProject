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

// GenesisHashForNetwork returns the deterministic genesis identity for a
// network. The caller can compare it with the canonical profile identity.
func GenesisHashForNetwork(profile config.NetworkConfig) string {
	return GenesisBlockForNetwork(profile).Hash
}
