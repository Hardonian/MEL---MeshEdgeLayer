package web

import (
	"net/http"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/fleet"
	"github.com/mel-project/mel/internal/logging"
)

// fleetTruthHandler returns canonical instance/site/fleet boundary truth for operators.
func (s *Server) fleetTruthHandler(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, logging.APIErrorResponse(
			logging.NewSafeError(db.ErrDatabaseUnavailable, nil, "database", false),
		))
		return
	}
	summary, err := fleet.BuildTruthSummary(s.cfg, s.db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, logging.APIErrorResponse(logging.ClassifyError(err)))
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
