package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Logger struct{ mu sync.Mutex }

func New() *Logger { return &Logger{} }

func (l *Logger) log(level, msg string, fields map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if fields == nil {
		fields = map[string]any{}
	}
	fields = redact(fields)
	fields["ts"] = time.Now().UTC().Format(time.RFC3339)
	fields["level"] = level
	fields["msg"] = msg
	b, _ := json.Marshal(fields)
	fmt.Fprintln(os.Stdout, string(b))
}
func (l *Logger) Info(msg string, f map[string]any)  { l.log("info", msg, f) }
func (l *Logger) Error(msg string, f map[string]any) { l.log("error", msg, f) }

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
