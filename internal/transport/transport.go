package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
)

const (
	StateDisabled       = "disabled"
	StateConfigured     = "configured"
	StateConnecting     = "connecting"
	StateLive           = "live"
	StateIdle           = "idle"
	StateRetrying       = "retrying"
	StateFailed         = "failed"
	StateHistoricalOnly = "historical_only"
)

const (
	StateConfiguredNotAttempted = StateConfigured
	StateAttempting             = StateConnecting
	StateConfiguredOffline      = StateRetrying
	StateConnectedNoIngest      = StateIdle
	StateIngesting              = StateLive
	StateError                  = StateFailed
)

const StateConnectedNoData = StateConnectedNoIngest

const (
	ReasonMalformedFrame         = "malformed_frame"
	ReasonDecodeFailure          = "decode_failure"
	ReasonRejectedSend           = "rejected_send"
	ReasonUnsupportedControlPath = "unsupported_control_path"
	ReasonTimeoutStall           = "timeout_stall"
	ReasonTimeoutFailure         = "timeout_failure"
	ReasonMalformedPublish       = "malformed_publish"
	ReasonTopicMismatch          = "topic_mismatch"
	ReasonHandlerRejection       = "handler_rejection"
	ReasonRejectedPublish        = "rejected_publish"
	ReasonStreamFailure          = "stream_failure"
	ReasonSubscribeFailure       = "subscribe_failure"
	ReasonRetryThresholdExceeded = "retry_threshold_exceeded"
	ReasonObservationDropped     = "observation_dropped"
)

const maxObservationPayloadBytes = 96

var observationSeq uint64

var terminalDeadLetterReasons = map[string]bool{
	ReasonRetryThresholdExceeded: true,
	ReasonRejectedSend:           true,
	ReasonRejectedPublish:        true,
}

var auditOnlyReasons = map[string]bool{
	ReasonMalformedFrame:         true,
	ReasonMalformedPublish:       true,
	ReasonTimeoutStall:           true,
	ReasonTopicMismatch:          true,
	ReasonStreamFailure:          true,
	ReasonSubscribeFailure:       false,
	ReasonTimeoutFailure:         false,
	ReasonObservationDropped:     false,
	ReasonUnsupportedControlPath: false,
}

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
	Name                  string           `json:"name"`
	Type                  string           `json:"type"`
	Source                string           `json:"source"`
	State                 string           `json:"state"`
	OK                    bool             `json:"ok"`
	Unsupported           bool             `json:"unsupported,omitempty"`
	Detail                string           `json:"detail"`
	Capabilities          CapabilityMatrix `json:"capabilities"`
	LastAttemptAt         string           `json:"last_attempt_at,omitempty"`
	LastError             string           `json:"last_error,omitempty"`
	LastConnectedAt       string           `json:"last_connected_at,omitempty"`
	LastSuccessAt         string           `json:"last_success_at,omitempty"`
	LastDisconnected      string           `json:"last_disconnected_at,omitempty"`
	LastIngestAt          string           `json:"last_ingest_at,omitempty"`
	LastHeartbeatAt       string           `json:"last_heartbeat_at,omitempty"`
	LastFailureAt         string           `json:"last_failure_at,omitempty"`
	LastObservationDropAt string           `json:"last_observation_drop_at,omitempty"`
	PacketsDropped        uint64           `json:"packets_dropped"`
	ReconnectAttempts     uint64           `json:"reconnect_attempts"`
	TotalMessages         uint64           `json:"total_messages"`
	ErrorCount            uint64           `json:"error_count"`
	ConsecutiveTimeouts   uint64           `json:"consecutive_timeouts"`
	FailureCount          uint64           `json:"failure_count"`
	ObservationDrops      uint64           `json:"observation_drops"`
	EpisodeID             string           `json:"episode_id,omitempty"`
}

type PacketHandler func(topic string, payload []byte) error

type Observation struct {
	ObservationID string         `json:"observation_id"`
	EpisodeID     string         `json:"episode_id,omitempty"`
	TransportName string         `json:"transport_name"`
	TransportType string         `json:"transport_type"`
	Topic         string         `json:"topic,omitempty"`
	Reason        string         `json:"reason"`
	PayloadHex    string         `json:"payload_hex,omitempty"`
	DeadLetter    bool           `json:"dead_letter"`
	Detail        string         `json:"detail,omitempty"`
	Details       map[string]any `json:"details,omitempty"`
	Timestamp     string         `json:"timestamp"`
}

func NewObservation(name, typ, topic, reason string, payload []byte, final bool, detail string, details map[string]any) Observation {
	detailsCopy := map[string]any{}
	for k, v := range details {
		detailsCopy[k] = v
	}
	if final {
		detailsCopy["final"] = true
	}
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	obs := Observation{
		ObservationID: nextObservationID(),
		TransportName: name,
		TransportType: typ,
		Topic:         topic,
		Reason:        reason,
		PayloadHex:    boundedPayloadHex(payload),
		DeadLetter:    ShouldDeadLetter(reason, detailsCopy),
		Detail:        detail,
		Timestamp:     timestamp,
	}
	if len(detailsCopy) > 0 {
		obs.Details = detailsCopy
		if episodeID := fmt.Sprint(detailsCopy["episode_id"]); episodeID != "" && episodeID != "<nil>" {
			obs.EpisodeID = episodeID
		}
	}
	return obs
}

func (o Observation) Valid() bool {
	if o.TransportName == "" || o.TransportType == "" || o.Reason == "" || o.ObservationID == "" || o.Timestamp == "" {
		return false
	}
	return knownReason(o.Reason)
}

func knownReason(reason string) bool {
	switch reason {
	case ReasonMalformedFrame, ReasonDecodeFailure, ReasonRejectedSend, ReasonUnsupportedControlPath, ReasonTimeoutStall,
		ReasonTimeoutFailure, ReasonMalformedPublish, ReasonTopicMismatch, ReasonHandlerRejection, ReasonRejectedPublish,
		ReasonStreamFailure, ReasonSubscribeFailure, ReasonRetryThresholdExceeded, ReasonObservationDropped:
		return true
	default:
		return false
	}
}

func ShouldDeadLetter(reason string, details map[string]any) bool {
	if terminalDeadLetterReasons[reason] {
		return true
	}
	switch reason {
	case ReasonHandlerRejection:
		return detailBool(details, "final")
	case ReasonDecodeFailure:
		return detailBool(details, "unrecoverable") || detailBool(details, "final")
	case ReasonSubscribeFailure, ReasonTimeoutFailure, ReasonObservationDropped:
		return detailBool(details, "final")
	case ReasonMalformedFrame, ReasonMalformedPublish, ReasonTimeoutStall, ReasonTopicMismatch, ReasonStreamFailure:
		return false
	default:
		return detailBool(details, "final") && !auditOnlyReasons[reason]
	}
}

func detailBool(details map[string]any, key string) bool {
	if len(details) == 0 {
		return false
	}
	v, ok := details[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func nextObservationID() string {
	seq := atomic.AddUint64(&observationSeq, 1)
	return fmt.Sprintf("obs-%d-%d", time.Now().UTC().UnixNano(), seq)
}

func boundedPayloadHex(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	if len(payload) > maxObservationPayloadBytes {
		payload = payload[:maxObservationPayloadBytes]
	}
	return fmt.Sprintf("%x", payload)
}

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

type RuntimeStateController interface {
	ForceState(state, detail, lastError string)
	BeginFailureEpisode(error) (string, uint64)
	CloseFailureEpisode()
	RecordObservationDrop(uint64)
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
