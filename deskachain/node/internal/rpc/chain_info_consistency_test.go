package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/nodestate"
	"deskachain/internal/p2p"
	"deskachain/internal/storage"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestChainInfoAfterP2PBlockImportReflectsCanonicalTip(t *testing.T) {
	profile := config.Testnet()
	nodeA := newProfileRPCTestNode(t, profile)
	nodeB := newProfileRPCTestNode(t, profile)
	block := mineChainInfoBlock(t, nodeA, profile)
	stateB := staleChainInfoState(t, nodeB, profile)

	p2pMux := http.NewServeMux()
	p2p.NewServerWithStateAndProfile(nodeB, ":0", "", stateB, profile).Register(p2pMux)
	p2pServer := httptest.NewServer(p2pMux)
	defer p2pServer.Close()
	resp, err := p2p.NewClient().BroadcastBlock(p2pServer.URL, block)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Accepted {
		t.Fatalf("p2p block rejected: %#v", resp)
	}

	rpcServer := newChainInfoRPCServer(t, nodeB, profile, stateB, false)
	info := getRPCMap(t, rpcServer.URL+"/chain/info", http.StatusOK)
	validate := getRPCMap(t, rpcServer.URL+"/chain/validate", http.StatusOK)
	assertChainInfoCanonical(t, info, validate, block)
}

func TestChainInfoAfterPeerSyncReflectsCanonicalTip(t *testing.T) {
	profile := config.Testnet()
	nodeA := newProfileRPCTestNode(t, profile)
	nodeB := newProfileRPCTestNode(t, profile)
	block := mineChainInfoBlock(t, nodeA, profile)
	stateB := staleChainInfoState(t, nodeB, profile)

	p2pMux := http.NewServeMux()
	p2p.NewServerWithAdvertiseAndProfile(nodeA, ":0", "", profile).Register(p2pMux)
	p2pServer := httptest.NewServer(p2pMux)
	defer p2pServer.Close()
	if err := p2p.SyncFromPeerWithProfile(nodeB, p2pServer.URL, nil, profile); err != nil {
		t.Fatal(err)
	}

	rpcServer := newChainInfoRPCServer(t, nodeB, profile, stateB, false)
	info := getRPCMap(t, rpcServer.URL+"/chain/info", http.StatusOK)
	validate := getRPCMap(t, rpcServer.URL+"/chain/validate", http.StatusOK)
	assertChainInfoCanonical(t, info, validate, block)
}

func TestChainInfoAfterRestartReflectsCanonicalTip(t *testing.T) {
	profile := config.Testnet()
	paths := newProfileRPCTestNode(t, profile)
	block := mineChainInfoBlock(t, paths, profile)
	restartedState, err := nodestate.NewWithProfile(paths, profile)
	if err != nil {
		t.Fatal(err)
	}

	rpcServer := newChainInfoRPCServer(t, paths, profile, restartedState, false)
	info := getRPCMap(t, rpcServer.URL+"/chain/info", http.StatusOK)
	validate := getRPCMap(t, rpcServer.URL+"/chain/validate", http.StatusOK)
	assertChainInfoCanonical(t, info, validate, block)
}

func TestChainInfoSnapshotConsistentFields(t *testing.T) {
	profile := config.Testnet()
	paths := newProfileRPCTestNode(t, profile)
	state := staleChainInfoState(t, paths, profile)
	block := mineChainInfoBlock(t, paths, profile)

	rpcServer := newChainInfoRPCServer(t, paths, profile, state, false)
	info := getRPCMap(t, rpcServer.URL+"/chain/info", http.StatusOK)
	assertChainInfoCanonicalFields(t, info, block)
	if info["height"].(float64) == 0 && info["blocks"].(float64) > 1 {
		t.Fatalf("inconsistent chain info height 0 with blocks > 1: %#v", info)
	}
	if info["tip_hash"] == chain.GenesisBlockForNetwork(profile).Hash && info["total_supply"] != "0 DKC" {
		t.Fatalf("inconsistent chain info genesis tip with non-zero supply: %#v", info)
	}
	if info["height"].(float64) == 0 && info["coinbase_blocks"].(float64) > 0 {
		t.Fatalf("inconsistent chain info height 0 with coinbase blocks: %#v", info)
	}
}

func TestPublicRPCChainInfoAfterP2PImport(t *testing.T) {
	profile := config.Testnet()
	nodeA := newProfileRPCTestNode(t, profile)
	nodeB := newProfileRPCTestNode(t, profile)
	block := mineChainInfoBlock(t, nodeA, profile)
	stateB := staleChainInfoState(t, nodeB, profile)

	p2pMux := http.NewServeMux()
	p2p.NewServerWithStateAndProfile(nodeB, ":0", "", stateB, profile).Register(p2pMux)
	p2pServer := httptest.NewServer(p2pMux)
	defer p2pServer.Close()
	resp, err := p2p.NewClient().BroadcastBlock(p2pServer.URL, block)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Accepted {
		t.Fatalf("p2p block rejected: %#v", resp)
	}

	rpcServer := newChainInfoRPCServer(t, nodeB, profile, stateB, true)
	info := getRPCMap(t, rpcServer.URL+"/chain/info", http.StatusOK)
	validate := getRPCMap(t, rpcServer.URL+"/chain/validate", http.StatusOK)
	assertChainInfoCanonical(t, info, validate, block)
}

func newChainInfoRPCServer(t *testing.T, paths config.Paths, profile config.NetworkConfig, state *nodestate.Store, public bool) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{
		RPCListen:       ":0",
		P2PListen:       ":0",
		Profile:         profile,
		State:           state,
		PublicRPC:       public,
		EnableMinerRPC:  !public,
		EnableAdminRPC:  !public,
		EnableWalletRPC: !public,
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func staleChainInfoState(t *testing.T, paths config.Paths, profile config.NetworkConfig) *nodestate.Store {
	t.Helper()
	state, err := nodestate.NewWithProfile(paths, profile)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := state.Snapshot()
	if snapshot.Height != 0 {
		t.Fatalf("test state must start at genesis, got %#v", snapshot)
	}
	return state
}

func mineChainInfoBlock(t *testing.T, paths config.Paths, profile config.NetworkConfig) types.Block {
	t.Helper()
	miner, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	bc := chain.New(store)
	block, err := bc.MineBlockWithContextAndNetwork(context.Background(), miner.Address, nil, chain.MineOptions{}, profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := bc.AddBlockWithNetwork(block, profile); err != nil {
		t.Fatal(err)
	}
	return block
}

func assertChainInfoCanonical(t *testing.T, info, validate map[string]any, block types.Block) {
	t.Helper()
	assertChainInfoCanonicalFields(t, info, block)
	if validate["valid"] != true || validate["height"].(float64) != info["height"].(float64) || validate["blocks"].(float64) != info["blocks"].(float64) {
		t.Fatalf("chain info and validate disagree: info=%#v validate=%#v", info, validate)
	}
}

func assertChainInfoCanonicalFields(t *testing.T, info map[string]any, block types.Block) {
	t.Helper()
	if info["height"].(float64) != 1 || info["tip_hash"] != block.Hash || info["tip_difficulty"].(float64) != float64(block.Difficulty) || info["blocks"].(float64) != 2 || info["total_supply"] != "50 DKC" || info["coinbase_blocks"].(float64) != 1 {
		t.Fatalf("unexpected chain info snapshot: %#v block=%#v", info, block)
	}
}
