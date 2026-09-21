package wallet

import (
	"testing"

	"indochain/internal/config"
	"indochain/internal/crypto"
	"indochain/internal/types"
)

func TestSignTransactionWithProfileBindsV2ToChainID(t *testing.T) {
	base := config.Localnet()
	base.TxVersion = types.TxVersionCanonical

	profileA := base
	profileB := base
	profileB.ChainID++

	w, err := NewWithProfile(profileA)
	if err != nil {
		t.Fatal(err)
	}
	wB, err := FromPrivateKeyHex(w.PrivateKeyHex, profileB)
	if err != nil {
		t.Fatal(err)
	}

	txA := types.Transaction{
		Version:   types.TxVersionCanonical,
		From:      w.Address,
		To:        w.Address,
		Amount:    1,
		Nonce:     1,
		Timestamp: 1700000000,
		Type:      types.TxTypeStakeLock,
	}
	txB := txA
	txB.From = wB.Address
	txB.To = wB.Address

	if err := w.SignTransactionWithProfile(&txA, profileA); err != nil {
		t.Fatal(err)
	}
	if err := wB.SignTransactionWithProfile(&txB, profileB); err != nil {
		t.Fatal(err)
	}
	if txA.ID == txB.ID {
		t.Fatal("v2 transaction ids are identical across chain ids")
	}
	if txA.Signature == txB.Signature {
		t.Fatal("v2 signatures are identical across chain ids")
	}

	payloadA, err := txA.SigningBytesWithChainID(profileA.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if !crypto.VerifyHex(txA.PublicKey, txA.Signature, payloadA) {
		t.Fatal("v2 signature does not verify on source chain")
	}

	payloadB, err := txB.SigningBytesWithChainID(profileB.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if !crypto.VerifyHex(txB.PublicKey, txB.Signature, payloadB) {
		t.Fatal("v2 signature does not verify on destination chain")
	}

	wrongPayload, err := txA.SigningBytesWithChainID(profileB.ChainID)
	if err != nil {
		t.Fatal(err)
	}
	if crypto.VerifyHex(txA.PublicKey, txA.Signature, wrongPayload) {
		t.Fatal("v2 signature verified under the wrong chain id")
	}
}
