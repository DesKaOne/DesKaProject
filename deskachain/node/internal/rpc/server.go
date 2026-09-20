package rpc

import (
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
	return NewHTTPServer(addr, paths, info).ListenAndServe()
}

func NewHTTPServer(addr string, paths config.Paths, info NodeInfo) *http.Server {
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, info)
	handler := http.Handler(mux)
	if normalizeRPCProfile(info).Name == "mainnet" && !mainnetRPCOperational(info) {
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

func mainnetRPCOperational(info NodeInfo) bool {
	return false
}

func normalizeRPCProfile(info NodeInfo) config.NetworkConfig {
	if info.Profile.Name == "" {
		return config.Localnet()
	}
	return info.Profile
}

func mainnetLaunchGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodDelete {
			if mainnetOperationalRPCPath(r.URL.Path) {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{
					"ok":    false,
					"error": "mainnet is not operational: launch gate is closed",
				})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func mainnetOperationalRPCPath(path string) bool {
	switch strings.TrimRight(path, "/") {
	case "/debug/p2p/ping",
		"/upstream/push",
		"/upstream/push-all",
		"/peers",
		"/peers/connect",
		"/peers/clear",
		"/peers/discover",
		"/peers/sync",
		"/reorg/preview",
		"/reorg/apply",
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
		"/stake/unlock":
		return true
	default:
		return false
	}
}
