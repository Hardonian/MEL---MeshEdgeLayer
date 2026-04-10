package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/security"
)

func workspaceContext(r *http.Request) (workspaceID string, actorID string, ok bool) {
	workspaceID = strings.TrimSpace(r.Header.Get("X-MEL-Workspace-ID"))
	if workspaceID == "" {
		return "", "", false
	}
	id, hasID := security.GetIdentity(r.Context())
	if !hasID {
		return "", "", false
	}
	return workspaceID, id.ActorID, true
}

func (s *Server) fleetDevicesUpsertHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, db.ErrDatabaseUnavailable, "")
		return
	}
	workspaceID, actorID, ok := workspaceContext(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace context required", "send X-MEL-Workspace-ID and valid identity")
		return
	}
	var req db.DeviceIdentityInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	req.WorkspaceID = workspaceID
	req.ActorID = actorID
	conflict, err := s.db.UpsertDeviceIdentity(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "device identity update failed", err.Error())
		return
	}
	if conflict != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"status": "conflict", "conflict": conflict})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "device_id": req.DeviceID})
}

func (s *Server) fleetRolloutCreateHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, db.ErrDatabaseUnavailable, "")
		return
	}
	workspaceID, actorID, ok := workspaceContext(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace context required", "send X-MEL-Workspace-ID and valid identity")
		return
	}
	var req struct {
		JobID        string                  `json:"job_id"`
		TemplateID   string                  `json:"template_id"`
		Action       string                  `json:"action"`
		Scope        string                  `json:"scope"`
		ScheduledFor string                  `json:"scheduled_for"`
		Targets      []db.RolloutTargetInput `json:"targets"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := s.db.CreateRolloutJob(workspaceID, actorID, req.JobID, req.TemplateID, req.Action, req.Scope, req.ScheduledFor, req.Targets); err != nil {
		writeError(w, http.StatusBadRequest, "could not create rollout", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "created", "job_id": req.JobID, "targets": len(req.Targets)})
}

func (s *Server) fleetRolloutTargetHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, db.ErrDatabaseUnavailable, "")
		return
	}
	workspaceID, actorID, ok := workspaceContext(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace context required", "send X-MEL-Workspace-ID and valid identity")
		return
	}
	var req struct {
		TargetID       string `json:"target_id"`
		State          string `json:"state"`
		FailureReason  string `json:"failure_reason"`
		IncrementRetry bool   `json:"increment_retry"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := s.db.UpdateRolloutTargetState(workspaceID, actorID, req.TargetID, req.State, req.FailureReason, req.IncrementRetry); err != nil {
		writeError(w, http.StatusBadRequest, "could not update rollout target", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "target_id": req.TargetID, "state": req.State})
}

func (s *Server) fleetDashboardHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, db.ErrDatabaseUnavailable, "")
		return
	}
	workspaceID, actorID, ok := workspaceContext(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace context required", "send X-MEL-Workspace-ID and valid identity")
		return
	}
	devices, err := s.db.ListFleetDashboard(workspaceID, actorID, parseLimit(r.URL.Query().Get("limit"), 100), parseOffset(r.URL.Query().Get("offset"), 0))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not load fleet dashboard", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": devices, "count": len(devices), "degraded": false})
}

func (s *Server) fleetAlertsHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeError(w, http.StatusServiceUnavailable, db.ErrDatabaseUnavailable, "")
		return
	}
	workspaceID, actorID, ok := workspaceContext(r)
	if !ok {
		writeError(w, http.StatusForbidden, "workspace context required", "send X-MEL-Workspace-ID and valid identity")
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			AlertID  string `json:"alert_id"`
			DeviceID string `json:"device_id"`
			RuleKind string `json:"rule_kind"`
			Severity string `json:"severity"`
			Title    string `json:"title"`
			Detail   string `json:"detail"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
			return
		}
		if err := s.db.TriggerAlert(workspaceID, actorID, req.AlertID, req.DeviceID, req.RuleKind, req.Severity, req.Title, req.Detail); err != nil {
			writeError(w, http.StatusBadRequest, "could not trigger alert", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"status": "triggered", "alert_id": req.AlertID})
		return
	}
	var req struct {
		AlertID string `json:"alert_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := s.db.AcknowledgeAlert(workspaceID, actorID, req.AlertID); err != nil {
		writeError(w, http.StatusBadRequest, "could not acknowledge alert", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "acknowledged", "alert_id": req.AlertID})
}

func parseLimit(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	if v > 500 {
		return 500
	}
	return v
}

func parseOffset(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return fallback
	}
	return v
}
