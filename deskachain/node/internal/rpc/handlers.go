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
	h := handler{paths: paths, info: info, limiter: newRateLimiter(), txMu: &sync.Mutex{}, chainMu: &sync.Mutex{}}
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
	net.GenesisHash = chain.GenesisBlockForNetwork(net).Hash
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
		"genesis_hash":        chain.GenesisBlockForNetwork(net).Hash,
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
		"genesis_hash":              chain.GenesisBlockForNetwork(net).Hash,
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
	if strings.Contains(path, "/") || path == "" {
		explorerError(w, http.StatusNotFound, "not_found", "explorer endpoint not found")
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

func (h handler) explorerAddressStakes(w http.ResponseWriter, _ *http.Request, address string) {
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		explorerError(w, http.StatusBadRequest, "invalid_address", err.Error())
		return
	}
	records, err := h.explorerStakeRecords(address)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "address": address, "stakes": records, "count": len(records)})
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
		GenesisHash:    chain.GenesisBlockForNetwork(profile).Hash,
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
		"genesis_hash":     chain.GenesisBlockForNetwork(profile).Hash,
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
		"genesis_hash":        chain.GenesisBlockForNetwork(profile).Hash,
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
		"genesis_hash":   chain.GenesisBlockForNetwork(profile).Hash,
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
		"genesis_hash":     chain.GenesisBlockForNetwork(profile).Hash,
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
		"genesis_hash":      chain.GenesisBlockForNetwork(profile).Hash,
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
		"genesis_hash": chain.GenesisBlockForNetwork(profile).Hash,
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
	net.GenesisHash = chain.GenesisBlockForNetwork(net).Hash
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

func (h handler) chainDifficulty(w http.ResponseWriter, _ *http.Request) {
	blocks, _, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, chainDifficultyMap(blocks, h.profile()))
}

func (h handler) miningStatus(w http.ResponseWriter, _ *http.Request) {
	blocks, pending, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	info := miningMetricsMap(blocks, h.profile(), len(pending), h.peerCount())
	h.addMiningGuardFields(info)
	writeJSON(w, http.StatusOK, info)
}

func (h handler) miningDifficulty(w http.ResponseWriter, _ *http.Request) {
	blocks, pending, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	info := miningMetricsMap(blocks, h.profile(), len(pending), h.peerCount())
	info["note"] = "difficulty observation is informational only; consensus rules are unchanged"
	h.addMiningGuardFields(info)
	writeJSON(w, http.StatusOK, info)
}

func (h handler) miningBlocks(w http.ResponseWriter, r *http.Request) {
	blocks, pending, err := h.chainInfoData()
	if err != nil {
		writeError(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	info := miningMetricsMap(blocks, h.profile(), len(pending), h.peerCount())
	h.addMiningGuardFields(info)
	info["limit"] = limit
	info["blocks"] = recentMiningBlocks(blocks, limit)
	writeJSON(w, http.StatusOK, info)
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
		"genesis_hash":              chain.GenesisBlockForNetwork(net).Hash,
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
		"min_difficulty":            params.MinDifficulty,
		"max_difficulty":            params.MaxDifficulty,
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
		"genesis_hash":                   chain.GenesisBlockForNetwork(net).Hash,
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
		"min_interval_seconds":           minInterval,
		"max_interval_seconds":           maxInterval,
		"recent_difficulties":            recentDifficulties(blocks, int(params.RetargetWindow)),
		"projected_retarget_direction":   projectedRetargetDirection(tip.Difficulty, nextDifficulty, avg, params.TargetBlockTimeSeconds),
		"total_supply":                   amount.Format(stats.TotalSupply) + " " + config.Ticker,
		"coinbase_maturity":              net.Consensus.CoinbaseMaturity,
		"pending_tx_count":               pendingCount,
		"peer_count":                     peerCount,
		"note":                           "testnet mining is for testing only; testnet IDR has no monetary value",
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
		writeJSON(w, http.StatusBadRequest, map[string]any{"valid": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"valid":        true,
		"height":       result.Height,
		"blocks":       result.Blocks,
		"total_supply": amount.Format(result.TotalSupply) + " " + config.Ticker,
	})
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
			"asset_burn":     p.Fee.BaseGasAssetBurn,
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
	address := strings.TrimPrefix(r.URL.Path, "/address/")
	info, err := h.inspectAddress(address)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"address": address, "valid": false})
		return
	}
	ticker := config.Ticker
	if h.profile().TxVersion >= types.TxVersionAsset {
		ticker = h.profile().Asset.NativeAssetSymbol
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"address":                 info.address,
		"valid":                   true,
		"format":                  addressFormat(info.address),
		"network":                 h.profile().Name,
		"key_curve":               "secp256k1",
		"legacy":                  crypto.IsLegacyDevAddress(info.address),
		"confirmed_balance":       amount.Format(info.confirmedBalance) + " " + ticker,
		"mature_balance":          amount.Format(info.matureBalance) + " " + ticker,
		"immature_balance":        amount.Format(info.immatureBalance) + " " + ticker,
		"spendable_balance":       amount.Format(info.spendableBalance) + " " + ticker,
		"confirmed_nonce":         info.confirmedNonce,
		"pending_outgoing_count":  info.pendingOutgoingCount,
		"pending_outgoing_amount": amount.Format(info.pendingOutgoingAmount) + " " + ticker,
		"pending_incoming_count":  info.pendingIncomingCount,
		"pending_incoming_amount": amount.Format(info.pendingIncomingAmount) + " " + ticker,
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
	broadcast := p2p.BroadcastTxToPeers(h.paths.Peers, peers, tx)
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
	// Do not broadcast while holding runtime/chain locks or an open chain store.
	peers, _ := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
	broadcast := p2p.BroadcastBlockToPeers(h.paths.Peers, peers, block)
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
		"status":            node.Status,
		"uptime_score":      score.UptimeScore,
		"latency_score":     score.LatencyScore,
		"bandwidth_score":   score.BandwidthScore,
		"reliability_score": score.ReliabilityScore,
		"abuse_penalty":     score.AbusePenalty,
		"service_score":     score.ServiceScore,
		"flags":             score.Flags,
		"note":              score.Note,
	})
}

func (h handler) serviceChallengeCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxServiceBody)).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	challenge, err := h.serviceStore().CreateChallenge(req.Address)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"challenge_id": challenge.ChallengeID,
		"address":      challenge.Address,
		"issued_at":    challenge.IssuedAt,
		"expires_at":   challenge.ExpiresAt,
		"nonce":        challenge.Nonce,
		"status":       challenge.Status,
	})
}

func (h handler) serviceChallengeSubmit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChallengeID string `json:"challenge_id"`
		LatencyMS   int64  `json:"latency_ms"`
		BytesUp     int64  `json:"bytes_up"`
		BytesDown   int64  `json:"bytes_down"`
		Success     bool   `json:"success"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxServiceBody)).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	score, challenge, err := h.serviceStore().SubmitChallenge(req.ChallengeID, req.LatencyMS, req.BytesUp, req.BytesDown, req.Success)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                true,
		"challenge_id":      challenge.ChallengeID,
		"status":            challenge.Status,
		"uptime_score":      score.UptimeScore,
		"latency_score":     score.LatencyScore,
		"bandwidth_score":   score.BandwidthScore,
		"reliability_score": score.ReliabilityScore,
		"abuse_penalty":     score.AbusePenalty,
		"service_score":     score.ServiceScore,
		"flags":             score.Flags,
		"note":              score.Note,
	})
}

func (h handler) serviceScore(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		writeError(w, errors.New("address is required"))
		return
	}
	score, err := h.serviceStore().Score(address)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "score": score})
}

func (h handler) serviceRewards(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		writeError(w, errors.New("address is required"))
		return
	}
	rewards, err := h.serviceStore().Rewards(address)
	if err != nil {
		writeError(w, err)
		return
	}
	score, _ := h.serviceStore().Score(address)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "address": address, "rewards": rewards, "note": score.Note})
}

func (h handler) serviceList(w http.ResponseWriter, _ *http.Request) {
	nodes, err := h.serviceStore().LoadNodes()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "nodes": nodes, "count": len(nodes)})
}

func (h handler) stakeInfo(w http.ResponseWriter, _ *http.Request) {
	blocks, err := h.blocks()
	if err != nil {
		writeError(w, err)
		return
	}
	params := h.profile().Consensus.Staking
	state, err := staking.Replay(blocks, params)
	if err != nil {
		writeError(w, err)
		return
	}
	summary := state.Summary(blocksHeight(blocks))
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                    true,
		"staking_enabled":       params.Enabled,
		"min_stake_amount":      amount.Format(params.MinStakeAmount),
		"min_service_stake":     amount.Format(params.MinServiceStake),
		"unbonding_period":      params.UnbondingPeriodBlocks,
		"total_active_stake":    amount.Format(summary.TotalActiveStake),
		"total_unlocking_stake": amount.Format(summary.TotalUnlockingStake),
		"active_stake_count":    summary.ActiveStakeCount,
	})
}

func (h handler) stakeList(w http.ResponseWriter, r *http.Request) {
	blocks, err := h.blocks()
	if err != nil {
		writeError(w, err)
		return
	}
	state, err := staking.Replay(blocks, h.profile().Consensus.Staking)
	if err != nil {
		writeError(w, err)
		return
	}
	filter := r.URL.Query().Get("address")
	records := make([]map[string]any, 0)
	for _, record := range state.Records(blocksHeight(blocks)) {
		if filter != "" && record.OwnerAddress != filter {
			continue
		}
		records = append(records, stakeRecordMap(record))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "stakes": records, "count": len(records)})
}

func (h handler) stakeStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, errors.New("stake id is required"))
		return
	}
	blocks, err := h.blocks()
	if err != nil {
		writeError(w, err)
		return
	}
	state, err := staking.Replay(blocks, h.profile().Consensus.Staking)
	if err != nil {
		writeError(w, err)
		return
	}
	record, ok := state.Find(id, blocksHeight(blocks))
	if !ok {
		writeError(w, errors.New("stake not found"))
		return
	}
	resp := stakeRecordMap(record)
	resp["ok"] = true
	writeJSON(w, http.StatusOK, resp)
}

func (h handler) stakeLock(w http.ResponseWriter, r *http.Request) {
	unlock := h.lockTxSubmission()
	defer unlock()
	var req struct {
		Address string `json:"address"`
		Amount  string `json:"amount"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody)).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	tx, err := h.createStakeLockTx(req.Address, req.Amount)
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
	broadcast := p2p.BroadcastTxToPeers(h.paths.Peers, peers, tx)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"tx_id":     tx.ID,
		"stake_id":  tx.StakeID,
		"address":   tx.From,
		"amount":    amount.Format(tx.Amount),
		"status":    "pending",
		"broadcast": broadcast,
	})
}

func (h handler) stakeUnlock(w http.ResponseWriter, r *http.Request) {
	unlock := h.lockTxSubmission()
	defer unlock()
	var req struct {
		Address string `json:"address"`
		StakeID string `json:"stake_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody)).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	tx, releaseHeight, err := h.createStakeUnlockTx(req.Address, req.StakeID)
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
	broadcast := p2p.BroadcastTxToPeers(h.paths.Peers, peers, tx)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":             true,
		"tx_id":          tx.ID,
		"stake_id":       tx.StakeID,
		"release_height": releaseHeight,
		"status":         "pending",
		"broadcast":      broadcast,
	})
}

func (h handler) mine(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	var req struct {
		Address     string `json:"address"`
		Blocks      uint64 `json:"blocks"`
		MaxNonce    uint64 `json:"max_nonce"`
		MineVerbose bool   `json:"mine_verbose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}
	if req.Blocks == 0 {
		req.Blocks = 1
	}
	if err := crypto.ValidateAddressForNetwork(req.Address, h.profile()); err != nil {
		writeError(w, err)
		return
	}
	job, err := h.info.Mining.Start(req.Address, int(req.Blocks))
	if err != nil {
		writeError(w, err)
		return
	}
	started := time.Now()
	log.Printf("mining started job=%s miner=%s blocks=%d", job.ID, req.Address, req.Blocks)
	defer func() {
		if recovered := recover(); recovered != nil {
			err := fmt.Errorf("panic: %v", recovered)
			h.info.Mining.Fail(err)
			log.Printf("mining failed job=%s error=%v", job.ID, err)
			writeError(w, err)
		}
	}()
	var mined []map[string]any
	broadcast := p2p.BroadcastSummary{}
	for i := uint64(0); i < req.Blocks; i++ {
		pending, err := mempool.New(h.paths.Mempool).Load()
		if err != nil {
			h.info.Mining.Fail(err)
			log.Printf("mining failed job=%s error=%v", job.ID, err)
			writeError(w, err)
			return
		}
		targetHeight, difficulty, err := h.nextMiningTarget()
		if err != nil {
			h.info.Mining.Fail(err)
			log.Printf("mining failed job=%s error=%v", job.ID, err)
			writeError(w, err)
			return
		}
		log.Printf("mining block started target_height=%d difficulty=%d pending_txs=%d", targetHeight, difficulty, len(pending))
		blockStart := time.Now()
		block, err := h.mineCandidateBlock(r.Context(), req.Address, pending, req.MaxNonce, req.MineVerbose)
		if err != nil {
			if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "cancel") {
				h.info.Mining.Cancel(err)
			} else {
				h.info.Mining.Fail(err)
			}
			log.Printf("mining failed job=%s error=%v", job.ID, err)
			writeError(w, err)
			return
		}
		log.Printf("mining block found height=%d hash=%s nonce=%d elapsed=%s", block.Height, block.Hash, block.Nonce, time.Since(blockStart))
		if err := h.commitMinedBlock(block); err != nil {
			h.info.Mining.Fail(err)
			log.Printf("mining failed job=%s error=%v", job.ID, err)
			writeError(w, err)
			return
		}
		h.refreshState()
		h.info.Mining.RecordBlock(block.Height, block.Hash)
		log.Printf("mining block committed height=%d hash=%s", block.Height, block.Hash)
		// Do not hold chain or mempool locks while performing network I/O.
		peers, _ := p2p.NewPeerStore(h.paths.Peers).LoadMetadata()
		blockBroadcast := p2p.BroadcastBlockToPeers(h.paths.Peers, peers, block)
		broadcast.Peers += blockBroadcast.Peers
		broadcast.Success += blockBroadcast.Success
		broadcast.Failed += blockBroadcast.Failed
		broadcast.Results = append(broadcast.Results, blockBroadcast.Results...)
		broadcast.Errors = append(broadcast.Errors, blockBroadcast.Errors...)
		log.Printf("mining block broadcast complete height=%d success=%d failed=%d", block.Height, blockBroadcast.Success, blockBroadcast.Failed)
		reward := uint64(0)
		if len(block.Transactions) > 0 && block.Transactions[0].Coinbase {
			reward = block.Transactions[0].Amount
		}
		mined = append(mined, map[string]any{
			"height":     block.Height,
			"hash":       block.Hash,
			"txs":        len(block.Transactions),
			"reward":     amount.Format(reward) + " " + config.Ticker,
			"difficulty": block.Difficulty,
			"nonce":      block.Nonce,
		})
	}
	h.info.Mining.Complete()
	log.Printf("mining completed job=%s mined_blocks=%d elapsed=%s", job.ID, len(mined), time.Since(started))
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
	pending, _ := mempool.New(h.paths.Mempool).Load()
	details, err := bc.BalanceDetailsForWithProfile(req.Address, pending, h.profile())
	if err != nil {
		writeError(w, err)
		return
	}
	net := h.profile()
	writeJSON(w, http.StatusOK, map[string]any{
		"network":                 net.Name,
		"network_id":              net.NetworkID,
		"chain_id":                net.ChainID,
		"protocol_version":        net.ProtocolVersion,
		"mined_blocks":            len(mined),
		"new_height":              tip.Height,
		"miner_balance":           amount.Format(details.Confirmed) + " " + config.Ticker,
		"miner_mature_balance":    amount.Format(details.Mature) + " " + config.Ticker,
		"miner_immature_balance":  amount.Format(details.Immature) + " " + config.Ticker,
		"miner_spendable_balance": amount.Format(details.Spendable) + " " + config.Ticker,
		"blocks":                  mined,
		"mined":                   mined,
		"broadcast":               broadcast,
	})
}

func (h handler) nextMiningTarget() (uint64, uint32, error) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		return 0, 0, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return 0, 0, err
	}
	tip := blocks[len(blocks)-1]
	return tip.Height + 1, chain.CalculateNextDifficultyWithParams(blocks, h.profile().Difficulty), nil
}

func (h handler) blockTemplate(miner string) (types.Block, int, error) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		return types.Block{}, 0, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return types.Block{}, 0, err
	}
	if len(blocks) == 0 {
		return types.Block{}, 0, errors.New("chain is not initialized")
	}
	pending, err := mempool.New(h.paths.Mempool).Load()
	if err != nil {
		return types.Block{}, 0, err
	}
	baseLedger, err := ledger.ReplayMatureWithProfile(blocks, h.profile().Consensus, h.profile())
	if err != nil {
		return types.Block{}, 0, err
	}
	workLedger := baseLedger.Clone()
	tip := blocks[len(blocks)-1]
	height, err := arith.Add(tip.Height, 1)
	if err != nil {
		return types.Block{}, 0, errors.New("block height overflow")
	}
	validPending := make([]types.Transaction, 0, len(pending))
	totalFees := uint64(0)
	for _, tx := range pending {
		if tx.Coinbase {
			continue
		}
		if err := workLedger.ApplyTransactionAtHeight(tx, height); err != nil {
			continue
		}
		validPending = append(validPending, tx)
		nextFees, feeErr := arith.Add(totalFees, tx.Fee)
		if feeErr != nil {
			return types.Block{}, 0, errors.New("transaction fees overflow")
		}
		totalFees = nextFees
	}
	reward := h.profile().Economic.BlockSubsidy
	if h.profile().TxVersion < types.TxVersionAsset {
		reward, err = arith.Add(config.InitialBlockReward, totalFees)
		if err != nil {
			return types.Block{}, 0, errors.New("block reward overflow")
		}
	}
	coinbase := types.NewCoinbaseTransactionWithVersion(miner, reward, height, h.profile().TxVersion)
	if coinbase.ProtocolVersion() >= types.TxVersionCanonical {
		if err := coinbase.RefreshIDForChainID(h.profile().ChainID); err != nil {
			return types.Block{}, 0, err
		}
	}
	txs := append([]types.Transaction{coinbase}, validPending...)
	block := types.NewBlockWithVersion(height, tip.Hash, miner, chain.CalculateNextDifficultyWithParams(blocks, h.profile().Difficulty), txs, h.profile().BlockVersion)
	if block.ProtocolVersion() == types.BlockVersionCanonical {
		candidate := baseLedger.Clone()
		if err := candidate.ApplyBlock(block); err != nil {
			return types.Block{}, 0, err
		}
		root, err := state.RootForLedger(candidate)
		if err != nil {
			return types.Block{}, 0, err
		}
		block.StateRoot = root
	}
	return block, len(validPending), nil
}

func (h handler) mineCandidateBlock(ctx context.Context, miner string, pending []types.Transaction, maxNonce uint64, verbose bool) (types.Block, error) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		return types.Block{}, err
	}
	defer closeFn()
	return bc.MineBlockWithContextAndNetwork(ctx, miner, pending, chain.MineOptions{
		MaxNonce: maxNonce,
		Verbose:  verbose,
		OnProgress: func(nonce uint64, hash string) {
			log.Printf("mining progress miner=%s nonce=%d hash_prefix=%s", miner, nonce, firstN(hash, 8))
		},
	}, h.profile())
}

func blockTemplateID(block types.Block) string {
	return crypto.DoubleSHA256Hex([]byte(fmt.Sprintf("%d|%s|%d|%d|%s|%s", block.Height, block.PreviousHash, block.Timestamp, block.Difficulty, block.MinerAddress, block.MerkleRoot)))
}

func difficultyTarget(difficulty uint32) string {
	zeros := strings.Repeat("0", int(difficulty))
	if len(zeros) >= 64 {
		return zeros[:64]
	}
	return zeros + strings.Repeat("f", 64-len(zeros))
}

func (h handler) commitMinedBlock(block types.Block) error {
	// Mining can run for a long time while sync/reorg operations continue.
	// Serialize only the final canonical-chain mutation so a stale candidate
	// fails cleanly instead of racing another chain writer.
	unlock := h.lockChainMutation()
	defer unlock()
	bc, closeFn, err := h.openChain()
	if err != nil {
		return err
	}
	defer closeFn()
	if err := bc.AddBlockWithNetwork(block, h.profile()); err != nil {
		return err
	}
	included := make(map[string]struct{})
	for _, tx := range block.Transactions {
		if tx.Coinbase {
			continue
		}
		included[tx.ID] = struct{}{}
	}
	return mempool.New(h.paths.Mempool).RemoveIDs(included)
}

func firstN(value string, n int) string {
	if len(value) <= n {
		return value
	}
	return value[:n]
}

func (h handler) lockChainMutation() func() {
	if h.chainMu == nil {
		return func() {}
	}
	h.chainMu.Lock()
	return h.chainMu.Unlock
}

func (h handler) lockTxSubmission() func() {
	if h.txMu == nil {
		return func() {}
	}
	h.txMu.Lock()
	return h.txMu.Unlock
}

func (h handler) admitMempoolTx(tx types.Transaction) error {
	policy := mempool.AdmissionPolicy{
		Profile: h.profile(),
		MaxTxs:  h.profile().Consensus.MaxTxCount,
		MaxGas:  h.profile().Consensus.MaxGasPerBlock,
	}
	return mempool.New(h.paths.Mempool).Admit(tx, policy)
}

func (h handler) refreshState() {
	if h.info.State == nil {
		return
	}
	_ = h.info.State.Refresh(h.paths)
}

func (h handler) debugChainTip() (uint64, string) {
	if h.info.State != nil {
		snapshot := h.info.State.Snapshot()
		return snapshot.Height, snapshot.TipHash
	}
	height, _ := h.currentHeight()
	bc, closeFn, err := h.openChain()
	if err != nil {
		return height, ""
	}
	defer closeFn()
	tip, err := bc.Tip()
	if err != nil {
		return height, ""
	}
	return tip.Height, tip.Hash
}

func (h handler) createPendingTransaction(from, to, amountText string) (types.Transaction, error) {
	if err := crypto.ValidateAddressForNetwork(from, h.profile()); err != nil {
		return types.Transaction{}, errors.New("invalid sender address")
	}
	if err := crypto.ValidateAddressForNetwork(to, h.profile()); err != nil {
		return types.Transaction{}, errors.New("invalid recipient address")
	}
	if from == to {
		return types.Transaction{}, errors.New("sender and recipient must differ")
	}
	txAmount, err := amount.Parse(amountText)
	if err != nil {
		return types.Transaction{}, err
	}
	bc, closeFn, err := h.openChain()
	if err != nil {
		return types.Transaction{}, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return types.Transaction{}, err
	}
	store := wallet.NewStore(h.paths.Wallets)
	fromWallet, ok, err := store.Find(from)
	if err != nil {
		return types.Transaction{}, err
	}
	if !ok {
		return types.Transaction{}, errors.New("local wallet not found for sender")
	}
	pending, err := mempool.New(h.paths.Mempool).Load()
	if err != nil {
		return types.Transaction{}, err
	}
	matureLedger, err := ledger.ReplayMatureWithProfile(blocks, h.profile().Consensus, h.profile())
	if err != nil {
		return types.Transaction{}, err
	}
	nonceBase, err := arith.Add(matureLedger.Nonce(from), pendingFromCount(pending, from))
	if err != nil {
		return types.Transaction{}, errors.New("account nonce overflow")
	}
	nonce, err := arith.Add(nonceBase, 1)
	if err != nil {
		return types.Transaction{}, errors.New("account nonce overflow")
	}
	var tx types.Transaction
	if h.profile().TxVersion >= types.TxVersionAsset {
		tx = types.NewAssetTransferTransaction(from, to, h.profile().Asset.NativeAssetID, txAmount, 0, nonce)
		minFee, feeErr := fees.MinimumFee(tx, h.profile())
		if feeErr != nil {
			return types.Transaction{}, feeErr
		}
		tx.Fee = minFee
	} else {
		details := matureLedger.BalanceDetails(from, pending, blocks[len(blocks)-1].Height)
		if details.Spendable < txAmount {
			return types.Transaction{}, fmt.Errorf("insufficient mature balance: spendable %s %s, required %s %s, active stake %s %s, unlocking stake %s %s", amount.Format(details.Spendable), config.Ticker, amount.Format(txAmount), config.Ticker, amount.Format(details.ActiveStake), config.Ticker, amount.Format(details.UnlockingStake), config.Ticker)
		}
		tx = types.NewUnsignedTransaction(from, to, txAmount, 0, nonce)
	}
	if h.profile().TxVersion >= types.TxVersionAsset {
		fee := tx.Fee
		details := matureLedger.BalanceDetails(from, pending, blocks[len(blocks)-1].Height)
		required, requiredErr := arith.Add(txAmount, fee)
		if requiredErr != nil {
			return types.Transaction{}, errors.New("transaction amount and fee overflow")
		}
		if details.Spendable < required {
			return types.Transaction{}, fmt.Errorf("insufficient mature balance: spendable %s %s, required %s %s, active stake %s %s, unlocking stake %s %s", amount.Format(details.Spendable), h.profile().Asset.NativeAssetSymbol, amount.Format(required), h.profile().Asset.NativeAssetSymbol, amount.Format(details.ActiveStake), h.profile().Asset.NativeAssetSymbol, amount.Format(details.UnlockingStake), h.profile().Asset.NativeAssetSymbol)
		}
	}
	if err := fromWallet.SignTransactionWithProfile(&tx, h.profile()); err != nil {
		return types.Transaction{}, err
	}
	return tx, nil
}

func (h handler) createStakeLockTx(address, amountText string) (types.Transaction, error) {
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		return types.Transaction{}, errors.New("invalid address")
	}
	txAmount, err := amount.Parse(amountText)
	if err != nil {
		return types.Transaction{}, err
	}
	params := h.profile().Consensus
	if !params.Staking.Enabled {
		return types.Transaction{}, errors.New("staking disabled")
	}
	if txAmount < params.Staking.MinStakeAmount {
		return types.Transaction{}, errors.New("invalid stake lock: amount below minimum")
	}
	blocks, err := h.blocks()
	if err != nil {
		return types.Transaction{}, err
	}
	store := wallet.NewStore(h.paths.Wallets)
	w, ok, err := store.Find(address)
	if err != nil {
		return types.Transaction{}, err
	}
	if !ok {
		return types.Transaction{}, errors.New("local wallet not found for address")
	}
	pending, err := mempool.New(h.paths.Mempool).Load()
	if err != nil {
		return types.Transaction{}, err
	}
	matureLedger, err := ledger.ReplayMatureWithProfile(blocks, params, h.profile())
	if err != nil {
		return types.Transaction{}, err
	}
	details := matureLedger.BalanceDetails(address, pending, blocksHeight(blocks))
	if details.Spendable < txAmount {
		return types.Transaction{}, errors.New("invalid stake lock: insufficient mature spendable balance")
	}
	tx := types.NewStakeLockTransaction(address, txAmount, matureLedger.Nonce(address)+pendingFromCount(pending, address)+1)
	if err := w.SignTransaction(&tx); err != nil {
		return types.Transaction{}, err
	}
	tx.StakeID = tx.ID
	return tx, nil
}

func (h handler) createStakeUnlockTx(address, stakeID string) (types.Transaction, uint64, error) {
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		return types.Transaction{}, 0, errors.New("invalid address")
	}
	if stakeID == "" {
		return types.Transaction{}, 0, errors.New("stake id is required")
	}
	params := h.profile().Consensus
	blocks, err := h.blocks()
	if err != nil {
		return types.Transaction{}, 0, err
	}
	state, err := staking.Replay(blocks, params.Staking)
	if err != nil {
		return types.Transaction{}, 0, err
	}
	record, ok := state.Find(stakeID, blocksHeight(blocks))
	if !ok {
		return types.Transaction{}, 0, errors.New("invalid stake unlock: stake not found")
	}
	if record.OwnerAddress != address {
		return types.Transaction{}, 0, errors.New("invalid stake unlock: owner mismatch")
	}
	if record.Status != staking.StatusActive {
		return types.Transaction{}, 0, fmt.Errorf("invalid stake unlock: stake %s", record.Status)
	}
	w, ok, err := wallet.NewStore(h.paths.Wallets).Find(address)
	if err != nil {
		return types.Transaction{}, 0, err
	}
	if !ok {
		return types.Transaction{}, 0, errors.New("local wallet not found for address")
	}
	pending, err := mempool.New(h.paths.Mempool).Load()
	if err != nil {
		return types.Transaction{}, 0, err
	}
	if _, exists := staking.PendingUnlockIDs(pending, address)[stakeID]; exists {
		return types.Transaction{}, 0, errors.New("invalid stake unlock: already pending")
	}
	matureLedger, err := ledger.ReplayMatureWithProfile(blocks, params, h.profile())
	if err != nil {
		return types.Transaction{}, 0, err
	}
	tx := types.NewStakeUnlockTransaction(address, stakeID, matureLedger.Nonce(address)+pendingFromCount(pending, address)+1)
	if err := w.SignTransaction(&tx); err != nil {
		return types.Transaction{}, 0, err
	}
	return tx, blocksHeight(blocks) + 1 + params.Staking.UnbondingPeriodBlocks, nil
}

func (h handler) blocks() ([]types.Block, error) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		return nil, err
	}
	defer closeFn()
	return bc.Blocks()
}

func blocksHeight(blocks []types.Block) uint64 {
	if len(blocks) == 0 {
		return 0
	}
	return blocks[len(blocks)-1].Height
}

func pendingFromCount(pending []types.Transaction, address string) uint64 {
	var count uint64
	for _, tx := range pending {
		if !tx.Coinbase && tx.From == address {
			count++
		}
	}
	return count
}

func stakeRecordMap(record staking.Record) map[string]any {
	return map[string]any{
		"stake_id":       record.StakeID,
		"owner_address":  record.OwnerAddress,
		"amount":         amount.Format(record.Amount),
		"status":         record.Status,
		"lock_tx_id":     record.LockTxID,
		"lock_height":    record.LockHeight,
		"unlock_tx_id":   record.UnlockTxID,
		"unlock_height":  record.UnlockHeight,
		"release_height": record.ReleaseHeight,
	}
}

func explorerBlockSummary(block types.Block) map[string]any {
	coinbaseID := ""
	reward := uint64(0)
	for _, tx := range block.Transactions {
		if tx.Coinbase {
			coinbaseID = tx.ID
			reward = tx.Amount
			break
		}
	}
	return map[string]any{
		"height":          block.Height,
		"hash":            block.Hash,
		"previous_hash":   block.PreviousHash,
		"timestamp":       block.Timestamp,
		"difficulty":      block.Difficulty,
		"nonce":           block.Nonce,
		"tx_count":        len(block.Transactions),
		"coinbase_tx_id":  coinbaseID,
		"miner_address":   block.MinerAddress,
		"reward":          amount.Format(reward) + " " + config.Ticker,
		"cumulative_work": chain.CalculateCumulativeWork([]types.Block{block}),
	}
}

func explorerBlockDetail(block types.Block, tipHeight int) map[string]any {
	txs := make([]map[string]any, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		txs = append(txs, explorerTxSummary(tx))
	}
	confirmations := uint64(0)
	if tipHeight >= 0 && uint64(tipHeight) >= block.Height {
		confirmations = uint64(tipHeight) - block.Height + 1
	}
	return map[string]any{
		"ok":             true,
		"height":         block.Height,
		"hash":           block.Hash,
		"previous_hash":  block.PreviousHash,
		"timestamp":      block.Timestamp,
		"difficulty":     block.Difficulty,
		"nonce":          block.Nonce,
		"miner_address":  block.MinerAddress,
		"merkle_root":    block.MerkleRoot,
		"genesis_marker": block.GenesisMarker,
		"tx_count":       len(block.Transactions),
		"transactions":   txs,
		"confirmations":  confirmations,
	}
}

func explorerTxSummary(tx types.Transaction) map[string]any {
	return map[string]any{
		"txid":               tx.ID,
		"type":               tx.TxType(),
		"from":               tx.From,
		"to":                 tx.To,
		"amount":             amount.Format(tx.Amount) + " " + config.Ticker,
		"fee":                amount.Format(tx.Fee) + " " + config.Ticker,
		"nonce":              tx.Nonce,
		"timestamp":          tx.Timestamp,
		"coinbase":           tx.Coinbase,
		"stake_id":           tx.StakeID,
		"involved_addresses": involvedAddresses(tx),
	}
}

func explorerTxDetail(tx types.Transaction, status string, block *types.Block, tipHeight uint64) map[string]any {
	out := explorerTxSummary(tx)
	out["ok"] = true
	out["status"] = status
	out["confirmations"] = uint64(0)
	if block != nil {
		out["block_height"] = block.Height
		out["block_hash"] = block.Hash
		out["block_time"] = block.Timestamp
		if tipHeight >= block.Height {
			out["confirmations"] = tipHeight - block.Height + 1
		}
	}
	return out
}

func (h handler) explorerStakeRecords(address string) ([]map[string]any, error) {
	blocks, _, err := h.chainInfoData()
	if err != nil {
		return nil, err
	}
	state, err := staking.Replay(blocks, h.profile().Consensus.Staking)
	if err != nil {
		return nil, err
	}
	currentHeight := blocks[len(blocks)-1].Height
	records := state.Records(currentHeight)
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		if address != "" && record.OwnerAddress != address {
			continue
		}
		item := stakeRecordMap(record)
		item["amount"] = amount.Format(record.Amount) + " " + config.Ticker
		item["unbonding_period"] = h.profile().Consensus.Staking.UnbondingPeriodBlocks
		if currentHeight >= record.LockHeight {
			item["confirmations"] = currentHeight - record.LockHeight + 1
		}
		out = append(out, item)
	}
	return out, nil
}

func (h handler) explorerServiceSummary(address string, node *servicenode.Node) map[string]any {
	score, _ := h.serviceStore().Score(address)
	if node == nil {
		nodes, _ := h.serviceStore().LoadNodes()
		for _, candidate := range nodes {
			if candidate.OwnerAddress == address {
				node = &candidate
				break
			}
		}
	}
	status := "not_registered"
	endpoint := ""
	lastSeen := int64(0)
	if node != nil {
		status = node.Status
		endpoint = node.AdvertisedEndpoint
		lastSeen = node.LastSeenAt
	}
	eligibleStatus := "not_eligible"
	if score.StakeEligible {
		eligibleStatus = "eligible"
	}
	return map[string]any{
		"ok":                          true,
		"owner_address":               address,
		"endpoint":                    endpoint,
		"registered_status":           status,
		"last_heartbeat":              lastSeen,
		"score":                       score.ServiceScore,
		"points":                      score.SimulatedPoints,
		"eligible_simulated_points":   score.EligibleSimulatedPoints,
		"required_stake":              amount.Format(score.RequiredStake) + " " + config.Ticker,
		"active_stake":                amount.Format(score.ActiveStake) + " " + config.Ticker,
		"collateral_eligible":         score.StakeEligible,
		"status":                      eligibleStatus,
		"simulation_only":             true,
		"scope":                       "local_node_service_store",
		"service_points_warning":      "service points are simulation-only and are not spendable IDR",
		"service_registry_consensus":  false,
		"stake_collateral_consensus":  true,
		"local_service_state_warning": "service registration and score samples are local to this RPC node",
	}
}

func explorerLimitOffset(w http.ResponseWriter, r *http.Request, def, max int) (int, int, bool) {
	limit := def
	offset := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			explorerError(w, http.StatusBadRequest, "invalid_limit", "limit must be an integer")
			return 0, 0, false
		}
		if parsed < 1 {
			explorerError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 100")
			return 0, 0, false
		}
		limit = parsed
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			explorerError(w, http.StatusBadRequest, "invalid_offset", "offset must be a non-negative integer")
			return 0, 0, false
		}
		offset = parsed
	}
	if limit > max {
		limit = max
	}
	return limit, offset, true
}

func explorerPagedResponse(key string, items []map[string]any, total, limit, offset int) map[string]any {
	resp := map[string]any{
		"ok":          true,
		key:           items,
		"count":       len(items),
		"total_count": total,
		"limit":       limit,
		"offset":      offset,
	}
	if offset > 0 {
		prev := offset - limit
		if prev < 0 {
			prev = 0
		}
		resp["prev_offset"] = prev
	}
	if offset+len(items) < total {
		resp["next_offset"] = offset + limit
	}
	return resp
}

func paginateMaps(items []map[string]any, limit, offset int) []map[string]any {
	if offset >= len(items) {
		return []map[string]any{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

func explorerAddressStats(address string, blocks []types.Block, pending []types.Transaction) (count int, received, sent uint64, first, last any) {
	for _, block := range blocks {
		for _, tx := range block.Transactions {
			if !txInvolvesAddress(tx, address) {
				continue
			}
			count++
			if first == nil {
				first = block.Height
			}
			last = block.Height
			if tx.To == address {
				received += tx.Amount
			}
			if tx.From == address {
				sent += tx.Amount + tx.Fee
			}
		}
	}
	for _, tx := range pending {
		if txInvolvesAddress(tx, address) {
			count++
		}
	}
	return count, received, sent, first, last
}

func txInvolvesAddress(tx types.Transaction, address string) bool {
	return tx.From == address || tx.To == address
}

func txDeltaForAddress(tx types.Transaction, address string) int64 {
	delta := int64(0)
	if tx.To == address && tx.Amount <= uint64(^uint64(0)>>1) {
		delta += int64(tx.Amount)
	}
	if tx.From == address {
		if tx.Amount <= uint64(^uint64(0)>>1) {
			delta -= int64(tx.Amount)
		}
		if tx.Fee <= uint64(^uint64(0)>>1) {
			delta -= int64(tx.Fee)
		}
	}
	return delta
}

func formatSignedAmount(units int64) string {
	if units < 0 {
		return "-" + amount.Format(uint64(-units)) + " " + config.Ticker
	}
	return amount.Format(uint64(units)) + " " + config.Ticker
}

func involvedAddresses(tx types.Transaction) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, address := range []string{tx.From, tx.To} {
		if address == "" || address == types.CoinbaseSender {
			continue
		}
		if _, ok := seen[address]; ok {
			continue
		}
		seen[address] = struct{}{}
		out = append(out, address)
	}
	return out
}

func isUnsignedInteger(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isHexHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

type txLookupResult struct {
	tx          types.Transaction
	status      string
	blockHeight uint64
}

func (h handler) findTransaction(txID string) (txLookupResult, bool, error) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		return txLookupResult{}, false, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return txLookupResult{}, false, err
	}
	for _, block := range blocks {
		for _, tx := range block.Transactions {
			if tx.ID == txID {
				return txLookupResult{tx: tx, status: "confirmed", blockHeight: block.Height}, true, nil
			}
		}
	}
	pending, err := mempool.New(h.paths.Mempool).Load()
	if err != nil {
		return txLookupResult{}, false, err
	}
	for _, tx := range pending {
		if tx.ID == txID {
			return txLookupResult{tx: tx, status: "pending"}, true, nil
		}
	}
	return txLookupResult{}, false, nil
}

type addressInspection struct {
	address               string
	confirmedBalance      uint64
	matureBalance         uint64
	immatureBalance       uint64
	spendableBalance      uint64
	confirmedNonce        uint64
	pendingOutgoingCount  uint64
	pendingOutgoingAmount uint64
	pendingIncomingCount  uint64
	pendingIncomingAmount uint64
}

func (h handler) inspectAddress(address string) (addressInspection, error) {
	if err := crypto.ValidateAddressForNetwork(address, h.profile()); err != nil {
		return addressInspection{}, err
	}
	bc, closeFn, err := h.openChain()
	if err != nil {
		return addressInspection{}, err
	}
	defer closeFn()
	pending, err := mempool.New(h.paths.Mempool).Load()
	if err != nil {
		return addressInspection{}, err
	}
	details, err := bc.BalanceDetailsForWithProfile(address, pending, h.profile())
	if err != nil {
		return addressInspection{}, err
	}
	nonce, err := bc.AccountNonceWithProfile(address, h.profile())
	if err != nil {
		return addressInspection{}, err
	}
	info := addressInspection{
		address:          address,
		confirmedBalance: details.Confirmed,
		matureBalance:    details.Mature,
		immatureBalance:  details.Immature,
		spendableBalance: details.Spendable,
		confirmedNonce:   nonce,
	}
	for _, tx := range pending {
		if tx.Coinbase {
			continue
		}
		involvesOutgoing := false
		outgoingAmount := uint64(0)
		if tx.TxType() == types.TxTypeTransfer && tx.From == address && asset.IsNative(tx.EffectiveAssetID()) {
			involvesOutgoing = true
			outgoingAmount = arith.AddCap(outgoingAmount, tx.Amount)
		}
		if tx.EffectiveFeePayer() == address && tx.Fee > 0 {
			involvesOutgoing = true
			outgoingAmount = arith.AddCap(outgoingAmount, tx.Fee)
		}
		if involvesOutgoing {
			info.pendingOutgoingCount++
			info.pendingOutgoingAmount = arith.AddCap(info.pendingOutgoingAmount, outgoingAmount)
		}
		if tx.TxType() == types.TxTypeTransfer && tx.To == address && asset.IsNative(tx.EffectiveAssetID()) {
			info.pendingIncomingCount++
			info.pendingIncomingAmount = arith.AddCap(info.pendingIncomingAmount, tx.Amount)
		}
	}
	return info, nil
}

func addressFormat(addr string) string {
	if crypto.IsLegacyDevAddress(addr) {
		return "legacy-dev"
	}
	return "base58check"
}

func (h handler) openChain() (*chain.Blockchain, func(), error) {
	store, err := storage.OpenBolt(h.paths.DB)
	if err != nil {
		return nil, nil, err
	}
	bc := chain.New(store)
	if err := bc.InitWithProfile(h.profile()); err != nil {
		_ = store.Close()
		return nil, nil, err
	}
	return bc, func() { _ = store.Close() }, nil
}

func (h handler) currentHeight() (uint64, error) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		return 0, err
	}
	defer closeFn()
	tip, err := bc.Tip()
	if err != nil {
		return 0, err
	}
	return tip.Height, nil
}

func (h handler) ensurePeerCapacity(store p2p.PeerStore, peerURL string) error {
	normalized, err := p2p.NormalizePeerURL(peerURL)
	if err != nil {
		return err
	}
	peerURL = normalized
	peers, err := store.LoadMetadata()
	if err != nil {
		return err
	}
	for _, peer := range peers {
		if peer.URL == peerURL {
			return nil
		}
	}
	if h.info.MaxPeers > 0 && len(peers) >= h.info.MaxPeers {
		return fmt.Errorf("max peers reached: %d", h.info.MaxPeers)
	}
	return nil
}

func peerView(peer p2p.PeerMetadata) map[string]any {
	return map[string]any{
		"url":                  peer.URL,
		"node_id":              peer.NodeID,
		"network_id":           peer.NetworkID,
		"chain_id":             peer.ChainID,
		"genesis_hash":         peer.GenesisHash,
		"protocol_version":     peer.ProtocolVersion,
		"height":               peer.LastHeight,
		"last_height":          peer.LastHeight,
		"tip_hash":             peer.LastTipHash,
		"last_tip_hash":        peer.LastTipHash,
		"status":               peer.Status,
		"score":                peer.Score,
		"source":               peer.Source,
		"last_seen_at":         peer.LastSeenAt,
		"first_seen":           peer.FirstSeenAt,
		"last_success":         peer.LastSuccessAt,
		"last_failure":         peer.LastFailureAt,
		"failure_count":        peer.FailureCount,
		"success_count":        peer.SuccessCount,
		"cooldown_until":       peer.CooldownUntil,
		"last_error":           peer.LastError,
		"last_latency_ms":      peer.LastLatencyMS,
		"latency_ms":           peer.LastLatencyMS,
		"last_status_check":    peer.LastStatusCheckAt,
		"last_status_check_at": peer.LastStatusCheckAt,
		"reason":               peer.LastScoreReason,
		"last_score_reason":    peer.LastScoreReason,
		"version":              peer.Version,
		"protocol":             peer.Protocol,
		"services":             peer.Services,
		"next_retry_at":        peer.NextRetryAt,
		"last_discovery_at":    peer.LastDiscoveryAt,
	}
}

func txView(tx types.Transaction, status string, profile config.NetworkConfig) map[string]any {
	view := map[string]any{
		"id":        tx.ID,
		"txid":      tx.ID,
		"status":    status,
		"from":      tx.From,
		"to":        tx.To,
		"nonce":     tx.Nonce,
		"coinbase":  tx.Coinbase,
		"timestamp": tx.Timestamp,
	}
	if tx.ProtocolVersion() >= types.TxVersionAsset {
		view["asset_id"] = tx.EffectiveAssetID()
		view["amount_units"] = tx.Amount
		// Keep a human-readable amount for generic RPC/CLI consumers; amount_units remains canonical.
		view["amount"] = amount.FormatUnits(tx.Amount, profile.Asset.NativeAssetDecimals) + " " + profile.Asset.FeeAssetID
		view["fee"] = amount.FormatUnits(tx.Fee, profile.Asset.NativeAssetDecimals) + " " + profile.Asset.FeeAssetID
		view["fee_units"] = tx.Fee
	} else {
		view["amount"] = amount.Format(tx.Amount) + " " + config.Ticker
		view["fee"] = amount.Format(tx.Fee) + " " + config.Ticker
	}
	return view
}

func miningJobView(job mining.MiningJob) map[string]any {
	if job.Status == "" || job.Status == mining.StatusIdle {
		return map[string]any{"status": mining.StatusIdle}
	}
	return map[string]any{
		"status":           job.Status,
		"job_id":           job.ID,
		"miner_address":    job.MinerAddress,
		"requested_blocks": job.RequestedBlocks,
		"mined_blocks":     job.MinedBlocks,
		"last_height":      job.LastHeight,
		"last_hash":        job.LastHash,
		"started_at":       job.StartedAt.Format(time.RFC3339),
		"updated_at":       job.UpdatedAt.Format(time.RFC3339),
		"error":            job.Error,
	}
}

func (h handler) p2pAdvertise() string {
	if h.info.P2PAdvertise != "" {
		return h.info.P2PAdvertise
	}
	if lock, err := p2p.ReadLock(h.paths.Lock); err == nil {
		return lock.P2PAdvertise
	}
	return ""
}

func (h handler) compareInfo() (map[string]any, error) {
	bc, closeFn, err := h.openChain()
	if err != nil {
		return nil, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return nil, err
	}
	tip := blocks[len(blocks)-1]
	pending, _ := mempool.New(h.paths.Mempool).Load()
	return map[string]any{
		"network_id":       h.profile().NetworkID,
		"chain_id":         h.profile().ChainID,
		"genesis_hash":     chain.GenesisBlockForNetwork(h.profile()).Hash,
		"protocol_version": h.profile().ProtocolVersion,
		"height":           tip.Height,
		"tip_hash":         tip.Hash,
		"total_supply":     amount.Format(ledger.TotalSupplyWithProfile(blocks, h.profile())) + " " + config.NativeAssetSymbol,
		"cumulative_work":  chain.CalculateCumulativeWork(blocks),
		"pending_tx_count": len(pending),
	}, nil
}

func remoteCompareInfo(peer string) (map[string]any, error) {
	resp, err := http.Get(strings.TrimRight(peer, "/") + "/chain/info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("peer returned %s", resp.Status)
	}
	return out, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"ok": false, "error": "request body too large"})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
}

func explorerError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": code, "message": message})
}

func normalizeNodeInfo(info NodeInfo) NodeInfo {
	if !info.EnableWalletRPCSet {
		info.EnableWalletRPC = !info.PublicRPC
	}
	if !info.EnableAdminRPCSet {
		info.EnableAdminRPC = !info.PublicRPC
	}
	if !info.EnableMinerRPCSet {
		info.EnableMinerRPC = !info.PublicRPC
	}
	if !info.EnableServiceRPCSet {
		info.EnableServiceRPC = !info.PublicRPC
	}
	if info.RateLimitPerMinute == 0 {
		info.RateLimitPerMinute = 300
	}
	if info.MinerRateLimitPerMinute == 0 {
		info.MinerRateLimitPerMinute = 120
	}
	if info.WalletRateLimitPerMinute == 0 {
		info.WalletRateLimitPerMinute = 30
	}
	if info.ServiceRateLimitPerMinute == 0 {
		info.ServiceRateLimitPerMinute = 60
	}
	if info.Profile.Name == "" {
		info.Profile = config.Localnet()
	}
	if info.MaxPeers == 0 {
		info.MaxPeers = info.Profile.MaxPeers
	}
	if info.MaxReorgDepth == 0 {
		info.MaxReorgDepth = config.DefaultMaxReorgDepth(info.Profile)
	}
	if info.MinMiningPeers == 0 {
		info.MinMiningPeers = info.Profile.MinMiningPeers
	}
	if !info.AllowIsolatedMining {
		info.AllowIsolatedMining = info.Profile.AllowIsolatedMining
	}
	if info.MinWritePeers == 0 {
		info.MinWritePeers = info.Profile.MinWritePeers
	}
	if !info.AllowIsolatedWrites {
		info.AllowIsolatedWrites = info.Profile.AllowIsolatedWrites
	}
	if info.StartedAt.IsZero() {
		info.StartedAt = time.Now()
	}
	return info
}

func (h handler) wrap(kind string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.applyCORS(w, r) {
			return
		}
		if !h.endpointEnabled(kind) {
			if strings.HasPrefix(kind, "service-") {
				writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "service RPC disabled"})
				return
			}
			writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "endpoint disabled in public RPC mode"})
			return
		}
		limit := h.limitFor(kind)
		if limit > 0 && !h.limiter.allow(remoteIP(r), kind, limit) {
			log.Printf("rpc rate limited ip=%s path=%s", remoteIP(r), r.URL.Path)
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"ok": false, "error": "rate limit exceeded"})
			return
		}
		next(w, r)
	}
}

func (h handler) endpointEnabled(kind string) bool {
	switch kind {
	case "wallet":
		return h.info.EnableWalletRPC
	case "miner":
		return h.info.EnableMinerRPC
	case "admin":
		return h.info.EnableAdminRPC
	case "service-register", "service-heartbeat", "service-challenge":
		return h.info.EnableServiceRPC
	default:
		return true
	}
}

func (h handler) limitFor(kind string) int {
	switch kind {
	case "wallet":
		return h.info.WalletRateLimitPerMinute
	case "miner":
		return h.info.MinerRateLimitPerMinute
	case "service-register":
		return 30
	case "service-heartbeat":
		return 120
	case "service-challenge":
		return h.info.ServiceRateLimitPerMinute
	default:
		return h.info.RateLimitPerMinute
	}
}

func (h handler) serviceStore() servicenode.Store {
	return servicenode.NewStore(h.paths, h.profile())
}

func (h handler) applyCORS(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin != "" && originAllowed(origin, h.info.CORSOrigins) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
	}
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func originAllowed(origin string, allowed []string) bool {
	for _, item := range allowed {
		item = strings.TrimSpace(item)
		if item == "*" || item == origin {
			return true
		}
	}
	return false
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]rateBucket
	now     func() time.Time
}

type rateBucket struct {
	window time.Time
	count  int
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{buckets: make(map[string]rateBucket), now: time.Now}
}

func (l *rateLimiter) allow(ip, kind string, limit int) bool {
	if ip == "" {
		ip = "unknown"
	}
	now := l.now()
	window := now.Truncate(time.Minute)
	key := ip + "|" + kind
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, bucket := range l.buckets {
		if now.Sub(bucket.window) > 2*time.Minute {
			delete(l.buckets, k)
		}
	}
	bucket := l.buckets[key]
	if !bucket.window.Equal(window) {
		bucket = rateBucket{window: window}
	}
	bucket.count++
	l.buckets[key] = bucket
	return bucket.count <= limit
}
