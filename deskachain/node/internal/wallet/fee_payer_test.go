package wallet

import (
	"indochain/internal/config"
	"indochain/internal/types"
	"testing"
)

func TestFeePayerAuthorization(t *testing.T) {
	profile := config.Localnet()
	profile.TxVersion = types.TxVersionAsset
	owner, err := NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	paymaster, err := NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	tx := types.NewAssetTransferTransaction(owner.Address, paymaster.Address, "asset:usd", 100, 3, 1)
	tx.FeePayer = paymaster.Address
	if err := owner.SignTransactionWithProfile(&tx, profile); err != nil {
		t.Fatal(err)
	}
	if err := paymaster.SignFeePayerAuthorization(&tx, profile); err != nil {
		t.Fatal(err)
	}
	if tx.FeePayer != paymaster.Address || tx.FeePayerPublicKey == "" || tx.FeePayerSignature == "" {
		t.Fatalf("missing paymaster authorization: %+v", tx)
	}
	if err := tx.ValidateFeePayerAuthorization(profile); err != nil {
		t.Fatal(err)
	}
	beforeID := tx.ID
	if err := paymaster.SignFeePayerAuthorization(&tx, profile); err != nil {
		t.Fatal(err)
	}
	if tx.ID != beforeID {
		t.Fatal("paymaster authorization changed transaction id")
	}
}
