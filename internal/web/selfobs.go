package web

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mel-project/mel/internal/models"
	"github.com/mel-project/mel/internal/selfobs"
)

// InternalHealthHandler returns internal component health status
func (s *Server) InternalHealthHandler(w http.ResponseWriter, r *http.Request) {
	registry := selfobs.GetGlobalRegistry()
	components := registry.GetAllComponents()
	
	result := make([]map[string]any, 0, len(components))
	for _, comp := range components {
		result = append(result, map[string]any{
			"name":          comp.Name,
			"health":        comp.Health,
			"last_success":  comp.LastSuccess.Format(time.RFC3339),
			"last_failure":  comp.LastFailure.Format(time.RFC3339),
			"error_count":   comp.ErrorCount,
			"success_count": comp.SuccessCount,
			"error_rate":    comp.ErrorRate(),
		})
	}
	
	writeJSON(w, http.StatusOK, map[string]any{
		"overall_health": registry.GetOverallHealth(),
		"components":    result,
	})
}

// FreshnessHandler returns freshness status for all components
func (s *Server) FreshnessHandler(w http.ResponseWriter, r *http.Request) {
	tracker := selfobs.GetGlobalFreshnessTracker()
	markers := tracker.GetAllMarkers()

	reports := make([]models.FreshnessReport, 0, len(markers))
	staleList := make([]string, 0)

	for _, marker := range markers {
		age := marker.Age()
		status := "fresh"
		if marker.IsStale() {
			status = "stale"
			staleList = append(staleList, marker.Component)
		}
		if marker.LastUpdate.IsZero() {
			status = "unknown"
		}

		reports = append(reports, models.FreshnessReport{
			Component:       marker.Component,
			LastUpdate:      marker.LastUpdate.Format(time.RFC3339),
			IntervalSeconds: int(marker.ExpectedInterval.Seconds()),
			StaleThreshold:  int(marker.StaleThreshold.Seconds()),
			Status:          status,
			AgeSeconds:      int(age.Seconds()),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"markers":          reports,
		"stale_components": staleList,
		"observed_at":      time.Now().UTC().Format(time.RFC3339),
	})
}

// SLOHandler returns SLO compliance status
func (s *Server) SLOHandler(w http.ResponseWriter, r *http.Request) {
	tracker := selfobs.GetGlobalSLOTracker()
	definitions := tracker.GetAllDefinitions()
	statuses := tracker.GetAllSLOStatuses()
	
	defMap := make(map[string]selfobs.SLODefinition)
	for _, def := range definitions {
		defMap[def.Name] = def
	}
	
	result := make([]map[string]any, 0, len(statuses))
	for _, status := range statuses {
		def := defMap[status.Name]
		result = append(result, map[string]any{
			"name":          status.Name,
			"description":  def.Description,
			"current_value": status.CurrentValue,
			"target":       status.Target,
			"status":       status.Status,
			"budget_used":  status.BudgetUsed,
			"unit":         def.Unit,
			"window":       def.Window.String(),
			"window_start": status.WindowStart.Format(time.RFC3339),
			"window_end":   status.WindowEnd.Format(time.RFC3339),
			"evaluated_at": status.EvaluatedAt.Format(time.RFC3339),
		})
	}
	
	writeJSON(w, http.StatusOK, map[string]any{
		"slos": result,
	})
}

// InternalMetricsHandler returns internal metrics snapshot
func (s *Server) InternalMetricsHandler(w http.ResponseWriter, r *http.Request) {
	snapshot := selfobs.GetMetricsSnapshot()
	
	// Convert to JSON-friendly format using json.Marshal
	data, err := json.Marshal(map[string]any{
		"timestamp":         snapshot.Timestamp.Format(time.RFC3339),
		"pipeline_latency": snapshot.PipelineLatency,
		"worker_heartbeats": snapshot.WorkerHeartbeats,
		"queue_depths":     snapshot.QueueDepths,
		"error_rates":       snapshot.ErrorRates,
		"resource_usage": map[string]any{
			"memory_used_bytes": snapshot.ResourceUsage.MemoryUsed,
			"goroutines":       snapshot.ResourceUsage.Goroutines,
			"num_gc":           snapshot.ResourceUsage.NumGC,
		},
		"operation_counts": snapshot.OperationCounts,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// RegisterSelfObsRoutes registers the self-observability API routes
func RegisterSelfObsRoutes(mux *http.ServeMux, server *Server) {
	mux.HandleFunc("/api/v1/health/internal", server.requireMethod(server.InternalHealthHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/health/freshness", server.requireMethod(server.FreshnessHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/health/slo", server.requireMethod(server.SLOHandler, http.MethodGet, http.MethodHead))
	mux.HandleFunc("/api/v1/metrics/internal", server.requireMethod(server.InternalMetricsHandler, http.MethodGet, http.MethodHead))
}
