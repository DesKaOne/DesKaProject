package wallet

import (
	"testing"
	"deskachain/internal/config"
	"deskachain/internal/types"
)

func TestFeePayerAuthorization(t *testing.T) {
	owner, err := NewWithProfile(config.Localnet()); if err != nil { t.Fatal(err) }
	paymaster, err := NewWithProfile(config.Localnet()); if err != nil { t.Fatal(err) }
	tx := types.NewAssetTransferTransaction(owner.Address, paymaster.Address, "asset:usd", 100, 3, 1)
	if err := owner.SignTransactionWithProfile(&tx, config.Localnet()); err != nil { t.Fatal(err) }
	if err := paymaster.SignFeePayerAuthorization(&tx, config.Localnet()); err != nil { t.Fatal(err) }
	if tx.FeePayer != paymaster.Address || tx.FeePayerPublicKey == "" || tx.FeePayerSignature == "" { t.Fatalf("missing paymaster authorization: %+v", tx) }
}