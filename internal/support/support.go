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
	"github.com/mel-project/mel/internal/fleet"
	"github.com/mel-project/mel/internal/investigation"
	"github.com/mel-project/mel/internal/privacy"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
	"github.com/mel-project/mel/internal/upgrade"
)

type Bundle struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Version     string        `json:"version"`
	Config      config.Config `json:"config"`
	// FleetTruth duplicates status fleet boundary for offline triage (canonical with status.fleet_truth when present).
	FleetTruth fleet.FleetTruthSummary `json:"fleet_truth,omitempty"`
	// RemoteImportBatches preserves batch-level import audit containers.
	RemoteImportBatches []db.RemoteImportBatchRecord `json:"remote_import_batches,omitempty"`
	// RemoteImportBatchInspections expands batch-level payload/source/validation drilldown.
	RemoteImportBatchInspections []fleet.RemoteImportBatchInspection `json:"remote_import_batch_inspections,omitempty"`
	// ImportedRemoteEvidence is offline bundle import audit (not live federation); empty when table absent or none.
	ImportedRemoteEvidence            []db.ImportedRemoteEvidenceRecord  `json:"imported_remote_evidence,omitempty"`
	ImportedRemoteEvidenceInspections []fleet.ImportedEvidenceInspection `json:"imported_remote_evidence_inspections,omitempty"`
	// RemoteEvidenceExchange is a canonical offline batch export of imported evidence rows for re-import elsewhere.
	RemoteEvidenceExchange *fleet.RemoteEvidenceBatch `json:"remote_evidence_exchange,omitempty"`
	RemoteEvidenceTimeline []db.TimelineEvent         `json:"remote_evidence_timeline,omitempty"`
	// FullTimeline contains ALL timeline events (not just remote imports) for full operator investigation.
	FullTimeline []db.TimelineEvent    `json:"full_timeline,omitempty"`
	Diagnostics  []diagnostics.Finding `json:"diagnostics"`
	// Operator evidence (offline-safe): status, control plane, incidents, upgrade posture.
	StatusSnapshot           *statuspkg.Snapshot             `json:"status_snapshot,omitempty"`
	StatusCollectError       string                          `json:"status_collect_error,omitempty"`
	Panel                    *statuspkg.Panel                `json:"operator_panel,omitempty"`
	UpgradeReadiness         *upgrade.UpgradeReadinessReport `json:"upgrade_readiness,omitempty"`
	ControlPlaneState        map[string]any                  `json:"control_plane_state,omitempty"`
	ControlPlaneStateErr     string                          `json:"control_plane_state_error,omitempty"`
	RecentControlActions     []db.ControlActionRecord        `json:"recent_control_actions,omitempty"`
	RecentControlDecisions   []db.ControlDecisionRecord      `json:"recent_control_decisions,omitempty"`
	RecentIncidents          []map[string]any                `json:"recent_incidents,omitempty"`
	ActiveTransportAlerts    []db.TransportAlertRecord       `json:"active_transport_alerts,omitempty"`
	PrivacySummary           map[string]int                  `json:"privacy_summary,omitempty"`
	DoctorJSON               map[string]any                  `json:"doctor_json,omitempty"`
	DoctorJSONNote           string                          `json:"doctor_json_note,omitempty"`
	Nodes                    []map[string]any                `json:"nodes"`
	Messages                 []map[string]any                `json:"messages"`
	DeadLetters              []map[string]any                `json:"dead_letters"`
	AuditLogs                []map[string]any                `json:"audit_logs"`
	Investigation            *investigation.Summary          `json:"investigation,omitempty"`
	InvestigationCaseDetails []investigation.CaseDetail      `json:"investigation_case_details,omitempty"`
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
	var fleetTruth fleet.FleetTruthSummary
	if ft, err := fleet.BuildTruthSummary(cfg, d); err == nil {
		fleetTruth = ft
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

	imports, _ := d.ListImportedRemoteEvidence(100)
	importInspections, _ := fleet.InspectImportedRemoteEvidenceRecords(fleetTruth, imports)
	importBatches, _ := d.ListRemoteImportBatches(100)
	batchInspections := make([]fleet.RemoteImportBatchInspection, 0, len(importBatches))
	for _, batch := range importBatches {
		batchItems, err := d.ImportedRemoteEvidenceByBatch(batch.ID)
		if err != nil {
			continue
		}
		inspection, err := fleet.InspectRemoteImportBatchRecord(fleetTruth, batch, batchItems, imports)
		if err != nil {
			continue
		}
		batchInspections = append(batchInspections, inspection)
	}
	remoteEvidenceExchange, _ := buildRemoteEvidenceExchangePayload(imports)
	timeline, _ := d.TimelineEvents("", "", 500)
	remoteTimeline := make([]db.TimelineEvent, 0, len(timeline))
	for _, event := range timeline {
		if strings.HasPrefix(event.EventType, "remote_") {
			remoteTimeline = append(remoteTimeline, event)
		}
	}

	var doctorForBundle map[string]any
	doctorNote := "Structured mel doctor payload (redacted for bundle export). Same checks as CLI; review before sharing externally."
	if p := strings.TrimSpace(cfgPath); p != "" {
		doctorRaw, _ := doctor.Run(cfg, p)
		doctorForBundle = doctor.RedactForSupportBundle(doctorRaw)
	} else {
		doctorNote = "doctor.json omitted: config file path was not provided to the bundle generator."
	}

	bundle := &Bundle{
		GeneratedAt:                       time.Now().UTC(),
		Version:                           version,
		Config:                            privacy.RedactConfig(cfg),
		FleetTruth:                        fleetTruth,
		RemoteImportBatches:               importBatches,
		RemoteImportBatchInspections:      batchInspections,
		ImportedRemoteEvidence:            imports,
		ImportedRemoteEvidenceInspections: importInspections,
		RemoteEvidenceExchange:            remoteEvidenceExchange,
		RemoteEvidenceTimeline:            remoteTimeline,
		FullTimeline:                      timeline,
		Diagnostics:                       diagnosticsRun.Diagnostics,
		StatusSnapshot:                    snapPtr,
		StatusCollectError:                statusErrStr,
		Panel:                             panel,
		UpgradeReadiness:                  upgrade.RunUpgradeChecks(cfg, d),
		ControlPlaneState:                 trustState,
		ControlPlaneStateErr:              controlPlaneErr,
		RecentControlActions:              actions,
		RecentControlDecisions:            decisions,
		RecentIncidents:                   incMaps,
		ActiveTransportAlerts:             alerts,
		PrivacySummary:                    privacy.Summary(privacy.Audit(cfg)),
		DoctorJSON:                        doctorForBundle,
		DoctorJSONNote:                    doctorNote,
		Nodes:                             nodes,
		Messages:                          messages,
		DeadLetters:                       deadLetters,
		AuditLogs:                         auditLogs,
	}

	runtimeStates, _ := d.TransportRuntimeStatuses()
	var mockHealth []transport.Health
	for _, rs := range runtimeStates {
		mockHealth = append(mockHealth, transport.Health{
			Name:            rs.Name,
			Type:            rs.Type,
			Source:          rs.Source,
			State:           rs.State,
			LastAttemptAt:   rs.LastAttemptAt,
			LastError:       rs.LastError,
			LastConnectedAt: rs.LastConnectedAt,
			LastSuccessAt:   rs.LastSuccessAt,
			LastIngestAt:    rs.LastMessageAt,
			LastHeartbeatAt: rs.LastHeartbeatAt,
			LastFailureAt:   rs.LastFailureAt,
			TotalMessages:   rs.TotalMessages,
			PacketsDropped:  rs.PacketsDropped,
			FailureCount:    rs.FailureCount,
		})
	}
	summary := investigation.Derive(cfg, d, mockHealth, runtimeStates, time.Now().UTC())
	bundle.Investigation = &summary
	bundle.InvestigationCaseDetails = summary.CaseDetails()

	if cfg.Privacy.RedactExports {
		bundle.Messages = redactMessages(messages)
	}

	return bundle, nil
}

func (b *Bundle) ToZip() ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Write the manifest/README first for operator navigation.
	if f, err := zipWriter.Create("MANIFEST.md"); err == nil {
		_, _ = f.Write([]byte(bundleManifest(b)))
	}

	// Core bundle (monolith — kept for backward-compat with ingestion tooling).
	files := map[string]any{
		"bundle.json": b,
	}
	if b.DoctorJSON != nil {
		files["doctor.json"] = b.DoctorJSON
	}

	// Structured per-section exports so second-line operators can inspect
	// individual investigation surfaces without parsing the monolith.
	if len(b.FullTimeline) > 0 {
		files["timeline.json"] = map[string]any{
			"events": b.FullTimeline,
			"count":  len(b.FullTimeline),
			"note":   "Full unified operator timeline. Order is instance-local; remote_imported events include provenance in details. No global total order is implied.",
		}
	}
	if len(b.RecentControlActions) > 0 {
		files["control_actions.json"] = map[string]any{
			"actions": b.RecentControlActions,
			"count":   len(b.RecentControlActions),
		}
	}
	if len(b.RecentIncidents) > 0 {
		files["incidents.json"] = map[string]any{
			"incidents": b.RecentIncidents,
			"count":     len(b.RecentIncidents),
		}
	}
	if len(b.ImportedRemoteEvidence) > 0 {
		files["imported_evidence.json"] = map[string]any{
			"batches":           b.RemoteImportBatches,
			"batch_inspections": b.RemoteImportBatchInspections,
			"imports":           b.ImportedRemoteEvidence,
			"inspections":       b.ImportedRemoteEvidenceInspections,
			"count":             len(b.ImportedRemoteEvidence),
			"note":              "Offline bundle imports only (not live federation). Validation, provenance, batch source, timing posture, and merge posture are preserved per batch and per row. Authenticity is not cryptographically verified unless external verification is added by operator.",
		}
	}
	if b.RemoteEvidenceExchange != nil {
		files["remote_evidence_export.json"] = b.RemoteEvidenceExchange
	}
	if len(b.Diagnostics) > 0 {
		files["diagnostics.json"] = b.Diagnostics
	}
	if b.Investigation != nil {
		files["investigation.json"] = b.Investigation
		files["investigation_cases.json"] = map[string]any{
			"generated_at":  b.Investigation.GeneratedAt,
			"case_counts":   b.Investigation.CaseCounts,
			"count":         len(b.Investigation.Cases),
			"cases":         b.Investigation.Cases,
			"case_details":  b.InvestigationCaseDetails,
			"scope_posture": b.Investigation.ScopePosture,
			"note":          "Case detail includes linked raw events, case-evolution entries, timing posture, and bounded uncertainty/recommendation context. Related events contribute context to cases; they do not automatically prove causality.",
		}
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

func buildRemoteEvidenceExchangePayload(rows []db.ImportedRemoteEvidenceRecord) (*fleet.RemoteEvidenceBatch, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	items := make([]fleet.RemoteEvidenceBundle, 0, len(rows))
	for _, row := range rows {
		var bundle fleet.RemoteEvidenceBundle
		if err := json.Unmarshal(row.Bundle, &bundle); err != nil {
			return nil, fmt.Errorf("decode imported evidence bundle %s for export: %w", row.ID, err)
		}
		items = append(items, bundle)
	}
	return &fleet.RemoteEvidenceBatch{
		SchemaVersion:     fleet.RemoteEvidenceBatchSchemaVersion,
		Kind:              fleet.RemoteEvidenceBatchKind,
		ExportedAt:        time.Now().UTC().Format(time.RFC3339),
		CapabilityPosture: fleet.DefaultCapabilityPosture(),
		SourceContext: fleet.RemoteEvidenceImportSource{
			SourceType: "support_bundle_export",
			SourceName: "remote_evidence_export.json",
		},
		Items: items,
	}, nil
}

// bundleManifest generates an operator-readable README for the support bundle zip.
func bundleManifest(b *Bundle) string {
	var sb strings.Builder
	sb.WriteString("# MEL Support Bundle\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", b.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Version:   %s\n\n", b.Version))
	sb.WriteString("## Files in this bundle\n\n")
	sb.WriteString("| File | Purpose | How to use |\n")
	sb.WriteString("|------|---------|------------|\n")
	sb.WriteString("| MANIFEST.md | This file — index and interpretation guide | Read first |\n")
	sb.WriteString("| bundle.json | Full monolith bundle (all sections) | Machine-readable ingestion |\n")
	sb.WriteString("| doctor.json | `mel doctor` output (redacted) | Check host/config health findings |\n")
	sb.WriteString("| timeline.json | Full unified event timeline | Investigate what happened and in what order |\n")
	sb.WriteString("| control_actions.json | Recent control actions and decisions | Audit who did what and why |\n")
	sb.WriteString("| incidents.json | Recent incidents | Correlation and handoff context |\n")
	sb.WriteString("| imported_evidence.json | Offline remote evidence imports | Inspect batch/source provenance, validation, and merge posture |\n")
	sb.WriteString("| remote_evidence_export.json | Canonical offline export of imported evidence | Re-importable batch payload; still offline-only and authenticity-unverified by default |\n")
	sb.WriteString("| diagnostics.json | Diagnostics findings | Active issues and recommended steps |\n")
	sb.WriteString("| investigation.json | Canonical investigation summary | High-level decision support, cases, findings, evidence gaps, and physics boundaries |\n")
	sb.WriteString("| investigation_cases.json | Expanded investigation cases | Case list plus expanded case-detail drilldown, linked events, timing posture, and case evolution for support reconstruction |\n\n")
	sb.WriteString("## Interpretation notes\n\n")
	sb.WriteString("- **Timeline order is instance-local.** Events from imported remote evidence include timing and scope posture.\n")
	sb.WriteString("  `scope_posture=remote_imported` means the event came from another truth domain (offline).\n")
	sb.WriteString("  `timing_posture=local_ordered` means strict local order; other values indicate best-effort correlation.\n")
	sb.WriteString("- **Case timelines are reconstructable, not theatrical.** `investigation_cases.json` includes raw linked events and typed case-evolution entries.\n")
	sb.WriteString("  Evolution entries explain how the current case posture was shaped. They are bounded by retained evidence and may be derived or inferred; they are not a hidden incident story log.\n")
	sb.WriteString("- **Imported evidence is not live federation.** It is file-scoped, instance-local storage.\n")
	sb.WriteString("  Authenticity is not cryptographically verified unless operator adds external verification.\n")
	sb.WriteString("- **Absence of evidence ≠ evidence of absence.** Missing data may indicate a transport is disconnected, not that the mesh is healthy.\n")
	sb.WriteString("- **Repeated observations ≠ flooding proof.** Multiple gateways seeing the same packet is a symptom; it does not prove congestion.\n")
	sb.WriteString("- **Config is redacted for export.** Secrets, keys, and sensitive operator data are stripped.\n\n")
	sb.WriteString("## Quick investigation commands\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# What happened (chronological investigation):\n")
	sb.WriteString("mel timeline list --config <path>\n")
	sb.WriteString("mel timeline inspect <event-id> --config <path>\n\n")
	sb.WriteString("# Control action drilldown:\n")
	sb.WriteString("mel action inspect <action-id> --config <path>\n")
	sb.WriteString("mel trace <action-id> --config <path>\n\n")
	sb.WriteString("# Incident investigation:\n")
	sb.WriteString("mel incident list --config <path>\n")
	sb.WriteString("mel incident inspect <id> --config <path>\n\n")
	sb.WriteString("# Import audit:\n")
	sb.WriteString("mel fleet evidence list --config <path>\n")
	sb.WriteString("mel fleet evidence show <import-id> --config <path>\n\n")
	sb.WriteString("# System health:\n")
	sb.WriteString("mel doctor --config <path>\n")
	sb.WriteString("mel diagnostics --config <path>\n")
	sb.WriteString("mel investigate --config <path>\n")
	sb.WriteString("mel investigate cases --config <path>\n")
	sb.WriteString("mel investigate show <case-id> --config <path>\n")
	sb.WriteString("mel investigate timeline <case-id> --config <path>\n")
	sb.WriteString("mel health trust --config <path>\n")
	sb.WriteString("```\n")
	if b.StatusCollectError != "" {
		sb.WriteString(fmt.Sprintf("\n## ⚠ Status snapshot error\n\n%s\n", b.StatusCollectError))
	}
	if b.ControlPlaneStateErr != "" {
		sb.WriteString(fmt.Sprintf("\n## ⚠ Control plane state error\n\n%s\n", b.ControlPlaneStateErr))
	}
	return sb.String()
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
