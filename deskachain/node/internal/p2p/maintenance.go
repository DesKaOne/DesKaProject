package p2p

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"sort"
	"time"

	"indochain/internal/chain"
	"indochain/internal/config"
)

type MaintenanceOptions struct {
	SelfURL       string
	TTL           time.Duration
	Interval      time.Duration
	Limit         int
	MaxPeers      int
	MaxPeersPerIP int
	BackoffBase   time.Duration
	BackoffMax    time.Duration
}

type MaintenanceResult struct {
	Checked    int      `json:"checked"`
	Active     int      `json:"active"`
	Failed     int      `json:"failed"`
	Discovered int      `json:"discovered"`
	Pruned     int      `json:"pruned"`
	Skipped    int      `json:"skipped"`
	Errors     []string `json:"errors,omitempty"`
}

func DefaultMaintenanceOptions(maxPeers int) MaintenanceOptions {
	if maxPeers <= 0 {
		maxPeers = DefaultMaxStoredPeers
	}
	return MaintenanceOptions{
		TTL:           DefaultPeerTTL,
		Interval:      30 * time.Second,
		Limit:         DefaultMaxDiscoveredPeers,
		MaxPeers:      maxPeers,
		MaxPeersPerIP: DefaultMaxPeersPerIP,
		BackoffBase:   DefaultPeerBackoffBase,
		BackoffMax:    DefaultPeerBackoffMax,
	}
}

func RunMaintenanceWorker(ctx context.Context, paths config.Paths, profile config.NetworkConfig, opts MaintenanceOptions) {
	if opts.Interval <= 0 {
		opts.Interval = 30 * time.Second
	}
	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := MaintainPeers(paths, profile, opts)
			if err != nil {
				log.Printf("peer maintenance failed: %v", err)
				continue
			}
			if result.Checked > 0 || result.Discovered > 0 || result.Pruned > 0 || result.Failed > 0 {
				log.Printf("peer maintenance checked=%d active=%d failed=%d discovered=%d pruned=%d skipped=%d", result.Checked, result.Active, result.Failed, result.Discovered, result.Pruned, result.Skipped)
			}
		}
	}
}

func MaintainPeers(paths config.Paths, profile config.NetworkConfig, opts MaintenanceOptions) (MaintenanceResult, error) {
	if opts.TTL == 0 {
		opts.TTL = DefaultPeerTTL
	}
	if opts.Limit <= 0 || opts.Limit > DefaultMaxDiscoveredPeers {
		opts.Limit = DefaultMaxDiscoveredPeers
	}
	if opts.MaxPeers <= 0 {
		opts.MaxPeers = DefaultMaxStoredPeers
	}
	if opts.MaxPeersPerIP <= 0 {
		opts.MaxPeersPerIP = DefaultMaxPeersPerIP
	}
	if opts.BackoffBase <= 0 {
		opts.BackoffBase = DefaultPeerBackoffBase
	}
	if opts.BackoffMax <= 0 {
		opts.BackoffMax = DefaultPeerBackoffMax
	}
	if profile.Name == "" {
		profile = config.Localnet()
	}
	store := NewPeerStore(paths.Peers)
	now := time.Now()
	pruned, err := store.PruneExpired(opts.TTL, now)
	if err != nil {
		return MaintenanceResult{}, err
	}
	result := MaintenanceResult{Pruned: pruned}
	peers, err := store.LoadMetadata()
	if err != nil {
		return result, err
	}
	self := normalizedOrEmpty(opts.SelfURL)
	peers = uniqueMetadata(peers)
	sort.SliceStable(peers, func(i, j int) bool {
		return effectivePeerScore(peers[i]) > effectivePeerScore(peers[j])
	})
	for _, peer := range peers {
		if result.Checked >= opts.Limit {
			break
		}
		if peer.Status == PeerStatusRejected || peer.Status == PeerStatusBad {
			result.Skipped++
			continue
		}
		if self != "" && peer.URL == self {
			result.Skipped++
			continue
		}
		if !peerRetryDue(peer, now) {
			result.Skipped++
			continue
		}
		result.Checked++
		hs, err := CheckPeerWithProfile(paths, peer.URL, profile)
		if err != nil {
			result.Failed++
			if markErr := markPeerMaintenanceFailure(store, peer, err, now, opts); markErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", peer.URL, markErr))
			}
			continue
		}
		local := profile
		local.GenesisHash = chain.GenesisHashForNetwork(profile)
		if err := ValidateHandshake(local, hs); err != nil {
			result.Failed++
			if markErr := markPeerMaintenanceFailure(store, peer, err, now, opts); markErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", peer.URL, markErr))
			}
			continue
		}
		result.Active++
		meta := MetadataFromHandshake(peer.URL, hs, 2)
		meta.Source = mergePeerSource(peer.Source, "maintenance")
		meta.LastDiscoveryAt = now.Format(time.RFC3339)
		if err := store.Upsert(meta); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", peer.URL, err))
			continue
		}
		discovery, err := DiscoverFromPeerLimited(paths, peer.URL, profile, opts.SelfURL, opts.Limit, opts.MaxPeers, opts.MaxPeersPerIP)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", peer.URL, err))
			continue
		}
		result.Discovered += discovery.Added
		result.Skipped += discovery.Skipped
		result.Errors = append(result.Errors, discovery.Errors...)
	}
	return result, nil
}

func markPeerMaintenanceFailure(store PeerStore, peer PeerMetadata, err error, now time.Time, opts MaintenanceOptions) error {
	peers, loadErr := store.LoadMetadata()
	if loadErr != nil {
		return loadErr
	}
	for i := range peers {
		if peers[i].URL != peer.URL {
			continue
		}
		peers[i].LastFailureAt = now.Format(time.RFC3339)
		peers[i].FailureCount++
		peers[i].LastError = err.Error()
		peers[i].Status = PeerStatusOffline
		peers[i].LastScoreReason = "maintenance failed"
		peers[i].LastScoreAt = now.Format(time.RFC3339)
		peers[i].NextRetryAt = now.Add(peerBackoff(peers[i].FailureCount, opts.BackoffBase, opts.BackoffMax)).Format(time.RFC3339)
		return store.SaveMetadata(peers)
	}
	return nil
}

func peerRetryDue(peer PeerMetadata, now time.Time) bool {
	if peer.NextRetryAt == "" {
		return true
	}
	next, err := time.Parse(time.RFC3339, peer.NextRetryAt)
	if err != nil {
		return true
	}
	return !now.Before(next)
}

func peerBackoff(failures int, base, max time.Duration) time.Duration {
	if failures <= 0 {
		return 0
	}
	delay := base
	for i := 1; i < failures; i++ {
		delay *= 2
		if delay >= max {
			return max
		}
	}
	if delay > max {
		return max
	}
	return delay
}

func peerLastSeen(peer PeerMetadata) (time.Time, bool) {
	for _, value := range []string{peer.LastSeenAt, peer.LastSuccessAt, peer.FirstSeenAt} {
		if value == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func normalizedOrEmpty(peer string) string {
	if peer == "" {
		return ""
	}
	normalized, err := NormalizePeerURL(peer)
	if err != nil {
		return ""
	}
	return normalized
}

func peerIPKey(peerURL string) string {
	parsed, err := url.Parse(peerURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	if host == "" {
		return ""
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2])
	}
	return ip.String()
}
