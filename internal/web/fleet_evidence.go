package web

import (
	"io"
	"net/http"
	"strings"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/fleet"
	"github.com/mel-project/mel/internal/security"
)

// SetFleetImport wires offline remote evidence import/list (instance-local; not live federation).
func (s *Server) SetFleetImport(
	importFn func(raw []byte, strictOrigin bool, actor string) (map[string]any, error),
	listFn func(limit int) ([]db.ImportedRemoteEvidenceRecord, error),
	getFn func(id string) (db.ImportedRemoteEvidenceRecord, bool, error),
) {
	s.importRemoteEvidence = importFn
	s.listImportedRemoteEvidence = listFn
	s.getImportedRemoteEvidence = getFn
}

func (s *Server) fleetRemoteEvidenceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		security.Require(security.CapExecuteAction, func(w http.ResponseWriter, r *http.Request) {
			s.fleetRemoteEvidenceImportHandler(w, r)
		})(w, r)
		return
	}
	security.RequireAny([]security.Capability{security.CapReadStatus, security.CapReadIncidents}, func(w http.ResponseWriter, r *http.Request) {
		s.fleetRemoteEvidenceListHandler(w, r)
	})(w, r)
}

func (s *Server) fleetRemoteEvidenceItemHandler(w http.ResponseWriter, r *http.Request) {
	security.RequireAny([]security.Capability{security.CapReadStatus, security.CapReadIncidents}, func(w http.ResponseWriter, r *http.Request) {
		s.fleetRemoteEvidenceGetHandler(w, r)
	})(w, r)
}

func (s *Server) fleetRemoteEvidenceImportHandler(w http.ResponseWriter, r *http.Request) {
	if s.importRemoteEvidence == nil {
		writeError(w, http.StatusServiceUnavailable, "remote evidence import not wired", "")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read body", err.Error())
		return
	}
	strict := strings.EqualFold(r.URL.Query().Get("strict_origin"), "1") ||
		strings.EqualFold(r.URL.Query().Get("strict_origin"), "true")
	actor := strings.TrimSpace(r.URL.Query().Get("actor"))
	if actor == "" {
		actor = s.actorFromTrustContext(r)
	}
	out, err := s.importRemoteEvidence(raw, strict, actor)
	if err != nil {
		writeError(w, http.StatusBadRequest, "import failed", err.Error())
		return
	}
	status := http.StatusOK
	if v, ok := out["status"].(string); ok {
		switch v {
		case "error":
			status = http.StatusBadRequest
		case "rejected":
			status = http.StatusUnprocessableEntity
		}
	}
	writeJSON(w, status, out)
}

func (s *Server) fleetRemoteEvidenceListHandler(w http.ResponseWriter, r *http.Request) {
	if s.listImportedRemoteEvidence == nil {
		writeJSON(w, http.StatusOK, map[string]any{"imports": []any{}, "note": "fleet import hooks not wired"})
		return
	}
	limit := parseIntOr(r.URL.Query().Get("limit"), 50)
	if limit > 500 {
		limit = 500
	}
	rows, err := s.listImportedRemoteEvidence(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list imports", err.Error())
		return
	}
	truth, err := fleet.BuildTruthSummary(s.cfg, s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not build fleet truth", err.Error())
		return
	}
	summaries, err := fleet.SummarizeImportedRemoteEvidenceRecords(truth, rows)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not inspect imports", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"imports":          rows,
		"summaries":        summaries,
		"count":            len(rows),
		"truth_posture":    truth,
		"inspection_notes": []string{"Imported remote evidence remains distinct from local observations. Related evidence analysis is explanatory only; rows are not silently merged."},
	})
}

func (s *Server) fleetRemoteEvidenceGetHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/fleet/remote-evidence/")
	id = strings.TrimSpace(id)
	if id == "" {
		writeError(w, http.StatusBadRequest, "import id required", "")
		return
	}
	if s.getImportedRemoteEvidence == nil {
		writeError(w, http.StatusServiceUnavailable, "remote evidence import not wired", "")
		return
	}
	rec, ok, err := s.getImportedRemoteEvidence(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load import", err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "import not found", "")
		return
	}
	rows, err := s.listImportedRemoteEvidence(500)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not inspect related imports", err.Error())
		return
	}
	truth, err := fleet.BuildTruthSummary(s.cfg, s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not build fleet truth", err.Error())
		return
	}
	inspection, err := fleet.InspectImportedRemoteEvidenceRecord(truth, rec, rows)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not inspect import", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"import":        rec,
		"inspection":    inspection,
		"truth_posture": truth,
	})
}

func (s *Server) fleetMergeExplainHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	keyA := q.Get("key_a")
	keyB := q.Get("key_b")
	sameObserver := strings.EqualFold(q.Get("same_observer"), "1") || strings.EqualFold(q.Get("same_observer"), "true")
	c := fleet.ClassifyMerge(keyA, keyB, sameObserver)
	writeJSON(w, http.StatusOK, map[string]any{
		"classification":   c,
		"explain_operator": fleet.ExplainMergeForOperator(c),
		"merge_inspection": fleet.MergeInspectionFromClassification(c),
		"physics_note":     "Structural dedupe only; does not prove RF coverage, flooding, or routing outcomes.",
	})
}
