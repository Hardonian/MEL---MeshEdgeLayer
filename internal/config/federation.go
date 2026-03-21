package config

// FederationConfig holds distributed kernel and federation settings.
type FederationConfig struct {
	// Enabled activates federation features.
	Enabled bool `json:"enabled"`

	// NodeID is this instance's unique identifier. Auto-generated if empty.
	NodeID string `json:"node_id"`

	// NodeName is a human-readable name for this instance.
	NodeName string `json:"node_name"`

	// Region is this instance's region identifier.
	Region string `json:"region"`

	// ListenAddr is the address for the federation API (separate from main API).
	ListenAddr string `json:"listen_addr"`

	// Peers lists known federation peers.
	Peers []FederationPeerConfig `json:"peers,omitempty"`

	// HeartbeatIntervalSeconds is how often to send heartbeats.
	HeartbeatIntervalSeconds int `json:"heartbeat_interval_seconds"`

	// SuspectAfterMissed is missed heartbeats before suspecting a peer.
	SuspectAfterMissed int `json:"suspect_after_missed"`

	// PartitionAfterMissed is missed heartbeats before declaring partition.
	PartitionAfterMissed int `json:"partition_after_missed"`

	// SyncBatchSize is max events per sync request.
	SyncBatchSize int `json:"sync_batch_size"`

	// SyncIntervalSeconds is how often to pull from peers.
	SyncIntervalSeconds int `json:"sync_interval_seconds"`

	// EventLogRetentionDays is how long kernel events are retained.
	EventLogRetentionDays int `json:"event_log_retention_days"`

	// SnapshotIntervalEvents is how many events between snapshots.
	SnapshotIntervalEvents int `json:"snapshot_interval_events"`

	// SnapshotRetentionCount is how many snapshots to retain.
	SnapshotRetentionCount int `json:"snapshot_retention_count"`

	// SplitBrainPolicy controls behavior during detected partitions.
	SplitBrainPolicy SplitBrainPolicyConfig `json:"split_brain_policy"`
}

// FederationPeerConfig is the static configuration for a known peer.
type FederationPeerConfig struct {
	NodeID      string   `json:"node_id"`
	Name        string   `json:"name"`
	Endpoint    string   `json:"endpoint"`
	Region      string   `json:"region"`
	TrustLevel  int      `json:"trust_level"`
	SyncTypes   []string `json:"sync_types,omitempty"`
	SyncRegions []string `json:"sync_regions,omitempty"`
}

// SplitBrainPolicyConfig defines behavior during network partitions.
type SplitBrainPolicyConfig struct {
	RestrictAutopilot    bool `json:"restrict_autopilot"`
	RequireApproval      bool `json:"require_approval"`
	AlertOperator        bool `json:"alert_operator"`
	MaxAutonomousActions int  `json:"max_autonomous_actions"`
}

func defaultFederationConfig() FederationConfig {
	return FederationConfig{
		Enabled:                  false,
		Region:                   "default",
		HeartbeatIntervalSeconds: 30,
		SuspectAfterMissed:       3,
		PartitionAfterMissed:     10,
		SyncBatchSize:            100,
		SyncIntervalSeconds:      60,
		EventLogRetentionDays:    14,
		SnapshotIntervalEvents:   1000,
		SnapshotRetentionCount:   10,
		SplitBrainPolicy: SplitBrainPolicyConfig{
			RestrictAutopilot:    true,
			RequireApproval:      false,
			AlertOperator:        true,
			MaxAutonomousActions: 5,
		},
	}
}

func normalizeFederation(cfg *Config) {
	defaults := defaultFederationConfig()
	if cfg.Federation.Region == "" {
		cfg.Federation.Region = defaults.Region
	}
	if cfg.Federation.HeartbeatIntervalSeconds <= 0 {
		cfg.Federation.HeartbeatIntervalSeconds = defaults.HeartbeatIntervalSeconds
	}
	if cfg.Federation.SuspectAfterMissed <= 0 {
		cfg.Federation.SuspectAfterMissed = defaults.SuspectAfterMissed
	}
	if cfg.Federation.PartitionAfterMissed <= 0 {
		cfg.Federation.PartitionAfterMissed = defaults.PartitionAfterMissed
	}
	if cfg.Federation.SyncBatchSize <= 0 {
		cfg.Federation.SyncBatchSize = defaults.SyncBatchSize
	}
	if cfg.Federation.SyncIntervalSeconds <= 0 {
		cfg.Federation.SyncIntervalSeconds = defaults.SyncIntervalSeconds
	}
	if cfg.Federation.EventLogRetentionDays <= 0 {
		cfg.Federation.EventLogRetentionDays = defaults.EventLogRetentionDays
	}
	if cfg.Federation.SnapshotIntervalEvents <= 0 {
		cfg.Federation.SnapshotIntervalEvents = defaults.SnapshotIntervalEvents
	}
	if cfg.Federation.SnapshotRetentionCount <= 0 {
		cfg.Federation.SnapshotRetentionCount = defaults.SnapshotRetentionCount
	}
	if cfg.Federation.SplitBrainPolicy.MaxAutonomousActions <= 0 {
		cfg.Federation.SplitBrainPolicy.MaxAutonomousActions = defaults.SplitBrainPolicy.MaxAutonomousActions
	}
}
