package ledger

import (
	"testing"

	"deskachain/internal/asset"
	"deskachain/internal/config"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func assetProfile() config.NetworkConfig {
	profile := config.Localnet()
	profile.TxVersion = types.TxVersionAsset
	profile.BlockVersion = types.BlockVersionCanonical
	profile.Consensus.CoinbaseMaturity = 0
	return profile
}

func signAssetTx(t *testing.T, w wallet.Wallet, tx *types.Transaction, profile config.NetworkConfig) {
	t.Helper()
	if err := w.SignTransactionWithProfile(tx, profile); err != nil {
		t.Fatal(err)
	}
}

func TestV3AssetLifecycleAndIDRFees(t *testing.T) {
	profile := assetProfile()
	l := NewMatureWithProfile(profile.Consensus, profile)
	issuer, err := wallet.NewWithProfile(profile); if err != nil { t.Fatal(err) }
	alice, err := wallet.NewWithProfile(profile); if err != nil { t.Fatal(err) }
	bob, err := wallet.NewWithProfile(profile); if err != nil { t.Fatal(err) }

	if err := l.ApplyCoinbaseAtHeight(types.NewCoinbaseTransactionWithVersion(issuer.Address, 100, 1, types.TxVersionAsset), 1); err != nil { t.Fatal(err) }
	l.matureCoinbases(1)
	if err := l.ApplyCoinbaseAtHeight(types.NewCoinbaseTransactionWithVersion(alice.Address, 20, 1, types.TxVersionAsset), 1); err != nil { t.Fatal(err) }
	l.matureCoinbases(1)
	l.syncNativeAssetsFromAccounts()

	create := types.NewAssetCreateTransaction(issuer.Address, "Example USD", "EUSD", 6, 1_000_000, true, true, false, false, 2, 1)
	signAssetTx(t, issuer, &create, profile)
	if err := l.ApplyTransactionAtHeight(create, 2); err != nil { t.Fatal(err) }

	mint := types.NewAssetMintTransaction(issuer.Address, create.AssetID, alice.Address, 500, 3, 2)
	signAssetTx(t, issuer, &mint, profile)
	if err := l.ApplyTransactionAtHeight(mint, 3); err != nil { t.Fatal(err) }
	if got := l.AssetBalance(alice.Address, create.AssetID); got != 500 { t.Fatalf("alice token balance=%d", got) }

	transfer := types.NewAssetTransferTransaction(alice.Address, bob.Address, create.AssetID, 200, 4, 1)
	signAssetTx(t, alice, &transfer, profile)
	if err := l.ApplyTransactionAtHeight(transfer, 4); err != nil { t.Fatal(err) }
	if l.AssetBalance(alice.Address, create.AssetID) != 300 || l.AssetBalance(bob.Address, create.AssetID) != 200 {
		t.Fatalf("unexpected token balances")
	}
	if l.Balance(alice.Address) != 16 { t.Fatalf("alice IDR balance=%d want 16", l.Balance(alice.Address)) }
}

func TestV3PaymasterPaysNativeIDRFee(t *testing.T) {
	profile := assetProfile()
	l := NewMatureWithProfile(profile.Consensus, profile)
	issuer, err := wallet.NewWithProfile(profile); if err != nil { t.Fatal(err) }
	owner, err := wallet.NewWithProfile(profile); if err != nil { t.Fatal(err) }
	paymaster, err := wallet.NewWithProfile(profile); if err != nil { t.Fatal(err) }
	receiver, err := wallet.NewWithProfile(profile); if err != nil { t.Fatal(err) }

	for _, recipient := range []string{issuer.Address, paymaster.Address} {
		if err := l.ApplyCoinbaseAtHeight(types.NewCoinbaseTransactionWithVersion(recipient, 20, 1, types.TxVersionAsset), 1); err != nil { t.Fatal(err) }
		l.matureCoinbases(1)
	}
	l.syncNativeAssetsFromAccounts()
	create := types.NewAssetCreateTransaction(issuer.Address, "Example", "EXT", 6, 0, true, false, false, false, 1, 1)
	signAssetTx(t, issuer, &create, profile)
	if err := l.ApplyTransactionAtHeight(create, 2); err != nil { t.Fatal(err) }
	mint := types.NewAssetMintTransaction(issuer.Address, create.AssetID, owner.Address, 100, 1, 2)
	signAssetTx(t, issuer, &mint, profile)
	if err := l.ApplyTransactionAtHeight(mint, 3); err != nil { t.Fatal(err) }

	tx := types.NewAssetTransferTransaction(owner.Address, receiver.Address, create.AssetID, 40, 3, 1)
	tx.FeePayer = paymaster.Address
	signAssetTx(t, owner, &tx, profile)
	if err := paymaster.SignFeePayerAuthorization(&tx, profile); err != nil { t.Fatal(err) }
	if err := l.ApplyTransactionAtHeight(tx, 4); err != nil { t.Fatal(err) }

	if l.AssetBalance(owner.Address, create.AssetID) != 60 || l.AssetBalance(receiver.Address, create.AssetID) != 40 {
		t.Fatalf("unexpected sponsored token balances")
	}
	if l.Balance(paymaster.Address) != 17 { t.Fatalf("paymaster IDR balance=%d want 17", l.Balance(paymaster.Address)) }
	if l.Balance(owner.Address) != 0 { t.Fatalf("owner IDR balance=%d want 0", l.Balance(owner.Address)) }
}

func TestV3NativeIDRTransferSeparatesFeePayer(t *testing.T) {
	profile := assetProfile()
	l := NewMatureWithProfile(profile.Consensus, profile)
	sender, err := wallet.NewWithProfile(profile); if err != nil { t.Fatal(err) }
	paymaster, err := wallet.NewWithProfile(profile); if err != nil { t.Fatal(err) }
	receiver, err := wallet.NewWithProfile(profile); if err != nil { t.Fatal(err) }
	if err := l.ApplyCoinbaseAtHeight(types.NewCoinbaseTransactionWithVersion(sender.Address, 50, 1, types.TxVersionAsset), 1); err != nil { t.Fatal(err) }
	l.matureCoinbases(1)
	if err := l.ApplyCoinbaseAtHeight(types.NewCoinbaseTransactionWithVersion(paymaster.Address, 10, 1, types.TxVersionAsset), 1); err != nil { t.Fatal(err) }
	l.matureCoinbases(1)
	l.syncNativeAssetsFromAccounts()

	tx := types.NewAssetTransferTransaction(sender.Address, receiver.Address, asset.NativeAssetID, 20, 3, 1)
	tx.FeePayer = paymaster.Address
	signAssetTx(t, sender, &tx, profile)
	if err := paymaster.SignFeePayerAuthorization(&tx, profile); err != nil { t.Fatal(err) }
	if err := l.ApplyTransactionAtHeight(tx, 2); err != nil { t.Fatal(err) }

	if l.Balance(sender.Address) != 30 || l.Balance(paymaster.Address) != 7 || l.Balance(receiver.Address) != 20 {
		t.Fatalf("unexpected native IDR balances: sender=%d paymaster=%d receiver=%d", l.Balance(sender.Address), l.Balance(paymaster.Address), l.Balance(receiver.Address))
	}
}

