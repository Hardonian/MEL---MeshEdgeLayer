// Package federation implements the MEL multi-instance federation model.
//
// Federation is NOT naive clustering. Each MEL instance:
//   - operates independently
//   - maintains its own event log and state
//   - may share selected data with peers
//
// Federation defines:
//   - peer discovery and management
//   - trust boundaries between instances
//   - sync scopes (what data is shared)
//   - conflict resolution rules
//   - split-brain detection and safety
package federation

import (
	"time"
)

// ─── Peer Types ──────────────────────────────────────────────────────────────

// PeerState represents the known state of a federation peer.
type PeerState string

const (
	PeerStateActive       PeerState = "active"
	PeerStateSuspected    PeerState = "suspected"    // missed heartbeats
	PeerStatePartitioned  PeerState = "partitioned"  // confirmed unreachable
	PeerStateDecommission PeerState = "decommission" // graceful removal
	PeerStateUnknown      PeerState = "unknown"
)

// Peer represents a known federation peer (another MEL instance).
type Peer struct {
	// NodeID uniquely identifies the peer MEL instance.
	NodeID string `json:"node_id"`

	// Name is a human-readable label for the peer.
	Name string `json:"name"`

	// Endpoint is the base URL for peer API communication.
	Endpoint string `json:"endpoint"`

	// Region is the peer's region identifier.
	Region string `json:"region"`

	// State is the current connectivity state.
	State PeerState `json:"state"`

	// LastSeen is the last time a heartbeat or sync was received.
	LastSeen time.Time `json:"last_seen"`

	// LastSyncSeq is the last event sequence number synced from this peer.
	LastSyncSeq uint64 `json:"last_sync_seq"`

	// TrustLevel controls what this peer is allowed to share/influence.
	// 0 = untrusted, 1 = read-only, 2 = full sync, 3 = authority
	TrustLevel int `json:"trust_level"`

	// SyncScope defines what event types are synced with this peer.
	SyncScope SyncScope `json:"sync_scope"`

	// JoinedAt is when this peer was first registered.
	JoinedAt time.Time `json:"joined_at"`

	// Metadata holds peer-specific key-value pairs.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ─── Sync Scope ──────────────────────────────────────────────────────────────

// SyncScope defines what data is shared between federated peers.
type SyncScope struct {
	// EventTypes lists which event types are replicated to/from this peer.
	EventTypes []string `json:"event_types,omitempty"`

	// Regions limits sync to events from specific regions.
	Regions []string `json:"regions,omitempty"`

	// Transports limits sync to events about specific transports.
	Transports []string `json:"transports,omitempty"`

	// NodeSubsets limits sync to events about specific node ranges.
	NodeSubsets []string `json:"node_subsets,omitempty"`

	// PolicyDomains limits sync to specific policy domains.
	PolicyDomains []string `json:"policy_domains,omitempty"`

	// ExcludeTypes lists event types that should never be synced.
	ExcludeTypes []string `json:"exclude_types,omitempty"`
}

// Matches returns true if the given event type, region, and transport
// fall within this sync scope.
func (ss SyncScope) Matches(eventType, region, transport string) bool {
	// Check exclusions first
	for _, ex := range ss.ExcludeTypes {
		if ex == eventType {
			return false
		}
	}

	// If event types specified, must match
	if len(ss.EventTypes) > 0 {
		found := false
		for _, et := range ss.EventTypes {
			if et == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// If regions specified, must match
	if len(ss.Regions) > 0 && region != "" {
		found := false
		for _, r := range ss.Regions {
			if r == region {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// If transports specified, must match
	if len(ss.Transports) > 0 && transport != "" {
		found := false
		for _, t := range ss.Transports {
			if t == transport {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// ─── Sync Messages ───────────────────────────────────────────────────────────

// SyncRequest is sent by a peer to request events.
type SyncRequest struct {
	// FromNodeID identifies the requesting peer.
	FromNodeID string `json:"from_node_id"`

	// AfterSequence requests events after this sequence number
	// (from the perspective of the source node's log).
	AfterSequence uint64 `json:"after_sequence"`

	// MaxEvents limits the number of events returned.
	MaxEvents int `json:"max_events"`

	// Scope restricts what events are requested.
	Scope SyncScope `json:"scope"`

	// RequestID for idempotency.
	RequestID string `json:"request_id"`
}

// SyncResponse is the reply to a SyncRequest.
type SyncResponse struct {
	// FromNodeID identifies the responding peer.
	FromNodeID string `json:"from_node_id"`

	// Events is the batch of events to sync.
	Events []SyncEvent `json:"events"`

	// LastSequence is the highest sequence number included.
	LastSequence uint64 `json:"last_sequence"`

	// HasMore indicates more events are available.
	HasMore bool `json:"has_more"`

	// RequestID echoes the request for correlation.
	RequestID string `json:"request_id"`
}

// SyncEvent wraps a kernel event for cross-node sync with dedup metadata.
type SyncEvent struct {
	EventID      string `json:"event_id"`
	SequenceNum  uint64 `json:"sequence_num"`
	EventType    string `json:"event_type"`
	Timestamp    string `json:"timestamp"`
	LogicalClock uint64 `json:"logical_clock"`
	SourceNodeID string `json:"source_node_id"`
	SourceRegion string `json:"source_region"`
	Subject      string `json:"subject"`
	Data         string `json:"data"`
	Checksum     string `json:"checksum"`
}

// ─── Heartbeat ───────────────────────────────────────────────────────────────

// Heartbeat is the periodic liveness signal between peers.
type Heartbeat struct {
	NodeID          string    `json:"node_id"`
	Region          string    `json:"region"`
	Timestamp       time.Time `json:"timestamp"`
	LastSequenceNum uint64    `json:"last_sequence_num"`
	LogicalClock    uint64    `json:"logical_clock"`
	State           string    `json:"state"` // healthy, degraded, partitioned
	PolicyVersion   string    `json:"policy_version"`
	NodeCount       int       `json:"node_count"`
	EventCount      uint64    `json:"event_count"`
}

// ─── Conflict Types ──────────────────────────────────────────────────────────

// ConflictType classifies a detected conflict between peers.
type ConflictType string

const (
	ConflictDuplicateAction  ConflictType = "duplicate_action"
	ConflictDivergentScore   ConflictType = "divergent_score"
	ConflictDivergentPolicy  ConflictType = "divergent_policy"
	ConflictSplitBrain       ConflictType = "split_brain"
	ConflictDuplicateEvent   ConflictType = "duplicate_event"
)

// Conflict represents a detected inconsistency between federated nodes.
type Conflict struct {
	ID           string       `json:"id"`
	Type         ConflictType `json:"type"`
	DetectedAt   time.Time    `json:"detected_at"`
	NodeA        string       `json:"node_a"`
	NodeB        string       `json:"node_b"`
	Description  string       `json:"description"`
	Resolution   string       `json:"resolution,omitempty"`
	ResolvedAt   time.Time    `json:"resolved_at,omitempty"`
	AutoResolved bool         `json:"auto_resolved"`
}

// ─── Federation Config ───────────────────────────────────────────────────────

// Config holds federation-specific configuration.
type Config struct {
	// Enabled activates federation features.
	Enabled bool `json:"enabled"`

	// NodeID is this instance's unique identifier. Auto-generated if empty.
	NodeID string `json:"node_id"`

	// NodeName is a human-readable name for this instance.
	NodeName string `json:"node_name"`

	// Region is this instance's region identifier.
	Region string `json:"region"`

	// ListenAddr is the address for the federation API.
	ListenAddr string `json:"listen_addr"`

	// Peers lists known federation peers.
	Peers []PeerConfig `json:"peers,omitempty"`

	// HeartbeatIntervalSeconds is how often to send heartbeats.
	HeartbeatIntervalSeconds int `json:"heartbeat_interval_seconds"`

	// SuspectAfterMissed is the number of missed heartbeats before
	// marking a peer as suspected.
	SuspectAfterMissed int `json:"suspect_after_missed"`

	// PartitionAfterMissed is the number of missed heartbeats before
	// marking a peer as partitioned.
	PartitionAfterMissed int `json:"partition_after_missed"`

	// SyncBatchSize is the max events per sync request.
	SyncBatchSize int `json:"sync_batch_size"`

	// SyncIntervalSeconds is how often to pull from peers.
	SyncIntervalSeconds int `json:"sync_interval_seconds"`

	// DefaultSyncScope is the default scope for new peers.
	DefaultSyncScope SyncScope `json:"default_sync_scope"`

	// SplitBrainPolicy controls behavior during detected partitions.
	SplitBrainPolicy SplitBrainPolicy `json:"split_brain_policy"`
}

// PeerConfig is the static configuration for a known peer.
type PeerConfig struct {
	NodeID     string    `json:"node_id"`
	Name       string    `json:"name"`
	Endpoint   string    `json:"endpoint"`
	Region     string    `json:"region"`
	TrustLevel int       `json:"trust_level"`
	SyncScope  SyncScope `json:"sync_scope"`
}

// SplitBrainPolicy defines behavior during network partitions.
type SplitBrainPolicy struct {
	// RestrictAutopilot: when true, autopilot actions require approval
	// during detected partitions.
	RestrictAutopilot bool `json:"restrict_autopilot"`

	// RequireApproval: when true, all actions require operator approval
	// during detected partitions.
	RequireApproval bool `json:"require_approval"`

	// AlertOperator: emit alerts when split-brain is detected.
	AlertOperator bool `json:"alert_operator"`

	// MaxAutonomousActions is the max actions allowed per node during
	// partition before requiring approval.
	MaxAutonomousActions int `json:"max_autonomous_actions"`
}

// DefaultConfig returns safe default federation configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:                  false,
		Region:                   "default",
		HeartbeatIntervalSeconds: 30,
		SuspectAfterMissed:       3,
		PartitionAfterMissed:     10,
		SyncBatchSize:            100,
		SyncIntervalSeconds:      60,
		SplitBrainPolicy: SplitBrainPolicy{
			RestrictAutopilot:    true,
			RequireApproval:      false,
			AlertOperator:        true,
			MaxAutonomousActions: 5,
		},
	}
}
