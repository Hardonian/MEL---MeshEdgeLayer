package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// tests run from package dir (…/internal/integration)
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func writeMinimalConfig(t *testing.T, dbPath string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "mel.json")
	payload := map[string]any{
		"storage": map[string]any{
			"data_dir":            filepath.ToSlash(filepath.Dir(dbPath)),
			"database_path":       filepath.ToSlash(dbPath),
			"encryption_required": false,
		},
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMelControlApproveBreakGlassRequiresAck(t *testing.T) {
	root := repoRoot(t)
	dbPath := filepath.Join(t.TempDir(), "mel.db")
	cfgPath := writeMinimalConfig(t, dbPath)
	cfg, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	database, err := db.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	actionID := "act-breakglass-1"
	if err := database.UpsertControlAction(db.ControlActionRecord{
		ID:               actionID,
		ActionType:       control.ActionRestartTransport,
		TargetTransport:  "mqtt-test",
		Reason:           "test",
		Confidence:       0.9,
		ExecutionMode:    control.ExecutionModeApprovalRequired,
		LifecycleState:   control.LifecyclePendingApproval,
		ProposedBy:       "system",
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		BlastRadiusClass: "transport",
	}); err != nil {
		t.Fatal(err)
	}

	goBin := os.Getenv("MEL_TEST_GO")
	if goBin == "" {
		goBin = "go"
	}

	run := func(args ...string) *exec.Cmd {
		cmd := exec.Command(goBin, args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=auto")
		return cmd
	}

	// Without acknowledgement: must fail closed (exit 2 from main)
	cmdNoAck := run("run", "./cmd/mel", "control", "approve", actionID, "--config", cfgPath)
	out, err := cmdNoAck.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error without --i-understand-bypasses-audit, output=%s", string(out))
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		// go run / toolchain may surface the subprocess exit differently; require non-zero + message.
		if exitErr.ExitCode() == 0 {
			t.Fatalf("expected non-zero exit, got 0 output=%s", string(out))
		}
	}
	if !bytes.Contains(out, []byte("mel action approve")) {
		t.Fatalf("expected guidance to mel action approve, got: %s", string(out))
	}

	cmdAck := run("run", "./cmd/mel", "control", "approve", actionID, "--config", cfgPath, "--i-understand-bypasses-audit")
	out2, err := cmdAck.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success with ack flag: %v output=%s", err, string(out2))
	}
	if !bytes.Contains(out2, []byte("WARNING: break-glass")) {
		t.Fatalf("expected stderr warning about break-glass, got: %s", string(out2))
	}
}
