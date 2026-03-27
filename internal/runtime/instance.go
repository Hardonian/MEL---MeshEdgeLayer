package runtime

import (
	"os"
	"time"
)

// ProcessIdentity is live-process truth (this OS process, not historical DB content).
type ProcessIdentity struct {
	PID       int    `json:"pid"`
	StartedAt string `json:"started_at"`
}

// InstanceTruth combines durable instance id (SQLite) with optional live process identity.
type InstanceTruth struct {
	InstanceID    string           `json:"instance_id,omitempty"`
	Process       *ProcessIdentity `json:"process,omitempty"`
	UptimeSeconds int64            `json:"uptime_seconds,omitempty"`
	DataDir       string           `json:"data_dir,omitempty"`
	DatabasePath  string           `json:"database_path,omitempty"`
	ConfigPath    string           `json:"config_path,omitempty"`
	BindAPI       string           `json:"bind_api,omitempty"`
}

// NewProcessIdentity captures pid and RFC3339 start time for the current process.
func NewProcessIdentity(startedAt time.Time) ProcessIdentity {
	return ProcessIdentity{
		PID:       os.Getpid(),
		StartedAt: startedAt.UTC().Format(time.RFC3339),
	}
}
