package types

import "testing"

func TestMerkleRootChangesWhenTransactionsChange(t *testing.T) {
	first := NewCoinbaseTransaction("idr10000000000000000000000000000000000000000", 1, 1)
	second := first
	second.Amount = 2
	second.RefreshID()
	if CalculateMerkleRoot([]Transaction{first}) == CalculateMerkleRoot([]Transaction{second}) {
		t.Fatal("merkle root did not change after transaction changed")
	}
}
