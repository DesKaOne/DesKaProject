package ledger

import (
	"testing"

	"deskachain/internal/asset"
	"deskachain/internal/config"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func feeTestProfile() config.NetworkConfig {
	p := config.Localnet()
	p.TxVersion = types.TxVersionAsset
	p.BlockVersion = types.BlockVersionCanonical
	p.Consensus.CoinbaseMaturity = 0
	return p
}

func TestV3FeesSettleToMinerWithoutIssuance(t *testing.T) {
	profile := feeTestProfile()
	miner, err := wallet.NewWithProfile(profile)
	if err != nil { t.Fatal(err) }
	sender, err := wallet.NewWithProfile(profile)
	if err != nil { t.Fatal(err) }
	receiver, err := wallet.NewWithProfile(profile)
	if err != nil { t.Fatal(err) }

	l := NewMatureWithProfile(profile.Consensus, profile)
	if err := l.ApplyBlock(types.Block{
		Height:      1,
		MinerAddress: miner.Address,
		Transactions: []types.Transaction{types.NewCoinbaseTransactionWithVersion(sender.Address, 100, 1, types.TxVersionAsset)},
	}); err != nil {
		t.Fatal(err)
	}

	tx := types.NewAssetTransferTransaction(sender.Address, receiver.Address, asset.NativeAssetID, 10, 7, 1)
	if err := sender.SignTransactionWithProfile(&tx, profile); err != nil {
		t.Fatal(err)
	}
	block := types.Block{
		Height:      2,
		MinerAddress: miner.Address,
		Transactions: []types.Transaction{types.NewCoinbaseTransactionWithVersion(miner.Address, 0, 2, types.TxVersionAsset), tx},
	}
	if err := l.ApplyBlock(block); err != nil {
		t.Fatal(err)
	}

	if got := l.Balance(sender.Address); got != 83 {
		t.Fatalf("sender balance=%d want 83", got)
	}
	if got := l.Balance(receiver.Address); got != 10 {
		t.Fatalf("receiver balance=%d want 10", got)
	}
	if got := l.Balance(miner.Address); got != 7 {
		t.Fatalf("miner fee balance=%d want 7", got)
	}
	if got := l.AssetBalance(asset.FeeCollectorAddress, asset.NativeAssetID); got != 0 {
		t.Fatalf("fee pool balance=%d want 0", got)
	}
}
