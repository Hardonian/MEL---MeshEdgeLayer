package eventlog

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/kernel"
)

func requireSQLite3(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not found in PATH")
	}
}

func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "test-eventlog.db")
}

func TestOpenAndAppend(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t)
	log, err := Open(Config{DBPath: dbPath, NodeID: "mel-test-001"})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	evt := &kernel.Event{
		ID:           kernel.NewEventID(),
		Type:         kernel.EventObservation,
		Timestamp:    time.Now().UTC(),
		SourceNodeID: "mel-test-001",
		Subject:      "mqtt-local",
		Data:         []byte(`{"transport":"mqtt-local"}`),
	}

	seq, err := log.Append(evt)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if seq != 1 {
		t.Errorf("expected sequence 1, got %d", seq)
	}
	if evt.SequenceNum != 1 {
		t.Errorf("event sequence should be set to 1, got %d", evt.SequenceNum)
	}
	if evt.Checksum == "" {
		t.Error("expected checksum to be set")
	}
}

func TestAppendBatch(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t)
	log, err := Open(Config{DBPath: dbPath, NodeID: "mel-test-001"})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	events := make([]*kernel.Event, 10)
	for i := range events {
		events[i] = &kernel.Event{
			ID:           kernel.NewEventID(),
			Type:         kernel.EventObservation,
			Timestamp:    time.Now().UTC(),
			SourceNodeID: "mel-test-001",
			Data:         []byte(`{}`),
		}
	}

	if err := log.AppendBatch(events); err != nil {
		t.Fatalf("AppendBatch failed: %v", err)
	}

	if log.LastSequenceNum() != 10 {
		t.Errorf("expected last seq 10, got %d", log.LastSequenceNum())
	}
}

func TestQuery(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t)
	log, err := Open(Config{DBPath: dbPath, NodeID: "mel-test-001"})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Insert events of different types
	types := []kernel.EventType{
		kernel.EventObservation,
		kernel.EventAnomaly,
		kernel.EventObservation,
		kernel.EventTopologyUpdate,
		kernel.EventObservation,
	}
	for _, et := range types {
		evt := &kernel.Event{
			ID:           kernel.NewEventID(),
			Type:         et,
			Timestamp:    time.Now().UTC(),
			SourceNodeID: "mel-test-001",
			Data:         []byte(`{}`),
		}
		if _, err := log.Append(evt); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	// Query all
	all, err := log.Query(QueryFilter{Limit: 100})
	if err != nil {
		t.Fatalf("Query all failed: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("expected 5 events, got %d", len(all))
	}

	// Query by type
	obs, err := log.Query(QueryFilter{
		EventTypes: []kernel.EventType{kernel.EventObservation},
		Limit:      100,
	})
	if err != nil {
		t.Fatalf("Query by type failed: %v", err)
	}
	if len(obs) != 3 {
		t.Errorf("expected 3 observations, got %d", len(obs))
	}

	// Query after sequence
	after3, err := log.Query(QueryFilter{AfterSequence: 3, Limit: 100})
	if err != nil {
		t.Fatalf("Query after seq failed: %v", err)
	}
	if len(after3) != 2 {
		t.Errorf("expected 2 events after seq 3, got %d", len(after3))
	}
}

func TestSequenceRecovery(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t)

	// First open: insert events
	log1, err := Open(Config{DBPath: dbPath, NodeID: "mel-test-001"})
	if err != nil {
		t.Fatalf("Open 1 failed: %v", err)
	}
	for i := 0; i < 5; i++ {
		_, _ = log1.Append(&kernel.Event{
			ID:           kernel.NewEventID(),
			Type:         kernel.EventObservation,
			Timestamp:    time.Now().UTC(),
			SourceNodeID: "mel-test-001",
			Data:         []byte(`{}`),
		})
	}

	// Second open: should recover sequence
	log2, err := Open(Config{DBPath: dbPath, NodeID: "mel-test-001"})
	if err != nil {
		t.Fatalf("Open 2 failed: %v", err)
	}

	if log2.LastSequenceNum() != 5 {
		t.Errorf("expected recovered seq 5, got %d", log2.LastSequenceNum())
	}

	// New event should be seq 6
	seq, _ := log2.Append(&kernel.Event{
		ID:           kernel.NewEventID(),
		Type:         kernel.EventObservation,
		Timestamp:    time.Now().UTC(),
		SourceNodeID: "mel-test-001",
		Data:         []byte(`{}`),
	})
	if seq != 6 {
		t.Errorf("expected seq 6, got %d", seq)
	}
}

func TestCompaction(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t)
	log, err := Open(Config{DBPath: dbPath, NodeID: "mel-test-001"})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Insert 20 events with timestamps in the past
	for i := 0; i < 20; i++ {
		_, _ = log.Append(&kernel.Event{
			ID:           kernel.NewEventID(),
			Type:         kernel.EventObservation,
			Timestamp:    time.Now().UTC().Add(-time.Duration(20-i) * time.Hour),
			SourceNodeID: "mel-test-001",
			Data:         []byte(`{}`),
		})
	}

	count, _ := log.Count()
	if count != 20 {
		t.Errorf("expected 20 events, got %d", count)
	}

	// Compact: remove events older than 5 hours, keep at least 5
	_, err = log.Compact(time.Now().UTC().Add(-5*time.Hour), 5)
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	countAfter, _ := log.Count()
	if countAfter >= count {
		t.Errorf("expected fewer events after compaction, got %d (was %d)", countAfter, count)
	}
}

func TestStats(t *testing.T) {
	requireSQLite3(t)
	dbPath := tempDB(t)
	log, err := Open(Config{DBPath: dbPath, NodeID: "mel-test-001"})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	_, _ = log.Append(&kernel.Event{
		ID:           kernel.NewEventID(),
		Type:         kernel.EventObservation,
		Timestamp:    time.Now().UTC(),
		SourceNodeID: "mel-test-001",
		Data:         []byte(`{}`),
	})

	stats := log.Stats()
	if stats.Appended != 1 {
		t.Errorf("expected 1 appended, got %d", stats.Appended)
	}
	if stats.LastSequenceNum != 1 {
		t.Errorf("expected last seq 1, got %d", stats.LastSequenceNum)
	}
}

func TestExists(t *testing.T) {
	if Exists("/nonexistent/path/db.sqlite") {
		t.Error("should return false for nonexistent path")
	}

	f, _ := os.CreateTemp("", "test-exists-*.db")
	f.Close()
	defer os.Remove(f.Name())

	if !Exists(f.Name()) {
		t.Error("should return true for existing file")
	}
}
