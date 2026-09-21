package model

type NetworkInfo struct { Network string `json:"network"`; NetworkID string `json:"network_id"`; ChainID uint64 `json:"chain_id"`; GenesisHash string `json:"genesis_hash,omitempty"` }
