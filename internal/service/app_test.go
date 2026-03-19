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
	connectSeq []error
	idx        int
	health     transport.Health
}

func (f *failingTransport) Connect(context.Context) error {
	f.health.Name = f.name
	f.health.Type = f.typ
	f.health.ReconnectAttempts++
	var err error
	if len(f.connectSeq) > 0 {
		if f.idx >= len(f.connectSeq) {
			err = f.connectSeq[len(f.connectSeq)-1]
		} else {
			err = f.connectSeq[f.idx]
		}
		f.idx++
	} else {
		err = f.connectErr
	}
	if err != nil {
		f.health.State = transport.StateFailed
		f.health.Detail = "connect failed"
		f.health.LastError = err.Error()
		return err
	}
	f.health.State = transport.StateIdle
	f.health.Detail = "connected"
	f.health.LastError = ""
	return nil
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
		Reason:        transport.ReasonMalformedPublish,
		Detail:        "short publish",
		PayloadHex:    "0102",
		DeadLetter:    true,
		Details:       map[string]any{"endpoint": "127.0.0.1:1883"},
	}})

	waitFor(t, 2*time.Second, func() bool {
		count, err := app.DB.Scalar("SELECT COUNT(*) FROM dead_letters WHERE transport_name='mqtt-primary' AND reason='malformed_publish';")
		return err == nil && count == "1"
	})
	rows, err := app.DB.QueryJSON("SELECT transport_name, transport_type, topic, reason, payload_hex FROM dead_letters WHERE transport_name='mqtt-primary';")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["topic"] != "msh/test/node" || rows[0]["payload_hex"] != "0102" || rows[0]["transport_type"] != "mqtt" {
		t.Fatalf("unexpected dead letter rows: %+v", rows)
	}
	auditCount, err := app.DB.Scalar("SELECT COUNT(*) FROM audit_logs WHERE category='transport' AND message='malformed_publish';")
	if err != nil || auditCount != "1" {
		t.Fatalf("expected transport audit log, count=%s err=%v", auditCount, err)
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
	go app.consumeTransportEvents(ctx)

	waitFor(t, 3*time.Second, func() bool {
		count, err := app.DB.Scalar("SELECT COUNT(*) FROM dead_letters WHERE transport_name='direct-primary' AND reason='retry_threshold_exceeded';")
		return err == nil && count == "1"
	})
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runTransport did not stop after cancellation")
	}
}

func TestRunTransportDoesNotDuplicateRetryThresholdDeadLetter(t *testing.T) {
	tc := config.TransportConfig{Name: "direct-primary", Type: "tcp", Enabled: true, Endpoint: "127.0.0.1:4403", ReconnectSeconds: 1, MaxTimeouts: 2}
	app := newTestApp(t, tc)
	ft := &failingTransport{name: tc.Name, typ: tc.Type, connectErr: errors.New("dial tcp 127.0.0.1:4403: connect: connection refused")}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go app.consumeTransportEvents(ctx)
	go app.runTransport(ctx, ft, tc)

	waitFor(t, 4*time.Second, func() bool {
		count, err := app.DB.Scalar("SELECT COUNT(*) FROM dead_letters WHERE transport_name='direct-primary' AND reason='retry_threshold_exceeded';")
		return err == nil && count == "1"
	})
	time.Sleep(1200 * time.Millisecond)
	count, err := app.DB.Scalar("SELECT COUNT(*) FROM dead_letters WHERE transport_name='direct-primary' AND reason='retry_threshold_exceeded';")
	if err != nil {
		t.Fatal(err)
	}
	if count != "1" {
		t.Fatalf("expected one retry_threshold_exceeded dead letter, got %s", count)
	}
}

func TestRunTransportRecoveryResetsRetryThresholdEpisode(t *testing.T) {
	tc := config.TransportConfig{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test", ReconnectSeconds: 1, MaxTimeouts: 2}
	app := newTestApp(t, tc)
	ft := &failingTransport{
		name: tc.Name,
		typ:  tc.Type,
		connectSeq: []error{
			errors.New("connect refused 1"),
			errors.New("connect refused 2"),
			nil,
			errors.New("connect refused 3"),
			errors.New("connect refused 4"),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go app.consumeTransportEvents(ctx)
	go app.runTransport(ctx, ft, tc)

	waitFor(t, 4*time.Second, func() bool {
		count, err := app.DB.Scalar("SELECT COUNT(*) FROM dead_letters WHERE transport_name='mqtt-primary' AND reason='retry_threshold_exceeded';")
		return err == nil && count == "1"
	})
	waitFor(t, 5*time.Second, func() bool {
		count, err := app.DB.Scalar("SELECT COUNT(*) FROM dead_letters WHERE transport_name='mqtt-primary' AND reason='retry_threshold_exceeded';")
		return err == nil && count == "2"
	})
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
		Cfg:        cfg,
		Log:        logging.New("debug", true),
		DB:         database,
		Bus:        events.New(),
		dlEpisodes: map[string]deadLetterEpisode{},
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
