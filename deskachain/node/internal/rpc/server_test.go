package rpc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"deskachain/internal/config"
)

func TestMainnetOperationalRPCPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/send", true},
		{"/wallet/new", true},
		{"/mine", true},
		{"/miner/template", true},
		{"/peers/", true},
		{"/stake/lock", true},
		{"/service/heartbeat", true},
		{"/network/info", false},
		{"/health", false},
		{"/stake/status", false},
		{"/faucet/info", false},
		{"/chain/info", false},
	}
	for _, tc := range cases {
		if got := mainnetOperationalRPCPath(tc.path); got != tc.want {
			t.Fatalf("mainnetOperationalRPCPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestMainnetLaunchGateBlocksOperationalRPC(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodPost, "/send", nil)
	rec := httptest.NewRecorder()
	mainnetLaunchGate(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if nextCalled {
		t.Fatal("mainnet launch gate allowed an operational RPC request")
	}
}

func TestMainnetLaunchGateCoversRegisteredMutationEndpoints(t *testing.T) {
	paths := []string{
		"/debug/p2p/ping",
		"/upstream/push",
		"/upstream/push-all",
		"/peers",
		"/peers/connect",
		"/peers/clear",
		"/peers/discover",
		"/peers/status",
		"/peers/sync",
		"/reorg/preview",
		"/reorg/apply",
		"/chain/common-ancestor",
		"/fork/inspect-datadir",
		"/mempool/clear",
		"/faucet/request",
		"/wallet/new",
		"/send",
		"/mine",
		"/miner/template",
		"/miner/submit",
		"/service/register",
		"/service/heartbeat",
		"/service/challenge/create",
		"/service/challenge/submit",
		"/stake/lock",
		"/stake/unlock",
	}
	for _, path := range paths {
		if !mainnetOperationalRPCPath(path) {
			t.Fatalf("mainnetOperationalRPCPath(%q) = false, want true", path)
		}
	}
}

func TestMainnetLaunchGateAllowsReadOnlyRPC(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/network/info", nil)
	rec := httptest.NewRecorder()
	mainnetLaunchGate(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !nextCalled {
		t.Fatal("mainnet launch gate blocked a read-only RPC request")
	}
}

func TestNormalizeRPCProfileDefaultsToLocalnet(t *testing.T) {
	profile := normalizeRPCProfile(NodeInfo{})
	if profile.Name != config.Localnet().Name {
		t.Fatalf("profile = %q, want %q", profile.Name, config.Localnet().Name)
	}
}
