
package support

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/diagnostics"
	"github.com/mel-project/mel/internal/privacy"
)

type Bundle struct {
	GeneratedAt time.Time             `json:"generated_at"`
	Version     string                `json:"version"`
	Config      config.Config         `json:"config"`
	Diagnostics []diagnostics.Finding `json:"diagnostics"`
	Nodes       []map[string]any      `json:"nodes"`
	Messages    []map[string]any      `json:"messages"`
	DeadLetters []map[string]any      `json:"dead_letters"`
	AuditLogs   []map[string]any      `json:"audit_logs"`
}

func Create(cfg config.Config, d *db.DB, version string) (*Bundle, error) {
	diagnosticsRun := diagnostics.RunAllChecks(cfg, d, nil, nil, time.Now().UTC())

	nodes, err := d.QueryRows("SELECT node_num,node_id,long_name,short_name,last_seen,lat_redacted,lon_redacted,altitude FROM nodes ORDER BY node_num;")
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}
	messages, err := d.QueryRows("SELECT transport_name,packet_id,channel_id,gateway_id,from_node,to_node,portnum,payload_text,payload_json,rx_time FROM messages ORDER BY id DESC LIMIT 250;")
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	deadLetters, err := d.QueryRows("SELECT transport_name,transport_type,topic,reason,payload_hex,details_json,created_at FROM dead_letters ORDER BY id DESC LIMIT 250;")
	if err != nil {
		return nil, fmt.Errorf("failed to get dead letters: %w", err)
	}
	auditLogs, err := d.QueryRows("SELECT category,level,message,details_json,created_at FROM audit_logs ORDER BY id DESC LIMIT 250;")
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}

	bundle := &Bundle{
		GeneratedAt: time.Now().UTC(),
		Version:     version,
		Config:      privacy.RedactConfig(cfg),
		Diagnostics: diagnosticsRun.Diagnostics,
		Nodes:       nodes,
		Messages:    messages,
		DeadLetters: deadLetters,
		AuditLogs:   auditLogs,
	}

	if cfg.Privacy.RedactExports {
		bundle.Messages = redactMessages(messages)
	}

	return bundle, nil
}

func (b *Bundle) ToZip() ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	files := map[string]any{
		"bundle.json": b,
	}

	for name, content := range files {
		f, err := zipWriter.Create(name)
		if err != nil {
			return nil, err
		}
		jsonContent, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			return nil, err
		}
		_, err = f.Write(jsonContent)
		if err != nil {
			return nil, err
		}
	}

	err := zipWriter.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func redactMessages(rows []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		cloned := map[string]any{}
		for k, v := range row {
			cloned[k] = v
		}
		cloned["payload_text"] = "[redacted]"
		cloned["payload_json"] = "[redacted]"
		out = append(out, cloned)
	}
	return out
}
