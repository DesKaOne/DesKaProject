package p2p

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/mempool"
	"deskachain/internal/nodestate"
	"deskachain/internal/storage"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestNodeIDPersistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node_id")
	first, err := LoadOrCreateNodeID(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadOrCreateNodeID(path)
	if err != nil {
		t.Fatal(err)
	}
	if first == "" || first != second {
		t.Fatalf("node id not persistent: %q %q", first, second)
	}
}

func TestHandshakeValidation(t *testing.T) {
	local := config.Localnet()
	local.GenesisHash = "abc"
	peer := Handshake{
		NetworkID:          local.NetworkID,
		ChainID:            local.ChainID,
		ProtocolVersion:    local.ProtocolVersion,
		MinProtocolVersion: local.MinProtocolVersion,
		GenesisHash:        local.GenesisHash,
	}
	if err := ValidateHandshake(local, peer); err != nil {
		t.Fatal(err)
	}
	peer.NetworkID = "other"
	if err := ValidateHandshake(local, peer); err == nil || !strings.Contains(err.Error(), "network id mismatch") {
		t.Fatalf("expected network mismatch, got %v", err)
	}
	peer.NetworkID = local.NetworkID
	peer.GenesisHash = "other"
	if err := ValidateHandshake(local, peer); err == nil || !strings.Contains(err.Error(), "genesis hash mismatch") {
		t.Fatalf("expected genesis mismatch, got %v", err)
	}
	peer.GenesisHash = local.GenesisHash
	peer.ProtocolVersion = 0
	if err := ValidateHandshake(local, peer); err == nil || !strings.Contains(err.Error(), "incompatible protocol") {
		t.Fatalf("expected protocol mismatch, got %v", err)
	}
}

func TestP2PServerTimeoutConfig(t *testing.T) {
	server := NewHTTPServer(":0", config.NewPaths(t.TempDir()), nil)
	if server.ReadHeaderTimeout <= 0 || server.ReadTimeout <= 0 || server.WriteTimeout <= 0 || server.IdleTimeout <= 0 || server.MaxHeaderBytes <= 0 {
		t.Fatalf("server timeouts/header limit not configured: %#v", server)
	}
}

func TestTwoChainsGenesisSame(t *testing.T) {
	a := newTestNode(t)
	b := newTestNode(t)
	tipA := tip(t, a)
	tipB := tip(t, b)
	if tipA.Hash != tipB.Hash {
		t.Fatalf("genesis mismatch: %s != %s", tipA.Hash, tipB.Hash)
	}
}

func TestSyncBlocksFromPeer(t *testing.T) {
	a := newTestNode(t)
	b := newTestNode(t)
	miner := newWallet(t)
	mineBlocks(t, a, miner.Address, int(config.Localnet().Consensus.CoinbaseMaturity)+1)
	server := newP2PTestServer(a)
	defer server.Close()
	var out bytes.Buffer
	if err := SyncFromPeer(b, server.URL, &out); err != nil {
		t.Fatal(err)
	}
	tipA := tip(t, a)
	tipB := tip(t, b)
	if tipA.Height != tipB.Height || tipA.Hash != tipB.Hash {
		t.Fatalf("tips differ: A=%d %s B=%d %s", tipA.Height, tipA.Hash, tipB.Height, tipB.Hash)
	}
	validateChain(t, b)
	if !strings.Contains(out.String(), "imported block height=3") {
		t.Fatalf("sync output missing imported block:\n%s", out.String())
	}
}

func TestBroadcastTxAndBlock(t *testing.T) {
	a := newTestNode(t)
	b := newTestNode(t)
	miner := newWallet(t)
	receiver := newWallet(t)
	mineBlocks(t, a, miner.Address, int(config.Localnet().Consensus.CoinbaseMaturity)+1)
	serverB := newP2PTestServer(b)
	defer serverB.Close()
	serverA := newP2PTestServer(a)
	defer serverA.Close()
	if err := SyncFromPeer(b, serverA.URL, nil); err != nil {
		t.Fatal(err)
	}
	tx := signedTx(t, a, miner, receiver.Address, 10*config.UnitsPerCoin)
	if err := mempool.New(a.Mempool).Add(tx); err != nil {
		t.Fatal(err)
	}
	response, err := NewClient().BroadcastTx(serverB.URL, tx)
	if err != nil {
		t.Fatal(err)
	}
	if !response.Accepted {
		t.Fatalf("tx rejected: %s", response.Error)
	}
	pendingB, err := mempool.New(b.Mempool).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(pendingB) != 1 || pendingB[0].ID != tx.ID {
		t.Fatalf("node B mempool = %#v", pendingB)
	}
	pendingA, _ := mempool.New(a.Mempool).Load()
	block := mineBlock(t, a, miner.Address, pendingA)
	blockResp, err := NewClient().BroadcastBlock(serverB.URL, block)
	if err != nil {
		t.Fatal(err)
	}
	if !blockResp.Accepted {
		t.Fatalf("block rejected: %s", blockResp.Error)
	}
	pendingB, _ = mempool.New(b.Mempool).Load()
	if len(pendingB) != 0 {
		t.Fatalf("node B mempool not cleared: %#v", pendingB)
	}
	tipA := tip(t, a)
	tipB := tip(t, b)
	if tipA.Hash != tipB.Hash {
		t.Fatalf("tips differ after block broadcast")
	}
	l := ledgerFor(t, b)
	if got := l.Balance(receiver.Address); got != 10*config.UnitsPerCoin {
		t.Fatalf("receiver balance = %d", got)
	}
}

func TestPeerStoreAddDuplicateRemove(t *testing.T) {
	store := NewPeerStore(filepath.Join(t.TempDir(), "peers.json"))
	peer := "http://127.0.0.1:9331"
	if err := store.Add(peer); err != nil {
		t.Fatal(err)
	}
	if err := store.Add(peer); err != nil {
		t.Fatal(err)
	}
	peers, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 {
		t.Fatalf("duplicate peer stored: %#v", peers)
	}
	if err := store.Remove(peer); err != nil {
		t.Fatal(err)
	}
	peers, _ = store.Load()
	if len(peers) != 0 {
		t.Fatalf("peer not removed: %#v", peers)
	}
}

func TestPeerStoreStrictPerDataDir(t *testing.T) {
	node1 := config.NewPaths(t.TempDir())
	node2 := config.NewPaths(t.TempDir())
	if err := NewPeerStore(node1.Peers).AddWithSource("http://127.0.0.1:9331", "manual"); err != nil {
		t.Fatal(err)
	}
	if err := NewPeerStore(node2.Peers).AddWithSource("http://127.0.0.1:9332", "manual"); err != nil {
		t.Fatal(err)
	}
	peers1, err := NewPeerStore(node1.Peers).Load()
	if err != nil {
		t.Fatal(err)
	}
	peers2, err := NewPeerStore(node2.Peers).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers1) != 1 || peers1[0] != "http://127.0.0.1:9331" {
		t.Fatalf("node1 peers leaked or missing: %#v", peers1)
	}
	if len(peers2) != 1 || peers2[0] != "http://127.0.0.1:9332" {
		t.Fatalf("node2 peers leaked or missing: %#v", peers2)
	}
}

func TestPeerStoreMetadataAndLegacyMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "peers.json")
	if err := os.WriteFile(path, []byte(`{"peers":["http://127.0.0.1:9331"]}`), 0644); err != nil {
		t.Fatal(err)
	}
	store := NewPeerStore(path)
	meta, err := store.LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(meta) != 1 || meta[0].URL != "http://127.0.0.1:9331" {
		t.Fatalf("legacy migration failed: %#v", meta)
	}
	hs := Handshake{NodeID: "n1", NetworkID: "dkc-local-1", ChainID: 777001, Height: 3, TipHash: "abc"}
	if err := store.Upsert(MetadataFromHandshake(meta[0].URL, hs, 1)); err != nil {
		t.Fatal(err)
	}
	meta, _ = store.LoadMetadata()
	if meta[0].NodeID != "n1" || meta[0].Status != PeerStatusActive || meta[0].Score != 1 {
		t.Fatalf("metadata not stored: %#v", meta[0])
	}
}

func TestPeerScoreClampedCooldownAndBadStatus(t *testing.T) {
	store := NewPeerStore(filepath.Join(t.TempDir(), "peers.json"))
	peer := "http://127.0.0.1:9331"
	if err := store.Upsert(PeerMetadata{URL: peer, Status: PeerStatusActive, Score: 787}); err != nil {
		t.Fatal(err)
	}
	meta, err := store.LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if meta[0].Score != PeerScoreMax {
		t.Fatalf("score not clamped to max: %#v", meta[0])
	}
	if err := store.AdjustPeerScore(peer, -250, "invalid block"); err != nil {
		t.Fatal(err)
	}
	meta, _ = store.LoadMetadata()
	if meta[0].Score != PeerScoreMin || meta[0].Status != PeerStatusBad {
		t.Fatalf("score/status not clamped to bad: %#v", meta[0])
	}
	coolPeer := "http://127.0.0.1:9332"
	if err := store.Upsert(PeerMetadata{URL: coolPeer, Status: PeerStatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := store.AdjustPeerScore(coolPeer, 1, "sync up to date"); err != nil {
		t.Fatal(err)
	}
	if err := store.AdjustPeerScore(coolPeer, 1, "sync up to date"); err != nil {
		t.Fatal(err)
	}
	meta, _ = store.LoadMetadata()
	for _, peer := range meta {
		if peer.URL == coolPeer && peer.Score != 1 {
			t.Fatalf("cooldown score = %d, want 1", peer.Score)
		}
	}
}

func TestPeerReputationLatencyFailureAndSelection(t *testing.T) {
	store := NewPeerStore(filepath.Join(t.TempDir(), "peers.json"))
	fast := "http://127.0.0.1:9331"
	slow := "http://127.0.0.1:9332"
	failed := "http://127.0.0.1:9333"
	if err := store.SaveMetadata([]PeerMetadata{
		{URL: slow, Status: PeerStatusActive, Score: 5, SuccessCount: 1},
		{URL: fast, Status: PeerStatusActive, Score: 5, SuccessCount: 1},
		{URL: failed, Status: PeerStatusActive, Score: 5, SuccessCount: 1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateLatency(slow, 2500, ""); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateLatency(fast, 50, ""); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateLatency(failed, 100, "timeout"); err != nil {
		t.Fatal(err)
	}
	peers, err := store.LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	selected := SelectPeers(peers, false, 2)
	if len(selected) == 0 || selected[0].URL != fast {
		t.Fatalf("expected fast peer first, selected=%+v peers=%+v", selected, peers)
	}
	for _, peer := range peers {
		switch peer.URL {
		case fast:
			if peer.Score <= 5 || peer.LastScoreReason != "latency excellent" || peer.LastLatencyMS != 50 {
				t.Fatalf("fast peer reputation not updated: %+v", peer)
			}
		case slow:
			if peer.Score >= 5 || peer.LastScoreReason != "latency high" {
				t.Fatalf("slow peer reputation not penalized: %+v", peer)
			}
		case failed:
			if peer.Status != PeerStatusOffline || peer.FailureCount == 0 || peer.LastScoreReason != "latency failed" {
				t.Fatalf("failed peer reputation not recorded: %+v", peer)
			}
		}
	}
	reputation := ReputationViews(peers)
	if len(reputation) != 3 || reputation[0].URL != fast || reputation[0].EffectiveScore <= reputation[1].EffectiveScore {
		t.Fatalf("unexpected reputation ordering: %+v", reputation)
	}
}

func TestPeerMaintenancePrunesBackoffAndSubnetLimit(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	store := NewPeerStore(paths.Peers)
	old := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	recent := time.Now().Format(time.RFC3339)
	if err := store.SaveMetadata([]PeerMetadata{
		{URL: "http://127.0.0.1:9331", Source: "discovered", Status: PeerStatusOffline, LastSeenAt: old},
		{URL: "http://127.0.0.1:9332", Source: "seed", Status: PeerStatusOffline, LastSeenAt: old},
		{URL: "http://127.0.0.1:9333", Source: "manual", Status: PeerStatusActive, LastSeenAt: recent},
	}); err != nil {
		t.Fatal(err)
	}
	pruned, err := store.PruneExpired(time.Hour, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 1 {
		t.Fatalf("pruned = %d, want 1", pruned)
	}
	peers, err := store.LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 2 {
		t.Fatalf("unexpected peers after prune: %+v", peers)
	}
	if !peerCapacityAllows(peers, "http://127.0.0.1:9334", 10, 3) {
		t.Fatalf("existing two peers should allow known subnet only up to limit before adding")
	}
	if err := store.Upsert(PeerMetadata{URL: "http://127.0.0.1:9334", Status: PeerStatusActive, LastSeenAt: recent}); err != nil {
		t.Fatal(err)
	}
	peers, _ = store.LoadMetadata()
	if peerCapacityAllows(peers, "http://127.0.0.1:9335", 10, 3) {
		t.Fatalf("subnet limit should reject third new peer: %+v", peers)
	}
	if got := peerBackoff(3, time.Second, 10*time.Second); got != 4*time.Second {
		t.Fatalf("backoff = %s, want 4s", got)
	}
	next := time.Now().Add(time.Minute).Format(time.RFC3339)
	if peerRetryDue(PeerMetadata{NextRetryAt: next}, time.Now()) {
		t.Fatalf("retry should not be due before next_retry_at")
	}
}

func TestHandshakeAdvertiseAndPeerIntroduction(t *testing.T) {
	remote := newTestNode(t)
	local := newTestNode(t)
	advertise := "http://127.0.0.1:9331"
	mux := http.NewServeMux()
	NewServerWithAdvertise(remote, ":9331", advertise).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()
	hs, err := NewClient().Handshake(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if hs.P2PAdvertise != advertise {
		t.Fatalf("p2p advertise = %q, want %q", hs.P2PAdvertise, advertise)
	}
	if hs.NetworkID != config.Localnet().NetworkID || hs.ChainID != config.Localnet().ChainID || hs.ProtocolVersion == 0 || hs.P2PProtocolVersion == "" {
		t.Fatalf("handshake missing protocol metadata: %#v", hs)
	}
	introURL := "http://127.0.0.1:9332"
	intro := PeerIntroduction{URL: introURL, NodeID: "node-2", NetworkID: config.Localnet().NetworkID, ChainID: config.Localnet().ChainID}
	if _, err := NewClient().IntroducePeer(server.URL, intro); err != nil {
		t.Fatal(err)
	}
	if _, err := NewClient().IntroducePeer(server.URL, intro); err != nil {
		t.Fatal(err)
	}
	peers, err := NewPeerStore(remote.Peers).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 || peers[0].URL != introURL || peers[0].Status != PeerStatusActive {
		t.Fatalf("unexpected introduced peers: %#v", peers)
	}
	if err := NewServerWithAdvertise(local, ":9332", introURL).acceptPeerIntroduction(PeerIntroduction{URL: introURL, NetworkID: config.Localnet().NetworkID, ChainID: config.Localnet().ChainID}); err != nil {
		t.Fatal(err)
	}
	selfPeers, _ := NewPeerStore(local.Peers).LoadMetadata()
	if len(selfPeers) != 0 {
		t.Fatalf("self peer stored: %#v", selfPeers)
	}
}

func TestStatusEndpointUsesRuntimeStateFast(t *testing.T) {
	paths := newTestNode(t)
	miner := newWallet(t)
	mineBlocks(t, paths, miner.Address, 2)
	state, err := nodestate.New(paths)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewServerWithState(paths, ":9331", "http://127.0.0.1:9331", state).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := http.Client{Timeout: 100 * time.Millisecond}
	start := time.Now()
	resp, err := client.Get(server.URL + "/p2p/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("status endpoint too slow")
	}
	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.Height != 2 || status.TipHash == "" {
		t.Fatalf("unexpected cached status: %#v", status)
	}
	if status.NetworkID != config.Localnet().NetworkID || status.ChainID != config.Localnet().ChainID || status.ProtocolVersion == 0 || status.P2PProtocolVersion == "" {
		t.Fatalf("status missing protocol metadata: %#v", status)
	}
}

func TestStatusEndpointDifficultyFields(t *testing.T) {
	paths := newTestNode(t)
	server := newP2PTestServer(paths)
	defer server.Close()
	status := fetchStatus(t, server.URL)
	if status.Height != 0 || status.TipDifficulty != 0 || status.NextDifficulty != config.InitialDifficulty || status.Difficulty != config.InitialDifficulty {
		t.Fatalf("unexpected genesis status difficulty: %#v", status)
	}
	miner := newWallet(t)
	block := mineBlock(t, paths, miner.Address, nil)
	if block.Height != 1 || block.Difficulty != config.InitialDifficulty {
		t.Fatalf("mined block difficulty = height %d difficulty %d", block.Height, block.Difficulty)
	}
	status = fetchStatus(t, server.URL)
	if status.Height != 1 || status.TipDifficulty != config.InitialDifficulty || status.NextDifficulty != config.InitialDifficulty || status.Difficulty != config.InitialDifficulty {
		t.Fatalf("unexpected mined status difficulty: %#v", status)
	}
}

func TestStatusRespondsDuringSlowBroadcast(t *testing.T) {
	paths := newTestNode(t)
	miner := newWallet(t)
	block := mineBlock(t, paths, miner.Address, nil)
	state, err := nodestate.New(paths)
	if err != nil {
		t.Fatal(err)
	}
	statusMux := http.NewServeMux()
	NewServerWithState(paths, ":9331", "http://127.0.0.1:9331", state).Register(statusMux)
	statusServer := httptest.NewServer(statusMux)
	defer statusServer.Close()
	slowMux := http.NewServeMux()
	slowMux.HandleFunc("POST /p2p/block", func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		writeJSON(w, http.StatusOK, BlockResponse{Accepted: true})
	})
	slowPeer := httptest.NewServer(slowMux)
	defer slowPeer.Close()
	done := make(chan struct{})
	go func() {
		_ = BroadcastBlockToPeers("", []PeerMetadata{{URL: slowPeer.URL, Status: PeerStatusActive}}, block)
		close(done)
	}()
	client := http.Client{Timeout: 100 * time.Millisecond}
	resp, err := client.Get(statusServer.URL + "/p2p/status")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("slow broadcast did not finish")
	}
}

func TestHeadersEndpoint(t *testing.T) {
	a := newTestNode(t)
	miner := newWallet(t)
	mineBlocks(t, a, miner.Address, 3)
	server := newP2PTestServer(a)
	defer server.Close()
	resp, err := http.Get(server.URL + "/p2p/headers?from=1&limit=999")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body HeadersResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Headers) != 3 || body.Headers[0].Height != 1 {
		t.Fatalf("unexpected headers: %#v", body.Headers)
	}
}

func TestLocatorAndCommonAncestorEndpoints(t *testing.T) {
	a := newTestNode(t)
	miner := newWallet(t)
	mineBlocks(t, a, miner.Address, 3)
	server := newP2PTestServer(a)
	defer server.Close()
	client := NewClient()
	locator, err := client.Locator(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if locator.Height != 3 || locator.TipHash == "" || len(locator.Locator) == 0 || locator.Locator[0].Height != 3 {
		t.Fatalf("unexpected locator: %#v", locator)
	}
	ancestor, err := client.CommonAncestor(server.URL, CommonAncestorRequest{Locator: locator.Locator})
	if err != nil {
		t.Fatal(err)
	}
	if !ancestor.Found || ancestor.Height != 3 || ancestor.Hash != locator.TipHash {
		t.Fatalf("unexpected ancestor: %#v", ancestor)
	}
	ancestor, err = client.CommonAncestor(server.URL, CommonAncestorRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if ancestor.Found || ancestor.Error != "empty locator" {
		t.Fatalf("expected empty locator error: %#v", ancestor)
	}
	ancestor, err = client.CommonAncestor(server.URL, CommonAncestorRequest{Locator: []chain.BlockLocatorEntry{{Height: 0, Hash: "other"}}})
	if err != nil {
		t.Fatal(err)
	}
	if ancestor.Found || !strings.Contains(ancestor.Error, "no common ancestor") {
		t.Fatalf("expected no ancestor: %#v", ancestor)
	}
}

func TestP2PReceiveTxBodyTooLarge(t *testing.T) {
	paths := newTestNode(t)
	server := newP2PTestServer(paths)
	defer server.Close()
	body := `{"id":"` + strings.Repeat("x", 1024*1024+1) + `"}`
	resp, err := http.Post(server.URL+"/p2p/tx", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", resp.StatusCode)
	}
}

func TestCommonAncestorSameChainAndGenesisFork(t *testing.T) {
	a := newTestNode(t)
	b := newTestNode(t)
	minerA := newWallet(t)
	mineBlocks(t, a, minerA.Address, 3)
	serverA := newP2PTestServer(a)
	defer serverA.Close()
	if err := SyncFromPeer(b, serverA.URL, nil); err != nil {
		t.Fatal(err)
	}
	locator, err := LocalLocator(a)
	if err != nil {
		t.Fatal(err)
	}
	ancestor, err := FindCommonAncestor(b, locator.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if !ancestor.Found || ancestor.Height != 3 || ancestor.Hash != locator.TipHash {
		t.Fatalf("same chain ancestor = %#v locator=%#v", ancestor, locator)
	}
	forkA := newTestNode(t)
	forkB := newTestNode(t)
	mineBlocks(t, forkA, newWallet(t).Address, 1)
	mineBlocks(t, forkB, newWallet(t).Address, 1)
	forkLocator, err := LocalLocator(forkA)
	if err != nil {
		t.Fatal(err)
	}
	ancestor, err = FindCommonAncestor(forkB, forkLocator.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if !ancestor.Found || ancestor.Height != 0 {
		t.Fatalf("fork ancestor = %#v", ancestor)
	}
	ancestor, err = FindCommonAncestor(forkB, []chain.BlockLocatorEntry{{Height: 0, Hash: "not-genesis"}})
	if err != nil {
		t.Fatal(err)
	}
	if ancestor.Found || ancestor.Error != "no common ancestor found" {
		t.Fatalf("expected no match: %#v", ancestor)
	}
}

func TestForkCheckScenarios(t *testing.T) {
	t.Run("in sync", func(t *testing.T) {
		a := newTestNode(t)
		server := newP2PTestServer(a)
		defer server.Close()
		result, err := CheckFork(a, server.URL)
		if err != nil {
			t.Fatal(err)
		}
		if result.ForkDetected || !result.InSync || result.Status != "in_sync" {
			t.Fatalf("unexpected result: %#v", result)
		}
	})
	t.Run("peer ahead", func(t *testing.T) {
		local := newTestNode(t)
		peer := newTestNode(t)
		miner := newWallet(t)
		mineBlocks(t, peer, miner.Address, 2)
		server := newP2PTestServer(peer)
		defer server.Close()
		result, err := CheckFork(local, server.URL)
		if err != nil {
			t.Fatal(err)
		}
		if result.ForkDetected || result.Status != "peer_ahead" || result.PeerAheadBlocks != 2 {
			t.Fatalf("unexpected result: %#v", result)
		}
	})
	t.Run("local ahead", func(t *testing.T) {
		local := newTestNode(t)
		peer := newTestNode(t)
		miner := newWallet(t)
		mineBlocks(t, local, miner.Address, 2)
		server := newP2PTestServer(peer)
		defer server.Close()
		result, err := CheckFork(local, server.URL)
		if err != nil {
			t.Fatal(err)
		}
		if result.ForkDetected || result.Status != "local_ahead" || result.LocalAheadBlocks != 2 {
			t.Fatalf("unexpected result: %#v", result)
		}
	})
	t.Run("fork from genesis", func(t *testing.T) {
		local := newTestNode(t)
		peer := newTestNode(t)
		mineBlocks(t, local, newWallet(t).Address, 1)
		mineBlocks(t, peer, newWallet(t).Address, 1)
		server := newP2PTestServer(peer)
		defer server.Close()
		result, err := CheckFork(local, server.URL)
		if err != nil {
			t.Fatal(err)
		}
		if !result.ForkDetected || result.CommonAncestorHeight != 0 || result.ReorgSupported {
			t.Fatalf("unexpected result: %#v", result)
		}
	})
}

func TestSyncForkDetected(t *testing.T) {
	a := newTestNode(t)
	b := newTestNode(t)
	minerA := newWallet(t)
	minerB := newWallet(t)
	mineBlocks(t, a, minerA.Address, 3)
	mineBlocks(t, b, minerB.Address, 3)
	beforeTip := tip(t, b)
	beforeSupply := totalSupply(t, b)
	serverA := newP2PTestServer(a)
	defer serverA.Close()
	err := SyncFromPeer(b, serverA.URL, nil)
	if err == nil || !strings.Contains(err.Error(), "sync failed: fork detected") || !strings.Contains(err.Error(), "fork_tie_same_work") {
		t.Fatalf("expected same-work fork rejection, got %v", err)
	}
	if strings.Contains(err.Error(), "network id mismatch") {
		t.Fatalf("same-network fork was mislabeled as network mismatch: %v", err)
	}
	afterTip := tip(t, b)
	afterSupply := totalSupply(t, b)
	if afterTip.Height != beforeTip.Height || afterTip.Hash != beforeTip.Hash || afterSupply != beforeSupply {
		t.Fatalf("sync fork changed local chain: before=%#v/%d after=%#v/%d", beforeTip, beforeSupply, afterTip, afterSupply)
	}
}

func TestSyncSameHeightSameTipUpToDate(t *testing.T) {
	a := newTestNode(t)
	b := newTestNode(t)
	miner := newWallet(t)
	mineBlocks(t, a, miner.Address, 3)
	serverA := newP2PTestServer(a)
	defer serverA.Close()
	if err := SyncFromPeer(b, serverA.URL, nil); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := SyncFromPeer(b, serverA.URL, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "local chain already up to date") {
		t.Fatalf("expected up to date output, got:\n%s", out.String())
	}
}

func TestSyncLowerPeerMatchingAncestorIsLocalAhead(t *testing.T) {
	local := newTestNode(t)
	peer := newTestNode(t)
	miner := newWallet(t)
	mineBlocks(t, peer, miner.Address, 2)
	serverPeer := newP2PTestServer(peer)
	defer serverPeer.Close()
	if err := SyncFromPeer(local, serverPeer.URL, nil); err != nil {
		t.Fatal(err)
	}
	mineBlocks(t, local, miner.Address, 1)
	before := tip(t, local)
	var out bytes.Buffer
	if err := SyncFromPeer(local, serverPeer.URL, &out); err != nil {
		t.Fatal(err)
	}
	after := tip(t, local)
	if after.Hash != before.Hash || !strings.Contains(out.String(), "local chain ahead of peer") {
		t.Fatalf("expected local ahead unchanged, before=%#v after=%#v out=%s", before, after, out.String())
	}
}

func TestSyncLowerPeerDifferentTipForkDetected(t *testing.T) {
	local := newTestNode(t)
	peer := newTestNode(t)
	mineBlocks(t, local, newWallet(t).Address, 3)
	mineBlocks(t, peer, newWallet(t).Address, 2)
	serverPeer := newP2PTestServer(peer)
	defer serverPeer.Close()
	err := SyncFromPeer(local, serverPeer.URL, nil)
	if err == nil || !strings.Contains(err.Error(), "sync failed: fork detected") || !strings.Contains(err.Error(), "local_ahead_more_work") {
		t.Fatalf("expected lower-work fork rejection, got %v", err)
	}
	if strings.Contains(err.Error(), "network id mismatch") {
		t.Fatalf("same-network lower-work fork was mislabeled as network mismatch: %v", err)
	}
}

func TestSyncHigherPeerForkReorgsToMoreWork(t *testing.T) {
	local := newTestNode(t)
	peer := newTestNode(t)
	mineBlocks(t, local, newWallet(t).Address, 1)
	mineBlocks(t, peer, newWallet(t).Address, 3)
	peerTip := tip(t, peer)
	serverPeer := newP2PTestServer(peer)
	defer serverPeer.Close()
	var out bytes.Buffer
	if err := SyncFromPeer(local, serverPeer.URL, &out); err != nil {
		t.Fatalf("expected higher-work fork to reorg, got %v", err)
	}
	after := tip(t, local)
	if after.Hash != peerTip.Hash || after.Height != peerTip.Height {
		t.Fatalf("sync did not reorg to peer tip: local=%#v peer=%#v", after, peerTip)
	}
	if !strings.Contains(out.String(), "reorg applied: true") || !strings.Contains(out.String(), "decision: reorg_apply_higher_work") {
		t.Fatalf("reorg output missing decision:\n%s", out.String())
	}
	validateChain(t, local)
}

func TestSyncAgainDoesNotDuplicate(t *testing.T) {
	a := newTestNode(t)
	b := newTestNode(t)
	miner := newWallet(t)
	mineBlocks(t, a, miner.Address, 2)
	server := newP2PTestServer(a)
	defer server.Close()
	if err := SyncFromPeer(b, server.URL, nil); err != nil {
		t.Fatal(err)
	}
	first := tip(t, b)
	if err := SyncFromPeer(b, server.URL, nil); err != nil {
		t.Fatal(err)
	}
	second := tip(t, b)
	if first.Height != second.Height || first.Hash != second.Hash {
		t.Fatalf("sync changed tip: %#v %#v", first, second)
	}
}

func TestUpstreamBackfillPushesMissingBlocks(t *testing.T) {
	local := newTestNode(t)
	upstream := newTestNode(t)
	miner := newWallet(t)
	mineBlocks(t, local, miner.Address, 3)
	server := newP2PTestServer(upstream)
	defer server.Close()

	result, err := BackfillToPeer(local, server.URL, config.Localnet(), config.DefaultMaxReorgDepth(config.Localnet()))
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Decision != "backfill_push" || result.Accepted != 3 {
		t.Fatalf("unexpected upstream backfill result: %#v", result)
	}
	if got, want := tip(t, upstream), tip(t, local); got.Hash != want.Hash || got.Height != want.Height {
		t.Fatalf("upstream tip = %#v want %#v", got, want)
	}
	validateChain(t, upstream)
}

func TestUpstreamBackfillAlreadyUpToDate(t *testing.T) {
	local := newTestNode(t)
	upstream := newTestNode(t)
	miner := newWallet(t)
	mineBlocks(t, local, miner.Address, 2)
	localServer := newP2PTestServer(local)
	defer localServer.Close()
	if err := SyncFromPeer(upstream, localServer.URL, nil); err != nil {
		t.Fatal(err)
	}
	upstreamServer := newP2PTestServer(upstream)
	defer upstreamServer.Close()

	result, err := BackfillToPeer(local, upstreamServer.URL, config.Localnet(), config.DefaultMaxReorgDepth(config.Localnet()))
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Decision != "already_up_to_date" || result.Accepted != 0 {
		t.Fatalf("unexpected upstream result: %#v", result)
	}
}

func TestUpstreamHigherWorkSyncsLocal(t *testing.T) {
	local := newTestNode(t)
	upstream := newTestNode(t)
	miner := newWallet(t)
	mineBlocks(t, local, miner.Address, 1)
	mineBlocks(t, upstream, miner.Address, 3)
	server := newP2PTestServer(upstream)
	defer server.Close()

	result, err := BackfillToPeer(local, server.URL, config.Localnet(), config.DefaultMaxReorgDepth(config.Localnet()))
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Decision != "local_sync_from_upstream" {
		t.Fatalf("unexpected upstream result: %#v", result)
	}
	if got, want := tip(t, local), tip(t, upstream); got.Hash != want.Hash || got.Height != want.Height {
		t.Fatalf("local tip = %#v want %#v", got, want)
	}
}

func TestUpstreamForkSameWorkDoesNotForce(t *testing.T) {
	local := newTestNode(t)
	upstream := newTestNode(t)
	mineBlocks(t, local, newWallet(t).Address, 1)
	mineBlocks(t, upstream, newWallet(t).Address, 1)
	server := newP2PTestServer(upstream)
	defer server.Close()

	result, err := BackfillToPeer(local, server.URL, config.Localnet(), config.DefaultMaxReorgDepth(config.Localnet()))
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != "no_force" {
		t.Fatalf("unexpected upstream result: %#v", result)
	}
}

func newTestNode(t *testing.T) config.Paths {
	t.Helper()
	paths := config.NewPaths(t.TempDir())
	bc, closeFn := openTestChain(t, paths)
	defer closeFn()
	if err := bc.Init(); err != nil {
		t.Fatal(err)
	}
	return paths
}

func newP2PTestServer(paths config.Paths) *httptest.Server {
	mux := http.NewServeMux()
	NewServer(paths).Register(mux)
	return httptest.NewServer(mux)
}

func fetchStatus(t *testing.T, baseURL string) Status {
	t.Helper()
	resp, err := http.Get(baseURL + "/p2p/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	return status
}

func openTestChain(t *testing.T, paths config.Paths) (*chain.Blockchain, func()) {
	t.Helper()
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	return chain.New(store), func() { _ = store.Close() }
}

func tip(t *testing.T, paths config.Paths) types.Block {
	t.Helper()
	bc, closeFn := openTestChain(t, paths)
	defer closeFn()
	tip, err := bc.Tip()
	if err != nil {
		t.Fatal(err)
	}
	return tip
}

func mineBlocks(t *testing.T, paths config.Paths, miner string, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		_ = mineBlock(t, paths, miner, nil)
	}
}

func mineBlock(t *testing.T, paths config.Paths, miner string, pending []types.Transaction) types.Block {
	t.Helper()
	bc, closeFn := openTestChain(t, paths)
	defer closeFn()
	if err := bc.Init(); err != nil {
		t.Fatal(err)
	}
	block, err := bc.MineBlock(miner, pending)
	if err != nil {
		t.Fatal(err)
	}
	if err := bc.AddBlock(block); err != nil {
		t.Fatal(err)
	}
	ids := make(map[string]struct{})
	for _, tx := range block.Transactions {
		if !tx.Coinbase {
			ids[tx.ID] = struct{}{}
		}
	}
	if err := mempool.New(paths.Mempool).RemoveIDs(ids); err != nil {
		t.Fatal(err)
	}
	return block
}

func newWallet(t *testing.T) wallet.Wallet {
	t.Helper()
	w, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func signedTx(t *testing.T, paths config.Paths, from wallet.Wallet, to string, value uint64) types.Transaction {
	t.Helper()
	l := ledgerFor(t, paths)
	pending, _ := mempool.New(paths.Mempool).Load()
	pendingCount, _, _ := mempool.PendingOutgoing(pending, from.Address)
	tx := types.NewUnsignedTransaction(from.Address, to, value, 0, l.Nonce(from.Address)+pendingCount+1)
	if err := from.SignTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	return tx
}

func ledgerFor(t *testing.T, paths config.Paths) *ledger.Ledger {
	t.Helper()
	bc, closeFn := openTestChain(t, paths)
	defer closeFn()
	l, err := bc.Ledger()
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func totalSupply(t *testing.T, paths config.Paths) uint64 {
	t.Helper()
	bc, closeFn := openTestChain(t, paths)
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		t.Fatal(err)
	}
	return ledger.TotalSupply(blocks)
}

func validateChain(t *testing.T, paths config.Paths) {
	t.Helper()
	bc, closeFn := openTestChain(t, paths)
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chain.ValidateChain(blocks); err != nil {
		t.Fatal(err)
	}
}

func TestReorgPreviewSameWorkRejected(t *testing.T) {
	local := newTestNode(t)
	peer := newTestNode(t)
	mineBlocks(t, local, newWallet(t).Address, 1)
	mineBlocks(t, peer, newWallet(t).Address, 1)
	server := newP2PTestServer(peer)
	defer server.Close()
	plan, _, err := BuildReorgPlan(local, server.URL, DefaultMaxReorgDepth)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Allowed || plan.Decision != "fork_tie_same_work" || plan.CommonAncestorHeight != 0 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestReorgPreviewPeerMoreWorkAllowed(t *testing.T) {
	local := newTestNode(t)
	peer := newTestNode(t)
	mineBlocks(t, local, newWallet(t).Address, 1)
	mineBlocks(t, peer, newWallet(t).Address, 2)
	server := newP2PTestServer(peer)
	defer server.Close()
	plan, _, err := BuildReorgPlan(local, server.URL, DefaultMaxReorgDepth)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Allowed || !plan.PeerHasMoreWork || plan.ReorgDepth != 1 || len(plan.ConnectBlocks) != 2 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestReorgApplyRequiresYes(t *testing.T) {
	local := newTestNode(t)
	peer := newTestNode(t)
	mineBlocks(t, local, newWallet(t).Address, 1)
	mineBlocks(t, peer, newWallet(t).Address, 2)
	server := newP2PTestServer(peer)
	defer server.Close()
	_, err := ApplyReorg(local, server.URL, DefaultMaxReorgDepth, false)
	if err == nil || !strings.Contains(err.Error(), "without --yes") {
		t.Fatalf("expected --yes error, got %v", err)
	}
}

func TestReorgApplyPeerMoreWork(t *testing.T) {
	local := newTestNode(t)
	peer := newTestNode(t)
	mineBlocks(t, local, newWallet(t).Address, 1)
	mineBlocks(t, peer, newWallet(t).Address, 2)
	server := newP2PTestServer(peer)
	defer server.Close()
	res, err := ApplyReorg(local, server.URL, DefaultMaxReorgDepth, true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied || !res.ChainValid || tip(t, local).Hash != tip(t, peer).Hash || tip(t, local).Height != 2 {
		t.Fatalf("unexpected reorg result: %#v local tip=%#v peer tip=%#v", res, tip(t, local), tip(t, peer))
	}
}

func TestReorgDepthLimit(t *testing.T) {
	local := newTestNode(t)
	peer := newTestNode(t)
	mineBlocks(t, local, newWallet(t).Address, 2)
	mineBlocks(t, peer, newWallet(t).Address, 3)
	server := newP2PTestServer(peer)
	defer server.Close()
	plan, _, err := BuildReorgPlan(local, server.URL, 1)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Allowed || plan.Reason != "reorg depth exceeds max depth" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestReorgRequeuesValidOrphanTx(t *testing.T) {
	local, peer, miner, _, user1, _ := reorgTxFixture(t)
	tx := signedTx(t, local, miner, user1.Address, 10)
	mineBlock(t, local, miner.Address, []types.Transaction{tx})
	mineBlocks(t, peer, newWallet(t).Address, 3)
	server := newP2PTestServer(peer)
	defer server.Close()
	res, err := ApplyReorg(local, server.URL, DefaultMaxReorgDepth, true)
	if err != nil {
		t.Fatal(err)
	}
	pending, _ := mempool.New(local.Mempool).Load()
	if res.RequeuedTransactions != 1 || res.MempoolCount != 1 || len(pending) != 1 || pending[0].ID != tx.ID || !res.ChainValid {
		t.Fatalf("unexpected result %#v pending=%#v", res, pending)
	}
	if ledgerFor(t, local).Balance(user1.Address) != 0 {
		t.Fatalf("orphan recipient balance carried into canonical ledger")
	}
	if totalSupply(t, local) != config.InitialBlockReward*16 {
		t.Fatalf("unexpected supply %d", totalSupply(t, local))
	}
	validateChain(t, local)
}

func TestReorgDoesNotRequeueTxConfirmedOnPeer(t *testing.T) {
	local, peer, miner, peerMiner, user1, _ := reorgTxFixture(t)
	tx := signedTx(t, local, miner, user1.Address, 10)
	mineBlock(t, local, miner.Address, []types.Transaction{tx})
	mineBlock(t, peer, peerMiner.Address, []types.Transaction{tx})
	mineBlocks(t, peer, peerMiner.Address, 2)
	server := newP2PTestServer(peer)
	defer server.Close()
	res, err := ApplyReorg(local, server.URL, DefaultMaxReorgDepth, true)
	if err != nil {
		t.Fatal(err)
	}
	pending, _ := mempool.New(local.Mempool).Load()
	if res.RequeuedTransactions != 0 || res.DroppedConfirmedTransactions != 1 || len(pending) != 0 || !res.ChainValid {
		t.Fatalf("unexpected result %#v pending=%#v", res, pending)
	}
	if ledgerFor(t, local).Balance(user1.Address) != 10 {
		t.Fatalf("recipient not credited on peer branch")
	}
	validateChain(t, local)
}

func TestReorgDropsInvalidOrphanTxDueInsufficientBalance(t *testing.T) {
	local, peer, miner, peerMiner, user1, user2 := reorgTxFixture(t)
	txA := signedTx(t, local, miner, user1.Address, 100)
	mineBlock(t, local, miner.Address, []types.Transaction{txA})
	txB := signedTx(t, peer, miner, user2.Address, 120)
	mineBlock(t, peer, peerMiner.Address, []types.Transaction{txB})
	mineBlocks(t, peer, peerMiner.Address, 2)
	server := newP2PTestServer(peer)
	defer server.Close()
	res, err := ApplyReorg(local, server.URL, DefaultMaxReorgDepth, true)
	if err != nil {
		t.Fatal(err)
	}
	pending, _ := mempool.New(local.Mempool).Load()
	l := ledgerFor(t, local)
	if res.RequeuedTransactions != 0 || res.DroppedInvalidTransactions != 1 || len(pending) != 0 || l.Balance(user1.Address) != 0 || l.Balance(user2.Address) != 120 || !res.ChainValid {
		t.Fatalf("unexpected result %#v pending=%#v balances user1=%d user2=%d", res, pending, l.Balance(user1.Address), l.Balance(user2.Address))
	}
	validateChain(t, local)
}

func TestReorgRemovesMempoolTxConfirmedByNewBranchAndDeduplicates(t *testing.T) {
	local, peer, miner, peerMiner, user1, _ := reorgTxFixture(t)
	tx := signedTx(t, local, miner, user1.Address, 10)
	if err := mempool.New(local.Mempool).Save([]types.Transaction{tx, tx}); err != nil {
		t.Fatal(err)
	}
	mineBlock(t, peer, peerMiner.Address, []types.Transaction{tx})
	mineBlocks(t, peer, peerMiner.Address, 2)
	server := newP2PTestServer(peer)
	defer server.Close()
	res, err := ApplyReorg(local, server.URL, DefaultMaxReorgDepth, true)
	if err != nil {
		t.Fatal(err)
	}
	pending, _ := mempool.New(local.Mempool).Load()
	if len(pending) != 0 || res.DroppedConfirmedTransactions == 0 || !res.ChainValid {
		t.Fatalf("unexpected result %#v pending=%#v", res, pending)
	}
}

func TestReorgCoinbaseNeverRequeued(t *testing.T) {
	local, peer, _, peerMiner, _, _ := reorgTxFixture(t)
	mineBlocks(t, local, newWallet(t).Address, 1)
	mineBlocks(t, peer, peerMiner.Address, 3)
	server := newP2PTestServer(peer)
	defer server.Close()
	res, err := ApplyReorg(local, server.URL, DefaultMaxReorgDepth, true)
	if err != nil {
		t.Fatal(err)
	}
	pending, _ := mempool.New(local.Mempool).Load()
	if res.RequeuedTransactions != 0 || len(pending) != 0 || totalSupply(t, local) != config.InitialBlockReward*16 || !res.ChainValid {
		t.Fatalf("unexpected result %#v pending=%#v supply=%d", res, pending, totalSupply(t, local))
	}
}

func reorgTxFixture(t *testing.T) (config.Paths, config.Paths, wallet.Wallet, wallet.Wallet, wallet.Wallet, wallet.Wallet) {
	t.Helper()
	local := newTestNode(t)
	miner := newWallet(t)
	peerMiner := newWallet(t)
	user1 := newWallet(t)
	user2 := newWallet(t)
	mineBlocks(t, local, miner.Address, int(config.Localnet().Consensus.CoinbaseMaturity)+3)
	peer := config.NewPaths(t.TempDir())
	copyTestDir(t, local.DataDir, peer.DataDir)
	return local, peer, miner, peerMiner, user1, user2
}

func copyTestDir(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.RemoveAll(dst); err != nil {
		t.Fatal(err)
	}
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	}); err != nil {
		t.Fatal(err)
	}
}
