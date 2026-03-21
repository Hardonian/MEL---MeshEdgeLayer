// Package integration provides outbound notification adapters (webhook, Slack-compatible, Telegram Bot API).
// Delivery is best-effort with bounded retries; callers receive structured results for audit and CLI verification.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/semantics"
)

const (
	maxBodyBytes = 256 * 1024
)

// Event is a stable, versioned envelope for outbound notifications.
type Event struct {
	SchemaVersion string         `json:"schema_version"`
	EventType     string         `json:"event_type"`
	Timestamp     string         `json:"timestamp"`
	Source        string         `json:"source"`
	Severity      string         `json:"severity,omitempty"`
	Transport     string         `json:"transport_name,omitempty"`
	Summary       string         `json:"summary"`
	Details       map[string]any `json:"details,omitempty"`
}

// DeliveryResult records one attempt outcome.
type DeliveryResult struct {
	Channel     string `json:"channel"`
	Target      string `json:"target"`
	Attempts    int    `json:"attempts"`
	Success     bool   `json:"success"`
	Status      int    `json:"http_status,omitempty"`
	Error       string `json:"error,omitempty"`
	RateLimited bool   `json:"rate_limited,omitempty"`
}

// Client performs HTTP posts with timeout and exponential backoff retries.
type Client struct {
	HTTP      *http.Client
	UserAgent string
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 12 * time.Second}
}

func (c *Client) userAgent() string {
	if c.UserAgent != "" {
		return c.UserAgent
	}
	return "mel-integration/1.0"
}

// PostJSON sends payload to url with retries (maxAttempts includes the first try).
func (c *Client) PostJSON(ctx context.Context, url string, payload any, maxAttempts int) (int, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	var lastStatus int
	var lastErr error
	backoff := 200 * time.Millisecond
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return 0, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", c.userAgent())
		resp, err := c.httpClient().Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts {
				select {
				case <-ctx.Done():
					return 0, ctx.Err()
				case <-time.After(backoff):
				}
				if backoff < 2*time.Second {
					backoff *= 2
				}
			}
			continue
		}
		lastStatus = resp.StatusCode
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return lastStatus, nil
		}
		lastErr = fmt.Errorf("http status %d", resp.StatusCode)
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return lastStatus, ctx.Err()
			case <-time.After(backoff):
			}
			if backoff < 2*time.Second {
				backoff *= 2
			}
			continue
		}
		if attempt < maxAttempts && resp.StatusCode >= 500 {
			select {
			case <-ctx.Done():
				return lastStatus, ctx.Err()
			case <-time.After(backoff):
			}
			if backoff < 2*time.Second {
				backoff *= 2
			}
			continue
		}
		break
	}
	return lastStatus, lastErr
}

// DeliverWebhook posts the event JSON to the given URL.
func (c *Client) DeliverWebhook(ctx context.Context, url string, ev Event) DeliveryResult {
	res := DeliveryResult{Channel: semantics.ChannelWebhook, Target: redactURL(url)}
	st, err := c.PostJSON(ctx, url, ev, 3)
	res.Attempts = 3
	if err != nil {
		res.Error = err.Error()
		if st == http.StatusTooManyRequests {
			res.RateLimited = true
		}
		return res
	}
	res.Status = st
	res.Success = true
	return res
}

// SlackWebhookPayload matches Slack incoming-webhook JSON shape.
type SlackWebhookPayload struct {
	Text string `json:"text"`
}

// DeliverSlack posts a plain-text message to a Slack incoming webhook URL.
func (c *Client) DeliverSlack(ctx context.Context, webhookURL string, text string) DeliveryResult {
	res := DeliveryResult{Channel: semantics.ChannelSlack, Target: redactURL(webhookURL)}
	st, err := c.PostJSON(ctx, webhookURL, SlackWebhookPayload{Text: text}, 3)
	res.Attempts = 3
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Status = st
	res.Success = true
	return res
}

// TelegramSendMessageRequest is the Telegram Bot API sendMessage JSON body.
type TelegramSendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// DeliverTelegram calls https://api.telegram.org/bot<token>/sendMessage
func (c *Client) DeliverTelegram(ctx context.Context, botToken, chatID, text string) DeliveryResult {
	token := strings.TrimSpace(botToken)
	if token == "" {
		return DeliveryResult{Channel: semantics.ChannelTelegram, Target: "[telegram]", Attempts: 0, Error: "empty bot token"}
	}
	url := "https://api.telegram.org/bot" + token + "/sendMessage"
	res := DeliveryResult{Channel: semantics.ChannelTelegram, Target: "api.telegram.org"}
	st, err := c.PostJSON(ctx, url, TelegramSendMessageRequest{ChatID: chatID, Text: text}, 3)
	res.Attempts = 3
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Status = st
	res.Success = st >= 200 && st < 300
	if !res.Success {
		res.Error = fmt.Sprintf("http status %d", st)
	}
	return res
}

func redactURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Avoid leaking query tokens in CLI output
	if i := strings.Index(raw, "?"); i >= 0 {
		return raw[:i] + "?…"
	}
	return raw
}
