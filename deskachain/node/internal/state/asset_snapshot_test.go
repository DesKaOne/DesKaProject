package state

import (
	"deskachain/internal/asset"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestV3SnapshotRoundTripKeepsAssets(t *testing.T) {
	profile := config.Localnet()
	profile.TxVersion = types.TxVersionAsset
	profile.BlockVersion = types.BlockVersionCanonical
	profile.Consensus.CoinbaseMaturity = 0

	l := ledger.NewMatureWithProfile(profile.Consensus, profile)
	w, err := wallet.NewWithProfile(profile)
	if err != nil { t.Fatal(err) }

	coinbase := types.NewCoinbaseTransactionWithVersion(w.Address, 100, 1, types.TxVersionAsset)
	if err := l.ApplyBlock(types.Block{Height: 1, MinerAddress: w.Address, Transactions: []types.Transaction{coinbase}}); err != nil { t.Fatal(err) }
	create := types.NewAssetCreateTransaction(w.Address, "Example", "EXT", 6, 1000, true, true, false, false, 1, 1)
	if err := w.SignTransactionWithProfile(&create, profile); err != nil { t.Fatal(err) }
	if err := l.ApplyBlock(types.Block{Height: 2, MinerAddress: w.Address, Transactions: []types.Transaction{create}}); err != nil { t.Fatal(err) }

	mint := types.NewAssetMintTransaction(w.Address, create.AssetID, w.Address, 10, 1, 2)
	if err := w.SignTransactionWithProfile(&mint, profile); err != nil { t.Fatal(err) }
	if err := l.ApplyBlock(types.Block{Height: 3, MinerAddress: w.Address, Transactions: []types.Transaction{mint}}); err != nil { t.Fatal(err) }

	snapshot, err := SnapshotForLedger(l)
	if err != nil { t.Fatal(err) }
	if len(snapshot.Assets) < 2 || len(snapshot.AssetBalances) == 0 {
		t.Fatalf("asset state missing from snapshot: %#v", snapshot)
	}
	restored, err := ledger.NewMatureFromStateWithAssets(profile.Consensus, profile, snapshot.Height, snapshot.Accounts, snapshot.Stakes, snapshot.Coinbases, snapshot.Assets, snapshot.AssetBalances)
	if err != nil { t.Fatal(err) }
	restoredSnapshot, err := SnapshotForLedger(restored)
	if err != nil { t.Fatal(err) }
	if !Equivalent(snapshot, restoredSnapshot) {
		t.Fatal("restored v3 state is not equivalent")
	}
}

func TestV3SnapshotRejectsNativeAccountBalanceMismatch(t *testing.T) {
	profile := config.Localnet()
	profile.TxVersion = types.TxVersionAsset
	l := ledger.NewMatureWithProfile(profile.Consensus, profile)
	w, err := wallet.NewWithProfile(profile)
	if err != nil { t.Fatal(err) }

	coinbase := types.NewCoinbaseTransactionWithVersion(w.Address, 100, 1, types.TxVersionAsset)
	if err := l.ApplyBlock(types.Block{Height: 1, MinerAddress: w.Address, Transactions: []types.Transaction{coinbase}}); err != nil { t.Fatal(err) }
	snapshot, err := SnapshotForLedger(l)
	if err != nil { t.Fatal(err) }
	for i := range snapshot.AssetBalances {
		if snapshot.AssetBalances[i].Address == w.Address && asset.IsNative(snapshot.AssetBalances[i].AssetID) {
			snapshot.AssetBalances[i].Amount++
			break
		}
	}
	if err := snapshot.Validate(); err == nil {
		t.Fatal("snapshot accepted mismatched native IDR/account balance")
	}
}
