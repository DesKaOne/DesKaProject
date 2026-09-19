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

func TestV2SigningBytesBindChainID(t *testing.T) {
	tx := Transaction{
		Version:   TxVersionCanonical,
		From:      "from",
		To:        "to",
		Amount:    123,
		Fee:       7,
		Nonce:     9,
		Timestamp: 1700000000,
		PublicKey: "pub",
		Type:      TxTypeTransfer,
	}
	first, err := tx.SigningBytesWithChainID(777001)
	if err != nil {
		t.Fatal(err)
	}
	second, err := tx.SigningBytesWithChainID(777101)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("v2 signing bytes do not bind chain id")
	}
	firstID, err := tx.CalculateIDForChainID(777001)
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := tx.CalculateIDForChainID(777101)
	if err != nil {
		t.Fatal(err)
	}
	if firstID == secondID {
		t.Fatal("v2 transaction ids do not bind chain id")
	}
}

func TestV2SigningDomainIsSeparated(t *testing.T) {
	tx := Transaction{
		Version:   TxVersionCanonical,
		From:      "from",
		To:        "to",
		Amount:    1,
		Nonce:     1,
		PublicKey: "pub",
		Type:      TxTypeTransfer,
	}
	canonical := tx.CanonicalSigningBytes()
	bound, err := tx.CanonicalSigningBytesWithChainID(777001)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(canonical, bound) {
		t.Fatal("chain-bound signing bytes equal canonical body")
	}
	if !bytes.Contains(bound, []byte(canonicalSigningDomain)) {
		t.Fatal("chain-bound signing bytes missing domain separator")
	}
}

func TestLegacyChainBoundSigningRemainsCompatible(t *testing.T) {
	tx := Transaction{
		From:      "from",
		To:        "to",
		Amount:    123,
		Fee:       7,
		Nonce:     9,
		Timestamp: 1700000000,
		PublicKey: "pub",
	}
	legacy := tx.SigningBytes()
	bound, err := tx.SigningBytesWithChainID(777001)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(legacy, bound) {
		t.Fatal("legacy chain-bound signing bytes changed legacy payload")
	}
	legacyID := tx.CalculateID()
	boundID, err := tx.CalculateIDForChainID(777001)
	if err != nil {
		t.Fatal(err)
	}
	if legacyID != boundID {
		t.Fatal("legacy chain-aware id changed transaction identity")
	}
}

func TestTransactionVersionActivation(t *testing.T) {
	if err := ValidateTransactionVersion(TxVersionCanonical, TxVersionLegacy); err == nil {
		t.Fatal("canonical version unexpectedly active on legacy network")
	}
	if err := ValidateTransactionVersion(TxVersionCanonical, TxVersionCanonical); err != nil {
		t.Fatal(err)
	}
}
