package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	btcecECDSA "github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

func GeneratePrivateKey() (*btcec.PrivateKey, error) {
	return btcec.NewPrivateKey()
}

func PrivateKeyToHex(key *btcec.PrivateKey) string {
	return hex.EncodeToString(key.Serialize())
}

func PrivateKeyFromHex(value string) (*btcec.PrivateKey, error) {
	if len(value) != 64 {
		return nil, errors.New("invalid private key length")
	}
	raw, err := hex.DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid private key hex")
	}
	priv, _ := btcec.PrivKeyFromBytes(raw)
	if priv.Key.IsZero() {
		return nil, errors.New("private key out of range")
	}
	if PrivateKeyToHex(priv) != strings.ToLower(value) {
		return nil, errors.New("private key out of range")
	}
	return priv, nil
}

func PublicKeyBytes(key *btcec.PrivateKey) []byte {
	return key.PubKey().SerializeCompressed()
}

func PublicKeyToHex(key *btcec.PublicKey) string {
	return hex.EncodeToString(key.SerializeCompressed())
}

func PublicKeyFromHex(value string) (*btcec.PublicKey, error) {
	raw, err := hex.DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid public key hex")
	}
	if len(raw) != 33 {
		return nil, errors.New("invalid public key length")
	}
	pub, err := btcec.ParsePubKey(raw)
	if err != nil {
		return nil, err
	}
	return pub, nil
}

func SignHex(privateKey *btcec.PrivateKey, payload []byte) (string, error) {
	digest := sha256.Sum256(payload)
	sig := btcecECDSA.Sign(privateKey, digest[:])
	return hex.EncodeToString(sig.Serialize()), nil
}

func VerifyHex(publicKeyHex, signatureHex string, payload []byte) bool {
	publicKey, err := PublicKeyFromHex(publicKeyHex)
	if err != nil {
		return false
	}
	rawSig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	sig, err := btcecECDSA.ParseDERSignature(rawSig)
	if err != nil {
		return false
	}
	digest := sha256.Sum256(payload)
	return sig.Verify(digest[:], publicKey)
}
