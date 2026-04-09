// Package teams provides Microsoft Teams incoming webhook integration.
package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Message is a Teams adaptive card message payload.
type Message struct {
	Type        string       `json:"type"`
	Attachments []Attachment `json:"attachments"`
}

// Attachment contains the adaptive card.
type Attachment struct {
	ContentType string  `json:"contentType"`
	Content     Content `json:"content"`
}

// Content is the adaptive card structure.
type Content struct {
	Schema  string  `json:"$schema"`
	Type    string  `json:"type"`
	Version string  `json:"version"`
	Body    []BodyItem `json:"body"`
}

// BodyItem is a container in the card.
type BodyItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Items []BodyItem `json:"items,omitempty"`
	Columns []Column `json:"columns,omitempty"`
	Actions []Action `json:"actions,omitempty"`
}

// Column layout for card elements.
type Column struct {
	Type string `json:"type"`
	Width string `json:"width"`
	Items []BodyItem `json:"items,omitempty"`
}

// Action is a button in the card.
type Action struct {
	Type string `json:"type"`
	Title string `json:"title"`
	URL string `json:"url"`
}

// NewMessage creates a Teams message with title and text.
func NewMessage(title, text string) Message {
	return Message{
		Type: "Message",
		Attachments: []Attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: Content{
					Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
					Type:    "AdaptiveCard",
					Version: "1.4",
					Body: []BodyItem{
						{
							Type: "TextBlock",
							Text: title,
						},
						{
							Type: "TextBlock",
							Text: text,
							Wrap: true,
						},
					},
				},
			},
		},
	}
}

// AddFact adds a key-value fact row to the message.
func AddFact(msg Message, title, value string) Message {
	fact := BodyItem{
		Type: "FactSet",
		Items: []BodyItem{
			{Type: "TextBlock", Text: title},
			{Type: "TextBlock", Text: value},
		},
	}
	msg.Attachments[0].Content.Body = append(msg.Attachments[0].Content.Body, fact)
	return msg
}

// AddAction adds a button to the message.
func AddAction(msg Message, title, url string) Message {
	action := Action{
		Type:  "Action.OpenUrl",
		Title: title,
		URL:   url,
	}
	msg.Attachments[0].Content.Body = append(msg.Attachments[0].Content.Body, BodyItem{
		Type: "ActionSet",
		Actions: []Action{action},
	})
	return msg
}

// Client posts messages to Teams webhooks.
type Client struct {
	HTTP *http.Client
}

// NewClient creates a Teams webhook client.
func NewClient() *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send posts a message to the Teams webhook URL.
func (c *Client) Send(ctx context.Context, webhookURL string, msg Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("teams status %d: %s", resp.StatusCode, string(respBody))
}

// SendSimple sends a simple text message.
func (c *Client) SendSimple(ctx context.Context, webhookURL, title, text string) error {
	msg := NewMessage(title, text)
	return c.Send(ctx, webhookURL, msg)
}
