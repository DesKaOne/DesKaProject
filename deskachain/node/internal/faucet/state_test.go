package faucet

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFaucetStatePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "faucet_state.json")
	store := NewStore(path)
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	if err := store.Record("addr1", "tx1", 100, now); err != nil {
		t.Fatal(err)
	}
	loaded, err := NewStore(path).Load()
	if err != nil {
		t.Fatal(err)
	}
	entry := loaded.Requests["addr1"]
	if entry.LastRequestTime == "" || entry.TotalAmountRequested != 100 || entry.DailyAmountRequested != 100 || len(entry.PendingTxIDs) != 1 || entry.PendingTxIDs[0] != "tx1" {
		t.Fatalf("unexpected state entry: %+v", entry)
	}
}

func TestFaucetStateAddressTotals(t *testing.T) {
	path := filepath.Join(t.TempDir(), "faucet_state.json")
	store := NewStore(path)
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	if err := store.Record("addr1", "tx1", 100, now); err != nil {
		t.Fatal(err)
	}
	if err := store.Record("addr1", "tx2", 200, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := store.Record("addr2", "tx3", 300, now); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Requests["addr1"].TotalAmountRequested; got != 300 {
		t.Fatalf("addr1 total = %d, want 300", got)
	}
	if got := loaded.Requests["addr1"].DailyAmountRequested; got != 300 {
		t.Fatalf("addr1 daily = %d, want 300", got)
	}
	if got := len(loaded.Requests["addr1"].PendingTxIDs); got != 2 {
		t.Fatalf("addr1 pending count = %d, want 2", got)
	}
	if got := loaded.Requests["addr2"].TotalAmountRequested; got != 300 {
		t.Fatalf("addr2 total = %d, want 300", got)
	}
}

func TestFaucetStateDailyAmountResetsOnNewDay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "faucet_state.json")
	store := NewStore(path)
	day1 := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	day2 := day1.Add(24 * time.Hour)
	if err := store.Record("addr1", "tx1", 100, day1); err != nil {
		t.Fatal(err)
	}
	if err := store.Record("addr1", "tx2", 200, day2); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	entry := loaded.Requests["addr1"]
	if entry.TotalAmountRequested != 300 || entry.DailyAmountRequested != 200 || entry.Day != "2026-06-23" {
		t.Fatalf("unexpected day rollover entry: %+v", entry)
	}
}

func TestFaucetStatePendingDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "faucet_state.json")
	store := NewStore(path)
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	if err := store.Record("addr1", "tx1", 100, now); err != nil {
		t.Fatal(err)
	}
	if err := store.Record("addr1", "tx1", 100, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(loaded.Requests["addr1"].PendingTxIDs); got != 1 {
		t.Fatalf("pending duplicate count = %d, want 1", got)
	}
}

func TestFaucetStateCorruptHandled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "faucet_state.json")
	if err := os.WriteFile(path, []byte("{bad"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(path).Load(); err == nil {
		t.Fatal("expected corrupt faucet state error")
	}
}
