package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"indochainwallet/internal/client"
)

func TestBackendWalletSendRelayShape(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"tx_id":"abc"}`))
	}))
	defer srv.Close()

	out, err := client.New(srv.URL).Send(context.Background(), map[string]any{
		"version": 3,
		"from": "from",
		"to": "to",
		"amount": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := out["tx_id"].(string); !ok || got != "abc" {
		t.Fatalf("unexpected response: %v", out)
	}
}

func TestBackendRejectsPrivateKeyPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	// The client itself is intentionally generic; the rejection belongs to the
	// backend HTTP handler and is covered by the handler-level integration test.
	s := &server{rpc: client.New(srv.URL)}
	req := httptest.NewRequest(http.MethodPost, "/v1/tx/send", strings.NewReader(`{"private_key":"secret","from":"A","to":"B"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.send(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
