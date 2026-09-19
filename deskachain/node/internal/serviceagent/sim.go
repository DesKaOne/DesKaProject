package serviceagent

import (
	"math/rand"
	"time"
)

type Measurement struct {
	LatencyMS int64 `json:"latency_ms"`
	BytesUp   int64 `json:"bytes_up"`
	BytesDown int64 `json:"bytes_down"`
	Success   bool  `json:"success"`
}

type Simulator struct {
	SafeMode             bool
	MaxBytesPerChallenge int64
	rng                  *rand.Rand
}

func NewSimulator(safeMode bool, maxBytesPerChallenge int64) Simulator {
	if maxBytesPerChallenge <= 0 {
		maxBytesPerChallenge = 100_000_000
	}
	return Simulator{
		SafeMode:             true,
		MaxBytesPerChallenge: maxBytesPerChallenge,
		rng:                  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s Simulator) Generate() Measurement {
	rng := s.rng
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	up := int64(1_000_000 + rng.Int63n(19_000_001))
	down := int64(5_000_000 + rng.Int63n(75_000_001))
	if s.MaxBytesPerChallenge > 0 {
		if up > s.MaxBytesPerChallenge {
			up = s.MaxBytesPerChallenge
		}
		if down > s.MaxBytesPerChallenge {
			down = s.MaxBytesPerChallenge
		}
	}
	return Measurement{
		LatencyMS: int64(30 + rng.Int63n(91)),
		BytesUp:   up,
		BytesDown: down,
		Success:   true,
	}
}
