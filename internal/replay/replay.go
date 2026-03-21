// Package replay implements deterministic replay of MEL event streams.
//
// Given an event stream and a policy version, the replay engine recomputes:
//   - node states and scores
//   - transport classifications
//   - proposed/executed actions
//   - all kernel effects
//
// Replay modes:
//   - Full: replay entire log from genesis
//   - Windowed: replay events in a time/sequence range
//   - Scenario: replay with modified policy (what-if)
//   - DryRun: replay against modified policy, compare with actual
//   - Verification: replay and compare against known-good state
package replay

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/eventlog"
	"github.com/mel-project/mel/internal/kernel"
)

// Mode specifies the type of replay.
type Mode string

const (
	ModeFull         Mode = "full"
	ModeWindowed     Mode = "windowed"
	ModeScenario     Mode = "scenario"
	ModeDryRun       Mode = "dry_run"
	ModeVerification Mode = "verification"
)

// Request specifies what to replay and how.
type Request struct {
	Mode           Mode           `json:"mode"`
	Policy         kernel.Policy  `json:"policy"`
	FromSequence   uint64         `json:"from_sequence,omitempty"`
	ToSequence     uint64         `json:"to_sequence,omitempty"`
	Since          time.Time      `json:"since,omitempty"`
	Until          time.Time      `json:"until,omitempty"`
	InitialState   *kernel.State  `json:"initial_state,omitempty"` // for snapshot+delta replay
	ExpectedState  *kernel.State  `json:"expected_state,omitempty"` // for verification mode
	MaxEvents      int            `json:"max_events,omitempty"`
}

// Result contains the output of a replay operation.
type Result struct {
	Mode             Mode            `json:"mode"`
	EventsProcessed  int             `json:"events_processed"`
	EffectsProduced  int             `json:"effects_produced"`
	FinalState       kernel.State    `json:"final_state"`
	Effects          []kernel.Effect `json:"effects,omitempty"`
	StartedAt        time.Time       `json:"started_at"`
	CompletedAt      time.Time       `json:"completed_at"`
	DurationMS       int64           `json:"duration_ms"`
	Verified         bool            `json:"verified,omitempty"`
	Divergences      []Divergence    `json:"divergences,omitempty"`
	FirstSequence    uint64          `json:"first_sequence"`
	LastSequence     uint64          `json:"last_sequence"`
}

// Divergence records a specific difference found during verification replay.
type Divergence struct {
	Category string `json:"category"` // node_score, transport_score, action_state, etc.
	Key      string `json:"key"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Severity string `json:"severity"` // info, warning, error
}

// Engine performs replay operations against an event log.
type Engine struct {
	log    *eventlog.Log
	nodeID string
}

// NewEngine creates a replay engine for the given event log.
func NewEngine(log *eventlog.Log, nodeID string) *Engine {
	return &Engine{log: log, nodeID: nodeID}
}

// Execute performs a replay according to the request.
func (e *Engine) Execute(req Request) (*Result, error) {
	started := time.Now()

	// Build query filter from request
	filter := eventlog.QueryFilter{
		AfterSequence:  req.FromSequence,
		BeforeSequence: req.ToSequence,
		Since:          req.Since,
		Until:          req.Until,
		Limit:          10000,
	}
	if req.MaxEvents > 0 && req.MaxEvents < filter.Limit {
		filter.Limit = req.MaxEvents
	}

	// Fetch events
	events, err := e.log.Query(filter)
	if err != nil {
		return nil, fmt.Errorf("replay: query events: %w", err)
	}

	if len(events) == 0 {
		return &Result{
			Mode:        req.Mode,
			StartedAt:   started,
			CompletedAt: time.Now(),
			DurationMS:  time.Since(started).Milliseconds(),
		}, nil
	}

	// Create kernel with specified policy
	k := kernel.New(e.nodeID, req.Policy)

	// Restore initial state if provided (snapshot+delta replay)
	if req.InitialState != nil {
		k.RestoreState(req.InitialState)
	}

	// Process all events
	var allEffects []kernel.Effect
	for _, evt := range events {
		effects := k.Apply(evt)
		allEffects = append(allEffects, effects...)
	}

	finalState := k.State()
	completed := time.Now()

	result := &Result{
		Mode:            req.Mode,
		EventsProcessed: len(events),
		EffectsProduced: len(allEffects),
		FinalState:      finalState,
		Effects:         allEffects,
		StartedAt:       started,
		CompletedAt:     completed,
		DurationMS:      completed.Sub(started).Milliseconds(),
		FirstSequence:   events[0].SequenceNum,
		LastSequence:    events[len(events)-1].SequenceNum,
	}

	// Verification mode: compare with expected state
	if req.Mode == ModeVerification && req.ExpectedState != nil {
		divergences := compareStates(&finalState, req.ExpectedState)
		result.Divergences = divergences
		result.Verified = len(divergences) == 0
	}

	return result, nil
}

// ─── State Comparison ────────────────────────────────────────────────────────

func compareStates(actual, expected *kernel.State) []Divergence {
	var divs []Divergence

	// Compare node scores
	for id, expectedScore := range expected.NodeScores {
		actualScore, ok := actual.NodeScores[id]
		if !ok {
			divs = append(divs, Divergence{
				Category: "node_score",
				Key:      id,
				Expected: mustJSONStr(expectedScore),
				Actual:   "missing",
				Severity: "error",
			})
			continue
		}
		if actualScore.Classification != expectedScore.Classification {
			divs = append(divs, Divergence{
				Category: "node_score",
				Key:      id + ".classification",
				Expected: expectedScore.Classification,
				Actual:   actualScore.Classification,
				Severity: "warning",
			})
		}
		// Allow small floating point tolerance
		if absDiff(actualScore.CompositeScore, expectedScore.CompositeScore) > 0.001 {
			divs = append(divs, Divergence{
				Category: "node_score",
				Key:      id + ".composite_score",
				Expected: fmt.Sprintf("%.4f", expectedScore.CompositeScore),
				Actual:   fmt.Sprintf("%.4f", actualScore.CompositeScore),
				Severity: "info",
			})
		}
	}

	// Check for extra nodes in actual
	for id := range actual.NodeScores {
		if _, ok := expected.NodeScores[id]; !ok {
			divs = append(divs, Divergence{
				Category: "node_score",
				Key:      id,
				Expected: "missing",
				Actual:   mustJSONStr(actual.NodeScores[id]),
				Severity: "warning",
			})
		}
	}

	// Compare transport scores
	for name, expectedScore := range expected.TransportScores {
		actualScore, ok := actual.TransportScores[name]
		if !ok {
			divs = append(divs, Divergence{
				Category: "transport_score",
				Key:      name,
				Expected: mustJSONStr(expectedScore),
				Actual:   "missing",
				Severity: "error",
			})
			continue
		}
		if actualScore.Classification != expectedScore.Classification {
			divs = append(divs, Divergence{
				Category: "transport_score",
				Key:      name + ".classification",
				Expected: expectedScore.Classification,
				Actual:   actualScore.Classification,
				Severity: "warning",
			})
		}
	}

	// Compare action states
	for id, expectedAction := range expected.ActionStates {
		actualAction, ok := actual.ActionStates[id]
		if !ok {
			divs = append(divs, Divergence{
				Category: "action_state",
				Key:      id,
				Expected: mustJSONStr(expectedAction),
				Actual:   "missing",
				Severity: "error",
			})
			continue
		}
		if actualAction.Lifecycle != expectedAction.Lifecycle {
			divs = append(divs, Divergence{
				Category: "action_state",
				Key:      id + ".lifecycle",
				Expected: expectedAction.Lifecycle,
				Actual:   actualAction.Lifecycle,
				Severity: "error",
			})
		}
	}

	// Compare policy version
	if actual.PolicyVersion != expected.PolicyVersion {
		divs = append(divs, Divergence{
			Category: "policy_version",
			Key:      "policy_version",
			Expected: expected.PolicyVersion,
			Actual:   actual.PolicyVersion,
			Severity: "warning",
		})
	}

	return divs
}

func absDiff(a, b float64) float64 {
	d := a - b
	if d < 0 {
		return -d
	}
	return d
}

func mustJSONStr(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
