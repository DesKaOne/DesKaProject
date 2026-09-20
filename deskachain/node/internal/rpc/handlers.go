package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"deskachain/internal/amount"
	"deskachain/internal/arith"
	"deskachain/internal/asset"
	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/crypto"
	"deskachain/internal/faucet"
	"deskachain/internal/fees"
	"deskachain/internal/ledger"
	"deskachain/internal/mempool"
	"deskachain/internal/mining"
	"deskachain/internal/p2p"
	"deskachain/internal/servicenode"
	"deskachain/internal/staking"
	"deskachain/internal/state"
	"deskachain/internal/storage"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

type handler struct {
	paths   config.Paths
	info    NodeInfo
	limiter *rateLimiter
	txMu    *sync.Mutex
	chainMu *sync.Mutex
}

func (h handler) profile() config.NetworkConfig {
	if h.info.Profile.Name == "" {
		return config.Localnet()
	}
	return h.info.Profile
}

func (h handler) maxReorgDepth() uint64 {
	if h.info.MaxReorgDepth > 0 {
		return h.info.MaxReorgDepth
	}
	return config.DefaultMaxReorgDepth(h.profile())
}

func (h handler) minMiningPeers() int {
	if h.info.MinMiningPeers > 0 {
		return h.info.MinMiningPeers
	}
	return h.profile().MinMiningPeers
}

func (h handler) allowIsolatedMining() bool {
	if h.info.AllowIsolatedMining {
		return true
	}
	return h.profile().AllowIsolatedMining
}

func (h handler) minWritePeers() int {
	if h.info.MinWritePeers > 0 {
		return h.info.MinWritePeers
	}
	return h.profile().MinWritePeers
}

func (h handler) allowIsolatedWrites() bool {
	if h.info.AllowIsolatedWrites {
		return true
	}
	return h.profile().AllowIsolatedWrites
}

const (
	maxJSONBody                 = 1 << 20
	maxTemplateBody             = 128 << 10
	maxServiceBody              = 128 << 10
	maxBlockBody                = 8 << 20
	upstreamReachabilityTimeout = 500 * time.Millisecond
)

func RegisterHandlers(mux *http.ServeMux, paths config.Paths, info NodeInfo) {
	if info.Mining == nil {
		info.Mining = mining.NewService()
	}
	info = normalizeNodeInfo(info)
	chainMu := info.ChainMutationMu
	if chainMu == nil {
		chainMu = &sync.Mutex{}
	}
	h := handler{paths: paths, info: info, limiter: newRateLimiter(), txMu: &sync.Mutex{}, chainMu: chainMu}
	mux.HandleFunc("GET /health", h.wrap("generic", h.health))
	mux.HandleFunc("GET /ready", h.wrap("generic", h.ready))
	mux.HandleFunc("GET /explorer-ui", h.wrap("generic", h.explorerUI))
	mux.HandleFunc("GET /explorer-ui/", h.wrap("generic", h.explorerUI))
	mux.HandleFunc("GET /explorer/status", h.wrap("generic", h.explorerStatus))
	mux.HandleFunc("GET /explorer/search", h.wrap("generic", h.explorerSearch))
	mux.HandleFunc("GET /explorer/blocks", h.wrap("generic", h.explorerBlocks))
	mux.HandleFunc("GET /explorer/blocks/", h.wrap("generic", h.explorerBlockByHeight))
	mux.HandleFunc("GET /explorer/block/", h.wrap("generic", h.explorerBlockByHash))
	mux.HandleFunc("GET /explorer/tx/", h.wrap("generic", h.explorerTx))
	mux.HandleFunc("GET /explorer/address/", h.wrap("generic", h.explorerAddressRouter))
	mux.HandleFunc("GET /explorer/stakes", h.wrap("generic", h.explorerStakes))
	mux.HandleFunc("GET /explorer/services", h.wrap("generic", h.explorerServices))
	mux.HandleFunc("GET /explorer/service/", h.wrap("generic", h.explorerService))
	mux.HandleFunc("GET /network/info", h.wrap("generic", h.networkInfo))
	mux.HandleFunc("GET /node/id", h.wrap("generic", h.nodeID))
	mux.HandleFunc("GET /node/compare", h.wrap("generic", h.nodeCompare))
	mux.HandleFunc("GET /node/status", h.wrap("generic", h.nodeStatus))
	mux.HandleFunc("GET /debug/p2p", h.wrap("admin", h.debugP2P))
	mux.HandleFunc("POST /debug/p2p/ping", h.wrap("admin", h.debugP2PPing))
	mux.HandleFunc("GET /peer/health", h.wrap("generic", h.peerHealth))
	mux.HandleFunc("GET /peer/list", h.wrap("generic", h.peers))
	mux.HandleFunc("GET /peer/seeds", h.wrap("generic", h.peerSeeds))
	mux.HandleFunc("GET /peer/status", h.wrap("generic", h.peerStatusGet))
	mux.HandleFunc("GET /p2p/peers", h.wrap("generic", h.p2pPeers))
	mux.HandleFunc("GET /p2p/reputation", h.wrap("generic", h.p2pReputation))
	mux.HandleFunc("GET /p2p/discovery", h.wrap("generic", h.p2pDiscovery))
	mux.HandleFunc("GET /p2p/bootstrap", h.wrap("generic", h.p2pBootstrap))
	mux.HandleFunc("GET /p2p/known-peers", h.wrap("generic", h.p2pKnownPeers))
	mux.HandleFunc("GET /upstream/status", h.wrap("generic", h.upstreamStatus))
	mux.HandleFunc("POST /upstream/push", h.wrap("admin", h.upstreamPush))
	mux.HandleFunc("POST /upstream/push-all", h.wrap("admin", h.upstreamPushAll))
	mux.HandleFunc("GET /peers", h.wrap("generic", h.peers))
	mux.HandleFunc("POST /peers", h.wrap("admin", h.peerAdd))
	mux.HandleFunc("POST /peers/connect", h.wrap("admin", h.peerConnect))
	mux.HandleFunc("DELETE /peers", h.wrap("admin", h.peerRemove))
	mux.HandleFunc("POST /peers/clear", h.wrap("admin", h.peerClear))
	mux.HandleFunc("POST /peers/check", h.wrap("generic", h.peerCheck))
	mux.HandleFunc("POST /peers/discover", h.wrap("admin", h.peerDiscover))
	mux.HandleFunc("POST /peers/status", h.wrap("generic", h.peerStatus))
	mux.HandleFunc("POST /peers/sync", h.wrap("admin", h.peerSync))
	mux.HandleFunc("POST /reorg/preview", h.wrap("admin", h.reorgPreview))
	mux.HandleFunc("POST /reorg/apply", h.wrap("admin", h.reorgApply))
	mux.HandleFunc("GET /chain/info", h.wrap("generic", h.chainInfo))
	mux.HandleFunc("GET /chain/difficulty", h.wrap("generic", h.chainDifficulty))
	mux.HandleFunc("GET /chain/locator", h.wrap("generic", h.chainLocator))
	mux.HandleFunc("GET /mining/status", h.wrap("generic", h.miningStatus))
	mux.HandleFunc("GET /mining/stats", h.wrap("generic", h.miningStatus))
	mux.HandleFunc("GET /mining/difficulty", h.wrap("generic", h.miningDifficulty))
	mux.HandleFunc("GET /mining/blocks", h.wrap("generic", h.miningBlocks))
	mux.HandleFunc("POST /chain/common-ancestor", h.wrap("generic", h.chainCommonAncestor))
	mux.HandleFunc("GET /chain/blocks", h.wrap("generic", h.chainBlocks))
	mux.HandleFunc("GET /chain/state", h.wrap("generic", h.chainState))
	mux.HandleFunc("GET /chain/state/validate", h.wrap("admin", h.chainStateValidate))
	mux.HandleFunc("GET /chain/validate", h.wrap("generic", h.chainValidate))
	mux.HandleFunc("POST /fork/check", h.wrap("generic", h.forkCheck))
	mux.HandleFunc("POST /fork/inspect-datadir", h.wrap("admin", h.forkInspectDatadir))
	mux.HandleFunc("GET /balance/", h.wrap("generic", h.balance))
	mux.HandleFunc("GET /address/", h.wrap("generic", h.address))
	mux.HandleFunc("GET /asset/info", h.wrap("generic", h.assetInfo))
	mux.HandleFunc("GET /asset/balance", h.wrap("generic", h.assetBalance))
	mux.HandleFunc("GET /asset/balances", h.wrap("generic", h.assetBalances))
	mux.HandleFunc("GET /fee/policy", h.wrap("generic", h.feePolicy))
	mux.HandleFunc("GET /fee/pool", h.wrap("generic", h.feePool))
	mux.HandleFunc("POST /fee/estimate", h.wrap("generic", h.feeEstimate))
	mux.HandleFunc("GET /tx/", h.wrap("generic", h.tx))
	mux.HandleFunc("GET /mempool", h.wrap("generic", h.mempoolList))
	mux.HandleFunc("GET /mempool/list", h.wrap("generic", h.mempoolList))
	mux.HandleFunc("POST /mempool/clear", h.wrap("admin", h.mempoolClear))
	mux.HandleFunc("GET /faucet/info", h.wrap("generic", h.faucetInfo))
	mux.HandleFunc("POST /faucet/request", h.wrap("faucet", h.faucetRequest))
	mux.HandleFunc("GET /wallets", h.wrap("wallet", h.walletList))
	mux.HandleFunc("POST /wallet/new", h.wrap("wallet", h.walletNew))
	mux.HandleFunc("POST /send", h.wrap("wallet", h.send))
	mux.HandleFunc("POST /mine", h.wrap("miner", h.mine))
	mux.HandleFunc("GET /mine/status", h.wrap("miner", h.mineStatus))
	mux.HandleFunc("GET /miner/template", h.wrap("miner", h.minerTemplate))
	mux.HandleFunc("POST /miner/template", h.wrap("miner", h.minerTemplate))
	mux.HandleFunc("POST /miner/submit", h.wrap("miner", h.minerSubmit))
	mux.HandleFunc("POST /service/register", h.wrap("service-register", h.serviceRegister))
	mux.HandleFunc("POST /service/heartbeat", h.wrap("service-heartbeat", h.serviceHeartbeat))
	mux.HandleFunc("POST /service/challenge/create", h.wrap("service-challenge", h.serviceChallengeCreate))
	mux.HandleFunc("POST /service/challenge/submit", h.wrap("service-challenge", h.serviceChallengeSubmit))
	mux.HandleFunc("GET /service/score", h.wrap("generic", h.serviceScore))
	mux.HandleFunc("GET /service/rewards", h.wrap("generic", h.serviceRewards))
	mux.HandleFunc("GET /service/list", h.wrap("generic", h.serviceList))
	mux.HandleFunc("GET /stake/info", h.wrap("generic", h.stakeInfo))
	mux.HandleFunc("GET /stake/list", h.wrap("generic", h.stakeList))
	mux.HandleFunc("GET /stake/status", h.wrap("generic", h.stakeStatus))
	mux.HandleFunc("POST /stake/lock", h.wrap("wallet", h.stakeLock))
	mux.HandleFunc("POST /stake/unlock", h.wrap("wallet", h.stakeUnlock))
	mux.HandleFunc("GET /debug/locks", h.wrap("admin", h.debugLocks))
}

func (h handler) networkInfo(w http.ResponseWriter, _ *http.Request) {
	net := h.profile()
	net.GenesisHash = chain.GenesisHashForNetwork(net)
	writeJSON(w, http.StatusOK, net)
}

func (h handler) health(w http.ResponseWriter, _ *http.Request) {
	net := h.profile()
	height, tipHash := h.debugChainTip()
	peers, _ := p2p.NewPeerStore(h.paths.Peers).Load()
	pending, _ := mempool.New(h.paths.Mempool).Load()
	serviceNodes, _ := servicenode.NewStore(h.paths, net).LoadNodes()
	blocks, _, _ := h.chainInfoData()
	stakingSummary := staking.Summary{}
	if len(blocks) > 0 {
		if state, err := staking.Replay(blocks, net.Consensus.Staking); err == nil {
			stakingSummary = state.Summary(blocksHeight(blocks))
		}
	}
	uptime := int64(0)
	if !h.info.StartedAt.IsZero() {
		uptime = int64(time.Since(h.info.StartedAt).Seconds())
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                  true,
		"network":             net.Name,
		"network_id":          net.NetworkID,
		"chain_id":            net.ChainID,
		"genesis_hash":        chain.GenesisHashForNetwork(net),
		"height":              height,
		"tip_hash":            tipHash,
		"peers":               len(peers),
		"mempool":             len(pending),
		"public_rpc":          h.info.PublicRPC,
		"wallet_rpc":          h.info.EnableWalletRPC,
		"miner_rpc":           h.info.EnableMinerRPC,
		"admin_rpc":           h.info.EnableAdminRPC,
		"faucet_rpc":          h.info.EnableFaucetRPC,
		"service_rpc":         h.info.EnableServiceRPC,
		"service_nodes":       len(serviceNodes),
		"staking_enabled":     net.Consensus.Staking.Enabled,
		"active_stake_count":  stakingSummary.ActiveStakeCount,
		"total_active_stake":  amount.Format(stakingSummary.TotalActiveStake),
		"uptime_seconds":      uptime,
		"upstream_peer_count": len(h.info.UpstreamPeers),
	})
}

func (h handler) ready(w http.ResponseWriter, _ *http.Request) {
	if _, err := h.currentHeight(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h handler) explorerStatus(w http.ResponseWriter, _ *http.Request) {
	blocks, pending, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	net := h.profile()
	stats := chain.CalculateChainStatsWithProfile(blocks, net)
	tip := blocks[len(blocks)-1]
	peers, _ := p2p.NewPeerStore(h.paths.Peers).Load()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                        true,
		"network":                   net.Name,
		"network_id":                net.NetworkID,
		"chain_id":                  net.ChainID,
		"genesis_hash":              chain.GenesisHashForNetwork(net),
		"height":                    tip.Height,
		"tip_hash":                  tip.Hash,
		"difficulty":                tip.Difficulty,
		"next_difficulty":           chain.CalculateNextDifficultyWithParams(blocks, net.Difficulty),
		"target_block_time_seconds": net.Difficulty.TargetBlockTimeSeconds,
		"retarget_window":           net.Difficulty.RetargetWindow,
		"coinbase_maturity":         net.Consensus.CoinbaseMaturity,
		"total_supply":              amount.Format(stats.TotalSupply) + " " + config.Ticker,
		"circulating_supply":        amount.Format(stats.CirculatingSupply) + " " + config.Ticker,
		"pending_tx_count":          len(pending),
		"peer_count":                len(peers),
		"public_rpc":                h.info.PublicRPC,
		"wallet_rpc":                h.info.EnableWalletRPC,
		"admin_rpc":                 h.info.EnableAdminRPC,
		"miner_rpc":                 h.info.EnableMinerRPC,
		"faucet_rpc":                h.info.EnableFaucetRPC,
		"service_rpc":               h.info.EnableServiceRPC,
		"mainnet_available":         false,
		"testnet_value_warning":     "testnet IDR has no monetary value",
		"indexer_mode":              "simple_scan",
	})
}

func (h handler) explorerBlocks(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := explorerLimitOffset(w, r, 20, 100)
	if !ok {
		return
	}
	blocks, _, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	all := make([]map[string]any, 0, len(blocks))
	for i := len(blocks) - 1; i >= 0; i-- {
		all = append(all, explorerBlockSummary(blocks[i]))
	}
	items := paginateMaps(all, limit, offset)
	resp := explorerPagedResponse("blocks", items, len(all), limit, offset)
	writeJSON(w, http.StatusOK, resp)
}

func (h handler) explorerSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		explorerError(w, http.StatusBadRequest, "empty_query", "search query is required")
		return
	}
	if len(query) > 128 {
		explorerError(w, http.StatusBadRequest, "invalid_query", "search query is too long")
		return
	}

	blocks, pending, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	results := []map[string]any{}
	if isUnsignedInteger(query) {
		height, err := strconv.ParseUint(query, 10, 64)
		if err != nil {
			explorerError(w, http.StatusBadRequest, "invalid_height", "block height is invalid")
			return
		}
		for _, block := range blocks {
			if block.Height == height {
				results = append(results, explorerSearchBlockResult(block))
				break
			}
		}
	} else if strings.HasPrefix(query, "IDR") {
		if err := crypto.ValidateAddressForNetwork(query, h.profile()); err != nil {
			explorerError(w, http.StatusBadRequest, "invalid_address", err.Error())
			return
		}
		results = append(results, map[string]any{
			"type":     "address",
			"label":    "Address " + query,
			"path":     "/explorer-ui/#/address/" + query,
			"api_path": "/explorer/address/" + query,
			"address":  query,
		})
	} else if isHexHash(query) {
		for _, block := range blocks {
			if strings.EqualFold(block.Hash, query) {
				results = append(results, explorerSearchBlockResult(block))
				break
			}
		}
		for _, block := range blocks {
			for _, tx := range block.Transactions {
				if strings.EqualFold(tx.ID, query) {
					results = append(results, explorerSearchTxResult(tx.ID, "confirmed", &block))
				}
			}
		}
		for _, tx := range pending {
			if strings.EqualFold(tx.ID, query) {
				results = append(results, explorerSearchTxResult(tx.ID, "pending", nil))
			}
		}
	} else {
		explorerError(w, http.StatusBadRequest, "invalid_query", "search query must be a block height, 64-character hash, or IDR address")
		return
	}

	if len(results) == 0 {
		explorerError(w, http.StatusNotFound, "not_found", "no explorer result found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "query": query, "results": results, "count": len(results)})
}

func explorerSearchBlockResult(block types.Block) map[string]any {
	return map[string]any{
		"type":     "block",
		"label":    fmt.Sprintf("Block %d", block.Height),
		"path":     fmt.Sprintf("/explorer-ui/#/block/%d", block.Height),
		"api_path": fmt.Sprintf("/explorer/blocks/%d", block.Height),
		"height":   block.Height,
		"hash":     block.Hash,
	}
}

func explorerSearchTxResult(txID, status string, block *types.Block) map[string]any {
	out := map[string]any{
		"type":     "tx",
		"label":    "Transaction " + txID,
		"path":     "/explorer-ui/#/tx/" + txID,
		"api_path": "/explorer/tx/" + txID,
		"txid":     txID,
		"status":   status,
	}
	if block != nil {
		out["block_height"] = block.Height
		out["block_hash"] = block.Hash
	}
	return out
}

func (h handler) explorerBlockByHeight(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/explorer/blocks/")
	if raw == "" || strings.Contains(raw, "/") {
		explorerError(w, http.StatusNotFound, "not_found", "explorer endpoint not found")
		return
	}
	height, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		explorerError(w, http.StatusBadRequest, "invalid_height", "block height is invalid")
		return
	}
	blocks, _, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	for _, block := range blocks {
		if block.Height == height {
			writeJSON(w, http.StatusOK, explorerBlockDetail(block, len(blocks)-1))
			return
		}
	}
	explorerError(w, http.StatusNotFound, "block_not_found", "block not found")
}

func (h handler) explorerBlockByHash(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimPrefix(r.URL.Path, "/explorer/block/")
	if !isHexHash(hash) {
		explorerError(w, http.StatusBadRequest, "invalid_hash", "block hash must be a 64-character hex string")
		return
	}
	blocks, _, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	for _, block := range blocks {
		if strings.EqualFold(block.Hash, hash) {
			writeJSON(w, http.StatusOK, explorerBlockDetail(block, len(blocks)-1))
			return
		}
	}
	explorerError(w, http.StatusNotFound, "block_not_found", "block not found")
}

func (h handler) explorerTx(w http.ResponseWriter, r *http.Request) {
	txID := strings.TrimPrefix(r.URL.Path, "/explorer/tx/")
	if !isHexHash(txID) {
		explorerError(w, http.StatusBadRequest, "invalid_txid", "transaction id must be a 64-character hex string")
		return
	}
	blocks, pending, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	for _, block := range blocks {
		for _, tx := range block.Transactions {
			if strings.EqualFold(tx.ID, txID) {
				writeJSON(w, http.StatusOK, explorerTxDetail(tx, "confirmed", &block, blocks[len(blocks)-1].Height))
				return
			}
		}
	}
	for _, tx := range pending {
		if strings.EqualFold(tx.ID, txID) {
			writeJSON(w, http.StatusOK, explorerTxDetail(tx, "pending", nil, blocks[len(blocks)-1].Height))
			return
		}
	}
	explorerError(w, http.StatusNotFound, "tx_not_found", "transaction not found")
}

func (h handler) explorerAddressRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/explorer/address/")
	if strings.HasSuffix(path, "/txs") {
		h.explorerAddressTxs(w, r, strings.TrimSuffix(path, "/txs"))
		return
	}
	if strings.HasSuffix(path, "/stakes") {
		h.explorerAddressStakes(w, r, strings.TrimSuffix(path, "/stakes"))
		return
	}
	h.explorerAddress(w, r, path)
}

func (h handler) explorerAddress(w http.ResponseWriter, _ *http.Request, address string) {
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		explorerError(w, http.StatusBadRequest, "invalid_address", err.Error())
		return
	}
	blocks, pending, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	details, err := h.openChainBalanceDetails(address, pending)
	if err != nil {
		writeError(w, err)
		return
	}
	txCount, received, sent, first, last := explorerAddressStats(address, blocks, pending)
	score, _ := h.serviceStore().Score(address)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                          true,
		"address":                     address,
		"network":                     h.profile().Name,
		"confirmed_balance":           amount.Format(details.Confirmed),
		"mature_balance":              amount.Format(details.Mature),
		"immature_balance":            amount.Format(details.Immature),
		"spendable_balance":           amount.Format(details.Spendable),
		"active_stake":                amount.Format(details.ActiveStake),
		"unlocking_stake":             amount.Format(details.UnlockingStake),
		"total_received":              amount.Format(received),
		"total_sent":                  amount.Format(sent),
		"tx_count":                    txCount,
		"first_seen_height":           first,
		"last_seen_height":            last,
		"service_collateral_required": amount.Format(score.RequiredStake),
		"service_collateral_eligible": score.StakeEligible,
		"service_points":              score.SimulatedPoints,
		"simulation_only":             true,
		"service_points_warning":      "service points are simulation-only and are not spendable IDR",
	})
}

func (h handler) openChainBalanceDetails(address string, pending []types.Transaction) (ledger.BalanceDetails, error) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		return ledger.BalanceDetails{}, err
	}
	defer closeFn()
	return bc.BalanceDetailsForWithProfile(address, pending, h.profile())
}

func (h handler) explorerAddressTxs(w http.ResponseWriter, r *http.Request, address string) {
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		explorerError(w, http.StatusBadRequest, "invalid_address", err.Error())
		return
	}
	limit, offset, ok := explorerLimitOffset(w, r, 20, 100)
	if !ok {
		return
	}
	blocks, pending, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	var all []map[string]any
	tipHeight := blocks[len(blocks)-1].Height
	for i := len(blocks) - 1; i >= 0; i-- {
		block := blocks[i]
		for _, tx := range block.Transactions {
			if txInvolvesAddress(tx, address) {
				item := explorerTxDetail(tx, "confirmed", &block, tipHeight)
				item["amount_delta"] = formatSignedAmount(txDeltaForAddress(tx, address))
				all = append(all, item)
			}
		}
	}
	for _, tx := range pending {
		if txInvolvesAddress(tx, address) {
			item := explorerTxDetail(tx, "pending", nil, tipHeight)
			item["amount_delta"] = formatSignedAmount(txDeltaForAddress(tx, address))
			all = append(all, item)
		}
	}
	items := paginateMaps(all, limit, offset)
	resp := explorerPagedResponse("transactions", items, len(all), limit, offset)
	resp["address"] = address
	writeJSON(w, http.StatusOK, resp)
}

func (h handler) explorerAddressStakes(w http.ResponseWriter, _ *http.Request, address string) error {
	return nil
}

func (h handler) explorerStakes(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := explorerLimitOffset(w, r, 20, 100)
	if !ok {
		return
	}
	records, err := h.explorerStakeRecords("")
	if err != nil {
		writeError(w, err)
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" {
		filtered := records[:0]
		for _, record := range records {
			if record["status"] == status {
				filtered = append(filtered, record)
			}
		}
		records = filtered
	}
	items := paginateMaps(records, limit, offset)
	writeJSON(w, http.StatusOK, explorerPagedResponse("stakes", items, len(records), limit, offset))
}

func (h handler) explorerServices(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := explorerLimitOffset(w, r, 20, 100)
	if !ok {
		return
	}
	nodes, err := h.serviceStore().LoadNodes()
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, h.explorerServiceSummary(node.OwnerAddress, &node))
	}
	resp := explorerPagedResponse("services", paginateMaps(items, limit, offset), len(items), limit, offset)
	resp["simulation_only"] = true
	resp["scope"] = "local_node_service_store"
	writeJSON(w, http.StatusOK, resp)
}

func (h handler) explorerService(w http.ResponseWriter, r *http.Request) {
	address := strings.TrimPrefix(r.URL.Path, "/explorer/service/")
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		explorerError(w, http.StatusBadRequest, "invalid_address", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.explorerServiceSummary(address, nil))
}

func (h handler) nodeID(w http.ResponseWriter, _ *http.Request) {
	nodeID, err := p2p.LoadOrCreateNodeID(h.paths.NodeID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"node_id": nodeID})
}

func (h handler) nodeCompare(w http.ResponseWriter, r *http.Request) {
	peer := r.URL.Query().Get("peer")
	if peer == "" {
		writeError(w, errors.New("peer is required"))
		return
	}
	local, err := h.compareInfo()
	if err != nil {
		writeError(w, err)
		return
	}
	remote, err := remoteCompareInfo(peer)
	if err != nil {
		writeError(w, err)
		return
	}
	inSync := fmt.Sprint(local["network_id"]) == fmt.Sprint(remote["network_id"]) &&
		fmt.Sprint(local["chain_id"]) == fmt.Sprint(remote["chain_id"]) &&
		fmt.Sprint(local["genesis_hash"]) == fmt.Sprint(remote["genesis_hash"]) &&
		fmt.Sprint(local["height"]) == fmt.Sprint(remote["height"]) &&
		fmt.Sprint(local["tip_hash"]) == fmt.Sprint(remote["tip_hash"])
	writeJSON(w, http.StatusOK, map[string]any{"in_sync": inSync, "local": local, "peer": remote})
}

func (h handler) nodeStatus(w http.ResponseWriter, _ *http.Request) {
	net := h.profile()
	if h.info.State != nil {
		snapshot := h.info.State.Snapshot()
		peers, _ := p2p.NewPeerStore(h.paths.Peers).Load()
		_, locked := p2p.IsLocked(h.paths.Lock)
		writeJSON(w, http.StatusOK, map[string]any{
			"datadir":          h.paths.DataDir,
			"network":          net.Name,
			"network_id":       net.NetworkID,
			"chain_id":         net.ChainID,
			"protocol_version": net.ProtocolVersion,
			"rpc_api_version":  net.RPCAPIVersion,
			"height":           snapshot.Height,
			"tip_hash":         snapshot.TipHash,
			"peers":            len(peers),
			"p2p_listen":       h.info.P2PListen,
			"p2p_advertise":    h.p2pAdvertise(),
			"rpc_listen":       h.info.RPCListen,
			"public_rpc":       h.info.PublicRPC,
			"wallet_rpc":       h.info.EnableWalletRPC,
			"miner_rpc":        h.info.EnableMinerRPC,
			"admin_rpc":        h.info.EnableAdminRPC,
			"faucet_rpc":       h.info.EnableFaucetRPC,
			"service_rpc":      h.info.EnableServiceRPC,
			"mempool_pending":  snapshot.MempoolCount,
			"locked":           locked,
		})
		return
	}
	bc, closeFn, err := h.openChain()
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
	peers, _ := p2p.NewPeerStore(h.paths.Peers).Load()
	pending, _ := mempool.New(h.paths.Mempool).Load()
	_, locked := p2p.IsLocked(h.paths.Lock)
	writeJSON(w, http.StatusOK, map[string]any{
		"datadir":          h.paths.DataDir,
		"network":          net.Name,
		"network_id":       net.NetworkID,
		"chain_id":         net.ChainID,
		"protocol_version": net.ProtocolVersion,
		"rpc_api_version":  net.RPCAPIVersion,
		"height":           tip.Height,
		"tip_hash":         tip.Hash,
		"peers":            len(peers),
		"p2p_listen":       h.info.P2PListen,
		"p2p_advertise":    h.p2pAdvertise(),
		"rpc_listen":       h.info.RPCListen,
		"public_rpc":       h.info.PublicRPC,
		"wallet_rpc":       h.info.EnableWalletRPC,
		"miner_rpc":        h.info.EnableMinerRPC,
		"admin_rpc":        h.info.EnableAdminRPC,
		"faucet_rpc":       h.info.EnableFaucetRPC,
		"service_rpc":      h.info.EnableServiceRPC,
		"mempool_pending":  len(pending),
		"locked":           locked,
	})
}

func (h handler) debugP2P(w http.ResponseWriter, _ *http.Request) {
	nodeID, err := p2p.LoadOrCreateNodeID(h.paths.NodeID)
	if err != nil {
		writeError(w, err)
		return
	}
	height, tipHash := h.debugChainTip()
	peers, err := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	views := make([]map[string]any, 0, len(peers))
	for _, peer := range peers {
		views = append(views, map[string]any{
			"url":                  peer.URL,
			"status":               peer.Status,
			"score":                peer.Score,
			"last_seen_at":         peer.LastSeenAt,
			"last_error":           peer.LastError,
			"last_latency_ms":      peer.LastLatencyMS,
			"last_status_check_at": peer.LastStatusCheckAt,
			"source":               peer.Source,
			"node_id":              peer.NodeID,
			"network_id":           peer.NetworkID,
			"chain_id":             peer.ChainID,
			"genesis_hash":         peer.GenesisHash,
			"protocol_version":     peer.ProtocolVersion,
			"height":               peer.LastHeight,
			"tip_hash":             peer.LastTipHash,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"node_id": nodeID, "height": height, "tip_hash": tipHash, "peers": views})
}

func (h handler) debugP2PPing(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	result, err := p2p.Ping(req.URL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}
	_ = p2p.NewPeerStore(h.paths.Peers).UpdateLatency(req.URL, result.StatusLatencyMS, result.Error)
	writeJSON(w, http.StatusOK, result)
}

func (h handler) mineStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, miningJobView(h.info.Mining.Snapshot()))
}

func (h handler) debugLocks(w http.ResponseWriter, _ *http.Request) {
	job := h.info.Mining.Snapshot()
	height, tipHash := h.debugChainTip()
	peers, _ := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	mempoolCount := 0
	if h.info.State != nil {
		mempoolCount = h.info.State.Snapshot().MempoolCount
	} else if pending, err := mempool.New(h.paths.Mempool).Load(); err == nil {
		mempoolCount = len(pending)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"mining_running": job.Status == mining.StatusRunning,
		"mining_job_id":  job.ID,
		"height":         height,
		"tip_hash":       tipHash,
		"mempool_count":  mempoolCount,
		"peer_count":     len(peers),
	})
}

func (h handler) peers(w http.ResponseWriter, _ *http.Request) {
	peers, err := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	banned := 0
	for _, peer := range peers {
		if peer.Status == p2p.PeerStatusBad {
			banned++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"peers": peers, "peer_count": len(peers), "max_peers": h.info.MaxPeers, "banned_count": banned})
}

func (h handler) p2pPeers(w http.ResponseWriter, _ *http.Request) {
	peers, err := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	views := make([]p2p.PeerView, 0, len(peers))
	active, seeds := 0, 0
	for _, peer := range peers {
		view := p2p.PeerViewFromMetadata(peer)
		if view.Status == p2p.PeerStatusActive {
			active++
		}
		if strings.Contains(peer.Source, "seed") || strings.Contains(peer.Source, "profile") || strings.Contains(peer.Source, "file") || strings.Contains(peer.Source, "cli") {
			seeds++
		}
		views = append(views, view)
	}
	profile := h.profile()
	writeJSON(w, http.StatusOK, p2p.PeersResponse{
		Network:        profile.Name,
		NetworkID:      profile.NetworkID,
		ChainID:        profile.ChainID,
		GenesisHash:    chain.GenesisHashForNetwork(profile),
		AdvertiseP2P:   h.p2pAdvertise(),
		KnownPeers:     views,
		KnownCount:     len(views),
		ActiveCount:    active,
		SeedCount:      seeds,
		MaxPeerEntries: p2p.DefaultMaxStoredPeers,
	})
}

func (h handler) p2pReputation(w http.ResponseWriter, _ *http.Request) {
	peers, err := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	profile := h.profile()
	reputation := p2p.ReputationViews(peers)
	writeJSON(w, http.StatusOK, map[string]any{
		"network":          profile.Name,
		"network_id":       profile.NetworkID,
		"chain_id":         profile.ChainID,
		"genesis_hash":     chain.GenesisHashForNetwork(profile),
		"peer_count":       len(peers),
		"reputation_count": len(reputation),
		"reputation":       reputation,
	})
}

func (h handler) p2pDiscovery(w http.ResponseWriter, _ *http.Request) {
	peers, err := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	active, failed, retryPending, discovered := 0, 0, 0, 0
	now := time.Now()
	for _, peer := range peers {
		view := p2p.PeerViewFromMetadata(peer)
		switch view.Status {
		case p2p.PeerStatusActive:
			active++
		case p2p.PeerStatusOffline, p2p.PeerStatusCooldown, p2p.PeerStatusBad:
			failed++
		}
		if strings.Contains(peer.Source, "discovered") || strings.Contains(peer.Source, "maintenance") {
			discovered++
		}
		if peer.NextRetryAt != "" {
			next, err := time.Parse(time.RFC3339, peer.NextRetryAt)
			if err == nil && now.Before(next) {
				retryPending++
			}
		}
	}
	profile := h.profile()
	writeJSON(w, http.StatusOK, map[string]any{
		"network":             profile.Name,
		"network_id":          profile.NetworkID,
		"chain_id":            profile.ChainID,
		"genesis_hash":        chain.GenesisHashForNetwork(profile),
		"known_peer_count":    len(peers),
		"active_peer_count":   active,
		"failed_peer_count":   failed,
		"discovered_count":    discovered,
		"retry_pending_count": retryPending,
		"peer_ttl_seconds":    int64(p2p.DefaultPeerTTL.Seconds()),
		"max_discovered":      p2p.DefaultMaxDiscoveredPeers,
		"max_peers":           h.info.MaxPeers,
		"max_peers_per_ip":    p2p.DefaultMaxPeersPerIP,
		"maintenance":         "enabled",
	})
}

func (h handler) p2pBootstrap(w http.ResponseWriter, _ *http.Request) {
	peers, err := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	seeds := make([]p2p.PeerView, 0)
	bootnodes := make([]p2p.PeerView, 0)
	for _, peer := range peers {
		view := p2p.PeerViewFromMetadata(peer)
		if strings.Contains(peer.Source, "seed") || strings.Contains(peer.Source, "profile") || strings.Contains(peer.Source, "file") || strings.Contains(peer.Source, "cli") {
			seeds = append(seeds, view)
		}
		if strings.Contains(peer.Source, "bootnode") {
			bootnodes = append(bootnodes, view)
		}
	}
	profile := h.profile()
	writeJSON(w, http.StatusOK, map[string]any{
		"network":        profile.Name,
		"network_id":     profile.NetworkID,
		"chain_id":       profile.ChainID,
		"genesis_hash":   chain.GenesisHashForNetwork(profile),
		"seed_count":     len(seeds),
		"bootnode_count": len(bootnodes),
		"seeds":          seeds,
		"bootnodes":      bootnodes,
	})
}

func (h handler) p2pKnownPeers(w http.ResponseWriter, _ *http.Request) {
	peers, err := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	views := make([]p2p.PeerView, 0, len(peers))
	for _, peer := range peers {
		views = append(views, p2p.PeerViewFromMetadata(peer))
	}
	profile := h.profile()
	writeJSON(w, http.StatusOK, map[string]any{
		"network":          profile.Name,
		"network_id":       profile.NetworkID,
		"chain_id":         profile.ChainID,
		"genesis_hash":     chain.GenesisHashForNetwork(profile),
		"known_peer_count": len(peers),
		"known_peers":      views,
	})
}

func (h handler) peerAdd(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	store := p2p.NewPeerStore(h.paths.Peers)
	if err := h.ensurePeerCapacity(store, req.URL); err != nil {
		writeError(w, err)
		return
	}
	hs, err := p2p.CheckPeerWithProfile(h.paths, req.URL, h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	meta := p2p.MetadataFromHandshake(req.URL, hs, 5)
	meta.Source = "manual"
	if err := store.Upsert(meta); err != nil {
		writeError(w, err)
		return
	}
	_ = store.AdjustPeerScore(req.URL, 0, "peer add")
	writeJSON(w, http.StatusOK, map[string]any{"added": true, "url": req.URL})
}

func (h handler) peerConnect(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	store := p2p.NewPeerStore(h.paths.Peers)
	if err := h.ensurePeerCapacity(store, req.URL); err != nil {
		writeError(w, err)
		return
	}
	hs, err := p2p.CheckPeerWithProfile(h.paths, req.URL, h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	meta := p2p.MetadataFromHandshake(req.URL, hs, 5)
	meta.Source = "peer connect"
	if err := store.Upsert(meta); err != nil {
		writeError(w, err)
		return
	}
	_ = store.AdjustPeerScore(req.URL, 0, "peer connect")
	introduced := false
	introError := ""
	if advertise := h.p2pAdvertise(); advertise != "" {
		nodeID, _ := p2p.LoadOrCreateNodeID(h.paths.NodeID)
		net := h.profile()
		intro := p2p.PeerIntroduction{
			URL:       advertise,
			NodeID:    nodeID,
			NetworkID: net.NetworkID,
			ChainID:   net.ChainID,
		}
		if _, err := p2p.NewClient().IntroducePeer(req.URL, intro); err != nil {
			introError = err.Error()
		} else {
			introduced = true
		}
	}
	response := map[string]any{"connected": true, "url": req.URL, "introduced": introduced}
	if introError != "" {
		response["introduce_error"] = introError
	}
	writeJSON(w, http.StatusOK, response)
}

func (h handler) peerRemove(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	if err := p2p.NewPeerStore(h.paths.Peers).Remove(req.URL); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"removed": true, "url": req.URL})
}

func (h handler) peerClear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Yes bool `json:"yes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if !req.Yes {
		writeError(w, errors.New("refusing to clear peers without --yes"))
		return
	}
	if err := p2p.NewPeerStore(h.paths.Peers).Clear(); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"cleared": true})
}

func (h handler) peerCheck(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	hs, err := p2p.CheckPeerWithProfile(h.paths, req.URL, h.profile())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "url": req.URL, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":               true,
		"url":              req.URL,
		"node_id":          hs.NodeID,
		"network_id":       hs.NetworkID,
		"chain_id":         hs.ChainID,
		"height":           hs.Height,
		"tip_hash":         hs.TipHash,
		"genesis_hash":     hs.GenesisHash,
		"protocol_version": hs.ProtocolVersion,
	})
}

func (h handler) peerStatus(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		IncludeBad bool `json:"include_bad"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	store := p2p.NewPeerStore(h.paths.Peers)
	peers, err := store.LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	active, bad := 0, 0
	for _, peer := range peers {
		if peer.Status == p2p.PeerStatusBad && !req.IncludeBad {
			bad++
			continue
		}
		if _, err := p2p.CheckPeerWithProfile(h.paths, peer.URL, h.profile()); err != nil {
			bad++
		} else {
			active++
		}
	}
	peers, _ = store.LoadMetadata()
	unknown := 0
	for _, peer := range peers {
		if peer.Status == p2p.PeerStatusUnknown {
			unknown++
		}
	}
	peerViews := make([]map[string]any, 0, len(peers))
	for _, peer := range peers {
		peerViews = append(peerViews, peerView(peer))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"checked_peers": len(peers),
		"active":        active,
		"bad":           bad,
		"unknown":       unknown,
		"peers":         peerViews,
	})
}

func (h handler) peerStatusGet(w http.ResponseWriter, r *http.Request) {
	h.peerStatus(w, r)
}

func (h handler) peerHealth(w http.ResponseWriter, _ *http.Request) {
	store := p2p.NewPeerStore(h.paths.Peers)
	peers, err := store.LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	height, tipHash := h.debugChainTip()
	active, seedCount, bestHeight := 0, 0, uint64(0)
	peerViews := make([]map[string]any, 0, len(peers))
	for _, peer := range peers {
		if strings.Contains(peer.Source, "seed") {
			seedCount++
		}
		if peer.Status == p2p.PeerStatusActive {
			active++
		}
		if peer.LastHeight > bestHeight {
			bestHeight = peer.LastHeight
		}
		peerViews = append(peerViews, peerView(peer))
	}
	profile := h.profile()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                true,
		"network":           profile.Name,
		"network_id":        profile.NetworkID,
		"chain_id":          profile.ChainID,
		"genesis_hash":      chain.GenesisHashForNetwork(profile),
		"local_height":      height,
		"local_tip":         tipHash,
		"active_peer_count": active,
		"known_peer_count":  len(peers),
		"seed_count":        seedCount,
		"best_peer_height":  bestHeight,
		"peers":             peerViews,
	})
}

func (h handler) peerSeeds(w http.ResponseWriter, _ *http.Request) {
	store := p2p.NewPeerStore(h.paths.Peers)
	peers, err := store.LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	seeds := make([]map[string]any, 0)
	for _, peer := range peers {
		if strings.Contains(peer.Source, "seed") || strings.Contains(peer.Source, "profile") || strings.Contains(peer.Source, "file") || strings.Contains(peer.Source, "cli") {
			seeds = append(seeds, peerView(peer))
		}
	}
	profile := h.profile()
	writeJSON(w, http.StatusOK, map[string]any{
		"network":      profile.Name,
		"network_id":   profile.NetworkID,
		"chain_id":     profile.ChainID,
		"genesis_hash": chain.GenesisHashForNetwork(profile),
		"seed_count":   len(seeds),
		"seeds":        seeds,
	})
}

func (h handler) peerCount() int {
	peers, err := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	if err != nil {
		return 0
	}
	return len(peers)
}

func (h handler) activePeerCount() int {
	peers, err := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	if err != nil {
		return 0
	}
	active := 0
	for _, peer := range peers {
		if peer.Status == p2p.PeerStatusActive {
			active++
		}
	}
	return active
}

func (h handler) upstreamReachableCount() int {
	if len(h.info.UpstreamPeers) == 0 {
		return 0
	}
	net := h.profile()
	net.GenesisHash = chain.GenesisHashForNetwork(net)
	client := p2p.NewClientWithTimeout(upstreamReachabilityTimeout)
	reachable := 0
	for _, peer := range h.info.UpstreamPeers {
		hs, err := client.Handshake(peer)
		if err != nil {
			continue
		}
		if err := p2p.ValidateHandshake(net, hs); err != nil {
			continue
		}
		reachable++
	}
	return reachable
}

func (h handler) miningTemplateGuard() (bool, string, int, int) {
	minPeers := h.minMiningPeers()
	activePeers := h.activePeerCount()
	upstreamReachable := h.upstreamReachableCount()
	if h.allowIsolatedMining() || minPeers <= 0 || h.profile().Name != "testnet" {
		return true, "", activePeers, upstreamReachable
	}
	if activePeers >= minPeers || upstreamReachable > 0 {
		return true, "", activePeers, upstreamReachable
	}
	return false, "miner RPC temporarily disabled: insufficient active peers/upstream unavailable; isolated mining is disabled", activePeers, upstreamReachable
}

func (h handler) writeGuard() (bool, string) {
	minPeers := h.minWritePeers()
	if h.allowIsolatedWrites() || minPeers <= 0 || h.profile().Name != "testnet" {
		return true, ""
	}
	if h.activePeerCount() >= minPeers || h.upstreamReachableCount() > 0 {
		return true, ""
	}
	return false, "faucet temporarily disabled: insufficient active peers/upstream unavailable; isolated writes are disabled"
}

func (h handler) scheduleUpstreamBackfill() {
	if len(h.info.UpstreamPeers) == 0 {
		return
	}
	peers := append([]string(nil), h.info.UpstreamPeers...)
	go func() {
		unlock := h.lockChainMutation()
		defer unlock()
		p2p.BackfillToPeers(h.paths, peers, h.profile(), h.maxReorgDepth())
	}()
}

func (h handler) peerDiscover(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		Peer  string `json:"peer"`
		Limit int    `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	store := p2p.NewPeerStore(h.paths.Peers)
	peers, err := store.LoadMetadata()
	if err != nil {
		writeError(w, err)
		return
	}
	targets := []string{}
	if req.Peer != "" {
		targets = append(targets, req.Peer)
	} else {
		for _, peer := range p2p.SelectPeers(peers, false, 8) {
			targets = append(targets, peer.URL)
		}
	}
	results := make([]p2p.DiscoveryResult, 0, len(targets))
	added := 0
	for _, target := range targets {
		result, err := p2p.DiscoverFromPeer(h.paths, target, h.profile(), h.p2pAdvertise(), req.Limit)
		if err != nil {
			result = p2p.DiscoveryResult{Peer: target, Errors: []string{err.Error()}}
		}
		added += result.Added
		results = append(results, result)
	}
	writeJSON(w, http.StatusOK, map[string]any{"discovered": added, "results": results})
}

func (h handler) upstreamStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"upstream_peer_count":      len(h.info.UpstreamPeers),
		"upstream_peers":           h.info.UpstreamPeers,
		"upstream_reachable_count": h.upstreamReachableCount(),
		"active_peer_count":        h.activePeerCount(),
	})
}

func (h handler) upstreamPush(w http.ResponseWriter, r *http.Request) {
	unlock := h.lockChainMutation()
	defer unlock()
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		Peer string `json:"peer"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Peer) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "peer is required"})
		return
	}
	result, err := p2p.BackfillToPeer(h.paths, req.Peer, h.profile(), h.maxReorgDepth())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error(), "result": result})
		return
	}
	h.refreshState()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

func (h handler) upstreamPushAll(w http.ResponseWriter, _ *http.Request) {
	unlock := h.lockChainMutation()
	defer unlock()
	results := p2p.BackfillToPeers(h.paths, h.info.UpstreamPeers, h.profile(), h.maxReorgDepth())
	h.refreshState()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "results": results, "upstream_peer_count": len(h.info.UpstreamPeers)})
}

type reorgRPCRequest struct {
	Peer     string `json:"peer"`
	MaxDepth uint64 `json:"max_depth"`
	Yes      bool   `json:"yes"`
}

func (h handler) reorgPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req reorgRPCRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.MaxDepth == 0 {
		req.MaxDepth = h.maxReorgDepth()
	}
	plan, _, err := p2p.BuildReorgPlanWithProfile(h.paths, req.Peer, req.MaxDepth, h.profile())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"allowed": false, "reason": err.Error(), "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"allowed": plan.Allowed, "reason": plan.Reason, "decision": plan.Decision, "network": plan.Network, "network_id": plan.NetworkID, "chain_id": plan.ChainID, "local_height": plan.LocalHeight, "peer_height": plan.PeerHeight, "common_ancestor_height": plan.CommonAncestorHeight, "common_ancestor_hash": plan.CommonAncestorHash, "disconnect_blocks": len(plan.DisconnectBlocks), "connect_blocks": len(plan.ConnectBlocks), "reorg_depth": plan.ReorgDepth, "max_reorg_depth": plan.MaxReorgDepth, "local_work": plan.LocalWork, "peer_work": plan.PeerWork})
}

func (h handler) reorgApply(w http.ResponseWriter, r *http.Request) {
	unlock := h.lockChainMutation()
	defer unlock()
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req reorgRPCRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.MaxDepth == 0 {
		req.MaxDepth = h.maxReorgDepth()
	}
	res, err := p2p.ApplyReorgWithProfile(h.paths, req.Peer, req.MaxDepth, req.Yes, h.profile())
	if err != nil {
		h.refreshState()
		writeJSON(w, http.StatusBadRequest, map[string]any{"applied": false, "error": res.Error})
		return
	}
	h.refreshState()
	h.scheduleUpstreamBackfill()
	writeJSON(w, http.StatusOK, map[string]any{"applied": true, "network": res.Plan.Network, "network_id": res.Plan.NetworkID, "chain_id": res.Plan.ChainID, "old_height": res.OldHeight, "new_height": res.NewHeight, "old_tip": res.OldTip, "new_tip": res.NewTip, "disconnected_blocks": res.DisconnectedBlocks, "connected_blocks": res.ConnectedBlocks, "requeued_transactions": res.RequeuedTransactions, "dropped_transactions": res.DroppedTransactions, "dropped_confirmed_transactions": res.DroppedConfirmedTransactions, "dropped_invalid_transactions": res.DroppedInvalidTransactions, "dropped_duplicate_transactions": res.DroppedDuplicateTransactions, "mempool_count": res.MempoolCount, "chain_valid": res.ChainValid})
}

func (h handler) peerSync(w http.ResponseWriter, r *http.Request) {
	unlock := h.lockChainMutation()
	defer unlock()
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		Peer          string `json:"peer"`
		IncludeBad    bool   `json:"include_bad"`
		AllowReorg    bool   `json:"allow_reorg"`
		MaxReorgDepth uint64 `json:"max_reorg_depth"`
		Yes           bool   `json:"yes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	var peers []p2p.PeerMetadata
	if req.Peer != "" {
		peers = []p2p.PeerMetadata{{URL: req.Peer}}
	} else {
		var err error
		peers, err = p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
		if err != nil {
			writeError(w, err)
			return
		}
	}
	before, _ := h.currentHeight()
	type syncResult struct {
		Peer                 string `json:"peer"`
		OK                   bool   `json:"ok"`
		HeightBefore         uint64 `json:"height_before"`
		HeightAfter          uint64 `json:"height_after"`
		ImportedBlocks       uint64 `json:"imported_blocks"`
		Message              string `json:"message"`
		PeerHeight           uint64 `json:"peer_height,omitempty"`
		CommonAncestorHeight uint64 `json:"common_ancestor_height,omitempty"`
		CommonAncestorHash   string `json:"common_ancestor_hash,omitempty"`
	}
	var results []syncResult
	success := true
	var firstErr string
	var firstFork *p2p.ForkError
	var firstReorgFork *p2p.ReorgSyncError
	for _, peer := range peers {
		if peer.Status == p2p.PeerStatusBad && !req.IncludeBad {
			continue
		}
		hBefore, _ := h.currentHeight()
		maxReorgDepth := req.MaxReorgDepth
		if maxReorgDepth == 0 {
			maxReorgDepth = h.maxReorgDepth()
		}
		err := p2p.SyncFromPeerWithProfileAndMaxDepth(h.paths, peer.URL, nil, h.profile(), maxReorgDepth)
		if err != nil && req.AllowReorg {
			if !req.Yes {
				err = errors.New("refusing to apply reorg without --yes")
			} else if res, reorgErr := p2p.ApplyReorgWithProfile(h.paths, peer.URL, maxReorgDepth, true, h.profile()); reorgErr == nil && res.Applied {
				err = nil
			}
		}
		hAfter, _ := h.currentHeight()
		imported := uint64(0)
		if hAfter > hBefore {
			imported = hAfter - hBefore
		}
		result := syncResult{Peer: peer.URL, HeightBefore: hBefore, HeightAfter: hAfter, ImportedBlocks: imported}
		if err != nil {
			result.OK = false
			result.Message = err.Error()
			success = false
			if firstErr == "" {
				firstErr = err.Error()
			}
			var forkErr *p2p.ForkError
			if errors.As(err, &forkErr) {
				result.Message = "fork detected"
				result.PeerHeight = forkErr.Result.PeerHeight
				result.CommonAncestorHeight = forkErr.Result.CommonAncestorHeight
				result.CommonAncestorHash = forkErr.Result.CommonAncestorHash
				if firstFork == nil {
					firstFork = forkErr
				}
			}
			var reorgErr *p2p.ReorgSyncError
			if errors.As(err, &reorgErr) {
				result.Message = "fork detected"
				result.PeerHeight = reorgErr.Plan.PeerHeight
				result.CommonAncestorHeight = reorgErr.Plan.CommonAncestorHeight
				result.CommonAncestorHash = reorgErr.Plan.CommonAncestorHash
				if firstReorgFork == nil {
					firstReorgFork = reorgErr
				}
			}
		} else {
			result.OK = true
			if imported == 0 {
				result.Message = "local chain already up to date"
			} else {
				result.Message = "sync complete"
			}
		}
		results = append(results, result)
	}
	after, _ := h.currentHeight()
	imported := uint64(0)
	if after > before {
		imported = after - before
	}
	if imported > 0 {
		h.scheduleUpstreamBackfill()
	}
	status := http.StatusOK
	response := map[string]any{
		"synced":              true,
		"local_height_before": before,
		"local_height_after":  after,
		"imported_blocks":     imported,
		"peers_checked":       len(peers),
		"results":             results,
	}
	if !success {
		status = http.StatusBadRequest
		response["synced"] = false
		response["error"] = firstErr
		if firstFork != nil {
			response["error"] = "fork detected"
			response["message"] = firstFork.Error()
			response["peer_height"] = firstFork.Result.PeerHeight
			response["common_ancestor_height"] = firstFork.Result.CommonAncestorHeight
			response["common_ancestor_hash"] = firstFork.Result.CommonAncestorHash
			response["reorg_supported"] = firstFork.Result.ReorgSupported
		}
		if firstReorgFork != nil {
			response["error"] = "fork detected"
			response["message"] = firstReorgFork.Error()
			response["peer_height"] = firstReorgFork.Plan.PeerHeight
			response["common_ancestor_height"] = firstReorgFork.Plan.CommonAncestorHeight
			response["common_ancestor_hash"] = firstReorgFork.Plan.CommonAncestorHash
			response["reorg_supported"] = firstReorgFork.Plan.Allowed
			response["decision"] = firstReorgFork.Plan.Decision
			response["reason"] = firstReorgFork.Plan.Reason
			response["local_cumulative_work"] = firstReorgFork.Plan.LocalWork
			response["peer_cumulative_work"] = firstReorgFork.Plan.PeerWork
			response["reorg_depth"] = firstReorgFork.Plan.ReorgDepth
			response["max_reorg_depth"] = firstReorgFork.Plan.MaxReorgDepth
		}
	}
	h.refreshState()
	writeJSON(w, status, response)
}

func (h handler) chainInfo(w http.ResponseWriter, _ *http.Request) {
	blocks, pending, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	net := h.profile()
	stats := chain.CalculateChainStatsWithProfile(blocks, net)
	tip := blocks[len(blocks)-1]
	nextDifficulty := chain.CalculateNextDifficultyWithParams(blocks, net.Difficulty)
	response := h.chainInfoMap(blocks, tip, nextDifficulty, stats, len(pending))
	for k, v := range stakingSummaryMap(blocks, net) {
		response[k] = v
	}
	writeJSON(w, http.StatusOK, response)
}

func (h handler) chainDifficulty(w http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (h handler) miningStatus(w http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (h handler) miningDifficulty(w http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (h handler) miningBlocks(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (h handler) addMiningGuardFields(info map[string]any) {
	enabled, reason, activePeers, upstreamReachable := h.miningTemplateGuard()
	info["min_mining_peers"] = h.minMiningPeers()
	info["active_peer_count"] = activePeers
	info["upstream_peer_count"] = len(h.info.UpstreamPeers)
	info["upstream_reachable_count"] = upstreamReachable
	info["isolated_mining_allowed"] = h.allowIsolatedMining()
	info["mining_template_enabled"] = enabled
	if !enabled {
		info["mining_template_disabled_reason"] = reason
	}
}

func (h handler) chainInfoData() ([]types.Block, []types.Transaction, error) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		return nil, nil, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return nil, nil, err
	}
	pending, _ := mempool.New(h.paths.Mempool).Load()
	return blocks, pending, nil
}

func (h handler) chainInfoMap(blocks []types.Block, tip types.Block, nextDifficulty uint32, stats chain.ChainStats, pendingCount int) map[string]any {
	net := h.profile()
	params := net.Difficulty
	blocksUntilRetarget := chain.BlocksUntilRetarget(blocks, params)
	return map[string]any{
		"network":                   net.Name,
		"network_id":                net.NetworkID,
		"chain_id":                  net.ChainID,
		"genesis_hash":              chain.GenesisHashForNetwork(net),
		"protocol_version":          net.ProtocolVersion,
		"rpc_api_version":           net.RPCAPIVersion,
		"p2p_protocol_version":      net.P2PProtocolVersion,
		"block_version":             net.BlockVersion,
		"tx_version":                net.TxVersion,
		"height":                    tip.Height,
		"tip_hash":                  tip.Hash,
		"difficulty":                nextDifficulty,
		"tip_difficulty":            tip.Difficulty,
		"next_difficulty":           nextDifficulty,
		"target_block_time_seconds": params.TargetBlockTimeSeconds,
		"retarget_window":           params.RetargetWindow,
		"min_difficulty":             params.MinDifficulty,
		"max_difficulty":             params.MaxDifficulty,
		"blocks_until_retarget":     blocksUntilRetarget,
		"coinbase_maturity":         net.Consensus.CoinbaseMaturity,
		"total_supply":              amount.Format(stats.TotalSupply) + " " + config.Ticker,
		"cumulative_work":           stats.CumulativeWork,
		"pending_tx_count":          pendingCount,
		"blocks":                    stats.Blocks,
		"coinbase_blocks":           stats.CoinbaseBlocks,
		"total_transactions":        stats.TotalTransactions,
		"coinbase_transactions":     stats.CoinbaseTransactions,
		"normal_transactions":       stats.NormalTransactions,
		"circulating_supply":        amount.Format(stats.CirculatingSupply) + " " + config.Ticker,
		"datadir":                   h.paths.DataDir,
	}
}

func chainDifficultyMap(blocks []types.Block, net config.NetworkConfig) map[string]any {
	params := net.Difficulty
	tip := blocks[len(blocks)-1]
	return map[string]any{
		"network":                   net.Name,
		"chain_id":                  net.ChainID,
		"height":                    tip.Height,
		"tip_difficulty":            tip.Difficulty,
		"next_difficulty":           chain.CalculateNextDifficultyWithParams(blocks, params),
		"target_block_time_seconds": params.TargetBlockTimeSeconds,
		"retarget_window":           params.RetargetWindow,
		"min_difficulty":            params.MinDifficulty,
		"max_difficulty":            params.MaxDifficulty,
		"blocks_until_retarget":     chain.BlocksUntilRetarget(blocks, params),
		"cumulative_work":           chain.CalculateCumulativeWork(blocks),
	}
}

func miningMetricsMap(blocks []types.Block, net config.NetworkConfig, pendingCount, peerCount int) map[string]any {
	params := net.Difficulty
	tip := blocks[len(blocks)-1]
	nextDifficulty := chain.CalculateNextDifficultyWithParams(blocks, params)
	stats := chain.CalculateChainStatsWithProfile(blocks, net)
	intervals := recentBlockIntervals(blocks, int(params.RetargetWindow))
	avg, minInterval, maxInterval := intervalStats(intervals)
	lastBlockAge := int64(0)
	if tip.Height > 0 {
		lastBlockAge = time.Now().Unix() - tip.Timestamp
		if lastBlockAge < 0 {
			lastBlockAge = 0
		}
	}
	return map[string]any{
		"ok":                             true,
		"network":                        net.Name,
		"network_id":                     net.NetworkID,
		"chain_id":                       net.ChainID,
		"genesis_hash":                   chain.GenesisHashForNetwork(net),
		"height":                         tip.Height,
		"tip_hash":                       tip.Hash,
		"current_difficulty":             tip.Difficulty,
		"tip_difficulty":                 tip.Difficulty,
		"next_difficulty":                nextDifficulty,
		"target_block_time_seconds":      params.TargetBlockTimeSeconds,
		"retarget_window":                params.RetargetWindow,
		"blocks_until_retarget":          chain.BlocksUntilRetarget(blocks, params),
		"last_block_time":                tip.Timestamp,
		"last_block_age_seconds":         lastBlockAge,
		"recent_block_intervals_seconds": intervals,
		"average_interval_seconds":       avg,
		"min_interval_seconds":            minInterval,
		"max_interval_seconds":            maxInterval,
		"recent_difficulties":              recentDifficulties(blocks, int(params.RetargetWindow)),
		"projected_retarget_direction":    projectedRetargetDirection(tip.Difficulty, nextDifficulty, avg, params.TargetBlockTimeSeconds),
		"total_supply":                    amount.Format(stats.TotalSupply) + " " + config.Ticker,
		"coinbase_maturity":               net.Consensus.CoinbaseMaturity,
		"pending_tx_count":                pendingCount,
		"peer_count":                      peerCount,
		"note":                            "testnet mining is for testing only; testnet IDR has no monetary value",
	}
}

func recentBlockIntervals(blocks []types.Block, limit int) []int64 {
	if limit <= 0 {
		limit = 30
	}
	if len(blocks) < 2 {
		return nil
	}
	start := len(blocks) - limit
	if start < 1 {
		start = 1
	}
	out := make([]int64, 0, len(blocks)-start)
	for i := start; i < len(blocks); i++ {
		interval := blocks[i].Timestamp - blocks[i-1].Timestamp
		if interval < 0 {
			interval = 0
		}
		out = append(out, interval)
	}
	return out
}

func intervalStats(intervals []int64) (float64, int64, int64) {
	if len(intervals) == 0 {
		return 0, 0, 0
	}
	minInterval, maxInterval := intervals[0], intervals[0]
	var total int64
	for _, interval := range intervals {
		total += interval
		if interval < minInterval {
			minInterval = interval
		}
		if interval > maxInterval {
			maxInterval = interval
		}
	}
	return float64(total) / float64(len(intervals)), minInterval, maxInterval
}

func recentDifficulties(blocks []types.Block, limit int) []uint32 {
	if limit <= 0 {
		limit = 30
	}
	start := len(blocks) - limit
	if start < 0 {
		start = 0
	}
	out := make([]uint32, 0, len(blocks)-start)
	for _, block := range blocks[start:] {
		out = append(out, block.Difficulty)
	}
	return out
}

func projectedRetargetDirection(current, next uint32, avg float64, target int64) string {
	if next > current {
		return "up"
	}
	if next < current {
		return "down"
	}
	if avg <= 0 || target <= 0 {
		return "unchanged"
	}
	if avg < float64(target) {
		return "up"
	}
	if avg > float64(target) {
		return "down"
	}
	return "unchanged"
}

func recentMiningBlocks(blocks []types.Block, limit int) []map[string]any {
	if limit <= 0 {
		limit = 30
	}
	start := len(blocks) - limit
	if start < 0 {
		start = 0
	}
	out := make([]map[string]any, 0, len(blocks)-start)
	for i := len(blocks) - 1; i >= start; i-- {
		block := blocks[i]
		interval := int64(0)
		if i > 0 {
			interval = block.Timestamp - blocks[i-1].Timestamp
			if interval < 0 {
				interval = 0
			}
		}
		out = append(out, map[string]any{
			"height":           block.Height,
			"hash":             block.Hash,
			"previous_hash":    block.PreviousHash,
			"timestamp":        block.Timestamp,
			"interval_seconds": interval,
			"difficulty":       block.Difficulty,
			"tx_count":         len(block.Transactions),
			"miner_address":    block.MinerAddress,
		})
	}
	return out
}

func stakingSummaryMap(blocks []types.Block, net config.NetworkConfig) map[string]any {
	params := net.Consensus.Staking
	state, err := staking.Replay(blocks, params)
	if err != nil {
		return map[string]any{}
	}
	summary := state.Summary(blocksHeight(blocks))
	return map[string]any{
		"staking_enabled":       params.Enabled,
		"min_stake_amount":      amount.Format(params.MinStakeAmount) + " " + config.Ticker,
		"min_service_stake":     amount.Format(params.MinServiceStake) + " " + config.Ticker,
		"unbonding_period":      params.UnbondingPeriodBlocks,
		"total_active_stake":    amount.Format(summary.TotalActiveStake) + " " + config.Ticker,
		"total_unlocking_stake": amount.Format(summary.TotalUnlockingStake) + " " + config.Ticker,
		"active_stake_count":    summary.ActiveStakeCount,
	}
}

func balanceDetailsMap(details ledger.BalanceDetails, profile config.NetworkConfig) map[string]any {
	ticker := config.Ticker
	if profile.TxVersion >= types.TxVersionAsset {
		ticker = profile.Asset.NativeAssetSymbol
	}
	total, err := arith.Add(details.Confirmed, details.PendingIncoming)
	if err != nil {
		total = ^uint64(0)
	}
	return map[string]any{
		"address":            details.Address,
		"balance":            amount.Format(details.Confirmed),
		"ticker":             ticker,
		"confirmed_balance":  amount.Format(details.Confirmed),
		"mature_balance":     amount.Format(details.Mature),
		"immature_balance":   amount.Format(details.Immature),
		"spendable_balance":  amount.Format(details.Spendable),
		"active_stake":       amount.Format(details.ActiveStake),
		"unlocking_stake":    amount.Format(details.UnlockingStake),
		"released_stake":     amount.Format(details.ReleasedStake),
		"pending_stake_lock": amount.Format(details.PendingStakeLock),
		"pending_outgoing":   amount.Format(details.PendingOutgoing),
		"pending_incoming":   amount.Format(details.PendingIncoming),
		"total_balance":      amount.Format(total),
		"coinbase_maturity":  details.CoinbaseMaturity,
		"current_height":     details.CurrentHeight,
	}
}

func (h handler) chainLocator(w http.ResponseWriter, _ *http.Request) {
	locator, err := p2p.LocalLocator(h.paths)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, locator)
}

func (h handler) chainCommonAncestor(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		Peer string `json:"peer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	_, ancestor, err := p2p.CommonAncestorWithPeer(h.paths, req.Peer)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ancestor)
}

func (h handler) chainBlocks(w http.ResponseWriter, _ *http.Request) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, blocks)
}

func (h handler) forkCheck(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		Peer string `json:"peer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	result, err := p2p.CheckForkWithProfile(h.paths, req.Peer, h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h handler) forkInspectDatadir(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		OtherDataDir string `json:"other_datadir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	if req.OtherDataDir == "" {
		writeError(w, errors.New("other_datadir is required"))
		return
	}
	result, err := p2p.InspectDatadirFork(h.paths, config.NewPaths(req.OtherDataDir))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h handler) chainState(w http.ResponseWriter, _ *http.Request) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	defer closeFn()
	result, err := bc.StateStatusWithNetwork(h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h handler) chainStateValidate(w http.ResponseWriter, _ *http.Request) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	defer closeFn()
	result, err := bc.ValidateStateWithNetwork(h.profile())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h handler) chainValidate(w http.ResponseWriter, _ *http.Request) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := chain.ValidateChainWithNetwork(blocks, h.profile())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h handler) balance(w http.ResponseWriter, r *http.Request) {
	address := strings.TrimPrefix(r.URL.Path, "/balance/")
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		writeError(w, err)
		return
	}
	bc, closeFn, err := h.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	defer closeFn()
	pending, err := mempool.New(h.paths.Mempool).Load()
	if err != nil {
		writeError(w, err)
		return
	}
	details, err := bc.BalanceDetailsForWithProfile(address, pending, h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, balanceDetailsMap(details, h.profile()))
}

func (h handler) feePolicy(w http.ResponseWriter, _ *http.Request) {
	p := h.profile()
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":        p.Fee.Enabled,
		"fee_asset_id":   config.FeeAssetID,
		"fee_asset":      config.NativeAssetID,
		"gas_price":      p.Fee.MinGasPrice,
		"min_fee":        p.Fee.MinFee,
		"bytes_per_gas":  p.Fee.BytesPerGas,
		"max_gas_per_tx": p.Fee.MaxGasPerTx,
		"base_gas": map[string]uint64{
			"transfer_idr":   p.Fee.BaseGasTransfer,
			"transfer_token": p.Fee.BaseGasAssetTransfer,
			"stake_lock":     p.Fee.BaseGasStakeLock,
			"stake_unlock":   p.Fee.BaseGasStakeUnlock,
			"asset_create":   p.Fee.BaseGasAssetCreate,
			"asset_mint":     p.Fee.BaseGasAssetMint,
			"asset_burn":     p.Fee.BaseGasBurn,
		},
		"paymaster_enabled": p.Asset.PaymasterEnabled,
	})
}

func (h handler) feePool(w http.ResponseWriter, _ *http.Request) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	defer closeFn()
	result, err := chain.FeePoolBalanceWithProfile(bc, h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"asset_id": result.AssetID,
		"symbol":   result.Symbol,
		"pool":     result.Pool,
		"amount":   amount.FormatUnits(result.Amount, 0),
		"units":    result.Amount,
	})
}

func (h handler) feeEstimate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var tx types.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		writeError(w, err)
		return
	}
	quote, err := chain.EstimateFee(tx, h.profile())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "quote": quote})
}

func (h handler) assetInfo(w http.ResponseWriter, r *http.Request) {
	assetID := strings.TrimSpace(r.URL.Query().Get("asset_id"))
	if assetID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "asset_id is required"})
		return
	}
	bc, closeFn, err := h.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	defer closeFn()
	def, found, err := bc.AssetDefinitionWithProfile(assetID, h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "asset not found", "asset_id": assetID})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"asset":     def,
		"native":    def.ID == config.NativeAssetID,
		"fee_asset": def.ID == config.FeeAssetID,
		"network":   h.profile().Name,
	})
}

func (h handler) assetBalances(w http.ResponseWriter, r *http.Request) {
	address := strings.TrimSpace(r.URL.Query().Get("address"))
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		writeError(w, err)
		return
	}
	bc, closeFn, err := h.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	defer closeFn()
	entries, err := bc.AssetBalancesForAddressWithProfile(address, h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		def, found, err := bc.AssetDefinitionWithProfile(entry.AssetID, h.profile())
		if err != nil {
			writeError(w, err)
			return
		}
		if !found {
			continue
		}
		out = append(out, map[string]any{
			"asset_id": entry.AssetID,
			"symbol":   def.Symbol,
			"decimals": def.Decimals,
			"amount":   amount.FormatUnits(entry.Amount, def.Decimals),
			"units":    entry.Amount,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"address":  address,
		"balances": out,
	})
}

func (h handler) assetBalance(w http.ResponseWriter, r *http.Request) {
	address := strings.TrimSpace(r.URL.Query().Get("address"))
	assetID := strings.TrimSpace(r.URL.Query().Get("asset_id"))
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		writeError(w, err)
		return
	}
	if assetID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "asset_id is required"})
		return
	}
	bc, closeFn, err := h.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	defer closeFn()
	def, found, err := bc.AssetDefinitionWithProfile(assetID, h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "asset not found", "asset_id": assetID})
		return
	}
	balance, err := bc.AssetBalanceWithProfile(address, assetID, h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"address":  address,
		"asset_id": assetID,
		"symbol":   def.Symbol,
		"decimals": def.Decimals,
		"amount":   amount.FormatUnits(balance, def.Decimals),
		"units":    balance,
	})
}

func (h handler) address(w http.ResponseWriter, r *http.Request) {
	info, err := h.inspectAddress(strings.TrimPrefix(r.URL.Path, "/address/"))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"address": strings.TrimPrefix(r.URL.Path, "/address/"), "valid": false})
		return
	}
	ticker := config.Ticker
	if h.profile().TxVersion >= types.TxVersionAsset {
		ticker = h.profile().Asset.NativeAssetSymbol
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"address":                  info.address,
		"valid":                    true,
		"format":                   addressFormat(info.address),
		"network":                  h.profile().Name,
		"key_curve":                "secp256k1",
		"legacy":                   crypto.IsLegacyDevAddress(info.address),
		"confirmed_balance":        amount.Format(info.confirmedBalance) + " " + ticker,
		"mature_balance":           amount.Format(info.matureBalance) + " " + ticker,
		"immature_balance":         amount.Format(info.immatureBalance) + " " + ticker,
		"spendable_balance":         amount.Format(info.spendableBalance) + " " + ticker,
		"confirmed_nonce":           info.confirmedNonce,
		"pending_outgoing_count":    info.pendingOutgoingCount,
		"pending_outgoing_amount":   amount.Format(info.pendingOutgoingAmount) + " " + ticker,
		"pending_incoming_count":    info.pendingIncomingCount,
		"pending_incoming_amount":   amount.Format(info.pendingIncomingAmount) + " " + ticker,
	})
}

func (h handler) tx(w http.ResponseWriter, r *http.Request) {
	txID := strings.TrimPrefix(r.URL.Path, "/tx/")
	result, ok, err := h.findTransaction(txID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tx not found"})
		return
	}
	response := map[string]any{
		"id":       result.tx.ID,
		"status":   result.status,
		"from":     result.tx.From,
		"to":       result.tx.To,
		"nonce":    result.tx.Nonce,
		"coinbase": result.tx.Coinbase,
	}
	if result.tx.ProtocolVersion() >= types.TxVersionAsset {
		response["asset_id"] = result.tx.EffectiveAssetID()
		response["amount_units"] = result.tx.Amount
		response["fee"] = amount.FormatUnits(result.tx.Fee, h.profile().Asset.NativeAssetDecimals) + " " + h.profile().Asset.FeeAssetID
		response["fee_units"] = result.tx.Fee
	} else {
		response["amount"] = amount.Format(result.tx.Amount) + " " + config.Ticker
		response["fee"] = amount.Format(result.tx.Fee) + " " + config.Ticker
	}
	if result.status == "confirmed" {
		response["block_height"] = result.blockHeight
	}
	writeJSON(w, http.StatusOK, response)
}

func (h handler) mempoolList(w http.ResponseWriter, r *http.Request) {
	txs, err := mempool.New(h.paths.Mempool).Load()
	if err != nil {
		writeError(w, err)
		return
	}
	detail := r.URL.Query().Get("detail") == "true"
	views := make([]map[string]any, 0, len(txs))
	for _, tx := range txs {
		v := txView(tx, "pending", h.profile())
		if !detail {
			delete(v, "timestamp")
		}
		views = append(views, v)
	}
	writeJSON(w, http.StatusOK, map[string]any{"pending_tx_count": len(txs), "transactions": views})
}

func (h handler) mempoolClear(w http.ResponseWriter, _ *http.Request) {
	if err := mempool.New(h.paths.Mempool).Clear(); err != nil {
		writeError(w, err)
		return
	}
	h.refreshState()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (h handler) faucetInfo(w http.ResponseWriter, _ *http.Request) {
	pending, _ := mempool.New(h.paths.Mempool).Load()
	net := h.profile()
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":              h.faucetEnabled(),
		"network":              net.Name,
		"network_id":           net.NetworkID,
		"chain_id":             net.ChainID,
		"faucet_address":       h.info.FaucetAddress,
		"amount":               amount.Format(h.faucetAmount()),
		"max_per_address":      amount.Format(h.faucetMaxPerAddress()),
		"min_interval_seconds": int64(h.faucetMinInterval().Seconds()),
		"mempool_pending":      len(pending),
		"note":                 "testnet faucet only; testnet IDR has no monetary value",
	})
}

func (h handler) faucetRequest(w http.ResponseWriter, r *http.Request) {
	unlock := h.lockTxSubmission()
	defer unlock()
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		Address string `json:"address"`
		Amount  string `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.createFaucetTransaction(req.Address, req.Amount, time.Now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h handler) createFaucetTransaction(recipient, amountText string, now time.Time) (map[string]any, error) {
	if !h.info.EnableFaucetRPC {
		return nil, errors.New("faucet disabled")
	}
	if ok, reason := h.writeGuard(); !ok {
		return nil, errors.New(reason)
	}
	if h.profile().Name != "testnet" {
		return nil, errors.New("faucet is testnet-only")
	}
	from := strings.TrimSpace(h.info.FaucetAddress)
	if from == "" {
		return nil, errors.New("faucet address not configured")
	}
	if err := crypto.ValidateAddressForNetwork(from, h.profile()); err != nil {
		return nil, fmt.Errorf("invalid faucet address: %w", err)
	}
	if err := crypto.ValidateAddressForNetwork(recipient, h.profile()); err != nil {
		return nil, errors.New("invalid address")
	}
	if amountText == "" {
		amountText = amount.Format(h.faucetAmount())
	}
	txAmount, err := amount.Parse(amountText)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	if txAmount == 0 {
		return nil, errors.New("invalid amount: must be greater than 0")
	}
	if txAmount > h.faucetAmount() {
		return nil, errors.New("invalid amount: exceeds faucet max per request")
	}
	maxPerAddress := h.faucetMaxPerAddress()
	if maxPerAddress > 0 && txAmount > maxPerAddress {
		return nil, errors.New("invalid amount: exceeds faucet max per address")
	}
	store := faucet.NewStore(h.paths.FaucetState)
	state, err := store.Load()
	if err != nil {
		return nil, err
	}
	entry := state.Requests[recipient]
	if entry.LastRequestTime != "" {
		last, err := time.Parse(time.RFC3339, entry.LastRequestTime)
		if err == nil && now.UTC().Sub(last) < h.faucetMinInterval() {
			return nil, errors.New("recipient rate limited")
		}
	}
	day := now.UTC().Format("2006-01-02")
	dailyAmount := entry.DailyAmountRequested
	if entry.Day != day {
		dailyAmount = 0
	}
	if maxPerAddress > 0 && dailyAmount+txAmount > maxPerAddress {
		return nil, errors.New("recipient faucet daily limit exceeded")
	}
	pending, err := mempool.New(h.paths.Mempool).Load()
	if err != nil {
		return nil, err
	}
	for _, tx := range pending {
		if tx.From == from && tx.To == recipient {
			return nil, errors.New("pending faucet tx already exists for address")
		}
	}
	tx, err := h.createPendingTransaction(from, recipient, amount.Format(txAmount))
	if err != nil {
		if strings.Contains(err.Error(), "insufficient mature balance") {
			return nil, fmt.Errorf("insufficient mature faucet balance: %w", err)
		}
		return nil, err
	}
	if err := h.admitMempoolTx(tx); err != nil {
		if errors.Is(err, mempool.ErrDuplicateTx) {
			return nil, errors.New("pending faucet tx already exists for address")
		}
		return nil, err
	}
	if err := store.Record(recipient, tx.ID, tx.Amount, now); err != nil {
		_ = mempool.New(h.paths.Mempool).RemoveIDs(map[string]struct{}{tx.ID: {}})
		return nil, err
	}
	h.refreshState()
	return map[string]any{
		"tx_id":  tx.ID,
		"from":   tx.From,
		"to":     tx.To,
		"amount": amount.Format(tx.Amount),
		"status": "pending",
		"note":   "mine a block to confirm faucet transaction",
	}, nil
}

func (h handler) faucetEnabled() bool {
	return h.info.EnableFaucetRPC && h.profile().Name == "testnet" && h.info.FaucetAddress != ""
}

func (h handler) faucetAmount() uint64 {
	if h.info.FaucetAmount == 0 {
		return 100 * config.UnitsPerCoin
	}
	return h.info.FaucetAmount
}

func (h handler) faucetMaxPerAddress() uint64 {
	if h.info.FaucetMaxPerAddress == 0 {
		return 1000 * config.UnitsPerCoin
	}
	return h.info.FaucetMaxPerAddress
}

func (h handler) faucetMinInterval() time.Duration {
	if h.info.FaucetMinInterval <= 0 {
		return time.Minute
	}
	return h.info.FaucetMinInterval
}

func (h handler) walletNew(w http.ResponseWriter, _ *http.Request) {
	// TODO: require explicit admin/auth mode before exposing wallet RPC on public nodes.
	store := wallet.NewStore(h.paths.Wallets)
	newWallet, err := wallet.NewWithProfile(h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	if err := store.Add(newWallet); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"address":   newWallet.Address,
		"format":    "base58check",
		"network":   h.profile().Name,
		"key_curve": "secp256k1",
	})
}

func (h handler) walletList(w http.ResponseWriter, _ *http.Request) {
	// TODO: require explicit admin/auth mode before exposing wallet RPC on public nodes.
	wallets, err := wallet.NewStore(h.paths.Wallets).Load()
	if err != nil {
		writeError(w, err)
		return
	}
	views := make([]map[string]any, 0, len(wallets))
	for _, wlt := range wallets {
		views = append(views, map[string]any{
			"address":   wlt.Address,
			"format":    addressFormat(wlt.Address),
			"network":   h.profile().Name,
			"key_curve": "secp256k1",
			"legacy":    crypto.IsLegacyDevAddress(wlt.Address),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"wallets": views, "count": len(views)})
}

func (h handler) send(w http.ResponseWriter, r *http.Request) {
	unlock := h.lockTxSubmission()
	defer unlock()
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Amount string `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	tx, err := h.createPendingTransaction(req.From, req.To, req.Amount)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.admitMempoolTx(tx); err != nil && !errors.Is(err, mempool.ErrDuplicateTx) {
		writeError(w, err)
		return
	}
	h.refreshState()
	peers, _ := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	broadcast := p2p.BroadcastTxToPeersWithProfile(h.paths, h.profile(), peers, tx)
	view := txView(tx, "pending", h.profile())
	view["status"] = "pending"
	view["broadcast"] = broadcast
	writeJSON(w, http.StatusOK, view)
}

func (h handler) minerTemplate(w http.ResponseWriter, r *http.Request) {
	if ok, reason, _, _ := h.miningTemplateGuard(); !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": reason})
		return
	}
	var address string
	switch r.Method {
	case http.MethodGet:
		address = r.URL.Query().Get("address")
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, maxTemplateBody)
		var req struct {
			Address string `json:"address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, err)
			return
		}
		address = req.Address
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		writeError(w, fmt.Errorf("invalid address: %w", err))
		return
	}
	block, selectedCount, err := h.blockTemplate(address)
	if err != nil {
		writeError(w, err)
		return
	}
	net := h.profile()
	writeJSON(w, http.StatusOK, map[string]any{
		"template_id":      blockTemplateID(block),
		"network":          net.Name,
		"network_id":       net.NetworkID,
		"chain_id":         net.ChainID,
		"protocol_version": net.ProtocolVersion,
		"height":           block.Height,
		"previous_hash":    block.PreviousHash,
		"difficulty":       block.Difficulty,
		"target":           difficultyTarget(block.Difficulty),
		"reward_address":   address,
		"coinbase_reward":  amount.Format(block.Transactions[0].Amount),
		"timestamp":        block.Timestamp,
		"transactions":     block.Transactions,
		"tx_count":         len(block.Transactions),
		"selected_txs":     selectedCount,
		"header": map[string]any{
			"height":        block.Height,
			"previous_hash": block.PreviousHash,
			"timestamp":     block.Timestamp,
			"difficulty":    block.Difficulty,
			"miner_address": block.MinerAddress,
			"merkle_root":   block.MerkleRoot,
		},
		"block": block,
	})
}

func (h handler) minerSubmit(w http.ResponseWriter, r *http.Request) {
	unlock := h.lockChainMutation()
	defer unlock()
	r.Body = http.MaxBytesReader(w, r.Body, maxBlockBody)
	var req struct {
		TemplateID string      `json:"template_id"`
		Block      types.Block `json:"block"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	block := req.Block
	if ok, reason, _, _ := h.miningTemplateGuard(); !ok {
		writeJSON(w, http.StatusOK, map[string]any{"accepted": false, "reason": reason})
		return
	}
	log.Printf("miner submit received height=%d hash=%s template=%s", block.Height, firstN(block.Hash, 16), firstN(req.TemplateID, 16))
	log.Printf("miner submit decoded height=%d txs=%d", block.Height, len(block.Transactions))
	bc, closeFn, err := h.openChain()
	if err != nil {
		writeError(w, err)
		return
	}
	chainClosed := false
	closeChain := func() {
		if !chainClosed {
			closeFn()
			chainClosed = true
		}
	}
	defer closeChain()
	blocks, err := bc.Blocks()
	if err != nil {
		closeChain()
		writeError(w, err)
		return
	}
	tip := blocks[len(blocks)-1]
	expectedDifficulty := chain.CalculateNextDifficultyWithParams(blocks, h.profile().Difficulty)
	log.Printf("miner submit validating height=%d difficulty=%d", block.Height, block.Difficulty)
	for _, known := range blocks {
		if known.Hash == block.Hash {
			log.Printf("miner submit rejected reason=%q duplicate=true", "duplicate block")
			closeChain()
			writeJSON(w, http.StatusOK, map[string]any{
				"accepted":       false,
				"reason":         "duplicate block",
				"duplicate":      true,
				"height":         known.Height,
				"hash":           known.Hash,
				"current_height": tip.Height,
				"current_tip":    tip.Hash,
			})
			return
		}
	}
	if block.PreviousHash != tip.Hash || block.Height != tip.Height+1 {
		log.Printf("miner submit rejected reason=%q", "stale template")
		closeChain()
		writeJSON(w, http.StatusOK, map[string]any{
			"accepted":         false,
			"reason":           "stale template",
			"current_height":   tip.Height,
			"current_tip":      tip.Hash,
			"expected_parent":  tip.Hash,
			"submitted_parent": block.PreviousHash,
			"height":           block.Height,
			"hash":             block.Hash,
		})
		return
	}
	if block.Difficulty != expectedDifficulty {
		reason := fmt.Sprintf("invalid difficulty expected %d got %d", expectedDifficulty, block.Difficulty)
		log.Printf("miner submit rejected reason=%q", reason)
		closeChain()
		writeJSON(w, http.StatusOK, map[string]any{
			"accepted":            false,
			"reason":              reason,
			"expected_difficulty": expectedDifficulty,
			"got_difficulty":      block.Difficulty,
			"current_height":      tip.Height,
			"current_tip":         tip.Hash,
			"expected_parent":     tip.Hash,
			"submitted_parent":    block.PreviousHash,
		})
		return
	}
	if err := validateMinerRewardAddresses(block, h.profile()); err != nil {
		reason := err.Error()
		log.Printf("miner submit rejected reason=%q", reason)
		closeChain()
		writeJSON(w, http.StatusOK, map[string]any{"accepted": false, "reason": reason})
		return
	}
	if !chain.ValidateProofOfWork(block) {
		log.Printf("miner submit rejected reason=%q", "invalid proof of work")
		closeChain()
		writeJSON(w, http.StatusOK, map[string]any{"accepted": false, "reason": "invalid proof of work"})
		return
	}
	if req.TemplateID != "" && req.TemplateID != blockTemplateID(block) {
		log.Printf("miner submit rejected reason=%q", "template id mismatch")
		closeChain()
		writeJSON(w, http.StatusOK, map[string]any{"accepted": false, "reason": "template id mismatch"})
		return
	}
	if err := bc.AddBlockWithNetwork(block, h.profile()); err != nil {
		log.Printf("miner submit rejected reason=%q", err.Error())
		closeChain()
		writeJSON(w, http.StatusOK, map[string]any{
			"accepted":         false,
			"reason":           err.Error(),
			"current_height":   tip.Height,
			"current_tip":      tip.Hash,
			"expected_parent":  tip.Hash,
			"submitted_parent": block.PreviousHash,
		})
		return
	}
	log.Printf("miner submit committed height=%d hash=%s", block.Height, block.Hash)
	included := make(map[string]struct{})
	for _, tx := range block.Transactions {
		if !tx.Coinbase {
			included[tx.ID] = struct{}{}
		}
	}
	if err := mempool.New(h.paths.Mempool).RemoveIDs(included); err != nil {
		closeChain()
		writeError(w, err)
		return
	}
	closeChain()
	h.refreshState()
	peers, _ := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	broadcast := p2p.BroadcastBlockToPeersWithProfile(h.paths, h.profile(), peers, block)
	log.Printf("miner submit broadcast complete height=%d success=%d failed=%d", block.Height, broadcast.Success, broadcast.Failed)
	h.scheduleUpstreamBackfill()
	writeJSON(w, http.StatusOK, map[string]any{
		"accepted":          true,
		"height":            block.Height,
		"hash":              block.Hash,
		"difficulty":        block.Difficulty,
		"tx_count":          len(block.Transactions),
		"broadcast_success": broadcast.Success,
		"broadcast_failed":  broadcast.Failed,
		"broadcast":         broadcast,
	})
}

func validateMinerRewardAddresses(block types.Block, profile config.NetworkConfig) error {
	if err := crypto.ValidateAddressForNetwork(block.MinerAddress, profile); err != nil {
		return fmt.Errorf("invalid miner address: %w", err)
	}
	for _, tx := range block.Transactions {
		if !tx.Coinbase {
			continue
		}
		if err := crypto.ValidateAddressForNetwork(tx.To, profile); err != nil {
			return fmt.Errorf("invalid coinbase recipient: %w", err)
		}
	}
	return nil
}

func (h handler) serviceRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address       string `json:"address"`
		Endpoint      string `json:"endpoint"`
		ClientVersion string `json:"client_version"`
		Platform      string `json:"platform"`
		UserAgent     string `json:"user_agent"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxServiceBody)).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	node, err := h.serviceStore().Register(req.Address, req.Endpoint, servicenode.Metadata{ClientVersion: req.ClientVersion, Platform: req.Platform, UserAgent: req.UserAgent})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service_node_id": node.ServiceNodeID, "owner_address": node.OwnerAddress, "status": node.Status})
}

func (h handler) serviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address       string `json:"address"`
		Endpoint      string `json:"endpoint"`
		ClientVersion string `json:"client_version"`
		Platform      string `json:"platform"`
		UserAgent     string `json:"user_agent"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxServiceBody)).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	score, node, err := h.serviceStore().Heartbeat(req.Address, req.Endpoint, servicenode.Metadata{ClientVersion: req.ClientVersion, Platform: req.Platform, UserAgent: req.UserAgent})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                true,
		"status":            node.Status,	...
// NOTE: canonical genesis reporting above uses GenesisHashForNetwork so launch-sensitive identity is never regenerated here.
