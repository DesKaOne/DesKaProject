package faucet

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Store struct {
	path string
}

type State struct {
	Requests map[string]AddressState `json:"requests"`
}

type AddressState struct {
	LastRequestTime      string   `json:"last_request_time"`
	TotalAmountRequested uint64   `json:"total_amount_requested"`
	Day                  string   `json:"day,omitempty"`
	DailyAmountRequested uint64   `json:"daily_amount_requested,omitempty"`
	PendingTxIDs         []string `json:"pending_tx_ids,omitempty"`
}

var locks sync.Map

func NewStore(path string) Store {
	return Store{path: path}
}

func (s Store) Load() (State, error) {
	s.lock().Lock()
	defer s.lock().Unlock()
	return s.loadUnlocked()
}

func (s Store) Record(address string, txID string, amount uint64, now time.Time) error {
	s.lock().Lock()
	defer s.lock().Unlock()
	state, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	if state.Requests == nil {
		state.Requests = make(map[string]AddressState)
	}
	entry := state.Requests[address]
	day := now.UTC().Format("2006-01-02")
	if entry.Day != day {
		entry.Day = day
		entry.DailyAmountRequested = 0
	}
	entry.LastRequestTime = now.UTC().Format(time.RFC3339)
	entry.TotalAmountRequested += amount
	entry.DailyAmountRequested += amount
	entry.PendingTxIDs = appendUnique(entry.PendingTxIDs, txID)
	state.Requests[address] = entry
	return s.saveUnlocked(state)
}

func (s Store) loadUnlocked() (State, error) {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return State{Requests: map[string]AddressState{}}, nil
	}
	if err != nil {
		return State{}, err
	}
	if len(raw) == 0 {
		return State{Requests: map[string]AddressState{}}, nil
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, fmt.Errorf("failed to load faucet state: invalid json: %w", err)
	}
	if state.Requests == nil {
		state.Requests = make(map[string]AddressState)
	}
	return state, nil
}

func (s Store) saveUnlocked(state State) error {
	if state.Requests == nil {
		state.Requests = make(map[string]AddressState)
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(s.path, raw, 0644)
}

func (s Store) lock() *sync.Mutex {
	key := filepath.Clean(s.path)
	lock, _ := locks.LoadOrStore(key, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
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
