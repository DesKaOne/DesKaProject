package types

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	canonicalTxCodecVersion uint8 = 1
	maxCanonicalFieldBytes        = 1 << 20
)

var (
	ErrInvalidCanonicalTx = errors.New("invalid canonical transaction")
)

func (tx Transaction) CanonicalSigningBytes() []byte {
	var buf bytes.Buffer
	writeCanonicalHeader(&buf, tx.ProtocolVersion(), tx.TxType(), tx.Coinbase)
	writeCanonicalString(&buf, tx.From)
	writeCanonicalString(&buf, tx.To)
	writeCanonicalUint64(&buf, tx.Amount)
	writeCanonicalUint64(&buf, tx.Fee)
	writeCanonicalUint64(&buf, tx.Nonce)
	writeCanonicalInt64(&buf, tx.Timestamp)
	writeCanonicalString(&buf, tx.PublicKey)
	stakeID := tx.StakeID
	if tx.TxType() == TxTypeStakeLock {
		stakeID = ""
	}
	writeCanonicalString(&buf, stakeID)
	return buf.Bytes()
}

func (tx Transaction) CanonicalBytes() ([]byte, error) {
	if tx.ProtocolVersion() == 0 {
		return nil, fmt.Errorf("%w: missing protocol version", ErrInvalidCanonicalTx)
	}
	var buf bytes.Buffer
	writeCanonicalHeader(&buf, tx.ProtocolVersion(), tx.TxType(), tx.Coinbase)
	writeCanonicalString(&buf, tx.ID)
	writeCanonicalString(&buf, tx.From)
	writeCanonicalString(&buf, tx.To)
	writeCanonicalUint64(&buf, tx.Amount)
	writeCanonicalUint64(&buf, tx.Fee)
	writeCanonicalUint64(&buf, tx.Nonce)
	writeCanonicalInt64(&buf, tx.Timestamp)
	writeCanonicalString(&buf, tx.Signature)
	writeCanonicalString(&buf, tx.PublicKey)
	writeCanonicalString(&buf, tx.StakeID)
	return buf.Bytes(), nil
}

func DecodeCanonicalTransaction(raw []byte) (Transaction, error) {
	r := bytes.NewReader(raw)
	codec, err := r.ReadByte()
	if err != nil || codec != canonicalTxCodecVersion {
		return Transaction{}, fmt.Errorf("%w: unsupported codec version", ErrInvalidCanonicalTx)
	}
	version, err := readUint32(r)
	if err != nil || version == 0 {
		return Transaction{}, fmt.Errorf("%w: invalid protocol version", ErrInvalidCanonicalTx)
	}
	txType, err := readCanonicalString(r)
	if err != nil {
		return Transaction{}, err
	}
	coinbase, err := r.ReadByte()
	if err != nil || (coinbase != 0 && coinbase != 1) {
		return Transaction{}, fmt.Errorf("%w: invalid coinbase flag", ErrInvalidCanonicalTx)
	}
	tx := Transaction{
		Version:  version,
		Type:     txType,
		Coinbase: coinbase == 1,
	}
	if tx.ID, err = readCanonicalString(r); err != nil { return Transaction{}, err }
	if tx.From, err = readCanonicalString(r); err != nil { return Transaction{}, err }
	if tx.To, err = readCanonicalString(r); err != nil { return Transaction{}, err }
	if tx.Amount, err = readUint64(r); err != nil { return Transaction{}, err }
	if tx.Fee, err = readUint64(r); err != nil { return Transaction{}, err }
	if tx.Nonce, err = readUint64(r); err != nil { return Transaction{}, err }
	if tx.Timestamp, err = readInt64(r); err != nil { return Transaction{}, err }
	if tx.Signature, err = readCanonicalString(r); err != nil { return Transaction{}, err }
	if tx.PublicKey, err = readCanonicalString(r); err != nil { return Transaction{}, err }
	if tx.StakeID, err = readCanonicalString(r); err != nil { return Transaction{}, err }
	if r.Len() != 0 {
		return Transaction{}, fmt.Errorf("%w: trailing bytes", ErrInvalidCanonicalTx)
	}
	return tx, nil
}

func writeCanonicalHeader(buf *bytes.Buffer, version uint32, txType string, coinbase bool) {
	buf.WriteByte(canonicalTxCodecVersion)
	writeCanonicalUint32(buf, version)
	writeCanonicalString(buf, txType)
	if coinbase {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
}

func writeCanonicalString(buf *bytes.Buffer, value string) {
	writeCanonicalUint32(buf, uint32(len(value)))
	buf.WriteString(value)
}

func writeCanonicalUint32(buf *bytes.Buffer, value uint32) {
	var raw [4]byte
	binary.BigEndian.PutUint32(raw[:], value)
	buf.Write(raw[:])
}

func writeCanonicalUint64(buf *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buf.Write(raw[:])
}

func writeCanonicalInt64(buf *bytes.Buffer, value int64) {
	writeCanonicalUint64(buf, uint64(value))
}

func readUint32(r *bytes.Reader) (uint32, error) {
	var raw [4]byte
	if _, err := r.Read(raw[:]); err != nil {
		return 0, fmt.Errorf("%w: uint32: %v", ErrInvalidCanonicalTx, err)
	}
	return binary.BigEndian.Uint32(raw[:]), nil
}

func readUint64(r *bytes.Reader) (uint64, error) {
	var raw [8]byte
	if _, err := r.Read(raw[:]); err != nil {
		return 0, fmt.Errorf("%w: uint64: %v", ErrInvalidCanonicalTx, err)
	}
	return binary.BigEndian.Uint64(raw[:]), nil
}

func readInt64(r *bytes.Reader) (int64, error) {
	value, err := readUint64(r)
	return int64(value), err
}

func readCanonicalString(r *bytes.Reader) (string, error) {
	length, err := readUint32(r)
	if err != nil {
		return "", err
	}
	if length > maxCanonicalFieldBytes {
		return "", fmt.Errorf("%w: field too large", ErrInvalidCanonicalTx)
	}
	if uint64(length) > uint64(r.Len()) {
		return "", fmt.Errorf("%w: truncated field", ErrInvalidCanonicalTx)
	}
	raw := make([]byte, length)
	if _, err := r.Read(raw); err != nil {
		return "", fmt.Errorf("%w: field: %v", ErrInvalidCanonicalTx, err)
	}
	return string(raw), nil
}
