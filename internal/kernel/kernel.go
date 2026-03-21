package kernel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Kernel is the deterministic core of MEL. It processes events and produces
// effects without executing side effects directly.
//
// Invariants:
//   - Given identical event streams and policy, the kernel produces identical state.
//   - The kernel never performs I/O directly.
//   - All outputs are Effect values collected by the caller.
//   - The kernel is safe for concurrent read access to state but requires
//     exclusive access during Apply.
type Kernel struct {
	mu    sync.RWMutex
	state *State
	// nodeID identifies this MEL instance.
	nodeID string
	// policy holds the current policy configuration affecting decisions.
	policy Policy
	// handlers maps event types to processing functions.
	handlers map[EventType]EventHandler
}

// Policy captures the kernel-relevant policy configuration.
type Policy struct {
	Version                  string   `json:"version"`
	Mode                     string   `json:"mode"` // disabled, advisory, guarded_auto
	AllowedActions           []string `json:"allowed_actions"`
	RequireMinConfidence     float64  `json:"require_min_confidence"`
	MaxActionsPerWindow      int      `json:"max_actions_per_window"`
	CooldownPerTargetSeconds int      `json:"cooldown_per_target_seconds"`
	ActionWindowSeconds      int      `json:"action_window_seconds"`
	AllowMeshLevelActions    bool     `json:"allow_mesh_level_actions"`
}

// EventHandler processes a single event against the current state and returns
// effects. Handlers MUST be deterministic.
type EventHandler func(state *State, event Event, policy Policy) []Effect

// New creates a new Kernel with the given node ID and policy.
func New(nodeID string, policy Policy) *Kernel {
	k := &Kernel{
		state:    NewState(),
		nodeID:   nodeID,
		policy:   policy,
		handlers: make(map[EventType]EventHandler),
	}
	k.registerDefaultHandlers()
	return k
}

// NodeID returns this kernel's node identifier.
func (k *Kernel) NodeID() string { return k.nodeID }

// State returns a read-only copy of the current kernel state.
func (k *Kernel) State() State {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.copyState()
}

// StatePtr returns a pointer to the internal state for snapshot purposes.
// Caller must hold appropriate locks.
func (k *Kernel) StatePtr() *State {
	k.mu.RLock()
	defer k.mu.RUnlock()
	cp := k.copyState()
	return &cp
}

// Apply processes a single event through the kernel and returns the resulting
// effects. This is the primary entry point for event processing.
//
// Apply is NOT safe for concurrent calls. The caller must serialize event
// application (which matches the append-only log ordering requirement).
func (k *Kernel) Apply(event Event) []Effect {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Update logical clock (Lamport: max(local, remote) + 1)
	if event.LogicalClock > k.state.LogicalClock {
		k.state.LogicalClock = event.LogicalClock
	}
	k.state.LogicalClock++

	// Track last processed event
	k.state.LastEventID = event.ID
	k.state.LastSequenceNum = event.SequenceNum

	// Dispatch to type-specific handler
	handler, ok := k.handlers[event.Type]
	if !ok {
		// Unknown event type: record but produce no effects
		return nil
	}

	return handler(k.state, event, k.policy)
}

// ApplyBatch processes multiple events in order and returns all effects.
func (k *Kernel) ApplyBatch(events []Event) []Effect {
	var allEffects []Effect
	for _, evt := range events {
		effects := k.Apply(evt)
		allEffects = append(allEffects, effects...)
	}
	return allEffects
}

// UpdatePolicy replaces the current policy. This is itself an event-worthy
// operation; the caller should also append an EventPolicyChange to the log.
func (k *Kernel) UpdatePolicy(policy Policy) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.policy = policy
	k.state.PolicyVersion = policy.Version
}

// RestoreState replaces the kernel state entirely (for snapshot restoration).
func (k *Kernel) RestoreState(state *State) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.state = state
}

// ─── Default Event Handlers ──────────────────────────────────────────────────

func (k *Kernel) registerDefaultHandlers() {
	k.handlers[EventObservation] = handleObservation
	k.handlers[EventAnomaly] = handleAnomaly
	k.handlers[EventTopologyUpdate] = handleTopologyUpdate
	k.handlers[EventTransportHealth] = handleTransportHealth
	k.handlers[EventNodeState] = handleNodeState
	k.handlers[EventActionProposed] = handleActionProposed
	k.handlers[EventActionExecuted] = handleActionExecuted
	k.handlers[EventActionCompleted] = handleActionCompleted
	k.handlers[EventFreezeCreated] = handleFreezeCreated
	k.handlers[EventFreezeCleared] = handleFreezeCleared
	k.handlers[EventPolicyChange] = handlePolicyChange
	k.handlers[EventOperatorAction] = handleOperatorAction
	k.handlers[EventPeerJoined] = handlePeerJoined
	k.handlers[EventPeerLeft] = handlePeerLeft
	k.handlers[EventRegionHealth] = handleRegionHealth
	k.handlers[EventAdapterState] = handleAdapterState
	k.handlers[EventApproval] = handleApproval
	k.handlers[EventRejection] = handleRejection
	k.handlers[EventMaintenanceStart] = handleMaintenanceStart
	k.handlers[EventMaintenanceEnd] = handleMaintenanceEnd
	k.handlers[EventSyncReceived] = handleSyncReceived
	k.handlers[EventSnapshotCreated] = handleSnapshotCreated
}

// ─── Observation Handler ──────────────────────────────────────────────────────

type ObservationData struct {
	Transport   string  `json:"transport"`
	NodeNum     int64   `json:"node_num,omitempty"`
	NodeID      string  `json:"node_id,omitempty"`
	MessageType string  `json:"message_type,omitempty"`
	SNR         float64 `json:"snr,omitempty"`
	RSSI        int64   `json:"rssi,omitempty"`
	PayloadSize int     `json:"payload_size,omitempty"`
}

func handleObservation(state *State, event Event, policy Policy) []Effect {
	var obs ObservationData
	if err := json.Unmarshal(event.Data, &obs); err != nil {
		return nil
	}

	var effects []Effect

	// Update transport score
	ts, ok := state.TransportScores[obs.Transport]
	if !ok {
		ts = TransportScore{
			Transport:   obs.Transport,
			HealthScore: 1.0,
		}
	}
	// Successful observation improves health slightly
	ts.HealthScore = clampFloat(ts.HealthScore*0.99+0.01, 0, 1)
	ts.ReliabilityPct = clampFloat(ts.ReliabilityPct*0.99+0.01, 0, 1)
	ts.UpdatedAt = event.Timestamp
	ts.EventID = event.ID
	ts.Classification = classifyTransportHealth(ts.HealthScore)
	state.TransportScores[obs.Transport] = ts

	// Update node score if we have a node reference
	if obs.NodeID != "" {
		ns, ok := state.NodeScores[obs.NodeID]
		if !ok {
			ns = NodeScore{
				NodeID:      obs.NodeID,
				Transport:   obs.Transport,
				HealthScore: 1.0,
				TrustScore:  0.5, // neutral starting trust
			}
		}
		ns.ActivityScore = clampFloat(ns.ActivityScore*0.95+0.05, 0, 1)
		ns.HealthScore = clampFloat(ns.HealthScore*0.99+0.01, 0, 1)
		ns.CompositeScore = computeComposite(ns)
		ns.Classification = classifyNodeHealth(ns.CompositeScore)
		ns.UpdatedAt = event.Timestamp
		ns.EventID = event.ID
		state.NodeScores[obs.NodeID] = ns

		effects = append(effects, Effect{
			ID:       NewEffectID(),
			Type:     EffectUpdateScore,
			CausedBy: event.ID,
			Subject:  obs.NodeID,
			Data:     mustJSON(ns),
		})
	}

	// Update node registry
	if obs.NodeNum != 0 {
		info := state.NodeRegistry[obs.NodeNum]
		info.NodeNum = obs.NodeNum
		if obs.NodeID != "" {
			info.NodeID = obs.NodeID
		}
		info.LastSeen = event.Timestamp
		if event.SourceRegion != "" {
			info.Region = event.SourceRegion
		}
		state.NodeRegistry[obs.NodeNum] = info
	}

	return effects
}

// ─── Anomaly Handler ──────────────────────────────────────────────────────────

type AnomalyData struct {
	Transport string  `json:"transport"`
	NodeID    string  `json:"node_id,omitempty"`
	Category  string  `json:"category"`
	Severity  string  `json:"severity"`
	Detail    string  `json:"detail"`
	Score     float64 `json:"score"`
}

func handleAnomaly(state *State, event Event, policy Policy) []Effect {
	var anomaly AnomalyData
	if err := json.Unmarshal(event.Data, &anomaly); err != nil {
		return nil
	}

	var effects []Effect

	// Degrade transport score
	ts, ok := state.TransportScores[anomaly.Transport]
	if !ok {
		ts = TransportScore{Transport: anomaly.Transport, HealthScore: 0.8}
	}
	degradation := 0.05 * anomaly.Score
	ts.HealthScore = clampFloat(ts.HealthScore-degradation, 0, 1)
	ts.AnomalyRate = clampFloat(ts.AnomalyRate*0.9+0.1*anomaly.Score, 0, 1)
	ts.UpdatedAt = event.Timestamp
	ts.EventID = event.ID
	ts.Classification = classifyTransportHealth(ts.HealthScore)
	state.TransportScores[anomaly.Transport] = ts

	// Degrade node score if applicable
	if anomaly.NodeID != "" {
		ns := state.NodeScores[anomaly.NodeID]
		ns.NodeID = anomaly.NodeID
		ns.AnomalyScore = clampFloat(ns.AnomalyScore*0.8+0.2*anomaly.Score, 0, 1)
		ns.HealthScore = clampFloat(ns.HealthScore-degradation*0.5, 0, 1)
		ns.CompositeScore = computeComposite(ns)
		ns.Classification = classifyNodeHealth(ns.CompositeScore)
		ns.UpdatedAt = event.Timestamp
		ns.EventID = event.ID
		state.NodeScores[anomaly.NodeID] = ns
	}

	// Propose action if conditions met
	if policy.Mode != "disabled" && ts.HealthScore < 0.3 {
		actionEffect := proposeActionEffect(event, anomaly.Transport, "trigger_health_recheck",
			fmt.Sprintf("transport %s health degraded to %.2f", anomaly.Transport, ts.HealthScore),
			ts.HealthScore, policy)
		if actionEffect != nil {
			effects = append(effects, *actionEffect)
		}
	}

	effects = append(effects, Effect{
		ID:       NewEffectID(),
		Type:     EffectEmitAlert,
		CausedBy: event.ID,
		Subject:  anomaly.Transport,
		Data:     event.Data,
	})

	return effects
}

// ─── Topology Handler ─────────────────────────────────────────────────────────

type TopologyData struct {
	NodeNum   int64  `json:"node_num"`
	NodeID    string `json:"node_id"`
	LongName  string `json:"long_name"`
	ShortName string `json:"short_name"`
	Region    string `json:"region,omitempty"`
	Action    string `json:"action"` // joined, left, updated
}

func handleTopologyUpdate(state *State, event Event, _ Policy) []Effect {
	var topo TopologyData
	if err := json.Unmarshal(event.Data, &topo); err != nil {
		return nil
	}

	switch topo.Action {
	case "left":
		delete(state.NodeRegistry, topo.NodeNum)
		delete(state.NodeScores, topo.NodeID)
	default: // joined, updated
		info := state.NodeRegistry[topo.NodeNum]
		info.NodeNum = topo.NodeNum
		info.NodeID = topo.NodeID
		info.LongName = topo.LongName
		info.ShortName = topo.ShortName
		info.LastSeen = event.Timestamp
		if topo.Region != "" {
			info.Region = topo.Region
		}
		state.NodeRegistry[topo.NodeNum] = info
	}

	return nil
}

// ─── Transport Health Handler ─────────────────────────────────────────────────

type TransportHealthData struct {
	Transport string  `json:"transport"`
	State     string  `json:"state"`
	Health    float64 `json:"health"`
	Detail    string  `json:"detail"`
}

func handleTransportHealth(state *State, event Event, policy Policy) []Effect {
	var th TransportHealthData
	if err := json.Unmarshal(event.Data, &th); err != nil {
		return nil
	}

	ts := state.TransportScores[th.Transport]
	ts.Transport = th.Transport
	ts.HealthScore = th.Health
	ts.Classification = classifyTransportHealth(th.Health)
	ts.UpdatedAt = event.Timestamp
	ts.EventID = event.ID
	state.TransportScores[th.Transport] = ts

	var effects []Effect
	if policy.Mode != "disabled" && th.Health < 0.2 {
		actionEffect := proposeActionEffect(event, th.Transport, "restart_transport",
			fmt.Sprintf("transport %s critically degraded (health=%.2f)", th.Transport, th.Health),
			1.0-th.Health, policy)
		if actionEffect != nil {
			effects = append(effects, *actionEffect)
		}
	}
	return effects
}

// ─── Node State Handler ──────────────────────────────────────────────────────

func handleNodeState(state *State, event Event, _ Policy) []Effect {
	var info NodeInfo
	if err := json.Unmarshal(event.Data, &info); err != nil {
		return nil
	}
	info.LastSeen = event.Timestamp
	state.NodeRegistry[info.NodeNum] = info
	return nil
}

// ─── Action Lifecycle Handlers ───────────────────────────────────────────────

type ActionProposedData struct {
	ActionID   string  `json:"action_id"`
	ActionType string  `json:"action_type"`
	Target     string  `json:"target"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
}

func handleActionProposed(state *State, event Event, _ Policy) []Effect {
	var ap ActionProposedData
	if err := json.Unmarshal(event.Data, &ap); err != nil {
		return nil
	}
	state.ActionStates[ap.ActionID] = ActionState{
		ActionID:    ap.ActionID,
		ActionType:  ap.ActionType,
		Target:      ap.Target,
		Lifecycle:   "proposed",
		ProposedAt:  event.Timestamp,
		OwnerNodeID: event.SourceNodeID,
	}
	return nil
}

type ActionExecutedData struct {
	ActionID string `json:"action_id"`
	Result   string `json:"result"`
}

func handleActionExecuted(state *State, event Event, _ Policy) []Effect {
	var ae ActionExecutedData
	if err := json.Unmarshal(event.Data, &ae); err != nil {
		return nil
	}
	as, ok := state.ActionStates[ae.ActionID]
	if !ok {
		return nil
	}
	as.Lifecycle = "running"
	as.ExecutedAt = event.Timestamp
	state.ActionStates[ae.ActionID] = as
	return nil
}

type ActionCompletedData struct {
	ActionID string `json:"action_id"`
	Result   string `json:"result"`
}

func handleActionCompleted(state *State, event Event, _ Policy) []Effect {
	var ac ActionCompletedData
	if err := json.Unmarshal(event.Data, &ac); err != nil {
		return nil
	}
	as, ok := state.ActionStates[ac.ActionID]
	if !ok {
		return nil
	}
	as.Lifecycle = "completed"
	as.CompletedAt = event.Timestamp
	as.Result = ac.Result
	state.ActionStates[ac.ActionID] = as
	return nil
}

// ─── Freeze Handlers ─────────────────────────────────────────────────────────

type FreezeData struct {
	FreezeID   string `json:"freeze_id"`
	ScopeType  string `json:"scope_type"`
	ScopeValue string `json:"scope_value"`
	Reason     string `json:"reason"`
	ExpiresAt  string `json:"expires_at,omitempty"`
}

func handleFreezeCreated(state *State, event Event, _ Policy) []Effect {
	var fd FreezeData
	if err := json.Unmarshal(event.Data, &fd); err != nil {
		return nil
	}
	fs := FreezeState{
		FreezeID:   fd.FreezeID,
		ScopeType:  fd.ScopeType,
		ScopeValue: fd.ScopeValue,
		Reason:     fd.Reason,
		CreatedAt:  event.Timestamp,
	}
	if fd.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, fd.ExpiresAt); err == nil {
			fs.ExpiresAt = t
		}
	}
	state.ActiveFreezes[fd.FreezeID] = fs
	return nil
}

func handleFreezeCleared(state *State, event Event, _ Policy) []Effect {
	var fd FreezeData
	if err := json.Unmarshal(event.Data, &fd); err != nil {
		return nil
	}
	delete(state.ActiveFreezes, fd.FreezeID)
	return nil
}

// ─── Policy Change Handler ───────────────────────────────────────────────────

func handlePolicyChange(state *State, event Event, _ Policy) []Effect {
	var p Policy
	if err := json.Unmarshal(event.Data, &p); err != nil {
		return nil
	}
	state.PolicyVersion = p.Version
	return nil
}

// ─── Stub Handlers (record event, no effects) ───────────────────────────────

func handleOperatorAction(_ *State, _ Event, _ Policy) []Effect  { return nil }
func handlePeerJoined(_ *State, _ Event, _ Policy) []Effect      { return nil }
func handlePeerLeft(_ *State, _ Event, _ Policy) []Effect        { return nil }
func handleRegionHealth(_ *State, _ Event, _ Policy) []Effect    { return nil }
func handleAdapterState(_ *State, _ Event, _ Policy) []Effect    { return nil }
func handleApproval(_ *State, _ Event, _ Policy) []Effect        { return nil }
func handleRejection(_ *State, _ Event, _ Policy) []Effect       { return nil }
func handleMaintenanceStart(_ *State, _ Event, _ Policy) []Effect { return nil }
func handleMaintenanceEnd(_ *State, _ Event, _ Policy) []Effect  { return nil }
func handleSyncReceived(_ *State, _ Event, _ Policy) []Effect    { return nil }
func handleSnapshotCreated(_ *State, _ Event, _ Policy) []Effect { return nil }

// ─── Helpers ─────────────────────────────────────────────────────────────────

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func computeComposite(ns NodeScore) float64 {
	// Weighted: health 0.4, trust 0.2, activity 0.2, inverse-anomaly 0.2
	return clampFloat(
		ns.HealthScore*0.4+ns.TrustScore*0.2+ns.ActivityScore*0.2+(1.0-ns.AnomalyScore)*0.2,
		0, 1,
	)
}

func classifyNodeHealth(composite float64) string {
	switch {
	case composite >= 0.8:
		return "healthy"
	case composite >= 0.5:
		return "degraded"
	case composite >= 0.2:
		return "failing"
	default:
		return "dead"
	}
}

func classifyTransportHealth(health float64) string {
	switch {
	case health >= 0.8:
		return "healthy"
	case health >= 0.5:
		return "degraded"
	case health >= 0.2:
		return "failing"
	default:
		return "dead"
	}
}

func proposeActionEffect(event Event, target, actionType, reason string, confidence float64, policy Policy) *Effect {
	// Check if action type is allowed
	allowed := false
	for _, a := range policy.AllowedActions {
		if a == actionType {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil
	}

	if confidence < policy.RequireMinConfidence {
		return nil
	}

	data := ActionProposedData{
		ActionID:   NewEventID(),
		ActionType: actionType,
		Target:     target,
		Reason:     reason,
		Confidence: confidence,
	}

	return &Effect{
		ID:       NewEffectID(),
		Type:     EffectProposeAction,
		CausedBy: event.ID,
		Subject:  target,
		Data:     mustJSON(data),
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (k *Kernel) copyState() State {
	cp := State{
		NodeScores:      make(map[string]NodeScore, len(k.state.NodeScores)),
		TransportScores: make(map[string]TransportScore, len(k.state.TransportScores)),
		ActionStates:    make(map[string]ActionState, len(k.state.ActionStates)),
		ActiveFreezes:   make(map[string]FreezeState, len(k.state.ActiveFreezes)),
		NodeRegistry:    make(map[int64]NodeInfo, len(k.state.NodeRegistry)),
		RegionHealth:    make(map[string]RegionHealthState, len(k.state.RegionHealth)),
		PolicyVersion:   k.state.PolicyVersion,
		LastEventID:     k.state.LastEventID,
		LastSequenceNum: k.state.LastSequenceNum,
		LogicalClock:    k.state.LogicalClock,
	}
	for k2, v := range k.state.NodeScores {
		cp.NodeScores[k2] = v
	}
	for k2, v := range k.state.TransportScores {
		cp.TransportScores[k2] = v
	}
	for k2, v := range k.state.ActionStates {
		cp.ActionStates[k2] = v
	}
	for k2, v := range k.state.ActiveFreezes {
		cp.ActiveFreezes[k2] = v
	}
	for k2, v := range k.state.NodeRegistry {
		cp.NodeRegistry[k2] = v
	}
	for k2, v := range k.state.RegionHealth {
		cp.RegionHealth[k2] = v
	}
	return cp
}

// ComputeChecksum computes the integrity checksum for an event.
func ComputeChecksum(e Event) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%d|%s|%s|", e.ID, e.SequenceNum, e.Type, e.Timestamp.Format(time.RFC3339Nano))
	h.Write(e.Data)
	return hex.EncodeToString(h.Sum(nil))
}
