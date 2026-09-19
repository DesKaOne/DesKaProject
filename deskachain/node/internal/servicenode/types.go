package servicenode

const (
	StatusRegistered = "registered"
	StatusActive     = "active"
	StatusInactive   = "inactive"
	StatusBanned     = "banned"

	ChallengePending = "pending"
	ChallengePassed  = "passed"
	ChallengeFailed  = "failed"
	ChallengeExpired = "expired"

	ModeLocal    = "local"
	ModeTest     = "test"
	ModeDisabled = "disabled"

	FlagHeartbeatSpam            = "heartbeat_spam"
	FlagImpossibleBandwidth      = "impossible_bandwidth"
	FlagRepeatedIdenticalSamples = "repeated_identical_samples"
	FlagChallengeFailed          = "challenge_failed"
	FlagChallengeExpired         = "challenge_expired"
	FlagEndpointChangedTooOften  = "endpoint_changed_too_often"
)

type Metadata struct {
	ClientVersion string `json:"client_version,omitempty"`
	Platform      string `json:"platform,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`
}

type Node struct {
	ServiceNodeID      string   `json:"service_node_id"`
	OwnerAddress       string   `json:"owner_address"`
	AdvertisedEndpoint string   `json:"advertised_endpoint,omitempty"`
	Mode               string   `json:"mode"`
	CreatedAt          int64    `json:"created_at"`
	LastSeenAt         int64    `json:"last_seen_at"`
	Status             string   `json:"status"`
	Metadata           Metadata `json:"metadata"`
	Heartbeats         []int64  `json:"heartbeats,omitempty"`
	Samples            []Sample `json:"samples,omitempty"`
	Flags              []string `json:"flags,omitempty"`
	AbusePenalty       int      `json:"abuse_penalty"`
	LastAbuseReason    string   `json:"last_abuse_reason,omitempty"`
	EndpointChanges    int      `json:"endpoint_changes,omitempty"`
}

type Sample struct {
	CreatedAt int64 `json:"created_at"`
	LatencyMS int64 `json:"latency_ms"`
	BytesUp   int64 `json:"bytes_up"`
	BytesDown int64 `json:"bytes_down"`
	Success   bool  `json:"success"`
}

type Challenge struct {
	ChallengeID  string  `json:"challenge_id"`
	Address      string  `json:"address"`
	IssuedAt     int64   `json:"issued_at"`
	ExpiresAt    int64   `json:"expires_at"`
	Nonce        string  `json:"nonce"`
	ExpectedMode string  `json:"expected_mode"`
	Status       string  `json:"status"`
	Result       *Sample `json:"result,omitempty"`
}

type Reward struct {
	Epoch                   string `json:"epoch"`
	Address                 string `json:"address"`
	ServiceScore            int    `json:"service_score"`
	SimulatedPoints         int    `json:"simulated_points"`
	EligibleSimulatedPoints int    `json:"eligible_simulated_points"`
	Reason                  string `json:"reason"`
	CreatedAt               int64  `json:"created_at"`
}

type Score struct {
	Address                 string   `json:"address"`
	UptimeScore             int      `json:"uptime_score"`
	LatencyScore            int      `json:"latency_score"`
	BandwidthScore          int      `json:"bandwidth_score"`
	ReliabilityScore        int      `json:"reliability_score"`
	AbusePenalty            int      `json:"abuse_penalty"`
	ServiceScore            int      `json:"service_score"`
	Flags                   []string `json:"flags,omitempty"`
	SimulatedPoints         int      `json:"simulated_points"`
	EligibleSimulatedPoints int      `json:"eligible_simulated_points"`
	RequiredStake           uint64   `json:"required_stake"`
	ActiveStake             uint64   `json:"active_stake"`
	StakeEligible           bool     `json:"stake_eligible"`
	CollateralStatus        string   `json:"collateral_status"`
	EligibilityNote         string   `json:"eligibility_note,omitempty"`
	Note                    string   `json:"note"`
}
