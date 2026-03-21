package selfobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type correlationKey string

const (
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey correlationKey = "mel_correlation_id"
)

// CorrelationID represents a unique identifier for tracing events through the MEL pipeline
type CorrelationID struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"`
}

// NewCorrelationID creates a new correlation ID using cryptographically secure random bytes
func NewCorrelationID(source string) CorrelationID {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based if crypto fails
		return CorrelationID{
			ID:        fmt.Sprintf("fallback-%d", time.Now().UnixNano()),
			CreatedAt: time.Now().UTC(),
			Source:    source,
		}
	}
	return CorrelationID{
		ID:        hex.EncodeToString(bytes),
		CreatedAt: time.Now().UTC(),
		Source:    source,
	}
}

// String returns the string representation of the correlation ID
func (c CorrelationID) String() string {
	return c.ID
}

// ContextWithCorrelationID adds a correlation ID to a context
func ContextWithCorrelationID(ctx context.Context, corr CorrelationID) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, corr)
}

// FromContext extracts a correlation ID from a context
func FromContext(ctx context.Context) (CorrelationID, bool) {
	corr, ok := ctx.Value(CorrelationIDKey).(CorrelationID)
	return corr, ok
}

// ContextWithNewCorrelationID creates a new correlation ID and adds it to context
func ContextWithNewCorrelationID(ctx context.Context, source string) (context.Context, CorrelationID) {
	corr := NewCorrelationID(source)
	return ContextWithCorrelationID(ctx, corr), corr
}

// CorrelationIDPool manages a pool of correlation IDs for reuse
type CorrelationIDPool struct {
	mu      sync.Mutex
	ids     []CorrelationID
	counter int
}

// NewCorrelationIDPool creates a new pool with pre-allocated IDs
func NewCorrelationIDPool(size int) *CorrelationIDPool {
	pool := &CorrelationIDPool{
		ids: make([]CorrelationID, 0, size),
	}
	// Pre-allocate some IDs
	for i := 0; i < size; i++ {
		pool.ids = append(pool.ids, NewCorrelationID("pool"))
	}
	return pool
}

// Get retrieves a correlation ID from the pool or creates a new one
func (p *CorrelationIDPool) Get(source string) CorrelationID {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if len(p.ids) > 0 {
		// Reuse an ID but update the source
		corr := p.ids[len(p.ids)-1]
		p.ids = p.ids[:len(p.ids)-1]
		corr.Source = source
		corr.CreatedAt = time.Now().UTC()
		return corr
	}
	return NewCorrelationID(source)
}

// Put returns a correlation ID to the pool
func (p *CorrelationIDPool) Put(corr CorrelationID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if len(p.ids) < cap(p.ids) {
		p.ids = append(p.ids, corr)
	}
}
