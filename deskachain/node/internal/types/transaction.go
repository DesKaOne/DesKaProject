package types

import (
	"encoding/json"
	"time"

	"deskachain/internal/crypto"
)

const (
	TxVersionLegacy    uint32 = 1
	TxVersionCanonical uint32 = 2
)

const CoinbaseSender = "COINBASE"

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
	tx := Transaction{
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
