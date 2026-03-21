package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mel-project/mel/internal/logging"
)

// FederationHandlers holds function hooks for federation API endpoints.
// These are set by the service layer when federation is enabled.
type FederationHandlers struct {
	// Status returns federation status
	Status func() (any, error)
	// Peers returns the peer list
	Peers func() (any, error)
	// Heartbeat processes an incoming heartbeat
	Heartbeat func(body []byte) error
	// SyncRequest handles sync requests from peers
	SyncRequest func(body []byte) (any, error)
	// SyncHealth returns sync health information
	SyncHealth func() (any, error)
	// ReplayExecute triggers a replay operation
	ReplayExecute func(body []byte) (any, error)
	// SnapshotList returns available snapshots
	SnapshotList func(limit int) (any, error)
	// SnapshotCreate triggers a new snapshot
	SnapshotCreate func() (any, error)
	// GlobalTopology returns the global topology view
	GlobalTopology func() (any, error)
	// RegionHealth returns region health info
	RegionHealth func(regionID string) (any, error)
	// EventLogStats returns event log statistics
	EventLogStats func() (any, error)
	// EventLogQuery queries the event log
	EventLogQuery func(body []byte) (any, error)
	// BackpressureStats returns backpressure statistics
	BackpressureStats func() (any, error)
	// DurabilityStatus returns storage durability status
	DurabilityStatus func() (any, error)
	// BackupCreate creates a backup
	BackupCreate func() (any, error)
	// BackupList lists available backups
	BackupList func() (any, error)
	// PushNotify handles push sync notifications from peers
	PushNotify func(body []byte) error
}

// SetFederationHandlers configures the federation API endpoints on the server.
func (s *Server) SetFederationHandlers(fh *FederationHandlers) {
	if fh == nil {
		return
	}
	s.federationHandlers = fh
}

// federationStatusHandler handles GET /api/v1/federation/status
func (s *Server) federationStatusHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.Status == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false,
			"status":  "federation not configured",
		})
		return
	}
	result, err := s.federationHandlers.Status()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("federation status unavailable", err, "federation", false),
		))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// federationPeersHandler handles GET /api/v1/federation/peers
func (s *Server) federationPeersHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.Peers == nil {
		writeJSON(w, http.StatusOK, map[string]any{"peers": []any{}})
		return
	}
	result, err := s.federationHandlers.Peers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("peer list unavailable", err, "federation", false),
		))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// federationHeartbeatHandler handles POST /api/v1/federation/heartbeat
func (s *Server) federationHeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.Heartbeat == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "federation not enabled"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := s.federationHandlers.Heartbeat(body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// federationSyncHandler handles POST /api/v1/federation/sync
func (s *Server) federationSyncHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.SyncRequest == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "federation not enabled"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	result, err := s.federationHandlers.SyncRequest(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// federationSyncHealthHandler handles GET /api/v1/federation/sync/health
func (s *Server) federationSyncHealthHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.SyncHealth == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "not configured"})
		return
	}
	result, err := s.federationHandlers.SyncHealth()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("sync health unavailable", err, "federation", false),
		))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// replayHandler handles POST /api/v1/kernel/replay
func (s *Server) replayHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.ReplayExecute == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "kernel not available"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	result, err := s.federationHandlers.ReplayExecute(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// snapshotListHandler handles GET /api/v1/kernel/snapshots
func (s *Server) snapshotListHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.SnapshotList == nil {
		writeJSON(w, http.StatusOK, map[string]any{"snapshots": []any{}})
		return
	}
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	result, err := s.federationHandlers.SnapshotList(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("snapshot list unavailable", err, "kernel", false),
		))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// snapshotCreateHandler handles POST /api/v1/kernel/snapshots
func (s *Server) snapshotCreateHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.SnapshotCreate == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "kernel not available"})
		return
	}
	result, err := s.federationHandlers.SnapshotCreate()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("snapshot creation failed", err, "kernel", false),
		))
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// globalTopologyHandler handles GET /api/v1/topology/global
func (s *Server) globalTopologyHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.GlobalTopology == nil {
		writeJSON(w, http.StatusOK, map[string]any{"regions": []any{}, "global_health": 1.0})
		return
	}
	result, err := s.federationHandlers.GlobalTopology()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("global topology unavailable", err, "topology", false),
		))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// regionHealthHandler handles GET /api/v1/topology/region/{region_id}
func (s *Server) regionHealthHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.RegionHealth == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "not configured"})
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/topology/region/"), "/")
	regionID := parts[0]
	if regionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "region_id required"})
		return
	}
	result, err := s.federationHandlers.RegionHealth(regionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// eventLogStatsHandler handles GET /api/v1/kernel/eventlog/stats
func (s *Server) eventLogStatsHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.EventLogStats == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "not available"})
		return
	}
	result, err := s.federationHandlers.EventLogStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("event log stats unavailable", err, "eventlog", false),
		))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// eventLogQueryHandler handles POST /api/v1/kernel/eventlog/query
func (s *Server) eventLogQueryHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.EventLogQuery == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "event log not available"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	result, err := s.federationHandlers.EventLogQuery(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// backpressureStatsHandler handles GET /api/v1/kernel/backpressure
func (s *Server) backpressureStatsHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.BackpressureStats == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "not available"})
		return
	}
	result, err := s.federationHandlers.BackpressureStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("backpressure stats unavailable", err, "kernel", false),
		))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// durabilityStatusHandler handles GET /api/v1/kernel/durability
func (s *Server) durabilityStatusHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.DurabilityStatus == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "not available"})
		return
	}
	result, err := s.federationHandlers.DurabilityStatus()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("durability status unavailable", err, "kernel", false),
		))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// backupCreateHandler handles POST /api/v1/kernel/backup
func (s *Server) backupCreateHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.BackupCreate == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "backup not available"})
		return
	}
	result, err := s.federationHandlers.BackupCreate()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("backup failed", err, "kernel", false),
		))
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// backupListHandler handles GET /api/v1/kernel/backups
func (s *Server) backupListHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.BackupList == nil {
		writeJSON(w, http.StatusOK, map[string]any{"backups": []any{}})
		return
	}
	result, err := s.federationHandlers.BackupList()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(
			logging.NewSafeError("backup list unavailable", err, "kernel", false),
		))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// snapshotSubHandler routes GET and POST for /api/v1/kernel/snapshots
func (s *Server) snapshotSubHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.snapshotListHandler(w, r)
	case http.MethodPost:
		s.snapshotCreateHandler(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// federationPushNotifyHandler handles POST /api/v1/federation/sync/notify
// This enables push-based sync: a peer sends a notification when it has new events.
func (s *Server) federationPushNotifyHandler(w http.ResponseWriter, r *http.Request) {
	if s.federationHandlers == nil || s.federationHandlers.PushNotify == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "federation not enabled"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := s.federationHandlers.PushNotify(body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
