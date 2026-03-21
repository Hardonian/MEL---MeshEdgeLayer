package config

func normalizeControl(cfg *Config) {
	defaults := Default().Control
	if cfg.Control.Mode == "" {
		cfg.Control.Mode = defaults.Mode
	}
	if len(cfg.Control.AllowedActions) == 0 {
		cfg.Control.AllowedActions = append([]string(nil), defaults.AllowedActions...)
	}
	if cfg.Control.MaxActionsPerWindow <= 0 {
		cfg.Control.MaxActionsPerWindow = defaults.MaxActionsPerWindow
	}
	if cfg.Control.CooldownPerTargetSeconds <= 0 {
		cfg.Control.CooldownPerTargetSeconds = defaults.CooldownPerTargetSeconds
	}
	if cfg.Control.RequireMinConfidence <= 0 {
		cfg.Control.RequireMinConfidence = defaults.RequireMinConfidence
	}
	if cfg.Control.ActionWindowSeconds <= 0 {
		cfg.Control.ActionWindowSeconds = defaults.ActionWindowSeconds
	}
	if cfg.Control.RestartCapPerWindow <= 0 {
		cfg.Control.RestartCapPerWindow = defaults.RestartCapPerWindow
	}
	if cfg.Control.MaxQueue <= 0 {
		cfg.Control.MaxQueue = defaults.MaxQueue
	}
	if cfg.Control.MaxQueue > 128 {
		cfg.Control.MaxQueue = 128
	}
	if cfg.Control.ActionTimeoutSeconds <= 0 {
		cfg.Control.ActionTimeoutSeconds = defaults.ActionTimeoutSeconds
	}
	if cfg.Control.RetentionDays <= 0 {
		cfg.Control.RetentionDays = defaults.RetentionDays
	}
	if cfg.Control.RetentionDays > 30 {
		cfg.Control.RetentionDays = 30
	}
	// Approval timeout: 0 = disabled, positive = seconds, max 24h
	if cfg.Control.ApprovalTimeoutSeconds < 0 {
		cfg.Control.ApprovalTimeoutSeconds = 0
	}
	if cfg.Control.ApprovalTimeoutSeconds > 86400 {
		cfg.Control.ApprovalTimeoutSeconds = 86400
	}
}

func validateControl(cfg Config) []string {
	var errs []string
	switch cfg.Control.Mode {
	case "disabled", "advisory", "guarded_auto":
	default:
		errs = append(errs, "control.mode must be one of disabled, advisory, guarded_auto")
	}
	if cfg.Control.RequireMinConfidence < 0.5 || cfg.Control.RequireMinConfidence > 1 {
		errs = append(errs, "control.require_min_confidence must be between 0.5 and 1")
	}
	if cfg.Control.MaxActionsPerWindow < 1 {
		errs = append(errs, "control.max_actions_per_window must be at least 1")
	}
	if cfg.Control.CooldownPerTargetSeconds < 30 {
		errs = append(errs, "control.cooldown_per_target_seconds must be at least 30")
	}
	if cfg.Control.ActionWindowSeconds < cfg.Control.CooldownPerTargetSeconds {
		errs = append(errs, "control.action_window_seconds must be at least cooldown_per_target_seconds")
	}
	if cfg.Control.RestartCapPerWindow < 1 {
		errs = append(errs, "control.restart_cap_per_window must be at least 1")
	}
	if cfg.Control.MaxQueue < 1 {
		errs = append(errs, "control.max_queue must be at least 1")
	}
	if cfg.Control.MaxQueue > 128 {
		errs = append(errs, "control.max_queue must not exceed 128")
	}
	if cfg.Control.ActionTimeoutSeconds < 1 {
		errs = append(errs, "control.action_timeout_seconds must be at least 1")
	}
	if cfg.Control.RetentionDays < 7 || cfg.Control.RetentionDays > 30 {
		errs = append(errs, "control.retention_days must be between 7 and 30")
	}
	if len(cfg.Control.AllowedActions) == 0 {
		errs = append(errs, "control.allowed_actions must not be empty")
	}
	if cfg.Control.ApprovalTimeoutSeconds < 0 || cfg.Control.ApprovalTimeoutSeconds > 86400 {
		errs = append(errs, "control.approval_timeout_seconds must be between 0 (disabled) and 86400 (24h)")
	}
	return errs
}
