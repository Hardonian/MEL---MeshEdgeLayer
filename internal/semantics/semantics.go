package semantics

import "strings"

// Package semantics holds canonical operator-facing vocabulary shared by CLI, API, and logs.

// Severity levels for alerts, diagnostics, and audit-style messages.
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
	SeverityInfo     = "info"
)

// Transport / mesh health coarse states (subset; see status package for full runtime truth).
const (
	HealthOK             = "ok"
	HealthDegraded       = "degraded"
	HealthCritical       = "critical"
	HealthUnknown        = "unknown"
	HealthIdle           = "idle"
	HealthHistoricalOnly = "historical_only"
)

// Control plane operating modes (config.control.mode).
const (
	ControlDisabled    = "disabled"
	ControlAdvisory    = "advisory"
	ControlGuardedAuto = "guarded_auto"
)

// Well-known audit / log categories (align with logging.Category* where applicable).
const (
	CategoryTransport   = "transport"
	CategorySecurity    = "security"
	CategoryAudit       = "audit"
	CategoryControl     = "control"
	CategoryConfig      = "config"
	CategoryIntegration = "integration"
)

// Integration delivery channels (outbound).
const (
	ChannelWebhook  = "webhook"
	ChannelSlack    = "slack"
	ChannelTelegram = "telegram"
)

// FormatSeverityForTTY returns a short label for terminal output; adds ANSI color when enabled.
func FormatSeverityForTTY(severity string, color bool) string {
	sev := strings.ToLower(strings.TrimSpace(severity))
	if !color {
		return strings.ToUpper(sev)
	}
	switch sev {
	case SeverityCritical:
		return "\x1b[31mCRITICAL\x1b[0m"
	case SeverityHigh:
		return "\x1b[91mHIGH\x1b[0m"
	case SeverityMedium:
		return "\x1b[33mMEDIUM\x1b[0m"
	case SeverityLow:
		return "\x1b[36mLOW\x1b[0m"
	case SeverityInfo:
		return "\x1b[32mINFO\x1b[0m"
	default:
		return strings.ToUpper(sev)
	}
}
