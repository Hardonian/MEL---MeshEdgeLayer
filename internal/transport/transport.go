package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
)

type Health struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	OK           bool     `json:"ok"`
	Detail       string   `json:"detail"`
	Capabilities []string `json:"capabilities"`
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

type Unsupported struct{ cfg config.TransportConfig }

func NewUnsupported(cfg config.TransportConfig) *Unsupported { return &Unsupported{cfg: cfg} }
func (u *Unsupported) Connect(context.Context) error {
	return errors.New("transport compiled as unsupported in v0.1")
}
func (u *Unsupported) Close(context.Context) error { return nil }
func (u *Unsupported) Health() Health {
	return Health{Name: u.cfg.Name, Type: u.cfg.Type, Detail: "feature-gated; not enabled in this milestone", Capabilities: u.cfg.Capabilities}
}
func (u *Unsupported) Name() string                                                { return u.cfg.Name }
func (u *Unsupported) Subscribe(context.Context, func(string, []byte) error) error { return nil }

func dialWithTimeout(endpoint string) (net.Conn, error) {
	return net.DialTimeout("tcp", endpoint, 5*time.Second)
}
