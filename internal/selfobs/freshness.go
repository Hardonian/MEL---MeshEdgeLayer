package selfobs

import (
	"sync"
	"time"
)

// FreshnessMarker tracks the last update time and freshness thresholds for a component
type FreshnessMarker struct {
	Component        string        `json:"component"`
	LastUpdate       time.Time     `json:"last_update"`
	ExpectedInterval time.Duration `json:"expected_interval"`
	StaleThreshold   time.Duration `json:"stale_threshold"`
}

// IsFresh returns true if the component's data is considered fresh (within stale threshold)
func (f *FreshnessMarker) IsFresh() bool {
	if f.LastUpdate.IsZero() {
		return false
	}
	return time.Since(f.LastUpdate) < f.StaleThreshold
}

// Age returns the time since the last update
func (f *FreshnessMarker) Age() time.Duration {
	if f.LastUpdate.IsZero() {
		return -1 // Indicates never updated
	}
	return time.Since(f.LastUpdate)
}

// IsStale returns true if the component's data is stale (beyond stale threshold)
func (f *FreshnessMarker) IsStale() bool {
	if f.LastUpdate.IsZero() {
		return true
	}
	return time.Since(f.LastUpdate) >= f.StaleThreshold
}

// FreshnessTracker tracks freshness markers for all components
type FreshnessTracker struct {
	mu      sync.RWMutex
	markers map[string]*FreshnessMarker
}

// DefaultFreshnessIntervals defines the default freshness expectations per component
var DefaultFreshnessIntervals = map[string]struct {
	Interval     time.Duration
	StaleDefault time.Duration
}{
	"ingest":    {Interval: 10 * time.Second, StaleDefault: 60 * time.Second},
	"classify":  {Interval: 30 * time.Second, StaleDefault: 120 * time.Second},
	"alert":     {Interval: 60 * time.Second, StaleDefault: 300 * time.Second},
	"control":   {Interval: 30 * time.Second, StaleDefault: 120 * time.Second},
	"retention": {Interval: 300 * time.Second, StaleDefault: 600 * time.Second},    // 5 min / 10 min
	"backup":    {Interval: 3600 * time.Second, StaleDefault: 86400 * time.Second}, // 1 hour / 24 hours
}

// NewFreshnessTracker creates a new freshness tracker with default intervals
func NewFreshnessTracker() *FreshnessTracker {
	tracker := &FreshnessTracker{
		markers: make(map[string]*FreshnessMarker),
	}
	// Initialize all known components with default intervals
	for component, intervals := range DefaultFreshnessIntervals {
		tracker.markers[component] = &FreshnessMarker{
			Component:        component,
			LastUpdate:       time.Time{},
			ExpectedInterval: intervals.Interval,
			StaleThreshold:   intervals.StaleDefault,
		}
	}
	return tracker
}

// GetMarker returns the freshness marker for a specific component
func (t *FreshnessTracker) GetMarker(component string) *FreshnessMarker {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if marker, ok := t.markers[component]; ok {
		return marker
	}
	return &FreshnessMarker{
		Component:        component,
		LastUpdate:       time.Time{},
		ExpectedInterval: 60 * time.Second,
		StaleThreshold:   300 * time.Second,
	}
}

// GetAllMarkers returns all freshness markers
func (t *FreshnessTracker) GetAllMarkers() []*FreshnessMarker {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]*FreshnessMarker, 0, len(t.markers))
	for _, marker := range t.markers {
		result = append(result, marker)
	}
	return result
}

// MarkFresh updates the last update timestamp for a component
func (t *FreshnessTracker) MarkFresh(component string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	marker, ok := t.markers[component]
	if !ok {
		// Use defaults for unknown components
		intervals, hasDefault := DefaultFreshnessIntervals[component]
		if hasDefault {
			marker = &FreshnessMarker{
				Component:        component,
				ExpectedInterval: intervals.Interval,
				StaleThreshold:   intervals.StaleDefault,
			}
		} else {
			marker = &FreshnessMarker{
				Component:        component,
				ExpectedInterval: 60 * time.Second,
				StaleThreshold:   300 * time.Second,
			}
		}
		t.markers[component] = marker
	}
	marker.LastUpdate = time.Now()
}

// IsFresh checks if a component's data is fresh
func (t *FreshnessTracker) IsFresh(component string) bool {
	marker := t.GetMarker(component)
	return marker.IsFresh()
}

// GetStaleComponents returns a list of components that are stale
func (t *FreshnessTracker) GetStaleComponents() []*FreshnessMarker {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var stale []*FreshnessMarker
	for _, marker := range t.markers {
		if marker.IsStale() {
			stale = append(stale, marker)
		}
	}
	return stale
}

// GetFreshComponents returns a list of components that are fresh
func (t *FreshnessTracker) GetFreshComponents() []*FreshnessMarker {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var fresh []*FreshnessMarker
	for _, marker := range t.markers {
		if marker.IsFresh() {
			fresh = append(fresh, marker)
		}
	}
	return fresh
}

// SetExpectedInterval sets the expected interval for a component
func (t *FreshnessTracker) SetExpectedInterval(component string, interval time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	marker, ok := t.markers[component]
	if !ok {
		marker = &FreshnessMarker{
			Component: component,
		}
		t.markers[component] = marker
	}
	marker.ExpectedInterval = interval
}

// SetStaleThreshold sets the stale threshold for a component
func (t *FreshnessTracker) SetStaleThreshold(component string, threshold time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	marker, ok := t.markers[component]
	if !ok {
		marker = &FreshnessMarker{
			Component: component,
		}
		t.markers[component] = marker
	}
	marker.StaleThreshold = threshold
}

// Global freshness tracker instance
var globalFreshnessTracker = NewFreshnessTracker()

// GetGlobalFreshnessTracker returns the global freshness tracker
func GetGlobalFreshnessTracker() *FreshnessTracker {
	return globalFreshnessTracker
}

// SetGlobalFreshnessTracker allows replacing the global tracker (useful for testing)
func SetGlobalFreshnessTracker(tracker *FreshnessTracker) {
	globalFreshnessTracker = tracker
}

// Package-level convenience functions

// MarkFresh updates the global freshness tracker
func MarkFresh(component string) {
	globalFreshnessTracker.MarkFresh(component)
}

// IsFresh checks if a component is fresh according to the global tracker
func IsFresh(component string) bool {
	return globalFreshnessTracker.IsFresh(component)
}

// GetStaleComponents returns stale components from the global tracker
func GetStaleComponents() []*FreshnessMarker {
	return globalFreshnessTracker.GetStaleComponents()
}
