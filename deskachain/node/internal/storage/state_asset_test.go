package storage

import (
	"testing"

	"indochain/internal/asset"
	"indochain/internal/config"
	"indochain/internal/ledger"
	"indochain/internal/state"
)

func TestStateStorePersistsMultiAssetState(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snapshot := state.Snapshot{
		Version:  state.SnapshotVersion,
		Height:   7,
		Accounts: []ledger.StateAccount{{Address: "iND-alice", Confirmed: 100, Mature: 100, Nonce: 3}},
		Assets: []asset.Definition{
			{ID: asset.NativeAssetID, Name: asset.NativeSymbol, Symbol: asset.NativeSymbol, Decimals: asset.NativeDecimals, Kind: asset.KindFungible, Issuer: "protocol", Status: asset.StatusActive},
			{ID: "asset:usd", Name: "Example USD", Symbol: "EUSD", Decimals: 6, Kind: asset.KindFungible, Issuer: "iND-alice", Mintable: true, Burnable: true, Status: asset.StatusActive},
		},
		AssetBalances: []asset.BalanceEntry{
			{Address: "iND-alice", AssetID: asset.NativeAssetID, Amount: 100},
			{Address: "iND-alice", AssetID: "asset:usd", Amount: 2500},
		},
	}
	snapshot.StateRoot, err = state.RootForCollectionsWithAssets(snapshot.Accounts, snapshot.Stakes, snapshot.Assets, snapshot.AssetBalances)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}

	got, err := store.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.Equivalent(snapshot, got) {
		t.Fatalf("multi-asset snapshot mismatch: %#v vs %#v", snapshot, got)
	}

	def, found, err := store.GetStateAsset("asset:usd")
	if err != nil || !found {
		t.Fatalf("asset query failed: found=%v err=%v", found, err)
	}
	if def.Symbol != "EUSD" || def.Decimals != 6 {
		t.Fatalf("unexpected asset: %#v", def)
	}

	balance, found, err := store.GetStateAssetBalance("iND-alice", "asset:usd")
	if err != nil || !found || balance != 2500 {
		t.Fatalf("unexpected token balance: balance=%d found=%v err=%v", balance, found, err)
	}

	entries, err := store.GetStateAssetBalancesForAddress("iND-alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].AssetID != "asset:usd" || entries[1].AssetID != asset.NativeAssetID {
		t.Fatalf("unexpected address asset balances: %#v", entries)
	}

	if err := store.ValidateStateIndexes(); err != nil {
		t.Fatal(err)
	}

	profile := config.Localnet()
	_ = profile
}
