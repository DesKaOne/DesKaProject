package address

import (
	"bytes"
	"testing"
)

func TestBase58CheckRoundTrip(t *testing.T) {
	payload := []byte{0x1E, 1, 2, 3, 4, 5}
	encoded := Base58CheckEncode(payload)
	decoded, err := Base58CheckDecode(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatalf("decoded = %x want %x", decoded, payload)
	}
}

func TestBase58InvalidChar(t *testing.T) {
	if _, err := Base58Decode("0OIl"); err == nil {
		t.Fatal("expected invalid char error")
	}
}

func TestBase58CheckChecksumMismatch(t *testing.T) {
	encoded := Base58CheckEncode([]byte{0x1E, 1, 2, 3})
	encoded = flipLast(encoded)
	if _, err := Base58CheckDecode(encoded); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestBase58LeadingZeroRoundTrip(t *testing.T) {
	payload := []byte{0, 0, 1, 2, 3}
	encoded := Base58Encode(payload)
	decoded, err := Base58Decode(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatalf("decoded = %x want %x", decoded, payload)
	}
}

func flipLast(value string) string {
	if value[len(value)-1] == '1' {
		return value[:len(value)-1] + "2"
	}
	return value[:len(value)-1] + "1"
}
