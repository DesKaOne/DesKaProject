package wallet

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// Phase 1 wallet storage is not production safe.
type Store struct {
	path string
}

var locks sync.Map

func NewStore(path string) Store {
	return Store{path: path}
}

func (s Store) Load() ([]Wallet, error) {
	s.lock().Lock()
	defer s.lock().Unlock()
	return s.loadUnlocked()
}

func (s Store) loadUnlocked() ([]Wallet, error) {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var wallets []Wallet
	if len(raw) == 0 {
		return nil, nil
	}
	return wallets, json.Unmarshal(raw, &wallets)
}

func (s Store) Save(wallets []Wallet) error {
	s.lock().Lock()
	defer s.lock().Unlock()
	return s.saveUnlocked(wallets)
}

func (s Store) saveUnlocked(wallets []Wallet) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(wallets, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(s.path, raw, 0600)
}

func (s Store) Add(wallet Wallet) error {
	s.lock().Lock()
	defer s.lock().Unlock()
	wallets, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	for _, existing := range wallets {
		if existing.Address == wallet.Address {
			return nil
		}
	}
	wallets = append(wallets, wallet)
	return s.saveUnlocked(wallets)
}

func (s Store) Find(address string) (Wallet, bool, error) {
	s.lock().Lock()
	defer s.lock().Unlock()
	wallets, err := s.loadUnlocked()
	if err != nil {
		return Wallet{}, false, err
	}
	for _, wallet := range wallets {
		if wallet.Address == address {
			return wallet, true, nil
		}
	}
	return Wallet{}, false, nil
}

func (s Store) lock() *sync.Mutex {
	clean := filepath.Clean(s.path)
	lock, _ := locks.LoadOrStore(clean, &sync.Mutex{})
	return lock.(*sync.Mutex)
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
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			_ = os.Remove(tmp)
			return err
		}
		return os.Rename(tmp, path)
	}
	return nil
}
