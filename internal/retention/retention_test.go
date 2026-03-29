package retention

import (
	"fmt"
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

func TestRunPrunesTimelineAndActionOutcomeSnapshots(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DataDir = filepath.Join(t.TempDir(), "data")
	cfg.Storage.DatabasePath = filepath.Join(cfg.Storage.DataDir, "mel.db")
	cfg.Retention.AuditDays = 2
	cfg.Control.RetentionDays = 1
	cfg.Intelligence.Retention.HealthSnapshotMaxRows = 2
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	oldTimeline := now.AddDate(0, 0, -5).Format(time.RFC3339)
	newTimeline := now.Add(-2 * time.Hour).Format(time.RFC3339)
	if err := database.Exec(fmt.Sprintf(`INSERT INTO timeline_events(id,event_time,event_type,summary,severity,actor_id,resource_id,details_json) VALUES('tl-old','%s','operator_note','old event','info','op-a','inc-1','{}');`, oldTimeline)); err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(fmt.Sprintf(`INSERT INTO timeline_events(id,event_time,event_type,summary,severity,actor_id,resource_id,details_json) VALUES('tl-new','%s','operator_note','new event','info','op-a','inc-1','{}');`, newTimeline)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		derivedAt := now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339)
		if i == 3 {
			derivedAt = now.AddDate(0, 0, -3).Format(time.RFC3339)
		}
		sql := fmt.Sprintf(`INSERT INTO incident_action_outcome_snapshots(snapshot_id,signature_key,incident_id,action_id,action_type,derived_classification,evidence_sufficiency,derivation_window_start,derivation_window_end,derived_at,created_at,updated_at) VALUES('snap-%d','sig-1','inc-1','act-%d','reroute','directional','partial','%s','%s','%s','%s','%s');`,
			i, i, now.Add(-6*time.Hour).Format(time.RFC3339), now.Format(time.RFC3339), derivedAt, derivedAt, derivedAt)
		if err := database.Exec(sql); err != nil {
			t.Fatal(err)
		}
	}
	if err := Run(database, cfg); err != nil {
		t.Fatal(err)
	}
	timelineCount, err := database.Scalar("SELECT COUNT(*) FROM timeline_events;")
	if err != nil {
		t.Fatal(err)
	}
	if timelineCount != "1" {
		t.Fatalf("expected old timeline rows pruned, got count=%s", timelineCount)
	}
	snapshotCount, err := database.Scalar("SELECT COUNT(*) FROM incident_action_outcome_snapshots;")
	if err != nil {
		t.Fatal(err)
	}
	if snapshotCount != "2" {
		t.Fatalf("expected action outcome snapshots capped/pruned to 2 rows, got %s", snapshotCount)
	}
}
