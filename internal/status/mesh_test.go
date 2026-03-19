package status

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

func TestInspectMeshPopulatesCorrelatedNodeIDs(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.DatabasePath = filepath.Join(t.TempDir(), "mel.db")
	cfg.Storage.DataDir = filepath.Dir(cfg.Storage.DatabasePath)
	cfg.Transports = []config.TransportConfig{
		{Name: "mqtt-a", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1883", Topic: "msh/test", ClientID: "a"},
		{Name: "mqtt-b", Type: "mqtt", Enabled: true, Endpoint: "127.0.0.1:1884", Topic: "msh/test", ClientID: "b"},
	}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	for _, node := range []map[string]any{
		{"node_num": int64(10), "node_id": "!0010", "long_name": "A", "short_name": "A", "last_seen": now.Format(time.RFC3339), "last_gateway_id": "gw-a", "last_snr": 1.0, "last_rssi": -70, "lat_redacted": 0.0, "lon_redacted": 0.0, "altitude": 0},
		{"node_num": int64(11), "node_id": "!0011", "long_name": "B", "short_name": "B", "last_seen": now.Format(time.RFC3339), "last_gateway_id": "gw-b", "last_snr": 1.0, "last_rssi": -70, "lat_redacted": 0.0, "lon_redacted": 0.0, "altitude": 0},
	} {
		if err := database.UpsertNode(node); err != nil {
			t.Fatal(err)
		}
	}
	for i, item := range []struct {
		name string
		node int64
		hash string
	}{
		{name: "mqtt-a", node: 10, hash: "mesh-a"},
		{name: "mqtt-b", node: 11, hash: "mesh-b"},
	} {
		if _, err := database.InsertMessage(map[string]any{
			"transport_name": item.name,
			"packet_id":      int64(i + 1),
			"dedupe_hash":    item.hash,
			"channel_id":     "",
			"gateway_id":     "",
			"from_node":      item.node,
			"to_node":        int64(0),
			"portnum":        int64(1),
			"payload_text":   "hello",
			"payload_json":   map[string]any{"message_type": "text"},
			"raw_hex":        "01",
			"rx_time":        now.Add(-30 * time.Second).Format(time.RFC3339),
			"hop_limit":      int64(0),
			"relay_node":     int64(0),
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, transportName := range []string{"mqtt-a", "mqtt-b"} {
		if err := database.UpsertTransportAlert(db.TransportAlertRecord{
			ID:               transportName + "|subscribe_failure|cluster",
			TransportName:    transportName,
			TransportType:    "mqtt",
			Severity:         "critical",
			Reason:           "subscribe_failure",
			Summary:          "subscribe failure cluster",
			FirstTriggeredAt: now.Add(-2 * time.Minute).Format(time.RFC3339),
			LastUpdatedAt:    now.Add(-20 * time.Second).Format(time.RFC3339),
			Active:           true,
			ClusterKey:       "sub-fail",
			TriggerCondition: "cluster_count=2",
		}); err != nil {
			t.Fatal(err)
		}
	}
	drilldown, err := InspectMesh(cfg, database, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(drilldown.CorrelatedFailures) == 0 {
		t.Fatalf("expected correlated failures, got %+v", drilldown)
	}
	if len(drilldown.CorrelatedFailures[0].NodeIDs) == 0 {
		t.Fatalf("expected node attribution in correlated failures, got %+v", drilldown.CorrelatedFailures[0])
	}
	if len(drilldown.MeshHealthExplanation.AffectedNodes) == 0 {
		t.Fatalf("expected affected nodes in mesh explanation, got %+v", drilldown.MeshHealthExplanation)
	}
}
