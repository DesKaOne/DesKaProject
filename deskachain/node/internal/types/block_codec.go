package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	canonicalBlockCodecVersion uint8 = 1
	canonicalBlockHeaderDomain       = "DesKaChain/block-header/v2"
)

var ErrInvalidCanonicalBlockHeader = errors.New("invalid canonical block header")

func (b Block) CanonicalHeaderBytesWithNonce(nonce uint64) ([]byte, error) {
	if b.ProtocolVersion() != BlockVersionCanonical {
		return nil, fmt.Errorf("%w: canonical header requires block version %d", ErrInvalidCanonicalBlockHeader, BlockVersionCanonical)
	}
	var buf bytes.Buffer
	buf.WriteByte(canonicalBlockCodecVersion)
	writeBlockCanonicalUint32(&buf, b.ProtocolVersion())
	writeBlockCanonicalString(&buf, canonicalBlockHeaderDomain)
	writeBlockCanonicalUint64(&buf, b.Height)
	writeBlockCanonicalString(&buf, b.PreviousHash)
	writeBlockCanonicalInt64(&buf, b.Timestamp)
	writeBlockCanonicalUint64(&buf, nonce)
	writeBlockCanonicalUint32(&buf, b.Difficulty)
	writeBlockCanonicalString(&buf, b.MinerAddress)
	writeBlockCanonicalString(&buf, b.MerkleRoot)
	writeBlockCanonicalString(&buf, b.StateRoot)
	writeBlockCanonicalString(&buf, b.GenesisMarker)
	return buf.Bytes(), nil
}

func writeBlockCanonicalString(buf *bytes.Buffer, value string) {
	writeBlockCanonicalUint32(buf, uint32(len(value)))
	buf.WriteString(value)
}

func writeBlockCanonicalUint32(buf *bytes.Buffer, value uint32) {
	var raw [4]byte
	binary.BigEndian.PutUint32(raw[:], value)
	buf.Write(raw[:])
}

func writeBlockCanonicalUint64(buf *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buf.Write(raw[:])
}

func writeBlockCanonicalInt64(buf *bytes.Buffer, value int64) {
	writeBlockCanonicalUint64(buf, uint64(value))
}
