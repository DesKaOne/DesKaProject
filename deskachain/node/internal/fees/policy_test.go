package fees

import (
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func testProfile() config.NetworkConfig {
	p := config.Localnet()
	p.TxVersion = types.TxVersionAsset
	p.BlockVersion = types.BlockVersionCanonical
	return p
}

func TestMinimumFeeUsesGasPolicy(t *testing.T) {
	p := testProfile()
	p.Fee.MinGasPrice = 2
	p.Fee.MinFee = 1
	tx := types.NewAssetTransferTransaction("from", "to", "asset:usd", 1, 0, 1)

	quote, err := Estimate(tx, p)
	if err != nil {
		t.Fatal(err)
	}
	if quote.GasUnits <= 12 {
		t.Fatalf("gas=%d want base plus encoded size", quote.GasUnits)
	}
	want := quote.GasUnits * 2
	if quote.MinFee != want {
		t.Fatalf("min fee=%d want=%d", quote.MinFee, want)
	}
	if quote.Sufficient {
		t.Fatal("zero fee unexpectedly sufficient")
	}
}

func TestMinimumFeeHonorsFloor(t *testing.T) {
	p := testProfile()
	p.Fee.MinGasPrice = 0
	p.Fee.MinFee = 7
	tx := types.NewAssetMintTransaction("issuer", "asset:usd", "to", 1, 7, 1)

	quote, err := Estimate(tx, p)
	if err != nil {
		t.Fatal(err)
	}
	if quote.MinFee != 7 || !quote.Sufficient {
		t.Fatalf("quote=%+v", quote)
	}
	if err := Validate(tx, p); err != nil {
		t.Fatal(err)
	}
}

func TestGasLimitRejected(t *testing.T) {
	p := testProfile()
	p.Fee.MaxGasPerTx = 1
	tx := types.NewAssetCreateTransaction("issuer", "Name", "SYM", 2, 0, true, false, false, false, 1, 1)
	if _, _, err := GasUsed(tx, p); err == nil {
		t.Fatal("gas limit overflow accepted")
	}
}

func TestLegacyTransactionsAreOutsideV3FeePolicy(t *testing.T) {
	p := config.Localnet()
	p.Fee.Enabled = true
	tx := types.NewUnsignedTransaction("from", "to", 1, 0, 1)
	if got, err := MinimumFee(tx, p); err != nil || got != 0 {
		t.Fatalf("legacy minimum fee=%d err=%v", got, err)
	}
}

func TestGasQuoteStableAfterSenderSigning(t *testing.T) {
	p := testProfile()
	w, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	tx := types.NewAssetTransferTransaction(w.Address, "recipient", p.Asset.NativeAssetID, 10, 1, 1)
	beforeGas, beforeBytes, err := GasUsed(tx, p)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.SignTransactionWithProfile(&tx, p); err != nil {
		t.Fatal(err)
	}
	afterGas, afterBytes, err := GasUsed(tx, p)
	if err != nil {
		t.Fatal(err)
	}
	if beforeGas != afterGas || beforeBytes != afterBytes {
		t.Fatalf("gas changed after signing: before=%v/%d after=%v/%d", beforeGas, beforeBytes, afterGas, afterBytes)
	}
}
