package wallet

import (
	"sync"
	"testing"

	"indochain/internal/crypto"
)

func TestStoreConcurrentAddCreatesUniqueValidWallets(t *testing.T) {
	store := NewStore(t.TempDir() + "/wallets.json")
	const count = 20
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w, err := New()
			if err != nil {
				errs <- err
				return
			}
			errs <- store.Add(w)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	wallets, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(wallets) != count {
		t.Fatalf("wallet count = %d, want %d", len(wallets), count)
	}
	seen := map[string]struct{}{}
	for _, w := range wallets {
		if err := crypto.ValidateAddress(w.Address); err != nil {
			t.Fatalf("invalid wallet address %s: %v", w.Address, err)
		}
		if _, ok := seen[w.Address]; ok {
			t.Fatalf("duplicate wallet address: %s", w.Address)
		}
		seen[w.Address] = struct{}{}
	}
}
