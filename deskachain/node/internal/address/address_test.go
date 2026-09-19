package address

import (
	"testing"

	"deskachain/internal/config"
)

func TestAddressEncodeValidate(t *testing.T) {
	addr, err := EncodeAddress([]byte{2, 1, 2, 3}, config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	if addr[:3] != "DKC" {
		t.Fatalf("address = %s", addr)
	}
	if !ValidateAddress(addr, config.Localnet()) {
		t.Fatalf("address did not validate: %s", addr)
	}
}

func TestAddressWrongPrefixAndChecksum(t *testing.T) {
	addr, err := EncodeAddress([]byte{2, 1, 2, 3}, config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	if ValidateAddress("dkc"+addr[3:], config.Localnet()) {
		t.Fatal("lowercase prefix should be invalid for new address")
	}
	bad := flipLast(addr)
	if ValidateAddress(bad, config.Localnet()) {
		t.Fatal("wrong checksum should be invalid")
	}
}

func TestAddressWrongNetworkVersion(t *testing.T) {
	addr, err := EncodeAddress([]byte{2, 1, 2, 3}, config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	if ValidateAddress(addr, config.Testnet()) {
		t.Fatal("localnet address should not validate on testnet")
	}
}

func TestLegacyDevAddressPolicy(t *testing.T) {
	legacy := "dkc10000000000000000000000000000000000000000"
	if !ValidateAddress(legacy, config.Localnet()) {
		t.Fatal("legacy localnet address should validate")
	}
	if ValidateAddress(legacy, config.Testnet()) {
		t.Fatal("legacy testnet address should be invalid")
	}
}
