package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"deskachain/internal/crypto"
)

const (
	BlockVersionLegacy    uint32 = 1
	BlockVersionCanonical uint32 = 2
	MaxSupportedBlockVersion uint32 = BlockVersionCanonical
)

type Block struct {
	Version       uint32        `json:"version,omitempty"`
	Height        uint64        `json:"height"`
	PreviousHash  string        `json:"previous_hash"`
	Timestamp     int64         `json:"timestamp"`
	Nonce         uint64        `json:"nonce"`
	Difficulty    uint32        `json:"difficulty"`
	MinerAddress  string        `json:"miner_address"`
	Transactions  []Transaction `json:"transactions"`
	MerkleRoot    string        `json:"merkle_root"`
	Hash          string        `json:"hash"`
	GenesisMarker string        `json:"genesis_marker,omitempty"`
}

func NewBlock(height uint64, previousHash, minerAddress string, difficulty uint32, transactions []Transaction) Block {
	return NewBlockWithVersion(height, previousHash, minerAddress, difficulty, transactions, BlockVersionLegacy)
}

func NewBlockWithVersion(height uint64, previousHash, minerAddress string, difficulty uint32, transactions []Transaction, version uint32) Block {
	block := Block{
		Version:      version,
		Height:       height,
		PreviousHash: previousHash,
		Timestamp:    time.Now().Unix(),
		Difficulty:   difficulty,
		MinerAddress: minerAddress,
		Transactions: transactions,
	}
	block.MerkleRoot = CalculateMerkleRoot(transactions)
	return block
}

func (b Block) ProtocolVersion() uint32 {
	if b.Version == 0 {
		return BlockVersionLegacy
	}
	return b.Version
}

func ValidateBlockVersion(version, activeVersion uint32) error {
	if version == 0 {
		version = BlockVersionLegacy
	}
	if activeVersion == 0 {
		activeVersion = BlockVersionLegacy
	}
	if version > MaxSupportedBlockVersion {
		return fmt.Errorf("unsupported block version: %d", version)
	}
	if version != activeVersion {
		return fmt.Errorf("block version %d is not active on this network (active version %d)", version, activeVersion)
	}
	return nil
}

func (b Block) HeaderBytesWithNonce(nonce uint64) []byte {
	if b.ProtocolVersion() == BlockVersionCanonical {
		raw, _ := b.CanonicalHeaderBytesWithNonce(nonce)
		return raw
	}
	parts := []string{
		strconv.FormatUint(b.Height, 10),
		b.PreviousHash,
		strconv.FormatInt(b.Timestamp, 10),
		strconv.FormatUint(nonce, 10),
		strconv.FormatUint(uint64(b.Difficulty), 10),
		b.MinerAddress,
		b.MerkleRoot,
		b.GenesisMarker,
	}
	return []byte(strings.Join(parts, "|"))
}

func (b Block) CalculateHash() string {
	return crypto.DoubleSHA256Hex(b.HeaderBytesWithNonce(b.Nonce))
}

func CalculateMerkleRoot(transactions []Transaction) string {
	if len(transactions) == 0 {
		return crypto.DoubleSHA256Hex([]byte{})
	}
	level := make([]string, len(transactions))
	for i, tx := range transactions {
		id := tx.ID
		if id == "" {
			id = tx.CalculateID()
		}
		level[i] = id
	}
	for len(level) > 1 {
		next := make([]string, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := left
			if i+1 < len(level) {
				right = level[i+1]
			}
			next = append(next, crypto.DoubleSHA256Hex([]byte(left+right)))
		}
		level = next
	}
	return level[0]
}

func (b Block) JSON() []byte {
	raw, _ := json.MarshalIndent(b, "", "  ")
	return raw
}
