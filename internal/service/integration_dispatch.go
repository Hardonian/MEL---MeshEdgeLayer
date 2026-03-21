package service

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/events"
	"github.com/mel-project/mel/internal/integration"
	"github.com/mel-project/mel/internal/semantics"
)

func (a *App) integrationWorker(ctx context.Context) {
	if a == nil || !a.Cfg.Integration.Enabled {
		return
	}
	eventsCh := a.Bus.Subscribe()
	var mu sync.Mutex
	lastPost := map[string]time.Time{}
	client := &integration.Client{UserAgent: "mel-daemon/1.0"}
	minGap := time.Duration(a.Cfg.Integration.MinIntervalSeconds) * time.Second
	if minGap <= 0 {
		minGap = 60 * time.Second
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-eventsCh:
			a.dispatchIntegrationEvent(ctx, client, evt, minGap, &mu, lastPost)
		}
	}
}

func (a *App) dispatchIntegrationEvent(ctx context.Context, client *integration.Client, evt events.Event, minGap time.Duration, mu *sync.Mutex, lastPost map[string]time.Time) {
	ic := a.Cfg.Integration
	now := time.Now().UTC()
	out := integration.Event{
		SchemaVersion: "mel.integration.v1",
		EventType:     evt.Type,
		Timestamp:     now.Format(time.RFC3339),
		Source:        "mel",
		Details:       map[string]any{},
	}

	switch evt.Type {
	case "meshtastic.packet":
		if !ic.StateChanges {
			return
		}
		if s, ok := evt.Data.(string); ok {
			out.Summary = s
		} else {
			out.Summary = "mesh packet observed"
		}
	case "integration.transport_state":
		if !ic.StateChanges {
			return
		}
		if m, ok := evt.Data.(map[string]any); ok {
			out.Transport = stringField(m, "transport")
			out.Summary = stringField(m, "summary")
			out.Details = m
		}
	case "integration.alert":
		if !ic.Alerts {
			return
		}
		if m, ok := evt.Data.(map[string]any); ok {
			out.Transport = stringField(m, "transport_name")
			out.Severity = stringField(m, "severity")
			out.Summary = stringField(m, "summary")
			out.Details = m
		}
	case "integration.anomaly":
		if !ic.Anomalies {
			return
		}
		if m, ok := evt.Data.(map[string]any); ok {
			out.Transport = stringField(m, "transport_name")
			out.Severity = stringField(m, "severity")
			out.Summary = stringField(m, "summary")
			out.Details = m
		}
	case "integration.control_action":
		if !ic.Actions {
			return
		}
		if m, ok := evt.Data.(map[string]any); ok {
			out.Summary = stringField(m, "summary")
			out.Severity = semantics.SeverityInfo
			out.Details = m
		}
	default:
		return
	}

	ctx2, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	for _, u := range ic.WebhookURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if !rateAllow(mu, lastPost, "webhook:"+u, minGap, now) {
			continue
		}
		res := client.DeliverWebhook(ctx2, u, out)
		a.logIntegrationResult(res)
	}
	if s := strings.TrimSpace(ic.SlackWebhookURL); s != "" {
		if rateAllow(mu, lastPost, "slack", minGap, now) {
			text := "[" + out.EventType + "] " + out.Summary
			res := client.DeliverSlack(ctx2, s, text)
			a.logIntegrationResult(res)
		}
	}
	if tokEnv := strings.TrimSpace(ic.TelegramBotTokenEnv); tokEnv != "" && strings.TrimSpace(ic.TelegramChatID) != "" {
		if rateAllow(mu, lastPost, "telegram", minGap, now) {
			token := strings.TrimSpace(os.Getenv(tokEnv))
			text := "[" + out.EventType + "] " + out.Summary
			res := client.DeliverTelegram(ctx2, token, ic.TelegramChatID, text)
			a.logIntegrationResult(res)
		}
	}
}

func (a *App) logIntegrationResult(res integration.DeliveryResult) {
	if a == nil || a.Log == nil {
		return
	}
	fields := map[string]any{
		"channel":  res.Channel,
		"target":   res.Target,
		"success":  res.Success,
		"attempts": res.Attempts,
	}
	if res.Status > 0 {
		fields["http_status"] = res.Status
	}
	if res.Error != "" {
		fields["error"] = res.Error
	}
	if res.RateLimited {
		fields["rate_limited"] = true
	}
	if res.Success {
		a.Log.Info("integration_delivered", "outbound integration delivery succeeded", fields)
	} else {
		a.Log.Warn("integration_delivery_failed", "outbound integration delivery failed", fields)
	}
}

func rateAllow(mu *sync.Mutex, last map[string]time.Time, key string, gap time.Duration, now time.Time) bool {
	mu.Lock()
	defer mu.Unlock()
	if t, ok := last[key]; ok && now.Sub(t) < gap {
		return false
	}
	last[key] = now
	return true
}

func stringField(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (a *App) publishTransportStateChange(name, typ, prevState, newState, detail string) {
	if a == nil || a.Bus == nil {
		return
	}
	a.Bus.Publish(events.Event{
		Type: "integration.transport_state",
		Data: map[string]any{
			"transport":  name,
			"type":       typ,
			"prev_state": prevState,
			"new_state":  newState,
			"detail":     detail,
			"summary":    name + " " + prevState + " -> " + newState,
		},
	})
}

func transportStatesEqual(a, b string) bool {
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}

func (a *App) integrationForwardAlert(alert db.TransportAlertRecord) {
	if a == nil || a.Bus == nil || !a.Cfg.Integration.Enabled || !a.Cfg.Integration.Alerts {
		return
	}
	a.Bus.Publish(events.Event{
		Type: "integration.alert",
		Data: map[string]any{
			"alert_id":        alert.ID,
			"transport_name":  alert.TransportName,
			"transport_type":  alert.TransportType,
			"severity":        alert.Severity,
			"reason":          alert.Reason,
			"summary":         alert.Summary,
			"active":          alert.Active,
			"first_triggered": alert.FirstTriggeredAt,
			"last_updated":    alert.LastUpdatedAt,
		},
	})
}

func (a *App) integrationForwardAnomaly(snap db.TransportAnomalySnapshot) {
	if a == nil || a.Bus == nil || !a.Cfg.Integration.Enabled || !a.Cfg.Integration.Anomalies {
		return
	}
	a.Bus.Publish(events.Event{
		Type: "integration.anomaly",
		Data: map[string]any{
			"transport_name":    snap.TransportName,
			"transport_type":    snap.TransportType,
			"reason":            snap.Reason,
			"bucket_start":      snap.BucketStart,
			"count":             snap.Count,
			"dead_letters":      snap.DeadLetters,
			"observation_drops": snap.ObservationDrops,
			"summary":           snap.TransportName + " anomaly " + snap.Reason,
			"severity":          semantics.SeverityMedium,
		},
	})
}

func (a *App) integrationForwardControlAction(summary string, details map[string]any) {
	if a == nil || a.Bus == nil || !a.Cfg.Integration.Enabled || !a.Cfg.Integration.Actions {
		return
	}
	payload := map[string]any{"summary": summary}
	for k, v := range details {
		payload[k] = v
	}
	a.Bus.Publish(events.Event{
		Type: "integration.control_action",
		Data: payload,
	})
}
