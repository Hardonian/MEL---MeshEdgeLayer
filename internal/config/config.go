package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	Enabled       bool   `json:"enabled"`
	SessionSecret string `json:"session_secret"`
	UIUser        string `json:"ui_user"`
	UIPassword    string `json:"ui_password"`
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
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Enabled      bool     `json:"enabled"`
	Endpoint     string   `json:"endpoint"`
	Topic        string   `json:"topic"`
	ClientID     string   `json:"client_id"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	Capabilities []string `json:"capabilities"`
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

func Default() Config {
	return Config{
		Bind:       BindConfig{API: "127.0.0.1:8080", Metrics: "127.0.0.1:9090"},
		Storage:    StorageConfig{DataDir: "./data", DatabasePath: "./data/mel.db", EncryptionKeyEnv: "MEL_STORAGE_KEY"},
		Logging:    LoggingConfig{Level: "info", Format: "json"},
		Retention:  RetentionConfig{MessagesDays: 30, TelemetryDays: 30, AuditDays: 90, PrecisePositionDays: 7},
		Privacy:    PrivacyConfig{MQTTEncryptionRequired: true, RedactExports: true},
		Features:   FeatureConfig{WebUI: true, Metrics: true},
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
	return nil
}

func Validate(cfg Config) error {
	var errs []string
	if cfg.Bind.API == "" {
		errs = append(errs, "bind.api is required")
	}
	if cfg.Storage.DatabasePath == "" {
		errs = append(errs, "storage.database_path is required")
	}
	if cfg.Storage.DataDir == "" {
		errs = append(errs, "storage.data_dir is required")
	}
	if cfg.Logging.Level == "" {
		errs = append(errs, "logging.level is required")
	}
	if cfg.Auth.Enabled && len(cfg.Auth.SessionSecret) < 16 {
		errs = append(errs, "auth.session_secret must be at least 16 chars when auth.enabled")
	}
	if cfg.Bind.AllowRemote && !cfg.Auth.Enabled {
		errs = append(errs, "remote bind requires auth.enabled")
	}
	if cfg.Retention.MessagesDays <= 0 || cfg.Retention.TelemetryDays <= 0 {
		errs = append(errs, "retention windows must be positive")
	}
	for _, t := range cfg.Transports {
		if t.Enabled && t.Name == "" {
			errs = append(errs, "enabled transport missing name")
		}
		if t.Enabled && t.Type == "" {
			errs = append(errs, fmt.Sprintf("transport %s missing type", t.Name))
		}
		if t.Enabled && t.Endpoint == "" {
			errs = append(errs, fmt.Sprintf("transport %s missing endpoint", t.Name))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
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
	setBool("MEL_PRIVACY_STORE_PRECISE_POSITIONS", &cfg.Privacy.StorePrecisePositions)
	setBool("MEL_PRIVACY_MAP_REPORTING_ALLOWED", &cfg.Privacy.MapReportingAllowed)
	setInt("MEL_RETENTION_MESSAGES_DAYS", &cfg.Retention.MessagesDays)
}

func SHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
