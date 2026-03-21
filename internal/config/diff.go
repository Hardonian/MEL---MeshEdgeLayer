package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DiffEntry describes one differing JSON path between two configs.
type DiffEntry struct {
	Path   string `json:"path"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
	Kind   string `json:"kind"` // changed | added | removed
}

// Diff compares two normalized configs as JSON object trees.
func Diff(a, b Config) []DiffEntry {
	ba, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	var ma, mb map[string]any
	_ = json.Unmarshal(ba, &ma)
	_ = json.Unmarshal(bb, &mb)
	return diffMaps("", ma, mb)
}

func diffMaps(prefix string, a, b map[string]any) []DiffEntry {
	var out []DiffEntry
	keys := map[string]struct{}{}
	for k := range a {
		keys[k] = struct{}{}
	}
	for k := range b {
		keys[k] = struct{}{}
	}
	for k := range keys {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		va, okA := a[k]
		vb, okB := b[k]
		switch {
		case okA && !okB:
			out = append(out, DiffEntry{Path: path, Before: va, Kind: "removed"})
		case !okA && okB:
			out = append(out, DiffEntry{Path: path, After: vb, Kind: "added"})
		case jsonEqual(va, vb):
			continue
		default:
			ama, aMap := va.(map[string]any)
			bmb, bMap := vb.(map[string]any)
			if aMap && bMap {
				out = append(out, diffMaps(path, ama, bmb)...)
				continue
			}
			out = append(out, DiffEntry{Path: path, Before: va, After: vb, Kind: "changed"})
		}
	}
	return out
}

func jsonEqual(a, b any) bool {
	ba, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(ba) == string(bb)
}

// FormatDiffText renders diff entries as lines for mel config diff --format text.
func FormatDiffText(entries []DiffEntry) string {
	var b strings.Builder
	for _, e := range entries {
		switch e.Kind {
		case "added":
			fmt.Fprintf(&b, "+ %s = %s\n", e.Path, trimJSON(e.After))
		case "removed":
			fmt.Fprintf(&b, "- %s = %s\n", e.Path, trimJSON(e.Before))
		default:
			fmt.Fprintf(&b, "~ %s: %s -> %s\n", e.Path, trimJSON(e.Before), trimJSON(e.After))
		}
	}
	return b.String()
}

func trimJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	s := string(b)
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}
