package p2p

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"indochain/internal/amount"
	"indochain/internal/chain"
	"indochain/internal/config"
	"indochain/internal/ledger"
	"indochain/internal/mempool"
	"indochain/internal/nodestate"
	"indochain/internal/storage"
	"indochain/internal/types"
)

type Server struct {
	paths        config.Paths
	p2pListen    string
	p2pAdvertise string
	state        *nodestate.Store
	profile      config.NetworkConfig
	mutationMu   *sync.Mutex
}

const (
	maxJSONBody  = 1 << 20
	maxBlockBody = 8 << 20
)

func NewServer(paths config.Paths) Server {
	return NewServerWithProfile(paths, config.Localnet())
}

func NewServerWithProfile(paths config.Paths, profile config.NetworkConfig) Server {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	return Server{paths: paths, profile: profile}
}

func NewServerWithAdvertise(paths config.Paths, listen, advertise string) Server {
	return NewServerWithAdvertiseAndProfile(paths, listen, advertise, config.Localnet())
}

func NewServerWithAdvertiseAndProfile(paths config.Paths, listen, advertise string, profile config.NetworkConfig) Server {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	return Server{paths: paths, p2pListen: listen, p2pAdvertise: advertise, profile: profile}
}

// SetMutationMutex shares the node-level canonical-chain mutation lock with RPC and background sync.
func (s *Server) SetMutationMutex(mu *sync.Mutex) {
	s.mutationMu = mu
}

func NewServerWithState(paths config.Paths, listen, advertise string, state *nodestate.Store) Server {
	return NewServerWithStateAndProfile(paths, listen, advertise, state, config.Localnet())
}

func NewServerWithStateAndProfile(paths config.Paths, listen, advertise string, state *nodestate.Store, profile config.NetworkConfig) Server {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	return Server{paths: paths, p2pListen: listen, p2pAdvertise: advertise, state: state, profile: profile}
}

func ListenAndServe(addr string, paths config.Paths, advertise ...string) error {
	return ListenAndServeWithState(addr, paths, nil, advertise...)
}

func ListenAndServeWithState(addr string, paths config.Paths, state *nodestate.Store, advertise ...string) error {
	return NewHTTPServer(addr, paths, state, advertise...).ListenAndServe()
}

func NewHTTPServer(addr string, paths config.Paths, state *nodestate.Store, advertise ...string) *http.Server {
	return NewHTTPServerWithProfile(addr, paths, state, config.Localnet(), advertise...)
}

func NewHTTPServerWithProfile(addr string, paths config.Paths, state *nodestate.Store, profile config.NetworkConfig, advertise ...string) *http.Server {
	return NewHTTPServerWithProfileAndMutationMutex(addr, paths, state, profile, nil, advertise...)
}

func NewHTTPServerWithProfileAndMutationMutex(addr string, paths config.Paths, state *nodestate.Store, profile config.NetworkConfig, mutationMu *sync.Mutex, advertise ...string) *http.Server {
	mux := http.NewServeMux()
	server := NewServerWithStateAndProfile(paths, addr, firstString(advertise), state, profile)
	server.SetMutationMutex(mutationMu)
	server.Register(mux)
	log.Printf("p2p listening on %s advertise=%s", addr, server.p2pAdvertise)
	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func (s Server) lockMutation() func() {
	if s.mutationMu == nil {
		return func() {}
	}
	s.mutationMu.Lock()
	return s.mutationMu.Unlock
}

func (s Server) network() config.NetworkConfig {
	if s.profile.Name == "" {
		return config.Localnet()
	}
	return s.profile
}

func (s Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /p2p/health", s.health)
	mux.HandleFunc("GET /p2p/handshake", s.handshake)
	mux.Handle("GET /p2p/peers", s.authenticated(http.HandlerFunc(s.peers)))
	mux.Handle("GET /p2p/status", s.authenticated(http.HandlerFunc(s.status)))
	mux.Handle("GET /p2p/tip", s.authenticated(http.HandlerFunc(s.tip)))
	mux.Handle("GET /p2p/block/", s.authenticated(http.HandlerFunc(s.block)))
	mux.Handle("GET /p2p/blocks", s.authenticated(http.HandlerFunc(s.blocks)))
	mux.Handle("GET /p2p/headers", s.authenticated(http.HandlerFunc(s.headers)))
	mux.Handle("GET /p2p/locator", s.authenticated(http.HandlerFunc(s.locator)))
	mux.Handle("POST /p2p/common-ancestor", s.authenticated(http.HandlerFunc(s.commonAncestor)))
	mux.Handle("POST /p2p/tx", s.authenticated(http.HandlerFunc(s.receiveTx)))
	mux.Handle("POST /p2p/block", s.authenticated(http.HandlerFunc(s.receiveBlock)))
	mux.Handle("POST /p2p/peer", s.authenticated(http.HandlerFunc(s.receivePeer)))
}

func (s Server) health(w http.ResponseWriter, _ *http.Request) {
	status, err := s.localStatus()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "network": status.Network, "network_id": status.NetworkID, "chain_id": status.ChainID, "genesis_hash": status.GenesisHash, "height": status.Height, "tip_hash": status.TipHash, "cumulative_work": status.CumulativeWork})
}

func (s Server) handshake(w http.ResponseWriter, r *http.Request) {
	status, err := s.localStatus()
	if err != nil {
		writeError(w, err)
		return
	}
	identity, err := LoadOrCreateNodeIdentity(s.paths.NodeID)
	if err != nil {
		writeError(w, err)
		return
	}
	net := s.network()
	net.GenesisHash = chain.GenesisHashForNetwork(net)
	challenge := strings.TrimSpace(r.URL.Query().Get("challenge"))
	if challenge != "" {
		if len(challenge) != 64 {
			writeError(w, errors.New("invalid handshake challenge"))
			return
		}
		if _, err := hex.DecodeString(challenge); err != nil {
			writeError(w, errors.New("invalid handshake challenge"))
			return
		}
	}
	handshake := Handshake{
		NetworkName:        net.NetworkName,
		NetworkID:          net.NetworkID,
		AuthChallenge:      challenge,
		ChainID:            net.ChainID,
		ProtocolVersion:    net.ProtocolVersion,
		P2PProtocolVersion: net.P2PProtocolVersion,
		MinProtocolVersion: net.MinProtocolVersion,
		GenesisHash:        net.GenesisHash,
		Height:             status.Height,
		TipHash:            status.TipHash,
		CumulativeWork:     status.CumulativeWork,
		NodeID:             identity.NodeID,
		IdentityVersion:    NodeIdentityVersion,
		NodePublicKey:      hex.EncodeToString(identity.PublicKey),
		P2PListen:          s.p2pListen,
		P2PAdvertise:       s.p2pAdvertise,
		Services:           []string{"p2p"},
		KnownPeers:         PeerViews(s.safeKnownPeers(), DefaultMaxDiscoveredPeers),
	}
	handshake.NodeSignature, err = SignHandshake(identity, handshake)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, handshake)
}

func (s Server) peers(w http.ResponseWriter, _ *http.Request) {
	net := s.network()
	writeJSON(w, http.StatusOK, PeersResponse{
		Network:        net.Name,
		NetworkID:      net.NetworkID,
		ChainID:        net.ChainID,
		GenesisHash:    chain.GenesisHashForNetwork(net),
		AdvertiseP2P:   s.p2pAdvertise,
		KnownPeers:     PeerViews(s.safeKnownPeers(), DefaultMaxDiscoveredPeers),
		KnownCount:     len(s.safeKnownPeers()),
		ActiveCount:    countActivePeers(s.safeKnownPeers()),
		SeedCount:      countSourcePeers(s.safeKnownPeers(), "seed"),
		MaxPeerEntries: DefaultMaxDiscoveredPeers,
	})
}

func (s Server) status(w http.ResponseWriter, _ *http.Request) {
	status, err := s.localStatus()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s Server) tip(w http.ResponseWriter, _ *http.Request) {
	if s.state != nil {
		snapshot := s.state.Snapshot()
		writeJSON(w, http.StatusOK, Tip{Height: snapshot.Height, Hash: snapshot.TipHash})
		return
	}
	bc, closeFn, err := s.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	defer closeFn()
	tip, err := bc.Tip()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Tip{Height: tip.Height, Hash: tip.Hash})
}

func (s Server) block(w http.ResponseWriter, r *http.Request) {
	heightText := strings.TrimPrefix(r.URL.Path, "/p2p/block/")
	height, err := strconv.ParseUint(heightText, 10, 64)
	if err != nil {
		writeError(w, err)
		return
	}
	blocks, err := s.allBlocks()
	if err != nil {
		writeError(w, err)
		return
	}
	for _, block := range blocks {
		if block.Height == height {
			writeJSON(w, http.StatusOK, block)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "block not found"})
}

func (s Server) blocks(w http.ResponseWriter, r *http.Request) {
	from, _ := strconv.ParseUint(r.URL.Query().Get("from"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	maxBlocks := s.network().NetworkLimits.MaxSyncBlocks
	if maxBlocks == 0 {
		maxBlocks = 500
	}
	if uint64(limit) > maxBlocks {
		limit = int(maxBlocks)
	}
	all, err := s.allBlocks()
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]types.Block, 0, limit)
	for _, block := range all {
		if block.Height < from {
			continue
		}
		out = append(out, block)
		if len(out) >= limit {
			break
		}
	}
	writeJSON(w, http.StatusOK, BlocksResponse{Blocks: out})
}

func (s Server) headers(w http.ResponseWriter, r *http.Request) {
	from, _ := strconv.ParseUint(r.URL.Query().Get("from"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	maxHeaders := s.network().NetworkLimits.MaxHeaderBatch
	if maxHeaders == 0 {
		maxHeaders = 500
	}
	if uint64(limit) > maxHeaders {
		limit = int(maxHeaders)
	}
	all, err := s.allBlocks()
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]BlockHeader, 0, limit)
	for _, block := range all {
		if block.Height < from {
			continue
		}
		out = append(out, HeaderFromBlock(block))
		if len(out) >= limit {
			break
		}
	}
	writeJSON(w, http.StatusOK, HeadersResponse{Headers: out})
}

func (s Server) locator(w http.ResponseWriter, _ *http.Request) {
	all, err := s.allBlocks()
	if err != nil {
		writeError(w, err)
		return
	}
	tip := all[len(all)-1]
	writeJSON(w, http.StatusOK, LocatorResponse{Height: tip.Height, TipHash: tip.Hash, Locator: chain.BuildBlockLocator(all)})
}

func (s Server) commonAncestor(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req CommonAncestorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	response, err := FindCommonAncestor(s.paths, req.Locator)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s Server) receiveTx(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var tx types.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, TxResponse{Accepted: false, Error: "request body too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, TxResponse{Accepted: false, Error: err.Error()})
		return
	}
	if err := s.acceptTx(tx); err != nil {
		log.Printf("tx received rejected id=%s error=%v", tx.ID, err)
		writeJSON(w, http.StatusBadRequest, TxResponse{Accepted: false, Error: err.Error()})
		return
	}
	s.refreshState()
	s.learnInboundPeer(r)
	log.Printf("tx received id=%s", tx.ID)
	writeJSON(w, http.StatusOK, TxResponse{Accepted: true, TxID: tx.ID})
}

func (s Server) receiveBlock(w http.ResponseWriter, r *http.Request) {
	unlock := s.lockMutation()
	defer unlock()
	r.Body = http.MaxBytesReader(w, r.Body, maxBlockBody)
	var block types.Block
	if err := json.NewDecoder(r.Body).Decode(&block); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, BlockResponse{Accepted: false, Error: "request body too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, BlockResponse{Accepted: false, Error: err.Error()})
		return
	}
	bc, closeFn, err := s.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	tip, err := bc.Tip()
	if err != nil {
		closeFn()
		writeError(w, err)
		return
	}
	if block.PreviousHash != tip.Hash {
		closeFn()
		writeJSON(w, http.StatusBadRequest, BlockResponse{Accepted: false, Error: "block does not extend local tip", LocalHeight: tip.Height, LocalTip: tip.Hash})
		return
	}
	if err := bc.AddBlockWithNetwork(block, s.network()); err != nil {
		closeFn()
		log.Printf("block received rejected height=%d error=%v", block.Height, err)
		writeJSON(w, http.StatusBadRequest, BlockResponse{Accepted: false, Error: err.Error(), LocalHeight: tip.Height, LocalTip: tip.Hash})
		return
	}
	closeFn()
	log.Printf("block received height=%d hash=%s", block.Height, block.Hash)
	if err := removeBlockTxs(s.paths.Mempool, block); err != nil {
		log.Printf("mempool cleanup failed: %v", err)
	}
	if err := RevalidateMempoolAgainstChainWithProfile(s.paths, s.network()); err != nil {
		log.Printf("mempool revalidate failed: %v", err)
	}
	s.refreshState()
	s.learnInboundPeer(r)
	writeJSON(w, http.StatusOK, BlockResponse{Accepted: true, Height: block.Height, Hash: block.Hash})
}

func (s Server) receivePeer(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var intro PeerIntroduction
	if err := json.NewDecoder(r.Body).Decode(&intro); err != nil {
		writeError(w, err)
		return
	}
	if err := s.acceptPeerIntroduction(intro); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"accepted": true})
}

func (s Server) acceptPeerIntroduction(intro PeerIntroduction) error {
	if intro.URL == "" {
		return fmt.Errorf("peer url is required")
	}
	if err := ValidatePeerURL(intro.URL); err != nil {
		return err
	}
	if isSelfPeerURL(intro.URL, s.p2pAdvertise) {
		return nil
	}
	net := s.network()
	if net.RequireAuthenticatedNode {
		if strings.TrimSpace(intro.NodeID) == "" {
			return errors.New("peer node id is required")
		}
		if strings.TrimSpace(intro.NetworkID) == "" {
			return errors.New("peer network id is required")
		}
		if intro.ChainID == 0 {
			return errors.New("peer chain id is required")
		}
	}
	if intro.NetworkID != "" && intro.NetworkID != net.NetworkID {
		return fmt.Errorf("network id mismatch: local=%s peer=%s", net.NetworkID, intro.NetworkID)
	}
	if intro.ChainID != 0 && intro.ChainID != net.ChainID {
		return fmt.Errorf("chain id mismatch: local=%d peer=%d", net.ChainID, intro.ChainID)
	}
	now := time.Now().Format(time.RFC3339)
	return NewPeerStore(s.paths.Peers).Upsert(PeerMetadata{
		URL:             intro.URL,
		NodeID:          intro.NodeID,
		NetworkID:       intro.NetworkID,
		ChainID:         intro.ChainID,
		Status:          PeerStatusActive,
		Score:           5,
		Source:          "inbound",
		LastSeenAt:      now,
		FirstSeenAt:     now,
		LastScoreReason: "peer connect",
		LastScoreAt:     now,
		Version:         intro.Version,
		Protocol:        intro.Protocol,
		Services:        intro.Services,
	})
}

func (s Server) learnInboundPeer(r *http.Request) {
	intro := PeerIntroduction{
		URL:       r.Header.Get("X-IND-P2P-URL"),
		NodeID:    r.Header.Get("X-IND-Node-ID"),
		NetworkID: r.Header.Get("X-IND-Network-ID"),
		ChainID:   s.network().ChainID,
		Version:   s.network().NetworkName,
		Protocol:  s.network().P2PProtocolVersion,
		Services:  "p2p",
	}
	if intro.URL == "" {
		return
	}
	if err := s.acceptPeerIntroduction(intro); err != nil {
		log.Printf("inbound peer learning skipped url=%s error=%v", intro.URL, err)
	}
}

func isSelfPeerURL(peerURL, selfURL string) bool {
	if selfURL == "" {
		return false
	}
	a, errA := url.Parse(peerURL)
	b, errB := url.Parse(selfURL)
	if errA != nil || errB != nil {
		return strings.TrimRight(peerURL, "/") == strings.TrimRight(selfURL, "/")
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

func (s Server) acceptTx(tx types.Transaction) error {
	if tx.Coinbase {
		return fmt.Errorf("coinbase tx is not accepted in mempool")
	}
	// Revalidate before opening the chain database. bbolt takes an exclusive
	// file lock on Windows, so holding an open chain handle while revalidation
	// opens the same database can deadlock.
	if _, err := RevalidateMempoolAgainstLedgerWithProfile(s.paths, s.network()); err != nil {
		return fmt.Errorf("mempool revalidation failed: %w", err)
	}
	bc, closeFn, err := s.openChain()
	if err != nil {
		return err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return err
	}
	for _, block := range blocks {
		for _, confirmed := range block.Transactions {
			if confirmed.ID == tx.ID {
				return fmt.Errorf("tx already confirmed")
			}
		}
	}
	mp := mempool.New(s.paths.Mempool)
	pending, err := mp.Load()
	if err != nil {
		return err
	}
	for _, existing := range pending {
		if existing.ID == tx.ID {
			return nil
		}
	}
	l, err := ledger.ReplayMatureWithProfile(blocks, s.network().Consensus, s.network())
	if err != nil {
		return err
	}
	work := l.Clone()
	for _, existing := range pending {
		_ = work.ApplyTransaction(existing)
	}
	if err := work.ValidateTransaction(tx); err != nil {
		return fmt.Errorf("sender balance insufficient or chain not synced: %w", err)
	}
	policy := mempool.AdmissionPolicy{
		Profile: s.network(),
		MaxTxs:  s.network().Consensus.MaxTxCount,
		MaxGas:  s.network().Consensus.MaxGasPerBlock,
	}
	if err := mp.Admit(tx, policy); err != nil && !errors.Is(err, mempool.ErrDuplicateTx) {
		return err
	}
	return nil
}

func (s Server) localStatus() (Status, error) {
	if s.state != nil {
		snapshot := s.state.Snapshot()
		net := s.network()
		return Status{
			Network:            net.Name,
			NetworkID:          net.NetworkID,
			ChainID:            net.ChainID,
			GenesisHash:        chain.GenesisHashForNetwork(net),
			ProtocolVersion:    net.ProtocolVersion,
			P2PProtocolVersion: net.P2PProtocolVersion,
			Height:             snapshot.Height,
			TipHash:            snapshot.TipHash,
			Difficulty:         snapshot.NextDifficulty,
			TipDifficulty:      snapshot.TipDifficulty,
			NextDifficulty:     snapshot.NextDifficulty,
			CumulativeWork:     snapshot.CumulativeWork,
			TotalSupply:        amount.Format(snapshot.TotalSupply) + " " + config.Ticker,
			MempoolCount:       snapshot.MempoolCount,
			AdvertiseP2P:       s.p2pAdvertise,
			KnownPeerCount:     len(s.safeKnownPeers()),
			ActivePeerCount:    countActivePeers(s.safeKnownPeers()),
		}, nil
	}
	bc, closeFn, err := s.openChain()
	if err != nil {
		return Status{}, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return Status{}, err
	}
	tip := blocks[len(blocks)-1]
	nextDifficulty := chain.CalculateNextDifficultyWithParams(blocks, s.network().Difficulty)
	pending, _ := mempool.New(s.paths.Mempool).Load()
	net := s.network()
	return Status{
		Network:            net.Name,
		NetworkID:          net.NetworkID,
		ChainID:            net.ChainID,
		GenesisHash:        chain.GenesisHashForNetwork(net),
		ProtocolVersion:    net.ProtocolVersion,
		P2PProtocolVersion: net.P2PProtocolVersion,
		Height:             tip.Height,
		TipHash:            tip.Hash,
		Difficulty:         nextDifficulty,
		TipDifficulty:      tip.Difficulty,
		NextDifficulty:     nextDifficulty,
		CumulativeWork:     chain.CalculateCumulativeWork(blocks),
		TotalSupply:        amount.Format(ledger.TotalSupply(blocks)) + " " + config.Ticker,
		MempoolCount:       len(pending),
		AdvertiseP2P:       s.p2pAdvertise,
		KnownPeerCount:     len(s.safeKnownPeers()),
		ActivePeerCount:    countActivePeers(s.safeKnownPeers()),
	}, nil
}

func (s Server) safeKnownPeers() []PeerMetadata {
	peers, err := NewPeerStore(s.paths.Peers).LoadMetadata()
	if err != nil {
		return nil
	}
	out := make([]PeerMetadata, 0, len(peers))
	for _, peer := range peers {
		if peer.URL == "" || isSelfPeerURL(peer.URL, s.p2pAdvertise) {
			continue
		}
		if _, err := NormalizePeerURL(peer.URL); err != nil {
			continue
		}
		out = append(out, peer)
	}
	return out
}

func countActivePeers(peers []PeerMetadata) int {
	total := 0
	for _, peer := range peers {
		if currentStatus(peer) == PeerStatusActive {
			total++
		}
	}
	return total
}

func countSourcePeers(peers []PeerMetadata, source string) int {
	total := 0
	for _, peer := range peers {
		if strings.Contains(peer.Source, source) {
			total++
		}
	}
	return total
}

func (s Server) refreshState() {
	if s.state == nil {
		return
	}
	if err := s.state.Refresh(s.paths); err != nil {
		log.Printf("runtime state refresh failed: %v", err)
	}
}

func (s Server) allBlocks() ([]types.Block, error) {
	bc, closeFn, err := s.openChain()
	if err != nil {
		return nil, err
	}
	defer closeFn()
	return bc.Blocks()
}

func (s Server) openChain() (*chain.Blockchain, func(), error) {
	store, err := storage.OpenBolt(s.paths.DB)
	if err != nil {
		return nil, nil, err
	}
	bc := chain.New(store)
	if err := bc.InitWithProfile(s.network()); err != nil {
		_ = store.Close()
		return nil, nil, err
	}
	return bc, func() { _ = store.Close() }, nil
}

func (s Server) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasAuth := hasP2PAuthHeaders(r.Header)
		if !hasAuth && !s.network().RequireAuthenticatedNode {
			next.ServeHTTP(w, r)
			return
		}
		body, err := readAuthenticatedRequestBody(r)
		if err != nil {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": err.Error()})
			return
		}
		if !hasAuth {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authenticated p2p message required"})
			return
		}
		auth, err := VerifyP2PRequest(s.network().NetworkID, s.network().ChainID, r, body, time.Now())
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		capture := newAuthenticatedResponseWriter(w)
		next.ServeHTTP(capture, r)
		responseBody := capture.body.Bytes()
		if len(responseBody) > p2pMessageAuthResponseLimit {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authenticated p2p response too large"})
			return
		}
		identity, err := LoadOrCreateNodeIdentity(s.paths.NodeID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		responseHeaders, err := SignP2PResponse(identity, s.network().NetworkID, s.network().ChainID, auth.Nonce, capture.status, responseBody, time.Now())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := capture.commit(responseHeaders, responseBody, capture.status); err != nil {
			log.Printf("authenticated p2p response write failed: %v", err)
		}
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})

}
