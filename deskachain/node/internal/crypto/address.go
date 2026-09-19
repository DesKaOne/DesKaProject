package crypto

import (
	"encoding/hex"
	"fmt"

	dkcaddress "deskachain/internal/address"
	"deskachain/internal/config"
)

const AddressPrefix = "DKC"

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
	return dkcaddress.EncodeAddress(raw, profile)
}

func ValidateAddress(address string) error {
	return ValidateAddressForNetwork(address, config.Localnet())
}

func ValidateAddressForNetwork(address string, profile config.NetworkConfig) error {
	if _, err := dkcaddress.DecodeAddress(address, profile); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	return nil
}

func IsLegacyDevAddress(address string) bool {
	return dkcaddress.IsLegacyDevAddress(address)
}
