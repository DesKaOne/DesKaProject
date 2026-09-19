package types

import (
	"encoding/json"
	"fmt"
	"time"

	"deskachain/internal/crypto"
)

const (
	TxVersionLegacy    uint32 = 1
	TxVersionCanonical uint32 = 2
)

const (
	CoinbaseSender        = "COINBASE"
	MaxSupportedTxVersion uint32 = TxVersionCanonical
)

const (
	TxTypeTransfer    = "transfer"
	TxTypeCoinbase    = "coinbase"
	TxTypeStakeLock   = "stake_lock"
	TxTypeStakeUnlock = "stake_unlock"
)

type Transaction struct {
	Version   uint32 `json:"version,omitempty"`
	ID        string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	Fee       uint64 `json:"fee"`
	Nonce     uint64 `json:"nonce"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
	PublicKey string `json:"public_key"`
	Coinbase  bool   `json:"coinbase"`
	Type      string `json:"type,omitempty"`
	StakeID   string `json:"stake_id,omitempty"`
}

func NewUnsignedTransaction(from, to string, amount, fee, nonce uint64) Transaction {
	tx := Transaction{
		Version:   TxVersionLegacy,
		From:      from,
		To:        to,
		Amount:    amount,
		Fee:       fee,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
	}
	tx.ID = tx.CalculateID()
	return tx
}

func NewStakeLockTransaction(address string, amount, nonce uint64) Transaction {
	tx := Transaction{
		Version:   TxVersionLegacy,
		From:      address,
		To:        address,
		Amount:    amount,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
		Type:      TxTypeStakeLock,
	}
	tx.ID = tx.CalculateID()
	tx.StakeID = tx.ID
	return tx
}

func NewStakeUnlockTransaction(address, stakeID string, nonce uint64) Transaction {
	tx := Transaction{
		Version:   TxVersionLegacy,
		From:      address,
		To:        address,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
		Type:      TxTypeStakeUnlock,
		StakeID:   stakeID,
	}
	tx.ID = tx.CalculateID()
	return tx
}

func NewCoinbaseTransaction(to string, amount uint64, height uint64) Transaction {
	return NewCoinbaseTransactionWithVersion(to, amount, height, TxVersionLegacy)
}

func NewCoinbaseTransactionWithVersion(to string, amount uint64, height uint64, version uint32) Transaction {
	tx := Transaction{
		Version:   version,
		From:      CoinbaseSender,
		To:        to,
		Amount:    amount,
		Timestamp: int64(height),
		Coinbase:  true,
	}
	tx.ID = tx.CalculateID()
	return tx
}

func (tx Transaction) ProtocolVersion() uint32 {
	if tx.Version == 0 {
		return TxVersionLegacy
	}
	return tx.Version
}

func ValidateTransactionVersion(version, activeVersion uint32) error {
	if version == 0 {
		version = TxVersionLegacy
	}
	if activeVersion == 0 {
		activeVersion = TxVersionLegacy
	}
	if version > MaxSupportedTxVersion {
		return fmt.Errorf("unsupported transaction version: %d", version)
	}
	if version != activeVersion {
		return fmt.Errorf("transaction version %d is not active on this network (active version %d)", version, activeVersion)
	}
	return nil
}

func (tx Transaction) SigningBytesWithChainID(chainID uint64) ([]byte, error) {
	switch tx.ProtocolVersion() {
	case TxVersionLegacy:
		return tx.SigningBytes(), nil
	case TxVersionCanonical:
		return tx.CanonicalSigningBytesWithChainID(chainID)
	default:
		return nil, fmt.Errorf("unsupported transaction version: %d", tx.ProtocolVersion())
	}
}

func (tx Transaction) CalculateIDForChainID(chainID uint64) (string, error) {
	raw, err := tx.SigningBytesWithChainID(chainID)
	if err != nil {
		return "", err
	}
	return crypto.DoubleSHA256Hex(raw), nil
}

func (tx *Transaction) RefreshIDForChainID(chainID uint64) error {
	id, err := tx.CalculateIDForChainID(chainID)
	if err != nil {
		return err
	}
	tx.ID = id
	return nil
}

func (tx Transaction) TxType() string {
	if tx.Type != "" {
		return tx.Type
	}
	if tx.Coinbase {
		return TxTypeCoinbase
	}
	return TxTypeTransfer
}

func (tx Transaction) SigningBytes() []byte {
	if tx.ProtocolVersion() >= TxVersionCanonical {
		return tx.CanonicalSigningBytes()
	}
	copy := tx
	copy.ID = ""
	copy.Signature = ""
	copy.Version = 0
	if copy.TxType() == TxTypeStakeLock {
		copy.StakeID = ""
	}
	raw, _ := json.Marshal(copy)
	return raw
}

func (tx Transaction) CalculateID() string {
	return crypto.DoubleSHA256Hex(tx.SigningBytes())
}

func (tx *Transaction) RefreshID() {
	tx.ID = tx.CalculateID()
}
