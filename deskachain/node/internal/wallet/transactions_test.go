package wallet

import (
	"testing"

	"indochain/internal/config"
	indochaincrypto "indochain/internal/crypto"
)

func TestSignTransferBindsNetworkAndProducesCompleteV3Transaction(t *testing.T) {
	profile := config.Testnet()
	sender, err := NewWithProfile(profile)
	if err != nil { t.Fatal(err) }
	recipient, err := NewWithProfile(profile)
	if err != nil { t.Fatal(err) }

	tx, err := sender.SignTransfer(profile, recipient.Address, 100, profile.Fee.MinFee, 7)
	if err != nil { t.Fatal(err) }

	if tx.Version != 3 { t.Fatalf("version=%d, want 3", tx.Version) }
	if tx.From != sender.Address || tx.To != recipient.Address { t.Fatal("sender/recipient mismatch") }
	if tx.PublicKey != sender.PublicKeyHex || tx.Signature == "" || tx.ID == "" { t.Fatal("incomplete signed transaction") }

	payload, err := tx.SigningBytesWithChainID(profile.ChainID)
	if err != nil { t.Fatal(err) }
	if !indochaincrypto.VerifyHex(tx.PublicKey, tx.Signature, payload) { t.Fatal("signature does not verify") }
}

func TestTransferRejectsWrongNetworkRecipient(t *testing.T) {
	testnet := config.Testnet()
	mainnet := config.Mainnet()
	sender, err := NewWithProfile(testnet)
	if err != nil { t.Fatal(err) }
	recipient, err := NewWithProfile(mainnet)
	if err != nil { t.Fatal(err) }

	if _, err := sender.NewTransfer(testnet, recipient.Address, 1, testnet.Fee.MinFee, 1); err == nil {
		t.Fatal("wrong-network recipient unexpectedly accepted")
	}
}


func TestNewStakeLockUsesNetworkMinimumFee(t *testing.T) {
	profile := config.Testnet()
	w, err := NewWithProfile(profile)
	if err != nil { t.Fatal(err) }
	tx, err := w.NewStakeLock(profile, profile.Consensus.Staking.MinStakeAmount, 2)
	if err != nil { t.Fatal(err) }
	if tx.Fee != profile.Fee.MinFee {
		t.Fatalf("fee=%d, want %d", tx.Fee, profile.Fee.MinFee)
	}
	if tx.From != w.Address || tx.To != w.Address {
		t.Fatal("stake transaction sender/recipient mismatch")
	}
}
