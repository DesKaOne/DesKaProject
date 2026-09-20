package p2p

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/storage"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestP2PHandshakeUsesActiveProfile(t *testing.T) {
	paths, server := newProfileP2PServer(t, config.Testnet())
	hs, err := CheckPeerWithProfile(paths, server.URL, config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	if hs.NetworkID != config.Testnet().NetworkID || hs.ChainID != config.Testnet().ChainID || hs.NetworkName != "testnet" {
		t.Fatalf("handshake did not use testnet profile: %+v", hs)
	}
}

func TestP2PRejectsNetworkMismatch(t *testing.T) {
	paths, server := newProfileP2PServer(t, config.Localnet())
	_, err := CheckPeerWithProfile(paths, server.URL, config.Testnet())
	if err == nil || !strings.Contains(err.Error(), "network id mismatch") {
		t.Fatalf("expected network mismatch, got %v", err)
	}
}

func TestP2PRejectsTestnetPeerFromLocalnet(t *testing.T) {
	paths, server := newProfileP2PServer(t, config.Testnet())
	_, err := CheckPeerWithProfile(paths, server.URL, config.Localnet())
	if err == nil || !strings.Contains(err.Error(), "network id mismatch") {
		t.Fatalf("expected network mismatch, got %v", err)
	}
}

func TestP2PReorgUsesActiveProfile(t *testing.T) {
	profile := config.Testnet()
	local := newProfileTestNode(t, profile)
	peer := newProfileTestNode(t, profile)
	mineProfileBlocks(t, local, profile, 1)
	mineProfileBlocks(t, peer, profile, 2)
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(peer, ":0", "", profile).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	plan, _, err := BuildReorgPlanWithProfile(local, server.URL, DefaultMaxReorgDepth, profile)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Allowed || plan.Network != profile.Name || plan.NetworkID != profile.NetworkID || plan.ChainID != profile.ChainID {
		t.Fatalf("reorg plan did not use testnet profile: %+v", plan)
	}
}

func TestP2PReorgRejectsNetworkMismatch(t *testing.T) {
	local := newProfileTestNode(t, config.Testnet())
	peer := newProfileTestNode(t, config.Localnet())
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(peer, ":0", "", config.Localnet()).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	_, _, err := BuildReorgPlanWithProfile(local, server.URL, DefaultMaxReorgDepth, config.Testnet())
	if err == nil || !strings.Contains(err.Error(), "network id mismatch") {
		t.Fatalf("expected reorg network mismatch, got %v", err)
	}
}

func TestPeerSyncTestnet(t *testing.T) {
	profile := config.Testnet()
	local := newProfileTestNode(t, profile)
	peer := newProfileTestNode(t, profile)
	mineProfileBlocks(t, peer, profile, 1)
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(peer, ":0", "", profile).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	if err := SyncFromPeerWithProfile(local, server.URL, nil, profile); err != nil {
		t.Fatal(err)
	}
	store, err := storage.OpenBolt(local.DB)
	if err != nil {
		t.Fatal(err)
	}
	bc := chain.New(store)
	blocks, err := bc.Blocks()
	_ = store.Close()
	if err != nil {
		t.Fatal(err)
	}
	if got := blocks[len(blocks)-1].Height; got != 1 {
		t.Fatalf("synced height = %d, want 1", got)
	}
	if _, err := chain.ValidateChainWithNetwork(blocks, profile); err != nil {
		t.Fatal(err)
	}
}

func TestPeerSyncTestnetHigherWorkForkReorgs(t *testing.T) {
	profile := config.Testnet()
	local := newProfileTestNode(t, profile)
	peer := newProfileTestNode(t, profile)
	mineProfileBlocks(t, local, profile, 1)
	mineProfileBlocks(t, peer, profile, 3)
	peerTip := profileBlocks(t, peer)[3]
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(peer, ":0", "", profile).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	var out bytes.Buffer
	if err := SyncFromPeerWithProfile(local, server.URL, &out, profile); err != nil {
		if strings.Contains(err.Error(), "network id mismatch") {
			t.Fatalf("same testnet fork was mislabeled as network mismatch: %v", err)
		}
		t.Fatalf("expected higher-work testnet fork to reorg, got %v", err)
	}
	blocks := profileBlocks(t, local)
	tip := blocks[len(blocks)-1]
	if tip.Height != peerTip.Height || tip.Hash != peerTip.Hash {
		t.Fatalf("sync did not reorg to peer tip: local=%#v peer=%#v", tip, peerTip)
	}
	if !strings.Contains(out.String(), "decision: reorg_apply_higher_work") {
		t.Fatalf("sync output missing reorg decision:\n%s", out.String())
	}
	if _, err := chain.ValidateChainWithNetwork(blocks, profile); err != nil {
		t.Fatal(err)
	}
}

func TestPeerSyncTestnetDeepHigherWorkForkHonorsMaxReorgDepth(t *testing.T) {
	profile := config.Testnet()
	localAllowed := newProfileTestNode(t, profile)
	localRejected := newProfileTestNode(t, profile)
	peer := newProfileTestNode(t, profile)
	mineProfileBlocks(t, localAllowed, profile, 26)
	mineProfileBlocks(t, localRejected, profile, 26)
	mineProfileBlocks(t, peer, profile, 32)
	peerTip := profileBlocks(t, peer)[32]
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(peer, ":0", "", profile).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	var out bytes.Buffer
	if err := SyncFromPeerWithProfileAndMaxDepth(localAllowed, server.URL, &out, profile, 128); err != nil {
		t.Fatalf("expected depth-26 higher-work fork to reorg with max 128, got %v", err)
	}
	allowedBlocks := profileBlocks(t, localAllowed)
	allowedTip := allowedBlocks[len(allowedBlocks)-1]
	if allowedTip.Height != peerTip.Height || allowedTip.Hash != peerTip.Hash {
		t.Fatalf("allowed reorg did not converge: local=%#v peer=%#v", allowedTip, peerTip)
	}
	if !strings.Contains(out.String(), "reorg depth: 26") || !strings.Contains(out.String(), "max reorg depth: 128") || !strings.Contains(out.String(), "decision: reorg_apply_higher_work") {
		t.Fatalf("deep reorg output missing depth/decision:\n%s", out.String())
	}
	if _, err := chain.ValidateChainWithNetwork(allowedBlocks, profile); err != nil {
		t.Fatal(err)
	}

	err := SyncFromPeerWithProfileAndMaxDepth(localRejected, server.URL, nil, profile, 10)
	if err == nil {
		t.Fatal("expected depth-26 fork to be rejected with max 10")
	}
	msg := err.Error()
	for _, want := range []string{"reorg_depth_exceeds_max", "reorg_depth: 26", "max_reorg_depth: 10", "common ancestor height: 0"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("rejection missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "network id mismatch") {
		t.Fatalf("same-network deep fork mislabeled as network mismatch: %v", err)
	}
	rejectedTip := profileBlocks(t, localRejected)[26]
	if rejectedTip.Height != 26 || rejectedTip.Hash == peerTip.Hash {
		t.Fatalf("rejected reorg changed local tip: %#v peer=%#v", rejectedTip, peerTip)
	}
}

func TestPeerSyncRejectsNetworkMismatch(t *testing.T) {
	local := newProfileTestNode(t, config.Testnet())
	peer := newProfileTestNode(t, config.Localnet())
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(peer, ":0", "", config.Localnet()).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	err := SyncFromPeerWithProfile(local, server.URL, nil, config.Testnet())
	if err == nil || !strings.Contains(err.Error(), "network id mismatch") {
		t.Fatalf("expected sync network mismatch, got %v", err)
	}
}

func TestPeerSyncRejectsNetworkMismatchEvenWhenLocalUpToDate(t *testing.T) {
	local := newProfileTestNode(t, config.Testnet())
	peer := newProfileTestNode(t, config.Localnet())
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(peer, ":0", "", config.Localnet()).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	err := SyncFromPeerWithProfile(local, server.URL, nil, config.Testnet())
	if err == nil || !strings.Contains(err.Error(), "network id mismatch") {
		t.Fatalf("expected network mismatch before up-to-date return, got %v", err)
	}
}

func TestPeerSyncRejectsGenesisMismatchEvenWhenLocalUpToDate(t *testing.T) {
	local := newProfileTestNode(t, config.Testnet())
	localGenesis := chain.GenesisBlockForNetwork(config.Testnet())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/p2p/handshake":
			writeJSON(w, http.StatusOK, Handshake{
				NetworkName:        config.Testnet().NetworkName,
				NetworkID:          config.Testnet().NetworkID,
				ChainID:            config.Testnet().ChainID,
				ProtocolVersion:    config.Testnet().ProtocolVersion,
				P2PProtocolVersion: config.Testnet().P2PProtocolVersion,
				MinProtocolVersion: config.Testnet().MinProtocolVersion,
				GenesisHash:        strings.Repeat("f", 64),
				Height:             0,
				TipHash:            localGenesis.Hash,
			})
		default:
			t.Fatalf("unexpected request before handshake rejection: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	err := SyncFromPeerWithProfile(local, server.URL, nil, config.Testnet())
	if err == nil || !strings.Contains(err.Error(), "genesis hash mismatch") {
		t.Fatalf("expected genesis mismatch before up-to-date return, got %v", err)
	}
}

func TestPeerSyncUpToDateValidPeer(t *testing.T) {
	profile := config.Testnet()
	local := newProfileTestNode(t, profile)
	peer := newProfileTestNode(t, profile)
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(peer, ":0", "", profile).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	var out strings.Builder
	if err := SyncFromPeerWithProfile(local, server.URL, &out, profile); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "local chain already up to date") {
		t.Fatalf("missing up-to-date output after valid handshake:\n%s", out.String())
	}
}

func TestNormalizePeerURLAndPeerStoreDedupesBootnodes(t *testing.T) {
	normalized, err := NormalizePeerURL(" http://127.0.0.1:9611/ ")
	if err != nil {
		t.Fatal(err)
	}
	if normalized != "http://127.0.0.1:9611" {
		t.Fatalf("normalized url = %q", normalized)
	}
	if _, err := NormalizePeerURL("http://127.0.0.1:9611/path"); err == nil {
		t.Fatal("expected path-bearing peer url to be rejected")
	}

	store := NewPeerStore(filepath.Join(t.TempDir(), "peers.json"))
	if err := store.AddWithSource("http://127.0.0.1:9611/", "bootnode"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddWithSource("http://127.0.0.1:9611", "bootnode"); err != nil {
		t.Fatal(err)
	}
	peers, err := store.LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 || peers[0].URL != "http://127.0.0.1:9611" || peers[0].Source != "bootnode" {
		t.Fatalf("unexpected bootnode peers: %+v", peers)
	}

	reloaded, err := NewPeerStore(store.path).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded) != 1 || reloaded[0].Source != "bootnode" {
		t.Fatalf("bootnode peer did not persist: %+v", reloaded)
	}
}

func TestPeerCheckStatusOfflineAndRecovery(t *testing.T) {
	profile := config.Testnet()
	paths := newProfileTestNode(t, profile)
	addr := freeTCPAddr(t)
	offlineURL := "http://" + addr
	if _, err := CheckPeerWithProfile(paths, offlineURL, profile); err == nil {
		t.Fatal("expected offline peer check to fail")
	}
	peers, err := NewPeerStore(paths.Peers).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 || peers[0].Status != PeerStatusOffline || peers[0].LastError == "" {
		t.Fatalf("offline peer metadata not recorded: %+v", peers)
	}

	server := newProfileP2PServerOnAddr(t, profile, addr)
	_, err = CheckPeerWithProfile(paths, server.URL+"/", profile)
	if err != nil {
		t.Fatal(err)
	}
	peers, err = NewPeerStore(paths.Peers).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 || peers[0].Status != PeerStatusActive || peers[0].LastError != "" || peers[0].NetworkID != profile.NetworkID || peers[0].GenesisHash == "" || peers[0].ProtocolVersion == 0 {
		t.Fatalf("recovered peer metadata not refreshed: %+v", peers)
	}
}

func TestPeerSyncAfterRestartUsesPersistedPeer(t *testing.T) {
	profile := config.Testnet()
	local := newProfileTestNode(t, profile)
	peer := newProfileTestNode(t, profile)
	mineProfileBlocks(t, peer, profile, 2)
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(peer, ":0", "", profile).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()
	if err := NewPeerStore(local.Peers).AddWithSource(server.URL+"/", "bootnode"); err != nil {
		t.Fatal(err)
	}

	restartedStore := NewPeerStore(local.Peers)
	peers, err := restartedStore.LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 || peers[0].URL != server.URL {
		t.Fatalf("persisted peer not normalized after restart: %+v", peers)
	}
	var out bytes.Buffer
	if err := SyncFromPeerWithProfile(local, peers[0].URL, &out, profile); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "imported blocks: 2") || !strings.Contains(out.String(), "tip hash:") {
		t.Fatalf("sync output missing import summary:\n%s", out.String())
	}
}

func TestPeerSyncNoCompleteOnMismatchAndLocalAheadOutput(t *testing.T) {
	local := newProfileTestNode(t, config.Testnet())
	mismatchPeer := newProfileTestNode(t, config.Localnet())
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(mismatchPeer, ":0", "", config.Localnet()).Register(mux)
	server := httptest.NewServer(mux)
	defer server.Close()
	var out bytes.Buffer
	err := SyncFromPeerWithProfile(local, server.URL, &out, config.Testnet())
	if err == nil || !strings.Contains(err.Error(), "network id mismatch") {
		t.Fatalf("expected network mismatch, got %v", err)
	}
	if strings.Contains(out.String(), "sync complete") {
		t.Fatalf("mismatch output must not claim sync complete:\n%s", out.String())
	}

	localAhead := newProfileTestNode(t, config.Testnet())
	peer := newProfileTestNode(t, config.Testnet())
	mineProfileBlocks(t, localAhead, config.Testnet(), 1)
	mux = http.NewServeMux()
	NewServerWithAdvertiseAndProfile(peer, ":0", "", config.Testnet()).Register(mux)
	server = httptest.NewServer(mux)
	defer server.Close()
	out.Reset()
	if err := SyncFromPeerWithProfile(localAhead, server.URL, &out, config.Testnet()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "local chain ahead of peer") || !strings.Contains(out.String(), "imported blocks: 0") {
		t.Fatalf("local-ahead output missing summary:\n%s", out.String())
	}
}

func TestPeerSyncInvalidBlockPenalizesPeer(t *testing.T) {
	profile := config.Testnet()
	local := newProfileTestNode(t, profile)
	peer := newProfileTestNode(t, profile)
	mineProfileBlocks(t, peer, profile, 1)
	blocks := profileBlocks(t, peer)
	genesisHash := chain.GenesisBlockForNetwork(profile).Hash
	tip := blocks[len(blocks)-1]
	badBlock := tip
	badBlock.Hash = strings.Repeat("0", 64)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/p2p/handshake":
			writeJSON(w, http.StatusOK, Handshake{
				NetworkName:        profile.NetworkName,
				NetworkID:          profile.NetworkID,
				ChainID:            profile.ChainID,
				ProtocolVersion:    profile.ProtocolVersion,
				P2PProtocolVersion: profile.P2PProtocolVersion,
				MinProtocolVersion: profile.MinProtocolVersion,
				GenesisHash:        genesisHash,
				Height:             tip.Height,
				TipHash:            tip.Hash,
			})
		case r.URL.Path == "/p2p/status":
			writeJSON(w, http.StatusOK, Status{Network: profile.NetworkName, NetworkID: profile.NetworkID, ChainID: profile.ChainID, GenesisHash: genesisHash, ProtocolVersion: profile.ProtocolVersion, Height: tip.Height, TipHash: tip.Hash})
		case r.URL.Path == "/p2p/headers":
			writeJSON(w, http.StatusOK, HeadersResponse{Headers: []BlockHeader{HeaderFromBlock(tip)}})
		case r.URL.Path == "/p2p/block/1":
			writeJSON(w, http.StatusOK, badBlock)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := SyncFromPeerWithProfile(local, server.URL, nil, profile)
	if err == nil || !strings.Contains(err.Error(), "import block 1") {
		t.Fatalf("expected invalid block import failure, got %v", err)
	}
	peers, err := NewPeerStore(local.Peers).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 || peers[0].Score >= 0 || peers[0].Status != PeerStatusBad || peers[0].LastError == "" {
		t.Fatalf("invalid block did not penalize peer: %+v", peers)
	}
}

func TestP2PPeersEndpointAndDiscovery(t *testing.T) {
	profile := config.Testnet()
	local := newProfileTestNode(t, profile)
	candidate := newProfileTestNode(t, profile)
	candidateMux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(candidate, ":0", "", profile).Register(candidateMux)
	candidateServer := httptest.NewServer(candidateMux)
	defer candidateServer.Close()

	seed := newProfileTestNode(t, profile)
	if err := NewPeerStore(seed.Peers).AddWithSource(candidateServer.URL, "seed"); err != nil {
		t.Fatal(err)
	}
	seedMux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(seed, ":0", "http://seed.example:10311", profile).Register(seedMux)
	seedServer := httptest.NewServer(seedMux)
	defer seedServer.Close()

	client, err := NewClientForProfile(local, profile, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Peers(seedServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	if response.NetworkID != profile.NetworkID || response.ChainID != profile.ChainID || response.KnownCount != 1 || len(response.KnownPeers) != 1 {
		t.Fatalf("unexpected peers response: %+v", response)
	}
	result, err := DiscoverFromPeer(local, seedServer.URL, profile, "", DefaultMaxDiscoveredPeers)
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 1 || result.Checked != 1 || result.Discovered[0] != candidateServer.URL {
		t.Fatalf("unexpected discovery result: %+v", result)
	}
	peers, err := NewPeerStore(local.Peers).LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 2 {
		t.Fatalf("expected seed check plus discovered candidate, got %+v", peers)
	}
	found := false
	for _, peer := range peers {
		if peer.URL == candidateServer.URL && strings.Contains(peer.Source, "discovered") && peer.NetworkID == profile.NetworkID {
			found = true
		}
	}
	if !found {
		t.Fatalf("discovered peer not stored with metadata: %+v", peers)
	}
}

func TestDiscoverySkipsSelfInvalidAndWrongNetwork(t *testing.T) {
	profile := config.Testnet()
	local := newProfileTestNode(t, profile)
	wrong := newProfileTestNode(t, config.Localnet())
	wrongMux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(wrong, ":0", "", config.Localnet()).Register(wrongMux)
	wrongServer := httptest.NewServer(wrongMux)
	defer wrongServer.Close()

	seed := newProfileTestNode(t, profile)
	if err := NewPeerStore(seed.Peers).AddWithSource("http://bad path", "seed"); err == nil {
		t.Fatal("expected invalid seed add to fail")
	}
	_ = NewPeerStore(seed.Peers).SaveMetadata([]PeerMetadata{
		{URL: wrongServer.URL, Source: "seed"},
		{URL: "http://127.0.0.1:10311", Source: "seed"},
	})
	seedMux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(seed, ":0", "http://seed.example:10311", profile).Register(seedMux)
	seedServer := httptest.NewServer(seedMux)
	defer seedServer.Close()

	result, err := DiscoverFromPeer(local, seedServer.URL, profile, "http://127.0.0.1:10311", DefaultMaxDiscoveredPeers)
	if err != nil {
		t.Fatal(err)
	}
	if result.Added != 0 || result.Rejected == 0 || result.Skipped == 0 {
		t.Fatalf("expected rejected wrong network and skipped self: %+v", result)
	}
}

func TestPeerStoreCorruptFileHandledAndCooldownRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peers.json")
	if err := os.WriteFile(path, []byte(`not-json`), 0644); err != nil {
		t.Fatal(err)
	}
	store := NewPeerStore(path)
	peers, err := store.LoadMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 0 {
		t.Fatalf("corrupt store should load empty, got %+v", peers)
	}
	if matches, _ := filepath.Glob(path + ".corrupt-*"); len(matches) != 1 {
		t.Fatalf("corrupt store backup not created")
	}
	peer := "http://127.0.0.1:10311"
	if err := store.AdjustPeerScore(peer, -10, "request failed"); err != nil {
		t.Fatal(err)
	}
	peers, _ = store.LoadMetadata()
	if len(peers) != 1 || peers[0].Status != PeerStatusCooldown || peers[0].CooldownUntil == "" {
		t.Fatalf("cooldown not recorded: %+v", peers)
	}
	peers[0].CooldownUntil = time.Now().Add(-time.Minute).Format(time.RFC3339)
	if err := store.SaveMetadata(peers); err != nil {
		t.Fatal(err)
	}
	peers, _ = store.LoadMetadata()
	if peers[0].Status == PeerStatusCooldown || peers[0].CooldownUntil != "" {
		t.Fatalf("cooldown did not expire: %+v", peers)
	}
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func newProfileP2PServerOnAddr(t *testing.T, profile config.NetworkConfig, addr string) *httptest.Server {
	t.Helper()
	paths := newProfileTestNode(t, profile)
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(paths, ":0", "", profile).Register(mux)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(mux)
	server.Listener = ln
	server.Start()
	t.Cleanup(server.Close)
	if server.URL != fmt.Sprintf("http://%s", addr) {
		t.Fatalf("server url = %s, want http://%s", server.URL, addr)
	}
	return server
}

func profileBlocks(t *testing.T, paths config.Paths) []types.Block {
	t.Helper()
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	bc := chain.New(store)
	blocks, err := bc.Blocks()
	_ = store.Close()
	if err != nil {
		t.Fatal(err)
	}
	return blocks
}

func newProfileP2PServer(t *testing.T, profile config.NetworkConfig) (config.Paths, *httptest.Server) {
	t.Helper()
	paths := newProfileTestNode(t, profile)
	mux := http.NewServeMux()
	NewServerWithAdvertiseAndProfile(paths, ":0", "", profile).Register(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return paths, server
}

func newProfileTestNode(t *testing.T, profile config.NetworkConfig) config.Paths {
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

func mineProfileBlocks(t *testing.T, paths config.Paths, profile config.NetworkConfig, count int) {
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
