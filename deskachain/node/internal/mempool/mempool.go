package mempool

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"indochain/internal/types"
)

var (
	ErrDuplicateTx = errors.New("mempool tx already exists")
	locks          sync.Map
)

type Mempool struct {
	path string
}

func New(path string) Mempool {
	return Mempool{path: path}
}

func (m Mempool) Load() ([]types.Transaction, error) {
	m.lock().Lock()
	defer m.lock().Unlock()
	return m.loadUnlocked()
}

func (m Mempool) loadUnlocked() ([]types.Transaction, error) {
	raw, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var txs []types.Transaction
	if len(raw) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &txs); err != nil {
		return nil, fmt.Errorf("failed to load mempool: invalid json: %w", err)
	}
	return uniqueTransactions(txs), nil
}

func (m Mempool) Save(txs []types.Transaction) error {
	m.lock().Lock()
	defer m.lock().Unlock()
	return m.saveUnlocked(txs)
}

func (m Mempool) saveUnlocked(txs []types.Transaction) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(uniqueTransactions(txs), "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(m.path, raw, 0644)
}

func (m Mempool) Add(tx types.Transaction) error {
	m.lock().Lock()
	defer m.lock().Unlock()
	txs, err := m.loadUnlocked()
	if err != nil {
		return err
	}
	for _, existing := range txs {
		if existing.ID == tx.ID {
			return ErrDuplicateTx
		}
	}
	txs = append(txs, tx)
	return m.saveUnlocked(txs)
}

func (m Mempool) Clear() error {
	return m.Save(nil)
}

func (m Mempool) RemoveIDs(ids map[string]struct{}) error {
	m.lock().Lock()
	defer m.lock().Unlock()
	txs, err := m.loadUnlocked()
	if err != nil {
		return err
	}
	remaining := make([]types.Transaction, 0, len(txs))
	for _, tx := range txs {
		if _, ok := ids[tx.ID]; ok {
			continue
		}
		remaining = append(remaining, tx)
	}
	return m.saveUnlocked(remaining)
}

func (m Mempool) lock() *sync.Mutex {
	key := filepath.Clean(m.path)
	value, _ := locks.LoadOrStore(key, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func uniqueTransactions(txs []types.Transaction) []types.Transaction {
	seen := make(map[string]struct{}, len(txs))
	out := make([]types.Transaction, 0, len(txs))
	for _, tx := range txs {
		if tx.ID == "" {
			out = append(out, tx)
			continue
		}
		if _, ok := seen[tx.ID]; ok {
			continue
		}
		seen[tx.ID] = struct{}{}
		out = append(out, tx)
	}
	return out
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func PendingOutgoing(txs []types.Transaction, address string) (count uint64, amount uint64, fee uint64) {
	for _, tx := range txs {
		if tx.Coinbase || tx.From != address {
			continue
		}
		count++
		amount += tx.Amount
		fee += tx.Fee
	}
	return count, amount, fee
}
