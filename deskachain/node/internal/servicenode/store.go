package servicenode

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"indochain/internal/chain"
	"indochain/internal/config"
	"indochain/internal/crypto"
	"indochain/internal/staking"
	"indochain/internal/storage"
)

const (
	challengeTTL        = 5 * time.Minute
	rewardWeight        = 10
	maxRecentItems      = 32
	impossibleBandwidth = int64(1_000_000_000)
)

var fileLocks sync.Map

type Store struct {
	paths   config.Paths
	profile config.NetworkConfig
	now     func() time.Time
}

func NewStore(paths config.Paths, profile config.NetworkConfig) Store {
	return Store{paths: paths, profile: profile, now: time.Now}
}

func (s Store) WithNow(now func() time.Time) Store {
	s.now = now
	return s
}

func (s Store) Register(address, endpoint string, meta Metadata) (Node, error) {
	if err := crypto.ValidateAddressForNetwork(address, s.profile); err != nil {
		return Node{}, err
	}
	// TODO: require a future signature-based registration proof from owner_address.
	nodes, err := s.LoadNodes()
	if err != nil {
		return Node{}, err
	}
	now := s.now().Unix()
	for i := range nodes {
		if nodes[i].OwnerAddress != address {
			continue
		}
		if endpoint != "" && endpoint != nodes[i].AdvertisedEndpoint {
			nodes[i].EndpointChanges++
			if nodes[i].EndpointChanges >= 3 {
				addFlag(&nodes[i], FlagEndpointChangedTooOften, 20)
			}
		}
		nodes[i].AdvertisedEndpoint = endpoint
		nodes[i].Metadata = meta
		nodes[i].Status = StatusRegistered
		if nodes[i].Mode == "" {
			nodes[i].Mode = ModeLocal
		}
		if err := s.SaveNodes(nodes); err != nil {
			return Node{}, err
		}
		return nodes[i], nil
	}
	node := Node{
		ServiceNodeID:      serviceNodeID(address),
		OwnerAddress:       address,
		AdvertisedEndpoint: endpoint,
		Mode:               ModeLocal,
		CreatedAt:          now,
		LastSeenAt:         now,
		Status:             StatusRegistered,
		Metadata:           meta,
	}
	nodes = append(nodes, node)
	if err := s.SaveNodes(nodes); err != nil {
		return Node{}, err
	}
	return node, nil
}

func (s Store) Heartbeat(address, endpoint string, meta Metadata) (Score, Node, error) {
	if err := crypto.ValidateAddressForNetwork(address, s.profile); err != nil {
		return Score{}, Node{}, err
	}
	nodes, err := s.LoadNodes()
	if err != nil {
		return Score{}, Node{}, err
	}
	now := s.now().Unix()
	for i := range nodes {
		if nodes[i].OwnerAddress != address {
			continue
		}
		if endpoint != "" && endpoint != nodes[i].AdvertisedEndpoint {
			nodes[i].EndpointChanges++
			if nodes[i].EndpointChanges >= 3 {
				addFlag(&nodes[i], FlagEndpointChangedTooOften, 20)
			}
			nodes[i].AdvertisedEndpoint = endpoint
		}
		if nodes[i].LastSeenAt > 0 && now-nodes[i].LastSeenAt < 2 {
			addFlag(&nodes[i], FlagHeartbeatSpam, 10)
		}
		nodes[i].LastSeenAt = now
		nodes[i].Status = StatusActive
		nodes[i].Metadata = meta
		nodes[i].Heartbeats = appendRecent(nodes[i].Heartbeats, now)
		if err := s.SaveNodes(nodes); err != nil {
			return Score{}, Node{}, err
		}
		score, err := s.Score(address)
		if err != nil {
			return Score{}, Node{}, err
		}
		return score, nodes[i], nil
	}
	return Score{}, Node{}, fmt.Errorf("service node not registered for address %s", address)
}

func (s Store) CreateChallenge(address string) (Challenge, error) {
	if err := crypto.ValidateAddressForNetwork(address, s.profile); err != nil {
		return Challenge{}, err
	}
	if _, ok, err := s.FindNode(address); err != nil {
		return Challenge{}, err
	} else if !ok {
		return Challenge{}, fmt.Errorf("service node not registered for address %s", address)
	}
	challenges, err := s.LoadChallenges()
	if err != nil {
		return Challenge{}, err
	}
	now := s.now()
	challenge := Challenge{
		ChallengeID:  randomHex(16),
		Address:      address,
		IssuedAt:     now.Unix(),
		ExpiresAt:    now.Add(challengeTTL).Unix(),
		Nonce:        randomHex(16),
		ExpectedMode: ModeLocal,
		Status:       ChallengePending,
	}
	challenges = append(challenges, challenge)
	if err := s.SaveChallenges(challenges); err != nil {
		return Challenge{}, err
	}
	return challenge, nil
}

func (s Store) SubmitChallenge(id string, latencyMS, bytesUp, bytesDown int64, success bool) (Score, Challenge, error) {
	if id == "" {
		return Score{}, Challenge{}, errors.New("challenge_id is required")
	}
	if latencyMS < 0 || bytesUp < 0 || bytesDown < 0 {
		return Score{}, Challenge{}, errors.New("latency and byte values must be non-negative")
	}
	challenges, err := s.LoadChallenges()
	if err != nil {
		return Score{}, Challenge{}, err
	}
	now := s.now().Unix()
	for i := range challenges {
		if challenges[i].ChallengeID != id {
			continue
		}
		if challenges[i].Status != ChallengePending {
			return Score{}, Challenge{}, fmt.Errorf("challenge already %s", challenges[i].Status)
		}
		nodes, err := s.LoadNodes()
		if err != nil {
			return Score{}, Challenge{}, err
		}
		nodeIndex := -1
		for j := range nodes {
			if nodes[j].OwnerAddress == challenges[i].Address {
				nodeIndex = j
				break
			}
		}
		if nodeIndex < 0 {
			return Score{}, Challenge{}, fmt.Errorf("service node not registered for address %s", challenges[i].Address)
		}
		if now > challenges[i].ExpiresAt {
			challenges[i].Status = ChallengeExpired
			addFlag(&nodes[nodeIndex], FlagChallengeExpired, 15)
			_ = s.SaveNodes(nodes)
			_ = s.SaveChallenges(challenges)
			return Score{}, challenges[i], errors.New("challenge expired")
		}
		sample := Sample{CreatedAt: now, LatencyMS: latencyMS, BytesUp: bytesUp, BytesDown: bytesDown, Success: success}
		challenges[i].Result = &sample
		if success {
			challenges[i].Status = ChallengePassed
		} else {
			challenges[i].Status = ChallengeFailed
			addFlag(&nodes[nodeIndex], FlagChallengeFailed, 15)
		}
		if bytesUp+bytesDown > impossibleBandwidth {
			addFlag(&nodes[nodeIndex], FlagImpossibleBandwidth, 40)
		}
		if repeatedSample(nodes[nodeIndex].Samples, sample) {
			addFlag(&nodes[nodeIndex], FlagRepeatedIdenticalSamples, 10)
		}
		nodes[nodeIndex].Samples = appendRecentSample(nodes[nodeIndex].Samples, sample)
		if err := s.SaveNodes(nodes); err != nil {
			return Score{}, Challenge{}, err
		}
		if err := s.SaveChallenges(challenges); err != nil {
			return Score{}, Challenge{}, err
		}
		score, err := s.Score(challenges[i].Address)
		if err != nil {
			return Score{}, Challenge{}, err
		}
		return score, challenges[i], nil
	}
	return Score{}, Challenge{}, fmt.Errorf("challenge not found: %s", id)
}

func (s Store) Score(address string) (Score, error) {
	node, ok, err := s.FindNode(address)
	if err != nil {
		return Score{}, err
	}
	if !ok {
		return Score{}, fmt.Errorf("service node not registered for address %s", address)
	}
	challenges, err := s.LoadChallenges()
	if err != nil {
		return Score{}, err
	}
	uptime := uptimeScore(s.now().Unix(), node.LastSeenAt)
	latency := 0
	bandwidth := 0
	if len(node.Samples) > 0 {
		last := node.Samples[len(node.Samples)-1]
		latency = latencyScore(last.LatencyMS)
		bandwidth = bandwidthScore(last.BytesUp + last.BytesDown)
	}
	reliability := reliabilityScore(address, challenges)
	penalty := clamp(node.AbusePenalty, 0, 100)
	raw := (uptime*30 + latency*20 + bandwidth*30 + reliability*20) / 100
	final := clamp(raw-penalty, 0, 100)
	requiredStake, activeStake, stakeEligible, collateralStatus := s.collateral(address)
	eligiblePoints := final * rewardWeight
	eligibilityNote := ""
	if s.profile.Consensus.Staking.RequireStakeForServiceRewards && !stakeEligible {
		eligiblePoints = 0
		eligibilityNote = "not eligible for service reward simulation until active stake >= required stake"
	}
	return Score{
		Address:                 address,
		UptimeScore:             uptime,
		LatencyScore:            latency,
		BandwidthScore:          bandwidth,
		ReliabilityScore:        reliability,
		AbusePenalty:            penalty,
		ServiceScore:            final,
		Flags:                   append([]string(nil), node.Flags...),
		SimulatedPoints:         final * rewardWeight,
		EligibleSimulatedPoints: eligiblePoints,
		RequiredStake:           requiredStake,
		ActiveStake:             activeStake,
		StakeEligible:           stakeEligible,
		CollateralStatus:        collateralStatus,
		EligibilityNote:         eligibilityNote,
		Note:                    "service points are simulation only and are not spendable dIDR",
	}, nil
}

func (s Store) Rewards(address string) ([]Reward, error) {
	score, err := s.Score(address)
	if err != nil {
		return nil, err
	}
	rewards, err := s.LoadRewards()
	if err != nil {
		return nil, err
	}
	epoch := s.now().UTC().Format("2006-01-02")
	for i := range rewards {
		if rewards[i].Address == address && rewards[i].Epoch == epoch {
			rewards[i].ServiceScore = score.ServiceScore
			rewards[i].SimulatedPoints = score.SimulatedPoints
			rewards[i].EligibleSimulatedPoints = score.EligibleSimulatedPoints
			rewards[i].Reason = rewardReason(score)
			if err := s.SaveRewards(rewards); err != nil {
				return nil, err
			}
			return rewardsFor(address, rewards), nil
		}
	}
	rewards = append(rewards, Reward{
		Epoch:                   epoch,
		Address:                 address,
		ServiceScore:            score.ServiceScore,
		SimulatedPoints:         score.SimulatedPoints,
		EligibleSimulatedPoints: score.EligibleSimulatedPoints,
		Reason:                  rewardReason(score),
		CreatedAt:               s.now().Unix(),
	})
	if err := s.SaveRewards(rewards); err != nil {
		return nil, err
	}
	return rewardsFor(address, rewards), nil
}

func rewardReason(score Score) string {
	if score.EligibilityNote != "" {
		return score.EligibilityNote + "; not dIDR"
	}
	return "daily service score simulation; not dIDR"
}

func (s Store) collateral(address string) (required uint64, active uint64, eligible bool, status string) {
	params := s.profile.Consensus.Staking
	required = params.MinServiceStake
	status = "none"
	store, err := storage.OpenBolt(s.paths.DB)
	if err != nil {
		return required, 0, required == 0, status
	}
	defer store.Close()
	bc := chain.New(store)
	blocks, err := bc.Blocks()
	if err != nil || len(blocks) == 0 {
		return required, 0, required == 0, status
	}
	state, err := staking.Replay(blocks, params)
	if err != nil {
		return required, 0, false, "insufficient"
	}
	active, unlocking, released := state.AddressSummary(address, blocks[len(blocks)-1].Height)
	switch {
	case active >= required:
		return required, active, true, "eligible"
	case unlocking > 0:
		return required, active, false, "unlocking"
	case released > 0:
		return required, active, false, "released"
	case active == 0:
		return required, active, required == 0, "none"
	default:
		return required, active, false, "insufficient"
	}
}

func (s Store) FindNode(address string) (Node, bool, error) {
	nodes, err := s.LoadNodes()
	if err != nil {
		return Node{}, false, err
	}
	for _, node := range nodes {
		if node.OwnerAddress == address {
			return node, true, nil
		}
	}
	return Node{}, false, nil
}

func (s Store) LoadNodes() ([]Node, error) {
	var nodes []Node
	if err := readJSON(s.paths.ServiceNodes, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (s Store) SaveNodes(nodes []Node) error {
	return writeJSONFile(s.paths.ServiceNodes, nodes, 0644)
}

func (s Store) LoadChallenges() ([]Challenge, error) {
	var challenges []Challenge
	if err := readJSON(s.paths.ServiceChallenges, &challenges); err != nil {
		return nil, err
	}
	return challenges, nil
}

func (s Store) SaveChallenges(challenges []Challenge) error {
	return writeJSONFile(s.paths.ServiceChallenges, challenges, 0644)
}

func (s Store) LoadRewards() ([]Reward, error) {
	var rewards []Reward
	if err := readJSON(s.paths.ServiceRewards, &rewards); err != nil {
		return nil, err
	}
	return rewards, nil
}

func (s Store) SaveRewards(rewards []Reward) error {
	return writeJSONFile(s.paths.ServiceRewards, rewards, 0644)
}

func serviceNodeID(address string) string {
	sum := sha256.Sum256([]byte("service-node:" + address))
	return "svc_" + hex.EncodeToString(sum[:8])
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		return hex.EncodeToString(sum[:n])
	}
	return hex.EncodeToString(buf)
}

func uptimeScore(now, lastSeen int64) int {
	age := now - lastSeen
	if lastSeen <= 0 || age < 0 {
		return 0
	}
	if age <= int64((5 * time.Minute).Seconds()) {
		return 100
	}
	if age <= int64((30 * time.Minute).Seconds()) {
		return 50
	}
	return 0
}

func latencyScore(ms int64) int {
	switch {
	case ms <= 50:
		return 100
	case ms <= 100:
		return 80
	case ms <= 250:
		return 60
	case ms <= 500:
		return 30
	default:
		return 10
	}
}

func bandwidthScore(bytes int64) int {
	switch {
	case bytes >= 100_000_000:
		return 100
	case bytes >= 50_000_000:
		return 80
	case bytes >= 10_000_000:
		return 50
	case bytes >= 1_000_000:
		return 20
	default:
		return 5
	}
}

func reliabilityScore(address string, challenges []Challenge) int {
	total := 0
	passed := 0
	for _, challenge := range challenges {
		if challenge.Address != address {
			continue
		}
		if challenge.Status == ChallengePassed || challenge.Status == ChallengeFailed || challenge.Status == ChallengeExpired {
			total++
		}
		if challenge.Status == ChallengePassed {
			passed++
		}
	}
	if total == 0 {
		return 0
	}
	return clamp((passed*100)/total, 0, 100)
}

func addFlag(node *Node, flag string, penalty int) {
	for _, existing := range node.Flags {
		if existing == flag {
			node.AbusePenalty = clamp(node.AbusePenalty+penalty, 0, 100)
			node.LastAbuseReason = flag
			return
		}
	}
	node.Flags = append(node.Flags, flag)
	node.AbusePenalty = clamp(node.AbusePenalty+penalty, 0, 100)
	node.LastAbuseReason = flag
}

func repeatedSample(samples []Sample, sample Sample) bool {
	if len(samples) == 0 {
		return false
	}
	last := samples[len(samples)-1]
	return last.LatencyMS == sample.LatencyMS && last.BytesUp == sample.BytesUp && last.BytesDown == sample.BytesDown
}

func appendRecent(values []int64, value int64) []int64 {
	values = append(values, value)
	if len(values) > maxRecentItems {
		return append([]int64(nil), values[len(values)-maxRecentItems:]...)
	}
	return values
}

func appendRecentSample(values []Sample, value Sample) []Sample {
	values = append(values, value)
	if len(values) > maxRecentItems {
		return append([]Sample(nil), values[len(values)-maxRecentItems:]...)
	}
	return values
}

func rewardsFor(address string, rewards []Reward) []Reward {
	out := make([]Reward, 0)
	for _, reward := range rewards {
		if reward.Address == address {
			out = append(out, reward)
		}
	}
	return out
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

func readJSON(path string, out any) error {
	lockFor(path).Lock()
	defer lockFor(path).Unlock()
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("invalid service store json %s: %w", path, err)
	}
	return nil
}

func writeJSONFile(path string, value any, perm os.FileMode) error {
	lockFor(path).Lock()
	defer lockFor(path).Unlock()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, raw, perm)
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			_ = os.Remove(tmp)
			return err
		}
		return os.Rename(tmp, path)
	}
	return nil
}

func lockFor(path string) *sync.Mutex {
	clean := filepath.Clean(path)
	lock, _ := fileLocks.LoadOrStore(clean, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
