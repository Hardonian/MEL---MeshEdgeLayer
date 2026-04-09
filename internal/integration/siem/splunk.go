// Package siem provides SIEM export adapters for Splunk HEC and Datadog API.
package siem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Event represents a normalized security/operations event for SIEM ingestion.
type Event struct {
	Timestamp   string                 `json:"timestamp"`
	Level       string                 `json:"level"` // info, warning, error, critical
	Source      string                 `json:"source"`
	EventType   string                 `json:"event_type"`
	Message     string                 `json:"message"`
	Hostname    string                 `json:"hostname,omitempty"`
	Service     string                 `json:"service,omitempty"`
	IncidentID  string                 `json:"incident_id,omitempty"`
	MeshNodeID  string                 `json:"mesh_node_id,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

// SplunkConfig holds Splunk HTTP Event Collector (HEC) settings.
type SplunkConfig struct {
	URL       string // e.g., https://localhost:8088/services/collector
	Token     string // HEC token
	Index     string // Target index
	Source    string // Source override
	Channel   string // GUID for acknowledgment
	SkipVerify bool
}

// SplunkClient posts events to Splunk HEC.
type SplunkClient struct {
	Client *http.Client
	Config SplunkConfig
}

// NewSplunkClient creates a Splunk HEC client.
func NewSplunkClient(cfg SplunkConfig) *SplunkClient {
	return &SplunkClient{
		Client: &http.Client{Timeout: 15 * time.Second},
		Config: cfg,
	}
}

// Push sends events to Splunk HEC in batches.
func (c *SplunkClient) Push(ctx context.Context, events []Event) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}

	payload := make([]map[string]interface{}, len(events))
	for i, ev := range events {
		payload[i] = map[string]interface{}{
			"time":       ev.Timestamp,
			"host":       ev.Hostname,
			"source":     ev.Source,
			"sourcetype": ev.EventType,
			"index":      c.Config.Index,
			"event":      ev,
		}
		if c.Config.Source != "" {
			payload[i]["source"] = c.Config.Source
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("json marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Config.URL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Splunk "+c.Config.Token)
	if c.Config.Channel != "" {
		req.Header.Set("X-Splunk-Request-Channel", c.Config.Channel)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return len(events), nil
	}
	return 0, fmt.Errorf("splunk hec status %d: %s", resp.StatusCode, string(respBody))
}

// DatadogConfig holds Datadog API settings.
type DatadogConfig struct {
	APIKey string // Datadog API key
	Site   string // e.g., datadoghq.com, datadoghq.eu
}

// DatadogClient posts events to Datadog Events API.
type DatadogClient struct {
	Client *http.Client
	Config DatadogConfig
}

// NewDatadogClient creates a Datadog client.
func NewDatadogClient(cfg DatadogConfig) *DatadogClient {
	site := cfg.Site
	if site == "" {
		site = "datadoghq.com"
	}
	return &DatadogClient{
		Client: &http.Client{Timeout: 15 * time.Second},
		Config: cfg,
	}
}

// Push sends an event to Datadog.
func (c *DatadogClient) Push(ctx context.Context, ev Event) error {
	ddEvent := map[string]interface{}{
		"title":       ev.Message,
		"text":        ev.Message,
		"timestamp":   time.Now().Unix(),
		"source_type_name": ev.Source,
		"host":        ev.Hostname,
		"service":     ev.Service,
		"tags":        ev.Tags,
		"alert_type":   mapLevelToAlertType(ev.Level),
		"event_type":  ev.EventType,
		"incident_id": ev.IncidentID,
	}

	body, err := json.Marshal(ddEvent)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	url := fmt.Sprintf("https://api.%s/api/v1/events", c.Config.Site)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", c.Config.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("datadog status %d: %s", resp.StatusCode, string(respBody))
}

// PushBatch sends multiple events to Datadog (creates separate events).
func (c *DatadogClient) PushBatch(ctx context.Context, events []Event) (int, error) {
	var sent int
	for _, ev := range events {
		if err := c.Push(ctx, ev); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}

func mapLevelToAlertType(level string) string {
	switch level {
	case "critical":
		return "error"
	case "warning":
		return "warning"
	default:
		return "info"
	}
}
