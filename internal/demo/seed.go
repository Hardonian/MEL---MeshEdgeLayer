package demo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/transport"
)

const (
	sandboxDBFile   = "demo_sandbox.db"
	manifestVersion = "demo.evidence.v1"
)

// SeedOptions controls sandbox checks and optional evidence capture.
type SeedOptions struct {
	Force       bool
	CaptureDir  string
	MELBinary   string // path to mel CLI; empty skips capture
	SkipCapture bool
}

// Execute seeds the database for the scenario. It requires a sandbox config
// (see IsSandboxConfig) unless Force is true or MEL_DEMO_FORCE=1.
func Execute(cfg config.Config, scenarioID string, opt SeedOptions) (DemoEvidenceBundle, error) {
	var empty DemoEvidenceBundle
	if err := IsSandboxConfig(cfg, opt.Force); err != nil {
		return empty, err
	}
	sc := ScenarioByID(scenarioID)
	if sc == nil {
		return empty, fmt.Errorf("demo: unknown scenario %q (use mel demo scenarios)", scenarioID)
	}
	database, err := db.Open(cfg)
	if err != nil {
		return empty, err
	}
	now := time.Date(2026, 3, 22, 14, 0, 0, 0, time.UTC)
	if err := clearScenarioTables(database, scenarioID, cfg); err != nil {
		return empty, err
	}
	transports := transportsForScenario(sc, cfg)
	for _, tr := range transports {
		if err := seedTransportRuntime(database, tr, scenarioID, now); err != nil {
			return empty, err
		}
	}
	if err := seedNodesAndMessages(database, sc, transports, now); err != nil {
		return empty, err
	}
	if err := seedAlertsIncidentsDeadLetters(database, sc, now); err != nil {
		return empty, err
	}
	if err := seedAnomalySnapshots(database, sc, now); err != nil {
		return empty, err
	}

	bundle := DemoEvidenceBundle{
		GeneratedAt:       time.Now().UTC(),
		ScenarioID:        scenarioID,
		ConfigPath:        "", // filled by CLI when known
		DatabasePath:      cfg.Storage.DatabasePath,
		ManifestVersion:   manifestVersion,
		SandboxMarkerNote: "Seeded data is confined to demo_sandbox.db; do not point production configs here.",
	}
	if opt.CaptureDir != "" {
		if err := os.MkdirAll(opt.CaptureDir, 0o755); err != nil {
			return empty, err
		}
	}
	return bundle, nil
}

// IsSandboxConfig returns an error if the config does not look like an explicit demo sandbox.
func IsSandboxConfig(cfg config.Config, force bool) error {
	if force || os.Getenv("MEL_DEMO_FORCE") == "1" {
		return nil
	}
	base := filepath.Base(cfg.Storage.DatabasePath)
	if base != sandboxDBFile {
		return fmt.Errorf("demo: database_path must end with %q for sandbox seeding (got %q); use configs/profiles/demo-sandbox.json or pass --force", sandboxDBFile, base)
	}
	p := strings.ToLower(cfg.Storage.DatabasePath)
	if !strings.Contains(p, "demo_sandbox") && !strings.Contains(p, "demo-sandbox") {
		return fmt.Errorf("demo: database path should live under a demo_sandbox directory for safety")
	}
	return nil
}

func transportsForScenario(sc *DemoScenario, cfg config.Config) []config.TransportConfig {
	// Prefer transports from loaded config when names align; else synthesize minimal valid transports.
	byName := map[string]config.TransportConfig{}
	for _, t := range cfg.Transports {
		byName[t.Name] = t
	}
	switch sc.Profile {
	case ProfilePrivateRFOnly, ProfileStoreForwardRelay:
		name := "local-serial"
		if t, ok := byName[name]; ok {
			return []config.TransportConfig{t}
		}
		return []config.TransportConfig{{
			Name: name, Type: "serial", Enabled: true,
			SerialDevice: "/dev/serial/by-id/usb-MESHTASTIC_DEMO-if00", SerialBaud: 115200,
			ReadTimeoutSec: 30, WriteTimeoutSec: 30, MaxTimeouts: 5, ReconnectSeconds: 10,
			MQTTKeepAliveSec: 60,
		}}
	case ProfileRFPlusMQTTBridge:
		var rfCfg, mqttCfg config.TransportConfig
		rf := "rf-serial"
		if t, ok := byName[rf]; ok {
			rfCfg = t
		} else {
			rfCfg = config.TransportConfig{
				Name: rf, Type: "serial", Enabled: true,
				SerialDevice: "/dev/serial/by-id/usb-MESHTASTIC_DEMO-if00", SerialBaud: 115200,
				ReadTimeoutSec: 30, WriteTimeoutSec: 30, MaxTimeouts: 5, ReconnectSeconds: 10,
				MQTTKeepAliveSec: 60,
			}
		}
		for _, br := range sc.Bridges {
			if t, ok := byName[br.Name]; ok {
				mqttCfg = t
				break
			}
			mqttCfg = config.TransportConfig{
				Name: br.Name, Type: "mqtt", Enabled: true, Endpoint: br.Endpoint, Topic: br.Topic, ClientID: br.ClientID,
				ReadTimeoutSec: 30, WriteTimeoutSec: 30, MaxTimeouts: 5, ReconnectSeconds: 10,
				MQTTQoS: 1, MQTTKeepAliveSec: 60, MQTTCleanSession: true,
			}
			break
		}
		// MQTT-first so default ingest path matches bridge-centric narratives.
		if mqttCfg.Name != "" {
			return []config.TransportConfig{mqttCfg, rfCfg}
		}
		return []config.TransportConfig{rfCfg}
	case ProfileDualMQTTIngest:
		out := make([]config.TransportConfig, 0, len(sc.Bridges)+1)
		rf := "rf-serial"
		if t, ok := byName[rf]; ok {
			out = append(out, t)
		} else {
			out = append(out, config.TransportConfig{
				Name: rf, Type: "serial", Enabled: true,
				SerialDevice: "/dev/serial/by-id/usb-MESHTASTIC_DEMO-if00", SerialBaud: 115200,
				ReadTimeoutSec: 30, WriteTimeoutSec: 30, MaxTimeouts: 5, ReconnectSeconds: 10,
				MQTTKeepAliveSec: 60,
			})
		}
		for _, br := range sc.Bridges {
			if t, ok := byName[br.Name]; ok {
				out = append(out, t)
				continue
			}
			out = append(out, config.TransportConfig{
				Name: br.Name, Type: "mqtt", Enabled: true, Endpoint: br.Endpoint, Topic: br.Topic, ClientID: br.ClientID,
				ReadTimeoutSec: 30, WriteTimeoutSec: 30, MaxTimeouts: 5, ReconnectSeconds: 10,
				MQTTQoS: 1, MQTTKeepAliveSec: 60, MQTTCleanSession: true,
			})
		}
		return out
	default:
		return nil
	}
}

func seedTransportRuntime(d *db.DB, t config.TransportConfig, scenarioID string, now time.Time) error {
	src := t.SourceLabel()
	state := transport.StateIdle
	detail := "demo sandbox: idle (no live broker or radio attached)"
	msgs := uint64(12)
	switch {
	case t.Type == "mqtt":
		state = transport.StateLive
		detail = "demo sandbox: synthetic connected state for fixture"
		msgs = 240
	case t.Type == "serial":
		msgs = 180
	}
	tr := db.TransportRuntime{
		Name:            t.Name,
		Type:            t.Type,
		Source:          src,
		Enabled:         t.Enabled,
		State:           state,
		Detail:          detail,
		LastAttemptAt:   now.Add(-2 * time.Minute).Format(time.RFC3339),
		LastConnectedAt: now.Add(-90 * time.Second).Format(time.RFC3339),
		LastSuccessAt:   now.Add(-60 * time.Second).Format(time.RFC3339),
		LastMessageAt:   now.Add(-20 * time.Second).Format(time.RFC3339),
		TotalMessages:   msgs,
	}
	if t.Name == "local-serial" || t.Name == "rf-serial" {
		tr.LastHeartbeatAt = now.Add(-15 * time.Second).Format(time.RFC3339)
		tr.PacketsDropped = 0
		tr.Reconnects = 0
		tr.Timeouts = 0
		tr.FailureCount = 0
		tr.ObservationDrops = 0
	}
	if strings.HasPrefix(t.Name, "mqtt") || t.Type == "mqtt" {
		tr.LastHeartbeatAt = now.Add(-10 * time.Second).Format(time.RFC3339)
		tr.PacketsDropped = 2
		tr.Reconnects = 1
		tr.Timeouts = 0
		tr.FailureCount = 0
		tr.ObservationDrops = 0
	}
	if t.Name == "mqtt-uplink" {
		tr.ObservationDrops = 40
		tr.LastObservationDrop = now.Add(-30 * time.Second).Format(time.RFC3339)
	}
	if scenarioID == "store-and-forward" && t.Name == "local-serial" {
		tr.ObservationDrops = 220
		tr.LastObservationDrop = now.Add(-25 * time.Second).Format(time.RFC3339)
		tr.PacketsDropped = 8
	}
	return d.UpsertTransportRuntime(tr)
}

func seedNodesAndMessages(d *db.DB, sc *DemoScenario, transports []config.TransportConfig, now time.Time) error {
	transportNames := make([]string, 0, len(transports))
	for _, t := range transports {
		transportNames = append(transportNames, t.Name)
	}
	defaultTransport := transportNames[0]
	if defaultTransport == "" {
		return fmt.Errorf("demo: no transports for scenario")
	}

	for _, n := range sc.Nodes {
		nodeRow := map[string]any{
			"node_num": n.NodeNum, "node_id": n.NodeID, "long_name": n.LongName, "short_name": n.ShortName,
			"last_seen":       now.Add(-5 * time.Minute).Format(time.RFC3339),
			"last_gateway_id": chooseGateway(n.GatewayID, defaultTransport), "last_snr": n.LastSNR, "last_rssi": n.LastRSSI,
			"lat_redacted": 0.0, "lon_redacted": 0.0, "altitude": n.AltitudeM,
		}
		if err := d.UpsertNode(nodeRow); err != nil {
			return err
		}
	}

	for i, n := range sc.Nodes {
		tname := defaultTransport
		if sc.ID == "rf-mqtt-duplicate-path" {
			if i == 0 {
				tname = "rf-serial"
			} else {
				tname = "mqtt-uplink"
			}
		}
		hop := int64(1)
		relay := int64(0)
		if sc.ID == "handheld-as-backbone" && n.NodeNum == 0x3003 {
			hop = 2
			relay = 0x3002
		}
		if sc.ID == "healthy-private-mesh" || sc.ID == "indoor-gateway-vs-rooftop" {
			if n.GatewayID != "" && strings.HasPrefix(n.GatewayID, "!") {
				hop = 1
			}
		}
		dedupe := fmt.Sprintf("demo-%s-%d", sc.ID, n.NodeNum)
		msg := map[string]any{
			"transport_name": tname,
			"packet_id":      int64(i + 1),
			"dedupe_hash":    dedupe,
			"channel_id":     "Demo",
			"gateway_id":     n.GatewayID,
			"from_node":      n.NodeNum,
			"to_node":        int64(0),
			"portnum":        int64(1),
			"payload_text":   "demo",
			"payload_json":   map[string]any{"mel_demo_scenario": sc.ID, "role": n.Role},
			"raw_hex":        fmt.Sprintf("de%04x", int(n.NodeNum&0xffff)),
			"rx_time":        now.Add(-time.Duration(30+i*5) * time.Second).Format(time.RFC3339),
			"hop_limit":      hop,
			"relay_node":     relay,
		}
		if _, err := d.InsertMessage(msg); err != nil {
			return err
		}
	}

	// Duplicate-path: second path carries a plausible duplicate (different dedupe_hash;
	// operators still validate byte-identical dedupe behavior in production).
	if sc.ID == "rf-mqtt-duplicate-path" && len(transportNames) >= 2 {
		n := sc.Nodes[0]
		msg := map[string]any{
			"transport_name": "mqtt-uplink",
			"packet_id":      int64(99),
			"dedupe_hash":    fmt.Sprintf("demo-%s-%d-mqtt", sc.ID, n.NodeNum),
			"channel_id":     "Demo",
			"gateway_id":     n.GatewayID,
			"from_node":      n.NodeNum,
			"to_node":        int64(0),
			"portnum":        int64(1),
			"payload_text":   "demo",
			"payload_json":   map[string]any{"mel_demo_scenario": sc.ID, "path": "mqtt", "note": "second ingest path"},
			"raw_hex":        fmt.Sprintf("df%04x", int(n.NodeNum&0xffff)),
			"rx_time":        now.Add(-10 * time.Second).Format(time.RFC3339),
			"hop_limit":      int64(1),
			"relay_node":     int64(0),
		}
		if _, err := d.InsertMessage(msg); err != nil {
			return err
		}
	}

	return nil
}

func chooseGateway(gw, defaultTransport string) string {
	if gw == "" {
		return defaultTransport
	}
	return gw
}

func seedAlertsIncidentsDeadLetters(d *db.DB, sc *DemoScenario, now time.Time) error {
	switch sc.ID {
	case "rf-mqtt-duplicate-path":
		for _, pair := range []struct {
			name, idSuffix, reason string
		}{
			{"rf-serial", "retry", transport.ReasonRetryThresholdExceeded},
			{"mqtt-uplink", "sub", transport.ReasonSubscribeFailure},
		} {
			if err := d.UpsertTransportAlert(db.TransportAlertRecord{
				ID:               fmt.Sprintf("demo-%s-%s|%s|fixture", sc.ID, pair.name, pair.idSuffix),
				TransportName:    pair.name,
				TransportType:    map[string]string{"rf-serial": "serial", "mqtt-uplink": "mqtt"}[pair.name],
				Severity:         "high",
				Reason:           pair.reason,
				Summary:          "Demo fixture: correlated stress across RF and MQTT paths",
				FirstTriggeredAt: now.Add(-3 * time.Minute).Format(time.RFC3339),
				LastUpdatedAt:    now.Add(-1 * time.Minute).Format(time.RFC3339),
				Active:           true,
				ClusterKey:       "demo-dual-path",
				TriggerCondition: "fixture_seed",
			}); err != nil {
				return err
			}
		}
	case "store-and-forward":
		if err := d.UpsertTransportAlert(db.TransportAlertRecord{
			ID:               "demo-store-forward|obs|fixture",
			TransportName:    "local-serial",
			TransportType:    "serial",
			Severity:         "medium",
			Reason:           transport.ReasonObservationDropped,
			Summary:          "Demo fixture: relay observation drops (store/forward pressure)",
			FirstTriggeredAt: now.Add(-4 * time.Minute).Format(time.RFC3339),
			LastUpdatedAt:    now.Add(-30 * time.Second).Format(time.RFC3339),
			Active:           true,
			ClusterKey:       "demo-store-forward",
			TriggerCondition: "observation_drops>0",
		}); err != nil {
			return err
		}
		if err := d.InsertDeadLetter(db.DeadLetter{
			TransportName: "local-serial",
			TransportType: "serial",
			Topic:         "",
			Reason:        transport.ReasonObservationDropped,
			PayloadHex:    "00",
			Details:       map[string]any{"mel_demo_scenario": sc.ID, "note": "synthetic dead letter for drill"},
		}); err != nil {
			return err
		}
	case "mqtt-privacy-json-risk":
		if err := d.UpsertIncident(models.Incident{
			ID:           "demo-mqtt-privacy-review",
			Category:     "security",
			Severity:     "high",
			Title:        "MQTT bridge exposure review",
			Summary:      "Wide topic filter with cleartext broker and map reporting enabled — operator validation required.",
			ResourceType: "transport",
			ResourceID:   "mqtt-bridge",
			State:        "open",
			OccurredAt:   now.Add(-2 * time.Hour).Format(time.RFC3339),
			Metadata:     map[string]any{"mel_demo_scenario": sc.ID},
		}); err != nil {
			return err
		}
	}
	return nil
}

func seedAnomalySnapshots(d *db.DB, sc *DemoScenario, now time.Time) error {
	if sc.ID != "store-and-forward" {
		return nil
	}
	bucket := now.Add(-5 * time.Minute).Truncate(time.Minute).Format(time.RFC3339)
	return d.UpsertTransportAnomalySnapshot(db.TransportAnomalySnapshot{
		BucketStart:      bucket,
		TransportName:    "local-serial",
		TransportType:    "serial",
		Reason:           transport.ReasonObservationDropped,
		Count:            12,
		DeadLetters:      1,
		ObservationDrops: 12,
		DropCauses:       map[string]uint64{"internal_queue": 12},
	})
}

func clearScenarioTables(d *db.DB, scenarioID string, cfg config.Config) error {
	names := make([]string, 0, len(cfg.Transports))
	for _, t := range cfg.Transports {
		names = append(names, t.Name)
	}
	// Also clear synthetic names used when config has no transports yet.
	names = append(names, "local-serial", "rf-serial", "mqtt-uplink", "mqtt-bridge")
	uniq := map[string]struct{}{}
	var list []string
	for _, n := range names {
		if n == "" {
			continue
		}
		if _, ok := uniq[n]; ok {
			continue
		}
		uniq[n] = struct{}{}
		list = append(list, n)
	}
	inClause := "'" + strings.Join(list, "','") + "'"
	stmts := []string{
		fmt.Sprintf(`DELETE FROM messages WHERE transport_name IN (%s);`, inClause),
		`DELETE FROM transport_alerts WHERE id LIKE 'demo-%';`,
		`DELETE FROM incidents WHERE id LIKE 'demo-%';`,
		`DELETE FROM dead_letters WHERE details_json LIKE '%"mel_demo_scenario"%';`,
		fmt.Sprintf(`DELETE FROM transport_anomaly_snapshots WHERE transport_name IN (%s) AND reason='%s';`, inClause, escSQL(transport.ReasonObservationDropped)),
	}
	for _, s := range stmts {
		if err := d.Exec(s); err != nil {
			return err
		}
	}
	// Nodes: remove demo node nums for this scenario
	sc := ScenarioByID(scenarioID)
	if sc != nil {
		for _, n := range sc.Nodes {
			if err := d.Exec(fmt.Sprintf(`DELETE FROM nodes WHERE node_num=%d;`, n.NodeNum)); err != nil {
				return err
			}
		}
	}
	return nil
}

func escSQL(s string) string { return strings.ReplaceAll(s, "'", "''") }
