package rpc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"indochain/internal/config"
	"indochain/internal/wallet"
)

func TestExplorerStatusAndBlocksAtGenesis(t *testing.T) {
	_, server := newProfileRPCServer(t, config.Testnet())
	status := getRPCMap(t, server.URL+"/explorer/status", http.StatusOK)
	if status["network"] != "testnet" || status["network_id"] != config.Testnet().NetworkID || status["height"].(float64) != 0 || status["mainnet_available"] != false {
		t.Fatalf("unexpected explorer status: %#v", status)
	}
	if !strings.Contains(status["testnet_value_warning"].(string), "no monetary value") {
		t.Fatalf("missing testnet warning: %#v", status)
	}
	blocks := getRPCMap(t, server.URL+"/explorer/blocks", http.StatusOK)
	if blocks["count"].(float64) != 1 {
		t.Fatalf("expected genesis block in list: %#v", blocks)
	}
	genesis := getRPCMap(t, server.URL+"/explorer/blocks/0", http.StatusOK)
	if genesis["height"].(float64) != 0 || genesis["hash"] == "" {
		t.Fatalf("unexpected genesis detail: %#v", genesis)
	}
}

func TestExplorerBlockTxAndAddress(t *testing.T) {
	paths, server := newProfileRPCServer(t, config.Testnet())
	miner, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	fundFaucetFixture(t, paths, miner.Address, config.Testnet(), 105)
	block := faucetBlocks(t, paths)[1]
	detail := getRPCMap(t, server.URL+"/explorer/blocks/1", http.StatusOK)
	if detail["hash"] != block.Hash || detail["confirmations"].(float64) == 0 {
		t.Fatalf("unexpected block by height: %#v", detail)
	}
	byHash := getRPCMap(t, server.URL+"/explorer/block/"+block.Hash, http.StatusOK)
	if byHash["height"].(float64) != 1 {
		t.Fatalf("unexpected block by hash: %#v", byHash)
	}
	txID := block.Transactions[0].ID
	tx := getRPCMap(t, server.URL+"/explorer/tx/"+txID, http.StatusOK)
	if tx["status"] != "confirmed" || tx["type"] != "coinbase" || tx["block_height"].(float64) != 1 {
		t.Fatalf("unexpected tx detail: %#v", tx)
	}
	addr := getRPCMap(t, server.URL+"/explorer/address/"+miner.Address, http.StatusOK)
	if addr["address"] != miner.Address || addr["tx_count"].(float64) == 0 || addr["total_received"] == "0" {
		t.Fatalf("unexpected address summary: %#v", addr)
	}
	history := getRPCMap(t, server.URL+"/explorer/address/"+miner.Address+"/txs?limit=5", http.StatusOK)
	if history["count"].(float64) == 0 {
		t.Fatalf("expected address tx history: %#v", history)
	}
}

func TestExplorerStakeAndServiceSummary(t *testing.T) {
	fixture := newFaucetStakeServiceFixture(t, 1000)
	postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": fixture.ownerWallet.Address}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	postRPCMap(t, fixture.serverURL+"/stake/lock", map[string]string{"address": fixture.ownerWallet.Address, "amount": "1000"}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	runServiceAgentEquivalent(t, fixture.serverURL, fixture.ownerWallet.Address)

	stakes := getRPCMap(t, fixture.serverURL+"/explorer/address/"+fixture.ownerWallet.Address+"/stakes", http.StatusOK)
	if stakes["count"].(float64) != 1 {
		t.Fatalf("unexpected address stakes: %#v", stakes)
	}
	allStakes := getRPCMap(t, fixture.serverURL+"/explorer/stakes?status=active", http.StatusOK)
	if allStakes["count"].(float64) != 1 {
		t.Fatalf("unexpected active stakes: %#v", allStakes)
	}
	service := getRPCMap(t, fixture.serverURL+"/explorer/service/"+fixture.ownerWallet.Address, http.StatusOK)
	if service["collateral_eligible"] != true || service["simulation_only"] != true || service["scope"] != "local_node_service_store" {
		t.Fatalf("unexpected service summary: %#v", service)
	}
	services := getRPCMap(t, fixture.serverURL+"/explorer/services", http.StatusOK)
	if services["count"].(float64) != 1 {
		t.Fatalf("unexpected services list: %#v", services)
	}
}

func TestExplorerPublicRPCSafetyAndInvalidInputs(t *testing.T) {
	paths := newProfileRPCTestNode(t, config.Testnet())
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{PublicRPC: true, Profile: config.Testnet()})
	server := httptest.NewServer(mux)
	defer server.Close()
	_ = getRPCMap(t, server.URL+"/explorer/status", http.StatusOK)
	resp, err := http.Post(server.URL+"/wallet/new", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wallet new status = %d want 403", resp.StatusCode)
	}
	resp, err = http.Post(server.URL+"/explorer/status", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST explorer status = %d want 405", resp.StatusCode)
	}
	resp, err = http.Get(server.URL + "/explorer/nope")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown explorer path = %d want 404", resp.StatusCode)
	}
	_ = getRPCMap(t, server.URL+"/explorer/address/bad", http.StatusBadRequest)
	localWallet, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	wrong := getRPCMap(t, server.URL+"/explorer/address/"+localWallet.Address, http.StatusBadRequest)
	if wrong["error"] != "invalid_address" || !strings.Contains(wrong["message"].(string), "wrong network") {
		t.Fatalf("expected wrong network rejection: %#v", wrong)
	}
	_ = getRPCMap(t, server.URL+"/explorer/blocks?offset=-1", http.StatusBadRequest)
}

func TestExplorerSearchHeightHashTxAndAddress(t *testing.T) {
	paths, server := newProfileRPCServer(t, config.Testnet())
	miner, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	fundFaucetFixture(t, paths, miner.Address, config.Testnet(), 105)
	block := faucetBlocks(t, paths)[1]
	txID := block.Transactions[0].ID

	height := getRPCMap(t, server.URL+"/explorer/search?q=1", http.StatusOK)
	assertSearchResult(t, height, "block", "/explorer-ui/#/block/1")
	hash := getRPCMap(t, server.URL+"/explorer/search?q="+block.Hash, http.StatusOK)
	assertSearchResult(t, hash, "block", "/explorer-ui/#/block/1")
	tx := getRPCMap(t, server.URL+"/explorer/search?q="+txID, http.StatusOK)
	assertSearchResult(t, tx, "tx", "/explorer-ui/#/tx/"+txID)
	address := getRPCMap(t, server.URL+"/explorer/search?q="+miner.Address, http.StatusOK)
	assertSearchResult(t, address, "address", "/explorer-ui/#/address/"+miner.Address)
}

func TestExplorerSearchInvalidNotFoundAndPublicRPC(t *testing.T) {
	paths := newProfileRPCTestNode(t, config.Testnet())
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{PublicRPC: true, Profile: config.Testnet()})
	server := httptest.NewServer(mux)
	defer server.Close()

	empty := getRPCMap(t, server.URL+"/explorer/search", http.StatusBadRequest)
	if empty["error"] != "empty_query" {
		t.Fatalf("unexpected empty query error: %#v", empty)
	}
	invalid := getRPCMap(t, server.URL+"/explorer/search?q=not-a-valid-query", http.StatusBadRequest)
	if invalid["error"] != "invalid_query" {
		t.Fatalf("unexpected invalid query error: %#v", invalid)
	}
	missing := getRPCMap(t, server.URL+"/explorer/search?q=999999", http.StatusNotFound)
	if missing["error"] != "not_found" {
		t.Fatalf("unexpected not found error: %#v", missing)
	}
	localWallet, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	wrong := getRPCMap(t, server.URL+"/explorer/search?q="+localWallet.Address, http.StatusBadRequest)
	if wrong["error"] != "invalid_address" || !strings.Contains(wrong["message"].(string), "wrong network") {
		t.Fatalf("unexpected wrong network search error: %#v", wrong)
	}
	_ = getRPCMap(t, server.URL+"/explorer/search?q=0", http.StatusOK)
}

func TestExplorerPaginationMetadataAndValidation(t *testing.T) {
	paths, server := newProfileRPCServer(t, config.Testnet())
	miner, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	fundFaucetFixture(t, paths, miner.Address, config.Testnet(), 105)

	blocks := getRPCMap(t, server.URL+"/explorer/blocks?limit=1&offset=0", http.StatusOK)
	if blocks["limit"].(float64) != 1 || blocks["offset"].(float64) != 0 || blocks["count"].(float64) != 1 || blocks["next_offset"].(float64) != 1 {
		t.Fatalf("unexpected blocks pagination: %#v", blocks)
	}
	capped := getRPCMap(t, server.URL+"/explorer/blocks?limit=9999", http.StatusOK)
	if capped["limit"].(float64) != 100 {
		t.Fatalf("limit not capped: %#v", capped)
	}
	invalidLimit := getRPCMap(t, server.URL+"/explorer/blocks?limit=0", http.StatusBadRequest)
	if invalidLimit["error"] != "invalid_limit" {
		t.Fatalf("unexpected invalid limit: %#v", invalidLimit)
	}
	invalidOffset := getRPCMap(t, server.URL+"/explorer/blocks?offset=-1", http.StatusBadRequest)
	if invalidOffset["error"] != "invalid_offset" {
		t.Fatalf("unexpected invalid offset: %#v", invalidOffset)
	}
	history := getRPCMap(t, server.URL+"/explorer/address/"+miner.Address+"/txs?limit=1&offset=1", http.StatusOK)
	if history["limit"].(float64) != 1 || history["offset"].(float64) != 1 || history["prev_offset"].(float64) != 0 {
		t.Fatalf("unexpected address tx pagination: %#v", history)
	}
	stakes := getRPCMap(t, server.URL+"/explorer/stakes?limit=1&offset=0", http.StatusOK)
	if stakes["limit"].(float64) != 1 || stakes["offset"].(float64) != 0 {
		t.Fatalf("unexpected stakes pagination: %#v", stakes)
	}
	services := getRPCMap(t, server.URL+"/explorer/services?limit=1&offset=0", http.StatusOK)
	if services["limit"].(float64) != 1 || services["offset"].(float64) != 0 || services["scope"] != "local_node_service_store" {
		t.Fatalf("unexpected services pagination: %#v", services)
	}
}

func assertSearchResult(t *testing.T, result map[string]any, wantType, wantPath string) {
	t.Helper()
	results, ok := result["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("missing search results: %#v", result)
	}
	first, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected search result shape: %#v", results[0])
	}
	if first["type"] != wantType || first["path"] != wantPath {
		t.Fatalf("unexpected search result: %#v", first)
	}
}
