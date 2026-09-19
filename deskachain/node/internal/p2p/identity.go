package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"deskachain/internal/config"
	"deskachain/internal/types"
)

func LoadOrCreateNodeID(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		id := string(raw)
		if id != "" {
			return id, nil
		}
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	id := hex.EncodeToString(buf)
	return id, os.WriteFile(path, []byte(id), 0644)
}

func ValidateHandshake(local config.NetworkConfig, peer Handshake) error {
	if peer.NetworkID != local.NetworkID {
		return errors.New("peer rejected: network id mismatch")
	}
	if peer.ChainID != local.ChainID {
		return errors.New("peer rejected: chain id mismatch")
	}
	if peer.GenesisHash != local.GenesisHash {
		return errors.New("peer rejected: genesis hash mismatch")
	}
	if peer.ProtocolVersion < local.MinProtocolVersion || local.ProtocolVersion < peer.MinProtocolVersion {
		return errors.New("peer rejected: incompatible protocol version")
	}
	return nil
}

func ValidateStatus(local config.NetworkConfig, peer Status) error {
	if peer.NetworkID != local.NetworkID {
		return errors.New("peer rejected: network id mismatch")
	}
	if peer.ChainID != local.ChainID {
		return errors.New("peer rejected: chain id mismatch")
	}
	if peer.GenesisHash != local.GenesisHash {
		return errors.New("peer rejected: genesis hash mismatch")
	}
	if peer.ProtocolVersion < local.MinProtocolVersion {
		return errors.New("peer rejected: incompatible protocol version")
	}
	return nil
}

func HeaderFromBlock(block types.Block) BlockHeader {
	return BlockHeader{
		Height:       block.Height,
		Hash:         block.Hash,
		PreviousHash: block.PreviousHash,
		Timestamp:    block.Timestamp,
		Difficulty:   block.Difficulty,
		MerkleRoot:   block.MerkleRoot,
		TxCount:      len(block.Transactions),
		MinerAddress: block.MinerAddress,
	}
}
