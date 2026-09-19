package serviceagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type RegisterRequest struct {
	Address       string `json:"address"`
	Endpoint      string `json:"endpoint,omitempty"`
	ClientVersion string `json:"client_version,omitempty"`
	Platform      string `json:"platform,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`
}

type RegisterResponse struct {
	OK            bool   `json:"ok"`
	ServiceNodeID string `json:"service_node_id"`
	OwnerAddress  string `json:"owner_address"`
	Status        string `json:"status"`
}

type HeartbeatResponse struct {
	OK              bool     `json:"ok"`
	Status          string   `json:"status"`
	UptimeScore     int      `json:"uptime_score"`
	LatencyScore    int      `json:"latency_score"`
	BandwidthScore  int      `json:"bandwidth_score"`
	ServiceScore    int      `json:"service_score"`
	SimulatedPoints int      `json:"simulated_points"`
	Flags           []string `json:"flags"`
	Note            string   `json:"note"`
}

type ChallengeResponse struct {
	OK          bool   `json:"ok"`
	ChallengeID string `json:"challenge_id"`
	Address     string `json:"address"`
	IssuedAt    int64  `json:"issued_at"`
	ExpiresAt   int64  `json:"expires_at"`
	Nonce       string `json:"nonce"`
	Status      string `json:"status"`
}

type SubmitResponse struct {
	OK              bool     `json:"ok"`
	ChallengeID     string   `json:"challenge_id"`
	Status          string   `json:"status"`
	ServiceScore    int      `json:"service_score"`
	SimulatedPoints int      `json:"simulated_points"`
	Flags           []string `json:"flags"`
	Note            string   `json:"note"`
}

type ScoreResponse struct {
	OK    bool  `json:"ok"`
	Score Score `json:"score"`
}

type RewardsResponse struct {
	OK      bool     `json:"ok"`
	Address string   `json:"address"`
	Rewards []Reward `json:"rewards"`
	Note    string   `json:"note"`
}

type Score struct {
	Address                 string `json:"address"`
	ServiceScore            int    `json:"service_score"`
	SimulatedPoints         int    `json:"simulated_points"`
	EligibleSimulatedPoints int    `json:"eligible_simulated_points"`
	StakeEligible           bool   `json:"stake_eligible"`
	CollateralStatus        string `json:"collateral_status"`
	EligibilityNote         string `json:"eligibility_note"`
	Note                    string `json:"note"`
}

type Reward struct {
	Epoch           string `json:"epoch"`
	ServiceScore    int    `json:"service_score"`
	SimulatedPoints int    `json:"simulated_points"`
	Reason          string `json:"reason"`
}

func NewClient(baseURL string) Client {
	return Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c Client) Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	var resp RegisterResponse
	err := c.post(ctx, "/service/register", req, &resp)
	return resp, err
}

func (c Client) Heartbeat(ctx context.Context, req RegisterRequest) (HeartbeatResponse, error) {
	var resp HeartbeatResponse
	err := c.post(ctx, "/service/heartbeat", req, &resp)
	return resp, err
}

func (c Client) CreateChallenge(ctx context.Context, address string) (ChallengeResponse, error) {
	var resp ChallengeResponse
	err := c.post(ctx, "/service/challenge/create", map[string]string{"address": address}, &resp)
	return resp, err
}

func (c Client) SubmitChallenge(ctx context.Context, challengeID string, m Measurement) (SubmitResponse, error) {
	var resp SubmitResponse
	err := c.post(ctx, "/service/challenge/submit", map[string]any{
		"challenge_id": challengeID,
		"latency_ms":   m.LatencyMS,
		"bytes_up":     m.BytesUp,
		"bytes_down":   m.BytesDown,
		"success":      m.Success,
	}, &resp)
	if err != nil && strings.Contains(err.Error(), "challenge already") {
		return resp, nil
	}
	return resp, err
}

func (c Client) Score(ctx context.Context, address string) (ScoreResponse, error) {
	var resp ScoreResponse
	err := c.get(ctx, "/service/score?address="+url.QueryEscape(address), &resp)
	return resp, err
}

func (c Client) Rewards(ctx context.Context, address string) (RewardsResponse, error) {
	var resp RewardsResponse
	err := c.get(ctx, "/service/rewards?address="+url.QueryEscape(address), &resp)
	return resp, err
}

func (c Client) Health(ctx context.Context) (map[string]any, error) {
	var resp map[string]any
	err := c.get(ctx, "/health", &resp)
	return resp, err
}

func (c Client) post(ctx context.Context, path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c Client) do(req *http.Request, out any) error {
	client := c.HTTPClient
	if client == nil {
		client = NewClient(c.BaseURL).HTTPClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return classifyError(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return responseError(raw)
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return err
	}
	return nil
}

func responseError(raw []byte) error {
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(raw, &body); err == nil && body.Error != "" {
		if strings.Contains(body.Error, "service RPC disabled") {
			return errors.New("service RPC disabled on target node")
		}
		if strings.Contains(body.Error, "rate limit") {
			return errors.New("rate limit exceeded")
		}
		return errors.New(body.Error)
	}
	return fmt.Errorf("service rpc error: %s", strings.TrimSpace(string(raw)))
}

func classifyError(err error) error {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return fmt.Errorf("node unavailable: %w", err)
	}
	return err
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "node unavailable") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "timeout") ||
		strings.Contains(text, "EOF") ||
		strings.Contains(text, "temporary")
}
