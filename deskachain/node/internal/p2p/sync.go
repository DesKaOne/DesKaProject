package p2p

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"indochain/internal/chain"
	"indochain/internal/config"
	"indochain/internal/mempool"
	"indochain/internal/storage"
	"indochain/internal/types"
)

func SyncFromPeer(paths config.Paths, peer string, out io.Writer) error {
	return SyncFromPeerWithProfile(paths, peer, out, config.Localnet())
}

func SyncFromPeerWithProfile(paths config.Paths, peer string, out io.Writer, profile config.NetworkConfig) error {
	return SyncFromPeerWithProfileAndMaxDepth(paths, peer, out, profile, 0)
}

func SyncFromPeerWithProfileAndMaxDepth(paths config.Paths, peer string, out io.Writer, profile config.NetworkConfig, maxReorgDepth uint64) error {
	normalized, err := NormalizePeerURL(peer)
	if err != nil {
		return err
	}
	peer = normalized
	if profile.Name == "" {
		profile = config.Localnet()
	}
	client, err := NewClientForProfile(paths, profile, 10*time.Second)
	if err != nil {
		return err
	}
	statusClient, err := NewClientForProfile(paths, profile, 2*time.Second)
	if err != nil {
		return err
	}
	if maxReorgDepth == 0 {
		maxReorgDepth = config.DefaultMaxReorgDepth(profile)
	}
	localNet := profile
	localNet.GenesisHash = chain.GenesisHashForNetwork(profile)
	handshake, err := statusClient.Handshake(peer)
	if err != nil {
		_ = notePeerFailure(paths, peer, err.Error(), -5, "request failed")
		return fmt.Errorf("peer unavailable: %w", err)
	}
	if err := ValidateHandshake(localNet, handshake); err != nil {
		_ = notePeerFailure(paths, peer, err.Error(), -20, "handshake mismatch")
		return err
	}
	_ = notePeerSuccess(paths, peer, handshake, 5, "handshake")
	bc, closeFn, err := openChainWithProfile(paths, profile)
	if err != nil {
		return err
	}
	localTip, err := bc.Tip()
	if err != nil {
		closeFn()
		return err
	}
	if out != nil {
		fmt.Fprintln(out, "sync started")
		fmt.Fprintf(out, "local height: %d\n", localTip.Height)
		fmt.Fprintf(out, "peer: %s\n", peer)
	}
	if out != nil {
		log.Printf("sync started peer=%s", peer)
	}
	status, err := statusClient.Status(peer)
	if err != nil {
		closeFn()
		_ = notePeerFailure(paths, peer, err.Error(), -5, "request failed")
		return fmt.Errorf("peer unavailable: %w", err)
	}
	if err := ValidateStatus(localNet, status); err != nil {
		closeFn()
		_ = notePeerFailure(paths, peer, err.Error(), -20, "status mismatch")
		return err
	}
	if out != nil {
		fmt.Fprintf(out, "peer height: %d\n", status.Height)
	}
	if status.Height <= localTip.Height {
		localAtPeerHeight := blockAtHeight(bc, status.Height)
		if localAtPeerHeight == nil {
			closeFn()
			err := fmt.Errorf("local block at peer height %d not found", status.Height)
			_ = notePeerFailure(paths, peer, err.Error(), -30, "invalid local chain")
			return err
		}
		if status.TipHash != localAtPeerHeight.Hash {
			closeFn()
			if err := syncReorgFromPeer(paths, peer, out, profile, maxReorgDepth); err != nil {
				_ = notePeerFailure(paths, peer, err.Error(), -10, "fork detected")
				return err
			}
			_ = notePeerSuccess(paths, peer, handshake, 10, "sync reorg")
			return nil
		}
		if out != nil {
			fmt.Fprintln(out, "sync complete")
			if status.Height == localTip.Height {
				fmt.Fprintln(out, "local chain already up to date")
			} else {
				fmt.Fprintln(out, "local chain ahead of peer")
			}
			fmt.Fprintf(out, "peer: %s\n", peer)
			fmt.Fprintf(out, "height: %d\n", localTip.Height)
			fmt.Fprintln(out, "imported blocks: 0")
		}
		if out != nil {
			log.Printf("sync complete peer=%s up_to_date=%t local_ahead=%t", peer, status.Height == localTip.Height, status.Height < localTip.Height)
		}
		closeFn()
		_ = notePeerSuccess(paths, peer, handshake, 1, "sync up to date")
		return nil
	}
	missing := status.Height - localTip.Height
	maxSyncBlocks := profile.NetworkLimits.MaxSyncBlocks
	if maxSyncBlocks == 0 {
		maxSyncBlocks = 500
	}
	if missing > maxSyncBlocks {
		closeFn()
		err := fmt.Errorf("sync range exceeds max remote fetch: missing=%d max=%d", missing, maxSyncBlocks)
		_ = notePeerFailure(paths, peer, err.Error(), -10, "sync range too large")
		return err
	}
	needed := int(missing)
	maxHeaderBatch := profile.NetworkLimits.MaxHeaderBatch
	if maxHeaderBatch == 0 {
		maxHeaderBatch = 500
	}
	if missing > maxHeaderBatch {
		closeFn()
		err := fmt.Errorf("sync header range exceeds max batch: missing=%d max=%d", missing, maxHeaderBatch)
		_ = notePeerFailure(paths, peer, err.Error(), -10, "header range too large")
		return err
	}
	headers, err := client.Headers(peer, localTip.Height+1, needed)
	if err != nil {
		closeFn()
		_ = notePeerFailure(paths, peer, err.Error(), -5, "request failed")
		return fmt.Errorf("peer unavailable: %w", err)
	}
	if len(headers) < needed {
		closeFn()
		err := fmt.Errorf("peer returned insufficient headers: got %d want %d", len(headers), needed)
		_ = notePeerFailure(paths, peer, err.Error(), -30, "invalid header")
		return err
	}
	if len(headers) > 0 && headers[0].PreviousHash != localTip.Hash {
		closeFn()
		if err := syncReorgFromPeer(paths, peer, out, profile, maxReorgDepth); err != nil {
			_ = notePeerFailure(paths, peer, err.Error(), -10, "fork detected")
			return err
		}
		_ = notePeerSuccess(paths, peer, handshake, 10, "sync reorg")
		return nil
	}
	if err := validateHeaders(localTip, headers); err != nil {
		closeFn()
		_ = notePeerFailure(paths, peer, err.Error(), -30, "invalid header")
		return err
	}
	if out != nil {
		fmt.Fprintf(out, "headers checked: %d\n", len(headers))
	}
	for _, header := range headers {
		existing := blockAtHeight(bc, header.Height)
		if existing != nil {
			if existing.Hash == header.Hash {
				continue
			}
			closeFn()
			if err := syncReorgFromPeer(paths, peer, out, profile, maxReorgDepth); err != nil {
				_ = notePeerFailure(paths, peer, err.Error(), -10, "fork detected")
				return err
			}
			_ = notePeerSuccess(paths, peer, handshake, 10, "sync reorg")
			return nil
		}
		block, err := client.Block(peer, header.Height)
		if err != nil {
			closeFn()
			_ = notePeerFailure(paths, peer, err.Error(), -5, "request failed")
			return fmt.Errorf("peer unavailable: %w", err)
		}
		if err := bc.AddBlockWithNetwork(block, profile); err != nil {
			closeFn()
			_ = notePeerFailure(paths, peer, err.Error(), -30, "invalid block")
			return fmt.Errorf("import block %d: %w", header.Height, err)
		}
		if err := removeBlockTxs(paths.Mempool, block); err != nil {
			closeFn()
			return err
		}
		if out != nil {
			fmt.Fprintf(out, "imported block height=%d hash=%s\n", block.Height, block.Hash)
		}
		log.Printf("imported block height=%d hash=%s", block.Height, block.Hash)
	}
	blocks, err := bc.Blocks()
	if err != nil {
		closeFn()
		return err
	}
	if _, err := chain.ValidateChainWithNetwork(blocks, profile); err != nil {
		closeFn()
		return err
	}
	closeFn()
	if err := RevalidateMempoolAgainstChainWithProfile(paths, profile); err != nil {
		return err
	}
	bc, closeFn, err = openChainWithProfile(paths, profile)
	if err != nil {
		return err
	}
	newTip, err := bc.Tip()
	closeFn()
	if err != nil {
		return err
	}
	if out != nil {
		fmt.Fprintln(out, "sync complete")
		fmt.Fprintf(out, "peer: %s\n", peer)
		fmt.Fprintf(out, "imported blocks: %d\n", needed)
		fmt.Fprintf(out, "height: %d\n", newTip.Height)
		fmt.Fprintf(out, "tip hash: %s\n", newTip.Hash)
	}
	log.Printf("sync complete peer=%s height=%d", peer, newTip.Height)
	_ = notePeerSuccess(paths, peer, handshake, 10, "sync imported block")
	return nil
}

func forkSyncError(paths config.Paths, peer string, profile config.NetworkConfig) error {
	result, checkErr := CheckForkWithProfile(paths, peer, profile)
	if checkErr != nil {
		return fmt.Errorf("sync failed: fork detected: %w", checkErr)
	}
	if err := result.SyncError(); err != nil {
		return err
	}
	return errors.New("sync failed: fork detected")
}

type ReorgSyncError struct {
	Plan ReorgPlan
}

func (e *ReorgSyncError) Error() string {
	return formatReorgSyncError(e.Plan)
}

func syncReorgFromPeer(paths config.Paths, peer string, out io.Writer, profile config.NetworkConfig, maxReorgDepth uint64) error {
	plan, _, err := BuildReorgPlanWithProfile(paths, peer, maxReorgDepth, profile)
	if err != nil {
		return fmt.Errorf("sync failed: fork detected: %w", err)
	}
	if !plan.Allowed {
		return &ReorgSyncError{Plan: plan}
	}
	result, err := ApplyReorgWithProfile(paths, peer, maxReorgDepth, true, profile)
	if err != nil {
		if result.Plan.Peer != "" {
			return &ReorgSyncError{Plan: result.Plan}
		}
		return fmt.Errorf("sync failed: reorg apply failed: %w", err)
	}
	if out != nil {
		fmt.Fprintln(out, "sync complete")
		fmt.Fprintf(out, "peer: %s\n", peer)
		fmt.Fprintln(out, "reorg applied: true")
		fmt.Fprintf(out, "decision: %s\n", result.Plan.Decision)
		fmt.Fprintf(out, "reorg depth: %d\n", result.Plan.ReorgDepth)
		fmt.Fprintf(out, "max reorg depth: %d\n", result.Plan.MaxReorgDepth)
		fmt.Fprintf(out, "common ancestor height: %d\n", result.Plan.CommonAncestorHeight)
		fmt.Fprintf(out, "common ancestor hash: %s\n", result.Plan.CommonAncestorHash)
		fmt.Fprintf(out, "imported blocks: %d\n", result.ConnectedBlocks)
		fmt.Fprintf(out, "height: %d\n", result.NewHeight)
		fmt.Fprintf(out, "tip hash: %s\n", result.NewTip)
	}
	log.Printf("sync reorg applied peer=%s network_id=%s chain_id=%d genesis=%s old_height=%d old_tip=%s new_height=%d new_tip=%s local_work=%d peer_work=%d reorg_depth=%d max_reorg_depth=%d ancestor_height=%d ancestor_hash=%s decision=%s reason=%q",
		peer,
		result.Plan.NetworkID,
		result.Plan.ChainID,
		result.Plan.LocalGenesis,
		result.OldHeight,
		result.OldTip,
		result.NewHeight,
		result.NewTip,
		result.Plan.LocalWork,
		result.Plan.PeerWork,
		result.Plan.ReorgDepth,
		result.Plan.MaxReorgDepth,
		result.Plan.CommonAncestorHeight,
		result.Plan.CommonAncestorHash,
		result.Plan.Decision,
		result.Plan.Reason,
	)
	return nil
}

func formatReorgSyncError(plan ReorgPlan) string {
	return fmt.Sprintf("sync failed: fork detected\nlocal network id: %s\npeer network id: %s\nlocal chain id: %d\npeer chain id: %d\nlocal genesis: %s\npeer genesis: %s\nlocal height: %d\nlocal tip: %s\npeer height: %d\npeer tip: %s\nlocal cumulative work: %d\npeer cumulative work: %d\nreorg_depth: %d\nmax_reorg_depth: %d\ncommon ancestor height: %d\ncommon ancestor hash: %s\ndecision: %s\nreason: %s",
		plan.LocalNetworkID,
		plan.PeerNetworkID,
		plan.LocalChainID,
		plan.PeerChainID,
		plan.LocalGenesis,
		plan.PeerGenesis,
		plan.LocalHeight,
		plan.LocalTip,
		plan.PeerHeight,
		plan.PeerTip,
		plan.LocalWork,
		plan.PeerWork,
		plan.ReorgDepth,
		plan.MaxReorgDepth,
		plan.CommonAncestorHeight,
		plan.CommonAncestorHash,
		plan.Decision,
		plan.Reason,
	)
}

func validateHeaders(localTip types.Block, headers []BlockHeader) error {
	if len(headers) == 0 {
		return nil
	}
	if headers[0].PreviousHash != localTip.Hash {
		// TODO Phase 2.5: block locator, common ancestor search, longest valid chain, safe reorg.
		return fmt.Errorf("fork detected: peer does not extend local tip\nlocal height: %d\nlocal tip: %s\npeer next height: %d\npeer previous hash: %s", localTip.Height, localTip.Hash, headers[0].Height, headers[0].PreviousHash)
	}
	prev := localTip
	for i, header := range headers {
		if header.Height != prev.Height+1 {
			return fmt.Errorf("invalid header height: got %d want %d", header.Height, prev.Height+1)
		}
		if header.Hash == "" {
			return errors.New("invalid header hash: empty")
		}
		if header.Difficulty > 16 {
			return fmt.Errorf("invalid header difficulty: %d", header.Difficulty)
		}
		if i > 0 && header.PreviousHash != headers[i-1].Hash {
			return fmt.Errorf("invalid header continuity at height %d", header.Height)
		}
		prev = types.Block{Height: header.Height, Hash: header.Hash}
	}
	return nil
}

func blockAtHeight(bc *chain.Blockchain, height uint64) *types.Block {
	blocks, err := bc.Blocks()
	if err != nil {
		return nil
	}
	for _, block := range blocks {
		if block.Height == height {
			copy := block
			return &copy
		}
	}
	return nil
}

func notePeerSuccess(paths config.Paths, peer string, hs Handshake, scoreDelta int, reason string) error {
	normalized, err := NormalizePeerURL(peer)
	if err == nil {
		peer = normalized
	}
	store := NewPeerStore(paths.Peers)
	peers, _ := store.LoadMetadata()
	score := 0
	lastReason := ""
	lastScoreAt := ""
	var existingMeta *PeerMetadata
	for i := range peers {
		existing := peers[i]
		if existing.URL == peer {
			// Keep the score deduplication behavior for repeated successful checks,
			// but do not return before refreshing liveness metadata. A peer that was
			// previously marked bad must still recover to active on an authenticated
			// successful check even when the score update is cooldown-deduplicated.
			score = existing.Score
			lastReason = existing.LastScoreReason
			lastScoreAt = existing.LastScoreAt
			existingMeta = &existing
			break
		}
	}
	meta := MetadataFromHandshake(peer, hs, score)
	// A successful authenticated handshake is an explicit recovery signal.
	// Do not let a previously negative reputation score reclassify the peer
	// back to bad during metadata refresh; score and liveness are separate.
	meta.Status = PeerStatusActive
	if existingMeta != nil {
		meta.FirstSeenAt = existingMeta.FirstSeenAt
		meta.Source = existingMeta.Source
		meta.FailureCount = existingMeta.FailureCount
		meta.LastFailureAt = existingMeta.LastFailureAt
		meta.SuccessCount = existingMeta.SuccessCount + 1
		meta.CooldownUntil = ""
	}
	meta.LastError = ""
	meta.LastScoreReason = lastReason
	meta.LastScoreAt = lastScoreAt
	if err := store.Upsert(meta); err != nil {
		return err
	}
	return store.AdjustPeerScore(peer, scoreDelta, reason)
}

func notePeerFailure(paths config.Paths, peer, errText string, scoreDelta int, reason string) error {
	normalized, err := NormalizePeerURL(peer)
	if err == nil {
		peer = normalized
	}
	store := NewPeerStore(paths.Peers)
	peers, _ := store.LoadMetadata()
	now := time.Now().Format(time.RFC3339)
	meta := PeerMetadata{URL: peer, Status: failureStatus(reason), LastError: errText, FirstSeenAt: now, LastFailureAt: now, FailureCount: 1}
	for _, existing := range peers {
		if existing.URL == peer {
			meta = existing
			meta.LastError = errText
			meta.Status = failureStatus(reason)
			meta.LastFailureAt = now
			meta.FailureCount++
			break
		}
	}
	if err := store.Upsert(meta); err != nil {
		return err
	}
	return store.AdjustPeerScore(peer, scoreDelta, reason)
}

func failureStatus(reason string) string {
	switch reason {
	case "request failed":
		return PeerStatusOffline
	case "handshake mismatch", "invalid block", "invalid header", "fork detected":
		return PeerStatusRejected
	default:
		return PeerStatusUnknown
	}
}

func RevalidateMempoolAgainstChain(paths config.Paths) error {
	// Compatibility wrapper for legacy callers that predate network profiles.
	return RevalidateMempoolAgainstChainWithProfile(paths, config.Localnet())
}

func RevalidateMempoolAgainstChainWithProfile(paths config.Paths, profile config.NetworkConfig) error {
	_, err := RevalidateMempoolAgainstLedgerWithProfile(paths, profile)
	return err
}

func removeBlockTxs(mempoolPath string, block types.Block) error {
	ids := make(map[string]struct{})
	for _, tx := range block.Transactions {
		if tx.Coinbase {
			continue
		}
		ids[tx.ID] = struct{}{}
	}
	return mempool.New(mempoolPath).RemoveIDs(ids)
}

func openChain(paths config.Paths) (*chain.Blockchain, func(), error) {
	return openChainWithProfile(paths, config.Localnet())
}

func openChainWithProfile(paths config.Paths, profile config.NetworkConfig) (*chain.Blockchain, func(), error) {
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		return nil, nil, err
	}
	bc := chain.New(store)
	if err := bc.InitWithProfile(profile); err != nil {
		_ = store.Close()
		return nil, nil, err
	}
	return bc, func() { _ = store.Close() }, nil
}
