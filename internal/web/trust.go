package web

// trust.go — HTTP handlers for the control-plane trust and operability layer.
// Routes:
//   GET  /api/v1/control/operational-state      — current automation mode, freeze, maintenance backlog
//   GET  /api/v1/control/actions/{id}/inspect   — full evidence + decision bundle for an action
//   POST /api/v1/control/actions/{id}/approve   — approve a pending-approval action
//   POST /api/v1/control/actions/{id}/reject    — reject a pending-approval action
//   GET  /api/v1/control/freeze                 — list active freezes
//   POST /api/v1/control/freeze                 — create a new freeze
//   DELETE /api/v1/control/freeze/{id}          — clear a freeze
//   GET  /api/v1/control/maintenance            — list maintenance windows
//   POST /api/v1/control/maintenance            — create a maintenance window
//   DELETE /api/v1/control/maintenance/{id}     — cancel a maintenance window
//   GET  /api/v1/timeline                       — unified event timeline
//   GET  /api/v1/operator/notes                 — list notes by ref
//   POST /api/v1/operator/notes                 — add an operator note

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

func actorFromRequest(r *http.Request) string {
	// Prefer X-Operator-ID header; fall back to Basic Auth user; fall back to "system"
	if actor := r.Header.Get("X-Operator-ID"); actor != "" {
		return actor
	}
	if user, _, ok := r.BasicAuth(); ok && user != "" {
		return user
	}
	return "system"
}

func decodeBody(r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	return json.NewDecoder(r.Body).Decode(v)
}

func idFromPath(path, prefix string) string {
	trimmed := strings.TrimPrefix(path, prefix)
	// Strip any further path components (e.g., /approve, /reject)
	if idx := strings.Index(trimmed, "/"); idx >= 0 {
		return trimmed[:idx]
	}
	return trimmed
}

func subPathAfterID(path, prefix string) string {
	trimmed := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(trimmed, "/"); idx >= 0 {
		return trimmed[idx+1:]
	}
	return ""
}

// ─── Operational state ────────────────────────────────────────────────────────

func (s *Server) operationalStateHandler(w http.ResponseWriter, r *http.Request) {
	if s.operationalState == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"automation_mode":  s.cfg.Control.Mode,
			"freeze_count":     0,
			"approval_backlog": 0,
			"note":             "trust hooks not wired",
		})
		return
	}
	state, err := s.operationalState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load operational state", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// ─── Action sub-handler (inspect / approve / reject) ─────────────────────────

// controlActionSubHandler dispatches to the right handler based on URL sub-path:
//
//	/api/v1/control/actions/{id}/inspect
//	/api/v1/control/actions/{id}/approve
//	/api/v1/control/actions/{id}/reject
func (s *Server) controlActionSubHandler(w http.ResponseWriter, r *http.Request) {
	const prefix = "/api/v1/control/actions/"
	actionID := idFromPath(r.URL.Path, prefix)
	subPath := subPathAfterID(r.URL.Path, prefix)

	if actionID == "" {
		writeError(w, http.StatusBadRequest, "action id required", "")
		return
	}

	switch subPath {
	case "inspect":
		s.inspectActionHandler(w, r, actionID)
	case "approve":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		s.approveActionHandler(w, r, actionID)
	case "reject":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		s.rejectActionHandler(w, r, actionID)
	default:
		// Fall through to regular action lookup
		writeError(w, http.StatusNotFound, "unknown action sub-path: "+subPath, "")
	}
}

func (s *Server) inspectActionHandler(w http.ResponseWriter, _ *http.Request, actionID string) {
	if s.inspectAction == nil {
		writeError(w, http.StatusServiceUnavailable, "inspect not available", "trust hooks not wired")
		return
	}
	result, err := s.inspectAction(actionID)
	if err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		}
		writeError(w, code, "could not inspect action", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) approveActionHandler(w http.ResponseWriter, r *http.Request, actionID string) {
	if s.approveAction == nil {
		writeError(w, http.StatusServiceUnavailable, "approve not available", "trust hooks not wired")
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	_ = decodeBody(r, &body)
	actor := actorFromRequest(r)

	if err := s.approveAction(actionID, actor, body.Note); err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		} else if strings.Contains(err.Error(), "not pending approval") || strings.Contains(err.Error(), "expired") {
			code = http.StatusConflict
		}
		writeError(w, code, "could not approve action", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "approved",
		"action_id": actionID,
		"actor":     actor,
	})
}

func (s *Server) rejectActionHandler(w http.ResponseWriter, r *http.Request, actionID string) {
	if s.rejectAction == nil {
		writeError(w, http.StatusServiceUnavailable, "reject not available", "trust hooks not wired")
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	_ = decodeBody(r, &body)
	actor := actorFromRequest(r)

	if err := s.rejectAction(actionID, actor, body.Note); err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		} else if strings.Contains(err.Error(), "not pending approval") {
			code = http.StatusConflict
		}
		writeError(w, code, "could not reject action", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "rejected",
		"action_id": actionID,
		"actor":     actor,
	})
}

// ─── Freeze ───────────────────────────────────────────────────────────────────

func (s *Server) freezeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		s.listFreezesHandler(w, r)
		return
	}
	s.createFreezeHandler(w, r)
}

func (s *Server) listFreezesHandler(w http.ResponseWriter, _ *http.Request) {
	if s.db == nil {
		writeJSON(w, http.StatusOK, map[string]any{"freezes": []any{}})
		return
	}
	freezes, err := s.db.ActiveFreezes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list freezes", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"freezes": freezes, "count": len(freezes)})
}

func (s *Server) createFreezeHandler(w http.ResponseWriter, r *http.Request) {
	if s.createFreeze == nil {
		writeError(w, http.StatusServiceUnavailable, "freeze not available", "trust hooks not wired")
		return
	}
	var body struct {
		ScopeType  string `json:"scope_type"`
		ScopeValue string `json:"scope_value"`
		Reason     string `json:"reason"`
		ExpiresAt  string `json:"expires_at"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if body.Reason == "" {
		writeError(w, http.StatusBadRequest, "reason is required", "")
		return
	}
	if body.ScopeType == "" {
		body.ScopeType = "global"
	}
	actor := actorFromRequest(r)
	id, err := s.createFreeze(body.ScopeType, body.ScopeValue, body.Reason, actor, body.ExpiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create freeze", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":     "created",
		"freeze_id":  id,
		"scope_type": body.ScopeType,
		"reason":     body.Reason,
	})
}

func (s *Server) freezeItemHandler(w http.ResponseWriter, r *http.Request) {
	// DELETE /api/v1/control/freeze/{id}
	freezeID := strings.TrimPrefix(r.URL.Path, "/api/v1/control/freeze/")
	if freezeID == "" {
		writeError(w, http.StatusBadRequest, "freeze id required", "")
		return
	}
	if s.clearFreeze == nil {
		writeError(w, http.StatusServiceUnavailable, "clear freeze not available", "trust hooks not wired")
		return
	}
	actor := actorFromRequest(r)
	if err := s.clearFreeze(freezeID, actor); err != nil {
		writeError(w, http.StatusInternalServerError, "could not clear freeze", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "cleared",
		"freeze_id": freezeID,
		"actor":     actor,
	})
}

// ─── Maintenance windows ──────────────────────────────────────────────────────

func (s *Server) maintenanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		s.listMaintenanceHandler(w, r)
		return
	}
	s.createMaintenanceHandler(w, r)
}

func (s *Server) listMaintenanceHandler(w http.ResponseWriter, _ *http.Request) {
	if s.db == nil {
		writeJSON(w, http.StatusOK, map[string]any{"maintenance_windows": []any{}})
		return
	}
	windows, err := s.db.AllMaintenanceWindows(50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list maintenance windows", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"maintenance_windows": windows, "count": len(windows)})
}

func (s *Server) createMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	if s.createMaintenanceWindow == nil {
		writeError(w, http.StatusServiceUnavailable, "maintenance window not available", "trust hooks not wired")
		return
	}
	var body struct {
		Title      string `json:"title"`
		Reason     string `json:"reason"`
		ScopeType  string `json:"scope_type"`
		ScopeValue string `json:"scope_value"`
		StartsAt   string `json:"starts_at"`
		EndsAt     string `json:"ends_at"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if body.StartsAt == "" || body.EndsAt == "" {
		writeError(w, http.StatusBadRequest, "starts_at and ends_at are required", "")
		return
	}
	if body.ScopeType == "" {
		body.ScopeType = "global"
	}
	actor := actorFromRequest(r)
	id, err := s.createMaintenanceWindow(body.Title, body.Reason, body.ScopeType, body.ScopeValue, actor, body.StartsAt, body.EndsAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create maintenance window", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":    "created",
		"window_id": id,
		"starts_at": body.StartsAt,
		"ends_at":   body.EndsAt,
	})
}

func (s *Server) maintenanceItemHandler(w http.ResponseWriter, r *http.Request) {
	// DELETE /api/v1/control/maintenance/{id}
	windowID := strings.TrimPrefix(r.URL.Path, "/api/v1/control/maintenance/")
	if windowID == "" {
		writeError(w, http.StatusBadRequest, "window id required", "")
		return
	}
	if s.cancelMaintenanceWindow == nil {
		writeError(w, http.StatusServiceUnavailable, "cancel maintenance not available", "trust hooks not wired")
		return
	}
	actor := actorFromRequest(r)
	if err := s.cancelMaintenanceWindow(windowID, actor); err != nil {
		writeError(w, http.StatusInternalServerError, "could not cancel maintenance window", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "cancelled",
		"window_id": windowID,
		"actor":     actor,
	})
}

// ─── Timeline ─────────────────────────────────────────────────────────────────

func (s *Server) timelineHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	start := q.Get("start")
	end := q.Get("end")
	limit := parseIntOr(q.Get("limit"), 100)
	if limit > 500 {
		limit = 500
	}

	if s.timeline == nil {
		writeJSON(w, http.StatusOK, map[string]any{"events": []any{}, "note": "trust hooks not wired"})
		return
	}
	events, err := s.timeline(start, end, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load timeline", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"events": events,
		"count":  len(events),
		"start":  start,
		"end":    end,
	})
}

// ─── Operator notes ───────────────────────────────────────────────────────────

func (s *Server) operatorNotesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.createOperatorNoteHandler(w, r)
		return
	}
	s.listOperatorNotesHandler(w, r)
}

func (s *Server) listOperatorNotesHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	refType := q.Get("ref_type")
	refID := q.Get("ref_id")
	if refType == "" || refID == "" {
		writeError(w, http.StatusBadRequest, "ref_type and ref_id query params are required", "")
		return
	}
	if s.db == nil {
		writeJSON(w, http.StatusOK, map[string]any{"notes": []any{}})
		return
	}
	notes, err := s.db.OperatorNotesByRef(refType, refID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list notes", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": notes, "count": len(notes)})
}

func (s *Server) createOperatorNoteHandler(w http.ResponseWriter, r *http.Request) {
	if s.addOperatorNote == nil {
		writeError(w, http.StatusServiceUnavailable, "notes not available", "trust hooks not wired")
		return
	}
	var body struct {
		RefType string `json:"ref_type"`
		RefID   string `json:"ref_id"`
		Content string `json:"content"`
	}
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if body.RefType == "" || body.RefID == "" || body.Content == "" {
		writeError(w, http.StatusBadRequest, "ref_type, ref_id, and content are required", "")
		return
	}
	actor := actorFromRequest(r)
	id, err := s.addOperatorNote(body.RefType, body.RefID, actor, body.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create note", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":   "created",
		"note_id":  id,
		"ref_type": body.RefType,
		"ref_id":   body.RefID,
	})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func parseIntOr(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return def
	}
	return v
}
