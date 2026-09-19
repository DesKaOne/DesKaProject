package crypto

import (
	"strings"
	"testing"
)

func TestPrivateKeyHexRoundTrip(t *testing.T) {
	priv, err := GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	hexKey := PrivateKeyToHex(priv)
	if len(hexKey) != 64 {
		t.Fatalf("private key length = %d", len(hexKey))
	}
	imported, err := PrivateKeyFromHex(hexKey)
	if err != nil {
		t.Fatal(err)
	}
	if PublicKeyToHex(priv.PubKey()) != PublicKeyToHex(imported.PubKey()) {
		t.Fatal("imported public key mismatch")
	}
}

func TestPrivateKeyInvalid(t *testing.T) {
	if _, err := PrivateKeyFromHex("abcd"); err == nil {
		t.Fatal("expected invalid length")
	}
	if _, err := PrivateKeyFromHex(strings.Repeat("0", 64)); err == nil {
		t.Fatal("expected invalid scalar")
	}
}

func TestSecp256k1SignVerify(t *testing.T) {
	priv, err := GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("message")
	sig, err := SignHex(priv, payload)
	if err != nil {
		t.Fatal(err)
	}
	pub := PublicKeyToHex(priv.PubKey())
	if !VerifyHex(pub, sig, payload) {
		t.Fatal("signature did not verify")
	}
	if VerifyHex(pub, sig, []byte("tampered")) {
		t.Fatal("tampered payload verified")
	}
	other, _ := GeneratePrivateKey()
	if VerifyHex(PublicKeyToHex(other.PubKey()), sig, payload) {
		t.Fatal("wrong public key verified")
	}
}
