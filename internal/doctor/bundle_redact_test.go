package doctor

import "testing"

func TestRedactForSupportBundleStripsConfigPath(t *testing.T) {
	src := map[string]any{
		"doctor_version": "v2",
		"config":         "/secret/path/mel.json",
		"config_inspect": map[string]any{"canonical_fingerprint": "abc", "raw": "x"},
	}
	out := RedactForSupportBundle(src)
	if out["config"] != nil {
		t.Fatalf("config key should be removed, got %v", out["config"])
	}
	ci, ok := out["config_inspect"].(map[string]any)
	if !ok {
		t.Fatalf("config_inspect: %T", out["config_inspect"])
	}
	if ci["raw"] != nil {
		t.Fatal("expected raw stripped from config_inspect")
	}
	if ci["canonical_fingerprint"] != "abc" {
		t.Fatalf("fingerprint: %#v", ci["canonical_fingerprint"])
	}
}
