package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/logging"
	"github.com/mel-project/mel/internal/transport"
)

type failingTransport struct {
	name       string
	typ        string
	connectErr error
	health     transport.Health
}

func (f *failingTransport) Connect(context.Context) error {
	f.health.Name = f.name
	f.health.Type = f.typ
	f.health.State = transport.StateError
	f.health.Detail = "connect failed"
	f.health.LastError = f.connectErr.Error()
	f.health.ReconnectAttempts++
	return f.connectErr
}

func (f *failingTransport) Close(context.Context) error { return nil }
func (f *failingTransport) Health() transport.Health    { return f.health }
func (f *failingTransport) Capabilities() transport.CapabilityMatrix {
	return transport.CapabilityMatrix{}
}
func (f *failingTransport) SourceType() string                                       { return f.typ }
func (f *failingTransport) Name() string                                             { return f.name }
func (f *failingTransport) Subscribe(context.Context, transport.PacketHandler) error { return nil }
func (f *failingTransport) SendPacket(context.Context, []byte) error                 { return nil }
func (f *failingTransport) FetchMetadata(context.Context) (map[string]any, error)    { return nil, nil }
func (f *failingTransport) FetchNodes(context.Context) ([]map[string]any, error)     { return nil, nil }
func (f *failingTransport) MarkIngest(time.Time)                                     {}
func (f *failingTransport) MarkDrop(string)                                          {}

func TestConsumeTransportEventsPersistsDeadLetterObservation(t *testing.T) {
	app := newTestApp(t, config.TransportConfig{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go app.consumeTransportEvents(ctx)
	time.Sleep(50 * time.Millisecond)
	app.Bus.Publish(events.Event{Type: "transport.observation", Data: transport.Observation{
		TransportName: "mqtt-primary",
		TransportType: "mqtt",
		Topic:         "msh/test/node",
		Reason:        "malformed mqtt publish",
		Detail:        "short publish",
		PayloadHex:    "0102",
		DeadLetter:    true,
		Details:       map[string]any{"endpoint": "127.0.0.1:1883"},
	}})

	waitFor(t, 2*time.Second, func() bool {
		count, err := app.DB.Scalar("SELECT COUNT(*) FROM dead_letters WHERE transport_name='mqtt-primary' AND reason='malformed mqtt publish';")
		return err == nil && count == "1"
	})
	rows, err := app.DB.QueryJSON("SELECT transport_name, topic, reason, payload_hex FROM dead_letters WHERE transport_name='mqtt-primary';")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["topic"] != "msh/test/node" || rows[0]["payload_hex"] != "0102" {
		t.Fatalf("unexpected dead letter rows: %+v", rows)
	}
}

func TestRunTransportPersistsRetryThresholdDeadLetter(t *testing.T) {
	tc := config.TransportConfig{Name: "direct-primary", Type: "tcp", Enabled: true, Endpoint: "127.0.0.1:4403", ReconnectSeconds: 1, MaxTimeouts: 2}
	app := newTestApp(t, tc)
	ft := &failingTransport{name: tc.Name, typ: tc.Type, connectErr: errors.New("dial tcp 127.0.0.1:4403: connect: connection refused")}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		app.runTransport(ctx, ft, tc)
	}()

	waitFor(t, 3*time.Second, func() bool {
		count, err := app.DB.Scalar("SELECT COUNT(*) FROM dead_letters WHERE transport_name='direct-primary' AND reason='retry threshold exceeded during connect';")
		return err == nil && count == "1"
	})
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runTransport did not stop after cancellation")
	}
}

func newTestApp(t *testing.T, tc config.TransportConfig) *App {
	t.Helper()
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Transports = []config.TransportConfig{tc}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return &App{
		Cfg: cfg,
		Log: logging.New("debug", true),
		DB:  database,
		Bus: events.New(),
	}
}

func waitFor(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
