package asset

import (
	"testing"

	"indochain/internal/config"
)

func TestNativeAssetDefinition(t *testing.T) {
	if NativeAssetID != "dIDR" || NativeSymbol != "dIDR" || NativeDecimals != config.NativeAssetDecimals {
		t.Fatal("native asset constants changed")
	}
}

func TestDerivedAssetID(t *testing.T) {
	if got := DerivedID("ABCDEF"); got != "asset:abcdef" {
		t.Fatalf("derived asset id = %q", got)
	}
	if IsNative("asset:abcdef") {
		t.Fatal("issued asset id treated as native")
	}
}

func TestValidateDefinition(t *testing.T) {
	valid := Definition{ID: "asset:abc", Name: "Example USD", Symbol: "EUSD", Decimals: 6, Kind: KindFungible, Issuer: "issuer", Mintable: true, Burnable: true, Status: StatusActive}
	if err := ValidateDefinition(valid); err != nil {
		t.Fatal(err)
	}
	bad := valid
	bad.ID = NativeAssetID
	if err := ValidateDefinition(bad); err == nil {
		t.Fatal("native asset accepted as issued asset")
	}
}
