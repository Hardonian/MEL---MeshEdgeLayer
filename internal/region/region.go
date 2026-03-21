// Package region implements region-aware operation for MEL.
//
// Regions provide:
//   - region-local scoring and preferred nodes
//   - region-level health aggregation
//   - regional degradation detection
//   - cross-region fallback
//   - region isolation support
package region

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/kernel"
)

// Region represents a geographic or logical deployment region.
type Region struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Priority int               `json:"priority"` // lower = higher priority
	Isolated bool              `json:"isolated"` // true = no cross-region sync
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Health summarizes the health of a region.
type Health struct {
	RegionID        string             `json:"region_id"`
	NodeCount       int                `json:"node_count"`
	HealthyNodes    int                `json:"healthy_nodes"`
	DegradedNodes   int                `json:"degraded_nodes"`
	FailingNodes    int                `json:"failing_nodes"`
	DeadNodes       int                `json:"dead_nodes"`
	OverallHealth   float64            `json:"overall_health"` // 0.0 to 1.0
	TransportHealth map[string]float64 `json:"transport_health"`
	LastUpdateAt    time.Time          `json:"last_update_at"`
	Degraded        bool               `json:"degraded"`
	Isolated        bool               `json:"isolated"`
}

// GlobalTopology aggregates regions and cross-region routing.
type GlobalTopology struct {
	Regions          []Health          `json:"regions"`
	TotalNodes       int               `json:"total_nodes"`
	TotalHealthy     int               `json:"total_healthy"`
	GlobalHealth     float64           `json:"global_health"`
	DegradedRegions  []string          `json:"degraded_regions"`
	IsolatedRegions  []string          `json:"isolated_regions"`
	CrossRegionLinks []CrossRegionLink `json:"cross_region_links"`
	ComputedAt       time.Time         `json:"computed_at"`
}

// CrossRegionLink represents connectivity between two regions.
type CrossRegionLink struct {
	RegionA     string    `json:"region_a"`
	RegionB     string    `json:"region_b"`
	Latency     float64   `json:"latency_ms"`
	Healthy     bool      `json:"healthy"`
	LastChecked time.Time `json:"last_checked"`
}

// Manager coordinates region-aware operations.
type Manager struct {
	mu      sync.RWMutex
	localID string
	regions map[string]*Region
	health  map[string]*Health
	links   []CrossRegionLink
	dbPath  string
}

// NewManager creates a region manager.
func NewManager(localRegionID string, dbPath string) (*Manager, error) {
	m := &Manager{
		localID: localRegionID,
		regions: make(map[string]*Region),
		health:  make(map[string]*Health),
		dbPath:  dbPath,
	}

	// Register local region
	m.regions[localRegionID] = &Region{
		ID:   localRegionID,
		Name: localRegionID,
	}
	m.health[localRegionID] = &Health{
		RegionID:        localRegionID,
		TransportHealth: make(map[string]float64),
	}

	if err := m.initSchema(); err != nil {
		return nil, fmt.Errorf("region: init schema: %w", err)
	}

	return m, nil
}

// LocalRegion returns the local region ID.
func (m *Manager) LocalRegion() string { return m.localID }

// AddRegion registers a new region.
func (m *Manager) AddRegion(r Region) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.regions[r.ID] = &r
	if _, ok := m.health[r.ID]; !ok {
		m.health[r.ID] = &Health{
			RegionID:        r.ID,
			TransportHealth: make(map[string]float64),
		}
	}
}

// UpdateHealthFromKernel recomputes region health from kernel state.
func (m *Manager) UpdateHealthFromKernel(state *kernel.State) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Reset counters
	for _, h := range m.health {
		h.NodeCount = 0
		h.HealthyNodes = 0
		h.DegradedNodes = 0
		h.FailingNodes = 0
		h.DeadNodes = 0
	}

	// Count nodes by region
	for _, node := range state.NodeRegistry {
		regionID := node.Region
		if regionID == "" {
			regionID = m.localID
		}

		h, ok := m.health[regionID]
		if !ok {
			h = &Health{
				RegionID:        regionID,
				TransportHealth: make(map[string]float64),
			}
			m.health[regionID] = h
		}
		h.NodeCount++

		// Classify node using score if available
		if score, ok := state.NodeScores[node.NodeID]; ok {
			switch score.Classification {
			case "healthy":
				h.HealthyNodes++
			case "degraded":
				h.DegradedNodes++
			case "failing":
				h.FailingNodes++
			case "dead":
				h.DeadNodes++
			default:
				h.HealthyNodes++ // assume healthy if unclassified
			}
		} else {
			h.HealthyNodes++
		}
	}

	// Update transport health per region
	for transport, score := range state.TransportScores {
		// Assign to local region (cross-region transport mapping would need config)
		h := m.health[m.localID]
		if h != nil {
			h.TransportHealth[transport] = score.HealthScore
		}
	}

	// Compute overall health for each region
	now := time.Now().UTC()
	for _, h := range m.health {
		if h.NodeCount > 0 {
			h.OverallHealth = float64(h.HealthyNodes) / float64(h.NodeCount)
		} else {
			h.OverallHealth = 1.0 // no nodes = no problems
		}
		h.Degraded = h.OverallHealth < 0.5
		h.LastUpdateAt = now

		// Check region isolation
		if r, ok := m.regions[h.RegionID]; ok {
			h.Isolated = r.Isolated
		}
	}
}

// RegionHealth returns health for a specific region.
func (m *Manager) RegionHealth(regionID string) (*Health, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.health[regionID]
	if !ok {
		return nil, false
	}
	cp := *h
	return &cp, true
}

// AllRegionHealth returns health for all known regions.
func (m *Manager) AllRegionHealth() []Health {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Health, 0, len(m.health))
	for _, h := range m.health {
		out = append(out, *h)
	}
	return out
}

// GlobalTopology computes a global topology view.
func (m *Manager) ComputeGlobalTopology() GlobalTopology {
	m.mu.RLock()
	defer m.mu.RUnlock()

	topo := GlobalTopology{
		ComputedAt: time.Now().UTC(),
	}

	for _, h := range m.health {
		topo.Regions = append(topo.Regions, *h)
		topo.TotalNodes += h.NodeCount
		topo.TotalHealthy += h.HealthyNodes

		if h.Degraded {
			topo.DegradedRegions = append(topo.DegradedRegions, h.RegionID)
		}
		if h.Isolated {
			topo.IsolatedRegions = append(topo.IsolatedRegions, h.RegionID)
		}
	}

	if topo.TotalNodes > 0 {
		topo.GlobalHealth = float64(topo.TotalHealthy) / float64(topo.TotalNodes)
	} else {
		topo.GlobalHealth = 1.0
	}

	topo.CrossRegionLinks = make([]CrossRegionLink, len(m.links))
	copy(topo.CrossRegionLinks, m.links)

	return topo
}

// UpdateCrossRegionLink records connectivity between two regions.
func (m *Manager) UpdateCrossRegionLink(link CrossRegionLink) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, existing := range m.links {
		if (existing.RegionA == link.RegionA && existing.RegionB == link.RegionB) ||
			(existing.RegionA == link.RegionB && existing.RegionB == link.RegionA) {
			m.links[i] = link
			return
		}
	}
	m.links = append(m.links, link)
}

// IsRegionDegraded returns whether a specific region is degraded.
func (m *Manager) IsRegionDegraded(regionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.health[regionID]
	if !ok {
		return false
	}
	return h.Degraded
}

// PreferredRegion returns the region with the best health for fallback.
func (m *Manager) PreferredRegion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bestID := m.localID
	bestHealth := 0.0

	for id, h := range m.health {
		if h.Isolated {
			continue
		}
		if h.OverallHealth > bestHealth {
			bestHealth = h.OverallHealth
			bestID = id
		}
	}

	return bestID
}

// ─── Persistence ─────────────────────────────────────────────────────────────

func (m *Manager) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS kernel_regions (
	region_id TEXT PRIMARY KEY,
	name      TEXT NOT NULL DEFAULT '',
	priority  INTEGER NOT NULL DEFAULT 0,
	isolated  INTEGER NOT NULL DEFAULT 0,
	metadata  TEXT NOT NULL DEFAULT '{}'
);
`
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", m.dbPath)
	cmd.Stdin = strings.NewReader(schema)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite exec: %w: %s", err, out)
	}
	return nil
}

// PersistRegion saves a region to the database.
func (m *Manager) PersistRegion(r Region) error {
	metaJSON, _ := json.Marshal(r.Metadata)
	isolated := 0
	if r.Isolated {
		isolated = 1
	}
	sql := fmt.Sprintf(
		"INSERT OR REPLACE INTO kernel_regions (region_id, name, priority, isolated, metadata) VALUES ('%s', '%s', %d, %d, '%s');",
		sqlEscape(r.ID), sqlEscape(r.Name), r.Priority, isolated, sqlEscape(string(metaJSON)),
	)
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", m.dbPath, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite exec: %w: %s", err, out)
	}
	return nil
}

func sqlEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
