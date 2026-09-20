package rpc

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/p2p"
	"deskachain/internal/storage"
	"deskachain/internal/wallet"
)

func TestRPCServerTimeoutConfig(t *testing.T) {
	server := NewHTTPServer(":0", config.NewPaths(t.TempDir()), NodeInfo{})
	if server.ReadHeaderTimeout <= 0 || server.ReadTimeout <= 0 || server.WriteTimeout <= 0 || server.IdleTimeout <= 0 || server.MaxHeaderBytes <= 0 {
		t.Fatalf("server timeouts/header limit not configured: %#v", server)
	}
}

func TestPublicRPCDisablesWalletButAllowsReadOnly(t *testing.T) {
	_, server := newHardeningRPCServer(t, NodeInfo{PublicRPC: true})
	resp, err := http.Post(server.URL+"/wallet/new", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wallet status = %d want 403", resp.StatusCode)
	}
	info, err := http.Get(server.URL + "/chain/info")
	if err != nil {
		t.Fatal(err)
	}
	defer info.Body.Close()
	if info.StatusCode != http.StatusOK {
		t.Fatalf("chain info status = %d", info.StatusCode)
	}
}

func TestPublicRPCSendIsDisabledWithWalletRPC(t *testing.T) {
	_, server := newHardeningRPCServer(t, NodeInfo{PublicRPC: true})
	resp, err := http.Post(server.URL+"/send", "application/json", strings.NewReader(`{"from":"IDR-invalid","to":"IDR-invalid","amount":"1"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("public send status = %d want 403", resp.StatusCode)
	}
}

func TestPublicRPCMinerEnableDisable(t *testing.T) {
	miner := newRPCWallet(t)
	_, defaultDisabled := newHardeningRPCServer(t, NodeInfo{PublicRPC: true})
	resp, err := http.Get(defaultDisabled.URL + "/miner/template?address=" + miner.Address)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("public miner default status = %d want 403", resp.StatusCode)
	}
	_, enabled := newHardeningRPCServer(t, NodeInfo{PublicRPC: true, EnableMinerRPC: true, EnableMinerRPCSet: true})
	resp, err = http.Get(enabled.URL + "/miner/template?address=" + miner.Address)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("miner enabled status = %d", resp.StatusCode)
	}
	_, disabled := newHardeningRPCServer(t, NodeInfo{PublicRPC: true, EnableMinerRPC: false, EnableMinerRPCSet: true})
	resp, err = http.Get(disabled.URL + "/miner/template?address=" + miner.Address)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("miner disabled status = %d want 403", resp.StatusCode)
	}
}

func TestHealthIncludesRPCModeSummary(t *testing.T) {
	_, server := newHardeningRPCServer(t, NodeInfo{PublicRPC: true})
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var info map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"public_rpc", "wallet_rpc", "miner_rpc", "admin_rpc", "faucet_rpc", "service_rpc"} {
		if _, ok := info[key]; !ok {
			t.Fatalf("health missing %s: %#v", key, info)
		}
	}
	if info["public_rpc"] != true || info["wallet_rpc"] != false || info["miner_rpc"] != false || info["admin_rpc"] != false || info["service_rpc"] != false {
		t.Fatalf("unexpected public RPC mode summary: %#v", info)
	}
}

func TestRPCRateLimitAndCORS(t *testing.T) {
	_, server := newHardeningRPCServer(t, NodeInfo{RateLimitPerMinute: 1, CORSOrigins: []string{"http://localhost:3000"}})
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("missing cors header")
	}
	resp, err = http.Get(server.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("rate limited status = %d want 429", resp.StatusCode)
	}
}

func TestAssetBalancesRPCUsesGenericRateLimit(t *testing.T) {
	address := newRPCWallet(t).Address
	_, server := newHardeningRPCServer(t, NodeInfo{RateLimitPerMinute: 1})
	url := server.URL + "/asset/balances?address=" + address
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first asset balances status = %d want 200", resp.StatusCode)
	}
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("second asset balances status = %d want 429", resp.StatusCode)
	}
}

func TestRPCBodyLimit(t *testing.T) {
	_, server := newHardeningRPCServer(t, NodeInfo{})
	body := `{"from":"` + strings.Repeat("x", 1024*1024+1) + `"}`
	resp, err := http.Post(server.URL+"/send", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status = %d want 413", resp.StatusCode)
	}
}

func TestHealthEndpoint(t *testing.T) {
	_, server := newHardeningRPCServer(t, NodeInfo{StartedAt: time.Now().Add(-time.Second)})
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", resp.StatusCode)
	}
}

func TestHealthEndpointIncludesServiceSummary(t *testing.T) {
	_, server := newHardeningRPCServer(t, NodeInfo{})
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var info map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if _, ok := info["service_nodes"]; !ok {
		t.Fatalf("health missing service_nodes: %+v", info)
	}
	if _, ok := info["service_rpc"]; !ok {
		t.Fatalf("health missing service_rpc: %+v", info)
	}
}

func TestPublicRPCServiceDisabledAndEnabled(t *testing.T) {
	address := newRPCWallet(t).Address
	_, disabled := newHardeningRPCServer(t, NodeInfo{PublicRPC: true})
	resp, err := http.Post(disabled.URL+"/service/register", "application/json", strings.NewReader(`{"address":"`+address+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(string(body), "service RPC disabled") {
		t.Fatalf("service disabled status=%d body=%s", resp.StatusCode, body)
	}

	_, enabled := newHardeningRPCServer(t, NodeInfo{PublicRPC: true, EnableServiceRPC: true, EnableServiceRPCSet: true})
	resp, err = http.Post(enabled.URL+"/service/register", "application/json", strings.NewReader(`{"address":"`+address+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("service enabled status = %d", resp.StatusCode)
	}
}

func TestPublicRPCFaucetServiceTogglesDoNotEnableWalletAdminOrMiner(t *testing.T) {
	testnetWallet, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	address := testnetWallet.Address
	_, server := newHardeningRPCServer(t, NodeInfo{
		PublicRPC:           true,
		Profile:             config.Testnet(),
		EnableFaucetRPC:     true,
		EnableFaucetRPCSet:  true,
		FaucetAddress:       address,
		FaucetAmount:        100 * config.UnitsPerCoin,
		EnableServiceRPC:    true,
		EnableServiceRPCSet: true,
	})
	resp, err := http.Post(server.URL+"/wallet/new", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wallet status = %d want 403", resp.StatusCode)
	}
	resp, err = http.Get(server.URL + "/debug/locks")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("admin status = %d want 403", resp.StatusCode)
	}
	resp, err = http.Get(server.URL + "/miner/template?address=" + address)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("miner status = %d want 403", resp.StatusCode)
	}
	resp, err = http.Post(server.URL+"/service/register", "application/json", strings.NewReader(`{"address":"`+address+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("service status = %d want 200", resp.StatusCode)
	}
	info := getRPCMap(t, server.URL+"/faucet/info", http.StatusOK)
	if info["enabled"] != true {
		t.Fatalf("faucet info not enabled: %#v", info)
	}
}

func TestServiceRequestBodyLimit(t *testing.T) {
	_, server := newHardeningRPCServer(t, NodeInfo{})
	body := `{"address":"` + strings.Repeat("x", 128*1024+1) + `"}`
	resp, err := http.Post(server.URL+"/service/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized service body status = %d want 413", resp.StatusCode)
	}
}

func TestPublicRPCStakeWriteDisabled(t *testing.T) {
	address := newRPCWallet(t).Address
	_, server := newHardeningRPCServer(t, NodeInfo{PublicRPC: true})
	resp, err := http.Get(server.URL + "/stake/info")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stake info status = %d", resp.StatusCode)
	}
	resp, err = http.Post(server.URL+"/stake/lock", "application/json", strings.NewReader(`{"address":"`+address+`","amount":"10"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("stake lock status = %d want 403", resp.StatusCode)
	}
}

func TestPublicRPCPeerDiagnosticsAllowedAndWritesRejected(t *testing.T) {
	paths, server := newHardeningRPCServer(t, NodeInfo{PublicRPC: true})
	if err := p2p.NewPeerStore(paths.Peers).Upsert(p2p.PeerMetadata{
		URL:           "http://127.0.0.1:10311",
		Source:        "seed",
		Status:        p2p.PeerStatusActive,
		Score:         5,
		LastHeight:    7,
		LastTipHash:   "abc",
		NetworkID:     config.Localnet().NetworkID,
		ChainID:       config.Localnet().ChainID,
		GenesisHash:   chain.GenesisBlockForNetwork(config.Localnet()).Hash,
		LastLatencyMS: 12,
	}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/peer/health", "/peer/list", "/peer/seeds", "/peer/status", "/peers", "/p2p/peers", "/p2p/reputation", "/p2p/discovery", "/p2p/bootstrap", "/p2p/known-peers"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d want 200", path, resp.StatusCode)
		}
	}
	health := getRPCMap(t, server.URL+"/peer/health", http.StatusOK)
	if numberFromTestAny(health["known_peer_count"]) != 1 || numberFromTestAny(health["seed_count"]) != 1 || numberFromTestAny(health["best_peer_height"]) != 7 {
		t.Fatalf("unexpected peer health: %#v", health)
	}
	p2pPeers := getRPCMap(t, server.URL+"/p2p/peers", http.StatusOK)
	if numberFromTestAny(p2pPeers["known_peer_count"]) != 1 || numberFromTestAny(p2pPeers["seed_count"]) != 1 {
		t.Fatalf("unexpected p2p peers: %#v", p2pPeers)
	}
	reputation := getRPCMap(t, server.URL+"/p2p/reputation", http.StatusOK)
	if numberFromTestAny(reputation["peer_count"]) != 1 || numberFromTestAny(reputation["reputation_count"]) != 1 {
		t.Fatalf("unexpected p2p reputation: %#v", reputation)
	}
	discovery := getRPCMap(t, server.URL+"/p2p/discovery", http.StatusOK)
	if numberFromTestAny(discovery["known_peer_count"]) != 1 || discovery["maintenance"] != "enabled" {
		t.Fatalf("unexpected p2p discovery: %#v", discovery)
	}
	bootstrap := getRPCMap(t, server.URL+"/p2p/bootstrap", http.StatusOK)
	if numberFromTestAny(bootstrap["seed_count"]) != 1 {
		t.Fatalf("unexpected p2p bootstrap: %#v", bootstrap)
	}
	known := getRPCMap(t, server.URL+"/p2p/known-peers", http.StatusOK)
	if numberFromTestAny(known["known_peer_count"]) != 1 {
		t.Fatalf("unexpected p2p known peers: %#v", known)
	}
	upstreamStatus := getRPCMap(t, server.URL+"/upstream/status", http.StatusOK)
	if numberFromTestAny(upstreamStatus["upstream_peer_count"]) != 0 || numberFromTestAny(upstreamStatus["active_peer_count"]) != 0 {
		t.Fatalf("unexpected upstream status: %#v", upstreamStatus)
	}
	resp, err := http.Post(server.URL+"/upstream/push", "application/json", strings.NewReader(`{"peer":"http://127.0.0.1:10311"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("upstream push status = %d want 403", resp.StatusCode)
	}
	resp, err = http.Post(server.URL+"/upstream/push-all", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("upstream push-all status = %d want 403", resp.StatusCode)
	}
	resp, err = http.Post(server.URL+"/peers/discover", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("discover write status = %d want 403", resp.StatusCode)
	}
	resp, err = http.Post(server.URL+"/peers/sync", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("peer sync write status = %d want 403", resp.StatusCode)
	}
}

func newHardeningRPCServer(t *testing.T, info NodeInfo) (config.Paths, *httptest.Server) {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.New(store).Init(); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, info)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return paths, server
}

func numberFromTestAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}
