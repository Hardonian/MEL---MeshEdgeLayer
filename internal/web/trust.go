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

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/security"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

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

type operatorControlQueueRequest struct {
	ActionType      string  `json:"action_type"`
	TargetTransport string  `json:"target_transport,omitempty"`
	TargetSegment   string  `json:"target_segment,omitempty"`
	TargetNode      string  `json:"target_node,omitempty"`
	Reason          string  `json:"reason"`
	Confidence      float64 `json:"confidence,omitempty"`
	IncidentID      string  `json:"incident_id,omitempty"`
}

func (s *Server) operatorControlQueueHandler(w http.ResponseWriter, r *http.Request) {
	if s.queueOperatorControl == nil {
		writeError(w, http.StatusServiceUnavailable, "operator action queue not available", "service hooks not wired")
		return
	}
	var body operatorControlQueueRequest
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	actor := s.actorFromTrustContext(r)
	id, err := s.queueOperatorControl(actor, body.ActionType, body.TargetTransport, body.TargetSegment, body.TargetNode, body.Reason, body.Confidence, body.IncidentID)
	if err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			code = http.StatusNotFound
		} else if strings.Contains(err.Error(), "queue full") {
			code = http.StatusServiceUnavailable
		}
		writeError(w, code, "could not queue control action", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "queued", "action_id": id})
}

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
	const prefixControl = "/api/v1/control/actions/"
	const prefixAlias = "/api/v1/actions/"
	var actionID, subPath string
	switch {
	case strings.HasPrefix(r.URL.Path, prefixControl):
		actionID = idFromPath(r.URL.Path, prefixControl)
		subPath = subPathAfterID(r.URL.Path, prefixControl)
	case strings.HasPrefix(r.URL.Path, prefixAlias):
		actionID = idFromPath(r.URL.Path, prefixAlias)
		subPath = subPathAfterID(r.URL.Path, prefixAlias)
	default:
		writeError(w, http.StatusBadRequest, "invalid actions path", "")
		return
	}

	if actionID == "" {
		writeError(w, http.StatusBadRequest, "action id required", "")
		return
	}

	switch subPath {
	case "inspect":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		security.RequireAny([]security.Capability{security.CapReadActions, security.CapReadStatus}, func(w http.ResponseWriter, r *http.Request) {
			s.inspectActionHandler(w, r, actionID)
		})(w, r)
	case "approve":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		security.Require(security.CapApproveControlAction, func(w http.ResponseWriter, r *http.Request) {
			s.approveActionHandler(w, r, actionID)
		})(w, r)
	case "reject":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		security.Require(security.CapRejectControlAction, func(w http.ResponseWriter, r *http.Request) {
			s.rejectActionHandler(w, r, actionID)
		})(w, r)
	default:
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
	var body models.ApproveActionRequest
	_ = decodeBody(r, &body)
	actor := s.actorFromTrustContext(r)

	resp, err := s.approveAction(actionID, actor, body.Note, body.BreakGlassSodAck, body.BreakGlassSodReason)
	if err != nil {
		code := http.StatusInternalServerError
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			code = http.StatusNotFound
		} else if strings.Contains(msg, "break_glass_sod_reason is required") {
			code = http.StatusUnprocessableEntity
		} else if strings.Contains(msg, "not pending approval") || strings.Contains(msg, "expired") {
			code = http.StatusConflict
		} else if strings.Contains(msg, "separation of duties") {
			code = http.StatusForbidden
		}
		writeError(w, code, "could not approve action", msg)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) rejectActionHandler(w http.ResponseWriter, r *http.Request, actionID string) {
	if s.rejectAction == nil {
		writeError(w, http.StatusServiceUnavailable, "reject not available", "trust hooks not wired")
		return
	}
	var body models.RejectActionRequest
	_ = decodeBody(r, &body)
	actor := s.actorFromTrustContext(r)

	resp, err := s.rejectAction(actionID, actor, body.Note, body.BreakGlassSodAck, body.BreakGlassSodReason)
	if err != nil {
		code := http.StatusInternalServerError
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			code = http.StatusNotFound
		} else if strings.Contains(msg, "break_glass_sod_reason is required") {
			code = http.StatusUnprocessableEntity
		} else if strings.Contains(msg, "not pending approval") {
			code = http.StatusConflict
		} else if strings.Contains(msg, "separation of duties") {
			code = http.StatusForbidden
		}
		writeError(w, code, "could not reject action", msg)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ─── Freeze ───────────────────────────────────────────────────────────────────

func (s *Server) freezeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		security.RequireAny([]security.Capability{security.CapReadActions, security.CapReadStatus}, func(w http.ResponseWriter, r *http.Request) {
			s.listFreezesHandler(w, r)
		})(w, r)
		return
	}
	security.Require(security.CapExecuteAction, func(w http.ResponseWriter, r *http.Request) {
		s.createFreezeHandler(w, r)
	})(w, r)
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
	actor := s.actorFromTrustContext(r)
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
	security.Require(security.CapExecuteAction, func(w http.ResponseWriter, r *http.Request) {
		s.freezeItemHandlerInner(w, r)
	})(w, r)
}

func (s *Server) freezeItemHandlerInner(w http.ResponseWriter, r *http.Request) {
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
	actor := s.actorFromTrustContext(r)
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
		security.RequireAny([]security.Capability{security.CapReadActions, security.CapReadStatus}, func(w http.ResponseWriter, r *http.Request) {
			s.listMaintenanceHandler(w, r)
		})(w, r)
		return
	}
	security.Require(security.CapExecuteAction, func(w http.ResponseWriter, r *http.Request) {
		s.createMaintenanceHandler(w, r)
	})(w, r)
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
	actor := s.actorFromTrustContext(r)
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
	security.Require(security.CapExecuteAction, func(w http.ResponseWriter, r *http.Request) {
		s.maintenanceItemHandlerInner(w, r)
	})(w, r)
}

func (s *Server) maintenanceItemHandlerInner(w http.ResponseWriter, r *http.Request) {
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
	actor := s.actorFromTrustContext(r)
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
	eventType := strings.TrimSpace(q.Get("event_type"))
	scopePosture := strings.TrimSpace(q.Get("scope_posture"))

	if s.timeline == nil {
		writeJSON(w, http.StatusOK, map[string]any{"events": []any{}, "note": "trust hooks not wired"})
		return
	}
	events, err := s.timeline(start, end, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load timeline", err.Error())
		return
	}
	// Post-query filters for event_type and scope_posture (parity with CLI --type/--scope).
	// Post-query because the timeline is a UNION ALL across disparate tables.
	if eventType != "" || scopePosture != "" {
		filtered := make([]db.TimelineEvent, 0, len(events))
		for _, ev := range events {
			if eventType != "" && ev.EventType != eventType {
				continue
			}
			if scopePosture != "" && ev.ScopePosture != scopePosture {
				continue
			}
			filtered = append(filtered, ev)
		}
		events = filtered
	}
	result := map[string]any{
		"events":                events,
		"count":                 len(events),
		"start":                 start,
		"end":                   end,
		"ordering_posture_note": "Timeline order is instance-local (this database). remote_evidence_import rows include validation and provenance in details; no global total order.",
	}
	if eventType != "" {
		result["filter_event_type"] = eventType
	}
	if scopePosture != "" {
		result["filter_scope_posture"] = scopePosture
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) timelineItemHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, "timeline database unavailable", "")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/timeline/")
	id = strings.TrimSpace(id)
	if id == "" {
		writeError(w, http.StatusBadRequest, "event id required", "")
		return
	}
	event, ok, err := s.db.TimelineEventByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load timeline event", err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "timeline event not found", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"event":                event,
		"ordering_posture_note": "Timeline detail remains instance-local or import-local only. Imported remote events preserve timing/provenance posture but do not imply global total order.",
	})
}

// ─── Operator notes ───────────────────────────────────────────────────────────

func (s *Server) operatorNotesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		security.Require(security.CapExecuteAction, func(w http.ResponseWriter, r *http.Request) {
			s.createOperatorNoteHandler(w, r)
		})(w, r)
		return
	}
	security.RequireAny([]security.Capability{security.CapReadActions, security.CapReadStatus}, func(w http.ResponseWriter, r *http.Request) {
		s.listOperatorNotesHandler(w, r)
	})(w, r)
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
	actor := s.actorFromTrustContext(r)
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
