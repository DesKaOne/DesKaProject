package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/p2p"
	"deskachain/internal/storage"
	"deskachain/internal/wallet"
)

func TestRPCHealthUsesActiveProfileTestnet(t *testing.T) {
	_, server := newProfileRPCServer(t, config.Testnet())
	info := getRPCMap(t, server.URL+"/health", http.StatusOK)
	if info["network"] != "testnet" || info["network_id"] != config.Testnet().NetworkID || uint64(info["chain_id"].(float64)) != config.Testnet().ChainID {
		t.Fatalf("health did not use testnet profile: %#v", info)
	}
}

func TestHealthIncludesGenesisAndNetworkID(t *testing.T) {
	_, localServer := newProfileRPCServer(t, config.Localnet())
	local := getRPCMap(t, localServer.URL+"/health", http.StatusOK)
	_, testnetServer := newProfileRPCServer(t, config.Testnet())
	testnet := getRPCMap(t, testnetServer.URL+"/health", http.StatusOK)
	if local["network_id"] != config.Localnet().NetworkID || testnet["network_id"] != config.Testnet().NetworkID {
		t.Fatalf("unexpected network ids local=%#v testnet=%#v", local, testnet)
	}
	if local["genesis_hash"] == "" || testnet["genesis_hash"] == "" || local["genesis_hash"] == testnet["genesis_hash"] {
		t.Fatalf("unexpected genesis hashes local=%#v testnet=%#v", local, testnet)
	}
}

func TestChainInfoUsesActiveProfileTestnet(t *testing.T) {
	_, server := newProfileRPCServer(t, config.Testnet())
	info := getRPCMap(t, server.URL+"/chain/info", http.StatusOK)
	if info["network"] != "testnet" || info["network_id"] != config.Testnet().NetworkID || uint64(info["chain_id"].(float64)) != config.Testnet().ChainID {
		t.Fatalf("chain info did not use testnet profile: %#v", info)
	}
}

func TestRemoteBalanceUsesRemoteTestnetProfile(t *testing.T) {
	_, server := newProfileRPCServer(t, config.Testnet())
	testnetWallet, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	info := getRPCMap(t, server.URL+"/balance/"+testnetWallet.Address, http.StatusOK)
	if info["address"] != testnetWallet.Address || info["confirmed_balance"] != "0" {
		t.Fatalf("unexpected testnet balance response: %#v", info)
	}
	localWallet, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	bad := getRPCMap(t, server.URL+"/balance/"+localWallet.Address, http.StatusBadRequest)
	if !strings.Contains(bad["error"].(string), "wrong network version") {
		t.Fatalf("expected wrong network version, got %#v", bad)
	}
}

func TestBalanceRejectsWrongNetworkAddress(t *testing.T) {
	_, localServer := newProfileRPCServer(t, config.Localnet())
	_, testnetServer := newProfileRPCServer(t, config.Testnet())
	localWallet, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	testnetWallet, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	localBad := getRPCMap(t, localServer.URL+"/balance/"+testnetWallet.Address, http.StatusBadRequest)
	if !strings.Contains(localBad["error"].(string), "wrong network version") {
		t.Fatalf("expected localnet rejection, got %#v", localBad)
	}
	testnetBad := getRPCMap(t, testnetServer.URL+"/balance/"+localWallet.Address, http.StatusBadRequest)
	if !strings.Contains(testnetBad["error"].(string), "wrong network version") {
		t.Fatalf("expected testnet rejection, got %#v", testnetBad)
	}
}

func TestStakeInfoUsesActiveProfile(t *testing.T) {
	_, localServer := newProfileRPCServer(t, config.Localnet())
	local := getRPCMap(t, localServer.URL+"/stake/info", http.StatusOK)
	_, testnetServer := newProfileRPCServer(t, config.Testnet())
	testnet := getRPCMap(t, testnetServer.URL+"/stake/info", http.StatusOK)
	if local["min_service_stake"] == testnet["min_service_stake"] {
		t.Fatalf("stake info should differ by profile local=%#v testnet=%#v", local, testnet)
	}
	if testnet["min_service_stake"] != "1000" {
		t.Fatalf("unexpected testnet stake info: %#v", testnet)
	}
}

func TestMinerTemplateUsesActiveProfile(t *testing.T) {
	_, server := newProfileRPCServer(t, config.Testnet())
	w, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(server.URL + "/miner/template?address=" + w.Address)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("template status = %d", resp.StatusCode)
	}
	var tpl map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&tpl); err != nil {
		t.Fatal(err)
	}
	if tpl["network"] != "testnet" || tpl["network_id"] != config.Testnet().NetworkID || uint64(tpl["chain_id"].(float64)) != config.Testnet().ChainID {
		t.Fatalf("template did not use testnet profile: %#v", tpl)
	}
}

func TestRPCReorgUsesActiveProfile(t *testing.T) {
	profile := config.Testnet()
	// The fixture peer is an unauthenticated httptest server; disable production
	// P2P authentication here so the test exercises profile selection/reorg logic.
	profile.RequireAuthenticatedNode = false
	local := newProfileRPCTestNode(t, profile)
	peer := newProfileRPCTestNode(t, profile)
	mineRPCProfileBlocks(t, peer, profile, 1)
	peerMux := http.NewServeMux()
	p2p.NewServerWithAdvertiseAndProfile(peer, ":0", "", profile).Register(peerMux)
	peerServer := httptest.NewServer(peerMux)
	defer peerServer.Close()
	rpcMux := http.NewServeMux()
	RegisterHandlers(rpcMux, local, NodeInfo{RPCListen: ":0", P2PListen: ":0", Profile: profile})
	rpcServer := httptest.NewServer(rpcMux)
	defer rpcServer.Close()

	info := postRPCMap(t, rpcServer.URL+"/reorg/preview", map[string]string{"peer": peerServer.URL}, http.StatusOK)
	if info["network"] != profile.Name || info["network_id"] != profile.NetworkID || uint64(info["chain_id"].(float64)) != profile.ChainID {
		t.Fatalf("reorg preview did not use testnet profile: %#v", info)
	}
}

func TestRPCPeerSyncRejectsNetworkMismatch(t *testing.T) {
	local := newProfileRPCTestNode(t, config.Testnet())
	peer := newProfileRPCTestNode(t, config.Localnet())
	peerMux := http.NewServeMux()
	p2p.NewServerWithAdvertiseAndProfile(peer, ":0", "", config.Localnet()).Register(peerMux)
	peerServer := httptest.NewServer(peerMux)
	defer peerServer.Close()
	rpcMux := http.NewServeMux()
	RegisterHandlers(rpcMux, local, NodeInfo{RPCListen: ":0", P2PListen: ":0", Profile: config.Testnet()})
	rpcServer := httptest.NewServer(rpcMux)
	defer rpcServer.Close()

	info := postRPCMap(t, rpcServer.URL+"/peers/sync", map[string]string{"peer": peerServer.URL}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "network id mismatch") {
		t.Fatalf("expected network mismatch from peer sync, got %#v", info)
	}
}

func TestMineRPCValidateAddressWithActiveProfile(t *testing.T) {
	_, localServer := newProfileRPCServer(t, config.Localnet())
	localMiner, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	info := postRPCMap(t, localServer.URL+"/mine", map[string]any{"address": localMiner.Address, "blocks": 1}, http.StatusOK)
	if info["network"] != "localnet" {
		t.Fatalf("localnet mine did not use local profile: %#v", info)
	}

	_, testnetServer := newProfileRPCServer(t, config.Testnet())
	resp, err := http.Post(testnetServer.URL+"/mine", "application/json", strings.NewReader(`{"address":"`+localMiner.Address+`","blocks":1}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected testnet mine to reject localnet address")
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body["error"].(string), "invalid address") {
		t.Fatalf("unexpected mine validation error: %#v", body)
	}
}

func newProfileRPCServer(t *testing.T, profile config.NetworkConfig) (config.Paths, *httptest.Server) {
	t.Helper()
	paths := newProfileRPCTestNode(t, profile)
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{RPCListen: ":0", P2PListen: ":0", Profile: profile, AllowIsolatedMining: true, AllowIsolatedWrites: true})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return paths, server
}

func newProfileRPCTestNode(t *testing.T, profile config.NetworkConfig) config.Paths {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.New(store).InitWithProfile(profile); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	if err := config.WriteNetworkMetadata(paths, profile, chain.GenesisBlockForNetwork(profile).Hash); err != nil {
		t.Fatal(err)
	}
	return paths
}

func mineRPCProfileBlocks(t *testing.T, paths config.Paths, profile config.NetworkConfig, count int) {
	t.Helper()
	miner, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < count; i++ {
		store, err := storage.OpenBolt(paths.DB)
		if err != nil {
			t.Fatal(err)
		}
		bc := chain.New(store)
		block, err := bc.MineBlockWithContextAndNetwork(context.Background(), miner.Address, nil, chain.MineOptions{}, profile)
		if err != nil {
			_ = store.Close()
			t.Fatal(err)
		}
		if err := bc.AddBlockWithNetwork(block, profile); err != nil {
			_ = store.Close()
			t.Fatal(err)
		}
		_ = store.Close()
	}
}
