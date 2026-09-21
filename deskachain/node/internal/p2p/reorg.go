package p2p

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"indochain/internal/chain"
	"indochain/internal/config"
	"indochain/internal/ledger"
	"indochain/internal/mempool"
	"indochain/internal/types"
)

const DefaultMaxReorgDepth uint64 = 64

type BlockSummary struct {
	Height       uint64 `json:"height"`
	Hash         string `json:"hash"`
	TxCount      int    `json:"tx_count"`
	MinerAddress string `json:"miner_address"`
	Difficulty   uint32 `json:"difficulty"`
}
type ReorgPlan struct {
	Peer                 string         `json:"peer"`
	Network              string         `json:"network"`
	NetworkID            string         `json:"network_id"`
	ChainID              uint64         `json:"chain_id"`
	LocalNetworkID       string         `json:"local_network_id"`
	PeerNetworkID        string         `json:"peer_network_id"`
	LocalChainID         uint64         `json:"local_chain_id"`
	PeerChainID          uint64         `json:"peer_chain_id"`
	LocalGenesis         string         `json:"local_genesis"`
	PeerGenesis          string         `json:"peer_genesis"`
	LocalHeight          uint64         `json:"local_height"`
	LocalTip             string         `json:"local_tip"`
	PeerHeight           uint64         `json:"peer_height"`
	PeerTip              string         `json:"peer_tip"`
	CommonAncestorHeight uint64         `json:"common_ancestor_height"`
	CommonAncestorHash   string         `json:"common_ancestor_hash"`
	DisconnectBlocks     []BlockSummary `json:"disconnect_blocks_detail"`
	ConnectBlocks        []BlockSummary `json:"connect_blocks_detail"`
	ReorgDepth           uint64         `json:"reorg_depth"`
	LocalWork            uint64         `json:"local_work"`
	PeerWork             uint64         `json:"peer_work"`
	PeerHasMoreWork      bool           `json:"peer_has_more_work"`
	MaxDepth             uint64         `json:"max_depth"`
	MaxReorgDepth        uint64         `json:"max_reorg_depth"`
	Allowed              bool           `json:"allowed"`
	Reason               string         `json:"reason"`
	Decision             string         `json:"decision"`
}
type ReorgResult struct {
	Applied                      bool      `json:"applied"`
	OldHeight                    uint64    `json:"old_height"`
	NewHeight                    uint64    `json:"new_height"`
	OldTip                       string    `json:"old_tip"`
	NewTip                       string    `json:"new_tip"`
	DisconnectedBlocks           int       `json:"disconnected_blocks"`
	ConnectedBlocks              int       `json:"connected_blocks"`
	RequeuedTransactions         int       `json:"requeued_transactions"`
	DroppedTransactions          int       `json:"dropped_transactions"`
	DroppedConfirmedTransactions int       `json:"dropped_confirmed_transactions"`
	DroppedInvalidTransactions   int       `json:"dropped_invalid_transactions"`
	DroppedDuplicateTransactions int       `json:"dropped_duplicate_transactions"`
	MempoolCount                 int       `json:"mempool_count"`
	ChainValid                   bool      `json:"chain_valid"`
	Error                        string    `json:"error,omitempty"`
	Plan                         ReorgPlan `json:"plan"`
}

func BuildReorgPlan(paths config.Paths, peer string, maxDepth uint64) (ReorgPlan, []types.Block, error) {
	// Compatibility wrapper for legacy callers that predate network profiles.
	return BuildReorgPlanWithProfile(paths, peer, maxDepth, config.Localnet())
}

func BuildReorgPlanWithProfile(paths config.Paths, peer string, maxDepth uint64, profile config.NetworkConfig) (ReorgPlan, []types.Block, error) {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	if maxDepth == 0 {
		maxDepth = config.DefaultMaxReorgDepth(profile)
	}
	if err := ValidatePeerURL(peer); err != nil {
		return ReorgPlan{}, nil, err
	}
	client, err := NewClientForProfile(paths, profile, 10*time.Second)
	if err != nil {
		return ReorgPlan{}, nil, err
	}
	statusClient, err := NewClientForProfile(paths, profile, 2*time.Second)
	if err != nil {
		return ReorgPlan{}, nil, err
	}
	hs, err := statusClient.Handshake(peer)
	if err != nil {
		return ReorgPlan{}, nil, err
	}
	net := profile
	net.GenesisHash = chain.GenesisBlockForNetwork(profile).Hash
	if err := ValidateHandshake(net, hs); err != nil {
		return ReorgPlan{}, nil, err
	}
	st, err := statusClient.Status(peer)
	if err != nil {
		return ReorgPlan{}, nil, err
	}
	if err := ValidateStatus(net, st); err != nil {
		return ReorgPlan{}, nil, err
	}
	bc, closeFn, err := openChainWithProfile(paths, profile)
	if err != nil {
		return ReorgPlan{}, nil, err
	}
	defer closeFn()
	localBlocks, err := bc.Blocks()
	if err != nil {
		return ReorgPlan{}, nil, err
	}
	localTip := localBlocks[len(localBlocks)-1]
	loc, err := client.Locator(peer)
	if err != nil {
		return ReorgPlan{}, nil, err
	}
	ancestor, ok := chain.FindCommonAncestor(localBlocks, loc.Locator)
	if !ok {
		return ReorgPlan{}, nil, fmt.Errorf("common ancestor not found")
	}
	localGenesis := chain.GenesisBlockForNetwork(profile).Hash
	plan := ReorgPlan{
		Peer:                 peer,
		Network:              profile.Name,
		NetworkID:            profile.NetworkID,
		ChainID:              profile.ChainID,
		LocalNetworkID:       profile.NetworkID,
		PeerNetworkID:        st.NetworkID,
		LocalChainID:         profile.ChainID,
		PeerChainID:          st.ChainID,
		LocalGenesis:         localGenesis,
		PeerGenesis:          st.GenesisHash,
		LocalHeight:          localTip.Height,
		LocalTip:             localTip.Hash,
		PeerHeight:           st.Height,
		PeerTip:              st.TipHash,
		CommonAncestorHeight: ancestor.Height,
		CommonAncestorHash:   ancestor.Hash,
		MaxDepth:             maxDepth,
		MaxReorgDepth:        maxDepth,
	}
	plan.ReorgDepth = localTip.Height - ancestor.Height
	if plan.ReorgDepth > maxDepth {
		plan.Allowed = false
		plan.Decision = "reorg_depth_exceeds_max"
		plan.Reason = "reorg depth exceeds max depth"
		return plan, nil, nil
	}
	for i := len(localBlocks) - 1; i >= 0; i-- {
		b := localBlocks[i]
		if b.Height <= ancestor.Height {
			break
		}
		plan.DisconnectBlocks = append(plan.DisconnectBlocks, summarizeBlock(b))
	}
	maxReorgFetch := profile.NetworkLimits.MaxReorgFetchBlocks
	if maxReorgFetch == 0 {
		maxReorgFetch = 512
	}
	peerBranchDepth := st.Height - ancestor.Height
	if peerBranchDepth > maxReorgFetch {
		plan.Allowed = false
		plan.Decision = "reorg_fetch_exceeds_max"
		plan.Reason = "peer reorg branch exceeds max remote fetch"
		return plan, nil, nil
	}
	peerBranch := make([]types.Block, 0, int(peerBranchDepth))
	for h := ancestor.Height + 1; h <= st.Height; h++ {
		b, err := client.Block(peer, h)
		if err != nil {
			return plan, nil, err
		}
		peerBranch = append(peerBranch, b)
		plan.ConnectBlocks = append(plan.ConnectBlocks, summarizeBlock(b))
	}
	plan.LocalWork = chain.CalculateCumulativeWork(localBlocks)
	peerFull := append([]types.Block{}, localBlocks[:ancestor.Height+1]...)
	peerFull = append(peerFull, peerBranch...)
	plan.PeerWork = chain.CalculateCumulativeWork(peerFull)
	plan.PeerHasMoreWork = plan.PeerWork > plan.LocalWork
	switch {
	case plan.PeerWork == plan.LocalWork:
		plan.Allowed = false
		plan.Decision = "fork_tie_same_work"
		plan.Reason = "fork tie: peer chain has equal cumulative work"
	case plan.PeerWork < plan.LocalWork:
		plan.Allowed = false
		plan.Decision = "local_ahead_more_work"
		plan.Reason = "local chain has higher cumulative work"
	default:
		if _, err := chain.ValidateChainWithNetwork(peerFull, profile); err != nil {
			plan.Allowed = false
			plan.Decision = "peer_branch_invalid"
			plan.Reason = "peer branch validation failed: " + err.Error()
		} else {
			plan.Allowed = true
			plan.Decision = "reorg_apply_higher_work"
			plan.Reason = "peer chain has more cumulative work and depth is within limit"
		}
	}
	return plan, peerBranch, nil
}

type MempoolRevalidationSummary struct {
	Kept             int `json:"kept"`
	DroppedInvalid   int `json:"dropped_invalid"`
	DroppedConfirmed int `json:"dropped_confirmed"`
	DroppedDuplicate int `json:"dropped_duplicate"`
}

func ApplyReorg(paths config.Paths, peer string, maxDepth uint64, yes bool) (ReorgResult, error) {
	// Compatibility wrapper for legacy callers that predate network profiles.
	return ApplyReorgWithProfile(paths, peer, maxDepth, yes, config.Localnet())
}

func ApplyReorgWithProfile(paths config.Paths, peer string, maxDepth uint64, yes bool, profile config.NetworkConfig) (ReorgResult, error) {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	if !yes {
		return ReorgResult{Applied: false, Error: "refusing to apply reorg without --yes"}, fmt.Errorf("refusing to apply reorg without --yes")
	}
	plan, branch, err := BuildReorgPlanWithProfile(paths, peer, maxDepth, profile)
	if err != nil {
		return ReorgResult{Applied: false, Error: err.Error()}, err
	}
	if !plan.Allowed {
		return ReorgResult{Applied: false, Error: plan.Reason, Plan: plan}, errors.New(plan.Reason)
	}
	bc, closeFn, err := openChainWithProfile(paths, profile)
	if err != nil {
		return ReorgResult{}, err
	}
	closed := false
	defer func() {
		if !closed {
			closeFn()
		}
	}()
	localBlocks, err := bc.Blocks()
	if err != nil {
		return ReorgResult{}, err
	}
	oldTip := localBlocks[len(localBlocks)-1]
	orphanTxs := map[string]types.Transaction{}
	for _, b := range localBlocks[plan.CommonAncestorHeight+1:] {
		for _, tx := range b.Transactions {
			if !tx.Coinbase {
				orphanTxs[tx.ID] = tx
			}
		}
	}
	if err := bc.ReplaceFromHeightWithNetwork(plan.CommonAncestorHeight+1, branch, profile); err != nil {
		return ReorgResult{}, err
	}
	newBlocks, err := bc.Blocks()
	if err != nil {
		return ReorgResult{}, err
	}
	valid := false
	if _, err := chain.ValidateChainWithNetwork(newBlocks, profile); err == nil {
		valid = true
	} else {
		return ReorgResult{}, err
	}
	l, err := ledger.ReplayMatureWithProfile(newBlocks, profile.Consensus, profile)
	if err != nil {
		return ReorgResult{}, err
	}
	confirmed := confirmedTransactionIDs(newBlocks)
	confirmedDropped := 0
	for id := range orphanTxs {
		if _, ok := confirmed[id]; ok {
			delete(orphanTxs, id)
			confirmedDropped++
		}
	}
	mp := mempool.New(paths.Mempool)
	pending, err := mp.Load()
	if err != nil {
		return ReorgResult{}, err
	}
	kept, reval := revalidateTransactions(pending, l, confirmed, profile)
	work := l.Clone()
	for _, tx := range kept {
		_ = work.ApplyTransaction(tx)
	}
	seen := map[string]struct{}{}
	for _, tx := range kept {
		seen[tx.ID] = struct{}{}
	}
	requeued, droppedInvalid, droppedDuplicate := 0, 0, reval.DroppedDuplicate
	for _, tx := range orphanTxs {
		if _, ok := seen[tx.ID]; ok {
			droppedDuplicate++
			continue
		}
		if err := work.ApplyTransaction(tx); err == nil {
			kept = append(kept, tx)
			seen[tx.ID] = struct{}{}
			requeued++
		} else {
			droppedInvalid++
		}
	}
	nextNonce := make(map[string]uint64)
	for _, tx := range kept {
		if _, ok := nextNonce[tx.From]; ok {
			continue
		}
		nonce := l.Nonce(tx.From)
		if nonce == ^uint64(0) {
			continue
		}
		nextNonce[tx.From] = nonce + 1
	}
	selected, selectErr := mempool.SelectWithNonces(kept, profile, profile.Consensus.MaxGasPerBlock, nextNonce)
	if selectErr != nil {
		closeFn()
		return ReorgResult{}, selectErr
	}
	if len(selected) < len(kept) {
		droppedInvalid += len(kept) - len(selected)
	}
	kept = selected
	if err := mp.Save(kept); err != nil {
		closeFn()
		return ReorgResult{}, err
	}
	mempoolCount := len(kept)
	closeFn()
	closed = true
	tip := newBlocks[len(newBlocks)-1]
	return ReorgResult{Applied: true, OldHeight: oldTip.Height, NewHeight: tip.Height, OldTip: oldTip.Hash, NewTip: tip.Hash, DisconnectedBlocks: len(plan.DisconnectBlocks), ConnectedBlocks: len(plan.ConnectBlocks), RequeuedTransactions: requeued, DroppedTransactions: confirmedDropped + droppedInvalid + droppedDuplicate + reval.DroppedConfirmed + reval.DroppedInvalid, DroppedConfirmedTransactions: confirmedDropped + reval.DroppedConfirmed, DroppedInvalidTransactions: droppedInvalid + reval.DroppedInvalid, DroppedDuplicateTransactions: droppedDuplicate, MempoolCount: mempoolCount, ChainValid: valid, Plan: plan}, nil
}

func RevalidateMempoolAgainstLedger(paths config.Paths) (MempoolRevalidationSummary, error) {
	// Compatibility wrapper for legacy callers that predate network profiles.
	return RevalidateMempoolAgainstLedgerWithProfile(paths, config.Localnet())
}

func RevalidateMempoolAgainstLedgerWithProfile(paths config.Paths, profile config.NetworkConfig) (MempoolRevalidationSummary, error) {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	bc, closeFn, err := openChainWithProfile(paths, profile)
	if err != nil {
		return MempoolRevalidationSummary{}, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return MempoolRevalidationSummary{}, err
	}
	l, err := ledger.ReplayMatureWithProfile(blocks, profile.Consensus, profile)
	if err != nil {
		return MempoolRevalidationSummary{}, err
	}
	mp := mempool.New(paths.Mempool)
	pending, err := mp.Load()
	if err != nil {
		return MempoolRevalidationSummary{}, err
	}
	kept, summary := revalidateTransactions(pending, l, confirmedTransactionIDs(blocks), profile)
	return summary, mp.Save(kept)
}

func confirmedTransactionIDs(blocks []types.Block) map[string]struct{} {
	ids := map[string]struct{}{}
	for _, b := range blocks {
		for _, tx := range b.Transactions {
			if !tx.Coinbase {
				ids[tx.ID] = struct{}{}
			}
		}
	}
	return ids
}

func revalidateTransactions(txs []types.Transaction, l *ledger.MatureLedger, confirmed map[string]struct{}, profile config.NetworkConfig) ([]types.Transaction, MempoolRevalidationSummary) {
	work := l.Clone()
	seen := map[string]struct{}{}
	kept := make([]types.Transaction, 0, len(txs))
	summary := MempoolRevalidationSummary{}
	ordered := append([]types.Transaction(nil), txs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].From != ordered[j].From {
			return ordered[i].From < ordered[j].From
		}
		if ordered[i].Nonce != ordered[j].Nonce {
			return ordered[i].Nonce < ordered[j].Nonce
		}
		return ordered[i].ID < ordered[j].ID
	})
	for _, tx := range ordered {
		if tx.Coinbase {
			summary.DroppedInvalid++
			continue
		}
		if _, ok := confirmed[tx.ID]; ok {
			summary.DroppedConfirmed++
			continue
		}
		if _, ok := seen[tx.ID]; ok {
			summary.DroppedDuplicate++
			continue
		}
		policy := mempool.AdmissionPolicy{
			Profile: profile,
			MaxTxs:  profile.Consensus.MaxTxCount,
			MaxGas:  profile.Consensus.MaxGasPerBlock,
		}
		if err := mempool.ValidateForRevalidation(tx, policy); err != nil {
			summary.DroppedInvalid++
			continue
		}
		if profile.Consensus.MaxTxCount > 0 && uint64(len(kept)) >= profile.Consensus.MaxTxCount {
			summary.DroppedInvalid++
			continue
		}
		if err := work.ApplyTransaction(tx); err != nil {
			summary.DroppedInvalid++
			continue
		}
		seen[tx.ID] = struct{}{}
		kept = append(kept, tx)
		summary.Kept++
	}
	return kept, summary
}

func summarizeBlock(b types.Block) BlockSummary {
	return BlockSummary{Height: b.Height, Hash: b.Hash, TxCount: len(b.Transactions), MinerAddress: b.MinerAddress, Difficulty: b.Difficulty}
}
