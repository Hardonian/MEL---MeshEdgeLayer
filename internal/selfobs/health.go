package selfobs

import (
	"sync"
	"time"
)

// ComponentHealth represents the health state of an internal component
type ComponentHealth string

const (
	// HealthHealthy - component is operating normally
	HealthHealthy ComponentHealth = "healthy"
	// HealthDegraded - component is operating but with reduced performance
	HealthDegraded ComponentHealth = "degraded"
	// HealthFailing - component is not functioning properly
	HealthFailing ComponentHealth = "failing"
	// HealthUnknown - component health cannot be determined
	HealthUnknown ComponentHealth = "unknown"
)

// InternalComponent represents the health state of an internal MEL component
type InternalComponent struct {
	Name        string         `json:"name"`
	Health      ComponentHealth `json:"health"`
	LastSuccess time.Time      `json:"last_success"`
	LastFailure time.Time      `json:"last_failure"`
	ErrorCount  int64          `json:"error_count"`
	SuccessCount int64         `json:"success_count"`
	TotalOps    int64          `json:"total_ops"`
}

// ErrorRate returns the error rate as a percentage (0-100)
func (c *InternalComponent) ErrorRate() float64 {
	if c.TotalOps == 0 {
		return 0
	}
	return float64(c.ErrorCount) / float64(c.TotalOps) * 100
}

// KnownComponents returns the list of tracked internal components
var KnownComponents = []string{
	"ingest",
	"classify",
	"alert",
	"control",
	"retention",
	"backup",
	"trust", // approval lifecycle, freeze checks, evidence capture
}

// HealthRegistry tracks health of all internal components
type HealthRegistry struct {
	mu         sync.RWMutex
	components map[string]*InternalComponent
}

// NewHealthRegistry creates a new health registry
func NewHealthRegistry() *HealthRegistry {
	reg := &HealthRegistry{
		components: make(map[string]*InternalComponent),
	}
	// Initialize all known components as unknown
	for _, name := range KnownComponents {
		reg.components[name] = &InternalComponent{
			Name:        name,
			Health:      HealthUnknown,
			LastSuccess: time.Time{},
			LastFailure: time.Time{},
		}
	}
	return reg
}

// GetComponent returns the health state of a specific component
func (r *HealthRegistry) GetComponent(name string) *InternalComponent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if comp, ok := r.components[name]; ok {
		return comp
	}
	return &InternalComponent{
		Name:   name,
		Health: HealthUnknown,
	}
}

// GetAllComponents returns all component health states
func (r *HealthRegistry) GetAllComponents() []*InternalComponent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*InternalComponent, 0, len(r.components))
	for _, comp := range r.components {
		result = append(result, comp)
	}
	return result
}

// RecordSuccess records a successful operation for a component
func (r *HealthRegistry) RecordSuccess(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	comp, ok := r.components[name]
	if !ok {
		comp = &InternalComponent{Name: name}
		r.components[name] = comp
	}
	comp.SuccessCount++
	comp.TotalOps++
	comp.LastSuccess = time.Now()
	
	// Update health based on error rate
	if comp.ErrorCount == 0 {
		comp.Health = HealthHealthy
	} else if comp.ErrorRate() > 10 {
		comp.Health = HealthFailing
	} else if comp.ErrorRate() > 1 {
		comp.Health = HealthDegraded
	} else {
		comp.Health = HealthHealthy
	}
}

// RecordFailure records a failed operation for a component
func (r *HealthRegistry) RecordFailure(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	comp, ok := r.components[name]
	if !ok {
		comp = &InternalComponent{Name: name}
		r.components[name] = comp
	}
	comp.ErrorCount++
	comp.TotalOps++
	comp.LastFailure = time.Now()
	
	// Update health based on error rate
	if comp.ErrorRate() > 20 {
		comp.Health = HealthFailing
	} else if comp.ErrorRate() > 5 {
		comp.Health = HealthDegraded
	}
}

// SetHealth directly sets the health state of a component
func (r *HealthRegistry) SetHealth(name string, health ComponentHealth) {
	r.mu.Lock()
	defer r.mu.Unlock()
	comp, ok := r.components[name]
	if !ok {
		comp = &InternalComponent{Name: name}
		r.components[name] = comp
	}
	comp.Health = health
}

// GetOverallHealth returns the worst health state across all components
func (r *HealthRegistry) GetOverallHealth() ComponentHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	hasFailing := false
	hasDegraded := false
	hasUnknown := false
	
	for _, comp := range r.components {
		switch comp.Health {
		case HealthFailing:
			hasFailing = true
		case HealthDegraded:
			hasDegraded = true
		case HealthUnknown:
			hasUnknown = true
		}
	}
	
	if hasFailing {
		return HealthFailing
	}
	if hasDegraded {
		return HealthDegraded
	}
	if hasUnknown {
		return HealthUnknown
	}
	return HealthHealthy
}

// Global health registry instance
var globalRegistry = NewHealthRegistry()

// GetGlobalRegistry returns the global health registry
func GetGlobalRegistry() *HealthRegistry {
	return globalRegistry
}

// SetGlobalRegistry allows replacing the global registry (useful for testing)
func SetGlobalRegistry(reg *HealthRegistry) {
	globalRegistry = reg
}
