package rpc

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"deskachain/internal/config"
	"deskachain/internal/mining"
	"deskachain/internal/nodestate"
)

type NodeInfo struct {
	RPCListen                 string
	P2PListen                 string
	P2PAdvertise              string
	PeerCount                 int
	State                     *nodestate.Store
	Mining                    *mining.Service
	PublicRPC                 bool
	EnableWalletRPC           bool
	EnableWalletRPCSet        bool
	EnableMinerRPC            bool
	EnableMinerRPCSet         bool
	EnableAdminRPC            bool
	EnableAdminRPCSet         bool
	EnableServiceRPC          bool
	EnableServiceRPCSet       bool
	EnableFaucetRPC           bool
	EnableFaucetRPCSet        bool
	FaucetAddress             string
	FaucetAmount              uint64
	FaucetMinInterval         time.Duration
	FaucetMaxPerAddress       uint64
	CORSOrigins               []string
	RateLimitPerMinute        int
	MinerRateLimitPerMinute   int
	WalletRateLimitPerMinute  int
	ServiceRateLimitPerMinute int
	MaxPeers                  int
	MaxReorgDepth             uint64
	UpstreamPeers             []string
	MinMiningPeers            int
	AllowIsolatedMining       bool
	MinWritePeers             int
	AllowIsolatedWrites       bool
	StartedAt                 time.Time
	Profile                   config.NetworkConfig
	ChainMutationMu           *sync.Mutex
}

func ListenAndServe(addr string, paths config.Paths) error {
	return ListenAndServeWithInfo(addr, paths, NodeInfo{RPCListen: addr})
}

func ListenAndServeWithInfo(addr string, paths config.Paths, info NodeInfo) error {
	server := NewHTTPServer(addr, paths, info)
	indexer := newExplorerIndexer(paths, normalizeRPCProfile(info))
	ctx, cancel := context.WithCancel(context.Background())
	go indexer.run(ctx, 2*time.Second)
	server.RegisterOnShutdown(cancel)
	return server.ListenAndServe()
}

func NewHTTPServer(addr string, paths config.Paths, info NodeInfo) *http.Server {
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, info)
	handler := http.Handler(mux)
	if normalizeRPCProfile(info).Name == "mainnet" {
		handler = mainnetLaunchGate(handler)
	}
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}


func normalizeRPCProfile(info NodeInfo) config.NetworkConfig {
	if info.Profile.Name == "" {
		return config.Localnet()
	}
	return info.Profile
}

func mainnetLaunchGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !mainnetReadOnlyRPCMethod(r.Method) || !mainnetReadOnlyRPCPath(r.URL.Path) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"ok":    false,
				"error": "mainnet is not operational: launch gate is closed",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mainnetReadOnlyRPCMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

func mainnetReadOnlyRPCPath(path string) bool {
	switch strings.TrimRight(path, "/") {
	case "/health",
		"/ready",
		"/network/info",
		"/node/id",
		"/node/compare",
		"/node/status",
		"/peer/health",
		"/peer/list",
		"/peer/seeds",
		"/peer/status",
		"/p2p/peers",
		"/p2p/reputation",
		"/p2p/discovery",
		"/p2p/bootstrap",
		"/p2p/known-peers",
		"/upstream/status",
		"/peers",
		"/chain/info",
		"/chain/difficulty",
		"/chain/locator",
		"/mining/status",
		"/mining/stats",
		"/mining/difficulty",
		"/mining/blocks",
		"/chain/blocks",
		"/chain/state",
		"/chain/validate",
		"/balance",
		"/address",
		"/asset/info",
		"/asset/balance",
		"/asset/balances",
		"/fee/policy",
		"/fee/pool",
		"/tx",
		"/mempool",
		"/mempool/list",
		"/faucet/info",
		"/wallets",
		"/mine/status",
		"/service/score",
		"/service/rewards",
		"/service/list",
		"/stake/info",
		"/stake/list",
		"/stake/status":
		return true
	default:
		return false
	}
}
