package types

import (
	"bytes"
	"testing"
)

func TestAssetTransactionUsesNativeIDRFeeModel(t *testing.T) {
	tx := NewAssetTransferTransaction("from", "to", "asset:usd", 1500, 7, 1)
	if tx.ProtocolVersion() != TxVersionAsset { t.Fatalf("version=%d", tx.ProtocolVersion()) }
	if tx.EffectiveAssetID() != "asset:usd" { t.Fatalf("asset=%q", tx.EffectiveAssetID()) }
	if tx.EffectiveFeePayer() != "from" { t.Fatalf("fee payer=%q", tx.EffectiveFeePayer()) }
	if tx.Fee != 7 { t.Fatalf("fee=%d", tx.Fee) }
}

func TestAssetTransactionCanonicalRoundTrip(t *testing.T) {
	tx := Transaction{Version:TxVersionAsset, ID:"abcd", From:"from", To:"to", AssetID:"asset:usd", Amount:123, Fee:7, Nonce:9, Timestamp:1700000000, Signature:"signature", PublicKey:"public-key", FeePayer:"paymaster", FeePayerPublicKey:"payer-pub", FeePayerSignature:"payer-sig", Type:TxTypeTransfer}
	raw, err := tx.CanonicalBytes(); if err != nil { t.Fatal(err) }
	decoded, err := DecodeCanonicalTransaction(raw); if err != nil { t.Fatal(err) }
	if decoded != tx { t.Fatalf("round-trip mismatch: got=%+v want=%+v", decoded, tx) }
}

func TestAssetSigningBindsFeePayerAndAsset(t *testing.T) {
	tx := Transaction{Version:TxVersionAsset, From:"from", To:"to", AssetID:"asset:usd", Amount:1, Fee:7, Nonce:1, Timestamp:1, PublicKey:"pub", FeePayer:"paymaster", Type:TxTypeTransfer}
	a, err := tx.SigningBytesWithChainID(777001); if err != nil { t.Fatal(err) }
	tx.FeePayer="other"
	b, err := tx.SigningBytesWithChainID(777001); if err != nil { t.Fatal(err) }
	if bytes.Equal(a,b) { t.Fatal("fee payer not bound into v3 signing domain") }
}

func TestFeePayerAuthorizationDoesNotDependOnSponsorSignature(t *testing.T) {
	tx := Transaction{Version:TxVersionAsset, ID:"abcd", From:"from", To:"to", AssetID:"asset:usd", Amount:1, Fee:7, Nonce:1, Timestamp:1, PublicKey:"pub", FeePayer:"paymaster", Type:TxTypeTransfer}
	a, err := tx.FeePayerSigningBytesWithChainID(777001); if err != nil { t.Fatal(err) }
	tx.FeePayerPublicKey="pub2"; tx.FeePayerSignature="sig"
	b, err := tx.FeePayerSigningBytesWithChainID(777001); if err != nil { t.Fatal(err) }
	if !bytes.Equal(a,b) { t.Fatal("paymaster signature fields changed authorization bytes") }
}

func TestValidateAssetEnvelope(t *testing.T) {
	tx := NewAssetTransferTransaction("from", "to", "asset:usd", 100, 2, 1)
	if err := tx.ValidateAssetEnvelope(); err != nil {
		t.Fatal(err)
	}
	sponsored := tx
	sponsored.FeePayer = "paymaster"
	if err := sponsored.ValidateAssetEnvelope(); err == nil {
		t.Fatal("incomplete paymaster authorization accepted")
	}
	sponsored.FeePayerPublicKey = "payer-pub"
	sponsored.FeePayerSignature = "payer-sig"
	if err := sponsored.ValidateAssetEnvelope(); err != nil {
		t.Fatal(err)
	}
	badMint := NewAssetMintTransaction("issuer", "IDR", "alice", 10, 1, 1)
	if err := badMint.ValidateAssetEnvelope(); err == nil {
		t.Fatal("native IDR mint accepted as issued asset operation")
	}
}


func TestActivatedV3NetworkAcceptsLegacyAndAssetTransactions(t *testing.T) {
	for _, version := range []uint32{TxVersionLegacy, TxVersionCanonical, TxVersionAsset} {
		if err := ValidateTransactionVersion(version, TxVersionAsset); err != nil {
			t.Fatalf("version %d rejected after v3 activation: %v", version, err)
		}
	}
	if err := ValidateTransactionVersion(MaxSupportedTxVersion+1, TxVersionAsset); err == nil {
		t.Fatal("unsupported transaction version accepted")
	}
}
