package state

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"

	"indochain/internal/asset"
	"indochain/internal/crypto"
	"indochain/internal/ledger"
	"indochain/internal/staking"
)

const (
	StateRootVersion uint8 = 1
	StateRootDomain        = "IndoChain/state-root/v1"
)

var ErrNilLedger = errors.New("nil mature ledger")

func RootForLedger(l *ledger.MatureLedger) (string, error) {
	if l == nil {
		return "", ErrNilLedger
	}
	if l.AssetModelEnabled() {
		return RootForCollectionsWithAssets(l.StateAccounts(), l.StateStakes(), l.StateAssetDefinitions(), l.StateAssetBalances())
	}
	return RootForCollections(l.StateAccounts(), l.StateStakes())
}

func RootForCollectionsWithAssets(accounts []ledger.StateAccount, stakes []staking.Record, definitions []asset.Definition, balances []asset.BalanceEntry) (string, error) {
	accounts, stakes = StableDigestInputs(accounts, stakes)
	definitions = append([]asset.Definition(nil), definitions...)
	balances = append([]asset.BalanceEntry(nil), balances...)
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].ID < definitions[j].ID })
	sort.Slice(balances, func(i, j int) bool {
		if balances[i].Address != balances[j].Address {
			return balances[i].Address < balances[j].Address
		}
		return balances[i].AssetID < balances[j].AssetID
	})

	var buf bytes.Buffer
	buf.WriteByte(StateRootVersion)
	writeString(&buf, "IndoChain/state-root/v2-assets")
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
	writeUint32(&buf, uint32(len(definitions)))
	for _, def := range definitions {
		writeString(&buf, def.ID)
		writeString(&buf, def.Name)
		writeString(&buf, def.Symbol)
		buf.WriteByte(def.Decimals)
		writeString(&buf, def.Kind)
		writeString(&buf, def.Issuer)
		writeUint64(&buf, def.MaxSupply)
		writeUint64(&buf, def.TotalSupply)
		writeBool(&buf, def.Mintable)
		writeBool(&buf, def.Burnable)
		writeBool(&buf, def.Pausable)
		writeBool(&buf, def.Permissioned)
		writeString(&buf, def.Status)
	}
	writeUint32(&buf, uint32(len(balances)))
	for _, entry := range balances {
		writeString(&buf, entry.Address)
		writeString(&buf, entry.AssetID)
		writeUint64(&buf, entry.Amount)
	}
	return crypto.DoubleSHA256Hex(buf.Bytes()), nil
}

func writeBool(buf *bytes.Buffer, value bool) {
	if value {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
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
