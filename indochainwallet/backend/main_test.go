package main

import (
	"context"
	"net/http"
	"net/http/httptest"
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
