package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"deskachain/internal/amount"
	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/mempool"
	"deskachain/internal/p2p"
	"deskachain/internal/storage"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

type minerTemplateTestResponse struct {
	TemplateID   string      `json:"template_id"`
	Height       uint64      `json:"height"`
	PreviousHash string      `json:"previous_hash"`
	Difficulty   uint32      `json:"difficulty"`
	TxCount      int         `json:"tx_count"`
	Block        types.Block `json:"block"`
}

type minerSubmitTestResponse struct {
	Accepted      bool   `json:"accepted"`
	Reason        string `json:"reason"`
	Height        uint64 `json:"height"`
	Hash          string `json:"hash"`
	CurrentHeight uint64 `json:"current_height"`
	CurrentTip    string `json:"current_tip"`
	Duplicate     bool   `json:"duplicate"`
}

func TestMinerTemplateAtGenesisAndInvalidAddress(t *testing.T) {
	paths, server := newMinerRPCServer(t)
	miner := newRPCWallet(t)
	tpl := fetchMinerTemplate(t, server.URL, miner.Address)
	if tpl.Height != 1 || tpl.PreviousHash != chain.GenesisBlock().Hash || tpl.Difficulty != config.Localnet().Difficulty.InitialDifficulty || tpl.TxCount != 1 {
		t.Fatalf("unexpected template: %#v", tpl)
	}
	resp, err := http.Get(server.URL + "/miner/template?address=bad")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid address status = %d", resp.StatusCode)
	}
	_ = paths
}

func TestMinerSubmitValidBlockAndImmatureReward(t *testing.T) {
	paths, server := newMinerRPCServer(t)
	miner := newRPCWallet(t)
	tpl := fetchMinerTemplate(t, server.URL, miner.Address)
	mined := chain.Mine(tpl.Block)
	submit := submitMinerBlockWithTimeout(t, server.URL, tpl.TemplateID, mined, 2*time.Second)
	if !submit.Accepted || submit.Height != 1 || submit.Hash != mined.Hash {
		t.Fatalf("submit rejected: %#v", submit)
	}
	next := fetchMinerTemplateWithTimeout(t, server.URL, miner.Address, 2*time.Second)
	if next.Height != 2 {
		t.Fatalf("template after submit height = %d want 2", next.Height)
	}
	blocks := rpcBlocks(t, paths)
	if blocks[len(blocks)-1].Height != 1 {
		t.Fatalf("height not advanced")
	}
	details, err := ledger.BalanceDetailsFor(miner.Address, blocks, nil, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	if details.Confirmed != config.InitialBlockReward || details.Immature != config.InitialBlockReward || details.Spendable != 0 {
		t.Fatalf("unexpected balance details: %#v", details)
	}
	if _, err := chain.ValidateChain(blocks); err != nil {
		t.Fatal(err)
	}
}

func TestMinerTemplateRejectsWrongNetworkRewardAddress(t *testing.T) {
	_, server := newProfileRPCServer(t, config.Testnet())
	localWallet, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(server.URL + "/miner/template?address=" + localWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("template status = %d, want bad request", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body["error"].(string), "wrong network version") {
		t.Fatalf("expected wrong network version, got %#v", body)
	}
}

func TestMinerSubmitRejectsWrongNetworkCoinbaseRecipient(t *testing.T) {
	paths, server := newProfileRPCServer(t, config.Testnet())
	testnetWallet, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	localWallet, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	tpl := fetchMinerTemplate(t, server.URL, testnetWallet.Address)
	block := tpl.Block
	block.Transactions[0].To = localWallet.Address
	block.Transactions[0].RefreshID()
	block.MerkleRoot = types.CalculateMerkleRoot(block.Transactions)
	block = chain.Mine(block)
	submit := submitMinerBlockWithTimeout(t, server.URL, tpl.TemplateID, block, 2*time.Second)
	if submit.Accepted || !strings.Contains(submit.Reason, "wrong network version") {
		t.Fatalf("expected wrong-network coinbase rejection, got %#v", submit)
	}
	if got := rpcBlocks(t, paths); got[len(got)-1].Height != 0 {
		t.Fatalf("chain height changed after rejected submit")
	}
}

func TestMinerSubmitAcceptsTestnetRewardAddress(t *testing.T) {
	paths, server := newProfileRPCServer(t, config.Testnet())
	testnetWallet, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	tpl := fetchMinerTemplate(t, server.URL, testnetWallet.Address)
	submit := submitMinerBlockWithTimeout(t, server.URL, tpl.TemplateID, chain.Mine(tpl.Block), 2*time.Second)
	if !submit.Accepted {
		t.Fatalf("testnet submit rejected: %#v", submit)
	}
	if got := rpcBlocks(t, paths); got[len(got)-1].Height != 1 {
		t.Fatalf("height = %d, want 1", got[len(got)-1].Height)
	}
}

func TestMiningGuardRejectsIsolatedTestnetTemplate(t *testing.T) {
	paths := newProfileRPCTestNode(t, config.Testnet())
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{RPCListen: ":0", P2PListen: ":0", Profile: config.Testnet(), EnableMinerRPC: true, EnableMinerRPCSet: true, MinMiningPeers: 1})
	server := httptest.NewServer(mux)
	defer server.Close()
	miner, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(server.URL + "/miner/template?address=" + miner.Address)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("template status = %d, want 503", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body["error"].(string), "isolated mining is disabled") {
		t.Fatalf("unexpected guard response: %#v", body)
	}
}

func TestMiningGuardAllowsTestnetWithActivePeer(t *testing.T) {
	paths := newProfileRPCTestNode(t, config.Testnet())
	if err := p2p.NewPeerStore(paths.Peers).Upsert(p2p.PeerMetadata{
		URL:         "http://127.0.0.1:10311",
		Status:      p2p.PeerStatusActive,
		NetworkID:   config.Testnet().NetworkID,
		ChainID:     config.Testnet().ChainID,
		GenesisHash: chain.GenesisBlockForNetwork(config.Testnet()).Hash,
	}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{RPCListen: ":0", P2PListen: ":0", Profile: config.Testnet(), EnableMinerRPC: true, EnableMinerRPCSet: true, MinMiningPeers: 1})
	server := httptest.NewServer(mux)
	defer server.Close()
	miner, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(server.URL + "/miner/template?address=" + miner.Address)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("template status = %d, want 200", resp.StatusCode)
	}
}

func TestMiningGuardAllowsTestnetWithReachableUpstream(t *testing.T) {
	peerPaths := newProfileRPCTestNode(t, config.Testnet())
	peerMux := http.NewServeMux()
	p2p.NewServerWithAdvertiseAndProfile(peerPaths, ":0", "", config.Testnet()).Register(peerMux)
	peerServer := httptest.NewServer(peerMux)
	defer peerServer.Close()

	paths := newProfileRPCTestNode(t, config.Testnet())
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{RPCListen: ":0", P2PListen: ":0", Profile: config.Testnet(), EnableMinerRPC: true, EnableMinerRPCSet: true, MinMiningPeers: 1, UpstreamPeers: []string{peerServer.URL}})
	server := httptest.NewServer(mux)
	defer server.Close()
	miner, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(server.URL + "/miner/template?address=" + miner.Address)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("template status = %d, want 200", resp.StatusCode)
	}
}

func TestCommittedBlocksAlwaysValidateCoinbaseRecipientWithProfile(t *testing.T) {
	profile := config.Testnet()
	testnetWallet, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	localWallet, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	genesis := chain.GenesisBlockForNetwork(profile)
	coinbase := types.NewCoinbaseTransaction(localWallet.Address, config.InitialBlockReward, 1)
	block := types.NewBlock(1, genesis.Hash, testnetWallet.Address, profile.Difficulty.InitialDifficulty, []types.Transaction{coinbase})
	block.Timestamp = genesis.Timestamp + profile.Difficulty.TargetBlockTimeSeconds
	block = chain.Mine(block)
	_, err = chain.ValidateChainWithNetwork([]types.Block{genesis, block}, profile)
	if err == nil || !strings.Contains(err.Error(), "wrong network version") {
		t.Fatalf("expected wrong-network coinbase validation error, got %v", err)
	}
}

func TestMinerSubmitInvalidPoWStaleAndWrongDifficulty(t *testing.T) {
	paths, server := newMinerRPCServer(t)
	miner := newRPCWallet(t)
	nextMiner := newRPCWallet(t)
	tpl := fetchMinerTemplate(t, server.URL, miner.Address)

	badPoW := tpl.Block
	badPoW.Hash = "bad"
	submit := submitMinerBlockWithTimeout(t, server.URL, tpl.TemplateID, badPoW, time.Second)
	if submit.Accepted || submit.Reason != "invalid proof of work" {
		t.Fatalf("expected invalid pow, got %#v", submit)
	}
	if next := fetchMinerTemplateWithTimeout(t, server.URL, miner.Address, time.Second); next.Height != 1 {
		t.Fatalf("template after invalid pow height = %d want 1", next.Height)
	}

	wrongDifficulty := tpl.Block
	wrongDifficulty.Difficulty++
	wrongDifficulty.MerkleRoot = types.CalculateMerkleRoot(wrongDifficulty.Transactions)
	wrongDifficulty = chain.Mine(wrongDifficulty)
	submit = submitMinerBlockWithTimeout(t, server.URL, "", wrongDifficulty, time.Second)
	if submit.Accepted || !strings.Contains(submit.Reason, "invalid difficulty") {
		t.Fatalf("expected invalid difficulty, got %#v", submit)
	}

	staleCandidate := chain.Mine(tpl.Block)
	fresh := fetchMinerTemplate(t, server.URL, nextMiner.Address)
	freshMined := chain.Mine(fresh.Block)
	if !submitMinerBlock(t, server.URL, fresh.TemplateID, freshMined).Accepted {
		t.Fatalf("fresh block not accepted")
	}
	submit = submitMinerBlockWithTimeout(t, server.URL, tpl.TemplateID, staleCandidate, time.Second)
	if submit.Accepted || submit.Duplicate || submit.Reason != "stale template" || submit.CurrentHeight != 1 {
		t.Fatalf("expected stale template, got %#v", submit)
	}
	start := time.Now()
	if next := fetchMinerTemplateWithTimeout(t, server.URL, miner.Address, time.Second); next.Height != 2 {
		t.Fatalf("template after stale submit height = %d want 2", next.Height)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("template after stale submit took too long: %s", elapsed)
	}
	if rpcBlocks(t, paths)[len(rpcBlocks(t, paths))-1].Height != 1 {
		t.Fatalf("stale submit changed height")
	}
}

func TestMinerSubmitDuplicateAndConcurrentRace(t *testing.T) {
	paths, server := newMinerRPCServer(t)
	miner := newRPCWallet(t)
	tpl := fetchMinerTemplate(t, server.URL, miner.Address)
	mined := chain.Mine(tpl.Block)
	var wg sync.WaitGroup
	results := make(chan minerSubmitTestResponse, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- submitMinerBlockWithTimeout(t, server.URL, tpl.TemplateID, mined, 2*time.Second)
		}()
	}
	wg.Wait()
	close(results)
	accepted, rejected := 0, 0
	for result := range results {
		if result.Accepted {
			accepted++
			continue
		}
		if result.Duplicate || result.Reason == "stale template" {
			rejected++
		}
	}
	if accepted != 1 || rejected != 1 {
		t.Fatalf("concurrent submit accepted=%d rejected=%d", accepted, rejected)
	}
	duplicate := submitMinerBlockWithTimeout(t, server.URL, tpl.TemplateID, mined, 2*time.Second)
	if duplicate.Accepted || !duplicate.Duplicate || duplicate.Reason != "duplicate block" {
		t.Fatalf("expected duplicate block response, got %#v", duplicate)
	}
	if _, err := chain.ValidateChain(rpcBlocks(t, paths)); err != nil {
		t.Fatal(err)
	}
}

func TestMiningObservationEndpointsPublicReadOnly(t *testing.T) {
	paths, server := newHardeningRPCServer(t, NodeInfo{PublicRPC: true})
	for _, path := range []string{"/mining/status", "/mining/stats", "/mining/difficulty", "/mining/blocks?limit=5"} {
		info := getRPCMap(t, server.URL+path, http.StatusOK)
		if info["network"] == "" || info["height"] == nil || info["next_difficulty"] == nil {
			t.Fatalf("unexpected mining response for %s: %#v", path, info)
		}
	}
	miner := newRPCWallet(t)
	resp, err := http.Get(server.URL + "/miner/template?address=" + miner.Address)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("public miner write/template status = %d want 403", resp.StatusCode)
	}
	_ = paths
}

func TestMinerTemplateIncludesMempoolTxAndSubmitClearsMempool(t *testing.T) {
	paths, server := newMinerRPCServer(t)
	miner := newRPCWallet(t)
	receiver := newRPCWallet(t)
	fundMatureRPCMiner(t, server.URL, miner.Address, int(config.Localnet().Consensus.CoinbaseMaturity)+1)
	tx := signedRPCTx(t, paths, miner, receiver.Address, 10*config.UnitsPerCoin)
	if err := mempool.New(paths.Mempool).Add(tx); err != nil {
		t.Fatal(err)
	}
	tpl := fetchMinerTemplate(t, server.URL, miner.Address)
	if tpl.TxCount != 2 {
		t.Fatalf("template tx count = %d want 2", tpl.TxCount)
	}
	mined := chain.Mine(tpl.Block)
	submit := submitMinerBlock(t, server.URL, tpl.TemplateID, mined)
	if !submit.Accepted {
		t.Fatalf("submit rejected: %#v", submit)
	}
	pending, err := mempool.New(paths.Mempool).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("mempool not cleared: %#v", pending)
	}
	blocks := rpcBlocks(t, paths)
	details, err := ledger.BalanceDetailsFor(receiver.Address, blocks, nil, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	if details.Mature != 10*config.UnitsPerCoin {
		t.Fatalf("receiver balance = %s", amount.Format(details.Mature))
	}
}

func newMinerRPCServer(t *testing.T) (config.Paths, *httptest.Server) {
	return newMinerRPCServerWithProfile(t, config.Localnet())
}

func newMinerRPCServerWithProfile(t *testing.T, profile config.NetworkConfig) (config.Paths, *httptest.Server) {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	bc := chain.New(store)
	if err := bc.InitWithProfile(profile); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{RPCListen: ":0", P2PListen: ":0", Profile: profile})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return paths, server
}

func fetchMinerTemplate(t *testing.T, base, address string) minerTemplateTestResponse {
	t.Helper()
	return fetchMinerTemplateWithTimeout(t, base, address, 0)
}

func fetchMinerTemplateWithTimeout(t *testing.T, base, address string, timeout time.Duration) minerTemplateTestResponse {
	t.Helper()
	client := http.DefaultClient
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Get(base + "/miner/template?address=" + address)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("template status = %d", resp.StatusCode)
	}
	var tpl minerTemplateTestResponse
	if err := json.NewDecoder(resp.Body).Decode(&tpl); err != nil {
		t.Fatal(err)
	}
	return tpl
}

func submitMinerBlock(t *testing.T, base, templateID string, block types.Block) minerSubmitTestResponse {
	t.Helper()
	return submitMinerBlockWithTimeout(t, base, templateID, block, 0)
}

func submitMinerBlockWithTimeout(t *testing.T, base, templateID string, block types.Block, timeout time.Duration) minerSubmitTestResponse {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"template_id": templateID, "block": block})
	client := http.DefaultClient
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}
	resp, err := client.Post(base+"/miner/submit", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("submit status = %d", resp.StatusCode)
	}
	var submit minerSubmitTestResponse
	if err := json.NewDecoder(resp.Body).Decode(&submit); err != nil {
		t.Fatal(err)
	}
	return submit
}

func fundMatureRPCMiner(t *testing.T, base, address string, blocks int) {
	t.Helper()
	for i := 0; i < blocks; i++ {
		tpl := fetchMinerTemplate(t, base, address)
		mined := chain.Mine(tpl.Block)
		if submit := submitMinerBlock(t, base, tpl.TemplateID, mined); !submit.Accepted {
			t.Fatalf("funding submit rejected: %#v", submit)
		}
	}
}

func signedRPCTx(t *testing.T, paths config.Paths, from wallet.Wallet, to string, value uint64) types.Transaction {
	t.Helper()
	blocks := rpcBlocks(t, paths)
	l, err := ledger.ReplayMature(blocks, config.Localnet().Consensus)
	if err != nil {
		t.Fatal(err)
	}
	tx := types.NewUnsignedTransaction(from.Address, to, value, 0, l.Nonce(from.Address)+1)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	return tx
}

func rpcBlocks(t *testing.T, paths config.Paths) []types.Block {
	t.Helper()
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	blocks, err := chain.New(store).Blocks()
	if err != nil {
		t.Fatal(err)
	}
	return blocks
}

func newRPCWallet(t *testing.T) wallet.Wallet {
	t.Helper()
	w, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	return w
}
