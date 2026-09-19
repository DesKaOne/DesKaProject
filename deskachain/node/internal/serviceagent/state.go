package serviceagent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type State struct {
	Address                 string `json:"address"`
	ServiceNodeID           string `json:"service_node_id"`
	Endpoint                string `json:"endpoint"`
	RPCURL                  string `json:"rpc_url"`
	RegisteredAt            int64  `json:"registered_at"`
	LastHeartbeatAt         int64  `json:"last_heartbeat_at"`
	LastChallengeID         string `json:"last_challenge_id"`
	LastChallengeAt         int64  `json:"last_challenge_at"`
	LastSubmitAt            int64  `json:"last_submit_at"`
	LastScore               int    `json:"last_score"`
	LastSimulatedPoints     int    `json:"last_simulated_points"`
	TotalChallenges         int    `json:"total_challenges"`
	SuccessfulChallenges    int    `json:"successful_challenges"`
	FailedChallenges        int    `json:"failed_challenges"`
	TotalSimulatedBytesUp   int64  `json:"total_simulated_bytes_up"`
	TotalSimulatedBytesDown int64  `json:"total_simulated_bytes_down"`
	ClientVersion           string `json:"client_version"`
	Platform                string `json:"platform"`
}

func LoadState(path string) (State, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}
	if len(raw) == 0 {
		return State{}, nil
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, fmt.Errorf("failed to load service agent state: invalid json: %w", err)
	}
	return state, nil
}

func SaveState(path string, state State) error {
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, raw, 0644)
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
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
