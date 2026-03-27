package web

import (
	"net/http"

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
