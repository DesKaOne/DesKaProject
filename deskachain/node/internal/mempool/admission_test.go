package mempool

import (
	"testing"

	"indochain/internal/config"
	"indochain/internal/fees"
	"indochain/internal/types"
	"indochain/internal/wallet"
)

func admissionProfile() config.NetworkConfig {
	p := config.Localnet()
	p.TxVersion = types.TxVersionAsset
	p.Fee.MinFee = 1
	p.Fee.MinGasPrice = 0
	p.Fee.MaxGasPerTx = 100000
	return p
}

func signedTransfer(t *testing.T, p config.NetworkConfig, w wallet.Wallet, fee, nonce uint64) types.Transaction {
	t.Helper()
	tx := types.NewAssetTransferTransaction(w.Address, "recipient", p.Asset.NativeAssetID, 1, fee, nonce)
	if err := w.SignTransactionWithProfile(&tx, p); err != nil {
		t.Fatal(err)
	}
	return tx
}

func TestAdmitRejectsLowFee(t *testing.T) {
	p := admissionProfile()
	p.Fee.MinFee = 7
	w, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	tx := signedTransfer(t, p, w, 1, 1)
	m := New(t.TempDir() + "/mempool.json")
	if err := m.Admit(tx, AdmissionPolicy{Profile: p, MaxTxs: 10, MaxGas: 1000}); err == nil {
		t.Fatal("low-fee transaction accepted")
	}
}

func TestAdmitRejectsDuplicateNonce(t *testing.T) {
	p := admissionProfile()
	w, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	m := New(t.TempDir() + "/mempool.json")
	tx1 := signedTransfer(t, p, w, 1, 1)
	tx2 := signedTransfer(t, p, w, 2, 1)
	policy := AdmissionPolicy{Profile: p, MaxTxs: 10, MaxGas: 1000}
	if err := m.Admit(tx1, policy); err != nil {
		t.Fatal(err)
	}
	if err := m.Admit(tx2, policy); err == nil {
		t.Fatal("duplicate nonce accepted")
	}
}

func TestAdmitRejectsGasCapacity(t *testing.T) {
	p := admissionProfile()
	w, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	m := New(t.TempDir() + "/mempool.json")
	tx := signedTransfer(t, p, w, 1, 1)
	gas, _, err := fees.GasUsed(tx, p)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Admit(tx, AdmissionPolicy{Profile: p, MaxTxs: 10, MaxGas: gas - 1}); err == nil {
		t.Fatal("gas-capacity overflow accepted")
	}
}

func TestSelectDeterministicFeePerGas(t *testing.T) {
	p := admissionProfile()
	w1, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	w2, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	low := signedTransfer(t, p, w1, 1, 1)
	high := signedTransfer(t, p, w2, 10, 1)
	selected, err := Select([]types.Transaction{low, high}, p, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 || selected[0].ID != high.ID {
		t.Fatalf("unexpected selection order: %+v", selected)
	}
}
