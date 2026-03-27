package web

import (
	"net/http"
	"strings"

	"github.com/mel-project/mel/internal/logging"
)

func (s *Server) investigationsHandler(w http.ResponseWriter, r *http.Request) {
	if s.investigationSummary == nil {
		writeJSON(w, http.StatusNotImplemented, logging.APIErrorResponse(
			logging.NewSafeError("investigation substrate not wired in this server", nil, "not_implemented", false),
		))
		return
	}

	summary := s.investigationSummary()
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) investigationsPathHandler(w http.ResponseWriter, r *http.Request) {
	if s.investigationSummary == nil {
		writeJSON(w, http.StatusNotImplemented, logging.APIErrorResponse(
			logging.NewSafeError("investigation substrate not wired in this server", nil, "not_implemented", false),
		))
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/investigations/")
	path = strings.Trim(path, "/")
	if path == "" {
		s.investigationsHandler(w, r)
		return
	}

	summary := s.investigationSummary()
	switch {
	case path == "cases":
		writeJSON(w, http.StatusOK, map[string]any{
			"generated_at": summary.GeneratedAt,
			"case_counts":  summary.CaseCounts,
			"count":        len(summary.Cases),
			"cases":        summary.Cases,
		})
	case strings.HasPrefix(path, "cases/"):
		rest := strings.TrimSpace(strings.TrimPrefix(path, "cases/"))
		if rest == "" {
			writeError(w, http.StatusBadRequest, "case id required", "")
			return
		}
		caseID := rest
		subpath := ""
		if cut := strings.Index(rest, "/"); cut >= 0 {
			caseID = strings.TrimSpace(rest[:cut])
			subpath = strings.Trim(strings.TrimSpace(rest[cut+1:]), "/")
		}
		if caseID == "" {
			writeError(w, http.StatusBadRequest, "case id required", "")
			return
		}
		detail, ok := summary.CaseDetail(caseID)
		if !ok {
			writeError(w, http.StatusNotFound, "investigation case not found", "")
			return
		}
		if subpath == "" {
			writeJSON(w, http.StatusOK, detail)
			return
		}
		switch subpath {
		case "timeline":
			writeJSON(w, http.StatusOK, map[string]any{
				"case_id":            detail.Case.ID,
				"title":              detail.Case.Title,
				"timing":             detail.Case.Timing,
				"linked_event_count": len(detail.LinkedEvents),
				"evolution_count":    len(detail.Evolution),
				"linked_events":      detail.LinkedEvents,
				"evolution":          detail.Evolution,
			})
		default:
			writeError(w, http.StatusNotFound, "investigation path not found", "")
		}
	default:
		writeError(w, http.StatusNotFound, "investigation path not found", "")
	}
}
