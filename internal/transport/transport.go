package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
)

const (
	StateDisabled               = "disabled"
	StateConfiguredNotAttempted = "configured_not_attempted"
	StateAttempting             = "attempting"
	StateConfiguredOffline      = "configured_offline"
	StateConnectedNoIngest      = "connected_no_ingest"
	StateIngesting              = "ingesting"
	StateHistoricalOnly         = "historical_only"
	StateError                  = "error"
)

const StateConfigured = StateConfiguredNotAttempted
const StateConnectedNoData = StateConnectedNoIngest

type CapabilityMatrix struct {
	IngestSupported        bool   `json:"ingest_supported"`
	SendSupported          bool   `json:"send_supported"`
	MetadataFetchSupported bool   `json:"metadata_fetch_supported"`
	NodeFetchSupported     bool   `json:"node_fetch_supported"`
	HealthSupported        bool   `json:"health_supported"`
	ConfigApplySupported   bool   `json:"config_apply_supported"`
	ImplementationStatus   string `json:"implementation_status"`
	Notes                  string `json:"notes,omitempty"`
}

type Health struct {
	Name                string           `json:"name"`
	Type                string           `json:"type"`
	Source              string           `json:"source"`
	State               string           `json:"state"`
	OK                  bool             `json:"ok"`
	Unsupported         bool             `json:"unsupported,omitempty"`
	Detail              string           `json:"detail"`
	Capabilities        CapabilityMatrix `json:"capabilities"`
	LastAttemptAt       string           `json:"last_attempt_at,omitempty"`
	LastError           string           `json:"last_error,omitempty"`
	LastConnectedAt     string           `json:"last_connected_at,omitempty"`
	LastSuccessAt       string           `json:"last_success_at,omitempty"`
	LastDisconnected    string           `json:"last_disconnected_at,omitempty"`
	LastIngestAt        string           `json:"last_ingest_at,omitempty"`
	LastHeartbeatAt     string           `json:"last_heartbeat_at,omitempty"`
	PacketsDropped      uint64           `json:"packets_dropped"`
	ReconnectAttempts   uint64           `json:"reconnect_attempts"`
	TotalMessages       uint64           `json:"total_messages"`
	ErrorCount          uint64           `json:"error_count"`
	ConsecutiveTimeouts uint64           `json:"consecutive_timeouts"`
}

type PacketHandler func(topic string, payload []byte) error

type Transport interface {
	Connect(context.Context) error
	Close(context.Context) error
	Health() Health
	Capabilities() CapabilityMatrix
	SourceType() string
	Name() string
	Subscribe(context.Context, PacketHandler) error
	SendPacket(context.Context, []byte) error
	FetchMetadata(context.Context) (map[string]any, error)
	FetchNodes(context.Context) ([]map[string]any, error)
	MarkIngest(time.Time)
	MarkDrop(string)
}

func Build(cfg config.TransportConfig, log *logging.Logger, bus *events.Bus) (Transport, error) {
	safeLog := log
	if safeLog == nil {
		safeLog = logging.New("info", false)
	}
	safeBus := bus
	if safeBus == nil {
		safeBus = events.New()
	}
	switch cfg.Type {
	case "mqtt":
		return NewMQTT(cfg, safeLog, safeBus), nil
	case "serial", "tcp", "serialtcp":
		return NewDirect(cfg, safeLog, safeBus), nil
	case "http", "ble":
		return NewUnsupported(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported transport type %s", cfg.Type)
	}
}

type Unsupported struct {
	cfg    config.TransportConfig
	mu     sync.Mutex
	health Health
}

func NewUnsupported(cfg config.TransportConfig) *Unsupported {
	caps := capabilityDefaults(cfg, false, false, false, false, true, false, "unsupported", "feature-gated; not enabled in this release")
	state := StateDisabled
	detail := "disabled by config"
	if cfg.Enabled {
		state = StateError
		detail = caps.Notes
	}
	return &Unsupported{cfg: cfg, health: Health{Name: cfg.Name, Type: cfg.Type, Source: cfg.SourceLabel(), State: state, Unsupported: true, Detail: detail, Capabilities: caps}}
}

func (u *Unsupported) Connect(context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.health.OK = false
	u.health.State = StateError
	u.health.ErrorCount++
	u.health.LastAttemptAt = time.Now().UTC().Format(time.RFC3339)
	u.health.LastError = "transport compiled as unsupported in this release"
	return errors.New(u.health.LastError)
}

func (u *Unsupported) Close(context.Context) error { return nil }

func (u *Unsupported) Health() Health {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.health
}

func (u *Unsupported) Capabilities() CapabilityMatrix                 { return u.Health().Capabilities }
func (u *Unsupported) SourceType() string                             { return u.cfg.Type }
func (u *Unsupported) Name() string                                   { return u.cfg.Name }
func (u *Unsupported) Subscribe(context.Context, PacketHandler) error { return nil }
func (u *Unsupported) SendPacket(context.Context, []byte) error {
	return errors.New("send not supported")
}
func (u *Unsupported) FetchMetadata(context.Context) (map[string]any, error) {
	return nil, errors.New("metadata fetch not supported")
}
func (u *Unsupported) FetchNodes(context.Context) ([]map[string]any, error) {
	return nil, errors.New("node fetch not supported")
}
func (u *Unsupported) MarkIngest(time.Time) {}
func (u *Unsupported) MarkDrop(string)      {}

func dialWithTimeout(endpoint string) (net.Conn, error) {
	return net.DialTimeout("tcp", endpoint, 5*time.Second)
}

func capabilityDefaults(_ config.TransportConfig, ingest, send, metadata, nodes, health, configApply bool, status, notes string) CapabilityMatrix {
	return CapabilityMatrix{
		IngestSupported:        ingest,
		SendSupported:          send,
		MetadataFetchSupported: metadata,
		NodeFetchSupported:     nodes,
		HealthSupported:        health,
		ConfigApplySupported:   configApply,
		ImplementationStatus:   status,
		Notes:                  notes,
	}
}
