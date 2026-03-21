package federation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/eventlog"
	"github.com/mel-project/mel/internal/kernel"
	"github.com/mel-project/mel/internal/logging"
)

// Manager coordinates federation activities: peer tracking, heartbeats,
// event sync, conflict detection, and split-brain safety.
type Manager struct {
	mu     sync.RWMutex
	config Config
	nodeID string
	region string
	log    *logging.Logger
	evtLog *eventlog.Log
	kernel *kernel.Kernel

	peers     map[string]*Peer
	conflicts []Conflict

	// dedup tracks event IDs already ingested to prevent duplicates.
	dedup map[string]bool

	// splitBrainDetected is true when a partition is suspected.
	splitBrainDetected bool

	// autonomousActionCount tracks actions taken during partition.
	autonomousActionCount int

	// dbPath for persistent peer state
	dbPath string

	// HTTP client for peer communication
	client *http.Client
}

// NewManager creates a federation manager.
func NewManager(cfg Config, log *logging.Logger, evtLog *eventlog.Log, k *kernel.Kernel, dbPath string) (*Manager, error) {
	nodeID := cfg.NodeID
	if nodeID == "" {
		nodeID = kernel.NewNodeID()
	}

	m := &Manager{
		config:  cfg,
		nodeID:  nodeID,
		region:  cfg.Region,
		log:     log,
		evtLog:  evtLog,
		kernel:  k,
		peers:   make(map[string]*Peer),
		dedup:   make(map[string]bool),
		dbPath:  dbPath,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	if err := m.initSchema(); err != nil {
		return nil, fmt.Errorf("federation: init schema: %w", err)
	}

	// Load configured peers
	for _, pc := range cfg.Peers {
		m.peers[pc.NodeID] = &Peer{
			NodeID:     pc.NodeID,
			Name:       pc.Name,
			Endpoint:   pc.Endpoint,
			Region:     pc.Region,
			State:      PeerStateUnknown,
			TrustLevel: pc.TrustLevel,
			SyncScope:  pc.SyncScope,
			JoinedAt:   time.Now().UTC(),
		}
	}

	// Load persisted peers
	if err := m.loadPersistedPeers(); err != nil {
		log.Error("federation_load_peers", "failed to load persisted peers", map[string]any{"error": err.Error()})
	}

	return m, nil
}

// NodeID returns this manager's node identifier.
func (m *Manager) NodeID() string { return m.nodeID }

// Region returns this manager's region.
func (m *Manager) Region() string { return m.region }

// ─── Peer Management ─────────────────────────────────────────────────────────

// Peers returns a snapshot of all known peers.
func (m *Manager) Peers() []Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Peer, 0, len(m.peers))
	for _, p := range m.peers {
		out = append(out, *p)
	}
	return out
}

// AddPeer registers a new federation peer.
func (m *Manager) AddPeer(peer Peer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if peer.NodeID == m.nodeID {
		return fmt.Errorf("cannot add self as peer")
	}

	peer.JoinedAt = time.Now().UTC()
	if peer.State == "" {
		peer.State = PeerStateUnknown
	}
	m.peers[peer.NodeID] = &peer

	// Persist
	return m.persistPeer(&peer)
}

// RemovePeer removes a federation peer.
func (m *Manager) RemovePeer(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.peers, nodeID)
	_ = m.deletePeer(nodeID)
}

// PeerByID returns a specific peer.
func (m *Manager) PeerByID(nodeID string) (*Peer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.peers[nodeID]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}

// ─── Heartbeat Processing ────────────────────────────────────────────────────

// ProcessHeartbeat handles a received heartbeat from a peer.
func (m *Manager) ProcessHeartbeat(hb Heartbeat) {
	m.mu.Lock()
	defer m.mu.Unlock()

	peer, ok := m.peers[hb.NodeID]
	if !ok {
		// Auto-register unknown peer with minimal trust
		peer = &Peer{
			NodeID:     hb.NodeID,
			Region:     hb.Region,
			State:      PeerStateActive,
			TrustLevel: 1, // read-only
			JoinedAt:   time.Now().UTC(),
			SyncScope:  m.config.DefaultSyncScope,
		}
		m.peers[hb.NodeID] = peer
		m.log.Info("federation_peer_discovered", "new peer discovered via heartbeat", map[string]any{
			"peer_id": hb.NodeID,
			"region":  hb.Region,
		})
	}

	peer.LastSeen = hb.Timestamp
	peer.State = PeerStateActive
	peer.LastSyncSeq = hb.LastSequenceNum

	// Check for policy divergence
	if m.kernel != nil {
		localState := m.kernel.State()
		if localState.PolicyVersion != "" && hb.PolicyVersion != "" &&
			localState.PolicyVersion != hb.PolicyVersion {
			m.conflicts = append(m.conflicts, Conflict{
				ID:          kernel.NewEventID(),
				Type:        ConflictDivergentPolicy,
				DetectedAt:  time.Now().UTC(),
				NodeA:       m.nodeID,
				NodeB:       hb.NodeID,
				Description: fmt.Sprintf("policy divergence: local=%s remote=%s", localState.PolicyVersion, hb.PolicyVersion),
			})
		}
	}
}

// GenerateHeartbeat creates a heartbeat for this node.
func (m *Manager) GenerateHeartbeat() Heartbeat {
	hb := Heartbeat{
		NodeID:    m.nodeID,
		Region:    m.region,
		Timestamp: time.Now().UTC(),
		State:     "healthy",
	}

	if m.evtLog != nil {
		hb.LastSequenceNum = m.evtLog.LastSequenceNum()
	}

	if m.kernel != nil {
		state := m.kernel.State()
		hb.LogicalClock = state.LogicalClock
		hb.PolicyVersion = state.PolicyVersion
		hb.NodeCount = len(state.NodeRegistry)
	}

	if m.splitBrainDetected {
		hb.State = "partitioned"
	}

	return hb
}

// ─── Event Sync ──────────────────────────────────────────────────────────────

// HandleSyncRequest processes an incoming sync request from a peer.
func (m *Manager) HandleSyncRequest(req SyncRequest) (*SyncResponse, error) {
	if m.evtLog == nil {
		return nil, fmt.Errorf("event log not available")
	}

	// Verify peer trust
	m.mu.RLock()
	peer, ok := m.peers[req.FromNodeID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown peer: %s", req.FromNodeID)
	}
	if peer.TrustLevel < 1 {
		return nil, fmt.Errorf("peer %s has insufficient trust level", req.FromNodeID)
	}

	maxEvents := m.config.SyncBatchSize
	if req.MaxEvents > 0 && req.MaxEvents < maxEvents {
		maxEvents = req.MaxEvents
	}

	// Query local event log
	filter := eventlog.QueryFilter{
		AfterSequence: req.AfterSequence,
		Limit:         maxEvents + 1, // +1 to detect hasMore
	}

	events, err := m.evtLog.Query(filter)
	if err != nil {
		return nil, fmt.Errorf("sync query: %w", err)
	}

	hasMore := len(events) > maxEvents
	if hasMore {
		events = events[:maxEvents]
	}

	// Filter by sync scope
	var filtered []SyncEvent
	for _, evt := range events {
		if !req.Scope.Matches(string(evt.Type), evt.SourceRegion, evt.Subject) {
			continue
		}
		filtered = append(filtered, SyncEvent{
			EventID:      evt.ID,
			SequenceNum:  evt.SequenceNum,
			EventType:    string(evt.Type),
			Timestamp:    evt.Timestamp.Format(time.RFC3339Nano),
			LogicalClock: evt.LogicalClock,
			SourceNodeID: evt.SourceNodeID,
			SourceRegion: evt.SourceRegion,
			Subject:      evt.Subject,
			Data:         string(evt.Data),
			Checksum:     evt.Checksum,
		})
	}

	lastSeq := uint64(0)
	if len(filtered) > 0 {
		lastSeq = filtered[len(filtered)-1].SequenceNum
	}

	return &SyncResponse{
		FromNodeID:   m.nodeID,
		Events:       filtered,
		LastSequence: lastSeq,
		HasMore:      hasMore,
		RequestID:    req.RequestID,
	}, nil
}

// IngestSyncedEvents processes events received from a peer sync.
// Returns the number of new events ingested (duplicates are skipped).
func (m *Manager) IngestSyncedEvents(fromNodeID string, events []SyncEvent) (int, error) {
	if m.evtLog == nil {
		return 0, fmt.Errorf("event log not available")
	}

	m.mu.Lock()
	peer, ok := m.peers[fromNodeID]
	if !ok {
		m.mu.Unlock()
		return 0, fmt.Errorf("unknown peer: %s", fromNodeID)
	}
	m.mu.Unlock()

	ingested := 0
	for _, se := range events {
		// Duplicate detection
		m.mu.Lock()
		if m.dedup[se.EventID] {
			m.mu.Unlock()
			continue
		}
		m.dedup[se.EventID] = true
		// Bound dedup map size
		if len(m.dedup) > 100000 {
			// Evict oldest entries (simple: clear half)
			count := 0
			for k := range m.dedup {
				if count > 50000 {
					break
				}
				delete(m.dedup, k)
				count++
			}
		}
		m.mu.Unlock()

		// Convert to kernel event
		ts, _ := time.Parse(time.RFC3339Nano, se.Timestamp)
		evt := &kernel.Event{
			ID:           se.EventID,
			Type:         kernel.EventType(se.EventType),
			Timestamp:    ts,
			LogicalClock: se.LogicalClock,
			SourceNodeID: se.SourceNodeID,
			SourceRegion: se.SourceRegion,
			Subject:      se.Subject,
			Data:         []byte(se.Data),
			Checksum:     se.Checksum,
		}

		// Append to local event log
		if _, err := m.evtLog.Append(evt); err != nil {
			m.log.Error("federation_ingest_failed", "failed to ingest synced event", map[string]any{
				"event_id":   se.EventID,
				"from_peer":  fromNodeID,
				"error":      err.Error(),
			})
			continue
		}

		// Apply to kernel
		if m.kernel != nil {
			m.kernel.Apply(*evt)
		}

		ingested++
	}

	// Update peer's last sync sequence
	m.mu.Lock()
	if len(events) > 0 {
		peer.LastSyncSeq = events[len(events)-1].SequenceNum
	}
	m.mu.Unlock()

	return ingested, nil
}

// ─── Split-Brain Detection ───────────────────────────────────────────────────

// CheckPartitions evaluates peer states and detects potential split-brain conditions.
func (m *Manager) CheckPartitions() []Conflict {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	heartbeatInterval := time.Duration(m.config.HeartbeatIntervalSeconds) * time.Second
	if heartbeatInterval == 0 {
		heartbeatInterval = 30 * time.Second
	}

	var newConflicts []Conflict
	partitionedCount := 0

	for _, peer := range m.peers {
		if peer.State == PeerStateDecommission {
			continue
		}

		sinceSeen := now.Sub(peer.LastSeen)

		suspectThreshold := heartbeatInterval * time.Duration(m.config.SuspectAfterMissed)
		partitionThreshold := heartbeatInterval * time.Duration(m.config.PartitionAfterMissed)

		switch {
		case sinceSeen > partitionThreshold && peer.State != PeerStatePartitioned:
			peer.State = PeerStatePartitioned
			partitionedCount++
			m.log.Warn("federation_peer_partitioned", "peer marked as partitioned", map[string]any{
				"peer_id":    peer.NodeID,
				"last_seen":  peer.LastSeen.Format(time.RFC3339),
				"since_seen": sinceSeen.String(),
			})

		case sinceSeen > suspectThreshold && peer.State == PeerStateActive:
			peer.State = PeerStateSuspected
			m.log.Warn("federation_peer_suspected", "peer connectivity suspected", map[string]any{
				"peer_id":    peer.NodeID,
				"last_seen":  peer.LastSeen.Format(time.RFC3339),
				"since_seen": sinceSeen.String(),
			})

		case peer.State == PeerStatePartitioned:
			partitionedCount++
		}
	}

	// Detect split-brain: more than half of peers partitioned
	totalPeers := len(m.peers)
	if totalPeers > 0 && partitionedCount > totalPeers/2 {
		if !m.splitBrainDetected {
			m.splitBrainDetected = true
			m.autonomousActionCount = 0
			conflict := Conflict{
				ID:          kernel.NewEventID(),
				Type:        ConflictSplitBrain,
				DetectedAt:  now,
				NodeA:       m.nodeID,
				NodeB:       "multiple",
				Description: fmt.Sprintf("split-brain detected: %d/%d peers partitioned", partitionedCount, totalPeers),
			}
			newConflicts = append(newConflicts, conflict)
			m.conflicts = append(m.conflicts, conflict)
			m.log.Error("federation_split_brain", "split-brain condition detected", map[string]any{
				"partitioned": partitionedCount,
				"total_peers": totalPeers,
			})
		}
	} else if m.splitBrainDetected && partitionedCount <= totalPeers/2 {
		m.splitBrainDetected = false
		m.log.Info("federation_split_brain_resolved", "split-brain condition resolved", nil)
	}

	return newConflicts
}

// IsSplitBrain returns whether a split-brain condition is currently detected.
func (m *Manager) IsSplitBrain() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.splitBrainDetected
}

// CanExecuteAutonomously returns whether this node can execute an action
// autonomously given the current partition state.
func (m *Manager) CanExecuteAutonomously() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.splitBrainDetected {
		return true
	}

	policy := m.config.SplitBrainPolicy
	if policy.RequireApproval {
		return false
	}
	if policy.RestrictAutopilot && m.autonomousActionCount >= policy.MaxAutonomousActions {
		return false
	}
	return true
}

// RecordAutonomousAction increments the autonomous action counter
// (used during partition to enforce limits).
func (m *Manager) RecordAutonomousAction() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autonomousActionCount++
}

// Conflicts returns all detected conflicts.
func (m *Manager) Conflicts() []Conflict {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Conflict, len(m.conflicts))
	copy(out, m.conflicts)
	return out
}

// ─── Background Workers ──────────────────────────────────────────────────────

// Run starts federation background workers. Blocks until ctx is cancelled.
func (m *Manager) Run(ctx context.Context) {
	if !m.config.Enabled {
		return
	}

	// Start heartbeat sender
	go m.heartbeatLoop(ctx)

	// Start sync puller
	go m.syncLoop(ctx)

	// Start partition checker
	go m.partitionCheckLoop(ctx)

	<-ctx.Done()
}

func (m *Manager) heartbeatLoop(ctx context.Context) {
	interval := time.Duration(m.config.HeartbeatIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.sendHeartbeats()
		}
	}
}

func (m *Manager) syncLoop(ctx context.Context) {
	interval := time.Duration(m.config.SyncIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.pullFromPeers()
		}
	}
}

func (m *Manager) partitionCheckLoop(ctx context.Context) {
	interval := time.Duration(m.config.HeartbeatIntervalSeconds) * time.Second * 2
	if interval <= 0 {
		interval = 60 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.CheckPartitions()
		}
	}
}

func (m *Manager) sendHeartbeats() {
	hb := m.GenerateHeartbeat()
	hbJSON, _ := json.Marshal(hb)

	m.mu.RLock()
	peers := make([]*Peer, 0, len(m.peers))
	for _, p := range m.peers {
		if p.State != PeerStateDecommission {
			peers = append(peers, p)
		}
	}
	m.mu.RUnlock()

	for _, peer := range peers {
		go func(p *Peer) {
			url := strings.TrimRight(p.Endpoint, "/") + "/api/v1/federation/heartbeat"
			resp, err := m.client.Post(url, "application/json", strings.NewReader(string(hbJSON)))
			if err != nil {
				return // Peer unreachable; partition detection handles this
			}
			defer resp.Body.Close()
			_, _ = io.ReadAll(resp.Body)
		}(peer)
	}
}

func (m *Manager) pullFromPeers() {
	m.mu.RLock()
	peers := make([]*Peer, 0)
	for _, p := range m.peers {
		if p.State == PeerStateActive || p.State == PeerStateSuspected {
			if p.TrustLevel >= 1 {
				peers = append(peers, p)
			}
		}
	}
	m.mu.RUnlock()

	for _, peer := range peers {
		go func(p *Peer) {
			req := SyncRequest{
				FromNodeID:    m.nodeID,
				AfterSequence: p.LastSyncSeq,
				MaxEvents:     m.config.SyncBatchSize,
				Scope:         p.SyncScope,
				RequestID:     kernel.NewEventID(),
			}
			reqJSON, _ := json.Marshal(req)

			url := strings.TrimRight(p.Endpoint, "/") + "/api/v1/federation/sync"
			resp, err := m.client.Post(url, "application/json", strings.NewReader(string(reqJSON)))
			if err != nil {
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}

			var syncResp SyncResponse
			if err := json.Unmarshal(body, &syncResp); err != nil {
				return
			}

			if len(syncResp.Events) > 0 {
				ingested, err := m.IngestSyncedEvents(p.NodeID, syncResp.Events)
				if err == nil && ingested > 0 {
					m.log.Info("federation_sync_complete", "synced events from peer", map[string]any{
						"peer_id":  p.NodeID,
						"ingested": ingested,
						"total":    len(syncResp.Events),
					})
				}
			}
		}(peer)
	}
}

// ─── Status ──────────────────────────────────────────────────────────────────

// Status returns the current federation status.
type Status struct {
	NodeID      string       `json:"node_id"`
	Region      string       `json:"region"`
	Enabled     bool         `json:"enabled"`
	PeerCount   int          `json:"peer_count"`
	Peers       []PeerStatus `json:"peers"`
	SplitBrain  bool         `json:"split_brain"`
	ConflictCount int        `json:"conflict_count"`
}

// PeerStatus is a summary of peer state for the status API.
type PeerStatus struct {
	NodeID     string    `json:"node_id"`
	Name       string    `json:"name"`
	Region     string    `json:"region"`
	State      PeerState `json:"state"`
	LastSeen   time.Time `json:"last_seen"`
	TrustLevel int       `json:"trust_level"`
}

func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := Status{
		NodeID:        m.nodeID,
		Region:        m.region,
		Enabled:       m.config.Enabled,
		PeerCount:     len(m.peers),
		SplitBrain:    m.splitBrainDetected,
		ConflictCount: len(m.conflicts),
	}

	for _, p := range m.peers {
		status.Peers = append(status.Peers, PeerStatus{
			NodeID:     p.NodeID,
			Name:       p.Name,
			Region:     p.Region,
			State:      p.State,
			LastSeen:   p.LastSeen,
			TrustLevel: p.TrustLevel,
		})
	}

	return status
}

// ─── Persistence ─────────────────────────────────────────────────────────────

func (m *Manager) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS federation_peers (
	node_id      TEXT PRIMARY KEY,
	name         TEXT NOT NULL DEFAULT '',
	endpoint     TEXT NOT NULL DEFAULT '',
	region       TEXT NOT NULL DEFAULT '',
	state        TEXT NOT NULL DEFAULT 'unknown',
	trust_level  INTEGER NOT NULL DEFAULT 1,
	last_seen    TEXT NOT NULL DEFAULT '',
	last_sync_seq INTEGER NOT NULL DEFAULT 0,
	sync_scope   TEXT NOT NULL DEFAULT '{}',
	joined_at    TEXT NOT NULL DEFAULT '',
	metadata     TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS federation_conflicts (
	conflict_id   TEXT PRIMARY KEY,
	conflict_type TEXT NOT NULL,
	detected_at   TEXT NOT NULL,
	node_a        TEXT NOT NULL,
	node_b        TEXT NOT NULL,
	description   TEXT NOT NULL DEFAULT '',
	resolution    TEXT NOT NULL DEFAULT '',
	resolved_at   TEXT NOT NULL DEFAULT '',
	auto_resolved INTEGER NOT NULL DEFAULT 0
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

func (m *Manager) persistPeer(p *Peer) error {
	scopeJSON, _ := json.Marshal(p.SyncScope)
	metaJSON, _ := json.Marshal(p.Metadata)
	sql := fmt.Sprintf(
		`INSERT OR REPLACE INTO federation_peers (node_id, name, endpoint, region, state, trust_level, last_seen, last_sync_seq, sync_scope, joined_at, metadata) VALUES ('%s', '%s', '%s', '%s', '%s', %d, '%s', %d, '%s', '%s', '%s');`,
		sqlEscape(p.NodeID), sqlEscape(p.Name), sqlEscape(p.Endpoint),
		sqlEscape(p.Region), string(p.State), p.TrustLevel,
		p.LastSeen.Format(time.RFC3339), p.LastSyncSeq,
		sqlEscape(string(scopeJSON)), p.JoinedAt.Format(time.RFC3339),
		sqlEscape(string(metaJSON)),
	)
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", m.dbPath, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite exec: %w: %s", err, out)
	}
	return nil
}

func (m *Manager) deletePeer(nodeID string) error {
	sql := fmt.Sprintf("DELETE FROM federation_peers WHERE node_id = '%s';", sqlEscape(nodeID))
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", m.dbPath, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite exec: %w: %s", err, out)
	}
	return nil
}

func (m *Manager) loadPersistedPeers() error {
	sql := "SELECT node_id, name, endpoint, region, state, trust_level, last_seen, last_sync_seq, sync_scope, joined_at, metadata FROM federation_peers;"
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", "-json", m.dbPath, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite query: %w: %s", err, out)
	}
	if len(out) == 0 {
		return nil
	}

	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		return fmt.Errorf("sqlite parse: %w", err)
	}

	for _, row := range rows {
		nodeID := asStr(row["node_id"])
		if _, exists := m.peers[nodeID]; exists {
			continue // Config peers take precedence
		}

		peer := &Peer{
			NodeID:   nodeID,
			Name:     asStr(row["name"]),
			Endpoint: asStr(row["endpoint"]),
			Region:   asStr(row["region"]),
			State:    PeerState(asStr(row["state"])),
		}
		if v, ok := row["trust_level"].(float64); ok {
			peer.TrustLevel = int(v)
		}
		if v, ok := row["last_sync_seq"].(float64); ok {
			peer.LastSyncSeq = uint64(v)
		}
		if ls := asStr(row["last_seen"]); ls != "" {
			if t, err := time.Parse(time.RFC3339, ls); err == nil {
				peer.LastSeen = t
			}
		}
		if ja := asStr(row["joined_at"]); ja != "" {
			if t, err := time.Parse(time.RFC3339, ja); err == nil {
				peer.JoinedAt = t
			}
		}
		if ss := asStr(row["sync_scope"]); ss != "" {
			_ = json.Unmarshal([]byte(ss), &peer.SyncScope)
		}

		m.peers[nodeID] = peer
	}

	return nil
}

func asStr(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func sqlEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
