package types

import (
	"bytes"
	"testing"
)

func TestLegacyBlockVersionDoesNotChangeHeaderBytes(t *testing.T) {
	legacy := NewBlock(1, "prev", "miner", 1, nil)
	explicit := NewBlockWithVersion(1, "prev", "miner", 1, nil, BlockVersionLegacy)
	legacy.Timestamp = 1700000000
	explicit.Timestamp = legacy.Timestamp
	legacy.MerkleRoot = "merkle"
	explicit.MerkleRoot = legacy.MerkleRoot
	if legacy.Version != 0 || explicit.Version != 0 {
		t.Fatalf("legacy constructor unexpectedly serialized block version: %d %d", legacy.Version, explicit.Version)
	}
	if !bytes.Equal(legacy.HeaderBytesWithNonce(9), explicit.HeaderBytesWithNonce(9)) {
		t.Fatal("legacy block header bytes changed")
	}
	if legacy.CalculateHash() != explicit.CalculateHash() {
		t.Fatal("legacy block hash changed")
	}
}

func TestCanonicalBlockHeaderIsDeterministicAndFieldBound(t *testing.T) {
	block := NewBlockWithVersion(2, "prev", "miner", 1, nil, BlockVersionCanonical)
	block.Timestamp = 1700000000
	block.MerkleRoot = "merkle"

	first, err := block.CanonicalHeaderBytesWithNonce(9)
	if err != nil {
		t.Fatal(err)
	}
	second, err := block.CanonicalHeaderBytesWithNonce(9)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("canonical block header encoding is not deterministic")
	}

	changed := block
	changed.MinerAddress = "other-miner"
	third, err := changed.CanonicalHeaderBytesWithNonce(9)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, third) {
		t.Fatal("canonical header did not bind miner address")
	}

	if bytes.Equal(first, block.HeaderBytesWithNonce(9)) == false {
		t.Fatal("block hash path is not using canonical header codec")
	}
}

func TestCanonicalBlockHeaderSeparatesDomainAndVersion(t *testing.T) {
	block := NewBlockWithVersion(2, "prev", "miner", 1, nil, BlockVersionCanonical)
	raw, err := block.CanonicalHeaderBytesWithNonce(0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(canonicalBlockHeaderDomain)) {
		t.Fatal("canonical header missing domain separator")
	}
	legacy := NewBlock(2, "prev", "miner", 1, nil)
	legacy.Timestamp = block.Timestamp
	legacy.MerkleRoot = block.MerkleRoot
	if bytes.Equal(raw, legacy.HeaderBytesWithNonce(0)) {
		t.Fatal("canonical and legacy header encodings unexpectedly match")
	}
}

func TestCanonicalBlockHeaderRejectsWrongVersion(t *testing.T) {
	block := NewBlock(2, "prev", "miner", 1, nil)
	if _, err := block.CanonicalHeaderBytesWithNonce(0); err == nil {
		t.Fatal("legacy block unexpectedly accepted by canonical header codec")
	}
}
