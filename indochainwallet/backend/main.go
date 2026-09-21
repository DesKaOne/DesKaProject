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
    mux.HandleFunc("/v1/tx/send", s.send)
    mux.HandleFunc("/v1/tx/", s.tx)
    log.Printf("IndoChainWallet backend listening on %s; RPC=%s", listen, rpcURL)
    log.Fatal(http.ListenAndServe(listen, jsonMiddleware(mux)))
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second); defer cancel()
    info, err := s.rpc.NetworkInfo(ctx)
    if err != nil { writeJSON(w, 503, map[string]any{"ok":false,"error":err.Error()}); return }
    writeJSON(w, 200, map[string]any{"ok":true,"network":info})
}
func (s *server) network(w http.ResponseWriter, r *http.Request) {
    info, err := s.rpc.NetworkInfo(r.Context())
    if err != nil { writeJSON(w, 502, map[string]any{"ok":false,"error":err.Error()}); return }
    writeJSON(w, 200, info)
}
func (s *server) balance(w http.ResponseWriter, r *http.Request) {
    address := strings.TrimSpace(r.URL.Query().Get("address"))
    if address == "" { writeJSON(w,400,map[string]any{"ok":false,"error":"address is required"}); return }
    out, err := s.rpc.Balance(r.Context(), address)
    if err != nil { writeJSON(w,502,map[string]any{"ok":false,"error":err.Error()}); return }
    writeJSON(w,200,out)
}
func (s *server) send(w http.ResponseWriter, r *http.Request) {
    var req map[string]any
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSON(w,400,map[string]any{"ok":false,"error":"invalid JSON"}); return }
    out, err := s.rpc.Send(r.Context(), mapToTx(req))
    if err != nil { writeJSON(w,502,map[string]any{"ok":false,"error":err.Error()}); return }
    writeJSON(w,200,out)
}
func (s *server) tx(w http.ResponseWriter, r *http.Request) {
    id := strings.TrimPrefix(r.URL.Path, "/v1/tx/")
    if id == "" { writeJSON(w,400,map[string]any{"ok":false,"error":"tx id is required"}); return }
    out, err := s.rpc.Tx(r.Context(), id)
    if err != nil { writeJSON(w,502,map[string]any{"ok":false,"error":err.Error()}); return }
    writeJSON(w,200,out)
}
func jsonMiddleware(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){ if r.Method==http.MethodOptions {w.WriteHeader(204);return}; w.Header().Set("Content-Type","application/json"); next.ServeHTTP(w,r) }) }
func writeJSON(w http.ResponseWriter,status int,value any){ w.WriteHeader(status); _=json.NewEncoder(w).Encode(value) }
