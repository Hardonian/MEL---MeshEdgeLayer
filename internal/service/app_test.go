package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
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
	episodeSeq int
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

func (f *failingTransport) ForceState(state, detail, lastError string) {
	f.health.State = state
	f.health.Detail = detail
	f.health.LastError = lastError
}

func (f *failingTransport) BeginFailureEpisode(err error) (string, uint64) {
	if f.health.EpisodeID == "" {
		f.episodeSeq++
		f.health.EpisodeID = fmt.Sprintf("%s-episode-%d", f.name, f.episodeSeq)
	}
	f.health.FailureCount++
	f.health.LastFailureAt = time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		f.health.LastError = err.Error()
	}
	return f.health.EpisodeID, f.health.FailureCount
}

func (f *failingTransport) CloseFailureEpisode() {
	f.health.FailureCount = 0
	f.health.EpisodeID = ""
	f.health.LastFailureAt = ""
}

func (f *failingTransport) RecordObservationDrop(count uint64) {
	f.health.ObservationDrops += count
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

func TestConsumeTransportEventsPersistsFinalObservationDeadLetter(t *testing.T) {
	app := newTestApp(t, config.TransportConfig{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.startWorkers(ctx)
	time.Sleep(50 * time.Millisecond)
	app.Bus.Publish(events.Event{Type: "transport.observation", Data: transport.NewObservation("mqtt-primary", "mqtt", "msh/test/node", transport.ReasonRejectedPublish, []byte{0x01, 0x02}, true, "publish rejected", map[string]any{"endpoint": "127.0.0.1:1883", "final": true})})

	waitFor(t, 2*time.Second, func() bool {
		count, err := app.DB.Scalar("SELECT COUNT(*) FROM dead_letters WHERE transport_name='mqtt-primary' AND reason='rejected_publish';")
		return err == nil && count == "1"
	})
	rows, err := app.DB.QueryJSON("SELECT transport_name, transport_type, topic, reason, payload_hex FROM dead_letters WHERE transport_name='mqtt-primary';")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["topic"] != "msh/test/node" || rows[0]["payload_hex"] != "0102" || rows[0]["transport_type"] != "mqtt" {
		t.Fatalf("unexpected dead letter rows: %+v", rows)
	}
	auditCount, err := app.DB.Scalar("SELECT COUNT(*) FROM audit_logs WHERE category='transport' AND message='rejected_publish';")
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
	app.startWorkers(ctx)

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
	app.wg.Wait()
}

func TestRunTransportDoesNotDuplicateRetryThresholdDeadLetter(t *testing.T) {
	tc := config.TransportConfig{Name: "direct-primary", Type: "tcp", Enabled: true, Endpoint: "127.0.0.1:4403", ReconnectSeconds: 1, MaxTimeouts: 2}
	app := newTestApp(t, tc)
	ft := &failingTransport{name: tc.Name, typ: tc.Type, connectErr: errors.New("dial tcp 127.0.0.1:4403: connect: connection refused")}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.startWorkers(ctx)
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
	app.startWorkers(ctx)
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
	cfg.Intelligence.Retention.PruneEverySeconds = 0
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return &App{
		Cfg:                 cfg,
		Log:                 logging.New("debug", true),
		DB:                  database,
		Bus:                 events.New(),
		dlEpisodes:          map[string]deadLetterEpisode{},
		observationEpisodes: map[string]deadLetterEpisode{},
		ingestCh:            make(chan ingestRequest, defaultIngestQueueSize),
		observationCh:       make(chan transport.Observation, defaultObservationQueueSize),
		controlQueue:        make(chan control.ControlAction, cfg.Control.MaxQueue),
		transportControls:   map[string]*transportControlState{tc.Name: newTransportControlState()},
		intelligenceEvery:   -1,
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

func mustInsertTransportAuditAt(t *testing.T, database *db.DB, createdAt time.Time, name, typ, message string, details map[string]any) {
	t.Helper()
	if details == nil {
		details = map[string]any{}
	}
	if _, ok := details["transport"]; !ok {
		details["transport"] = name
	}
	if _, ok := details["type"]; !ok {
		details["type"] = typ
	}
	if err := database.InsertAuditLog("transport", "warning", message, details); err != nil {
		t.Fatal(err)
	}
	if err := database.Exec("UPDATE audit_logs SET created_at='" + createdAt.UTC().Format(time.RFC3339) + "' WHERE id=(SELECT MAX(id) FROM audit_logs);"); err != nil {
		t.Fatal(err)
	}
}

type stubTransport struct {
	name   string
	typ    string
	health transport.Health
}

func (s *stubTransport) Connect(context.Context) error { return nil }
func (s *stubTransport) Close(context.Context) error   { return nil }
func (s *stubTransport) Health() transport.Health      { return s.health }
func (s *stubTransport) Capabilities() transport.CapabilityMatrix {
	return transport.CapabilityMatrix{}
}
func (s *stubTransport) SourceType() string                                       { return s.typ }
func (s *stubTransport) Name() string                                             { return s.name }
func (s *stubTransport) Subscribe(context.Context, transport.PacketHandler) error { return nil }
func (s *stubTransport) SendPacket(context.Context, []byte) error                 { return nil }
func (s *stubTransport) FetchMetadata(context.Context) (map[string]any, error)    { return nil, nil }
func (s *stubTransport) FetchNodes(context.Context) ([]map[string]any, error)     { return nil, nil }
func (s *stubTransport) MarkIngest(time.Time)                                     {}
func (s *stubTransport) MarkDrop(string)                                          {}
func (s *stubTransport) ForceState(state, detail, lastError string)               {}
func (s *stubTransport) BeginFailureEpisode(err error) (string, uint64)           { return "", 0 }
func (s *stubTransport) CloseFailureEpisode()                                     {}
func (s *stubTransport) RecordObservationDrop(count uint64) {
	s.health.ObservationDrops += count
}

func TestEnqueueIngestDropsWhenQueueFull(t *testing.T) {
	tc := config.TransportConfig{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"}
	app := newTestApp(t, tc)
	stub := &stubTransport{name: tc.Name, typ: tc.Type}
	for i := 0; i < cap(app.ingestCh); i++ {
		app.ingestCh <- ingestRequest{transport: stub, topic: tc.Topic, payload: []byte{0x01}}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := time.Now()
	err := app.enqueueIngest(ctx, stub, tc.Topic, []byte{0x02})
	if err == nil {
		t.Fatal("expected queue-full error")
	}
	if time.Since(started) > 200*time.Millisecond {
		t.Fatalf("enqueue blocked too long: %v", time.Since(started))
	}
}

func TestObservationPipelineHandlesBurstWithoutDeadlock(t *testing.T) {
	tc := config.TransportConfig{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"}
	app := newTestApp(t, tc)
	ctx, cancel := context.WithCancel(context.Background())
	app.startWorkers(ctx)
	for i := 0; i < 10000; i++ {
		app.emitTransportObservation(tc, transport.NewObservation(tc.Name, tc.Type, tc.Topic, transport.ReasonTimeoutStall, []byte{0x01, 0x02}, false, "stall", map[string]any{"sequence": i}))
	}
	waitFor(t, 5*time.Second, func() bool {
		count, err := app.DB.Scalar("SELECT COUNT(*) FROM audit_logs WHERE category='transport' AND message='timeout_stall';")
		if err != nil {
			return false
		}
		return count != "0"
	})
	cancel()
	app.wg.Wait()
}

func TestWorkerShutdownCompletes(t *testing.T) {
	app := newTestApp(t, config.TransportConfig{Name: "mqtt-primary", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test"})
	ctx, cancel := context.WithCancel(context.Background())
	app.startWorkers(ctx)
	cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		app.wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("workers did not stop after cancellation")
	}
}
