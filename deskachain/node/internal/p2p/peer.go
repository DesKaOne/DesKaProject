package p2p

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	PeerStatusActive   = "active"
	PeerStatusBad      = "bad"
	PeerStatusCooldown = "cooldown"
	PeerStatusOffline  = "offline"
	PeerStatusRejected = "rejected"
	PeerStatusUnknown  = "unknown"

	PeerScoreMax                = 100
	PeerScoreMin                = -100
	PeerScoreBadThreshold       = -20
	PeerScoreExcellentThreshold = 50
	DefaultMaxDiscoveredPeers   = 16
	DefaultMaxStoredPeers       = 128
	DefaultPeerTTL              = 7 * 24 * time.Hour
	DefaultPeerBackoffBase      = 30 * time.Second
	DefaultPeerBackoffMax       = 30 * time.Minute
	DefaultMaxPeersPerIP        = 4
)

type PeerStore struct {
	path string
}

type PeerMetadata struct {
	URL               string `json:"url"`
	NodeID            string `json:"node_id"`
	NetworkID         string `json:"network_id"`
	ChainID           uint64 `json:"chain_id"`
	GenesisHash       string `json:"genesis_hash,omitempty"`
	ProtocolVersion   uint32 `json:"protocol_version,omitempty"`
	LastHeight        uint64 `json:"last_height"`
	LastTipHash       string `json:"last_tip_hash"`
	LastSeenAt        string `json:"last_seen_at"`
	FirstSeenAt       string `json:"first_seen,omitempty"`
	LastSuccessAt     string `json:"last_success,omitempty"`
	LastFailureAt     string `json:"last_failure,omitempty"`
	FailureCount      int    `json:"failure_count,omitempty"`
	SuccessCount      int    `json:"success_count,omitempty"`
	CooldownUntil     string `json:"cooldown_until,omitempty"`
	Status            string `json:"status"`
	Score             int    `json:"score"`
	Source            string `json:"source,omitempty"`
	LastError         string `json:"last_error,omitempty"`
	LastScoreReason   string `json:"last_score_reason,omitempty"`
	LastScoreAt       string `json:"last_score_at,omitempty"`
	LastLatencyMS     int64  `json:"last_latency_ms,omitempty"`
	LastStatusCheckAt string `json:"last_status_check_at,omitempty"`
	Version           string `json:"version,omitempty"`
	Protocol          string `json:"protocol,omitempty"`
	Services          string `json:"services,omitempty"`
	NextRetryAt       string `json:"next_retry_at,omitempty"`
	LastDiscoveryAt   string `json:"last_discovery_at,omitempty"`
}

type peerFile struct {
	Peers []PeerMetadata `json:"peers"`
}

func NewPeerStore(path string) PeerStore {
	return PeerStore{path: path}
}

func (s PeerStore) Load() ([]string, error) {
	peers, err := s.LoadMetadata()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(peers))
	for _, peer := range peers {
		out = append(out, peer.URL)
	}
	return uniquePeers(out), nil
}

func (s PeerStore) LoadMetadata() ([]PeerMetadata, error) {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var modern peerFile
	if err := json.Unmarshal(raw, &modern); err == nil && len(modern.Peers) > 0 {
		return uniqueMetadata(modern.Peers), nil
	}
	var legacy struct {
		Peers []string `json:"peers"`
	}
	if err := json.Unmarshal(raw, &legacy); err == nil {
		peers := make([]PeerMetadata, 0, len(legacy.Peers))
		for _, peer := range legacy.Peers {
			peers = append(peers, PeerMetadata{URL: peer, Status: PeerStatusUnknown})
		}
		return uniqueMetadata(peers), nil
	}
	backup := s.path + ".corrupt-" + time.Now().Format("20060102150405")
	_ = os.Rename(s.path, backup)
	return nil, nil
}

func (s PeerStore) Save(peers []string) error {
	meta := make([]PeerMetadata, 0, len(peers))
	for _, peer := range peers {
		meta = append(meta, PeerMetadata{URL: peer, Status: PeerStatusUnknown})
	}
	return s.SaveMetadata(meta)
}

func (s PeerStore) SaveMetadata(peers []PeerMetadata) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(peerFile{Peers: uniqueMetadata(peers)}, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), "peers-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	_ = os.Remove(s.path)
	return os.Rename(tmpName, s.path)
}

func (s PeerStore) Add(peer string) error {
	return s.AddWithSource(peer, "")
}

func (s PeerStore) AddWithSource(peer, source string) error {
	normalized, err := NormalizePeerURL(peer)
	if err != nil {
		return err
	}
	peer = normalized
	peers, err := s.LoadMetadata()
	if err != nil {
		return err
	}
	for _, existing := range peers {
		if existing.URL == peer {
			if source != "" {
				existing.Source = mergePeerSource(existing.Source, source)
				return s.Upsert(existing)
			}
			return nil
		}
	}
	now := time.Now().Format(time.RFC3339)
	return s.SaveMetadata(append(peers, PeerMetadata{URL: peer, Status: PeerStatusUnknown, Source: source, FirstSeenAt: now}))
}

func (s PeerStore) Upsert(peer PeerMetadata) error {
	normalized, err := NormalizePeerURL(peer.URL)
	if err != nil {
		return err
	}
	peer.URL = normalized
	peer.Score = clampScore(peer.Score)
	peer.Status = statusForScore(peer.Score, peer.Status)
	peers, err := s.LoadMetadata()
	if err != nil {
		return err
	}
	for i := range peers {
		if peers[i].URL == peer.URL {
			if peer.FirstSeenAt == "" {
				peer.FirstSeenAt = peers[i].FirstSeenAt
			}
			if peer.Source == "" {
				peer.Source = peers[i].Source
			}
			peers[i] = peer
			return s.SaveMetadata(peers)
		}
	}
	return s.SaveMetadata(append(peers, peer))
}

func mergePeerSource(existing, incoming string) string {
	if existing == "" {
		existing = "peers.json"
	}
	if incoming == "" || existing == incoming || strings.Contains(existing, incoming) {
		return existing
	}
	return existing + " + " + incoming
}

func (s PeerStore) AdjustPeerScore(peerURL string, delta int, reason string) error {
	normalized, err := NormalizePeerURL(peerURL)
	if err != nil {
		return err
	}
	peerURL = normalized
	peers, err := s.LoadMetadata()
	if err != nil {
		return err
	}
	now := time.Now()
	for i := range peers {
		if peers[i].URL != peerURL {
			continue
		}
		if reasonRequiresCooldown(reason) && peers[i].LastScoreReason == reason && !scoreCooldownElapsed(peers[i].LastScoreAt, now) {
			return nil
		}
		peers[i].Score = clampScore(peers[i].Score + delta)
		peers[i].Status = statusForScore(peers[i].Score, peers[i].Status)
		if reasonTriggersCooldown(reason, peers[i].Score) {
			peers[i].CooldownUntil = now.Add(time.Minute).Format(time.RFC3339)
			peers[i].Status = PeerStatusCooldown
		}
		peers[i].LastScoreReason = reason
		peers[i].LastScoreAt = now.Format(time.RFC3339)
		return s.SaveMetadata(peers)
	}
	meta := PeerMetadata{URL: peerURL, Status: PeerStatusUnknown}
	meta.Score = clampScore(delta)
	meta.Status = statusForScore(meta.Score, PeerStatusActive)
	if reasonTriggersCooldown(reason, meta.Score) {
		meta.CooldownUntil = now.Add(time.Minute).Format(time.RFC3339)
		meta.Status = PeerStatusCooldown
	}
	meta.LastScoreReason = reason
	meta.LastScoreAt = now.Format(time.RFC3339)
	return s.SaveMetadata(append(peers, meta))
}

func (s PeerStore) UpdateLatency(peerURL string, latencyMS int64, errText string) error {
	normalized, err := NormalizePeerURL(peerURL)
	if err != nil {
		return err
	}
	peerURL = normalized
	peers, err := s.LoadMetadata()
	if err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	for i := range peers {
		if peers[i].URL != peerURL {
			continue
		}
		peers[i].LastLatencyMS = latencyMS
		peers[i].LastStatusCheckAt = now
		peers[i].LastError = errText
		if errText == "" && peers[i].Status != PeerStatusBad {
			peers[i].Status = PeerStatusActive
			peers[i].LastSuccessAt = now
			peers[i].SuccessCount++
		} else if errText != "" && peers[i].Status != PeerStatusBad && peers[i].Status != PeerStatusRejected {
			peers[i].Status = PeerStatusOffline
			peers[i].LastFailureAt = now
			peers[i].FailureCount++
		}
		delta, reason := latencyScoreDelta(latencyMS, errText)
		if delta != 0 {
			peers[i].Score = clampScore(peers[i].Score + delta)
			peers[i].Status = statusForScore(peers[i].Score, peers[i].Status)
			peers[i].LastScoreReason = reason
			peers[i].LastScoreAt = now
			if reasonTriggersCooldown(reason, peers[i].Score) {
				peers[i].CooldownUntil = time.Now().Add(time.Minute).Format(time.RFC3339)
				peers[i].Status = PeerStatusCooldown
			}
		}
		return s.SaveMetadata(peers)
	}
	status := PeerStatusUnknown
	if errText != "" {
		status = PeerStatusOffline
	}
	meta := PeerMetadata{URL: peerURL, Status: status, FirstSeenAt: now, LastLatencyMS: latencyMS, LastStatusCheckAt: now, LastError: errText}
	if errText == "" {
		meta.Status = PeerStatusActive
		meta.LastSuccessAt = now
		meta.SuccessCount = 1
	} else {
		meta.LastFailureAt = now
		meta.FailureCount = 1
	}
	delta, reason := latencyScoreDelta(latencyMS, errText)
	meta.Score = clampScore(delta)
	meta.Status = statusForScore(meta.Score, meta.Status)
	meta.LastScoreReason = reason
	meta.LastScoreAt = now
	if reasonTriggersCooldown(reason, meta.Score) {
		meta.CooldownUntil = time.Now().Add(time.Minute).Format(time.RFC3339)
		meta.Status = PeerStatusCooldown
	}
	return s.SaveMetadata(append(peers, meta))
}

func (s PeerStore) Remove(peer string) error {
	normalized, err := NormalizePeerURL(peer)
	if err != nil {
		return err
	}
	peer = normalized
	peers, err := s.LoadMetadata()
	if err != nil {
		return err
	}
	filtered := make([]PeerMetadata, 0, len(peers))
	for _, existing := range peers {
		if existing.URL != peer {
			filtered = append(filtered, existing)
		}
	}
	return s.SaveMetadata(filtered)
}

func (s PeerStore) Clear() error {
	return s.SaveMetadata(nil)
}

func (s PeerStore) PruneExpired(ttl time.Duration, now time.Time) (int, error) {
	if ttl <= 0 {
		return 0, nil
	}
	peers, err := s.LoadMetadata()
	if err != nil {
		return 0, err
	}
	kept := make([]PeerMetadata, 0, len(peers))
	pruned := 0
	for _, peer := range peers {
		if peer.Source == "seed" || strings.Contains(peer.Source, "seed") {
			kept = append(kept, peer)
			continue
		}
		lastSeen, ok := peerLastSeen(peer)
		if !ok || now.Sub(lastSeen) <= ttl {
			kept = append(kept, peer)
			continue
		}
		pruned++
	}
	if pruned == 0 {
		return 0, nil
	}
	return pruned, s.SaveMetadata(kept)
}

func MetadataFromHandshake(url string, hs Handshake, scoreDelta int) PeerMetadata {
	normalized, err := NormalizePeerURL(url)
	if err == nil {
		url = normalized
	}
	score := clampScore(scoreDelta)
	services := strings.Join(hs.Services, ",")
	if services == "" {
		services = "p2p"
	}
	return PeerMetadata{
		URL:             url,
		NodeID:          hs.NodeID,
		NetworkID:       hs.NetworkID,
		ChainID:         hs.ChainID,
		GenesisHash:     hs.GenesisHash,
		ProtocolVersion: hs.ProtocolVersion,
		LastHeight:      hs.Height,
		LastTipHash:     hs.TipHash,
		LastSeenAt:      time.Now().Format(time.RFC3339),
		FirstSeenAt:     time.Now().Format(time.RFC3339),
		LastSuccessAt:   time.Now().Format(time.RFC3339),
		SuccessCount:    1,
		Status:          statusForScore(score, PeerStatusActive),
		Score:           score,
		Version:         hs.NetworkName,
		Protocol:        hs.P2PProtocolVersion,
		Services:        services,
	}
}

func PeerViewFromMetadata(peer PeerMetadata) PeerView {
	return PeerView{
		URL:             peer.URL,
		Source:          peer.Source,
		Status:          currentStatus(peer),
		Score:           peer.Score,
		LastHeight:      peer.LastHeight,
		LastTipHash:     peer.LastTipHash,
		LastNetworkID:   peer.NetworkID,
		LastChainID:     peer.ChainID,
		LastGenesisHash: peer.GenesisHash,
		LastSuccess:     peer.LastSuccessAt,
		LastFailure:     peer.LastFailureAt,
		FailureCount:    peer.FailureCount,
		SuccessCount:    peer.SuccessCount,
		LastLatencyMS:   peer.LastLatencyMS,
		CooldownUntil:   peer.CooldownUntil,
		LastError:       peer.LastError,
		Version:         peer.Version,
		Protocol:        peer.Protocol,
		Services:        peer.Services,
		NextRetryAt:     peer.NextRetryAt,
	}
}

func PeerViews(peers []PeerMetadata, limit int) []PeerView {
	if limit <= 0 || limit > DefaultMaxDiscoveredPeers {
		limit = DefaultMaxDiscoveredPeers
	}
	out := make([]PeerView, 0, len(peers))
	for _, peer := range SelectPeers(peers, true, limit) {
		out = append(out, PeerViewFromMetadata(peer))
	}
	return out
}

func ReputationViews(peers []PeerMetadata) []PeerReputation {
	unique := uniqueMetadata(peers)
	sort.SliceStable(unique, func(i, j int) bool {
		return effectivePeerScore(unique[i]) > effectivePeerScore(unique[j])
	})
	out := make([]PeerReputation, 0, len(unique))
	for _, peer := range unique {
		out = append(out, PeerReputationFromMetadata(peer))
	}
	return out
}

func PeerReputationFromMetadata(peer PeerMetadata) PeerReputation {
	latency := latencyReputationScore(peer.LastLatencyMS)
	uptime := uptimeReputationScore(peer)
	reliability := reliabilityReputationScore(peer)
	penalty := failurePenalty(peer)
	return PeerReputation{
		URL:              peer.URL,
		Source:           peer.Source,
		Status:           currentStatus(peer),
		Score:            peer.Score,
		EffectiveScore:   effectivePeerScore(peer),
		UptimeScore:      uptime,
		LatencyScore:     latency,
		ReliabilityScore: reliability,
		FailurePenalty:   penalty,
		SuccessCount:     peer.SuccessCount,
		FailureCount:     peer.FailureCount,
		LastLatencyMS:    peer.LastLatencyMS,
		LastSuccess:      peer.LastSuccessAt,
		LastFailure:      peer.LastFailureAt,
		CooldownUntil:    peer.CooldownUntil,
		LastScoreReason:  peer.LastScoreReason,
		LastError:        peer.LastError,
	}
}

func SelectPeers(peers []PeerMetadata, includeCooldown bool, limit int) []PeerMetadata {
	filtered := make([]PeerMetadata, 0, len(peers))
	for _, peer := range uniqueMetadata(peers) {
		if peer.Status == PeerStatusBad || peer.Status == PeerStatusRejected || peer.Status == PeerStatusOffline {
			continue
		}
		if !includeCooldown && currentStatus(peer) == PeerStatusCooldown {
			continue
		}
		filtered = append(filtered, peer)
	}
	rand.Shuffle(len(filtered), func(i, j int) {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	})
	sort.SliceStable(filtered, func(i, j int) bool {
		return effectivePeerScore(filtered[i]) > effectivePeerScore(filtered[j])
	})
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

func ValidatePeerURL(peer string) error {
	_, err := NormalizePeerURL(peer)
	return err
}

func NormalizePeerURL(peer string) (string, error) {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		return "", errors.New("peer url is required")
	}
	parsed, err := url.Parse(peer)
	if err != nil {
		return "", err
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", errors.New("peer url must be http(s)://host:port")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("peer url must not include path, query, or fragment")
	}
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func MergePeers(groups ...[]string) []string {
	var merged []string
	for _, group := range groups {
		merged = append(merged, group...)
	}
	return uniquePeers(merged)
}

func uniquePeers(peers []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(peers))
	for _, peer := range peers {
		normalized, err := NormalizePeerURL(peer)
		if err != nil {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func uniqueMetadata(peers []PeerMetadata) []PeerMetadata {
	seen := make(map[string]struct{})
	out := make([]PeerMetadata, 0, len(peers))
	for _, peer := range peers {
		normalized, err := NormalizePeerURL(peer.URL)
		if err != nil {
			continue
		}
		peer.URL = normalized
		if _, ok := seen[peer.URL]; ok {
			continue
		}
		if peer.Status == "" {
			peer.Status = PeerStatusUnknown
		}
		if peer.CooldownUntil != "" {
			if until, parseErr := time.Parse(time.RFC3339, peer.CooldownUntil); parseErr == nil && !time.Now().Before(until) {
				peer.CooldownUntil = ""
				if peer.Status == PeerStatusCooldown {
					peer.Status = PeerStatusUnknown
				}
			}
		}
		if peer.Source == "" {
			peer.Source = "peers.json"
		}
		peer.Score = clampScore(peer.Score)
		peer.Status = currentStatus(peer)
		if peer.FirstSeenAt == "" {
			peer.FirstSeenAt = peer.LastSeenAt
		}
		seen[peer.URL] = struct{}{}
		out = append(out, peer)
	}
	return out
}

func clampScore(score int) int {
	if score > PeerScoreMax {
		return PeerScoreMax
	}
	if score < PeerScoreMin {
		return PeerScoreMin
	}
	return score
}

func statusForScore(score int, fallback string) string {
	if score <= PeerScoreBadThreshold {
		return PeerStatusBad
	}
	if fallback == "" || fallback == PeerStatusBad || fallback == PeerStatusCooldown {
		return PeerStatusActive
	}
	return fallback
}

func currentStatus(peer PeerMetadata) string {
	if peer.CooldownUntil != "" {
		until, err := time.Parse(time.RFC3339, peer.CooldownUntil)
		if err == nil && time.Now().Before(until) {
			return PeerStatusCooldown
		}
		if peer.Status == PeerStatusCooldown {
			peer.Status = PeerStatusUnknown
		}
	}
	return statusForScore(peer.Score, peer.Status)
}

func reasonTriggersCooldown(reason string, score int) bool {
	if score <= PeerScoreBadThreshold {
		return false
	}
	switch reason {
	case "request failed", "handshake mismatch", "invalid block", "invalid header", "fork detected", "latency failed", "latency high", "maintenance failed":
		return score <= PeerScoreBadThreshold/2
	default:
		return false
	}
}

func reasonRequiresCooldown(reason string) bool {
	return reason == "sync up to date" || reason == "peer status ok" || reason == "latency excellent" || reason == "latency ok"
}

func scoreCooldownElapsed(value string, now time.Time) bool {
	if value == "" {
		return true
	}
	last, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return true
	}
	return now.Sub(last) >= time.Minute
}

func latencyScoreDelta(latencyMS int64, errText string) (int, string) {
	if errText != "" {
		return -3, "latency failed"
	}
	switch {
	case latencyMS <= 0:
		return 0, ""
	case latencyMS <= 100:
		return 2, "latency excellent"
	case latencyMS <= 500:
		return 1, "latency ok"
	case latencyMS > 2000:
		return -2, "latency high"
	default:
		return 0, ""
	}
}

func effectivePeerScore(peer PeerMetadata) int {
	return clampScore(peer.Score + latencyReputationScore(peer.LastLatencyMS) + uptimeReputationScore(peer) + reliabilityReputationScore(peer) - failurePenalty(peer))
}

func latencyReputationScore(latencyMS int64) int {
	switch {
	case latencyMS <= 0:
		return 0
	case latencyMS <= 100:
		return 10
	case latencyMS <= 500:
		return 5
	case latencyMS <= 2000:
		return 0
	default:
		return -5
	}
}

func uptimeReputationScore(peer PeerMetadata) int {
	if currentStatus(peer) != PeerStatusActive || peer.LastSuccessAt == "" {
		return 0
	}
	last, err := time.Parse(time.RFC3339, peer.LastSuccessAt)
	if err != nil {
		return 0
	}
	age := time.Since(last)
	switch {
	case age <= time.Hour:
		return 8
	case age <= 24*time.Hour:
		return 4
	default:
		return 1
	}
}

func reliabilityReputationScore(peer PeerMetadata) int {
	total := peer.SuccessCount + peer.FailureCount
	if total == 0 {
		return 0
	}
	return clamp(peer.SuccessCount*4-peer.FailureCount*6, -50, 50)
}

func failurePenalty(peer PeerMetadata) int {
	if peer.FailureCount <= 0 {
		return 0
	}
	penalty := peer.FailureCount * 2
	if peer.LastError != "" {
		penalty += 3
	}
	return clamp(penalty, 0, 40)
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
