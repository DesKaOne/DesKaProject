package p2p

import (
	"time"

	"indochain/internal/chain"
	"indochain/internal/config"
	"indochain/internal/types"
)

func LocalLocator(paths config.Paths) (LocatorResponse, error) {
	return LocalLocatorWithProfile(paths, config.Localnet())
}

func LocalLocatorWithProfile(paths config.Paths, profile config.NetworkConfig) (LocatorResponse, error) {
	bc, closeFn, err := openChainWithProfile(paths, profile)
	if err != nil {
		return LocatorResponse{}, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return LocatorResponse{}, err
	}
	return LocatorFromBlocks(blocks), nil
}

func LocatorFromBlocks(blocks []types.Block) LocatorResponse {
	tip := blocks[len(blocks)-1]
	return LocatorResponse{
		Height:  tip.Height,
		TipHash: tip.Hash,
		Locator: chain.BuildBlockLocator(blocks),
	}
}

func FindCommonAncestor(paths config.Paths, locator []chain.BlockLocatorEntry) (CommonAncestorResponse, error) {
	return FindCommonAncestorWithProfile(paths, locator, config.Localnet())
}

func FindCommonAncestorWithProfile(paths config.Paths, locator []chain.BlockLocatorEntry, profile config.NetworkConfig) (CommonAncestorResponse, error) {
	if len(locator) == 0 {
		return CommonAncestorResponse{Found: false, Error: "empty locator"}, nil
	}
	bc, closeFn, err := openChainWithProfile(paths, profile)
	if err != nil {
		return CommonAncestorResponse{}, err
	}
	defer closeFn()
	blocks, err := bc.Blocks()
	if err != nil {
		return CommonAncestorResponse{}, err
	}
	ancestor, ok := chain.FindCommonAncestor(blocks, locator)
	if !ok {
		return CommonAncestorResponse{Found: false, Error: "no common ancestor found"}, nil
	}
	return CommonAncestorResponse{Found: true, Height: ancestor.Height, Hash: ancestor.Hash}, nil
}

func CommonAncestorWithPeer(paths config.Paths, peer string) (LocatorResponse, CommonAncestorResponse, error) {
	return CommonAncestorWithPeerAndProfile(paths, peer, config.Localnet())
}

func CommonAncestorWithPeerAndProfile(paths config.Paths, peer string, profile config.NetworkConfig) (LocatorResponse, CommonAncestorResponse, error) {
	if err := ValidatePeerURL(peer); err != nil {
		return LocatorResponse{}, CommonAncestorResponse{}, err
	}
	locator, err := LocalLocatorWithProfile(paths, profile)
	if err != nil {
		return LocatorResponse{}, CommonAncestorResponse{}, err
	}
	client, err := NewClientForProfile(paths, profile, 5*time.Second)
	if err != nil {
		return LocatorResponse{}, CommonAncestorResponse{}, err
	}
	ancestor, err := client.CommonAncestor(peer, CommonAncestorRequest{Locator: locator.Locator})
	if err != nil {
		return locator, CommonAncestorResponse{}, err
	}
	return locator, ancestor, nil
}

func InspectDatadirFork(local, other config.Paths) (ForkInspectResult, error) {
	localLocator, err := LocalLocator(local)
	if err != nil {
		return ForkInspectResult{}, err
	}
	otherLocator, err := LocalLocator(other)
	if err != nil {
		return ForkInspectResult{}, err
	}
	result := ForkInspectResult{
		LocalHeight:    localLocator.Height,
		LocalTip:       localLocator.TipHash,
		OtherHeight:    otherLocator.Height,
		OtherTip:       otherLocator.TipHash,
		ReorgSupported: false,
	}
	if localLocator.Height == otherLocator.Height && localLocator.TipHash == otherLocator.TipHash {
		result.InSync = true
		result.Status = "in_sync"
		return result, nil
	}
	ancestor, err := FindCommonAncestor(other, localLocator.Locator)
	if err != nil {
		return result, err
	}
	if ancestor.Found {
		result.CommonAncestorFound = true
		result.CommonAncestorHeight = ancestor.Height
		result.CommonAncestorHash = ancestor.Hash
		if result.LocalHeight >= ancestor.Height {
			result.LocalAheadBlocks = result.LocalHeight - ancestor.Height
		}
		if result.OtherHeight >= ancestor.Height {
			result.OtherAheadBlocks = result.OtherHeight - ancestor.Height
		}
	}
	switch {
	case ancestor.Found && ancestor.Height == result.LocalHeight && result.OtherHeight > result.LocalHeight:
		result.Status = "other_ahead"
	case ancestor.Found && ancestor.Height == result.OtherHeight && result.LocalHeight > result.OtherHeight:
		result.Status = "local_ahead"
	default:
		result.ForkDetected = true
		result.Status = "fork"
	}
	return result, nil
}
