package config

import "testing"

func TestNativeAssetAndFeePolicy(t *testing.T) {
	for _, profile := range []NetworkConfig{Localnet(), Testnet(), Mainnet()} {
		if profile.Asset.NativeAssetID != "dIDR" || profile.Asset.NativeAssetSymbol != "dIDR" {
			t.Fatalf("%s native asset = %#v", profile.Name, profile.Asset)
		}
		const expectedDecimals = 8
		if profile.Asset.NativeAssetDecimals != expectedDecimals || profile.Asset.FeeAssetID != "dIDR" {
			t.Fatalf("%s native decimals/fee asset = %#v", profile.Name, profile.Asset)
		}
		if !profile.Fee.Enabled || profile.Fee.MinFee == 0 || profile.Fee.BytesPerGas == 0 || profile.Fee.MaxGasPerTx == 0 {
			t.Fatalf("%s fee policy is not configured: %#v", profile.Name, profile.Fee)
		}
		if !profile.Asset.TokenTransfersEnabled || !profile.Asset.UserIssuedTokensEnabled || !profile.Asset.PaymasterEnabled {
			t.Fatalf("%s asset features not enabled", profile.Name)
		}
		if profile.Economic.BlockSubsidy != 0 || !profile.Economic.FeeOnlyBlocks {
			t.Fatalf("%s economic policy is not fee-only", profile.Name)
		}
	}
}
