package selfobs

import (
	"runtime"
	"sync"
	"time"
)

// InternalMetrics holds all internal MEL metrics for observability
type InternalMetrics struct {
	mu sync.RWMutex
	
	// Pipeline latency tracking
	PipelineLatencies PipelineLatencies
	
	// Worker heartbeats
	WorkerHeartbeats map[string]time.Time
	
	// Queue depths per stage
	QueueDepths map[string]int
	
	// Error rates by component (using pointer to allow mutation)
	ErrorRates map[string]*ErrorRateTracker
	
	// Resource usage
	ResourceUsage ResourceMetrics
	
	// Operation counts
	OperationCounts map[string]int64
}

// PipelineLatencies tracks latency between pipeline stages
type PipelineLatencies struct {
	mu sync.RWMutex
	// Ingest to classify latency (ms)
	IngestToClassify []time.Duration
	// Classify to alert latency (ms)
	ClassifyToAlert []time.Duration
	// Alert to action latency (ms)
	AlertToAction []time.Duration
}

// RecordIngestToClassify records the latency from ingest to classification
func (p *PipelineLatencies) RecordIngestToClassify(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.IngestToClassify = append(p.IngestToClassify, d)
	// Keep last 1000 samples
	if len(p.IngestToClassify) > 1000 {
		p.IngestToClassify = p.IngestToClassify[1:]
	}
}

// RecordClassifyToAlert records the latency from classification to alert
func (p *PipelineLatencies) RecordClassifyToAlert(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ClassifyToAlert = append(p.ClassifyToAlert, d)
	if len(p.ClassifyToAlert) > 1000 {
		p.ClassifyToAlert = p.ClassifyToAlert[1:]
	}
}

// RecordAlertToAction records the latency from alert to action
func (p *PipelineLatencies) RecordAlertToAction(d time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.AlertToAction = append(p.AlertToAction, d)
	if len(p.AlertToAction) > 1000 {
		p.AlertToAction = p.AlertToAction[1:]
	}
}

// GetP99 returns the P99 latency for a specific pipeline stage
func (p *PipelineLatencies) GetP99(stage string) time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	var samples []time.Duration
	switch stage {
	case "ingest_to_classify":
		samples = p.IngestToClassify
	case "classify_to_alert":
		samples = p.ClassifyToAlert
	case "alert_to_action":
		samples = p.AlertToAction
	default:
		return 0
	}
	
	if len(samples) == 0 {
		return 0
	}
	
	// Sort for percentile calculation
	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	quickSortDurations(sorted)
	
	p99Index := len(sorted) * 99 / 100
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}
	return sorted[p99Index]
}

// GetAverage returns the average latency for a specific pipeline stage
func (p *PipelineLatencies) GetAverage(stage string) time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	var samples []time.Duration
	switch stage {
	case "ingest_to_classify":
		samples = p.IngestToClassify
	case "classify_to_alert":
		samples = p.ClassifyToAlert
	case "alert_to_action":
		samples = p.AlertToAction
	default:
		return 0
	}
	
	if len(samples) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, s := range samples {
		total += s
	}
	return total / time.Duration(len(samples))
}

// ErrorRateTracker tracks success/failure counts
type ErrorRateTracker struct {
	mu           sync.RWMutex
	SuccessCount int64
	FailureCount int64
	// Timestamps for windowed calculation
	RecentFailures []time.Time
}

// RecordSuccess records a successful operation
func (e *ErrorRateTracker) RecordSuccess() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.SuccessCount++
}

// RecordFailure records a failed operation
func (e *ErrorRateTracker) RecordFailure() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.FailureCount++
	e.RecentFailures = append(e.RecentFailures, time.Now())
	// Keep last hour of failures
	cutoff := time.Now().Add(-time.Hour)
	var valid []time.Time
	for _, t := range e.RecentFailures {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	e.RecentFailures = valid
}

// GetErrorRate returns the error rate as a percentage
func (e *ErrorRateTracker) GetErrorRate() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	total := e.SuccessCount + e.FailureCount
	if total == 0 {
		return 0
	}
	return float64(e.FailureCount) / float64(total) * 100
}

// ResourceMetrics holds runtime resource metrics
type ResourceMetrics struct {
	mu sync.RWMutex
	
	MemoryUsed uint64
	Goroutines int
	NumGC      uint32
}

// CollectResourceMetrics collects current resource usage
func (r *ResourceMetrics) CollectResourceMetrics() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	r.MemoryUsed = m.Alloc
	r.Goroutines = runtime.NumGoroutine()
	r.NumGC = m.NumGC
}

// ResourceMetricsSnapshot represents a point-in-time snapshot of resource usage
type ResourceMetricsSnapshot struct {
	MemoryUsed uint64 `json:"memory_used"`
	Goroutines int    `json:"goroutines"`
	NumGC      uint32 `json:"num_gc"`
}

// MetricsSnapshot represents a point-in-time snapshot of all metrics
type MetricsSnapshot struct {
	Timestamp        time.Time               `json:"timestamp"`
	PipelineLatency  map[string]int64        `json:"pipeline_latency_ms"`
	WorkerHeartbeats map[string]string         `json:"worker_heartbeats"`
	QueueDepths      map[string]int          `json:"queue_depths"`
	ErrorRates       map[string]float64      `json:"error_rates"`
	ResourceUsage    ResourceMetricsSnapshot `json:"resource_usage"`
	OperationCounts map[string]int64         `json:"operation_counts"`
}

// NewInternalMetrics creates a new InternalMetrics collection
func NewInternalMetrics() *InternalMetrics {
	return &InternalMetrics{
		WorkerHeartbeats: make(map[string]time.Time),
		QueueDepths:      make(map[string]int),
		ErrorRates:      make(map[string]*ErrorRateTracker),
		ResourceUsage:   ResourceMetrics{},
		OperationCounts: make(map[string]int64),
	}
}

// RecordWorkerHeartbeat records a worker's heartbeat
func (m *InternalMetrics) RecordWorkerHeartbeat(worker string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WorkerHeartbeats[worker] = time.Now()
}

// GetWorkerHeartbeat returns the last heartbeat time for a worker
func (m *InternalMetrics) GetWorkerHeartbeat(worker string) time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.WorkerHeartbeats[worker]
}

// SetQueueDepth sets the queue depth for a stage
func (m *InternalMetrics) SetQueueDepth(stage string, depth int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QueueDepths[stage] = depth
}

// GetQueueDepth returns the queue depth for a stage
func (m *InternalMetrics) GetQueueDepth(stage string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.QueueDepths[stage]
}

// RecordOperation records an operation count
func (m *InternalMetrics) RecordOperation(operation string, count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OperationCounts[operation] += count
}

// GetOperationCount returns the count for an operation
func (m *InternalMetrics) GetOperationCount(operation string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.OperationCounts[operation]
}

// GetErrorTracker returns the error rate tracker for a component
func (m *InternalMetrics) GetErrorTracker(component string) *ErrorRateTracker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ErrorRates[component]
}

// RecordSuccess records a success for a component
func (m *InternalMetrics) RecordSuccess(component string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tracker, ok := m.ErrorRates[component]; ok {
		tracker.RecordSuccess()
	} else {
		m.ErrorRates[component] = &ErrorRateTracker{}
		m.ErrorRates[component].RecordSuccess()
	}
}

// RecordFailure records a failure for a component
func (m *InternalMetrics) RecordFailure(component string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tracker, ok := m.ErrorRates[component]; ok {
		tracker.RecordFailure()
	} else {
		m.ErrorRates[component] = &ErrorRateTracker{}
		m.ErrorRates[component].RecordFailure()
	}
}

// GetMetricsSnapshot returns a point-in-time snapshot of all metrics
func (m *InternalMetrics) GetMetricsSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	snapshot := MetricsSnapshot{
		Timestamp:        time.Now(),
		WorkerHeartbeats: make(map[string]string),
		QueueDepths:      make(map[string]int),
		ErrorRates:       make(map[string]float64),
		OperationCounts:  make(map[string]int64),
	}
	
	// Pipeline latencies
	snapshot.PipelineLatency = map[string]int64{
		"ingest_to_classify_p99": m.PipelineLatencies.GetP99("ingest_to_classify").Milliseconds(),
		"classify_to_alert_p99":  m.PipelineLatencies.GetP99("classify_to_alert").Milliseconds(),
		"alert_to_action_p99":    m.PipelineLatencies.GetP99("alert_to_action").Milliseconds(),
		"ingest_to_classify_avg": m.PipelineLatencies.GetAverage("ingest_to_classify").Milliseconds(),
		"classify_to_alert_avg":  m.PipelineLatencies.GetAverage("classify_to_alert").Milliseconds(),
		"alert_to_action_avg":    m.PipelineLatencies.GetAverage("alert_to_action").Milliseconds(),
	}
	
	// Worker heartbeats
	for worker, heartbeat := range m.WorkerHeartbeats {
		snapshot.WorkerHeartbeats[worker] = heartbeat.Format(time.RFC3339)
	}
	
	// Queue depths
	for stage, depth := range m.QueueDepths {
		snapshot.QueueDepths[stage] = depth
	}
	
	// Error rates
	for component, tracker := range m.ErrorRates {
		snapshot.ErrorRates[component] = tracker.GetErrorRate()
	}
	
	// Operation counts
	for op, count := range m.OperationCounts {
		snapshot.OperationCounts[op] = count
	}
	
	// Resource usage
	m.ResourceUsage.mu.RLock()
	snapshot.ResourceUsage = ResourceMetricsSnapshot{
		MemoryUsed: m.ResourceUsage.MemoryUsed,
		Goroutines: m.ResourceUsage.Goroutines,
		NumGC:      m.ResourceUsage.NumGC,
	}
	m.ResourceUsage.mu.RUnlock()
	
	return snapshot
}

// Global metrics instance
var globalMetrics = NewInternalMetrics()

// GetGlobalMetrics returns the global metrics instance
func GetGlobalMetrics() *InternalMetrics {
	return globalMetrics
}

// SetGlobalMetrics allows replacing the global metrics (useful for testing)
func SetGlobalMetrics(m *InternalMetrics) {
	globalMetrics = m
}

// Package-level convenience functions

// RecordWorkerHeartbeat records a heartbeat to the global metrics
func RecordWorkerHeartbeat(worker string) {
	globalMetrics.RecordWorkerHeartbeat(worker)
}

// SetQueueDepth sets queue depth in global metrics
func SetQueueDepth(stage string, depth int) {
	globalMetrics.SetQueueDepth(stage, depth)
}

// RecordMetricSuccess records a success to global metrics
func RecordMetricSuccess(component string) {
	globalMetrics.RecordSuccess(component)
}

// RecordMetricFailure records a failure to global metrics
func RecordMetricFailure(component string) {
	globalMetrics.RecordFailure(component)
}

// GetMetricsSnapshot returns a snapshot of global metrics
func GetMetricsSnapshot() MetricsSnapshot {
	return globalMetrics.GetMetricsSnapshot()
}

// quickSortDurations is a simple quicksort for durations
func quickSortDurations(a []time.Duration) {
	if len(a) < 2 {
		return
	}
	pivot := a[len(a)/2]
	i, j := 0, len(a)-1
	for i <= j {
		for i <= j && a[i] < pivot {
			i++
		}
		for i <= j && a[j] > pivot {
			j--
		}
		if i <= j {
			a[i], a[j] = a[j], a[i]
			i++
			j--
		}
	}
	if j > 0 {
		quickSortDurations(a[:j+1])
	}
	if i < len(a) {
		quickSortDurations(a[i:])
	}
}
