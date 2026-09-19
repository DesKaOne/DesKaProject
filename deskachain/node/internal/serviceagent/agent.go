package serviceagent

import (
	"context"
	"fmt"
	"io"
	"time"

	"deskachain/internal/config"
	"deskachain/internal/crypto"
)

type Options struct {
	RPCURL               string
	Address              string
	Endpoint             string
	StatePath            string
	HeartbeatInterval    time.Duration
	ChallengeInterval    time.Duration
	ScoreInterval        time.Duration
	RetryInterval        time.Duration
	Once                 bool
	SafeMode             bool
	MaxBytesPerChallenge int64
	ClientVersion        string
	Platform             string
}

type Agent struct {
	Options Options
	Client  Client
	Out     io.Writer
	Now     func() time.Time
}

func DefaultOptions() Options {
	return Options{
		StatePath:            "./idrservice-state.json",
		HeartbeatInterval:    30 * time.Second,
		ChallengeInterval:    time.Minute,
		ScoreInterval:        time.Minute,
		RetryInterval:        5 * time.Second,
		SafeMode:             true,
		MaxBytesPerChallenge: 100_000_000,
		ClientVersion:        "idrservice/dev",
	}
}

func (a Agent) Run(ctx context.Context) error {
	if a.Now == nil {
		a.Now = time.Now
	}
	if a.Out == nil {
		a.Out = io.Discard
	}
	if err := a.validate(); err != nil {
		return err
	}
	state, err := LoadState(a.Options.StatePath)
	if err != nil {
		return err
	}
	a.printStartup()
	if !a.Options.SafeMode {
		fmt.Fprintln(a.Out, "warning: unsafe mode is not implemented yet; running in safe simulation mode")
	}
	if health, err := a.Client.Health(ctx); err == nil {
		if network, ok := health["network"]; ok {
			fmt.Fprintf(a.Out, "network: %v\n", network)
		}
		if serviceRPC, ok := health["service_rpc"]; ok {
			fmt.Fprintf(a.Out, "service rpc: %v\n", serviceRPC)
		}
	} else if a.Options.Once {
		return err
	}
	if a.Options.Once {
		if err := a.runCycle(ctx, &state); err != nil {
			return err
		}
		if err := SaveState(a.Options.StatePath, state); err != nil {
			return err
		}
		fmt.Fprintln(a.Out, "state saved")
		return nil
	}
	if err := a.register(ctx, &state); err != nil {
		return err
	}
	heartbeatTicker := time.NewTicker(a.Options.HeartbeatInterval)
	challengeTicker := time.NewTicker(a.Options.ChallengeInterval)
	scoreTicker := time.NewTicker(a.Options.ScoreInterval)
	defer heartbeatTicker.Stop()
	defer challengeTicker.Stop()
	defer scoreTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(a.Out, "shutdown requested")
			if err := SaveState(a.Options.StatePath, state); err != nil {
				return err
			}
			fmt.Fprintln(a.Out, "state saved")
			fmt.Fprintln(a.Out, "service agent stopped")
			return nil
		case <-heartbeatTicker.C:
			if err := a.heartbeat(ctx, &state); err != nil {
				fmt.Fprintf(a.Out, "retrying after error: %v\n", err)
			}
		case <-challengeTicker.C:
			if err := a.challenge(ctx, &state); err != nil {
				fmt.Fprintf(a.Out, "retrying after error: %v\n", err)
			}
		case <-scoreTicker.C:
			if err := a.score(ctx, &state); err != nil {
				fmt.Fprintf(a.Out, "retrying after error: %v\n", err)
			}
			_ = SaveState(a.Options.StatePath, state)
		}
	}
}

func (a Agent) runCycle(ctx context.Context, state *State) error {
	if err := a.register(ctx, state); err != nil {
		return err
	}
	if err := a.heartbeat(ctx, state); err != nil {
		return err
	}
	if err := a.challenge(ctx, state); err != nil {
		return err
	}
	return a.score(ctx, state)
}

func (a Agent) register(ctx context.Context, state *State) error {
	resp, err := withRetry(ctx, a, func() (RegisterResponse, error) {
		return a.Client.Register(ctx, a.request())
	})
	if err != nil {
		return err
	}
	state.Address = a.Options.Address
	state.Endpoint = a.Options.Endpoint
	state.RPCURL = a.Options.RPCURL
	state.ServiceNodeID = resp.ServiceNodeID
	state.ClientVersion = a.Options.ClientVersion
	state.Platform = a.Options.Platform
	if state.RegisteredAt == 0 {
		state.RegisteredAt = a.Now().Unix()
	}
	fmt.Fprintf(a.Out, "service agent registered id=%s\n", resp.ServiceNodeID)
	return nil
}

func (a Agent) heartbeat(ctx context.Context, state *State) error {
	resp, err := withRetry(ctx, a, func() (HeartbeatResponse, error) {
		return a.Client.Heartbeat(ctx, a.request())
	})
	if err != nil {
		return err
	}
	state.LastHeartbeatAt = a.Now().Unix()
	state.LastScore = resp.ServiceScore
	fmt.Fprintf(a.Out, "heartbeat ok score=%d\n", resp.ServiceScore)
	return nil
}

func (a Agent) challenge(ctx context.Context, state *State) error {
	challenge, err := withRetry(ctx, a, func() (ChallengeResponse, error) {
		return a.Client.CreateChallenge(ctx, a.Options.Address)
	})
	if err != nil {
		return err
	}
	state.LastChallengeID = challenge.ChallengeID
	state.LastChallengeAt = a.Now().Unix()
	state.TotalChallenges++
	fmt.Fprintf(a.Out, "challenge created id=%s\n", challenge.ChallengeID)
	// TODO: future real verifier should use signed challenge and controlled verifier endpoint.
	// TODO: future mobile safe mode should check WiFi/charging/battery/temperature.
	measurement := NewSimulator(true, a.Options.MaxBytesPerChallenge).Generate()
	submit, err := a.Client.SubmitChallenge(ctx, challenge.ChallengeID, measurement)
	if err != nil {
		state.FailedChallenges++
		return err
	}
	state.LastSubmitAt = a.Now().Unix()
	state.TotalSimulatedBytesUp += measurement.BytesUp
	state.TotalSimulatedBytesDown += measurement.BytesDown
	state.LastScore = submit.ServiceScore
	if submit.Status == "passed" {
		state.SuccessfulChallenges++
	} else {
		state.FailedChallenges++
	}
	fmt.Fprintf(a.Out, "challenge submitted status=%s score=%d\n", submit.Status, submit.ServiceScore)
	return nil
}

func (a Agent) score(ctx context.Context, state *State) error {
	resp, err := withRetry(ctx, a, func() (ScoreResponse, error) {
		return a.Client.Score(ctx, a.Options.Address)
	})
	if err != nil {
		return err
	}
	state.LastScore = resp.Score.ServiceScore
	state.LastSimulatedPoints = resp.Score.SimulatedPoints
	fmt.Fprintf(a.Out, "service score=%d points=%d\n", resp.Score.ServiceScore, resp.Score.SimulatedPoints)
	if resp.Score.Note != "" {
		fmt.Fprintln(a.Out, resp.Score.Note)
	}
	return nil
}

func (a Agent) request() RegisterRequest {
	return RegisterRequest{
		Address:       a.Options.Address,
		Endpoint:      a.Options.Endpoint,
		ClientVersion: a.Options.ClientVersion,
		Platform:      a.Options.Platform,
		UserAgent:     "idrservice",
	}
}

func withRetry[T any](ctx context.Context, a Agent, fn func() (T, error)) (T, error) {
	var zero T
	attempts := 3
	if !a.Options.Once {
		attempts = 1
	}
	var last error
	for i := 0; i < attempts; i++ {
		value, err := fn()
		if err == nil {
			return value, nil
		}
		last = err
		if !IsRetryable(err) || i == attempts-1 {
			break
		}
		fmt.Fprintf(a.Out, "retrying after error: %v\n", err)
		timer := time.NewTimer(a.Options.RetryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
	return zero, last
}

func (a Agent) validate() error {
	if a.Options.RPCURL == "" {
		return fmt.Errorf("rpc-url is required")
	}
	if a.Options.Address == "" {
		return fmt.Errorf("address is required")
	}
	if err := validateAddressAnyNetwork(a.Options.Address); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	return nil
}

func validateAddressAnyNetwork(address string) error {
	profiles := []config.NetworkConfig{config.Localnet(), config.Testnet(), config.Mainnet()}
	var last error
	for _, profile := range profiles {
		if err := crypto.ValidateAddressForNetwork(address, profile); err == nil {
			return nil
		} else {
			last = err
		}
	}
	return last
}

func (a Agent) printStartup() {
	fmt.Fprintln(a.Out, "DesKaChain Service Node Agent")
	fmt.Fprintf(a.Out, "rpc: %s\n", a.Options.RPCURL)
	fmt.Fprintf(a.Out, "address: %s\n", a.Options.Address)
	fmt.Fprintf(a.Out, "endpoint: %s\n", a.Options.Endpoint)
	fmt.Fprintf(a.Out, "safe mode: %t\n", true)
	fmt.Fprintf(a.Out, "heartbeat interval: %s\n", a.Options.HeartbeatInterval)
	fmt.Fprintf(a.Out, "challenge interval: %s\n", a.Options.ChallengeInterval)
}
