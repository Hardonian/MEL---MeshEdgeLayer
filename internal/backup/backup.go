package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)

type Manifest struct {
	CreatedAt     string   `json:"created_at"`
	SchemaVersion string   `json:"schema_version"`
	DatabasePath  string   `json:"database_path"`
	ConfigPath    string   `json:"config_path"`
	Warnings      []string `json:"warnings,omitempty"`
}

type RestoreReport struct {
	Manifest    Manifest `json:"manifest"`
	Files       []string `json:"files"`
	Warnings    []string `json:"warnings,omitempty"`
	Actions     []string `json:"actions"`
	Valid       bool     `json:"valid"`
	CheckedAt   string   `json:"checked_at"`
	Destination string   `json:"destination,omitempty"`
}

func Create(cfg config.Config, cfgPath, outPath string) (Manifest, error) {
	manifest := Manifest{
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		DatabasePath: cfg.Storage.DatabasePath,
		ConfigPath:   cfgPath,
	}
	d, err := db.Open(cfg)
	if err != nil {
		return manifest, err
	}
	version, err := d.SchemaVersion()
	if err != nil {
		return manifest, err
	}
	manifest.SchemaVersion = version
	if outPath == "" {
		outPath = filepath.Join(cfg.Storage.DataDir, fmt.Sprintf("mel-backup-%s.tgz", time.Now().UTC().Format("20060102T150405Z")))
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return manifest, err
	}
	out, err := os.Create(outPath)
	if err != nil {
		return manifest, err
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	writeJSON := func(name string, v any, mode int64) error {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		b = append(b, '\n')
		hdr := &tar.Header{Name: name, Mode: mode, Size: int64(len(b)), ModTime: time.Now()}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err = tw.Write(b)
		return err
	}
	copyFile := func(src, dst string, mode int64) error {
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return err
		}
		hdr := &tar.Header{Name: dst, Mode: mode, Size: info.Size(), ModTime: info.ModTime()}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err = io.Copy(tw, f)
		return err
	}
	if err := writeJSON("manifest.json", manifest, 0o600); err != nil {
		return manifest, err
	}
	if err := copyFile(cfg.Storage.DatabasePath, "db/mel.db", 0o600); err != nil {
		return manifest, err
	}
	if cfgPath != "" {
		if err := copyFile(cfgPath, "config/mel.json", 0o600); err != nil {
			return manifest, err
		}
	}
	return manifest, nil
}

func ValidateRestore(bundlePath, destination string) (RestoreReport, error) {
	report := RestoreReport{Valid: false, CheckedAt: time.Now().UTC().Format(time.RFC3339), Destination: destination}
	f, err := os.Open(bundlePath)
	if err != nil {
		return report, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return report, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	seen := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return report, err
		}
		report.Files = append(report.Files, hdr.Name)
		seen[hdr.Name] = true
		if hdr.Name == "manifest.json" {
			if err := json.NewDecoder(tr).Decode(&report.Manifest); err != nil {
				return report, err
			}
		}
	}
	if !seen["manifest.json"] {
		report.Warnings = append(report.Warnings, "bundle is missing manifest.json")
	}
	if !seen["db/mel.db"] {
		report.Warnings = append(report.Warnings, "bundle is missing db/mel.db")
	}
	if destination != "" {
		report.Actions = append(report.Actions, fmt.Sprintf("would restore database to %s", filepath.Join(destination, "mel.db")))
		report.Actions = append(report.Actions, fmt.Sprintf("would restore config snapshot to %s", filepath.Join(destination, "mel.json")))
	}
	if report.Manifest.SchemaVersion == "" {
		report.Warnings = append(report.Warnings, "manifest schema version is empty")
	}
	for _, warning := range report.Warnings {
		if strings.Contains(warning, "missing") {
			report.Valid = false
			return report, nil
		}
	}
	report.Valid = true
	return report, nil
}
