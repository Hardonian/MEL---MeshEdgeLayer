package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warning"
	LevelError = "error"
)

const (
	CategoryTransport = "transport"
	CategorySecurity  = "security"
	CategoryAudit     = "audit"
	CategoryPrivacy   = "privacy"
	CategoryDatabase  = "database"
	CategoryControl   = "control"
	CategoryNode      = "node"
	CategoryPlugin    = "plugin"
)

type correlationKey struct{}

var (
	correlationIDKey = correlationKey{}
	globalLogger     atomic.Value
)

func init() {
	globalLogger.Store(New(LevelInfo, false))
}

func SetGlobalLogger(l *Logger) {
	if l != nil {
		globalLogger.Store(l)
	}
}

func Global() *Logger {
	return globalLogger.Load().(*Logger)
}

type Logger struct {
	mu                sync.Mutex
	level             int
	debug             bool
	rateLimiter       *RateLimiter
	redactIPs         bool
	redactNodeIDs     bool
	redactTopics      bool
	enableCorrelation bool
}

type LoggerOptions struct {
	Level             string
	Debug             bool
	RateLimitInterval time.Duration
	RateLimitBurst    int
	RedactIPs         bool
	RedactNodeIDs     bool
	RedactTopics      bool
	EnableCorrelation bool
}

func DefaultOptions() LoggerOptions {
	return LoggerOptions{
		Level:             LevelInfo,
		Debug:             false,
		RateLimitInterval: time.Second,
		RateLimitBurst:    100,
		RedactIPs:         false,
		RedactNodeIDs:     false,
		RedactTopics:      false,
		EnableCorrelation: true,
	}
}

func New(level string, debug bool) *Logger {
	opts := DefaultOptions()
	opts.Level = level
	opts.Debug = debug
	return NewWithOptions(opts)
}

func NewWithOptions(opts LoggerOptions) *Logger {
	return &Logger{
		level:             levelValue(opts.Level),
		debug:             opts.Debug,
		rateLimiter:       NewRateLimiter(opts.RateLimitInterval, opts.RateLimitBurst),
		redactIPs:         opts.RedactIPs,
		redactNodeIDs:     opts.RedactNodeIDs,
		redactTopics:      opts.RedactTopics,
		enableCorrelation: opts.EnableCorrelation,
	}
}

func (l *Logger) WithContext(ctx context.Context) *ContextLogger {
	return &ContextLogger{logger: l, ctx: ctx}
}

func (l *Logger) Debug(eventType, msg string, f map[string]any) { l.log(LevelDebug, eventType, msg, f) }
func (l *Logger) Info(eventType, msg string, f map[string]any)  { l.log(LevelInfo, eventType, msg, f) }
func (l *Logger) Warn(eventType, msg string, f map[string]any)  { l.log(LevelWarn, eventType, msg, f) }
func (l *Logger) Error(eventType, msg string, f map[string]any) { l.log(LevelError, eventType, msg, f) }

func (l *Logger) Audit(category, msg string, f map[string]any) {
	if f == nil {
		f = map[string]any{}
	}
	f["audit_category"] = category
	f["is_audit"] = true
	l.log(LevelInfo, CategoryAudit, msg, f)
}

func (l *Logger) Security(eventType, msg string, severity string, f map[string]any) {
	if f == nil {
		f = map[string]any{}
	}
	f["security_event"] = true
	f["severity"] = severity
	level := LevelWarn
	if severity == "critical" || severity == "high" {
		level = LevelError
	}
	l.log(level, eventType, msg, f)
}

func (l *Logger) log(level, eventType, msg string, fields map[string]any) {
	if levelValue(level) < l.level {
		return
	}

	if l.rateLimiter != nil && !l.rateLimiter.Allow(eventType) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if fields == nil {
		fields = map[string]any{}
	}

	fields = l.redact(fields)
	fields["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	fields["level"] = level
	fields["event_type"] = eventType
	fields["msg"] = msg
	fields["debug_enabled"] = l.debug

	if l.enableCorrelation {
		if cid := getCorrelationID(nil); cid != "" {
			fields["correlation_id"] = cid
		}
	}

	b, _ := json.Marshal(fields)
	fmt.Fprintln(os.Stdout, string(b))
}

func (l *Logger) redact(fields map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range fields {
		if looksSensitive(k) {
			out[k] = "[redacted]"
			continue
		}
		if l.redactIPs && looksLikeIP(k, v) {
			out[k] = redactIPValue(v)
			continue
		}
		if l.redactNodeIDs && looksLikeNodeID(k) {
			out[k] = "[node_id_redacted]"
			continue
		}
		if l.redactTopics && looksLikeTopic(k) && containsIdentifiableInfo(v) {
			out[k] = redactTopicValue(v)
			continue
		}
		out[k] = v
	}
	return out
}

type ContextLogger struct {
	logger *Logger
	ctx    context.Context
}

func (cl *ContextLogger) Debug(eventType, msg string, f map[string]any) {
	cl.log(LevelDebug, eventType, msg, f)
}
func (cl *ContextLogger) Info(eventType, msg string, f map[string]any) {
	cl.log(LevelInfo, eventType, msg, f)
}
func (cl *ContextLogger) Warn(eventType, msg string, f map[string]any) {
	cl.log(LevelWarn, eventType, msg, f)
}
func (cl *ContextLogger) Error(eventType, msg string, f map[string]any) {
	cl.log(LevelError, eventType, msg, f)
}

func (cl *ContextLogger) Audit(category, msg string, f map[string]any) {
	if f == nil {
		f = map[string]any{}
	}
	f["audit_category"] = category
	f["is_audit"] = true
	cl.log(LevelInfo, CategoryAudit, msg, f)
}

func (cl *ContextLogger) Security(eventType, msg string, severity string, f map[string]any) {
	if f == nil {
		f = map[string]any{}
	}
	f["security_event"] = true
	f["severity"] = severity
	level := LevelWarn
	if severity == "critical" || severity == "high" {
		level = LevelError
	}
	cl.log(level, eventType, msg, f)
}

func (cl *ContextLogger) log(level, eventType, msg string, fields map[string]any) {
	if cl.logger == nil {
		return
	}
	if fields == nil {
		fields = map[string]any{}
	}
	if cid := getCorrelationID(cl.ctx); cid != "" {
		fields["correlation_id"] = cid
	}
	cl.logger.log(level, eventType, msg, fields)
}

type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*tokenBucket
	interval time.Duration
	burst    int
}

type tokenBucket struct {
	tokens  int
	lastAdd time.Time
	mu      sync.Mutex
}

func NewRateLimiter(interval time.Duration, burst int) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		interval: interval,
		burst:    burst,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		bucket = &tokenBucket{tokens: rl.burst - 1, lastAdd: time.Now()}
		rl.buckets[key] = bucket
		rl.mu.Unlock()
		return true
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(bucket.lastAdd)
	tokensToAdd := int(elapsed / rl.interval)
	if tokensToAdd > 0 {
		bucket.tokens = minInt(bucket.tokens+tokensToAdd, rl.burst)
		bucket.lastAdd = now
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for key, bucket := range rl.buckets {
		bucket.mu.Lock()
		if now.Sub(bucket.lastAdd) > maxAge {
			delete(rl.buckets, key)
		}
		bucket.mu.Unlock()
	}
}

func levelValue(level string) int {
	switch strings.ToLower(level) {
	case LevelDebug:
		return 10
	case LevelInfo, "":
		return 20
	case LevelWarn:
		return 25
	case LevelError:
		return 30
	default:
		return 20
	}
}

func looksSensitive(key string) bool {
	k := strings.ToLower(key)
	sensitive := []string{"password", "secret", "token", "key", "credential", "auth", "private", "session_secret", "ui_password", "encryption_key"}
	for _, s := range sensitive {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}

func looksLikeIP(key string, value any) bool {
	k := strings.ToLower(key)
	ipKeys := []string{"ip", "addr", "address", "host", "remote", "client_ip", "source_ip", "endpoint"}
	for _, ik := range ipKeys {
		if strings.Contains(k, ik) {
			return true
		}
	}

	if s, ok := value.(string); ok {
		return net.ParseIP(s) != nil || isIPPort(s)
	}
	return false
}

func isIPPort(s string) bool {
	host, _, err := net.SplitHostPort(s)
	if err != nil {
		return false
	}
	return net.ParseIP(host) != nil
}

func redactIPValue(v any) any {
	switch val := v.(type) {
	case string:
		if ip := net.ParseIP(val); ip != nil {
			return redactIP(ip)
		}
		if host, port, err := net.SplitHostPort(val); err == nil {
			if ip := net.ParseIP(host); ip != nil {
				return redactIP(ip) + ":" + port
			}
		}
		return "[ip_redacted]"
	default:
		return "[ip_redacted]"
	}
}

func redactIP(ip net.IP) string {
	if ip.To4() != nil {
		return "xxx.xxx.xxx.xxx"
	}
	return "xxxx:xxxx:xxxx:xxxx:xxxx:xxxx:xxxx:xxxx"
}

func looksLikeNodeID(key string) bool {
	k := strings.ToLower(key)
	nodeKeys := []string{"node_id", "node_num", "from_node", "to_node", "gateway_id", "relay_node"}
	for _, nk := range nodeKeys {
		if strings.Contains(k, nk) {
			return true
		}
	}
	return false
}

func looksLikeTopic(key string) bool {
	k := strings.ToLower(key)
	return strings.Contains(k, "topic")
}

func containsIdentifiableInfo(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	s = strings.ToLower(s)
	identifiable := []string{"user", "name", "id", "device", "serial", "mac", "uuid"}
	for _, id := range identifiable {
		if strings.Contains(s, id) {
			return true
		}
	}
	return false
}

func redactTopicValue(v any) any {
	s, ok := v.(string)
	if !ok {
		return "[topic_redacted]"
	}
	parts := strings.Split(s, "/")
	for i := range parts {
		if looksLikeNodeID(parts[i]) || containsIdentifiableInfo(parts[i]) {
			parts[i] = "[redacted]"
		}
	}
	return strings.Join(parts, "/")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func WithCorrelationID(ctx context.Context, cid string) context.Context {
	if cid == "" {
		cid = generateCorrelationID()
	}
	return context.WithValue(ctx, correlationIDKey, cid)
}

func getCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if cid, ok := ctx.Value(correlationIDKey).(string); ok {
		return cid
	}
	return ""
}

func generateCorrelationID() string {
	return fmt.Sprintf("%d-%x", time.Now().UnixNano(), randomBytes(8))
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() % 256)
	}
	return b
}

type SafeError struct {
	Public    string
	Internal  string
	Category  string
	Transient bool
	Fields    map[string]any
}

func (e SafeError) Error() string {
	if e.Internal != "" {
		return e.Internal
	}
	return e.Public
}

func NewSafeError(public string, internal error, category string, transient bool) SafeError {
	se := SafeError{
		Public:    public,
		Category:  category,
		Transient: transient,
		Fields:    map[string]any{},
	}
	if internal != nil {
		se.Internal = internal.Error()
	}
	return se
}

func ClassifyError(err error) SafeError {
	if err == nil {
		return SafeError{Public: "no error", Category: "none", Transient: false}
	}

	if se, ok := err.(SafeError); ok {
		return se
	}

	msg := strings.ToLower(err.Error())

	transientPatterns := []string{
		"timeout", "deadline exceeded", "temporary", "retry", "connection refused",
		"no such host", "network is unreachable", "broken pipe", "reset by peer",
		"i/o timeout", "context canceled", "context deadline exceeded",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(msg, pattern) {
			return NewSafeError(
				"Service temporarily unavailable. Please retry.",
				err,
				"transient",
				true,
			)
		}
	}

	terminalPatterns := []string{
		"permission denied", "unauthorized", "forbidden", "invalid", "bad request",
		"not found", "already exists", "conflict", "unprocessable",
	}

	for _, pattern := range terminalPatterns {
		if strings.Contains(msg, pattern) {
			return NewSafeError(
				"Request could not be processed.",
				err,
				"terminal",
				false,
			)
		}
	}

	return NewSafeError(
		"An unexpected error occurred.",
		err,
		"unknown",
		true,
	)
}

func SanitizeDBError(err error) SafeError {
	if err == nil {
		return SafeError{Public: "no error", Category: "none"}
	}

	msg := strings.ToLower(err.Error())

	sensitivePatterns := []string{
		"select", "insert", "update", "delete", "from", "where",
		"table", "column", "schema", "index", "constraint",
		"sqlite", "database is locked", "constraint failed",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(msg, pattern) {
			return NewSafeError(
				"Database operation failed. Please try again later.",
				err,
				"database",
				strings.Contains(msg, "locked") || strings.Contains(msg, "busy"),
			)
		}
	}

	return ClassifyError(err)
}

func APIErrorResponse(err error) map[string]any {
	se := ClassifyError(err)

	if se.Transient {
		return map[string]any{
			"error": map[string]any{
				"code":    "transient_error",
				"message": se.Public,
				"retry":   true,
			},
		}
	}

	return map[string]any{
		"error": map[string]any{
			"code":    "request_error",
			"message": se.Public,
			"retry":   false,
		},
	}
}

var (
	ErrTransient = errors.New("transient error")
	ErrTerminal  = errors.New("terminal error")
)

func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if se, ok := err.(SafeError); ok {
		return se.Transient
	}
	se := ClassifyError(err)
	return se.Transient
}

func IsTerminal(err error) bool {
	if err == nil {
		return false
	}
	return !IsTransient(err)
}

// WrapError wraps an error with a public message, category, and transient flag.
func WrapError(err error, public string, category string, transient bool) error {
	if err == nil {
		return nil
	}
	return NewSafeError(public, err, category, transient)
}

// SanitizeForDebug redacts sensitive keys from a field map for safe debug logging.
func SanitizeForDebug(fields map[string]any) map[string]any {
	if fields == nil {
		return map[string]any{}
	}

	out := map[string]any{}
	sensitiveKeys := []string{"password", "secret", "token", "key", "credential", "auth", "private"}

	for k, v := range fields {
		kLower := strings.ToLower(k)
		isSensitive := false
		for _, sk := range sensitiveKeys {
			if strings.Contains(kLower, sk) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			continue
		}

		if s, ok := v.(string); ok {
			if len(s) > 256 {
				out[k] = s[:256] + "...[truncated]"
				continue
			}
		}

		out[k] = v
	}

	return out
}

var hexPattern = regexp.MustCompile(`^[0-9a-fA-F]+$`)
var nodeIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}$|^[0-9a-f]{16}$`)

func SanitizePayloadForLog(payload []byte, maxLen int) string {
	if len(payload) == 0 {
		return ""
	}
	if len(payload) > maxLen {
		payload = payload[:maxLen]
	}
	hexStr := fmt.Sprintf("%x", payload)
	if len(hexStr) > maxLen*2 {
		hexStr = hexStr[:maxLen*2] + "..."
	}
	return hexStr
}

func SanitizeStringForLog(s string, maxLen int) string {
	if len(s) > maxLen {
		s = s[:maxLen] + "...[truncated]"
	}
	return s
}
