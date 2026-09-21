package wallet

import (
	indaddress "indochain/internal/address"
	"indochain/internal/config"
)

func indochainAddressFromPublicKey(publicKey string, profile config.NetworkConfig) (string, error) {
	return indaddress.EncodeAddress([]byte(publicKey), profile)
}

func validateAddressForNetwork(address string, profile config.NetworkConfig) error {
	return indaddressValidation(address, profile)
}

func indaddressValidation(address string, profile config.NetworkConfig) error {
	_, err := indaddress.DecodeAddress(address, profile)
	return err
}
