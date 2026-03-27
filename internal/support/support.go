package support

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/doctor"
	"github.com/mel-project/mel/internal/privacy"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/upgrade"
)

type Bundle struct {
	GeneratedAt time.Time             `json:"generated_at"`
	Version     string                `json:"version"`
	Config      config.Config         `json:"config"`
	Diagnostics []diagnostics.Finding `json:"diagnostics"`
	// Operator evidence (offline-safe): status, control plane, incidents, upgrade posture.
	StatusSnapshot         *statuspkg.Snapshot             `json:"status_snapshot,omitempty"`
	StatusCollectError     string                          `json:"status_collect_error,omitempty"`
	Panel                  *statuspkg.Panel                `json:"operator_panel,omitempty"`
	UpgradeReadiness       *upgrade.UpgradeReadinessReport `json:"upgrade_readiness,omitempty"`
	ControlPlaneState      map[string]any                  `json:"control_plane_state,omitempty"`
	ControlPlaneStateErr   string                          `json:"control_plane_state_error,omitempty"`
	RecentControlActions   []db.ControlActionRecord        `json:"recent_control_actions,omitempty"`
	RecentControlDecisions []db.ControlDecisionRecord      `json:"recent_control_decisions,omitempty"`
	RecentIncidents        []map[string]any                `json:"recent_incidents,omitempty"`
	ActiveTransportAlerts  []db.TransportAlertRecord       `json:"active_transport_alerts,omitempty"`
	PrivacySummary         map[string]int                  `json:"privacy_summary,omitempty"`
	DoctorJSON             map[string]any                  `json:"doctor_json,omitempty"`
	DoctorJSONNote         string                          `json:"doctor_json_note,omitempty"`
	Nodes                  []map[string]any                `json:"nodes"`
	Messages               []map[string]any                `json:"messages"`
	DeadLetters            []map[string]any                `json:"dead_letters"`
	AuditLogs              []map[string]any                `json:"audit_logs"`
}

// Create builds a support bundle. cfgPath is the operator config path on disk (used for doctor.json parity with mel doctor); pass empty if unknown.
// processStartedAt is optional: when non-zero (e.g. bundle from running mel serve), the status snapshot includes process PID and uptime.
func Create(cfg config.Config, d *db.DB, version string, cfgPath string, processStartedAt time.Time) (*Bundle, error) {
	diagnosticsRun := diagnostics.RunAllChecks(cfg, d, nil, nil, time.Now().UTC())

	nodes, err := d.QueryRows("SELECT node_num,node_id,long_name,short_name,last_seen,lat_redacted,lon_redacted,altitude FROM nodes ORDER BY node_num;")
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}
	messages, err := d.QueryRows("SELECT transport_name,packet_id,channel_id,gateway_id,from_node,to_node,portnum,payload_text,payload_json,rx_time FROM messages ORDER BY id DESC LIMIT 250;")
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	deadLetters, err := d.QueryRows("SELECT transport_name,transport_type,topic,reason,payload_hex,details_json,created_at FROM dead_letters ORDER BY id DESC LIMIT 250;")
	if err != nil {
		return nil, fmt.Errorf("failed to get dead letters: %w", err)
	}
	auditLogs, err := d.QueryRows("SELECT category,level,message,details_json,created_at FROM audit_logs ORDER BY id DESC LIMIT 250;")
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}

	var pt *time.Time
	if !processStartedAt.IsZero() {
		t := processStartedAt.UTC()
		pt = &t
	}
	snap, serr := statuspkg.Collect(cfg, d, nil, pt, cfgPath)
	var panel *statuspkg.Panel
	var snapPtr *statuspkg.Snapshot
	statusErrStr := ""
	if serr != nil {
		statusErrStr = serr.Error()
	} else {
		snapCopy := snap
		snapPtr = &snapCopy
		p := statuspkg.BuildPanel(snap)
		panel = &p
	}

	trustState, terr := d.ControlPlaneStateSnapshot(time.Now().UTC())
	controlPlaneErr := ""
	if terr != nil {
		trustState = nil
		controlPlaneErr = terr.Error()
	}

	actions, _ := d.ControlActions("", "", "", "", "", 50, 0)
	decisions, _ := d.ControlDecisions("", "", "", "", 50, 0)
	incidents, _ := d.RecentIncidents(25)
	incMaps := make([]map[string]any, 0, len(incidents))
	for _, inc := range incidents {
		b, _ := json.Marshal(inc)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		incMaps = append(incMaps, m)
	}
	alerts, _ := d.TransportAlerts(true)

	var doctorForBundle map[string]any
	doctorNote := "Structured mel doctor payload (redacted for bundle export). Same checks as CLI; review before sharing externally."
	if p := strings.TrimSpace(cfgPath); p != "" {
		doctorRaw, _ := doctor.Run(cfg, p)
		doctorForBundle = doctor.RedactForSupportBundle(doctorRaw)
	} else {
		doctorNote = "doctor.json omitted: config file path was not provided to the bundle generator."
	}

	bundle := &Bundle{
		GeneratedAt:            time.Now().UTC(),
		Version:                version,
		Config:                 privacy.RedactConfig(cfg),
		Diagnostics:            diagnosticsRun.Diagnostics,
		StatusSnapshot:         snapPtr,
		StatusCollectError:     statusErrStr,
		Panel:                  panel,
		UpgradeReadiness:       upgrade.RunUpgradeChecks(cfg, d),
		ControlPlaneState:      trustState,
		ControlPlaneStateErr:   controlPlaneErr,
		RecentControlActions:   actions,
		RecentControlDecisions: decisions,
		RecentIncidents:        incMaps,
		ActiveTransportAlerts:  alerts,
		PrivacySummary:         privacy.Summary(privacy.Audit(cfg)),
		DoctorJSON:             doctorForBundle,
		DoctorJSONNote:         doctorNote,
		Nodes:                  nodes,
		Messages:               messages,
		DeadLetters:            deadLetters,
		AuditLogs:              auditLogs,
	}

	if cfg.Privacy.RedactExports {
		bundle.Messages = redactMessages(messages)
	}

	return bundle, nil
}

func (b *Bundle) ToZip() ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	files := map[string]any{
		"bundle.json": b,
	}
	if b.DoctorJSON != nil {
		files["doctor.json"] = b.DoctorJSON
	}

	for name, content := range files {
		f, err := zipWriter.Create(name)
		if err != nil {
			return nil, err
		}
		jsonContent, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			return nil, err
		}
		_, err = f.Write(jsonContent)
		if err != nil {
			return nil, err
		}
	}

	err := zipWriter.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func redactMessages(rows []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		cloned := map[string]any{}
		for k, v := range row {
			cloned[k] = v
		}
		cloned["payload_text"] = "[redacted]"
		cloned["payload_json"] = "[redacted]"
		out = append(out, cloned)
	}
	return out
}
