package mempool

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"deskachain/internal/types"
)

func TestAtomicSaveAndDuplicateGuard(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mempool.json")
	mp := New(path)
	tx := types.Transaction{ID: "tx1", From: "a", To: "b", Amount: 1}
	if err := mp.Add(tx); err != nil {
		t.Fatal(err)
	}
	if err := mp.Add(tx); !errors.Is(err, ErrDuplicateTx) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	txs, err := mp.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 || txs[0].ID != tx.ID {
		t.Fatalf("unexpected txs: %#v", txs)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []types.Transaction
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp file left behind: %v", err)
	}
}

func TestLoadInvalidJSONReturnsClearError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mempool.json")
	if err := os.WriteFile(path, []byte("{bad"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(path).Load(); err == nil || !strings.Contains(err.Error(), "failed to load mempool: invalid json") {
		t.Fatalf("expected invalid json error, got %v", err)
	}
}

func TestConcurrentDuplicateAdd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mempool.json")
	mp := New(path)
	tx := types.Transaction{ID: "same", From: "a", To: "b", Amount: 1}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := mp.Add(tx)
			if err != nil && !errors.Is(err, ErrDuplicateTx) {
				t.Errorf("add error: %v", err)
			}
		}()
	}
	wg.Wait()
	txs, err := mp.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 {
		t.Fatalf("duplicate txs stored: %#v", txs)
	}
}

func TestMempoolAcceptsValidStakeLockAndRejectsDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mempool.json")
	mp := New(path)
	tx := types.Transaction{ID: "stake-lock-1", Type: types.TxTypeStakeLock, From: "addr1", To: "addr1", Amount: 10, StakeID: "stake-lock-1"}
	if err := mp.Add(tx); err != nil {
		t.Fatal(err)
	}
	if err := mp.Add(tx); !errors.Is(err, ErrDuplicateTx) {
		t.Fatalf("expected duplicate stake lock rejection, got %v", err)
	}
	txs, err := mp.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 || txs[0].TxType() != types.TxTypeStakeLock || txs[0].StakeID != "stake-lock-1" {
		t.Fatalf("unexpected stake lock mempool contents: %#v", txs)
	}
}

func TestMempoolStoresStakeUnlockPending(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mempool.json")
	mp := New(path)
	tx := types.Transaction{ID: "stake-unlock-1", Type: types.TxTypeStakeUnlock, From: "addr1", To: "addr1", StakeID: "stake-lock-1"}
	if err := mp.Add(tx); err != nil {
		t.Fatal(err)
	}
	txs, err := mp.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 || txs[0].TxType() != types.TxTypeStakeUnlock || txs[0].StakeID != "stake-lock-1" {
		t.Fatalf("unexpected stake unlock mempool contents: %#v", txs)
	}
}
