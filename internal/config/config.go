package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Bind         BindConfig         `json:"bind"`
	Auth         AuthConfig         `json:"auth"`
	Storage      StorageConfig      `json:"storage"`
	Logging      LoggingConfig      `json:"logging"`
	Retention    RetentionConfig    `json:"retention"`
	Privacy      PrivacyConfig      `json:"privacy"`
	Transports   []TransportConfig  `json:"transports"`
	Features     FeatureConfig      `json:"features"`
	RateLimits   RateLimitConfig    `json:"rate_limits"`
	Intelligence IntelligenceConfig `json:"intelligence"`
	Control      ControlConfig      `json:"control"`
	StrictMode   bool               `json:"strict_mode"`
}

type BindConfig struct {
	API         string `json:"api"`
	Metrics     string `json:"metrics"`
	AllowRemote bool   `json:"allow_remote"`
}

type AuthConfig struct {
	Enabled             bool   `json:"enabled"`
	SessionSecret       string `json:"session_secret"`
	UIUser              string `json:"ui_user"`
	UIPassword          string `json:"ui_password"`
	AllowInsecureRemote bool   `json:"allow_insecure_remote"`
}

type StorageConfig struct {
	DataDir            string `json:"data_dir"`
	DatabasePath       string `json:"database_path"`
	EncryptionKeyEnv   string `json:"encryption_key_env"`
	EncryptionRequired bool   `json:"encryption_required"`
}

type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

type RetentionConfig struct {
	Enabled             bool `json:"enabled"`
	MessagesDays        int  `json:"messages_days"`
	TelemetryDays       int  `json:"telemetry_days"`
	AuditDays           int  `json:"audit_days"`
	PrecisePositionDays int  `json:"precise_position_days"`
}

type PrivacyConfig struct {
	StorePrecisePositions  bool     `json:"store_precise_positions"`
	MQTTEncryptionRequired bool     `json:"mqtt_encryption_required"`
	MapReportingAllowed    bool     `json:"map_reporting_allowed"`
	RedactExports          bool     `json:"redact_exports"`
	TrustList              []string `json:"trust_list"`
}

type TransportConfig struct {
	Name                string `json:"name"`
	Type                string `json:"type"`
	Enabled             bool   `json:"enabled"`
	Endpoint            string `json:"endpoint"`
	Topic               string `json:"topic"`
	ClientID            string `json:"client_id"`
	Username            string `json:"username"`
	Password            string `json:"password"`
	MQTTQoS             int    `json:"mqtt_qos"`
	MQTTKeepAliveSec    int    `json:"mqtt_keepalive_seconds"`
	MQTTCleanSession    bool   `json:"mqtt_clean_session"`
	SerialDevice        string `json:"serial_device"`
	SerialBaud          int    `json:"serial_baud"`
	TCPHost             string `json:"tcp_host"`
	TCPPort             int    `json:"tcp_port"`
	ReconnectSeconds    int    `json:"reconnect_seconds"`
	ReadTimeoutSec      int    `json:"read_timeout_seconds"`
	WriteTimeoutSec     int    `json:"write_timeout_seconds"`
	MaxTimeouts         int    `json:"max_consecutive_timeouts"`
	Notes               string `json:"notes"`
	ManualOnly          bool   `json:"manual_only"`
	SuppressAutoActions bool   `json:"suppress_auto_actions"`
	FreezeRouting       bool   `json:"freeze_routing"`
}

func (t TransportConfig) SourceLabel() string {
	switch t.Type {
	case "serial":
		return t.SerialDevice
	case "tcp":
		if t.TCPHost != "" && t.TCPPort > 0 {
			return net.JoinHostPort(t.TCPHost, strconv.Itoa(t.TCPPort))
		}
	}
	return t.Endpoint
}

type FeatureConfig struct {
	WebUI           bool `json:"web_ui"`
	Metrics         bool `json:"metrics"`
	BLEExperimental bool `json:"ble_experimental"`
}

type RateLimitConfig struct {
	HTTPRPS                   int `json:"http_rps"`
	TransportReconnectSeconds int `json:"transport_reconnect_seconds"`
}

type ControlConfig struct {
	Mode                     string   `json:"mode"`
	EmergencyDisable         bool     `json:"emergency_disable"`
	AllowedActions           []string `json:"allowed_actions"`
	MaxActionsPerWindow      int      `json:"max_actions_per_window"`
	CooldownPerTargetSeconds int      `json:"cooldown_per_target_seconds"`
	RequireMinConfidence     float64  `json:"require_min_confidence"`
	AllowMeshLevelActions    bool     `json:"allow_mesh_level_actions"`
	AllowTransportRestart    bool     `json:"allow_transport_restart"`
	AllowSourceSuppression   bool     `json:"allow_source_suppression"`
	ActionWindowSeconds      int      `json:"action_window_seconds"`
	RestartCapPerWindow      int      `json:"restart_cap_per_window"`
	MaxQueue                 int      `json:"max_queue"`
	ActionTimeoutSeconds     int      `json:"action_timeout_seconds"`
	RetentionDays            int      `json:"retention_days"`
}

type Lint struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

func Default() Config {
	return Config{
		Bind:    BindConfig{API: "127.0.0.1:8080", Metrics: "", AllowRemote: false},
		Auth:    AuthConfig{Enabled: false, UIUser: "admin", UIPassword: "change-me", AllowInsecureRemote: false},
		Storage: StorageConfig{DataDir: "./data", DatabasePath: "./data/mel.db", EncryptionKeyEnv: "MEL_STORAGE_KEY", EncryptionRequired: false},
		Logging: LoggingConfig{Level: "info", Format: "json"},
		Retention: RetentionConfig{
			Enabled:             true,
			MessagesDays:        30,
			TelemetryDays:       30,
			AuditDays:           90,
			PrecisePositionDays: 7,
		},
		Privacy: PrivacyConfig{
			StorePrecisePositions:  false,
			MQTTEncryptionRequired: true,
			MapReportingAllowed:    false,
			RedactExports:          true,
			TrustList:              []string{},
		},
		Transports:   []TransportConfig{},
		Features:     FeatureConfig{WebUI: true, Metrics: false, BLEExperimental: false},
		RateLimits:   RateLimitConfig{HTTPRPS: 20, TransportReconnectSeconds: 10},
		Intelligence: defaultIntelligenceConfig(),
		Control: ControlConfig{
			Mode:                     "advisory",
			EmergencyDisable:         false,
			AllowedActions:           []string{"restart_transport", "resubscribe_transport", "backoff_increase", "backoff_reset", "temporarily_deprioritize_transport", "temporarily_suppress_noisy_source", "clear_suppression", "trigger_health_recheck"},
			MaxActionsPerWindow:      8,
			CooldownPerTargetSeconds: 300,
			RequireMinConfidence:     0.75,
			AllowMeshLevelActions:    false,
			AllowTransportRestart:    false,
			AllowSourceSuppression:   false,
			ActionWindowSeconds:      900,
			RestartCapPerWindow:      2,
			MaxQueue:                 32,
			ActionTimeoutSeconds:     10,
			RetentionDays:            14,
		},
		StrictMode: false,
	}
}

func Load(path string) (Config, []byte, error) {
	cfg := Default()
	if path == "" {
		path = "configs/mel.example.json"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, nil, err
	}
	applyEnv(&cfg)
	if err := normalize(&cfg); err != nil {
		return cfg, b, err
	}
	if err := Validate(cfg); err != nil {
		return cfg, b, err
	}
	if cfg.StrictMode {
		if err := ValidateStrict(cfg); err != nil {
			return cfg, b, err
		}
	}
	return cfg, b, nil
}

func normalize(cfg *Config) error {
	defaults := Default()
	if cfg.Storage.DataDir == "" && cfg.Storage.DatabasePath != "" {
		cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	}
	if cfg.Storage.DatabasePath == "" {
		cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	}
	if cfg.Retention.MessagesDays == 0 {
		cfg.Retention.MessagesDays = defaults.Retention.MessagesDays
	}
	if cfg.Retention.TelemetryDays == 0 {
		cfg.Retention.TelemetryDays = defaults.Retention.TelemetryDays
	}
	if cfg.Retention.AuditDays == 0 {
		cfg.Retention.AuditDays = defaults.Retention.AuditDays
	}
	if cfg.Retention.PrecisePositionDays == 0 {
		cfg.Retention.PrecisePositionDays = defaults.Retention.PrecisePositionDays
	}
	for i := range cfg.Transports {
		t := &cfg.Transports[i]
		if t.SerialBaud == 0 {
			t.SerialBaud = 115200
		}
		if t.MQTTKeepAliveSec <= 0 {
			t.MQTTKeepAliveSec = 30
		}
		if t.Type == "mqtt" && t.MQTTQoS == 0 {
			t.MQTTQoS = 1
		}
		if t.MQTTQoS < 0 || t.MQTTQoS > 2 {
			t.MQTTQoS = 1
		}
		if t.ReadTimeoutSec <= 0 {
			t.ReadTimeoutSec = 15
		}
		if t.WriteTimeoutSec <= 0 {
			t.WriteTimeoutSec = 5
		}
		if t.MaxTimeouts <= 0 {
			t.MaxTimeouts = 3
		}
		if t.Type == "serial" && t.Endpoint == "" && t.SerialDevice != "" {
			t.Endpoint = t.SerialDevice
		}
		if t.Type == "tcp" && t.Endpoint == "" && t.TCPHost != "" && t.TCPPort > 0 {
			t.Endpoint = net.JoinHostPort(t.TCPHost, strconv.Itoa(t.TCPPort))
		}
	}
	if cfg.Auth.SessionSecret == "" {
		cfg.Auth.SessionSecret = randomHex(32)
	}
	normalizeIntelligence(cfg)
	normalizeControl(cfg)
	if cfg.Bind.API != "" && !cfg.Bind.AllowRemote {
		host, _, err := net.SplitHostPort(cfg.Bind.API)
		if err == nil && host == "" {
			cfg.Bind.API = "127.0.0.1:" + strings.TrimPrefix(cfg.Bind.API, ":")
		}
	}
	return nil
}

func Validate(cfg Config) error {
	var errs []string
	if cfg.Bind.API == "" {
		errs = appendErr(errs, "bind.api is required")
	}
	if cfg.Storage.DatabasePath == "" {
		errs = appendErr(errs, "storage.database_path is required")
	}
	if cfg.Storage.DataDir == "" {
		errs = appendErr(errs, "storage.data_dir is required")
	}
	if cfg.Logging.Level == "" {
		errs = appendErr(errs, "logging.level is required")
	}
	if cfg.Auth.Enabled && len(cfg.Auth.SessionSecret) < 16 {
		errs = appendErr(errs, "auth.session_secret must be at least 16 chars when auth.enabled")
	}
	if cfg.Bind.AllowRemote && !cfg.Auth.Enabled && !cfg.Auth.AllowInsecureRemote {
		errs = appendErr(errs, "remote bind requires auth.enabled unless auth.allow_insecure_remote=true")
	}
	if cfg.Retention.MessagesDays <= 0 || cfg.Retention.TelemetryDays <= 0 || cfg.Retention.AuditDays <= 0 {
		errs = appendErr(errs, "retention windows must be positive")
	}
	if cfg.Retention.PrecisePositionDays < 0 {
		errs = appendErr(errs, "retention.precise_position_days must be zero or positive")
	}
	if cfg.Storage.EncryptionRequired {
		key := os.Getenv(cfg.Storage.EncryptionKeyEnv)
		if len(key) != 32 {
			errs = appendErr(errs, fmt.Sprintf("storage.encryption_required needs %s to be exactly 32 bytes", cfg.Storage.EncryptionKeyEnv))
		}
	}
	enabledNames := map[string]struct{}{}
	for _, t := range cfg.Transports {
		if !t.Enabled {
			continue
		}
		if t.Name == "" {
			errs = appendErr(errs, "enabled transport missing name")
		}
		if t.Type == "" {
			errs = appendErr(errs, fmt.Sprintf("transport %s missing type", t.Name))
		}
		switch t.Type {
		case "mqtt":
			if t.Endpoint == "" {
				errs = appendErr(errs, fmt.Sprintf("transport %s missing endpoint", t.Name))
			} else if err := validateEndpoint(t.Endpoint); err != nil {
				errs = appendErr(errs, fmt.Sprintf("transport %s invalid endpoint: %v", t.Name, err))
			}
			if t.ClientID == "" {
				errs = appendErr(errs, fmt.Sprintf("transport %s missing client_id", t.Name))
			}
			if t.Topic == "" {
				errs = appendErr(errs, fmt.Sprintf("transport %s missing topic", t.Name))
			}
			if t.MQTTQoS < 0 || t.MQTTQoS > 2 {
				errs = appendErr(errs, fmt.Sprintf("transport %s mqtt_qos must be 0, 1, or 2", t.Name))
			}
			if t.MQTTKeepAliveSec <= 0 {
				errs = appendErr(errs, fmt.Sprintf("transport %s mqtt_keepalive_seconds must be positive", t.Name))
			}
		case "serial":
			if t.SerialDevice == "" && t.Endpoint == "" {
				errs = appendErr(errs, fmt.Sprintf("transport %s missing serial_device", t.Name))
			}
			if t.SerialBaud <= 0 {
				errs = appendErr(errs, fmt.Sprintf("transport %s serial_baud must be positive", t.Name))
			}
		case "tcp":
			if t.Endpoint == "" && (t.TCPHost == "" || t.TCPPort <= 0) {
				errs = appendErr(errs, fmt.Sprintf("transport %s missing tcp_host/tcp_port or endpoint", t.Name))
			} else if t.Endpoint != "" {
				if err := validateEndpoint(t.Endpoint); err != nil {
					errs = appendErr(errs, fmt.Sprintf("transport %s invalid endpoint: %v", t.Name, err))
				}
			} else if t.TCPPort <= 0 || t.TCPPort > 65535 {
				errs = appendErr(errs, fmt.Sprintf("transport %s tcp_port must be between 1 and 65535", t.Name))
			}
		case "serialtcp":
			if t.Endpoint == "" {
				errs = appendErr(errs, fmt.Sprintf("transport %s missing endpoint", t.Name))
			} else if err := validateEndpoint(t.Endpoint); err != nil {
				errs = appendErr(errs, fmt.Sprintf("transport %s invalid endpoint: %v", t.Name, err))
			}
		default:
			if t.Endpoint == "" && t.Type != "ble" && t.Type != "http" {
				errs = appendErr(errs, fmt.Sprintf("transport %s missing endpoint", t.Name))
			}
		}
		if t.ReadTimeoutSec <= 0 {
			errs = appendErr(errs, fmt.Sprintf("transport %s read_timeout_seconds must be positive", t.Name))
		}
		if t.WriteTimeoutSec <= 0 {
			errs = appendErr(errs, fmt.Sprintf("transport %s write_timeout_seconds must be positive", t.Name))
		}
		if t.MaxTimeouts <= 0 {
			errs = appendErr(errs, fmt.Sprintf("transport %s max_consecutive_timeouts must be positive", t.Name))
		}
		if _, ok := enabledNames[t.Name]; ok {
			errs = appendErr(errs, fmt.Sprintf("duplicate enabled transport name %s", t.Name))
		}
		enabledNames[t.Name] = struct{}{}
	}
	errs = append(errs, validateIntelligence(cfg)...)
	errs = append(errs, validateControl(cfg)...)
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func appendErr(errs []string, msg string) []string { return append(errs, msg) }

func LintConfig(cfg Config) []Lint {
	out := make([]Lint, 0)
	if cfg.Bind.AllowRemote {
		out = append(out, Lint{"remote-bind", "high", "API/UI listens beyond localhost.", "Keep MEL bound to localhost unless remote access is deliberate and defended."})
	}
	if cfg.Storage.EncryptionRequired {
		out = append(out, Lint{"storage-encryption", "high", "storage.encryption_required does not encrypt SQLite at rest in RC1.", "Treat this flag as a validation guard only and use filesystem or volume encryption if you need encrypted storage today."})
	}
	if cfg.Bind.Metrics != "" || cfg.Features.Metrics {
		out = append(out, Lint{"metrics-bind-unused", "medium", "Metrics are served on the main API listener; bind.metrics and features.metrics do not start a second listener.", "Scrape /metrics on bind.api or leave the reserved metrics knobs unset."})
	}
	if cfg.Features.BLEExperimental {
		out = append(out, Lint{"ble-experimental", "high", "BLE remains explicitly unsupported in RC1 even if features.ble_experimental is set.", "Use serial, TCP, or MQTT instead and treat BLE as planned work."})
	}
	if cfg.Bind.AllowRemote && !cfg.Auth.Enabled {
		out = append(out, Lint{"remote-bind-auth", "critical", "Remote bind is enabled without authentication.", "Enable auth or turn off remote bind."})
	}
	if cfg.Bind.AllowRemote && cfg.Auth.AllowInsecureRemote {
		out = append(out, Lint{"unsafe-dev-remote-override", "high", "Unsafe development override is allowing remote bind without auth.", "Use only for short-lived local development and never for go-live deployments."})
	}
	if !cfg.Privacy.MQTTEncryptionRequired {
		out = append(out, Lint{"mqtt-encryption", "high", "MQTT encryption requirement is disabled.", "Require encrypted broker transport or disable MQTT."})
	}
	if cfg.Privacy.MapReportingAllowed {
		out = append(out, Lint{"map-reporting", "high", "Map reporting can expose node metadata and location.", "Disable map reporting unless operators have explicitly accepted the risk."})
	}
	if cfg.Retention.MessagesDays > 90 {
		out = append(out, Lint{"long-message-retention", "medium", "Message retention exceeds 90 days.", "Shorten message retention or document the operational need."})
	}
	if cfg.Privacy.StorePrecisePositions {
		out = append(out, Lint{"precise-position-storage", "high", "Precise position storage is enabled.", "Prefer redacted positions unless precise storage is required for the deployment."})
	}
	directEnabled := 0
	for _, t := range cfg.Transports {
		if !t.Enabled {
			continue
		}
		if t.Type == "serial" || t.Type == "tcp" || t.Type == "serialtcp" {
			directEnabled++
		}
	}
	if directEnabled > 1 {
		out = append(out, Lint{"multi-direct-transport", "high", "More than one direct-node transport is enabled.", "Choose one direct serial/TCP attachment path to avoid device ownership contention."})
	}
	if len(cfg.Transports) > 1 {
		enabled := 0
		for _, t := range cfg.Transports {
			if t.Enabled {
				enabled++
			}
		}
		if enabled > 1 {
			out = append(out, Lint{"multi-transport-contention", "medium", "Multiple transports are enabled at once.", "Use one primary radio path unless you have verified shared ownership behavior."})
		}
	}
	for _, t := range cfg.Transports {
		if t.Enabled && t.Type == "mqtt" && (strings.Contains(strings.ToLower(t.Topic), "default") || strings.Contains(strings.ToLower(t.Topic), "public")) {
			out = append(out, Lint{"mqtt-default-channel", "medium", "MQTT topic naming suggests widely-known or default channel usage.", "Confirm the channel is intentional and avoid public/default identifiers where possible."})
		}
		if !t.Enabled {
			continue
		}
		switch t.Type {
		case "ble", "http":
			out = append(out, Lint{"unsupported-transport-" + t.Type, "high", fmt.Sprintf("Transport %s uses %s, which is explicitly unsupported in RC1.", t.Name, t.Type), "Disable this transport and switch to serial, tcp, or mqtt before expecting ingest."})
		case "serialtcp":
			out = append(out, Lint{"experimental-serialtcp", "high", fmt.Sprintf("Transport %s uses serialtcp, which is present in code but not hardened as a primary operator path.", t.Name), "Prefer the primary tcp transport type unless you are deliberately testing the alias path."})
		}
	}
	for _, t := range cfg.Transports {
		if !t.Enabled {
			continue
		}
		switch t.Type {
		case "serial":
			device := t.SerialDevice
			if device == "" {
				device = t.Endpoint
			}
			if strings.TrimSpace(device) == "" {
				continue
			}
			if !filepath.IsAbs(device) {
				out = append(out, Lint{"serial-relative-path", "medium", fmt.Sprintf("transport %s uses a non-absolute serial path.", t.Name), "Prefer an absolute device path such as /dev/serial/by-id/... so MEL does not depend on the working directory."})
			}
			if !strings.Contains(device, "/dev/serial/by-id/") {
				out = append(out, Lint{"serial-stable-path", "medium", fmt.Sprintf("transport %s is not using a stable /dev/serial/by-id path.", t.Name), "Use /dev/serial/by-id/... when available to survive USB re-enumeration."})
			}
		case "tcp", "serialtcp":
			endpoint := t.Endpoint
			if endpoint == "" {
				endpoint = net.JoinHostPort(t.TCPHost, strconv.Itoa(t.TCPPort))
			}
			if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
				out = append(out, Lint{"tcp-http-endpoint", "high", fmt.Sprintf("transport %s endpoint looks like HTTP, but direct TCP expects raw Meshtastic framing.", t.Name), "Point MEL at the raw Meshtastic TCP stream instead of a web UI or HTTP API."})
			}
		}
	}
	return out
}

func validateEndpoint(endpoint string) error {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return err
	}
	if strings.TrimSpace(host) == "" {
		return errors.New("host is empty")
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port is not numeric: %w", err)
	}
	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("port %d is out of range", portNum)
	}
	return nil
}

func WriteInit(path string) (Config, error) {
	cfg := Default()
	cfg.Auth.SessionSecret = randomHex(32)
	cfg.Storage.DataDir = "./data"
	cfg.Storage.DatabasePath = "./data/mel.db"
	if err := normalize(&cfg); err != nil {
		return cfg, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return cfg, err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return cfg, err
	}
	b = append(b, '\n')
	return cfg, os.WriteFile(path, b, 0o600)
}

func applyEnv(cfg *Config) {
	setString := func(env string, dst *string) {
		if v := os.Getenv(env); v != "" {
			*dst = v
		}
	}
	setBool := func(env string, dst *bool) {
		if v := os.Getenv(env); v != "" {
			*dst = strings.EqualFold(v, "true") || v == "1"
		}
	}
	setInt := func(env string, dst *int) {
		if v := os.Getenv(env); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
	}
	setString("MEL_BIND_API", &cfg.Bind.API)
	setString("MEL_BIND_METRICS", &cfg.Bind.Metrics)
	setBool("MEL_BIND_ALLOW_REMOTE", &cfg.Bind.AllowRemote)
	setString("MEL_DB_PATH", &cfg.Storage.DatabasePath)
	setString("MEL_DATA_DIR", &cfg.Storage.DataDir)
	setBool("MEL_AUTH_ENABLED", &cfg.Auth.Enabled)
	setString("MEL_SESSION_SECRET", &cfg.Auth.SessionSecret)
	setString("MEL_UI_USER", &cfg.Auth.UIUser)
	setString("MEL_UI_PASSWORD", &cfg.Auth.UIPassword)
	setBool("MEL_AUTH_ALLOW_INSECURE_REMOTE", &cfg.Auth.AllowInsecureRemote)
	setBool("MEL_PRIVACY_STORE_PRECISE_POSITIONS", &cfg.Privacy.StorePrecisePositions)
	setBool("MEL_PRIVACY_MAP_REPORTING_ALLOWED", &cfg.Privacy.MapReportingAllowed)
	setBool("MEL_PRIVACY_MQTT_ENCRYPTION_REQUIRED", &cfg.Privacy.MQTTEncryptionRequired)
	setBool("MEL_PRIVACY_REDACT_EXPORTS", &cfg.Privacy.RedactExports)
	setBool("MEL_RETENTION_ENABLED", &cfg.Retention.Enabled)
	setInt("MEL_RETENTION_MESSAGES_DAYS", &cfg.Retention.MessagesDays)
	setInt("MEL_RETENTION_TELEMETRY_DAYS", &cfg.Retention.TelemetryDays)
	setInt("MEL_RETENTION_AUDIT_DAYS", &cfg.Retention.AuditDays)
	setBool("MEL_CONTROL_EMERGENCY_DISABLE", &cfg.Control.EmergencyDisable)
	setBool("MEL_CONTROL_ALLOW_TRANSPORT_RESTART", &cfg.Control.AllowTransportRestart)
	setBool("MEL_CONTROL_ALLOW_SOURCE_SUPPRESSION", &cfg.Control.AllowSourceSuppression)
	setInt("MEL_CONTROL_MAX_QUEUE", &cfg.Control.MaxQueue)
	setBool("MEL_STRICT_MODE", &cfg.StrictMode)
}

type SafetyViolation struct {
	Field   string `json:"field"`
	Issue   string `json:"issue"`
	Current string `json:"current"`
	Safe    string `json:"safe"`
}

func ValidateSafeDefaults(cfg Config) []SafetyViolation {
	var violations []SafetyViolation

	if cfg.Control.Mode != "advisory" && cfg.Control.Mode != "disabled" {
		violations = append(violations, SafetyViolation{
			Field:   "control.mode",
			Issue:   "non-advisory control mode enabled",
			Current: cfg.Control.Mode,
			Safe:    "advisory or disabled",
		})
	}

	if cfg.Control.EmergencyDisable {
		violations = append(violations, SafetyViolation{
			Field:   "control.emergency_disable",
			Issue:   "emergency disable is active",
			Current: "true",
			Safe:    "false",
		})
	}

	if cfg.Control.AllowTransportRestart {
		violations = append(violations, SafetyViolation{
			Field:   "control.allow_transport_restart",
			Issue:   "automatic transport restart enabled",
			Current: "true",
			Safe:    "false",
		})
	}

	if cfg.Control.AllowSourceSuppression {
		violations = append(violations, SafetyViolation{
			Field:   "control.allow_source_suppression",
			Issue:   "automatic source suppression enabled",
			Current: "true",
			Safe:    "false",
		})
	}

	if cfg.Bind.AllowRemote {
		if !cfg.Auth.Enabled {
			violations = append(violations, SafetyViolation{
				Field:   "bind.allow_remote",
				Issue:   "remote bind without authentication",
				Current: "true (auth disabled)",
				Safe:    "false or enable auth",
			})
		}
		if cfg.Auth.AllowInsecureRemote {
			violations = append(violations, SafetyViolation{
				Field:   "auth.allow_insecure_remote",
				Issue:   "insecure remote override enabled",
				Current: "true",
				Safe:    "false",
			})
		}
	}

	if cfg.Retention.MessagesDays > 90 {
		violations = append(violations, SafetyViolation{
			Field:   "retention.messages_days",
			Issue:   "excessive message retention",
			Current: strconv.Itoa(cfg.Retention.MessagesDays),
			Safe:    "<= 90",
		})
	}

	if cfg.Retention.TelemetryDays > 90 {
		violations = append(violations, SafetyViolation{
			Field:   "retention.telemetry_days",
			Issue:   "excessive telemetry retention",
			Current: strconv.Itoa(cfg.Retention.TelemetryDays),
			Safe:    "<= 90",
		})
	}

	if !cfg.Retention.Enabled {
		violations = append(violations, SafetyViolation{
			Field:   "retention.enabled",
			Issue:   "retention job disabled",
			Current: "false",
			Safe:    "true",
		})
	}

	if cfg.Privacy.StorePrecisePositions {
		violations = append(violations, SafetyViolation{
			Field:   "privacy.store_precise_positions",
			Issue:   "precise position storage enabled",
			Current: "true",
			Safe:    "false",
		})
	}

	if !cfg.Privacy.MQTTEncryptionRequired {
		violations = append(violations, SafetyViolation{
			Field:   "privacy.mqtt_encryption_required",
			Issue:   "MQTT encryption not required",
			Current: "false",
			Safe:    "true",
		})
	}

	if cfg.Privacy.MapReportingAllowed {
		violations = append(violations, SafetyViolation{
			Field:   "privacy.map_reporting_allowed",
			Issue:   "map reporting enabled",
			Current: "true",
			Safe:    "false",
		})
	}

	if !cfg.Privacy.RedactExports {
		violations = append(violations, SafetyViolation{
			Field:   "privacy.redact_exports",
			Issue:   "export redaction disabled",
			Current: "false",
			Safe:    "true",
		})
	}

	if cfg.Control.MaxQueue > 64 {
		violations = append(violations, SafetyViolation{
			Field:   "control.max_queue",
			Issue:   "excessive action queue size",
			Current: strconv.Itoa(cfg.Control.MaxQueue),
			Safe:    "<= 64",
		})
	}

	if cfg.RateLimits.HTTPRPS > 100 {
		violations = append(violations, SafetyViolation{
			Field:   "rate_limits.http_rps",
			Issue:   "excessive HTTP rate limit",
			Current: strconv.Itoa(cfg.RateLimits.HTTPRPS),
			Safe:    "<= 100",
		})
	}

	if cfg.RateLimits.TransportReconnectSeconds < 5 {
		violations = append(violations, SafetyViolation{
			Field:   "rate_limits.transport_reconnect_seconds",
			Issue:   "reconnect interval too short",
			Current: strconv.Itoa(cfg.RateLimits.TransportReconnectSeconds),
			Safe:    ">= 5",
		})
	}

	return violations
}

func ValidateStrict(cfg Config) error {
	violations := ValidateSafeDefaults(cfg)
	if len(violations) > 0 {
		var msgs []string
		for _, v := range violations {
			msgs = append(msgs, fmt.Sprintf("%s: %s (current=%s, safe=%s)", v.Field, v.Issue, v.Current, v.Safe))
		}
		return errors.New("strict mode validation failed: " + strings.Join(msgs, "; "))
	}
	return nil
}

type SecurityPosture struct {
	Mode               string   `json:"mode"`
	ControlMode        string   `json:"control_mode"`
	AuthEnabled        bool     `json:"auth_enabled"`
	RemoteBind         bool     `json:"remote_bind"`
	EncryptionRequired bool     `json:"encryption_required"`
	PrecisePositions   bool     `json:"precise_positions"`
	MapReporting       bool     `json:"map_reporting"`
	Violations         []string `json:"violations,omitempty"`
	Safe               bool     `json:"safe"`
}

func SecurityBanner(cfg Config) SecurityPosture {
	violations := ValidateSafeDefaults(cfg)
	var violationMsgs []string
	for _, v := range violations {
		violationMsgs = append(violationMsgs, v.Issue)
	}

	return SecurityPosture{
		Mode:               "production",
		ControlMode:        cfg.Control.Mode,
		AuthEnabled:        cfg.Auth.Enabled,
		RemoteBind:         cfg.Bind.AllowRemote,
		EncryptionRequired: cfg.Storage.EncryptionRequired,
		PrecisePositions:   cfg.Privacy.StorePrecisePositions,
		MapReporting:       cfg.Privacy.MapReportingAllowed,
		Violations:         violationMsgs,
		Safe:               len(violations) == 0,
	}
}

func SHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("0", size*2)
	}
	return hex.EncodeToString(buf)
}
