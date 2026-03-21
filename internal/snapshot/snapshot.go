// Package snapshot implements periodic state snapshots and checkpointing
// for the MEL kernel, enabling snapshot+delta replay instead of full replay.
//
// Snapshots include:
//   - node registry state
//   - scoring state
//   - action lifecycle state
//   - active freezes
//   - policy version
//   - last event sequence number
//
// Snapshots are stored durably and include integrity verification.
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/kernel"
)

// Snapshot is a complete point-in-time capture of kernel state.
type Snapshot struct {
	ID              string       `json:"id"`
	NodeID          string       `json:"node_id"`
	SequenceNum     uint64       `json:"sequence_num"`
	State           kernel.State `json:"state"`
	PolicyVersion   string       `json:"policy_version"`
	CreatedAt       time.Time    `json:"created_at"`
	IntegrityHash   string       `json:"integrity_hash"`
	EventCount      uint64       `json:"event_count"`
	SizeBytes       int          `json:"size_bytes"`
}

// Store manages snapshot persistence.
type Store struct {
	dbPath string
	nodeID string
}

// NewStore creates a snapshot store backed by the given SQLite database.
func NewStore(dbPath, nodeID string) (*Store, error) {
	s := &Store{dbPath: dbPath, nodeID: nodeID}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("snapshot: init schema: %w", err)
	}
	return s, nil
}

// Create takes a snapshot of the current kernel state.
func (s *Store) Create(state *kernel.State, sequenceNum uint64) (*Snapshot, error) {
	snap := &Snapshot{
		ID:            kernel.NewEventID(), // reuse ID generator
		NodeID:        s.nodeID,
		SequenceNum:   sequenceNum,
		State:         *state,
		PolicyVersion: state.PolicyVersion,
		CreatedAt:     time.Now().UTC(),
	}

	// Compute integrity hash
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("snapshot: marshal state: %w", err)
	}
	hash := sha256.Sum256(stateJSON)
	snap.IntegrityHash = hex.EncodeToString(hash[:])
	snap.SizeBytes = len(stateJSON)

	// Persist
	sql := fmt.Sprintf(
		`INSERT INTO kernel_snapshots (
			snapshot_id, node_id, sequence_num, state_json, policy_version,
			created_at, integrity_hash, size_bytes
		) VALUES (%s, %s, %d, %s, %s, %s, %s, %d);`,
		sqlQuote(snap.ID), sqlQuote(snap.NodeID), snap.SequenceNum,
		sqlQuote(string(stateJSON)), sqlQuote(snap.PolicyVersion),
		sqlQuote(snap.CreatedAt.Format(time.RFC3339Nano)),
		sqlQuote(snap.IntegrityHash), snap.SizeBytes,
	)

	if err := s.exec(sql); err != nil {
		return nil, fmt.Errorf("snapshot: create: %w", err)
	}

	return snap, nil
}

// Latest returns the most recent snapshot, or nil if none exists.
func (s *Store) Latest() (*Snapshot, error) {
	rows, err := s.queryRows(
		"SELECT snapshot_id, node_id, sequence_num, state_json, policy_version, created_at, integrity_hash, size_bytes FROM kernel_snapshots ORDER BY sequence_num DESC LIMIT 1;",
	)
	if err != nil {
		return nil, fmt.Errorf("snapshot: latest: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return parseSnapshotRow(rows[0])
}

// ByID retrieves a snapshot by its ID.
func (s *Store) ByID(id string) (*Snapshot, error) {
	rows, err := s.queryRows(fmt.Sprintf(
		"SELECT snapshot_id, node_id, sequence_num, state_json, policy_version, created_at, integrity_hash, size_bytes FROM kernel_snapshots WHERE snapshot_id = %s LIMIT 1;",
		sqlQuote(id),
	))
	if err != nil {
		return nil, fmt.Errorf("snapshot: by_id: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return parseSnapshotRow(rows[0])
}

// List returns recent snapshots, newest first.
func (s *Store) List(limit int) ([]Snapshot, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.queryRows(fmt.Sprintf(
		"SELECT snapshot_id, node_id, sequence_num, state_json, policy_version, created_at, integrity_hash, size_bytes FROM kernel_snapshots ORDER BY sequence_num DESC LIMIT %d;",
		limit,
	))
	if err != nil {
		return nil, fmt.Errorf("snapshot: list: %w", err)
	}

	snapshots := make([]Snapshot, 0, len(rows))
	for _, row := range rows {
		snap, err := parseSnapshotRow(row)
		if err != nil {
			continue
		}
		snapshots = append(snapshots, *snap)
	}
	return snapshots, nil
}

// Verify checks the integrity hash of a snapshot.
func (s *Store) Verify(snap *Snapshot) (bool, error) {
	stateJSON, err := json.Marshal(snap.State)
	if err != nil {
		return false, fmt.Errorf("snapshot: verify marshal: %w", err)
	}
	hash := sha256.Sum256(stateJSON)
	actual := hex.EncodeToString(hash[:])
	return actual == snap.IntegrityHash, nil
}

// Prune removes snapshots older than the retention limit, keeping at least
// the most recent `keepMin` snapshots.
func (s *Store) Prune(olderThan time.Time, keepMin int) error {
	if keepMin < 1 {
		keepMin = 3
	}
	sql := fmt.Sprintf(
		`DELETE FROM kernel_snapshots WHERE created_at < %s AND snapshot_id NOT IN (
			SELECT snapshot_id FROM kernel_snapshots ORDER BY sequence_num DESC LIMIT %d
		);`,
		sqlQuote(olderThan.Format(time.RFC3339Nano)), keepMin,
	)
	return s.exec(sql)
}

// ─── Internal ────────────────────────────────────────────────────────────────

func (s *Store) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS kernel_snapshots (
	snapshot_id    TEXT PRIMARY KEY,
	node_id        TEXT NOT NULL,
	sequence_num   INTEGER NOT NULL,
	state_json     TEXT NOT NULL,
	policy_version TEXT NOT NULL DEFAULT '',
	created_at     TEXT NOT NULL,
	integrity_hash TEXT NOT NULL,
	size_bytes     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_ks_seq ON kernel_snapshots(sequence_num);
CREATE INDEX IF NOT EXISTS idx_ks_created ON kernel_snapshots(created_at);
`
	return s.exec(schema)
}

func parseSnapshotRow(row map[string]any) (*Snapshot, error) {
	snap := &Snapshot{
		ID:            asStr(row["snapshot_id"]),
		NodeID:        asStr(row["node_id"]),
		PolicyVersion: asStr(row["policy_version"]),
		IntegrityHash: asStr(row["integrity_hash"]),
	}

	if v, ok := row["sequence_num"].(float64); ok {
		snap.SequenceNum = uint64(v)
	}
	if v, ok := row["size_bytes"].(float64); ok {
		snap.SizeBytes = int(v)
	}
	if ts := asStr(row["created_at"]); ts != "" {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			snap.CreatedAt = t
		}
	}

	stateJSON := asStr(row["state_json"])
	if stateJSON != "" {
		if err := json.Unmarshal([]byte(stateJSON), &snap.State); err != nil {
			return nil, fmt.Errorf("snapshot: parse state: %w", err)
		}
	}

	return snap, nil
}

func (s *Store) exec(sql string) error {
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", s.dbPath)
	cmd.Stdin = strings.NewReader(sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite exec: %w: %s", err, out)
	}
	return nil
}

func (s *Store) queryRows(sql string) ([]map[string]any, error) {
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", "-json", s.dbPath, sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sqlite query: %w: %s", err, out)
	}
	if len(out) == 0 {
		return nil, nil
	}
	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("sqlite parse: %w", err)
	}
	return rows, nil
}

func asStr(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func sqlQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
