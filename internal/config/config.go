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
	Bind       BindConfig        `json:"bind"`
	Auth       AuthConfig        `json:"auth"`
	Storage    StorageConfig     `json:"storage"`
	Logging    LoggingConfig     `json:"logging"`
	Retention  RetentionConfig   `json:"retention"`
	Privacy    PrivacyConfig     `json:"privacy"`
	Transports []TransportConfig `json:"transports"`
	Features   FeatureConfig     `json:"features"`
	RateLimits RateLimitConfig   `json:"rate_limits"`
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
	MessagesDays        int `json:"messages_days"`
	TelemetryDays       int `json:"telemetry_days"`
	AuditDays           int `json:"audit_days"`
	PrecisePositionDays int `json:"precise_position_days"`
}

type PrivacyConfig struct {
	StorePrecisePositions  bool     `json:"store_precise_positions"`
	MQTTEncryptionRequired bool     `json:"mqtt_encryption_required"`
	MapReportingAllowed    bool     `json:"map_reporting_allowed"`
	RedactExports          bool     `json:"redact_exports"`
	TrustList              []string `json:"trust_list"`
}

type TransportConfig struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	Enabled          bool   `json:"enabled"`
	Endpoint         string `json:"endpoint"`
	Topic            string `json:"topic"`
	ClientID         string `json:"client_id"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	SerialDevice     string `json:"serial_device"`
	SerialBaud       int    `json:"serial_baud"`
	TCPHost          string `json:"tcp_host"`
	TCPPort          int    `json:"tcp_port"`
	ReconnectSeconds int    `json:"reconnect_seconds"`
	Notes            string `json:"notes"`
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

type Lint struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

func Default() Config {
	return Config{
		Bind:    BindConfig{API: "127.0.0.1:8080", Metrics: ""},
		Auth:    AuthConfig{Enabled: false, UIUser: "admin", UIPassword: "change-me"},
		Storage: StorageConfig{DataDir: "./data", DatabasePath: "./data/mel.db", EncryptionKeyEnv: "MEL_STORAGE_KEY"},
		Logging: LoggingConfig{Level: "info", Format: "json"},
		Retention: RetentionConfig{
			MessagesDays:        30,
			TelemetryDays:       30,
			AuditDays:           90,
			PrecisePositionDays: 7,
		},
		Privacy:    PrivacyConfig{MQTTEncryptionRequired: true, RedactExports: true, TrustList: []string{}},
		Transports: []TransportConfig{},
		Features:   FeatureConfig{WebUI: true, Metrics: false},
		RateLimits: RateLimitConfig{HTTPRPS: 20, TransportReconnectSeconds: 10},
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
	return cfg, b, Validate(cfg)
}

func normalize(cfg *Config) error {
	if cfg.Storage.DataDir == "" && cfg.Storage.DatabasePath != "" {
		cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	}
	if cfg.Storage.DatabasePath == "" {
		cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	}
	for i := range cfg.Transports {
		t := &cfg.Transports[i]
		if t.SerialBaud == 0 {
			t.SerialBaud = 115200
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
			}
		case "serialtcp":
			if t.Endpoint == "" {
				errs = appendErr(errs, fmt.Sprintf("transport %s missing endpoint", t.Name))
			}
		default:
			if t.Endpoint == "" && t.Type != "ble" && t.Type != "http" {
				errs = appendErr(errs, fmt.Sprintf("transport %s missing endpoint", t.Name))
			}
		}
		if _, ok := enabledNames[t.Name]; ok {
			errs = appendErr(errs, fmt.Sprintf("duplicate enabled transport name %s", t.Name))
		}
		enabledNames[t.Name] = struct{}{}
		if t.Type == "mqtt" && t.Topic == "" {
			errs = appendErr(errs, fmt.Sprintf("transport %s missing topic", t.Name))
		}
	}
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
		out = append(out, Lint{"metrics-placeholder", "medium", "Metrics listener settings are present but no metrics server is implemented in RC1.", "Do not rely on bind.metrics or features.metrics for scraping until a real metrics endpoint ships."})
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
		if !t.Enabled || t.Type != "mqtt" {
			continue
		}
		if strings.Contains(strings.ToLower(t.Topic), "default") || strings.Contains(strings.ToLower(t.Topic), "public") {
			out = append(out, Lint{"mqtt-default-channel", "medium", "MQTT topic naming suggests widely-known or default channel usage.", "Confirm the channel is intentional and avoid public/default identifiers where possible."})
		}
	}
	return out
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
	setInt("MEL_RETENTION_MESSAGES_DAYS", &cfg.Retention.MessagesDays)
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
