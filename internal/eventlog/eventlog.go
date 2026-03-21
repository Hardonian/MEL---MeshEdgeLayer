// Package eventlog implements the canonical append-only event log for MEL.
//
// All core inputs become events. The event log is:
//   - append-only
//   - ordered (locally monotonic sequence numbers)
//   - uniquely identifiable (event_id)
//   - timestamped (logical + wall clock)
//   - source-attributed (node_id)
//   - durable (SQLite-backed)
//
// The event log is the source of truth for the MEL kernel.
package eventlog

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mel-project/mel/internal/kernel"
)

// Log is the durable append-only event log backed by SQLite.
type Log struct {
	mu          sync.Mutex
	dbPath      string
	sequenceNum atomic.Uint64
	nodeID      string

	// Stats
	appended  atomic.Uint64
	queried   atomic.Uint64
	compacted atomic.Uint64
}

// Config holds event log configuration.
type Config struct {
	DBPath           string `json:"db_path"`
	NodeID           string `json:"node_id"`
	RetentionDays    int    `json:"retention_days"`
	CompactionTarget int    `json:"compaction_target"` // target max events before compaction
	MaxBatchSize     int    `json:"max_batch_size"`
}

// Stats holds event log operational statistics.
type Stats struct {
	TotalEvents     uint64 `json:"total_events"`
	LastSequenceNum uint64 `json:"last_sequence_num"`
	Appended        uint64 `json:"appended"`
	Queried         uint64 `json:"queried"`
	Compacted       uint64 `json:"compacted"`
}

// Open creates or opens an event log at the given database path.
func Open(cfg Config) (*Log, error) {
	l := &Log{
		dbPath: cfg.DBPath,
		nodeID: cfg.NodeID,
	}

	// Initialize the event log table
	if err := l.initSchema(); err != nil {
		return nil, fmt.Errorf("eventlog: init schema: %w", err)
	}

	// Recover last sequence number
	seq, err := l.recoverSequence()
	if err != nil {
		return nil, fmt.Errorf("eventlog: recover sequence: %w", err)
	}
	l.sequenceNum.Store(seq)

	return l, nil
}

// Append adds a new event to the log. The event's SequenceNum and Checksum
// are set by the log. Returns the assigned sequence number.
func (l *Log) Append(event *kernel.Event) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	seq := l.sequenceNum.Add(1)
	event.SequenceNum = seq
	event.Checksum = kernel.ComputeChecksum(*event)

	if event.SourceNodeID == "" {
		event.SourceNodeID = l.nodeID
	}

	metaJSON, _ := json.Marshal(event.Metadata)
	dataStr := string(event.Data)
	metaStr := string(metaJSON)

	sql := fmt.Sprintf(
		`INSERT INTO kernel_event_log (
			event_id, sequence_num, event_type, timestamp, logical_clock,
			source_node_id, source_region, subject, data, metadata,
			policy_version, causal_parent, checksum
		) VALUES (%s, %d, %s, %s, %d, %s, %s, %s, %s, %s, %s, %s, %s);`,
		sqlQuote(event.ID), seq, sqlQuote(string(event.Type)),
		sqlQuote(event.Timestamp.UTC().Format(time.RFC3339Nano)),
		event.LogicalClock,
		sqlQuote(event.SourceNodeID), sqlQuote(event.SourceRegion),
		sqlQuote(event.Subject), sqlQuote(dataStr), sqlQuote(metaStr),
		sqlQuote(event.PolicyVersion), sqlQuote(event.CausalParent),
		sqlQuote(event.Checksum),
	)

	if err := l.exec(sql); err != nil {
		return 0, fmt.Errorf("eventlog: append: %w", err)
	}

	l.appended.Add(1)
	return seq, nil
}

// AppendBatch appends multiple events atomically within a single transaction.
func (l *Log) AppendBatch(events []*kernel.Event) error {
	if len(events) == 0 {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	var stmts []string
	stmts = append(stmts, "BEGIN TRANSACTION;")

	for _, event := range events {
		seq := l.sequenceNum.Add(1)
		event.SequenceNum = seq
		event.Checksum = kernel.ComputeChecksum(*event)

		if event.SourceNodeID == "" {
			event.SourceNodeID = l.nodeID
		}

		metaJSON, _ := json.Marshal(event.Metadata)

		stmts = append(stmts, fmt.Sprintf(
			`INSERT INTO kernel_event_log (
				event_id, sequence_num, event_type, timestamp, logical_clock,
				source_node_id, source_region, subject, data, metadata,
				policy_version, causal_parent, checksum
			) VALUES (%s, %d, %s, %s, %d, %s, %s, %s, %s, %s, %s, %s, %s);`,
			sqlQuote(event.ID), seq, sqlQuote(string(event.Type)),
			sqlQuote(event.Timestamp.UTC().Format(time.RFC3339Nano)),
			event.LogicalClock,
			sqlQuote(event.SourceNodeID), sqlQuote(event.SourceRegion),
			sqlQuote(event.Subject), sqlQuote(string(event.Data)), sqlQuote(string(metaJSON)),
			sqlQuote(event.PolicyVersion), sqlQuote(event.CausalParent),
			sqlQuote(event.Checksum),
		))
	}

	stmts = append(stmts, "COMMIT;")

	if err := l.exec(strings.Join(stmts, "\n")); err != nil {
		return fmt.Errorf("eventlog: append batch: %w", err)
	}

	l.appended.Add(uint64(len(events)))
	return nil
}

// Query retrieves events matching the given filter, ordered by sequence number.
type QueryFilter struct {
	AfterSequence  uint64             `json:"after_sequence"`
	BeforeSequence uint64             `json:"before_sequence"`
	EventTypes     []kernel.EventType `json:"event_types,omitempty"`
	SourceNodeID   string             `json:"source_node_id,omitempty"`
	SourceRegion   string             `json:"source_region,omitempty"`
	Subject        string             `json:"subject,omitempty"`
	Since          time.Time          `json:"since,omitempty"`
	Until          time.Time          `json:"until,omitempty"`
	Limit          int                `json:"limit"`
}

// Query returns events matching the filter.
func (l *Log) Query(filter QueryFilter) ([]kernel.Event, error) {
	var conditions []string

	if filter.AfterSequence > 0 {
		conditions = append(conditions, fmt.Sprintf("sequence_num > %d", filter.AfterSequence))
	}
	if filter.BeforeSequence > 0 {
		conditions = append(conditions, fmt.Sprintf("sequence_num < %d", filter.BeforeSequence))
	}
	if len(filter.EventTypes) > 0 {
		quoted := make([]string, len(filter.EventTypes))
		for i, et := range filter.EventTypes {
			quoted[i] = sqlQuote(string(et))
		}
		conditions = append(conditions, fmt.Sprintf("event_type IN (%s)", strings.Join(quoted, ",")))
	}
	if filter.SourceNodeID != "" {
		conditions = append(conditions, fmt.Sprintf("source_node_id = %s", sqlQuote(filter.SourceNodeID)))
	}
	if filter.SourceRegion != "" {
		conditions = append(conditions, fmt.Sprintf("source_region = %s", sqlQuote(filter.SourceRegion)))
	}
	if filter.Subject != "" {
		conditions = append(conditions, fmt.Sprintf("subject = %s", sqlQuote(filter.Subject)))
	}
	if !filter.Since.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= %s", sqlQuote(filter.Since.UTC().Format(time.RFC3339Nano))))
	}
	if !filter.Until.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= %s", sqlQuote(filter.Until.UTC().Format(time.RFC3339Nano))))
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	limit := 1000
	if filter.Limit > 0 {
		limit = filter.Limit
		if limit > 100000 {
			limit = 100000 // absolute safety cap
		}
	}

	sql := fmt.Sprintf(
		"SELECT event_id, sequence_num, event_type, timestamp, logical_clock, source_node_id, source_region, subject, data, metadata, policy_version, causal_parent, checksum FROM kernel_event_log%s ORDER BY sequence_num ASC LIMIT %d;",
		where, limit,
	)

	rows, err := l.queryRows(sql)
	if err != nil {
		return nil, fmt.Errorf("eventlog: query: %w", err)
	}

	l.queried.Add(1)
	return parseEventRows(rows)
}

// LastSequenceNum returns the current highest sequence number.
func (l *Log) LastSequenceNum() uint64 {
	return l.sequenceNum.Load()
}

// Count returns the total number of events in the log.
func (l *Log) Count() (uint64, error) {
	rows, err := l.queryRows("SELECT COUNT(*) as cnt FROM kernel_event_log;")
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	cnt, _ := rows[0]["cnt"]
	switch v := cnt.(type) {
	case float64:
		return uint64(v), nil
	default:
		return 0, nil
	}
}

// Compact removes events older than the given retention period, keeping
// at least the most recent `keepMin` events.
func (l *Log) Compact(olderThan time.Time, keepMin uint64) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Find the sequence number threshold
	minKeepSeq := uint64(0)
	if l.sequenceNum.Load() > keepMin {
		minKeepSeq = l.sequenceNum.Load() - keepMin
	}

	sql := fmt.Sprintf(
		"DELETE FROM kernel_event_log WHERE timestamp < %s AND sequence_num < %d;",
		sqlQuote(olderThan.UTC().Format(time.RFC3339Nano)),
		minKeepSeq,
	)

	if err := l.exec(sql); err != nil {
		return 0, fmt.Errorf("eventlog: compact: %w", err)
	}

	// Get count of remaining events to estimate deleted
	rows, err := l.queryRows("SELECT changes() as deleted;")
	if err != nil {
		return 0, nil
	}
	if len(rows) > 0 {
		if v, ok := rows[0]["deleted"].(float64); ok {
			deleted := uint64(v)
			l.compacted.Add(deleted)
			return deleted, nil
		}
	}
	return 0, nil
}

// Stats returns operational statistics.
func (l *Log) Stats() Stats {
	return Stats{
		LastSequenceNum: l.sequenceNum.Load(),
		Appended:        l.appended.Load(),
		Queried:         l.queried.Load(),
		Compacted:       l.compacted.Load(),
	}
}

// ─── Internal ────────────────────────────────────────────────────────────────

func (l *Log) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS kernel_event_log (
	event_id        TEXT PRIMARY KEY,
	sequence_num    INTEGER NOT NULL UNIQUE,
	event_type      TEXT NOT NULL,
	timestamp       TEXT NOT NULL,
	logical_clock   INTEGER NOT NULL DEFAULT 0,
	source_node_id  TEXT NOT NULL DEFAULT '',
	source_region   TEXT NOT NULL DEFAULT '',
	subject         TEXT NOT NULL DEFAULT '',
	data            TEXT NOT NULL DEFAULT '{}',
	metadata        TEXT NOT NULL DEFAULT '{}',
	policy_version  TEXT NOT NULL DEFAULT '',
	causal_parent   TEXT NOT NULL DEFAULT '',
	checksum        TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_kel_seq ON kernel_event_log(sequence_num);
CREATE INDEX IF NOT EXISTS idx_kel_type ON kernel_event_log(event_type);
CREATE INDEX IF NOT EXISTS idx_kel_timestamp ON kernel_event_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_kel_source ON kernel_event_log(source_node_id);
CREATE INDEX IF NOT EXISTS idx_kel_region ON kernel_event_log(source_region);
CREATE INDEX IF NOT EXISTS idx_kel_subject ON kernel_event_log(subject);
`
	return l.exec(schema)
}

func (l *Log) recoverSequence() (uint64, error) {
	rows, err := l.queryRows("SELECT MAX(sequence_num) as max_seq FROM kernel_event_log;")
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	v, ok := rows[0]["max_seq"]
	if !ok || v == nil {
		return 0, nil
	}
	switch val := v.(type) {
	case float64:
		return uint64(val), nil
	default:
		return 0, nil
	}
}

func (l *Log) exec(sql string) error {
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", l.dbPath)
	cmd.Stdin = strings.NewReader(sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sqlite exec: %w: %s", err, out)
	}
	return nil
}

func (l *Log) queryRows(sql string) ([]map[string]any, error) {
	cmd := exec.Command("sqlite3", "-cmd", ".timeout 5000", "-json", l.dbPath, sql)
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

func parseEventRows(rows []map[string]any) ([]kernel.Event, error) {
	events := make([]kernel.Event, 0, len(rows))
	for _, row := range rows {
		evt := kernel.Event{
			ID:            asStr(row["event_id"]),
			Type:          kernel.EventType(asStr(row["event_type"])),
			SourceNodeID:  asStr(row["source_node_id"]),
			SourceRegion:  asStr(row["source_region"]),
			Subject:       asStr(row["subject"]),
			Data:          []byte(asStr(row["data"])),
			PolicyVersion: asStr(row["policy_version"]),
			CausalParent:  asStr(row["causal_parent"]),
			Checksum:      asStr(row["checksum"]),
		}

		if v, ok := row["sequence_num"].(float64); ok {
			evt.SequenceNum = uint64(v)
		}
		if v, ok := row["logical_clock"].(float64); ok {
			evt.LogicalClock = uint64(v)
		}
		if ts := asStr(row["timestamp"]); ts != "" {
			if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				evt.Timestamp = t
			}
		}

		// Parse metadata
		metaStr := asStr(row["metadata"])
		if metaStr != "" && metaStr != "{}" {
			meta := make(map[string]string)
			_ = json.Unmarshal([]byte(metaStr), &meta)
			evt.Metadata = meta
		}

		events = append(events, evt)
	}
	return events, nil
}

func asStr(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func sqlQuote(s string) string {
	escaped := strings.ReplaceAll(s, "'", "''")
	return "'" + escaped + "'"
}

// DBPath returns the database file path.
func (l *Log) DBPath() string { return l.dbPath }

// Exists checks if the event log database exists on disk.
func Exists(dbPath string) bool {
	_, err := os.Stat(dbPath)
	return err == nil
}
