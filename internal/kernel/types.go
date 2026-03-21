// Package kernel defines the MEL distributed control-plane kernel.
//
// The kernel is the deterministic core of MEL. It owns:
//   - event ingestion and normalization
//   - observation scoring and classification
//   - action lifecycle management
//   - evidence bundle construction
//   - policy evaluation
//   - decision generation
//
// The kernel is side-effect isolated: it emits actions but does not execute
// them directly. All inputs are events; all outputs are effects.
//
// The kernel MUST be deterministic for identical input streams and replayable.
package kernel

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// ─── Event Types ──────────────────────────────────────────────────────────────

// EventType classifies kernel events.
type EventType string

const (
	EventObservation      EventType = "observation"
	EventAnomaly          EventType = "anomaly"
	EventTopologyUpdate   EventType = "topology_update"
	EventPolicyChange     EventType = "policy_change"
	EventOperatorAction   EventType = "operator_action"
	EventApproval         EventType = "approval"
	EventRejection        EventType = "rejection"
	EventAdapterState     EventType = "adapter_state"
	EventTransportHealth  EventType = "transport_health"
	EventNodeState        EventType = "node_state"
	EventActionProposed   EventType = "action_proposed"
	EventActionExecuted   EventType = "action_executed"
	EventActionCompleted  EventType = "action_completed"
	EventFreezeCreated    EventType = "freeze_created"
	EventFreezeCleared    EventType = "freeze_cleared"
	EventMaintenanceStart EventType = "maintenance_start"
	EventMaintenanceEnd   EventType = "maintenance_end"
	EventSnapshotCreated  EventType = "snapshot_created"
	EventPeerJoined       EventType = "peer_joined"
	EventPeerLeft         EventType = "peer_left"
	EventSyncReceived     EventType = "sync_received"
	EventRegionHealth     EventType = "region_health"
)

// Event is the canonical unit of the MEL event log. Every input to the kernel
// is normalized into an Event before processing. Events are append-only,
// ordered, uniquely identifiable, and durable.
type Event struct {
	// ID is globally unique. Format: "evt-<hex>".
	ID string `json:"id"`

	// SequenceNum is locally monotonic within a single MEL instance.
	// Assigned by the event log on append.
	SequenceNum uint64 `json:"sequence_num"`

	// Type classifies this event.
	Type EventType `json:"type"`

	// Timestamp is the wall-clock time the event was created (UTC).
	Timestamp time.Time `json:"timestamp"`

	// LogicalClock is a Lamport-style counter for causal ordering across nodes.
	LogicalClock uint64 `json:"logical_clock"`

	// SourceNodeID identifies the MEL instance that originated this event.
	SourceNodeID string `json:"source_node_id"`

	// SourceRegion is the region of the originating node.
	SourceRegion string `json:"source_region,omitempty"`

	// Subject is the primary entity this event concerns (transport name,
	// node ID, action ID, etc.).
	Subject string `json:"subject,omitempty"`

	// Data is the event-type-specific payload, serialized as JSON.
	Data []byte `json:"data"`

	// Metadata holds optional key-value pairs for filtering and routing.
	Metadata map[string]string `json:"metadata,omitempty"`

	// PolicyVersion identifies the policy in effect when this event was created.
	PolicyVersion string `json:"policy_version,omitempty"`

	// CausalParent links to a prior event that caused this one (optional).
	CausalParent string `json:"causal_parent,omitempty"`

	// Checksum is SHA-256 of (ID + SequenceNum + Type + Timestamp + Data)
	// for integrity verification.
	Checksum string `json:"checksum,omitempty"`
}

// ─── Effect Types ─────────────────────────────────────────────────────────────

// EffectType classifies kernel outputs.
type EffectType string

const (
	EffectProposeAction  EffectType = "propose_action"
	EffectUpdateScore    EffectType = "update_score"
	EffectClassifyNode   EffectType = "classify_node"
	EffectEmitAlert      EffectType = "emit_alert"
	EffectRecordEvidence EffectType = "record_evidence"
	EffectUpdateState    EffectType = "update_state"
)

// Effect is a side-effect emitted by the kernel during event processing.
// Effects are NOT executed by the kernel; they are collected and dispatched
// by the adapter layer.
type Effect struct {
	ID        string            `json:"id"`
	Type      EffectType        `json:"type"`
	CausedBy  string            `json:"caused_by"` // Event ID that triggered this effect
	Timestamp time.Time         `json:"timestamp"`
	Subject   string            `json:"subject,omitempty"`
	Data      []byte            `json:"data"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ─── Node Scoring ─────────────────────────────────────────────────────────────

// NodeScore represents the kernel's computed score for a mesh node.
type NodeScore struct {
	NodeID         string    `json:"node_id"`
	Transport      string    `json:"transport"`
	HealthScore    float64   `json:"health_score"`    // 0.0 (dead) to 1.0 (healthy)
	TrustScore     float64   `json:"trust_score"`     // 0.0 (untrusted) to 1.0 (fully trusted)
	ActivityScore  float64   `json:"activity_score"`  // 0.0 (dormant) to 1.0 (highly active)
	AnomalyScore   float64   `json:"anomaly_score"`   // 0.0 (normal) to 1.0 (highly anomalous)
	CompositeScore float64   `json:"composite_score"` // weighted aggregate
	Classification string    `json:"classification"`  // healthy, degraded, failing, dead, suspicious
	UpdatedAt      time.Time `json:"updated_at"`
	EventID        string    `json:"event_id"` // event that produced this score
}

// TransportScore represents the kernel's computed score for a transport.
type TransportScore struct {
	Transport      string    `json:"transport"`
	HealthScore    float64   `json:"health_score"`
	ReliabilityPct float64   `json:"reliability_pct"`
	AnomalyRate    float64   `json:"anomaly_rate"`
	Classification string    `json:"classification"`
	UpdatedAt      time.Time `json:"updated_at"`
	EventID        string    `json:"event_id"`
}

// ─── Kernel State ─────────────────────────────────────────────────────────────

// State is the complete deterministic state of the kernel at any point in time.
// Given the same event stream and policy, the kernel MUST produce the same state.
type State struct {
	// NodeScores maps node_id to current score.
	NodeScores map[string]NodeScore `json:"node_scores"`

	// TransportScores maps transport name to current score.
	TransportScores map[string]TransportScore `json:"transport_scores"`

	// ActionStates maps action_id to lifecycle state.
	ActionStates map[string]ActionState `json:"action_states"`

	// ActiveFreezes tracks current freeze conditions.
	ActiveFreezes map[string]FreezeState `json:"active_freezes"`

	// PolicyVersion is the current policy in effect.
	PolicyVersion string `json:"policy_version"`

	// LastEventID is the ID of the most recently processed event.
	LastEventID string `json:"last_event_id"`

	// LastSequenceNum is the sequence number of the most recently processed event.
	LastSequenceNum uint64 `json:"last_sequence_num"`

	// LogicalClock is the current Lamport clock value.
	LogicalClock uint64 `json:"logical_clock"`

	// NodeRegistry maps node_num to basic node info.
	NodeRegistry map[int64]NodeInfo `json:"node_registry"`

	// RegionHealth maps region_id to health summary.
	RegionHealth map[string]RegionHealthState `json:"region_health"`
}

// ActionState tracks the lifecycle of a control action in the kernel.
type ActionState struct {
	ActionID       string    `json:"action_id"`
	ActionType     string    `json:"action_type"`
	Target         string    `json:"target"`
	Lifecycle      string    `json:"lifecycle"`
	ProposedAt     time.Time `json:"proposed_at"`
	ExecutedAt     time.Time `json:"executed_at,omitempty"`
	CompletedAt    time.Time `json:"completed_at,omitempty"`
	Result         string    `json:"result,omitempty"`
	OwnerNodeID    string    `json:"owner_node_id"`
	CoordinationID string    `json:"coordination_id,omitempty"`
}

// FreezeState tracks an active freeze in the kernel.
type FreezeState struct {
	FreezeID   string    `json:"freeze_id"`
	ScopeType  string    `json:"scope_type"`
	ScopeValue string    `json:"scope_value"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
}

// NodeInfo holds basic node registry data.
type NodeInfo struct {
	NodeNum   int64     `json:"node_num"`
	NodeID    string    `json:"node_id"`
	LongName  string    `json:"long_name"`
	ShortName string    `json:"short_name"`
	LastSeen  time.Time `json:"last_seen"`
	Region    string    `json:"region,omitempty"`
}

// RegionHealthState summarizes health for a region.
type RegionHealthState struct {
	RegionID       string    `json:"region_id"`
	NodeCount      int       `json:"node_count"`
	HealthyNodes   int       `json:"healthy_nodes"`
	DegradedNodes  int       `json:"degraded_nodes"`
	FailingNodes   int       `json:"failing_nodes"`
	OverallHealth  float64   `json:"overall_health"`
	LastUpdateAt   time.Time `json:"last_update_at"`
	Isolated       bool      `json:"isolated"`
}

// NewState returns an initialized empty kernel state.
func NewState() *State {
	return &State{
		NodeScores:      make(map[string]NodeScore),
		TransportScores: make(map[string]TransportScore),
		ActionStates:    make(map[string]ActionState),
		ActiveFreezes:   make(map[string]FreezeState),
		NodeRegistry:    make(map[int64]NodeInfo),
		RegionHealth:    make(map[string]RegionHealthState),
	}
}

// ─── ID Generation ────────────────────────────────────────────────────────────

// NewEventID generates a unique event ID.
func NewEventID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "evt-" + hex.EncodeToString(b)
}

// NewEffectID generates a unique effect ID.
func NewEffectID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "eff-" + hex.EncodeToString(b)
}

// NewNodeID generates a unique MEL instance node ID.
func NewNodeID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("mel-%s", hex.EncodeToString(b))
}
