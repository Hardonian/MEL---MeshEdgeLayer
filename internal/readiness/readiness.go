// Package readiness evaluates HTTP readiness from a status snapshot and config (no fake readiness).
package readiness

import (
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	statuspkg "github.com/mel-project/mel/internal/status"
	"github.com/mel-project/mel/internal/transport"
)

// StaleIngestThreshold is how old last_successful_ingest may be before we flag stale evidence.
const StaleIngestThreshold = 24 * time.Hour

// Component is a bounded, supportable readiness sub-check.
type Component struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

// Result is the machine-readable readiness evaluation (shared by /readyz and /api/v1/readyz).
type Result struct {
	Ready           bool     `json:"ready"`
	Status          string   `json:"status"`
	ReasonCodes     []string `json:"reason_codes"`
	Summary         string   `json:"summary"`
	CheckedAt       string   `json:"checked_at"`
	IngestReady     bool     `json:"ingest_ready"`
	ProcessReady    bool     `json:"process_ready"`
	OperatorState   string   `json:"operator_state"`
	MeshState       string   `json:"mesh_state,omitempty"`
	SnapshotAt      string   `json:"snapshot_generated_at,omitempty"`
	SchemaVersion   string   `json:"schema_version,omitempty"`
	StaleIngest     bool     `json:"stale_ingest_evidence"`
	Components      []Component `json:"components"`
}

// Evaluate returns readiness from an assembled snapshot. processReady is true when the HTTP handler is running.
func Evaluate(cfg config.Config, snap statuspkg.Snapshot, processReady bool, checkedAt time.Time) Result {
	panel := statuspkg.BuildPanel(snap)
	ingestReady := false
	for _, tr := range snap.Transports {
		if tr.EffectiveState == transport.StateIngesting {
			ingestReady = true
			break
		}
	}
	enabled := 0
	for _, t := range cfg.Transports {
		if t.Enabled {
			enabled++
		}
	}
	meshState := strings.TrimSpace(snap.Mesh.MeshHealth.State)
	stale := staleLastIngest(snap.LastSuccessfulIngest, checkedAt)

	reasons := make([]string, 0)
	if enabled == 0 {
		reasons = append(reasons, "TRANSPORT_IDLE")
	}
	if !ingestReady && enabled > 0 {
		reasons = append(reasons, "INGEST_NOT_PROVEN")
	}
	if panel.OperatorState == "degraded" && ingestReady {
		reasons = append(reasons, "DEGRADED_TRANSPORT")
	}
	if stale {
		reasons = append(reasons, "STALE_INGEST_EVIDENCE")
	}
	if meshState != "" && meshState != "healthy" && ingestReady {
		reasons = append(reasons, "MESH_"+strings.ToUpper(meshState))
	}

	ready := true
	if enabled > 0 && !ingestReady {
		ready = false
	}

	status := "ready"
	if !ready {
		status = "not_ready"
	}

	summary := panel.Summary
	if enabled == 0 {
		summary = "No transports are enabled; process is up and MEL is explicitly idle (not expecting ingest)."
	} else if !ingestReady {
		summary = "Subsystems are reachable but live ingest is not proven on any enabled transport; see /api/v1/status for transport truth."
	}
	if stale && ingestReady {
		summary += " Last persisted ingest timestamp is older than " + StaleIngestThreshold.String() + "; verify traffic is still reaching MEL."
	}

	components := []Component{
		{Name: "process", State: "ok", Detail: "HTTP handler responding"},
		{Name: "snapshot", State: "ok", Detail: "status snapshot assembled"},
	}
	if enabled == 0 {
		components = append(components, Component{Name: "transports", State: "idle", Detail: "no enabled transports"})
	} else if ingestReady {
		components = append(components, Component{Name: "ingest", State: "ok", Detail: "at least one transport in ingesting state"})
	} else {
		components = append(components, Component{Name: "ingest", State: "not_ready", Detail: "no transport in ingesting state"})
	}
	if meshState != "" {
		ms := "ok"
		switch meshState {
		case "degraded", "unstable":
			ms = "degraded"
		case "failed":
			ms = "degraded"
		}
		components = append(components, Component{Name: "mesh", State: ms, Detail: meshState})
	}

	return Result{
		Ready:         ready,
		Status:        status,
		ReasonCodes:   dedupeReasons(reasons),
		Summary:       summary,
		CheckedAt:     checkedAt.UTC().Format(time.RFC3339),
		IngestReady:   ingestReady,
		ProcessReady:  processReady,
		OperatorState: panel.OperatorState,
		MeshState:     meshState,
		SnapshotAt:    snap.GeneratedAt,
		SchemaVersion: snap.SchemaVersion,
		StaleIngest:   stale,
		Components:    components,
	}
}

func staleLastIngest(lastIngest string, now time.Time) bool {
	lastIngest = strings.TrimSpace(lastIngest)
	if lastIngest == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, lastIngest)
	if err != nil {
		return false
	}
	return now.Sub(t) > StaleIngestThreshold
}

func dedupeReasons(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, r := range in {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	return out
}
