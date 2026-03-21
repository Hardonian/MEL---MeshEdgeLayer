package kernel

import (
	"sync"
	"sync/atomic"
	"time"
)

// Backpressure implements rate limiting, batching, and memory bounds
// for the kernel event processing pipeline.
type Backpressure struct {
	mu sync.Mutex

	// Rate limiting
	maxEventsPerSecond int
	eventCount         atomic.Int64
	windowStart        time.Time

	// Memory bounds
	maxPendingEvents int
	pendingCount     atomic.Int64

	// Batching
	batchSize    int
	batchTimeout time.Duration

	// Stats
	accepted  atomic.Uint64
	rejected  atomic.Uint64
	throttled atomic.Uint64
}

// BackpressureConfig configures the backpressure system.
type BackpressureConfig struct {
	MaxEventsPerSecond int           `json:"max_events_per_second"`
	MaxPendingEvents   int           `json:"max_pending_events"`
	BatchSize          int           `json:"batch_size"`
	BatchTimeout       time.Duration `json:"batch_timeout"`
}

// DefaultBackpressureConfig returns safe defaults.
func DefaultBackpressureConfig() BackpressureConfig {
	return BackpressureConfig{
		MaxEventsPerSecond: 10000,
		MaxPendingEvents:   50000,
		BatchSize:          100,
		BatchTimeout:       100 * time.Millisecond,
	}
}

// NewBackpressure creates a backpressure controller.
func NewBackpressure(cfg BackpressureConfig) *Backpressure {
	if cfg.MaxEventsPerSecond <= 0 {
		cfg.MaxEventsPerSecond = 10000
	}
	if cfg.MaxPendingEvents <= 0 {
		cfg.MaxPendingEvents = 50000
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 100 * time.Millisecond
	}

	return &Backpressure{
		maxEventsPerSecond: cfg.MaxEventsPerSecond,
		windowStart:        time.Now(),
		maxPendingEvents:   cfg.MaxPendingEvents,
		batchSize:          cfg.BatchSize,
		batchTimeout:       cfg.BatchTimeout,
	}
}

// Admit checks whether a new event should be accepted.
// Returns true if the event is accepted, false if rejected due to backpressure.
func (bp *Backpressure) Admit() bool {
	// Check pending count
	if int(bp.pendingCount.Load()) >= bp.maxPendingEvents {
		bp.rejected.Add(1)
		return false
	}

	// Check rate limit
	bp.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(bp.windowStart)
	if elapsed >= time.Second {
		// Reset window
		bp.windowStart = now
		bp.eventCount.Store(0)
	}
	bp.mu.Unlock()

	count := bp.eventCount.Add(1)
	if int(count) > bp.maxEventsPerSecond {
		bp.throttled.Add(1)
		bp.eventCount.Add(-1)
		return false
	}

	bp.pendingCount.Add(1)
	bp.accepted.Add(1)
	return true
}

// Release signals that an event has been fully processed.
func (bp *Backpressure) Release() {
	bp.pendingCount.Add(-1)
}

// BackpressureStats returns current statistics.
type BackpressureStats struct {
	Accepted       uint64 `json:"accepted"`
	Rejected       uint64 `json:"rejected"`
	Throttled      uint64 `json:"throttled"`
	PendingCount   int64  `json:"pending_count"`
	RateWindowUsed int64  `json:"rate_window_used"`
}

// Stats returns current backpressure statistics.
func (bp *Backpressure) Stats() BackpressureStats {
	return BackpressureStats{
		Accepted:       bp.accepted.Load(),
		Rejected:       bp.rejected.Load(),
		Throttled:      bp.throttled.Load(),
		PendingCount:   bp.pendingCount.Load(),
		RateWindowUsed: bp.eventCount.Load(),
	}
}

// IsUnderPressure returns true if the system is near capacity.
func (bp *Backpressure) IsUnderPressure() bool {
	pending := int(bp.pendingCount.Load())
	return pending > bp.maxPendingEvents*80/100 // 80% threshold
}
