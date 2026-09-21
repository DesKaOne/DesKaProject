package crypto

import (
	"encoding/hex"
	"fmt"

	indaddress "indochain/internal/address"
	"indochain/internal/config"
)

const AddressPrefix = "iND"

func AddressFromPublicKey(publicKeyHex string) string {
	addr, err := AddressFromPublicKeyForNetwork(publicKeyHex, config.Localnet())
	if err != nil {
		return ""
	}
	return addr
}

func AddressFromPublicKeyForNetwork(publicKeyHex string, profile config.NetworkConfig) (string, error) {
	raw, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", err
	}
	return indaddress.EncodeAddress(raw, profile)
}

func ValidateAddress(address string) error {
	return ValidateAddressForNetwork(address, config.Localnet())
}

func ValidateAddressForNetwork(address string, profile config.NetworkConfig) error {
	if _, err := indaddress.DecodeAddress(address, profile); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	return nil
}

func IsLegacyDevAddress(address string) bool {
	return indaddress.IsLegacyDevAddress(address)
}
