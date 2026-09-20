package rpc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"deskachain/internal/config"
)

func TestMainnetReadOnlyRPCPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/network/info", true},
		{"/health", true},
		{"/chain/info", true},
		{"/mempool", true},
		{"/stake/status", true},
		{"/send", false},
		{"/wallet/new", false},
		{"/mine", false},
		{"/miner/template", false},
		{"/peers/connect", false},
		{"/stake/lock", false},
		{"/service/heartbeat", false},
	}
	for _, tc := range cases {
		if got := mainnetReadOnlyRPCPath(tc.path); got != tc.want {
			t.Fatalf("mainnetReadOnlyRPCPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestMainnetReadOnlyRPCMethod(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		if !mainnetReadOnlyRPCMethod(method) {
			t.Fatalf("mainnetReadOnlyRPCMethod(%q) = false, want true", method)
		}
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		if mainnetReadOnlyRPCMethod(method) {
			t.Fatalf("mainnetReadOnlyRPCMethod(%q) = true, want false", method)
		}
	}
}

func TestMainnetLaunchGateBlocksNonReadOnlyRPC(t *testing.T) {
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
		t.Fatal("mainnet launch gate allowed a non-read-only RPC request")
	}
}

func TestMainnetLaunchGateBlocksMutationMethodsEvenOnUnknownPaths(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		nextCalled := false
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })
		req := httptest.NewRequest(method, "/future/mutation-endpoint", nil)
		rec := httptest.NewRecorder()
		mainnetLaunchGate(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s status = %d, want %d", method, rec.Code, http.StatusServiceUnavailable)
		}
		if nextCalled {
			t.Fatalf("%s reached downstream handler", method)
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

func TestMainnetLaunchGateAllowsReadOnlyMethodsOnReadOnlyPaths(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})
		req := httptest.NewRequest(method, "/network/info", nil)
		rec := httptest.NewRecorder()
		mainnetLaunchGate(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", method, rec.Code, http.StatusOK)
		}
		if !nextCalled {
			t.Fatalf("%s read-only request was blocked", method)
		}
	}
}

func TestNormalizeRPCProfileDefaultsToLocalnet(t *testing.T) {
	profile := normalizeRPCProfile(NodeInfo{})
	if profile.Name != config.Localnet().Name {
		t.Fatalf("profile = %q, want %q", profile.Name, config.Localnet().Name)
	}
}
