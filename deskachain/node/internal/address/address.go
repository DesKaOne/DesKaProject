package address

import (
	"crypto/sha256"
	"errors"
	"strings"

	"indochain/internal/config"

	"golang.org/x/crypto/ripemd160"
)

const legacyDevPrefix = "iND1"

type AddressPayload struct {
	Version    byte
	PubKeyHash []byte
	Legacy     bool
}

func EncodeAddress(pubKeyBytes []byte, profile config.NetworkConfig) (string, error) {
	if len(pubKeyBytes) == 0 {
		return "", errors.New("public key is required")
	}
	payload := append([]byte{profile.AddressVersion}, Hash160(pubKeyBytes)...)
	return profile.AddressPrefix + Base58CheckEncode(payload), nil
}

func DecodeAddress(addr string, profile config.NetworkConfig) (AddressPayload, error) {
	if IsLegacyDevAddress(addr) {
		if !profile.LegacyAddressAllowed {
			return AddressPayload{}, errors.New("unsupported legacy address")
		}
		return AddressPayload{Legacy: true}, nil
	}
	if !strings.HasPrefix(addr, profile.AddressPrefix) {
		return AddressPayload{}, errors.New("wrong address prefix")
	}
	if strings.HasPrefix(strings.ToLower(addr), strings.ToLower(profile.AddressPrefix)) && !strings.HasPrefix(addr, profile.AddressPrefix) {
		return AddressPayload{}, errors.New("wrong address prefix")
	}
	body := strings.TrimPrefix(addr, profile.AddressPrefix)
	payload, err := Base58CheckDecode(body)
	if err != nil {
		return AddressPayload{}, err
	}
	if len(payload) != 21 {
		return AddressPayload{}, errors.New("payload length is invalid")
	}
	if payload[0] != profile.AddressVersion {
		return AddressPayload{}, errors.New("wrong network version")
	}
	return AddressPayload{Version: payload[0], PubKeyHash: append([]byte(nil), payload[1:]...)}, nil
}

func ValidateAddress(addr string, profile config.NetworkConfig) bool {
	_, err := DecodeAddress(addr, profile)
	return err == nil
}

func IsLegacyDevAddress(addr string) bool {
	if !strings.HasPrefix(addr, legacyDevPrefix) {
		return false
	}
	if len(addr) != len(legacyDevPrefix)+40 {
		return false
	}
	for _, r := range addr[len(legacyDevPrefix):] {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return false
		}
	}
	return true
}

func Hash160(pubKeyBytes []byte) []byte {
	sha := sha256.Sum256(pubKeyBytes)
	r := ripemd160.New()
	_, _ = r.Write(sha[:])
	return r.Sum(nil)
}
