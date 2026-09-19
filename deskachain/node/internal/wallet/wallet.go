package wallet

import (
	"deskachain/internal/config"
	dkcrypto "deskachain/internal/crypto"
	"deskachain/internal/types"
	"github.com/btcsuite/btcd/btcec/v2"
)

type Wallet struct {
	Address       string `json:"address"`
	PrivateKeyHex string `json:"private_key"`
	PublicKeyHex  string `json:"public_key"`
}

func New() (Wallet, error) {
	return NewWithProfile(config.Localnet())
}

func NewWithProfile(profile config.NetworkConfig) (Wallet, error) {
	privateKey, err := dkcrypto.GeneratePrivateKey()
	if err != nil {
		return Wallet{}, err
	}
	publicKey := dkcrypto.PublicKeyToHex(privateKey.PubKey())
	address, err := dkcrypto.AddressFromPublicKeyForNetwork(publicKey, profile)
	if err != nil {
		return Wallet{}, err
	}
	return Wallet{
		Address:       address,
		PrivateKeyHex: dkcrypto.PrivateKeyToHex(privateKey),
		PublicKeyHex:  publicKey,
	}, nil
}

func FromPrivateKeyHex(privateKeyHex string, profile config.NetworkConfig) (Wallet, error) {
	privateKey, err := dkcrypto.PrivateKeyFromHex(privateKeyHex)
	if err != nil {
		return Wallet{}, err
	}
	publicKey := dkcrypto.PublicKeyToHex(privateKey.PubKey())
	address, err := dkcrypto.AddressFromPublicKeyForNetwork(publicKey, profile)
	if err != nil {
		return Wallet{}, err
	}
	return Wallet{
		Address:       address,
		PrivateKeyHex: dkcrypto.PrivateKeyToHex(privateKey),
		PublicKeyHex:  publicKey,
	}, nil
}

func (w Wallet) PrivateKey() (*btcec.PrivateKey, error) {
	return dkcrypto.PrivateKeyFromHex(w.PrivateKeyHex)
}

func (w Wallet) SignTransaction(tx *types.Transaction) error {
	return w.SignTransactionWithProfile(tx, config.Localnet())
}

func (w Wallet) SignTransactionWithProfile(tx *types.Transaction, profile config.NetworkConfig) error {
	if err := types.ValidateTransactionVersion(tx.ProtocolVersion(), profile.TxVersion); err != nil {
		return err
	}
	privateKey, err := w.PrivateKey()
	if err != nil {
		return err
	}
	tx.PublicKey = w.PublicKeyHex
	if err := tx.RefreshIDForChainID(profile.ChainID); err != nil {
		return err
	}
	signingBytes, err := tx.SigningBytesWithChainID(profile.ChainID)
	if err != nil {
		return err
	}
	sig, err := dkcrypto.SignHex(privateKey, signingBytes)
	if err != nil {
		return err
	}
	tx.Signature = sig
	return tx.RefreshIDForChainID(profile.ChainID)
}
