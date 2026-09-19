package p2p

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PingResult struct {
	OK                 bool   `json:"ok"`
	URL                string `json:"url"`
	HealthLatencyMS    int64  `json:"health_latency_ms,omitempty"`
	StatusLatencyMS    int64  `json:"status_latency_ms,omitempty"`
	HandshakeLatencyMS int64  `json:"handshake_latency_ms,omitempty"`
	Height             uint64 `json:"height,omitempty"`
	TipHash            string `json:"tip_hash,omitempty"`
	NodeID             string `json:"node_id,omitempty"`
	Error              string `json:"error,omitempty"`
	HealthError        string `json:"health_error,omitempty"`
	StatusError        string `json:"status_error,omitempty"`
	HandshakeError     string `json:"handshake_error,omitempty"`
}

func Ping(peer string) (PingResult, error) {
	result := PingResult{URL: peer}
	if err := ValidatePeerURL(peer); err != nil {
		result.Error = err.Error()
		return result, err
	}
	client := &http.Client{Timeout: 2 * time.Second}
	var health map[string]any
	latency, err := getJSONWithLatency(client, peer, "/p2p/health", &health)
	result.HealthLatencyMS = latency
	if err != nil {
		result.HealthError = err.Error()
		result.Error = err.Error()
		return result, err
	}
	var status Status
	latency, err = getJSONWithLatency(client, peer, "/p2p/status", &status)
	result.StatusLatencyMS = latency
	if err != nil {
		result.StatusError = err.Error()
		result.Error = err.Error()
		return result, err
	}
	result.Height = status.Height
	result.TipHash = status.TipHash
	var hs Handshake
	latency, err = getJSONWithLatency(client, peer, "/p2p/handshake", &hs)
	result.HandshakeLatencyMS = latency
	if err != nil {
		result.HandshakeError = err.Error()
		result.Error = err.Error()
		return result, err
	}
	result.NodeID = hs.NodeID
	result.OK = true
	return result, nil
}

func getJSONWithLatency(client *http.Client, peer, path string, target any) (int64, error) {
	start := time.Now()
	resp, err := client.Get(strings.TrimRight(peer, "/") + path)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return latency, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return latency, fmt.Errorf("peer returned %s", resp.Status)
	}
	return latency, json.NewDecoder(resp.Body).Decode(target)
}
