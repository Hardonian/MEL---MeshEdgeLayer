package db

import (
	"fmt"
	"strings"
)

// AuditChainReport is the result of verifying tamper-evident hashes on audit_logs rows.
type AuditChainReport struct {
	OK            bool   `json:"ok"`
	LegacyRows    int    `json:"legacy_rows"`
	VerifiedRows  int    `json:"verified_rows"`
	TotalRows     int    `json:"total_rows"`
	FirstBrokenID string `json:"first_broken_id,omitempty"`
	Error         string `json:"error,omitempty"`
	HeadChainHash string `json:"head_chain_hash,omitempty"`
}

// VerifyAuditLogChain recomputes content_hash and chain_hash for each chained row.
// Rows created before migration 0019 have empty chain_hash and are counted as legacy only
// if no chained row appears later (otherwise reported as error).
func (d *DB) VerifyAuditLogChain() (AuditChainReport, error) {
	var rep AuditChainReport
	rows, err := d.QueryRows(`SELECT id, category, level, message, COALESCE(details_json,'') AS details_json, created_at,
		COALESCE(chain_prev_hash,'') AS chain_prev_hash, COALESCE(content_hash,'') AS content_hash, COALESCE(chain_hash,'') AS chain_hash
		FROM audit_logs ORDER BY id ASC;`)
	if err != nil {
		return rep, err
	}
	rep.TotalRows = len(rows)
	expectedPrev := ""
	chainStarted := false
	for _, row := range rows {
		id := asString(row["id"])
		cat := asString(row["category"])
		lvl := asString(row["level"])
		msg := asString(row["message"])
		details := asString(row["details_json"])
		created := asString(row["created_at"])
		prevH := asString(row["chain_prev_hash"])
		contentH := asString(row["content_hash"])
		chainH := asString(row["chain_hash"])

		if chainH == "" {
			if chainStarted {
				rep.Error = fmt.Sprintf("audit chain gap: row id=%s has no chain_hash after chained entries", id)
				rep.FirstBrokenID = id
				return rep, nil
			}
			rep.LegacyRows++
			continue
		}

		chainStarted = true
		canon := auditLogContentCanonical(id, cat, lvl, msg, details, created)
		wantContent := sha256Hex([]byte(canon))
		if wantContent != contentH {
			rep.Error = fmt.Sprintf("content_hash mismatch at id=%s", id)
			rep.FirstBrokenID = id
			return rep, nil
		}
		if prevH != expectedPrev {
			rep.Error = fmt.Sprintf("chain_prev_hash mismatch at id=%s (expected %q, got %q)", id, expectedPrev, prevH)
			rep.FirstBrokenID = id
			return rep, nil
		}
		var chainInput string
		if expectedPrev == "" {
			chainInput = wantContent
		} else {
			chainInput = expectedPrev + "\n" + wantContent
		}
		wantChain := sha256Hex([]byte(chainInput))
		if wantChain != chainH {
			rep.Error = fmt.Sprintf("chain_hash mismatch at id=%s", id)
			rep.FirstBrokenID = id
			return rep, nil
		}
		expectedPrev = chainH
		rep.VerifiedRows++
	}
	rep.HeadChainHash = strings.TrimSpace(expectedPrev)
	rep.OK = rep.TotalRows == 0 || rep.VerifiedRows > 0 || rep.LegacyRows == rep.TotalRows
	if rep.TotalRows > 0 && rep.VerifiedRows == 0 && rep.LegacyRows < rep.TotalRows {
		rep.OK = false
		rep.Error = "inconsistent audit log chain state"
	}
	return rep, nil
}
