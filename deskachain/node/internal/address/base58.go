package address

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math/big"
)

const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

var bigRadix = big.NewInt(58)

func Base58Encode(input []byte) string {
	if len(input) == 0 {
		return ""
	}
	x := new(big.Int).SetBytes(input)
	zero := big.NewInt(0)
	mod := new(big.Int)
	var out []byte
	for x.Cmp(zero) > 0 {
		x.DivMod(x, bigRadix, mod)
		out = append(out, alphabet[mod.Int64()])
	}
	for _, b := range input {
		if b != 0 {
			break
		}
		out = append(out, alphabet[0])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

func Base58Decode(input string) ([]byte, error) {
	if input == "" {
		return nil, nil
	}
	x := big.NewInt(0)
	for _, r := range input {
		index := bytes.IndexRune([]byte(alphabet), r)
		if index < 0 {
			return nil, errors.New("invalid base58 character")
		}
		x.Mul(x, bigRadix)
		x.Add(x, big.NewInt(int64(index)))
	}
	out := x.Bytes()
	leading := 0
	for _, r := range input {
		if byte(r) != alphabet[0] {
			break
		}
		leading++
	}
	if leading > 0 {
		out = append(bytes.Repeat([]byte{0x00}, leading), out...)
	}
	return out, nil
}

func Base58CheckEncode(payload []byte) string {
	raw := make([]byte, 0, len(payload)+4)
	raw = append(raw, payload...)
	raw = append(raw, checksum(payload)...)
	return Base58Encode(raw)
}

func Base58CheckDecode(encoded string) ([]byte, error) {
	raw, err := Base58Decode(encoded)
	if err != nil {
		return nil, err
	}
	if len(raw) < 5 {
		return nil, errors.New("payload too short")
	}
	payload := raw[:len(raw)-4]
	got := raw[len(raw)-4:]
	want := checksum(payload)
	if !bytes.Equal(got, want) {
		return nil, errors.New("checksum mismatch")
	}
	return payload, nil
}

func checksum(payload []byte) []byte {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	return second[:4]
}
