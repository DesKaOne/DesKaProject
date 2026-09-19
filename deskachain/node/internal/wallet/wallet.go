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
	privateKey, err := w.PrivateKey()
	if err != nil {
		return err
	}
	tx.PublicKey = w.PublicKeyHex
	tx.RefreshID()
	sig, err := dkcrypto.SignHex(privateKey, tx.SigningBytes())
	if err != nil {
		return err
	}
	tx.Signature = sig
	tx.RefreshID()
	return nil
}
