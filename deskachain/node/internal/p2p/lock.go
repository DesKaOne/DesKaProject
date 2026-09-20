package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type LockInfo struct {
	PID          int    `json:"pid"`
	RPC          string `json:"rpc"`
	P2P          string `json:"p2p"`
	P2PAdvertise string `json:"p2p_advertise"`
	StartedAt    string `json:"started_at"`
}

func CreateLock(path, rpcAddr, p2pAddr string, advertise ...string) (LockInfo, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return LockInfo{}, err
	}

	info := LockInfo{
		PID:          os.Getpid(),
		RPC:          rpcAddr,
		P2P:          p2pAddr,
		P2PAdvertise: firstString(advertise),
		StartedAt:    time.Now().Format(time.RFC3339),
	}
	raw, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return LockInfo{}, err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			existing, readErr := ReadLock(path)
			if readErr == nil {
				return existing, errors.New("datadir is already locked")
			}
			return LockInfo{}, fmt.Errorf("datadir lock already exists: %w", readErr)
		}
		return LockInfo{}, err
	}
	defer file.Close()

	if _, err := file.Write(raw); err != nil {
		_ = os.Remove(path)
		return LockInfo{}, err
	}
	return info, nil
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func ReadLock(path string) (LockInfo, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return LockInfo{}, err
	}
	var info LockInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return LockInfo{}, err
	}
	return info, nil
}

func RemoveLock(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func IsLocked(path string) (LockInfo, bool) {
	info, err := ReadLock(path)
	return info, err == nil
}
