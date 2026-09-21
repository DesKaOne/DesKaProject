package wallet

import (
	"errors"
	"fmt"

	"indochain/internal/config"
	"indochain/internal/types"
)

var (
	ErrInvalidRecipient = errors.New("invalid recipient address")
	ErrInvalidAmount    = errors.New("amount must be greater than zero")
	ErrInvalidFee       = errors.New("fee is below the network minimum")
)

type TransferRequest struct {
	To       string
	Amount   uint64
	Fee      uint64
	Nonce    uint64
	AssetID  string
}

func (w Wallet) NewTransfer(profile config.NetworkConfig, to string, amount, fee, nonce uint64) (types.Transaction, error) {
	return w.NewTransferForAsset(profile, to, config.NativeAssetID, amount, fee, nonce)
}

func (w Wallet) NewTransferForAsset(profile config.NetworkConfig, to, assetID string, amount, fee, nonce uint64) (types.Transaction, error) {
	if err := config.ValidateNetworkProfile(profile); err != nil {
		return types.Transaction{}, fmt.Errorf("invalid network profile: %w", err)
	}
	if err := ValidateWalletNetwork(w, profile); err != nil {
		return types.Transaction{}, err
	}
	if err := validateRecipient(profile, to, w.Address); err != nil {
		return types.Transaction{}, err
	}
	if amount == 0 {
		return types.Transaction{}, ErrInvalidAmount
	}
	if fee < profile.Fee.MinFee {
		return types.Transaction{}, fmt.Errorf("%w: got %d want >= %d", ErrInvalidFee, fee, profile.Fee.MinFee)
	}
	if assetID == "" {
		assetID = config.NativeAssetID
	}
	tx := types.NewAssetTransferTransaction(w.Address, to, assetID, amount, fee, nonce)
	if err := tx.ValidateAssetEnvelope(); err != nil {
		return types.Transaction{}, err
	}
	return tx, nil
}

func (w Wallet) NewStakeLock(profile config.NetworkConfig, amount, nonce uint64) (types.Transaction, error) {
	if err := config.ValidateNetworkProfile(profile); err != nil {
		return types.Transaction{}, fmt.Errorf("invalid network profile: %w", err)
	}
	if err := ValidateWalletNetwork(w, profile); err != nil {
		return types.Transaction{}, err
	}
	if !profile.Consensus.Staking.Enabled {
		return types.Transaction{}, errors.New("staking is disabled on this network")
	}
	if amount < profile.Consensus.Staking.MinStakeAmount {
		return types.Transaction{}, fmt.Errorf("stake amount below network minimum")
	}
	tx := types.NewStakeLockTransaction(w.Address, amount, nonce)
	tx.Fee = profile.Fee.MinFee
	return tx, nil
}

func (w Wallet) SignTransfer(profile config.NetworkConfig, to string, amount, fee, nonce uint64) (types.Transaction, error) {
	tx, err := w.NewTransfer(profile, to, amount, fee, nonce)
	if err != nil {
		return types.Transaction{}, err
	}
	if err := w.SignTransactionWithProfile(&tx, profile); err != nil {
		return types.Transaction{}, err
	}
	if tx.PublicKey == "" || tx.Signature == "" || tx.ID == "" {
		return types.Transaction{}, errors.New("signed transaction is incomplete")
	}
	return tx, nil
}

func ValidateWalletNetwork(w Wallet, profile config.NetworkConfig) error {
	if w.Address == "" || w.PublicKeyHex == "" || w.PrivateKeyHex == "" {
		return errors.New("wallet credentials are incomplete")
	}
	address, err := indochainAddressFromPublicKey(w.PublicKeyHex, profile)
	if err != nil {
		return fmt.Errorf("wallet public key is invalid for network: %w", err)
	}
	if address != w.Address {
		return errors.New("wallet address does not match public key for selected network")
	}
	return nil
}

func validateRecipient(profile config.NetworkConfig, to, from string) error {
	if to == "" || to == from {
		return ErrInvalidRecipient
	}
	if err := validateAddressForNetwork(to, profile); err != nil {
		return ErrInvalidRecipient
	}
	return nil
}
