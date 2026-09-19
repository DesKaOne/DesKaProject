package state

import (
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

	if err := l.ApplyCoinbaseAtHeight(types.NewCoinbaseTransactionWithVersion(w.Address, 100, 1, types.TxVersionAsset), 1); err != nil {
		t.Fatal(err)
	}
	create := types.NewAssetCreateTransaction(w.Address, "Example", "EXT", 6, 1000, true, true, false, false, 1, 1)
	if err := w.SignTransactionWithProfile(&create, profile); err != nil { t.Fatal(err) }
	if err := l.ApplyTransactionAtHeight(create, 2); err != nil { t.Fatal(err) }

	mint := types.NewAssetMintTransaction(w.Address, create.AssetID, w.Address, 10, 1, 2)
	if err := w.SignTransactionWithProfile(&mint, profile); err != nil { t.Fatal(err) }
	if err := l.ApplyTransactionAtHeight(mint, 3); err != nil { t.Fatal(err) }

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
