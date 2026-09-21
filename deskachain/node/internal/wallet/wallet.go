package wallet

import (
	"errors"

	"indochain/internal/config"
	indrypto "indochain/internal/crypto"
	"indochain/internal/types"

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
	privateKey, err := indrypto.GeneratePrivateKey()
	if err != nil {
		return Wallet{}, err
	}
	publicKey := indrypto.PublicKeyToHex(privateKey.PubKey())
	address, err := indrypto.AddressFromPublicKeyForNetwork(publicKey, profile)
	if err != nil {
		return Wallet{}, err
	}
	return Wallet{
		Address:       address,
		PrivateKeyHex: indrypto.PrivateKeyToHex(privateKey),
		PublicKeyHex:  publicKey,
	}, nil
}

func FromPrivateKeyHex(privateKeyHex string, profile config.NetworkConfig) (Wallet, error) {
	privateKey, err := indrypto.PrivateKeyFromHex(privateKeyHex)
	if err != nil {
		return Wallet{}, err
	}
	publicKey := indrypto.PublicKeyToHex(privateKey.PubKey())
	address, err := indrypto.AddressFromPublicKeyForNetwork(publicKey, profile)
	if err != nil {
		return Wallet{}, err
	}
	return Wallet{
		Address:       address,
		PrivateKeyHex: indrypto.PrivateKeyToHex(privateKey),
		PublicKeyHex:  publicKey,
	}, nil
}

func (w Wallet) PrivateKey() (*btcec.PrivateKey, error) {
	return indrypto.PrivateKeyFromHex(w.PrivateKeyHex)
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
	if tx.ProtocolVersion() == types.TxVersionAsset && tx.TxType() == types.TxTypeAssetCreate {
		if err := tx.RefreshDerivedAssetID(); err != nil {
			return err
		}
	}
	signingBytes, err := tx.SigningBytesWithChainID(profile.ChainID)
	if err != nil {
		return err
	}
	sig, err := indrypto.SignHex(privateKey, signingBytes)
	if err != nil {
		return err
	}
	tx.Signature = sig
	return tx.RefreshIDForChainID(profile.ChainID)
}

// SignFeePayerAuthorization signs the native-dIDR fee sponsorship authorization.
// It deliberately does not recalculate tx.ID because the sender transaction
// identity must remain stable when a paymaster adds its authorization.
func (w Wallet) SignFeePayerAuthorization(tx *types.Transaction, profile config.NetworkConfig) error {
	if tx == nil {
		return errors.New("transaction is required")
	}
	if tx.ProtocolVersion() != types.TxVersionAsset {
		return errors.New("fee payer authorization requires transaction version 3")
	}
	if tx.FeePayer == "" {
		tx.FeePayer = w.Address
	}
	if tx.EffectiveFeePayer() != w.Address {
		return errors.New("wallet is not the transaction fee payer")
	}
	privateKey, err := w.PrivateKey()
	if err != nil {
		return err
	}
	tx.FeePayerPublicKey = w.PublicKeyHex
	signingBytes, err := tx.FeePayerSigningBytesWithChainID(profile.ChainID)
	if err != nil {
		return err
	}
	sig, err := indrypto.SignHex(privateKey, signingBytes)
	if err != nil {
		return err
	}
	tx.FeePayerSignature = sig
	return nil
}
