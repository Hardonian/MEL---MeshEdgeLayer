package web

// runbook_review.go — HTTP handlers for the remediation-memory loop.
//
// Exposes:
//   GET  /api/v1/runbooks                       → list runbook entries
//   GET  /api/v1/runbooks/{id}                  → detail with recent applications
//   POST /api/v1/runbooks/{id}/promote          → lifecycle: proposed/reviewing → promoted
//   POST /api/v1/runbooks/{id}/deprecate        → lifecycle: → deprecated (reason required)
//   POST /api/v1/incidents/{id}/apply-runbook   → record operator applying runbook
//   GET  /api/v1/operator/worklist              → per-operator worklist
//   GET  /api/v1/operator/shift-handoff         → bounded-window shift handoff packet
//
// All reads use CapReadIncidents / CapReadStatus. All writes use CapIncidentUpdate.
// Writes are attributed to the authenticated actor via actorFromTrustContext.

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/security"
)

// runbookListHandler — GET /api/v1/runbooks.
// Query parameters: status, signature_key, fingerprint_hash, q, limit.
func (s *Server) runbookListHandler(w http.ResponseWriter, r *http.Request) {
	if s.listRunbookEntries == nil {
		writeError(w, http.StatusServiceUnavailable, "runbook review not available", "")
		return
	}
	q := r.URL.Query()
	limit := 50
	if v := strings.TrimSpace(q.Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	rows, err := s.listRunbookEntries(
		q.Get("status"),
		q.Get("signature_key"),
		q.Get("fingerprint_hash"),
		q.Get("q"),
		limit,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list runbooks", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"runbooks": rows,
		"count":    len(rows),
	})
}

// runbooksPathHandler — routes under /api/v1/runbooks/{id}[/...].
func (s *Server) runbooksPathHandler(w http.ResponseWriter, r *http.Request) {
	const prefix = "/api/v1/runbooks/"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) {
		writeError(w, http.StatusNotFound, "not found", "")
		return
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" {
		writeError(w, http.StatusBadRequest, "runbook id required", "")
		return
	}
	parts := strings.SplitN(rest, "/", 2)
	id := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	switch {
	case sub == "" && (r.Method == http.MethodGet || r.Method == http.MethodHead):
		security.RequireAny([]security.Capability{security.CapReadIncidents, security.CapReadStatus}, func(w http.ResponseWriter, r *http.Request) {
			s.runbookDetailHandler(w, r, id)
		})(w, r)
	case sub == "promote" && r.Method == http.MethodPost:
		security.Require(security.CapIncidentUpdate, func(w http.ResponseWriter, r *http.Request) {
			s.runbookPromoteHandler(w, r, id)
		})(w, r)
	case sub == "deprecate" && r.Method == http.MethodPost:
		security.Require(security.CapIncidentUpdate, func(w http.ResponseWriter, r *http.Request) {
			s.runbookDeprecateHandler(w, r, id)
		})(w, r)
	default:
		writeError(w, http.StatusNotFound, "unknown runbook path", "")
	}
}

func (s *Server) runbookDetailHandler(w http.ResponseWriter, r *http.Request, id string) {
	if s.getRunbookEntry == nil {
		writeError(w, http.StatusServiceUnavailable, "runbook detail not available", "")
		return
	}
	detail, ok, err := s.getRunbookEntry(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load runbook", err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "runbook not found", "")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) runbookPromoteHandler(w http.ResponseWriter, r *http.Request, id string) {
	if s.promoteRunbookEntry == nil {
		writeError(w, http.StatusServiceUnavailable, "runbook promote not available", "")
		return
	}
	var req models.PromoteRunbookRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
			return
		}
	}
	actor := s.actorFromTrustContext(r)
	if err := s.promoteRunbookEntry(id, actor, strings.TrimSpace(req.Note)); err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		} else if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			code = http.StatusBadRequest
		}
		writeError(w, code, "could not promote runbook", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "promoted",
		"runbook_id": id,
		"actor":      actor,
	})
}

func (s *Server) runbookDeprecateHandler(w http.ResponseWriter, r *http.Request, id string) {
	if s.deprecateRunbookEntry == nil {
		writeError(w, http.StatusServiceUnavailable, "runbook deprecate not available", "")
		return
	}
	var req models.DeprecateRunbookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	actor := s.actorFromTrustContext(r)
	if err := s.deprecateRunbookEntry(id, actor, strings.TrimSpace(req.Reason)); err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		} else if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			code = http.StatusBadRequest
		}
		writeError(w, code, "could not deprecate runbook", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "deprecated",
		"runbook_id": id,
		"actor":      actor,
	})
}

// incidentApplyRunbookHandler — POST /api/v1/incidents/{id}/apply-runbook.
// Routed from incidentsPathHandler to keep incident sub-routes co-located.
func (s *Server) incidentApplyRunbookHandler(w http.ResponseWriter, r *http.Request, incidentID string) {
	if s.applyRunbookToIncident == nil {
		writeError(w, http.StatusServiceUnavailable, "apply runbook not available", "")
		return
	}
	var req models.ApplyRunbookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	actor := s.actorFromTrustContext(r)
	rec, err := s.applyRunbookToIncident(incidentID, actor, req)
	if err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		} else if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			code = http.StatusBadRequest
		}
		writeError(w, code, "could not record runbook application", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "recorded",
		"incident_id": incidentID,
		"application": rec,
	})
}

// operatorWorklistHandler — GET /api/v1/operator/worklist. Actor is the caller.
func (s *Server) operatorWorklistHandler(w http.ResponseWriter, r *http.Request) {
	if s.buildOperatorWorklist == nil {
		writeError(w, http.StatusServiceUnavailable, "operator worklist not available", "")
		return
	}
	actor := s.actorFromTrustContext(r)
	// Allow overriding via ?actor= only when the caller has admin capability;
	// otherwise the worklist is always for the authenticated caller.
	if override := strings.TrimSpace(r.URL.Query().Get("actor")); override != "" {
		if id, ok := security.GetIdentity(r.Context()); ok && id.Can(security.CapAdminSystem) {
			actor = override
		}
	}
	packet, err := s.buildOperatorWorklist(actor)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not build worklist", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, packet)
}

// shiftHandoffHandler — GET /api/v1/operator/shift-handoff?window_hours=N.
func (s *Server) shiftHandoffHandler(w http.ResponseWriter, r *http.Request) {
	if s.buildShiftHandoff == nil {
		writeError(w, http.StatusServiceUnavailable, "shift handoff not available", "")
		return
	}
	actor := s.actorFromTrustContext(r)
	if override := strings.TrimSpace(r.URL.Query().Get("actor")); override != "" {
		if id, ok := security.GetIdentity(r.Context()); ok && id.Can(security.CapAdminSystem) {
			actor = override
		}
	}
	window := 0
	if v := strings.TrimSpace(r.URL.Query().Get("window_hours")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			window = n
		}
	}
	packet, err := s.buildShiftHandoff(actor, window)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not build shift handoff", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, packet)
}
