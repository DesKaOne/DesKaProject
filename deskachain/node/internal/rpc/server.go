package rpc

import (
	"net/http"
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
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}
