package retention

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func TestRunPrunesTransportHealthSnapshotsByAgeAndCap(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Intelligence.Retention.HealthSnapshotDays = 7
	cfg.Intelligence.Retention.HealthSnapshotMaxRows = 3
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	base := now.Add(-48 * time.Hour)
	for i := 0; i < 5; i++ {
		if err := database.InsertTransportHealthSnapshot(db.TransportHealthSnapshot{TransportName: "mqtt", TransportType: "mqtt", Score: 90 - i, State: "degraded", SnapshotTime: base.Add(time.Duration(i) * time.Hour).Format(time.RFC3339)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.InsertTransportHealthSnapshot(db.TransportHealthSnapshot{TransportName: "mqtt", TransportType: "mqtt", Score: 10, State: "failed", SnapshotTime: now.AddDate(0, 0, -20).Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	if err := Run(database, cfg); err != nil {
		t.Fatal(err)
	}
	count, err := database.Scalar("SELECT COUNT(*) FROM transport_health_snapshots;")
	if err != nil {
		t.Fatal(err)
	}
	if count != "3" {
		t.Fatalf("expected capped snapshots after pruning, got %s", count)
	}
}

func TestRunPrunesTransportAnomalySnapshotsByAgeAndCap(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Intelligence.Retention.HealthSnapshotDays = 7
	cfg.Intelligence.Retention.HealthSnapshotMaxRows = 3
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	base := now.Add(-48 * time.Hour)
	for i := 0; i < 5; i++ {
		if err := database.UpsertTransportAnomalySnapshot(db.TransportAnomalySnapshot{BucketStart: base.Add(time.Duration(i) * time.Hour).Format(time.RFC3339), TransportName: "mqtt", TransportType: "mqtt", Reason: "timeout_failure", Count: uint64(i + 1)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.UpsertTransportAnomalySnapshot(db.TransportAnomalySnapshot{BucketStart: now.AddDate(0, 0, -20).Format(time.RFC3339), TransportName: "mqtt", TransportType: "mqtt", Reason: "dead_letter_burst", Count: 1, DeadLetters: 1}); err != nil {
		t.Fatal(err)
	}
	if err := Run(database, cfg); err != nil {
		t.Fatal(err)
	}
	count, err := database.Scalar("SELECT COUNT(*) FROM transport_anomaly_snapshots;")
	if err != nil {
		t.Fatal(err)
	}
	if count != "3" {
		t.Fatalf("expected capped anomaly snapshots after pruning, got %s", count)
	}
}
