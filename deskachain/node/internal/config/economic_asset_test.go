package config

import "testing"

func TestNativeAssetAndFeePolicy(t *testing.T) {
	for _, profile := range []NetworkConfig{Localnet(), Testnet(), Mainnet()} {
		if profile.Asset.NativeAssetID != "IDR" || profile.Asset.NativeAssetSymbol != "IDR" {
			t.Fatalf("%s native asset = %#v", profile.Name, profile.Asset)
		}
		if profile.Asset.NativeAssetDecimals != 0 || profile.Asset.FeeAssetID != "IDR" {
			t.Fatalf("%s native decimals/fee asset = %#v", profile.Name, profile.Asset)
		}
		if !profile.Asset.TokenTransfersEnabled || !profile.Asset.UserIssuedTokensEnabled || !profile.Asset.PaymasterEnabled {
			t.Fatalf("%s asset features not enabled", profile.Name)
		}
		if profile.Economic.BlockSubsidy != 0 || !profile.Economic.FeeOnlyBlocks {
			t.Fatalf("%s economic policy is not fee-only", profile.Name)
		}
	}
}
