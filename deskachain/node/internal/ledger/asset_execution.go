package ledger

import (
	"errors"
	"fmt"

	"deskachain/internal/arith"
	"deskachain/internal/asset"
	"deskachain/internal/crypto"
	"deskachain/internal/fees"
	"deskachain/internal/types"
)

func (l *MatureLedger) validateAssetTransactionV3(tx types.Transaction) error {
	if err := l.validateFeePolicy(tx); err != nil {
		return err
	}
	if err := tx.ValidateAssetEnvelope(); err != nil {
		return err
	}
	if tx.AssetID == "" && tx.TxType() == types.TxTypeTransfer {
		return errors.New("asset transfer asset id is required")
	}
	if err := crypto.ValidateAddressForNetwork(tx.From, l.profile); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	switch tx.TxType() {
	case types.TxTypeTransfer, types.TxTypeAssetMint:
		if err := crypto.ValidateAddressForNetwork(tx.To, l.profile); err != nil {
			return fmt.Errorf("invalid recipient address: %w", err)
		}
	case types.TxTypeAssetBurn:
		if err := crypto.ValidateAddressForNetwork(tx.From, l.profile); err != nil {
			return fmt.Errorf("invalid burn owner: %w", err)
		}
	}
	address, err := crypto.AddressFromPublicKeyForNetwork(tx.PublicKey, l.profile)
	if err != nil || address != tx.From {
		return errors.New("public key does not match sender address")
	}
	expectedID, err := tx.CalculateIDForChainID(l.profile.ChainID)
	if err != nil {
		return err
	}
	if tx.ID != expectedID {
		return errors.New("transaction id mismatch")
	}
	if tx.TxType() == types.TxTypeAssetCreate {
		if want := asset.DerivedID(tx.ID); tx.AssetID != want {
			return fmt.Errorf("asset create id mismatch: got %s want %s", tx.AssetID, want)
		}
	}
	signingBytes, err := tx.SigningBytesWithChainID(l.profile.ChainID)
	if err != nil {
		return err
	}
	if !crypto.VerifyHex(tx.PublicKey, tx.Signature, signingBytes) {
		return errors.New("invalid transaction signature")
	}
	if err := tx.ValidateFeePayerAuthorization(l.profile); err != nil {
		return err
	}

	sender := l.accounts[tx.From]
	expectedNonce, err := arith.Add(sender.Nonce, 1)
	if err != nil {
		return errors.New("account nonce overflow")
	}
	if tx.Nonce != expectedNonce {
		return fmt.Errorf("invalid account nonce: got %d want %d", tx.Nonce, expectedNonce)
	}

	switch tx.TxType() {
	case types.TxTypeTransfer:
		assetID := tx.EffectiveAssetID()
		if asset.IsNative(assetID) {
			cost := tx.Amount
			if tx.EffectiveFeePayer() == tx.From {
				cost, err = arith.Add(tx.Amount, tx.Fee)
				if err != nil {
					return fmt.Errorf("native IDR transfer cost overflow: %w", err)
				}
			}
			if err := l.validateNativeSpend(tx.From, cost); err != nil {
				return err
			}
			if tx.EffectiveFeePayer() != tx.From {
				if err := l.validateNativeSpend(tx.EffectiveFeePayer(), tx.Fee); err != nil {
					return fmt.Errorf("fee payer: %w", err)
				}
			}
			if l.assets.Balance(tx.From, asset.NativeAssetID) < tx.Amount {
				return errors.New("native IDR asset balance insufficient")
			}
			return nil
		}
		if l.assets.Balance(tx.From, assetID) < tx.Amount {
			return asset.ErrInsufficientBalance
		}
		return l.validateNativeSpend(tx.EffectiveFeePayer(), tx.Fee)
	case types.TxTypeAssetCreate:
		if _, ok := l.assets.Definition(tx.AssetID); ok {
			return asset.ErrAssetExists
		}
		def := asset.Definition{
			ID:           tx.AssetID,
			Name:         tx.AssetName,
			Symbol:       tx.AssetSymbol,
			Decimals:     tx.AssetDecimals,
			Kind:         asset.KindFungible,
			Issuer:       tx.From,
			MaxSupply:    tx.AssetMaxSupply,
			Mintable:     tx.AssetMintable,
			Burnable:     tx.AssetBurnable,
			Pausable:     tx.AssetPausable,
			Permissioned: tx.AssetPermissioned,
			Status:       asset.StatusActive,
		}
		if err := asset.ValidateDefinition(def); err != nil {
			return err
		}
		return l.validateNativeSpend(tx.EffectiveFeePayer(), tx.Fee)
	case types.TxTypeAssetMint:
		def, ok := l.assets.Definition(tx.AssetID)
		if !ok {
			return asset.ErrAssetNotFound
		}
		if asset.IsNative(def.ID) {
			return errors.New("native IDR cannot be minted through asset operation")
		}
		if def.Issuer != tx.From {
			return asset.ErrUnauthorized
		}
		if !def.Mintable {
			return errors.New("asset is not mintable")
		}
		if def.Status == asset.StatusFrozen {
			return asset.ErrAssetFrozen
		}
		newSupply, err := arith.Add(def.TotalSupply, tx.Amount)
		if err != nil {
			return fmt.Errorf("asset supply overflow: %w", err)
		}
		if def.MaxSupply > 0 && newSupply > def.MaxSupply {
			return errors.New("asset max supply exceeded")
		}
		if next, err := arith.Add(l.assets.Balance(tx.To, tx.AssetID), tx.Amount); err != nil || next == 0 {
			if err != nil {
				return fmt.Errorf("asset balance overflow: %w", err)
			}
		}
		return l.validateNativeSpend(tx.EffectiveFeePayer(), tx.Fee)
	case types.TxTypeAssetBurn:
		def, ok := l.assets.Definition(tx.AssetID)
		if !ok {
			return asset.ErrAssetNotFound
		}
		if asset.IsNative(def.ID) {
			return errors.New("native IDR cannot be burned through asset operation")
		}
		if !def.Burnable {
			return errors.New("asset is not burnable")
		}
		if def.Status == asset.StatusFrozen {
			return asset.ErrAssetFrozen
		}
		if l.assets.Balance(tx.From, tx.AssetID) < tx.Amount {
			return asset.ErrInsufficientBalance
		}
		if def.TotalSupply < tx.Amount {
			return errors.New("asset total supply underflow")
		}
		return l.validateNativeSpend(tx.EffectiveFeePayer(), tx.Fee)
	default:
		return fmt.Errorf("unsupported v3 asset transaction type: %s", tx.TxType())
	}
}

func (l *MatureLedger) validateNativeSpend(address string, amount uint64) error {
	if amount == 0 {
		return nil
	}
	account := l.accounts[address]
	if account.Confirmed < amount {
		return errors.New("insufficient native IDR balance")
	}
	if account.Mature < amount {
		return ErrImmatureBalance
	}
	active, unlocking, _ := l.stakes.AddressSummary(address, l.currentHeight)
	locked, err := arith.Add(active, unlocking)
	if err != nil {
		return errors.New("locked stake total overflow")
	}
	if account.Mature < locked || account.Mature-locked < amount {
		return errors.New("invalid transaction: spends locked stake")
	}
	if l.assets.Balance(address, asset.NativeAssetID) < amount {
		return errors.New("native IDR asset balance insufficient")
	}
	return nil
}

func (l *MatureLedger) spendNative(address string, amount uint64) error {
	if err := l.validateNativeSpend(address, amount); err != nil {
		return err
	}
	account := l.accounts[address]
	confirmed, err := arith.Sub(account.Confirmed, amount)
	if err != nil {
		return errors.New("native IDR balance underflow")
	}
	mature, err := arith.Sub(account.Mature, amount)
	if err != nil {
		return ErrImmatureBalance
	}
	if err := l.assets.Debit(address, asset.NativeAssetID, amount); err != nil {
		return err
	}
	account.Confirmed = confirmed
	account.Mature = mature
	l.accounts[address] = account
	return nil
}

func (l *MatureLedger) receiveNative(address string, amount uint64) error {
	if amount == 0 {
		return nil
	}
	account := l.accounts[address]
	confirmed, err := arith.Add(account.Confirmed, amount)
	if err != nil {
		return fmt.Errorf("recipient native IDR balance overflow: %w", err)
	}
	mature, err := arith.Add(account.Mature, amount)
	if err != nil {
		return fmt.Errorf("recipient mature IDR balance overflow: %w", err)
	}
	if err := l.assets.Credit(address, asset.NativeAssetID, amount); err != nil {
		return err
	}
	account.Confirmed = confirmed
	account.Mature = mature
	l.accounts[address] = account
	return nil
}

func (l *MatureLedger) chargeNativeFee(payer string, fee uint64) error {
	if fee == 0 {
		return nil
	}
	if err := l.spendNative(payer, fee); err != nil {
		return err
	}
	if err := l.assets.Credit(asset.FeeCollectorAddress, asset.NativeAssetID, fee); err != nil {
		// spendNative already mutated both account and native-asset balance.
		// Restore the payer before returning the collector failure.
		if restoreErr := l.receiveNative(payer, fee); restoreErr != nil {
			return fmt.Errorf("fee collector credit failed: %v; payer restore failed: %w", err, restoreErr)
		}
		return fmt.Errorf("fee collector credit failed: %w", err)
	}
	return nil
}

func (l *MatureLedger) settleFees(miner string) error {
	if fee := l.assets.Balance(asset.FeeCollectorAddress, asset.NativeAssetID); fee != 0 {
		if miner == "" {
			return errors.New("miner address is required to settle fees")
		}
		if err := l.assets.Debit(asset.FeeCollectorAddress, asset.NativeAssetID, fee); err != nil {
			return err
		}
		if err := l.receiveNative(miner, fee); err != nil {
			_ = l.assets.Credit(asset.FeeCollectorAddress, asset.NativeAssetID, fee)
			return fmt.Errorf("fee settlement failed: %w", err)
		}
	}
	return nil
}

func (l *MatureLedger) validateFeePolicy(tx types.Transaction) error {
	return fees.Validate(tx, l.profile)
}

func (l *MatureLedger) applyAssetTransactionV3(tx types.Transaction, height uint64) error {
	backup := l.Clone()
	var err error

	switch tx.TxType() {
	case types.TxTypeTransfer:
		if asset.IsNative(tx.EffectiveAssetID()) {
			if tx.EffectiveFeePayer() == tx.From {
				cost, costErr := arith.Add(tx.Amount, tx.Fee)
				if costErr != nil {
					return costErr
				}
				if err = l.spendNative(tx.From, cost); err != nil {
					break
				}
			} else {
				if err = l.spendNative(tx.From, tx.Amount); err != nil {
					break
				}
				if err = l.chargeNativeFee(tx.EffectiveFeePayer(), tx.Fee); err != nil {
					break
				}
			}
			err = l.receiveNative(tx.To, tx.Amount)
		} else {
			if err = l.assets.Transfer(tx.AssetID, tx.From, tx.To, tx.Amount); err != nil {
				break
			}
			err = l.chargeNativeFee(tx.EffectiveFeePayer(), tx.Fee)
		}
	case types.TxTypeAssetCreate:
		_, err = l.assets.CreateFromTransactionID(
			tx.ID,
			tx.AssetName,
			tx.AssetSymbol,
			tx.AssetDecimals,
			tx.AssetMaxSupply,
			tx.AssetMintable,
			tx.AssetBurnable,
			tx.AssetPausable,
			tx.AssetPermissioned,
			tx.From,
		)
		if err == nil {
			err = l.chargeNativeFee(tx.EffectiveFeePayer(), tx.Fee)
		}
	case types.TxTypeAssetMint:
		err = l.assets.Mint(tx.AssetID, tx.From, tx.To, tx.Amount)
		if err == nil {
			err = l.chargeNativeFee(tx.EffectiveFeePayer(), tx.Fee)
		}
	case types.TxTypeAssetBurn:
		err = l.assets.Burn(tx.AssetID, tx.From, tx.Amount)
		if err == nil {
			err = l.chargeNativeFee(tx.EffectiveFeePayer(), tx.Fee)
		}
	default:
		err = fmt.Errorf("unsupported v3 asset transaction type at height %d: %s", height, tx.TxType())
	}

	if err != nil {
		*l = *backup
		return err
	}

	from := l.accounts[tx.From]
	from.Nonce = tx.Nonce
	l.accounts[tx.From] = from
	return nil
}

func (l *MatureLedger) syncNativeAssetsFromAccounts() {
	for address, account := range l.accounts {
		current := l.assets.Balance(address, asset.NativeAssetID)
		switch {
		case current < account.Confirmed:
			_ = l.assets.Credit(address, asset.NativeAssetID, account.Confirmed-current)
		case current > account.Confirmed:
			_ = l.assets.Debit(address, asset.NativeAssetID, current-account.Confirmed)
		}
	}
}
