package web

import (
	"net/http"
	"time"

	"github.com/mel-project/mel/internal/diagnostics"
)

func (s *Server) diagnosticsHandler(w http.ResponseWriter, r *http.Request) {
	rep := diagnostics.RunAllChecks(s.cfg, s.db, s.transportHealth(), nil, time.Now().UTC())
	writeJSON(w, http.StatusOK, map[string]any{
		"generated_at": rep.GeneratedAt,
		"summary":      rep.Summary,
		"findings":     rep.Diagnostics,
	})
}
