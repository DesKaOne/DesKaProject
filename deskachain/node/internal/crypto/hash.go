package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func DoubleSHA256Hex(data []byte) string {
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	return hex.EncodeToString(second[:])
}
