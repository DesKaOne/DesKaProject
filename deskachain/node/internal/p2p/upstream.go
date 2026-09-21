package p2p

import (
	"fmt"
	"log"
	"time"

	"indochain/internal/chain"
	"indochain/internal/config"
	"indochain/internal/types"
)

type UpstreamBackfillResult struct {
	Peer                 string `json:"peer"`
	OK                   bool   `json:"ok"`
	Decision             string `json:"decision"`
	Reason               string `json:"reason,omitempty"`
	LocalHeight          uint64 `json:"local_height"`
	LocalTip             string `json:"local_tip"`
	PeerHeight           uint64 `json:"peer_height"`
	PeerTip              string `json:"peer_tip"`
	LocalWork            uint64 `json:"local_work"`
	PeerWork             uint64 `json:"peer_work"`
	CommonAncestorHeight uint64 `json:"common_ancestor_height,omitempty"`
	CommonAncestorHash   string `json:"common_ancestor_hash,omitempty"`
	FromHeight           uint64 `json:"from_height,omitempty"`
	ToHeight             uint64 `json:"to_height,omitempty"`
	Pushed               int    `json:"pushed"`
	Accepted             int    `json:"accepted"`
	FailedHeight         uint64 `json:"failed_height,omitempty"`
}

func BackfillToPeers(paths config.Paths, peers []string, profile config.NetworkConfig, maxReorgDepth uint64) []UpstreamBackfillResult {
	results := make([]UpstreamBackfillResult, 0, len(peers))
	for _, peer := range peers {
		result, err := BackfillToPeer(paths, peer, profile, maxReorgDepth)
		if err != nil {
			result.OK = false
			result.Reason = err.Error()
			log.Printf("upstream backfill failed peer=%s reason=%q", peer, err.Error())
		}
		results = append(results, result)
	}
	return results
}

func BackfillToPeer(paths config.Paths, peer string, profile config.NetworkConfig, maxReorgDepth uint64) (UpstreamBackfillResult, error) {
	normalized, err := NormalizePeerURL(peer)
	if err != nil {
		return UpstreamBackfillResult{Peer: peer}, err
	}
	peer = normalized
	if profile.Name == "" {
		profile = config.Localnet()
	}
	if maxReorgDepth == 0 {
		maxReorgDepth = config.DefaultMaxReorgDepth(profile)
	}
	localNet := profile
	localNet.GenesisHash = chain.GenesisHashForNetwork(profile)
	client, err := NewClientForProfile(paths, profile, 5*time.Second)
	if err != nil {
		return UpstreamBackfillResult{Peer: peer}, err
	}
	hs, err := client.Handshake(peer)
	if err != nil {
		return UpstreamBackfillResult{Peer: peer}, err
	}
	if err := ValidateHandshake(localNet, hs); err != nil {
		return UpstreamBackfillResult{Peer: peer}, err
	}
	status, err := client.Status(peer)
	if err != nil {
		return UpstreamBackfillResult{Peer: peer}, err
	}
	if err := ValidateStatus(localNet, status); err != nil {
		return UpstreamBackfillResult{Peer: peer}, err
	}
	bc, closeFn, err := openChainWithProfile(paths, profile)
	if err != nil {
		return UpstreamBackfillResult{Peer: peer}, err
	}
	localBlocks, err := bc.Blocks()
	closeFn()
	if err != nil {
		return UpstreamBackfillResult{Peer: peer}, err
	}
	localTip := localBlocks[len(localBlocks)-1]
	result := UpstreamBackfillResult{
		Peer:        peer,
		LocalHeight: localTip.Height,
		LocalTip:    localTip.Hash,
		PeerHeight:  status.Height,
		PeerTip:     status.TipHash,
		LocalWork:   chain.CalculateCumulativeWork(localBlocks),
		PeerWork:    status.CumulativeWork,
	}
	if status.Height == localTip.Height && status.TipHash == localTip.Hash {
		result.OK = true
		result.Decision = "already_up_to_date"
		log.Printf("upstream already up to date peer=%s height=%d", peer, localTip.Height)
		return result, nil
	}
	if status.Height > localTip.Height || status.CumulativeWork > result.LocalWork {
		result.Decision = "local_sync_from_upstream"
		log.Printf("upstream ahead peer=%s local_height=%d peer_height=%d local_work=%d peer_work=%d decision=%s", peer, localTip.Height, status.Height, result.LocalWork, status.CumulativeWork, result.Decision)
		if status.CumulativeWork > result.LocalWork {
			if err := SyncFromPeerWithProfileAndMaxDepth(paths, peer, nil, profile, maxReorgDepth); err != nil {
				return result, err
			}
			result.OK = true
		}
		return result, nil
	}
	ancestor, ok, err := upstreamCommonAncestor(client, peer, localBlocks)
	if err != nil {
		return result, err
	}
	if !ok {
		result.Decision = "common_ancestor_not_found"
		return result, fmt.Errorf("common ancestor not found")
	}
	result.CommonAncestorHeight = ancestor.Height
	result.CommonAncestorHash = ancestor.Hash
	if status.CumulativeWork == result.LocalWork && status.TipHash != localTip.Hash {
		result.Decision = "no_force"
		result.Reason = "fork tie"
		log.Printf("upstream fork tie peer=%s local_work=%d peer_work=%d ancestor_height=%d ancestor_hash=%s decision=no_force", peer, result.LocalWork, status.CumulativeWork, ancestor.Height, ancestor.Hash)
		return result, nil
	}
	if status.CumulativeWork > result.LocalWork {
		result.Decision = "local_sync_from_upstream"
		if err := SyncFromPeerWithProfileAndMaxDepth(paths, peer, nil, profile, maxReorgDepth); err != nil {
			return result, err
		}
		result.OK = true
		return result, nil
	}
	if status.Height < ancestor.Height {
		result.Decision = "peer_locator_inconsistent"
		return result, fmt.Errorf("peer locator inconsistent")
	}
	return pushBlocksFromAncestor(client, peer, localBlocks, ancestor, result)
}

func upstreamCommonAncestor(client Client, peer string, localBlocks []types.Block) (types.Block, bool, error) {
	loc, err := client.Locator(peer)
	if err != nil {
		return types.Block{}, false, err
	}
	ancestor, ok := chain.FindCommonAncestor(localBlocks, loc.Locator)
	if !ok {
		return types.Block{}, false, nil
	}
	for _, block := range localBlocks {
		if block.Height == ancestor.Height && block.Hash == ancestor.Hash {
			return block, true, nil
		}
	}
	return types.Block{}, false, nil
}

func pushBlocksFromAncestor(client Client, peer string, localBlocks []types.Block, ancestor types.Block, result UpstreamBackfillResult) (UpstreamBackfillResult, error) {
	from := ancestor.Height + 1
	result.FromHeight = from
	result.ToHeight = result.LocalHeight
	if from > result.LocalHeight {
		result.OK = true
		result.Decision = "already_up_to_date"
		return result, nil
	}
	result.Decision = "backfill_push"
	for _, block := range localBlocks {
		if block.Height < from {
			continue
		}
		response, err := client.BroadcastBlock(peer, block)
		if err != nil {
			result.FailedHeight = block.Height
			return result, err
		}
		result.Pushed++
		if !response.Accepted {
			result.FailedHeight = block.Height
			return result, fmt.Errorf("height %d rejected: %s", block.Height, response.Error)
		}
		result.Accepted++
	}
	result.OK = true
	log.Printf("upstream backfill complete peer=%s from_height=%d to_height=%d pushed=%d accepted=%d", peer, result.FromHeight, result.ToHeight, result.Pushed, result.Accepted)
	return result, nil
}
