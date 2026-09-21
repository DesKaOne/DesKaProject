package serviceagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"indochain/internal/config"
	"indochain/internal/wallet"
)

func TestAgentStateLoadMissingSaveLoadAndCorrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	state, err := LoadState(path)
	if err != nil {
		t.Fatal(err)
	}
	if state.Address != "" {
		t.Fatalf("missing state should be empty: %+v", state)
	}
	want := State{
		Address:                 testAddress(t),
		ServiceNodeID:           "svc_test",
		Endpoint:                "http://127.0.0.1:9501",
		RPCURL:                  "http://127.0.0.1:8431",
		LastScore:               80,
		TotalSimulatedBytesUp:   1,
		TotalSimulatedBytesDown: 2,
		ClientVersion:           "indoservice/test",
		Platform:                "test/os",
	}
	if err := SaveState(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadState(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Address != want.Address || got.ServiceNodeID != want.ServiceNodeID || got.LastScore != want.LastScore {
		t.Fatalf("state mismatch got=%+v want=%+v", got, want)
	}
	if err := os.WriteFile(path, []byte("{bad"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = LoadState(path)
	if err == nil || !strings.Contains(err.Error(), "failed to load service agent state: invalid json") {
		t.Fatalf("unexpected corrupt state error: %v", err)
	}
}

func TestMeasurementSimulatorSafeMode(t *testing.T) {
	sim := NewSimulator(true, 10_000_000)
	for i := 0; i < 20; i++ {
		m := sim.Generate()
		if m.LatencyMS < 0 || m.BytesUp < 0 || m.BytesDown < 0 {
			t.Fatalf("negative measurement: %+v", m)
		}
		if m.BytesUp > 10_000_000 || m.BytesDown > 10_000_000 {
			t.Fatalf("measurement exceeded max bytes: %+v", m)
		}
		if !m.Success {
			t.Fatalf("safe simulation should succeed by default: %+v", m)
		}
	}
}

func TestServiceRPCClientMethods(t *testing.T) {
	address := testAddress(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/service/register":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "service_node_id": "svc_1", "owner_address": address, "status": "registered"})
		case "/service/heartbeat":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": "active", "uptime_score": 100, "service_score": 30})
		case "/service/challenge/create":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "challenge_id": "ch_1", "address": address, "issued_at": 1, "expires_at": 2, "status": "pending"})
		case "/service/challenge/submit":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "challenge_id": "ch_1", "status": "passed", "service_score": 94})
		case "/service/score":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "score": map[string]any{"address": address, "service_score": 94, "simulated_points": 940, "note": "simulation only"}})
		case "/service/rewards":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "address": address, "rewards": []map[string]any{{"epoch": "2026-06-20", "service_score": 94, "simulated_points": 940, "reason": "test"}}, "note": "simulation only"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := NewClient(server.URL)
	ctx := context.Background()
	reg, err := client.Register(ctx, RegisterRequest{Address: address})
	if err != nil || reg.ServiceNodeID != "svc_1" {
		t.Fatalf("register resp=%+v err=%v", reg, err)
	}
	heartbeat, err := client.Heartbeat(ctx, RegisterRequest{Address: address})
	if err != nil || heartbeat.ServiceScore != 30 {
		t.Fatalf("heartbeat resp=%+v err=%v", heartbeat, err)
	}
	challenge, err := client.CreateChallenge(ctx, address)
	if err != nil || challenge.ChallengeID != "ch_1" {
		t.Fatalf("challenge resp=%+v err=%v", challenge, err)
	}
	submit, err := client.SubmitChallenge(ctx, "ch_1", Measurement{Success: true})
	if err != nil || submit.Status != "passed" {
		t.Fatalf("submit resp=%+v err=%v", submit, err)
	}
	score, err := client.Score(ctx, address)
	if err != nil || score.Score.ServiceScore != 94 || score.Score.SimulatedPoints != 940 {
		t.Fatalf("score resp=%+v err=%v", score, err)
	}
	rewards, err := client.Rewards(ctx, address)
	if err != nil || len(rewards.Rewards) != 1 {
		t.Fatalf("rewards resp=%+v err=%v", rewards, err)
	}
}

func TestServiceRPCDisabledError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "service RPC disabled"})
	}))
	defer server.Close()
	_, err := NewClient(server.URL).Register(context.Background(), RegisterRequest{Address: testAddress(t)})
	if err == nil || err.Error() != "service RPC disabled on target node" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgentOnceWorkflow(t *testing.T) {
	address := testAddress(t)
	var registerCalls, heartbeatCalls, createCalls, submitCalls, scoreCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "network": "localnet", "service_rpc": true})
		case "/service/register":
			registerCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "service_node_id": "svc_1", "owner_address": address, "status": "registered"})
		case "/service/heartbeat":
			heartbeatCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": "active", "uptime_score": 100, "service_score": 30})
		case "/service/challenge/create":
			createCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "challenge_id": "ch_1", "address": address, "issued_at": 1, "expires_at": 2, "status": "pending"})
		case "/service/challenge/submit":
			submitCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "challenge_id": "ch_1", "status": "passed", "service_score": 94})
		case "/service/score":
			scoreCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "score": map[string]any{"address": address, "service_score": 94, "simulated_points": 940, "note": "service points are simulation only and are not spendable dIDR"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	statePath := filepath.Join(t.TempDir(), "agent.json")
	var out strings.Builder
	agent := Agent{
		Options: Options{
			RPCURL:               server.URL,
			Address:              address,
			Endpoint:             "http://127.0.0.1:9501",
			StatePath:            statePath,
			HeartbeatInterval:    time.Second,
			ChallengeInterval:    time.Second,
			ScoreInterval:        time.Second,
			RetryInterval:        time.Millisecond,
			Once:                 true,
			SafeMode:             true,
			MaxBytesPerChallenge: 100_000_000,
			ClientVersion:        "indoservice/test",
			Platform:             "test/os",
		},
		Client: NewClient(server.URL),
		Out:    &out,
	}
	if err := agent.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if registerCalls.Load() != 1 || heartbeatCalls.Load() != 1 || createCalls.Load() != 1 || submitCalls.Load() != 1 || scoreCalls.Load() != 1 {
		t.Fatalf("unexpected calls register=%d heartbeat=%d create=%d submit=%d score=%d", registerCalls.Load(), heartbeatCalls.Load(), createCalls.Load(), submitCalls.Load(), scoreCalls.Load())
	}
	state, err := LoadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if state.ServiceNodeID != "svc_1" || state.LastScore != 94 || state.LastSimulatedPoints != 940 || state.SuccessfulChallenges != 1 {
		t.Fatalf("unexpected state: %+v", state)
	}
	if !strings.Contains(out.String(), "state saved") || !strings.Contains(out.String(), "service points are simulation only") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestAgentRetryRegister(t *testing.T) {
	address := testAddress(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
			return
		}
		calls.Add(1)
		if calls.Load() == 1 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "temporary", http.StatusServiceUnavailable)
				return
			}
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
			return
		}
		switch r.URL.Path {
		case "/service/register":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "service_node_id": "svc_retry", "owner_address": address, "status": "registered"})
		case "/service/heartbeat":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": "active", "service_score": 1})
		case "/service/challenge/create":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "challenge_id": "ch_retry", "status": "pending"})
		case "/service/challenge/submit":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "challenge_id": "ch_retry", "status": "passed", "service_score": 1})
		case "/service/score":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "score": map[string]any{"address": address, "service_score": 1, "simulated_points": 10}})
		}
	}))
	defer server.Close()
	err := Agent{
		Options: Options{
			RPCURL:               server.URL,
			Address:              address,
			StatePath:            filepath.Join(t.TempDir(), "agent.json"),
			RetryInterval:        time.Millisecond,
			Once:                 true,
			SafeMode:             true,
			MaxBytesPerChallenge: 100,
			ClientVersion:        "test",
			Platform:             "test",
		},
		Client: NewClient(server.URL),
		Out:    ioDiscard{},
	}.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() < 2 {
		t.Fatalf("expected retry, got %d calls", calls.Load())
	}
}

func testAddress(t *testing.T) string {
	t.Helper()
	w, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	return w.Address
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
