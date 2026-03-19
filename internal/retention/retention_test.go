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
	base := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		if err := database.InsertTransportHealthSnapshot(db.TransportHealthSnapshot{TransportName: "mqtt", TransportType: "mqtt", Score: 90 - i, State: "degraded", SnapshotTime: base.Add(time.Duration(i) * time.Hour).Format(time.RFC3339)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.InsertTransportHealthSnapshot(db.TransportHealthSnapshot{TransportName: "mqtt", TransportType: "mqtt", Score: 10, State: "failed", SnapshotTime: base.AddDate(0, 0, -10).Format(time.RFC3339)}); err != nil {
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
