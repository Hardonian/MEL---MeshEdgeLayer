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
		id := strings.TrimSpace(strings.TrimPrefix(path, "cases/"))
		if id == "" {
			writeError(w, http.StatusBadRequest, "case id required", "")
			return
		}
		detail, ok := summary.CaseDetail(id)
		if !ok {
			writeError(w, http.StatusNotFound, "investigation case not found", "")
			return
		}
		writeJSON(w, http.StatusOK, detail)
	default:
		writeError(w, http.StatusNotFound, "investigation path not found", "")
	}
}
