package web

import (
	"net/http"

	"github.com/mel-project/mel/internal/diagnostics"
)

func (s *Server) diagnosticsHandler(w http.ResponseWriter, r *http.Request) {
	findings := diagnostics.Run(s.cfg, s.db)
	writeJSON(w, http.StatusOK, findings)
}
