package state

import (
	"bytes"
	"encoding/binary"
	"errors"

	"deskachain/internal/crypto"
	"deskachain/internal/ledger"
	"deskachain/internal/staking"
)

const (
	StateRootVersion uint8 = 1
	StateRootDomain         = "DesKaChain/state-root/v1"
)

var ErrNilLedger = errors.New("nil mature ledger")

func RootForLedger(l *ledger.MatureLedger) (string, error) {
	if l == nil {
		return "", ErrNilLedger
	}
	return RootForCollections(l.StateAccounts(), l.StateStakes())
}

func RootForCollections(accounts []ledger.StateAccount, stakes []staking.Record) (string, error) {
	accounts, stakes = StableDigestInputs(accounts, stakes)

	var buf bytes.Buffer
	buf.WriteByte(StateRootVersion)
	writeString(&buf, StateRootDomain)
	writeUint32(&buf, uint32(len(accounts)))
	for _, account := range accounts {
		writeString(&buf, account.Address)
		writeUint64(&buf, account.Confirmed)
		writeUint64(&buf, account.Mature)
		writeUint64(&buf, account.Nonce)
	}
	writeUint32(&buf, uint32(len(stakes)))
	for _, record := range stakes {
		writeStakeRecord(&buf, record)
	}
	return crypto.DoubleSHA256Hex(buf.Bytes()), nil
}

func writeStakeRecord(buf *bytes.Buffer, record staking.Record) {
	writeString(buf, record.StakeID)
	writeString(buf, record.OwnerAddress)
	writeUint64(buf, record.Amount)
	writeString(buf, record.LockTxID)
	writeUint64(buf, record.LockHeight)
	writeString(buf, record.UnlockTxID)
	writeUint64(buf, record.UnlockHeight)
	writeUint64(buf, record.ReleaseHeight)
	writeString(buf, record.Status)
}

func writeString(buf *bytes.Buffer, value string) {
	writeUint32(buf, uint32(len(value)))
	buf.WriteString(value)
}

func writeUint32(buf *bytes.Buffer, value uint32) {
	var raw [4]byte
	binary.BigEndian.PutUint32(raw[:], value)
	buf.Write(raw[:])
}

func writeUint64(buf *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buf.Write(raw[:])
}
