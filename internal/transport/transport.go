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

type Health struct {
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	OK               bool     `json:"ok"`
	Unsupported      bool     `json:"unsupported,omitempty"`
	Detail           string   `json:"detail"`
	Capabilities     []string `json:"capabilities"`
	LastError        string   `json:"last_error,omitempty"`
	LastConnectedAt  string   `json:"last_connected_at,omitempty"`
	LastDisconnected string   `json:"last_disconnected_at,omitempty"`
	PacketsRead      uint64   `json:"packets_read"`
	PacketsDropped   uint64   `json:"packets_dropped"`
	Reconnects       uint64   `json:"reconnects"`
}

type Transport interface {
	Connect(context.Context) error
	Close(context.Context) error
	Health() Health
	Name() string
	Subscribe(context.Context, func(topic string, payload []byte) error) error
}

func Build(cfg config.TransportConfig, log *logging.Logger, bus *events.Bus) (Transport, error) {
	switch cfg.Type {
	case "mqtt":
		return NewMQTT(cfg, log, bus), nil
	case "http", "serialtcp", "ble":
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
	return &Unsupported{cfg: cfg, health: Health{Name: cfg.Name, Type: cfg.Type, Unsupported: true, Detail: "feature-gated; not enabled in this release candidate", Capabilities: cfg.Capabilities}}
}

func (u *Unsupported) Connect(context.Context) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.health.OK = false
	u.health.LastError = "transport compiled as unsupported in this release candidate"
	return errors.New(u.health.LastError)
}
func (u *Unsupported) Close(context.Context) error { return nil }
func (u *Unsupported) Health() Health {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.health
}
func (u *Unsupported) Name() string                                                { return u.cfg.Name }
func (u *Unsupported) Subscribe(context.Context, func(string, []byte) error) error { return nil }

func dialWithTimeout(endpoint string) (net.Conn, error) {
	return net.DialTimeout("tcp", endpoint, 5*time.Second)
}
