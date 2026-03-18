package plugins

import (
	"strings"

	"github.com/mel-project/mel/internal/events"
)

type Plugin interface {
	Name() string
	Handle(events.Event) *Alert
}

type Alert struct {
	Plugin  string `json:"plugin"`
	Message string `json:"message"`
}

type UnsafeMQTTPlugin struct{}

func (UnsafeMQTTPlugin) Name() string { return "unsafe-mqtt" }
func (UnsafeMQTTPlugin) Handle(evt events.Event) *Alert {
	if evt.Type != "privacy.audit" {
		return nil
	}
	if s, ok := evt.Data.(string); ok && strings.Contains(s, "mqtt") {
		return &Alert{Plugin: "unsafe-mqtt", Message: s}
	}
	return nil
}
