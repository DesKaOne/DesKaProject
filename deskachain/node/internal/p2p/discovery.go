package p2p

import (
	"fmt"
	"time"

	"indochain/internal/chain"
	"indochain/internal/config"
)

type DiscoveryResult struct {
	Peer       string   `json:"peer"`
	Checked    int      `json:"checked"`
	Added      int      `json:"added"`
	Skipped    int      `json:"skipped"`
	Rejected   int      `json:"rejected"`
	Discovered []string `json:"discovered"`
	Errors     []string `json:"errors,omitempty"`
}

func DiscoverFromPeer(paths config.Paths, peer string, profile config.NetworkConfig, selfURL string, limit int) (DiscoveryResult, error) {
	return DiscoverFromPeerLimited(paths, peer, profile, selfURL, limit, DefaultMaxStoredPeers, DefaultMaxPeersPerIP)
}

func DiscoverFromPeerLimited(paths config.Paths, peer string, profile config.NetworkConfig, selfURL string, limit, maxPeers, maxPeersPerIP int) (DiscoveryResult, error) {
	if limit <= 0 || limit > DefaultMaxDiscoveredPeers {
		limit = DefaultMaxDiscoveredPeers
	}
	if maxPeers <= 0 {
		maxPeers = DefaultMaxStoredPeers
	}
	if maxPeersPerIP <= 0 {
		maxPeersPerIP = DefaultMaxPeersPerIP
	}
	normalized, err := NormalizePeerURL(peer)
	if err != nil {
		return DiscoveryResult{}, err
	}
	peer = normalized
	result := DiscoveryResult{Peer: peer}
	if profile.Name == "" {
		profile = config.Localnet()
	}
	local := profile
	local.GenesisHash = chain.GenesisHashForNetwork(profile)
	hs, err := CheckPeerWithProfile(paths, peer, profile)
	if err != nil {
		return result, err
	}
	client, err := NewClientForProfile(paths, profile, 3*time.Second)
	if err != nil {
		return result, err
	}
	response, err := client.Peers(peer)
	if err != nil {
		if len(hs.KnownPeers) == 0 {
			return result, err
		}
		response = PeersResponse{KnownPeers: hs.KnownPeers}
	}
	self := ""
	if selfURL != "" {
		if normalizedSelf, err := NormalizePeerURL(selfURL); err == nil {
			self = normalizedSelf
		}
	}
	store := NewPeerStore(paths.Peers)
	stored, err := store.LoadMetadata()
	if err != nil {
		return result, err
	}
	for _, candidate := range response.KnownPeers {
		if result.Checked >= limit {
			break
		}
		result.Checked++
		url, err := NormalizePeerURL(candidate.URL)
		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid url: %v", candidate.URL, err))
			continue
		}
		if url == peer || (self != "" && url == self) {
			result.Skipped++
			continue
		}
		if !peerCapacityAllows(stored, url, maxPeers, maxPeersPerIP) {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: peer capacity or subnet limit reached", url))
			continue
		}
		candidateHS, err := CheckPeerWithProfile(paths, url, profile)
		if err != nil {
			result.Rejected++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		if err := ValidateHandshake(local, candidateHS); err != nil {
			result.Rejected++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		stored, err = store.LoadMetadata()
		if err != nil {
			result.Rejected++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		if len(stored) >= maxPeers {
			alreadyKnown := false
			for _, existing := range stored {
				if existing.URL == url {
					alreadyKnown = true
					break
				}
			}
			if !alreadyKnown {
				result.Skipped++
				result.Errors = append(result.Errors, fmt.Sprintf("%s: peer store limit reached", url))
				continue
			}
		}
		meta := MetadataFromHandshake(url, candidateHS, 3)
		meta.Source = "discovered"
		if candidate.Source != "" {
			meta.Source = mergePeerSource(candidate.Source, "discovered")
		}
		if err := store.Upsert(meta); err != nil {
			result.Rejected++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		stored, _ = store.LoadMetadata()
		result.Added++
		result.Discovered = append(result.Discovered, url)
	}
	return result, nil
}

func peerCapacityAllows(peers []PeerMetadata, candidate string, maxPeers, maxPeersPerIP int) bool {
	known := false
	ipKey := peerIPKey(candidate)
	sameIP := 0
	for _, peer := range peers {
		if peer.URL == candidate {
			known = true
		}
		if ipKey != "" && peerIPKey(peer.URL) == ipKey {
			sameIP++
		}
	}
	if !known && maxPeers > 0 && len(peers) >= maxPeers {
		return false
	}
	if !known && maxPeersPerIP > 0 && ipKey != "" && sameIP >= maxPeersPerIP {
		return false
	}
	return true
}
