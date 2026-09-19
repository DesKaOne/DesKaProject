package asset

import (
	"errors"
	"testing"
)

func TestAssetStateCreateMintTransferBurn(t *testing.T) {
	s := NewState()
	def, err := s.CreateFromTransactionID("abcdef", "Example USD", "EUSD", 6, 1_000_000, true, true, false, true, "issuer")
	if err != nil { t.Fatal(err) }
	if def.ID != "asset:abcdef" { t.Fatalf("asset id=%q", def.ID) }

	if err := s.Mint(def.ID, "issuer", "alice", 1000); err != nil { t.Fatal(err) }
	if got := s.Balance("alice", def.ID); got != 1000 { t.Fatalf("alice token balance=%d", got) }

	if err := s.Transfer(def.ID, "alice", "bob", 250); err != nil { t.Fatal(err) }
	if s.Balance("alice", def.ID) != 750 || s.Balance("bob", def.ID) != 250 {
		t.Fatalf("unexpected balances: alice=%d bob=%d", s.Balance("alice", def.ID), s.Balance("bob", def.ID))
	}

	if err := s.Burn(def.ID, "bob", 50); err != nil { t.Fatal(err) }
	updated, _ := s.Definition(def.ID)
	if updated.TotalSupply != 950 { t.Fatalf("total supply=%d", updated.TotalSupply) }
}

func TestAssetStateNativeIDRFeeAndPaymaster(t *testing.T) {
	s := NewState()
	if err := s.Create(Definition{ID:"asset:usd", Name:"Example USD", Symbol:"EUSD", Decimals:6, Issuer:"issuer", Mintable:true, Burnable:true, Status:StatusActive}); err != nil { t.Fatal(err) }
	if err := s.Mint("asset:usd", "issuer", "alice", 100); err != nil { t.Fatal(err) }

	s.balances["alice"] = map[string]uint64{NativeAssetID: 10}
	s.balances["paymaster"] = map[string]uint64{NativeAssetID: 5}
	if err := s.ExecuteTokenTransfer("asset:usd", "alice", "merchant", "paymaster", 40, 2); err != nil { t.Fatal(err) }

	if s.Balance("alice", "asset:usd") != 60 || s.Balance("merchant", "asset:usd") != 40 {
		t.Fatalf("token transfer not applied")
	}
	if s.Balance("alice", NativeAssetID) != 10 || s.Balance("paymaster", NativeAssetID) != 3 || s.Balance(FeeCollectorAddress, NativeAssetID) != 2 {
		t.Fatalf("paymaster fee not charged in native IDR")
	}
}

func TestAssetStateFeeFailureRollsBackTransfer(t *testing.T) {
	s := NewState()
	_ = s.Create(Definition{ID:"asset:usd", Name:"Example USD", Symbol:"EUSD", Decimals:6, Issuer:"issuer", Mintable:true, Burnable:true, Status:StatusActive})
	_ = s.Mint("asset:usd", "issuer", "alice", 100)
	err := s.ExecuteTokenTransfer("asset:usd", "alice", "merchant", "paymaster", 40, 1)
	if err == nil || !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected fee payer failure, got %v", err)
	}
	if s.Balance("alice", "asset:usd") != 100 || s.Balance("merchant", "asset:usd") != 0 {
		t.Fatalf("token transfer was not rolled back")
	}
}
