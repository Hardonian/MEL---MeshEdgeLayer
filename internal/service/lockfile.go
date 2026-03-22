package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const lockFileName = "mel.lock"

// acquireLockFile attempts to acquire a file lock to prevent multiple MEL instances
// from running against the same data directory. Returns a cleanup function to release
// the lock on shutdown.
func acquireLockFile(dataDir string) (cleanup func(), err error) {
	lockPath := filepath.Join(dataDir, lockFileName)

	// Check if lock file already exists
	if data, err := os.ReadFile(lockPath); err == nil {
		// Lock file exists — check if the PID is still alive
		parts := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
		if len(parts) >= 1 {
			pid, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err == nil && pid > 0 {
				proc, err := os.FindProcess(pid)
				if err == nil {
					// On Unix, FindProcess always succeeds. Send signal 0 to check if alive.
					if err := proc.Signal(syscall.Signal(0)); err == nil {
						return nil, fmt.Errorf("another MEL instance (PID %d) is already running with data directory %s — remove %s if the process is stale", pid, dataDir, lockPath)
					}
				}
			}
		}
		// Stale lock file — previous instance crashed without cleanup
	}

	// Write our PID to the lock file
	content := fmt.Sprintf("%d\n%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(lockPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("failed to create lock file %s: %w", lockPath, err)
	}

	cleanup = func() {
		_ = os.Remove(lockPath)
	}
	return cleanup, nil
}
