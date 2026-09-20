package p2p

import (
	"fmt"
	"strings"
	"time"

	"deskachain/internal/chain"
	"deskachain/internal/config"
)

func CheckFork(paths config.Paths, peer string) (ForkCheckResult, error) {
	return CheckForkWithProfile(paths, peer, config.Localnet())
}

func CheckForkWithProfile(paths config.Paths, peer string, profile config.NetworkConfig) (ForkCheckResult, error) {
	if err := ValidatePeerURL(peer); err != nil {
		return ForkCheckResult{}, err
	}
	client := NewClientWithTimeout(5 * time.Second)
	if profile.Name == "" {
		profile = config.Localnet()
	}
	localNet := profile
	localNet.GenesisHash = chain.GenesisHashForNetwork(profile)
	handshake, err := client.Handshake(peer)
	if err != nil {
		return ForkCheckResult{}, err
	}
	if err := ValidateHandshake(localNet, handshake); err != nil {
		return ForkCheckResult{}, err
	}
	status, err := client.Status(peer)
	if err != nil {
		return ForkCheckResult{}, err
	}
	if err := ValidateStatus(localNet, status); err != nil {
		return ForkCheckResult{}, err
	}
	bc, closeFn, err := openChainWithProfile(paths, profile)
	if err != nil {
		return ForkCheckResult{}, err
	}
	blocks, err := bc.Blocks()
	closeFn()
	if err != nil {
		return ForkCheckResult{}, err
	}
	tip := blocks[len(blocks)-1]
	result := ForkCheckResult{
		LocalHeight:    tip.Height,
		LocalTip:       tip.Hash,
		PeerHeight:     status.Height,
		PeerTip:        status.TipHash,
		ReorgSupported: false,
	}
	if tip.Height == status.Height && tip.Hash == status.TipHash {
		result.InSync = true
		result.Status = "in_sync"
		return result, nil
	}
	_, ancestor, err := CommonAncestorWithPeerAndProfile(paths, peer, profile)
	if err != nil {
		return result, err
	}
	if ancestor.Found {
		result.CommonAncestorFound = true
		result.CommonAncestorHeight = ancestor.Height
		result.CommonAncestorHash = ancestor.Hash
		if tip.Height >= ancestor.Height {
			result.LocalAheadBlocks = tip.Height - ancestor.Height
		}
		if status.Height >= ancestor.Height {
			result.PeerAheadBlocks = status.Height - ancestor.Height
		}
	}
	switch {
	case ancestor.Found && ancestor.Height == tip.Height && status.Height > tip.Height:
		result.Status = "peer_ahead"
	case ancestor.Found && ancestor.Height == status.Height && tip.Height > status.Height:
		result.Status = "local_ahead"
	default:
		result.ForkDetected = true
		result.Status = "fork"
	}
	return result, nil
}

type ForkError struct {
	Result ForkCheckResult
}

func (e *ForkError) Error() string {
	return formatForkSyncError(e.Result)
}

func (r ForkCheckResult) SyncError() error {
	if !r.ForkDetected {
		return nil
	}
	return &ForkError{Result: r}
}

func formatForkSyncError(r ForkCheckResult) string {
	var b strings.Builder
	fmt.Fprintln(&b, "sync failed: fork detected")
	fmt.Fprintf(&b, "local height: %d\n", r.LocalHeight)
	fmt.Fprintf(&b, "local tip: %s\n", r.LocalTip)
	fmt.Fprintf(&b, "peer height: %d\n", r.PeerHeight)
	fmt.Fprintf(&b, "peer tip: %s\n", r.PeerTip)
	if r.CommonAncestorFound {
		fmt.Fprintf(&b, "common ancestor height: %d\n", r.CommonAncestorHeight)
		fmt.Fprintf(&b, "common ancestor hash: %s\n", r.CommonAncestorHash)
	} else {
		fmt.Fprintln(&b, "common ancestor: not found")
	}
	fmt.Fprintln(&b, "automatic reorg: disabled")
	fmt.Fprintln(&b, "TODO Phase 2.5 automatic safe reorg")
	return strings.TrimSpace(b.String())
}
