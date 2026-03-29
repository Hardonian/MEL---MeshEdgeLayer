package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/auth"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/proofpack"
)

// AssembleProofpack builds an incident-scoped evidence bundle for audit/export.
// The returned map is the JSON-serializable proofpack structure.
func (a *App) AssembleProofpack(incidentID, actorID string) (map[string]any, error) {
	if a == nil || a.DB == nil {
		return nil, fmt.Errorf("service not available")
	}
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return nil, fmt.Errorf("incident id is required")
	}
	if strings.TrimSpace(actorID) == "" {
		actorID = "system"
	}

	src := proofpack.NewDBAdapter(a.DB)
	cfg := proofpack.DefaultConfig()
	cfg.ActorID = actorID
	cfg.InstanceID = a.instanceID()

	assembler := proofpack.NewAssembler(src, cfg)
	pack, err := assembler.Assemble(incidentID)
	if err != nil {
		return nil, err
	}

	// Serialize to map for JSON response (avoids double-encoding).
	raw, err := json.Marshal(pack)
	if err != nil {
		return nil, fmt.Errorf("could not serialize proofpack: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("could not parse proofpack: %w", err)
	}

	// Audit the proofpack export.
	_ = a.DB.InsertRBACAuditLog(auth.AuditEntry{
		ID:           newTrustID("aud"),
		ActorID:      auth.OperatorID(actorID),
		ActionClass:  auth.ActionExport,
		ActionDetail: "proofpack_export",
		ResourceType: "incident",
		ResourceID:   incidentID,
		Reason:       "proofpack assembled for export",
		Result:       auth.AuditResultSuccess,
		Timestamp:    time.Now().UTC(),
	})
	_ = a.DB.InsertTimelineEvent(db.TimelineEvent{
		EventID:    newTrustID("tl"),
		EventType:  "proofpack_export",
		Summary:    "proofpack exported for incident: " + incidentID,
		Severity:   "info",
		ActorID:    actorID,
		ResourceID: incidentID,
		Details: map[string]any{
			"incident_id":   incidentID,
			"action_count":  pack.Assembly.ActionCount,
			"timeline_count": pack.Assembly.TimelineCount,
			"gap_count":     pack.Assembly.EvidenceGapCount,
		},
	})

	return result, nil
}

// instanceID returns the MEL instance identifier from config or a fallback.
func (a *App) instanceID() string {
	if a.Cfg.Scope.SiteID != "" {
		return a.Cfg.Scope.SiteID
	}
	if a.Cfg.Scope.GatewayLabel != "" {
		return a.Cfg.Scope.GatewayLabel
	}
	return "mel-instance"
}
