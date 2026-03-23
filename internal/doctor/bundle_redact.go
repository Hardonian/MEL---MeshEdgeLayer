package doctor

import (
	"encoding/json"
)

// RedactForSupportBundle returns a copy of a doctor report safe to embed in a support zip.
// It omits the raw config file path and trims config_inspect to a fingerprint-only stub so
// broker endpoints and device paths are not duplicated beyond bundle.json (which uses privacy.RedactConfig).
func RedactForSupportBundle(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	b, err := json.Marshal(src)
	if err != nil {
		return map[string]any{"doctor_redact_error": err.Error()}
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{"doctor_redact_error": err.Error()}
	}
	delete(out, "config")
	if ci, ok := out["config_inspect"].(map[string]any); ok {
		fp, _ := ci["canonical_fingerprint"].(string)
		out["config_inspect"] = map[string]any{
			"canonical_fingerprint": fp,
			"note":                  "full config_inspect omitted from doctor.json; use redacted bundle.config or mel config inspect on a trusted host",
		}
	}
	out["bundle_note"] = "doctor.json is derived from the same checks as mel doctor with bundle-specific redaction; review before external sharing."
	return out
}
