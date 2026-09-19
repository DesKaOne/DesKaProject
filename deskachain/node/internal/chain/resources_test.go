package chain

import (
	"encoding/json"
	"strings"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/types"
)

func TestValidateTransactionSizeUsesConfiguredBoundary(t *testing.T) {
	tx := types.Transaction{ID: "id", From: "from", To: "to", Amount: 1}
	raw, err := json.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}

	params := config.Localnet().Consensus
	params.MaxTxBytes = uint64(len(raw))
	if err := ValidateTransactionSize(tx, params); err != nil {
		t.Fatalf("exact transaction size rejected: %v", err)
	}

	params.MaxTxBytes = uint64(len(raw) - 1)
	if err := ValidateTransactionSize(tx, params); err == nil || !strings.Contains(err.Error(), "exceeds max size") {
		t.Fatalf("expected transaction size rejection, got %v", err)
	}
}

func TestValidateBlockResourcesRejectsTxCount(t *testing.T) {
	params := config.Localnet().Consensus
	params.MaxTxCount = 1

	block := types.Block{
		Height: 0,
		Transactions: []types.Transaction{
			{ID: "a"},
			{ID: "b"},
		},
	}
	err := ValidateBlockResources(block, params)
	if err == nil || !strings.Contains(err.Error(), "max transaction count") {
		t.Fatalf("expected tx count rejection, got %v", err)
	}
}

func TestValidateBlockResourcesRejectsOversizedTransaction(t *testing.T) {
	params := config.Localnet().Consensus
	params.MaxTxBytes = 32

	block := types.Block{
		Height: 0,
		Transactions: []types.Transaction{
			{ID: "this transaction is deliberately oversized"},
		},
	}
	err := ValidateBlockResources(block, params)
	if err == nil || !strings.Contains(err.Error(), "tx 0") || !strings.Contains(err.Error(), "exceeds max size") {
		t.Fatalf("expected oversized transaction rejection, got %v", err)
	}
}

func TestValidateBlockResourcesRejectsOversizedBlock(t *testing.T) {
	params := config.Localnet().Consensus
	params.MaxBlockBytes = 16

	block := types.Block{
		Height: 0,
		Transactions: []types.Transaction{{ID: "a"}},
	}
	err := ValidateBlockResources(block, params)
	if err == nil || !strings.Contains(err.Error(), "block exceeds max size") {
		t.Fatalf("expected oversized block rejection, got %v", err)
	}
}

func TestValidateBlockResourcesRejectsExcessGas(t *testing.T) {
	profile := config.Localnet()
	profile.Fee.BaseGasTransfer = 10
	profile.Fee.BaseGasAssetTransfer = 10
	profile.Fee.BytesPerGas = 1
	profile.Fee.MaxGasPerTx = 100000

	profile.Consensus.MaxGasPerBlock = 15

	block := types.Block{
		Height: 0,
		Transactions: []types.Transaction{
			{Version: types.TxVersionAsset, ID: "a", From: "from", To: "to", Amount: 1, Fee: 1},
			{Version: types.TxVersionAsset, ID: "b", From: "from", To: "to", Amount: 1, Fee: 1},
		},
	}
	if err := ValidateBlockResourcesWithProfile(block, profile); err == nil || !strings.Contains(err.Error(), "max gas") {
		t.Fatalf("expected block gas rejection, got %v", err)
	}
}
