package state

import (
	"testing"

	"indochain/internal/asset"
	"indochain/internal/ledger"
)

func TestStateRootIncludesIssuedAssetAndBalances(t *testing.T) {
	accounts := []ledger.StateAccount{{Address: "alice", Confirmed: 100, Mature: 100, Nonce: 1}}
	assets := []asset.Definition{
		{ID: asset.NativeAssetID, Name: "dIDR", Symbol: "dIDR", Decimals: 0, Kind: asset.KindFungible, Issuer: "protocol", Status: asset.StatusActive},
		{ID: "asset:usd", Name: "Example USD", Symbol: "EUSD", Decimals: 6, Kind: asset.KindFungible, Issuer: "alice", Mintable: true, Burnable: true, Status: asset.StatusActive},
	}
	balances := []asset.BalanceEntry{
		{Address: "alice", AssetID: asset.NativeAssetID, Amount: 100},
		{Address: "alice", AssetID: "asset:usd", Amount: 500},
	}
	rootA, err := RootForCollectionsWithAssets(accounts, nil, assets, balances)
	if err != nil {
		t.Fatal(err)
	}

	balances[1].Amount = 501
	rootB, err := RootForCollectionsWithAssets(accounts, nil, assets, balances)
	if err != nil {
		t.Fatal(err)
	}
	if rootA == rootB {
		t.Fatal("asset balance mutation did not change state root")
	}
}
