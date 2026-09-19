package mining

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const (
	StatusIdle      = "idle"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

var ErrAlreadyRunning = errors.New("mining already running")

type MiningJob struct {
	ID              string    `json:"job_id,omitempty"`
	MinerAddress    string    `json:"miner_address,omitempty"`
	RequestedBlocks int       `json:"requested_blocks,omitempty"`
	MinedBlocks     int       `json:"mined_blocks,omitempty"`
	StartedAt       time.Time `json:"started_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
	Status          string    `json:"status"`
	LastHeight      uint64    `json:"last_height,omitempty"`
	LastHash        string    `json:"last_hash,omitempty"`
	Error           string    `json:"error,omitempty"`
}

type Service struct {
	mu         sync.Mutex
	currentJob *MiningJob
}

func NewService() *Service {
	return &Service{currentJob: &MiningJob{Status: StatusIdle}}
}

func (s *Service) Start(miner string, requestedBlocks int) (*MiningJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentJob != nil && s.currentJob.Status == StatusRunning {
		return nil, ErrAlreadyRunning
	}
	now := time.Now()
	job := &MiningJob{
		ID:              newJobID(),
		MinerAddress:    miner,
		RequestedBlocks: requestedBlocks,
		Status:          StatusRunning,
		StartedAt:       now,
		UpdatedAt:       now,
	}
	s.currentJob = job
	copy := *job
	return &copy, nil
}

func (s *Service) RecordBlock(height uint64, hash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentJob == nil || s.currentJob.Status != StatusRunning {
		return
	}
	s.currentJob.MinedBlocks++
	s.currentJob.LastHeight = height
	s.currentJob.LastHash = hash
	s.currentJob.UpdatedAt = time.Now()
}

func (s *Service) Complete() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentJob == nil {
		s.currentJob = &MiningJob{Status: StatusIdle, UpdatedAt: time.Now()}
		return
	}
	s.currentJob.Status = StatusCompleted
	s.currentJob.UpdatedAt = time.Now()
}

func (s *Service) Fail(err error) {
	s.finish(StatusFailed, err)
}

func (s *Service) Cancel(err error) {
	s.finish(StatusCancelled, err)
}

func (s *Service) Snapshot() MiningJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentJob == nil {
		return MiningJob{Status: StatusIdle}
	}
	return *s.currentJob
}

func (s *Service) finish(status string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentJob == nil {
		s.currentJob = &MiningJob{Status: status, UpdatedAt: time.Now()}
	} else {
		s.currentJob.Status = status
		s.currentJob.UpdatedAt = time.Now()
	}
	if err != nil {
		s.currentJob.Error = err.Error()
	}
}

func newJobID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b[:])
}
