package config

import "fmt"

type IntelligenceConfig struct {
	Scoring   IntelligenceScoringConfig   `json:"scoring"`
	Alerts    IntelligenceAlertsConfig    `json:"alerts"`
	Retention IntelligenceRetentionConfig `json:"retention"`
	Queries   IntelligenceQueryConfig     `json:"queries"`
}

type IntelligenceScoringConfig struct {
	AnomalyWindowsSeconds      []int          `json:"anomaly_windows_seconds"`
	ReasonWeights              map[string]int `json:"reason_weights"`
	DeadLetterPenalty          int            `json:"dead_letter_penalty"`
	HistoricalDeadLetterWeight int            `json:"historical_dead_letter_weight"`
	ResidualPenaltyCap         int            `json:"residual_penalty_cap"`
	ActiveEpisodePenalty       int            `json:"active_episode_penalty"`
	ActiveEpisodePenaltyCap    int            `json:"active_episode_penalty_cap"`
	ObservationDropBasePenalty int            `json:"observation_drop_base_penalty"`
	ObservationDropStepPenalty int            `json:"observation_drop_step_penalty"`
	ObservationDropPenaltyCap  int            `json:"observation_drop_penalty_cap"`
	RuntimeRetryingPenalty     int            `json:"runtime_retrying_penalty"`
	RuntimeFailedPenalty       int            `json:"runtime_failed_penalty"`
	HeartbeatPenalty120        int            `json:"heartbeat_penalty_120_seconds"`
	HeartbeatPenalty300        int            `json:"heartbeat_penalty_300_seconds"`
	HeartbeatPenalty900        int            `json:"heartbeat_penalty_900_seconds"`
}

type IntelligenceAlertsConfig struct {
	MinimumStateDurationSeconds int `json:"minimum_state_duration_seconds"`
	CooldownSeconds             int `json:"cooldown_seconds"`
	RecoveryScoreHealthy        int `json:"recovery_score_healthy"`
	RecoveryScoreDegraded       int `json:"recovery_score_degraded"`
	RecoveryScoreUnstable       int `json:"recovery_score_unstable"`
}

type IntelligenceRetentionConfig struct {
	HealthSnapshotDays    int `json:"health_snapshot_days"`
	HealthSnapshotMaxRows int `json:"health_snapshot_max_rows"`
	PruneEverySeconds     int `json:"prune_every_seconds"`
}

type IntelligenceQueryConfig struct {
	DefaultLimit int `json:"default_limit"`
	MaxLimit     int `json:"max_limit"`
}

func defaultIntelligenceConfig() IntelligenceConfig {
	return IntelligenceConfig{
		Scoring: IntelligenceScoringConfig{
			AnomalyWindowsSeconds: []int{60, 300, 900},
			ReasonWeights: map[string]int{
				"retry_threshold_exceeded": 30,
				"timeout_failure":          10,
				"timeout_stall":            5,
				"malformed_frame":          5,
				"malformed_publish":        5,
				"decode_failure":           5,
				"topic_mismatch":           3,
				"handler_rejection":        6,
				"rejected_send":            6,
				"rejected_publish":         6,
				"stream_failure":           8,
				"subscribe_failure":        8,
			},
			DeadLetterPenalty:          25,
			HistoricalDeadLetterWeight: 5,
			ResidualPenaltyCap:         15,
			ActiveEpisodePenalty:       5,
			ActiveEpisodePenaltyCap:    25,
			ObservationDropBasePenalty: 6,
			ObservationDropStepPenalty: 3,
			ObservationDropPenaltyCap:  30,
			RuntimeRetryingPenalty:     12,
			RuntimeFailedPenalty:       20,
			HeartbeatPenalty120:        10,
			HeartbeatPenalty300:        20,
			HeartbeatPenalty900:        30,
		},
		Alerts: IntelligenceAlertsConfig{
			MinimumStateDurationSeconds: 30,
			CooldownSeconds:             90,
			RecoveryScoreHealthy:        93,
			RecoveryScoreDegraded:       78,
			RecoveryScoreUnstable:       45,
		},
		Retention: IntelligenceRetentionConfig{
			HealthSnapshotDays:    14,
			HealthSnapshotMaxRows: 50000,
			PruneEverySeconds:     21600,
		},
		Queries: IntelligenceQueryConfig{
			DefaultLimit: 50,
			MaxLimit:     500,
		},
	}
}

func normalizeIntelligence(cfg *Config) {
	defaults := defaultIntelligenceConfig()
	if len(cfg.Intelligence.Scoring.AnomalyWindowsSeconds) == 0 {
		cfg.Intelligence.Scoring.AnomalyWindowsSeconds = append([]int(nil), defaults.Scoring.AnomalyWindowsSeconds...)
	}
	if len(cfg.Intelligence.Scoring.ReasonWeights) == 0 {
		cfg.Intelligence.Scoring.ReasonWeights = map[string]int{}
		for reason, weight := range defaults.Scoring.ReasonWeights {
			cfg.Intelligence.Scoring.ReasonWeights[reason] = weight
		}
	}
	if cfg.Intelligence.Scoring.DeadLetterPenalty <= 0 {
		cfg.Intelligence.Scoring.DeadLetterPenalty = defaults.Scoring.DeadLetterPenalty
	}
	if cfg.Intelligence.Scoring.HistoricalDeadLetterWeight <= 0 {
		cfg.Intelligence.Scoring.HistoricalDeadLetterWeight = defaults.Scoring.HistoricalDeadLetterWeight
	}
	if cfg.Intelligence.Scoring.ResidualPenaltyCap <= 0 {
		cfg.Intelligence.Scoring.ResidualPenaltyCap = defaults.Scoring.ResidualPenaltyCap
	}
	if cfg.Intelligence.Scoring.ActiveEpisodePenalty <= 0 {
		cfg.Intelligence.Scoring.ActiveEpisodePenalty = defaults.Scoring.ActiveEpisodePenalty
	}
	if cfg.Intelligence.Scoring.ActiveEpisodePenaltyCap <= 0 {
		cfg.Intelligence.Scoring.ActiveEpisodePenaltyCap = defaults.Scoring.ActiveEpisodePenaltyCap
	}
	if cfg.Intelligence.Scoring.ObservationDropBasePenalty <= 0 {
		cfg.Intelligence.Scoring.ObservationDropBasePenalty = defaults.Scoring.ObservationDropBasePenalty
	}
	if cfg.Intelligence.Scoring.ObservationDropStepPenalty <= 0 {
		cfg.Intelligence.Scoring.ObservationDropStepPenalty = defaults.Scoring.ObservationDropStepPenalty
	}
	if cfg.Intelligence.Scoring.ObservationDropPenaltyCap <= 0 {
		cfg.Intelligence.Scoring.ObservationDropPenaltyCap = defaults.Scoring.ObservationDropPenaltyCap
	}
	if cfg.Intelligence.Scoring.RuntimeRetryingPenalty <= 0 {
		cfg.Intelligence.Scoring.RuntimeRetryingPenalty = defaults.Scoring.RuntimeRetryingPenalty
	}
	if cfg.Intelligence.Scoring.RuntimeFailedPenalty <= 0 {
		cfg.Intelligence.Scoring.RuntimeFailedPenalty = defaults.Scoring.RuntimeFailedPenalty
	}
	if cfg.Intelligence.Scoring.HeartbeatPenalty120 <= 0 {
		cfg.Intelligence.Scoring.HeartbeatPenalty120 = defaults.Scoring.HeartbeatPenalty120
	}
	if cfg.Intelligence.Scoring.HeartbeatPenalty300 <= 0 {
		cfg.Intelligence.Scoring.HeartbeatPenalty300 = defaults.Scoring.HeartbeatPenalty300
	}
	if cfg.Intelligence.Scoring.HeartbeatPenalty900 <= 0 {
		cfg.Intelligence.Scoring.HeartbeatPenalty900 = defaults.Scoring.HeartbeatPenalty900
	}
	if cfg.Intelligence.Alerts.MinimumStateDurationSeconds <= 0 {
		cfg.Intelligence.Alerts.MinimumStateDurationSeconds = defaults.Alerts.MinimumStateDurationSeconds
	}
	if cfg.Intelligence.Alerts.CooldownSeconds <= 0 {
		cfg.Intelligence.Alerts.CooldownSeconds = defaults.Alerts.CooldownSeconds
	}
	if cfg.Intelligence.Alerts.RecoveryScoreHealthy <= 0 {
		cfg.Intelligence.Alerts.RecoveryScoreHealthy = defaults.Alerts.RecoveryScoreHealthy
	}
	if cfg.Intelligence.Alerts.RecoveryScoreDegraded <= 0 {
		cfg.Intelligence.Alerts.RecoveryScoreDegraded = defaults.Alerts.RecoveryScoreDegraded
	}
	if cfg.Intelligence.Alerts.RecoveryScoreUnstable <= 0 {
		cfg.Intelligence.Alerts.RecoveryScoreUnstable = defaults.Alerts.RecoveryScoreUnstable
	}
	if cfg.Intelligence.Retention.HealthSnapshotDays <= 0 {
		cfg.Intelligence.Retention.HealthSnapshotDays = defaults.Retention.HealthSnapshotDays
	}
	if cfg.Intelligence.Retention.HealthSnapshotMaxRows <= 0 {
		cfg.Intelligence.Retention.HealthSnapshotMaxRows = defaults.Retention.HealthSnapshotMaxRows
	}
	if cfg.Intelligence.Retention.PruneEverySeconds <= 0 {
		cfg.Intelligence.Retention.PruneEverySeconds = defaults.Retention.PruneEverySeconds
	}
	if cfg.Intelligence.Queries.DefaultLimit <= 0 {
		cfg.Intelligence.Queries.DefaultLimit = defaults.Queries.DefaultLimit
	}
	if cfg.Intelligence.Queries.MaxLimit <= 0 {
		cfg.Intelligence.Queries.MaxLimit = defaults.Queries.MaxLimit
	}
}

func validateIntelligence(cfg Config) []string {
	var errs []string
	windows := cfg.Intelligence.Scoring.AnomalyWindowsSeconds
	if len(windows) < 3 {
		errs = append(errs, "intelligence.scoring.anomaly_windows_seconds must contain at least 3 ascending windows")
	}
	prev := 0
	for _, window := range windows {
		if window <= 0 {
			errs = append(errs, "intelligence.scoring.anomaly_windows_seconds entries must be positive")
			break
		}
		if prev > 0 && window <= prev {
			errs = append(errs, "intelligence.scoring.anomaly_windows_seconds must be strictly ascending")
			break
		}
		prev = window
	}
	for reason, weight := range cfg.Intelligence.Scoring.ReasonWeights {
		if weight < 0 {
			errs = append(errs, fmt.Sprintf("intelligence.scoring.reason_weights.%s must be zero or positive", reason))
		}
	}
	if cfg.Intelligence.Alerts.RecoveryScoreHealthy < 90 || cfg.Intelligence.Alerts.RecoveryScoreHealthy > 100 {
		errs = append(errs, "intelligence.alerts.recovery_score_healthy must be between 90 and 100")
	}
	if cfg.Intelligence.Alerts.RecoveryScoreDegraded < 70 || cfg.Intelligence.Alerts.RecoveryScoreDegraded >= cfg.Intelligence.Alerts.RecoveryScoreHealthy {
		errs = append(errs, "intelligence.alerts.recovery_score_degraded must be at least 70 and less than recovery_score_healthy")
	}
	if cfg.Intelligence.Alerts.RecoveryScoreUnstable < 40 || cfg.Intelligence.Alerts.RecoveryScoreUnstable >= cfg.Intelligence.Alerts.RecoveryScoreDegraded {
		errs = append(errs, "intelligence.alerts.recovery_score_unstable must be at least 40 and less than recovery_score_degraded")
	}
	if cfg.Intelligence.Retention.HealthSnapshotDays < 7 || cfg.Intelligence.Retention.HealthSnapshotDays > 30 {
		errs = append(errs, "intelligence.retention.health_snapshot_days must be between 7 and 30")
	}
	if cfg.Intelligence.Retention.HealthSnapshotMaxRows < 1000 {
		errs = append(errs, "intelligence.retention.health_snapshot_max_rows must be at least 1000")
	}
	if cfg.Intelligence.Queries.DefaultLimit <= 0 || cfg.Intelligence.Queries.DefaultLimit > cfg.Intelligence.Queries.MaxLimit {
		errs = append(errs, "intelligence.queries.default_limit must be positive and not exceed intelligence.queries.max_limit")
	}
	if cfg.Intelligence.Alerts.MinimumStateDurationSeconds < 5 {
		errs = append(errs, "intelligence.alerts.minimum_state_duration_seconds must be at least 5")
	}
	if cfg.Intelligence.Alerts.CooldownSeconds < cfg.Intelligence.Alerts.MinimumStateDurationSeconds {
		errs = append(errs, "intelligence.alerts.cooldown_seconds must be at least minimum_state_duration_seconds")
	}
	return errs
}
