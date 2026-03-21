package kernel

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// CoordinationToken is a lightweight distributed lock for action execution.
// It prevents duplicate execution across federated nodes.
type CoordinationToken struct {
	// TokenID uniquely identifies this coordination token.
	TokenID string `json:"token_id"`

	// ActionID is the action this token coordinates.
	ActionID string `json:"action_id"`

	// OwnerNodeID is the node that holds this token.
	OwnerNodeID string `json:"owner_node_id"`

	// AcquiredAt is when the token was acquired.
	AcquiredAt time.Time `json:"acquired_at"`

	// ExpiresAt is when the token automatically releases.
	ExpiresAt time.Time `json:"expires_at"`

	// Executed indicates whether the action was executed.
	Executed bool `json:"executed"`

	// Fenced indicates the token is fenced (no longer valid for new actions).
	Fenced bool `json:"fenced"`
}

// ActionCoordinator manages distributed action execution to prevent duplicates.
type ActionCoordinator struct {
	mu     sync.Mutex
	tokens map[string]*CoordinationToken // keyed by ActionID
	nodeID string
	ttl    time.Duration
}

// NewActionCoordinator creates an action coordinator for the given node.
func NewActionCoordinator(nodeID string, tokenTTL time.Duration) *ActionCoordinator {
	if tokenTTL <= 0 {
		tokenTTL = 5 * time.Minute
	}
	return &ActionCoordinator{
		tokens: make(map[string]*CoordinationToken),
		nodeID: nodeID,
		ttl:    tokenTTL,
	}
}

// TryAcquire attempts to acquire a coordination token for an action.
// Returns (token, true) if acquired, (nil, false) if already held by another node.
func (ac *ActionCoordinator) TryAcquire(actionID string) (*CoordinationToken, bool) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	now := time.Now().UTC()

	// Check for existing token
	if existing, ok := ac.tokens[actionID]; ok {
		if existing.OwnerNodeID == ac.nodeID {
			return existing, true // Already own it
		}
		if now.Before(existing.ExpiresAt) && !existing.Fenced {
			return nil, false // Held by another node, not expired
		}
		// Expired or fenced: can take over
	}

	token := &CoordinationToken{
		TokenID:     newTokenID(),
		ActionID:    actionID,
		OwnerNodeID: ac.nodeID,
		AcquiredAt:  now,
		ExpiresAt:   now.Add(ac.ttl),
	}
	ac.tokens[actionID] = token
	return token, true
}

// Release releases a coordination token after action completion.
func (ac *ActionCoordinator) Release(actionID string, executed bool) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if token, ok := ac.tokens[actionID]; ok {
		if token.OwnerNodeID == ac.nodeID {
			token.Executed = executed
			token.Fenced = true
		}
	}
}

// IsOwned returns whether this node currently holds the token for an action.
func (ac *ActionCoordinator) IsOwned(actionID string) bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	token, ok := ac.tokens[actionID]
	if !ok {
		return false
	}
	return token.OwnerNodeID == ac.nodeID &&
		time.Now().UTC().Before(token.ExpiresAt) &&
		!token.Fenced
}

// Cleanup removes expired tokens.
func (ac *ActionCoordinator) Cleanup() int {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	now := time.Now().UTC()
	removed := 0
	for id, token := range ac.tokens {
		if now.After(token.ExpiresAt) || token.Fenced {
			delete(ac.tokens, id)
			removed++
		}
	}
	return removed
}

// ActiveTokens returns the count of active (non-expired, non-fenced) tokens.
func (ac *ActionCoordinator) ActiveTokens() int {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	now := time.Now().UTC()
	count := 0
	for _, token := range ac.tokens {
		if now.Before(token.ExpiresAt) && !token.Fenced {
			count++
		}
	}
	return count
}

// RecordRemoteToken records a coordination token acquired by a remote node.
// This prevents local acquisition of the same action.
func (ac *ActionCoordinator) RecordRemoteToken(token CoordinationToken) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	existing, ok := ac.tokens[token.ActionID]
	if ok && existing.OwnerNodeID == ac.nodeID && !existing.Fenced {
		// We hold it locally and it's not fenced; don't overwrite
		return
	}
	ac.tokens[token.ActionID] = &token
}

func newTokenID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("tok-%s", hex.EncodeToString(b))
}
