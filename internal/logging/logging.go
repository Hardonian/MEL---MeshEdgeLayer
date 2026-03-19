package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	mu    sync.Mutex
	level int
	debug bool
}

func New(level string, debug bool) *Logger {
	return &Logger{level: levelValue(level), debug: debug}
}

func (l *Logger) log(level, eventType, msg string, fields map[string]any) {
	if levelValue(level) < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if fields == nil {
		fields = map[string]any{}
	}
	fields = redact(fields)
	fields["ts"] = time.Now().UTC().Format(time.RFC3339)
	fields["level"] = level
	fields["event_type"] = eventType
	fields["msg"] = msg
	fields["debug_enabled"] = l.debug
	b, _ := json.Marshal(fields)
	fmt.Fprintln(os.Stdout, string(b))
}
func (l *Logger) Debug(eventType, msg string, f map[string]any) { l.log("debug", eventType, msg, f) }
func (l *Logger) Info(eventType, msg string, f map[string]any)  { l.log("info", eventType, msg, f) }
func (l *Logger) Error(eventType, msg string, f map[string]any) { l.log("error", eventType, msg, f) }

func levelValue(level string) int {
	switch strings.ToLower(level) {
	case "debug":
		return 10
	case "info", "":
		return 20
	case "error":
		return 30
	default:
		return 20
	}
}

func redact(fields map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range fields {
		if looksSensitive(k) {
			out[k] = "[redacted]"
			continue
		}
		out[k] = v
	}
	return out
}

func looksSensitive(key string) bool {
	k := strings.ToLower(key)
	return strings.Contains(k, "password") || strings.Contains(k, "secret") || strings.Contains(k, "token") || strings.Contains(k, "key")
}
