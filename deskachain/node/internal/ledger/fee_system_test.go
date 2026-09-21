package ledger

import (
	"testing"

	"indochain/internal/asset"
	"indochain/internal/config"
	"indochain/internal/types"
	"indochain/internal/wallet"
)

func feeTestProfile() config.NetworkConfig {
	p := config.Localnet()
	p.TxVersion = types.TxVersionAsset
	p.BlockVersion = types.BlockVersionCanonical
	p.Consensus.CoinbaseMaturity = 0
	p.Fee.MinGasPrice = 0
	p.Fee.MinFee = 1
	return p
}

func TestV3FeePoolSettlesToBlockProducer(t *testing.T) {
	p := feeTestProfile()
	l := NewMatureWithProfile(p.Consensus, p)
	sender, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	miner, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}

	for _, recipient := range []string{sender.Address} {
		if err := l.ApplyCoinbaseAtHeight(types.NewCoinbaseTransactionWithVersion(recipient, 10, 1, types.TxVersionAsset), 1); err != nil {
			t.Fatal(err)
		}
	}
	l.matureCoinbases(1)
	l.syncNativeAssetsFromAccounts()

	tx := types.NewAssetTransferTransaction(sender.Address, receiver.Address, asset.NativeAssetID, 5, 1, 1)
	if err := sender.SignTransactionWithProfile(&tx, p); err != nil {
		t.Fatal(err)
	}
	if err := l.ApplyTransactionAtHeight(tx, 2); err != nil {
		t.Fatal(err)
	}
	if got := l.FeePoolBalance(); got != 1 {
		t.Fatalf("fee pool=%d want 1", got)
	}

	beforeMiner := l.Balance(miner.Address)
	if err := l.ApplyBlock(types.Block{Height: 3, MinerAddress: miner.Address}); err != nil {
		t.Fatal(err)
	}
	if got := l.FeePoolBalance(); got != 0 {
		t.Fatalf("fee pool after settlement=%d want 0", got)
	}
	if got := l.Balance(miner.Address); got != beforeMiner+1 {
		t.Fatalf("miner fee balance=%d want=%d", got, beforeMiner+1)
	}
}

func TestV3ApplyBlockRollsBackFeePoolOnSettlementFailure(t *testing.T) {
	p := feeTestProfile()
	l := NewMatureWithProfile(p.Consensus, p)
	sender, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := wallet.NewWithProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.ApplyCoinbaseAtHeight(types.NewCoinbaseTransactionWithVersion(sender.Address, 10, 1, types.TxVersionAsset), 1); err != nil {
		t.Fatal(err)
	}
	l.matureCoinbases(1)
	l.syncNativeAssetsFromAccounts()

	tx := types.NewAssetTransferTransaction(sender.Address, receiver.Address, asset.NativeAssetID, 5, 1, 1)
	if err := sender.SignTransactionWithProfile(&tx, p); err != nil {
		t.Fatal(err)
	}
	if err := l.ApplyTransactionAtHeight(tx, 2); err != nil {
		t.Fatal(err)
	}
	if l.FeePoolBalance() != 1 {
		t.Fatal("expected pending fee")
	}
	beforeSender := l.Balance(sender.Address)
	err = l.ApplyBlock(types.Block{Height: 3, MinerAddress: "invalid"})
	if err == nil {
		t.Fatal("invalid fee recipient unexpectedly accepted")
	}
	if l.FeePoolBalance() != 1 {
		t.Fatalf("fee pool=%d want 1 after rollback", l.FeePoolBalance())
	}
	if l.Balance(sender.Address) != beforeSender {
		t.Fatalf("sender balance changed after failed block")
	}
}
