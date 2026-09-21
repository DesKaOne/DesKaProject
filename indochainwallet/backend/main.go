package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"indochainwallet/internal/client"
)

type server struct{ rpc *client.RPCClient }

type balanceResponse struct {
	OK      bool              `json:"ok"`
	Network map[string]any    `json:"network,omitempty"`
	Data    map[string]any    `json:"data,omitempty"`
	Error   string            `json:"error,omitempty"`
}

func main() {
	rpcURL := os.Getenv("INDOCHAIN_RPC_URL")
	if rpcURL == "" { rpcURL = "http://127.0.0.1:18331" }
	listen := os.Getenv("WALLET_BACKEND_ADDR")
	if listen == "" { listen = "127.0.0.1:8787" }

	s := &server{rpc: client.New(rpcURL)}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/v1/network", s.network)
	mux.HandleFunc("/v1/wallet/balance", s.balance)
	mux.HandleFunc("/v1/tx/", s.tx)
	mux.HandleFunc("/v1/tx/send", s.send)
	log.Printf("IndoChainWallet backend listening on %s; RPC=%s", listen, rpcURL)
	log.Fatal(http.ListenAndServe(listen, jsonMiddleware(mux)))
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	info, err := s.rpc.NetworkInfo(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "network": info})
}

func (s *server) network(w http.ResponseWriter, r *http.Request) {
	info, err := s.rpc.NetworkInfo(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *server) balance(w http.ResponseWriter, r *http.Request) {
	address := strings.TrimSpace(r.URL.Query().Get("address"))
	if address == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "address is required"})
		return
	}
	out, err := s.rpc.Balance(r.Context(), address)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, balanceResponse{OK: true, Data: out})
}

func (s *server) tx(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/tx/"))
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "tx id is required"})
		return
	}
	out, err := s.rpc.Tx(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) send(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req map[string]any
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}
	out, err := s.rpc.Send(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "method not allowed"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
