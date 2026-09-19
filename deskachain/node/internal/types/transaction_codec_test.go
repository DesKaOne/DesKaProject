package types

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

func TestLegacyTransactionVersionDoesNotChangeSigningBytes(t *testing.T) {
	withoutVersion := Transaction{
		From:      "from",
		To:        "to",
		Amount:    123,
		Fee:       7,
		Nonce:     9,
		Timestamp: 1700000000,
		PublicKey: "pub",
	}
	withVersion := withoutVersion
	withVersion.Version = TxVersionLegacy
	if !bytes.Equal(withoutVersion.SigningBytes(), withVersion.SigningBytes()) {
		t.Fatal("explicit legacy version changed signing bytes")
	}
	if withoutVersion.CalculateID() != withVersion.CalculateID() {
		t.Fatal("explicit legacy version changed transaction id")
	}
}

func TestLegacyTransactionSigningBytesRemainVersionCompatible(t *testing.T) {
	tx := Transaction{
		From:      "from",
		To:        "to",
		Amount:    123,
		Fee:       7,
		Nonce:     9,
		Timestamp: 1700000000,
		PublicKey: "pub",
	}
	got := string(tx.SigningBytes())
	if strings.Contains(got, "\"version\"") {
		t.Fatalf("legacy signing bytes unexpectedly contain version: %s", got)
	}
	want := "{\"id\":\"\",\"from\":\"from\",\"to\":\"to\",\"amount\":123,\"fee\":7,\"nonce\":9,\"timestamp\":1700000000,\"signature\":\"\",\"public_key\":\"pub\",\"coinbase\":false}"
	if got != want {
		t.Fatalf("legacy signing bytes changed: got=%s want=%s", got, want)
	}
}

func TestCanonicalTransactionEncodingRoundTrip(t *testing.T) {
	tx := Transaction{
		Version:   TxVersionCanonical,
		ID:        "abcd",
		From:      "from",
		To:        "to",
		Amount:    123,
		Fee:       7,
		Nonce:     9,
		Timestamp: 1700000000,
		Signature: "signature",
		PublicKey: "public-key",
		Type:      TxTypeTransfer,
	}
	raw1, err := tx.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	raw2, err := tx.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw1, raw2) {
		t.Fatal("canonical encoding is not deterministic")
	}
	decoded, err := DecodeCanonicalTransaction(raw1)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != tx {
		t.Fatalf("round-trip mismatch: got=%+v want=%+v", decoded, tx)
	}
	signing := tx.CanonicalSigningBytes()
	if len(signing) == 0 {
		t.Fatal("canonical signing bytes are empty")
	}
	t.Logf("canonical tx bytes: %s", hex.EncodeToString(raw1))
}

func TestCanonicalStakeLockSigningOmitsDerivedStakeID(t *testing.T) {
	tx := Transaction{
		Version:  TxVersionCanonical,
		From:     "owner",
		To:       "owner",
		Amount:   100,
		Nonce:    1,
		Type:     TxTypeStakeLock,
		StakeID:  "derived-stake-id",
	}
	got := tx.CanonicalSigningBytes()
	tx.StakeID = ""
	want := tx.CanonicalSigningBytes()
	if !bytes.Equal(got, want) {
		t.Fatal("stake lock canonical signing bytes depend on derived stake id")
	}
}
