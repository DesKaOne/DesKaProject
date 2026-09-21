package rpc

import (
	"net/http"
	"strings"
	"testing"

	"indochain/internal/chain"
	"indochain/internal/config"
	"indochain/internal/mempool"
	"indochain/internal/types"
)

func TestPhase8IntegrationNodeChainExplorerMonitoring(t *testing.T) {
	paths, server := newMinerRPCServer(t)
	miner := newRPCWallet(t)

	// Node -> miner RPC -> canonical chain.
	tpl := fetchMinerTemplate(t, server.URL, miner.Address)
	mined := chain.Mine(tpl.Block)
	submit := submitMinerBlock(t, server.URL, tpl.TemplateID, mined)
	if !submit.Accepted || submit.Height != 1 || submit.Hash != mined.Hash {
		t.Fatalf("mined block was not committed: %#v", submit)
	}

	blocks := rpcBlocks(t, paths)
	if len(blocks) != 2 || blocks[1].Hash != mined.Hash {
		t.Fatalf("canonical chain mismatch: len=%d tip=%s want=%s", len(blocks), blocks[len(blocks)-1].Hash, mined.Hash)
	}
	if _, err := chain.ValidateChain(blocks); err != nil {
		t.Fatalf("canonical chain validation failed: %v", err)
	}

	// Mempool -> monitoring snapshot.
	receiver := newRPCWallet(t)
	pending := types.NewUnsignedTransaction(miner.Address, receiver.Address, 7, 2, 1)
	if err := mempool.New(paths.Mempool).Add(pending); err != nil {
		t.Fatalf("add integration mempool transaction: %v", err)
	}

	metrics := getRPCMap(t, server.URL+"/node/metrics", http.StatusOK)
	chainInfo, ok := metrics["chain"].(map[string]any)
	if !ok || chainInfo["height"] != float64(1) || chainInfo["tip_hash"] != mined.Hash {
		t.Fatalf("monitoring chain snapshot mismatch: %#v", metrics["chain"])
	}
	mempoolInfo, ok := metrics["mempool"].(map[string]any)
	if !ok || mempoolInfo["pending_tx_count"] != float64(1) || mempoolInfo["pending_fee_total"] != float64(2) || mempoolInfo["pending_native_amount"] != float64(7) {
		t.Fatalf("monitoring mempool snapshot mismatch: %#v", metrics["mempool"])
	}

	// Canonical chain -> persistent explorer read model.
	indexer := newExplorerIndexer(paths, config.Localnet())
	status, err := indexer.sync()
	if err != nil {
		t.Fatalf("explorer indexer sync failed: %v", err)
	}
	if !status.Ready || status.IndexedHeight != 1 {
		t.Fatalf("explorer indexer not caught up: %#v", status)
	}

	indexerStats, err := indexer.stats(status)
	if err != nil {
		t.Fatalf("explorer indexer stats failed: %v", err)
	}
	if indexerStats.BlockCount != 1 || indexerStats.IndexedHeight != 1 || indexerStats.Lag != 0 {
		t.Fatalf("unexpected explorer indexer stats: %#v", indexerStats)
	}

	// Explorer API -> indexed/read-only view.
	explorerStatus := getRPCMap(t, server.URL+"/explorer/status", http.StatusOK)
	if explorerStatus["height"] != float64(1) {
		t.Fatalf("explorer status height = %#v, want 1", explorerStatus["height"])
	}
	indexedStats := getRPCMap(t, server.URL+"/explorer/indexer/stats", http.StatusOK)
	indexedStatsBody, ok := indexedStats["stats"].(map[string]any)
	if !ok || indexedStatsBody["indexed_height"] != float64(1) || indexedStatsBody["lag"] != float64(0) {
		t.Fatalf("explorer API stats mismatch: %#v", indexedStats)
	}
	blocksView := getRPCMap(t, server.URL+"/explorer/blocks?limit=2", http.StatusOK)
	if blocksView["blocks"] == nil {
		t.Fatalf("explorer block view missing blocks: %#v", blocksView)
	}

	// Monitoring must expose the same explorer read model without mutating it.
	metrics = getRPCMap(t, server.URL+"/node/metrics", http.StatusOK)
	indexerInfo, ok := metrics["indexer"].(map[string]any)
	if !ok {
		t.Fatalf("monitoring indexer section missing: %#v", metrics["indexer"])
	}
	monitoringStats, ok := indexerInfo["stats"].(map[string]any)
	if !ok || monitoringStats["indexed_height"] != float64(1) || monitoringStats["lag"] != float64(0) {
		t.Fatalf("monitoring indexer snapshot mismatch: %#v", indexerInfo)
	}

	// Explorer UI -> monitoring presentation contract.
	index := getText(t, server.URL+"/explorer-ui/", http.StatusOK)
	if !strings.Contains(index, "IndoChain Explorer") {
		t.Fatalf("explorer UI shell missing")
	}
	appJS := getText(t, server.URL+"/explorer-ui/assets/app.js", http.StatusOK)
	for _, feature := range []string{"/node/metrics", "renderMonitoring", "#/monitoring", `data-refresh="monitoring"`} {
		if !strings.Contains(appJS, feature) {
			t.Fatalf("explorer monitoring UI feature %q missing", feature)
		}
	}

	// Public monitoring contract remains read-only.
	resp, err := http.Post(server.URL+"/node/metrics", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST /node/metrics status = %d, want 405", resp.StatusCode)
	}
}
