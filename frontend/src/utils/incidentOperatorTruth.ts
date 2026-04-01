/**
 * Canonical incident × control × memory semantics for operator surfaces.
 * Single-operator honest: distinguishes observed linkage, references on the record,
 * capability limits, and degraded intelligence — not team workflow or implied certainty.
 */
import type { Incident, IncidentIntelligence, TrustUIHints } from '@/types/api'

/** While panel is loading, assume linked rows are visible to avoid a false “visibility limited” flash. */
export function operatorCanReadLinkedControlRows(state: {
  loading: boolean
  error: string | null
  trustUI: TrustUIHints | null
  capabilities: string[]
}): boolean {
  if (state.loading) return true
  if (state.error) return false
  return state.trustUI?.read_actions === true || state.capabilities.includes('read_actions')
}

export type IncidentActionVisibilityKind =
  | 'visibility_limited'
  | 'linked_observed'
  | 'references_only'
  | 'action_context_degraded'
  | 'no_linked_historical_signals'
  | 'no_linked_observed'

export interface IncidentActionVisibilityContext {
  /** When false, FK-linked rows may be hidden even if they exist — same as trust_ui.read_actions. */
  canReadLinkedActions: boolean
}

export interface IncidentActionVisibility {
  kind: IncidentActionVisibilityKind
  /** Short label for badges and scan lines */
  shortLabel: string
  /** Full sentence for strips and why-lines */
  explanation: string
  /** Open control queue filtered to this incident when true */
  suggestControlQueue: boolean
  linkedCount: number
  awaitingApproval: number
  inFlight: number
  pendingRefCount: number
  recentActionIdCount: number
}

function lifecycleCounts(linked: NonNullable<Incident['linked_control_actions']>) {
  let awaitingApproval = 0
  let inFlight = 0
  for (const a of linked) {
    const ls = (a.lifecycle_state || '').toLowerCase()
    if (ls === 'pending_approval') awaitingApproval++
    else if (ls === 'pending' || ls === 'running') inFlight++
  }
  return { awaitingApproval, inFlight }
}

function actionTraceDegraded(intel: IncidentIntelligence | undefined): boolean {
  const t = intel?.action_outcome_trace
  if (!t) return false
  if (t.snapshot_retrieval_status === 'error' || t.snapshot_retrieval_status === 'unavailable') return true
  if ((t.snapshot_write_failures ?? 0) > 0) return true
  if (t.completeness === 'unavailable') return true
  return false
}

function hasHistoricalActionSignals(intel: IncidentIntelligence | undefined): boolean {
  if (!intel) return false
  if ((intel.action_outcome_memory?.length ?? 0) > 0) return true
  if ((intel.governance_memory?.length ?? 0) > 0) return true
  if ((intel.historically_used_actions?.length ?? 0) > 0) return true
  return false
}

const DEFAULT_CTX: IncidentActionVisibilityContext = { canReadLinkedActions: true }

/**
 * Classifies what the operator can truthfully know about control actions for this incident row.
 */
export function incidentActionVisibility(
  inc: Incident,
  ctx: IncidentActionVisibilityContext = DEFAULT_CTX,
): IncidentActionVisibility {
  const linked = inc.linked_control_actions ?? []
  const pendingRefs = inc.pending_actions?.filter(Boolean) ?? []
  const recentIds = inc.recent_actions?.filter(Boolean) ?? []
  const { awaitingApproval, inFlight } = lifecycleCounts(linked)

  if (!ctx.canReadLinkedActions) {
    return {
      kind: 'visibility_limited',
      shortLabel: 'Control visibility limited',
      explanation:
        'Linked control rows are not shown for this session (read_actions may be off). Absence here does not prove the queue is empty — open the control queue with appropriate credentials.',
      suggestControlQueue: true,
      linkedCount: 0,
      awaitingApproval: 0,
      inFlight: 0,
      pendingRefCount: pendingRefs.length,
      recentActionIdCount: recentIds.length,
    }
  }

  if (linked.length > 0) {
    const parts: string[] = [
      `${linked.length} FK-linked control row${linked.length > 1 ? 's' : ''} on this incident`,
    ]
    if (awaitingApproval > 0) parts.push(`${awaitingApproval} awaiting approval`)
    if (inFlight > 0) parts.push(`${inFlight} queued or executing`)
    return {
      kind: 'linked_observed',
      shortLabel:
        awaitingApproval > 0
          ? `${awaitingApproval} approval wait`
          : inFlight > 0
            ? `${inFlight} in flight`
            : `${linked.length} linked`,
      explanation: `${parts.join(' · ')}. Approval ≠ execution; verify lifecycle on each row.`,
      suggestControlQueue: awaitingApproval > 0 || inFlight > 0,
      linkedCount: linked.length,
      awaitingApproval,
      inFlight,
      pendingRefCount: pendingRefs.length,
      recentActionIdCount: recentIds.length,
    }
  }

  if (pendingRefs.length > 0 || recentIds.length > 0) {
    const bits: string[] = []
    if (pendingRefs.length > 0) bits.push(`${pendingRefs.length} pending action ID${pendingRefs.length > 1 ? 's' : ''} on the record`)
    if (recentIds.length > 0) bits.push(`${recentIds.length} recent action ID${recentIds.length > 1 ? 's' : ''} on the record`)
    return {
      kind: 'references_only',
      shortLabel: 'Action refs only',
      explanation: `${bits.join(' · ')} — no FK-linked rows in this response. Match IDs in the control queue; linkage requires incident_id on the action.`,
      suggestControlQueue: true,
      linkedCount: 0,
      awaitingApproval: 0,
      inFlight: 0,
      pendingRefCount: pendingRefs.length,
      recentActionIdCount: recentIds.length,
    }
  }

  if (actionTraceDegraded(inc.intelligence)) {
    return {
      kind: 'action_context_degraded',
      shortLabel: 'Action memory degraded',
      explanation:
        'Outcome snapshot / trace for this incident is partial or unavailable — do not read “no linked actions” as proof nothing ran; verify the control queue and replay.',
      suggestControlQueue: true,
      linkedCount: 0,
      awaitingApproval: 0,
      inFlight: 0,
      pendingRefCount: 0,
      recentActionIdCount: 0,
    }
  }

  if (hasHistoricalActionSignals(inc.intelligence)) {
    return {
      kind: 'no_linked_historical_signals',
      shortLabel: 'History without linkage',
      explanation:
        'No FK-linked control rows here, but intelligence still carries historical action / governance signals — association only; confirm live queue state before acting.',
      suggestControlQueue: true,
      linkedCount: 0,
      awaitingApproval: 0,
      inFlight: 0,
      pendingRefCount: 0,
      recentActionIdCount: 0,
    }
  }

  return {
    kind: 'no_linked_observed',
    shortLabel: 'No linked actions',
    explanation:
      'No linked control rows, references, or degraded trace flags in this view — if you still expect work in flight, check the control queue.',
    suggestControlQueue: false,
    linkedCount: 0,
    awaitingApproval: 0,
    inFlight: 0,
    pendingRefCount: 0,
    recentActionIdCount: 0,
  }
}

/**
 * Single deterministic scan-line: what prior history should change about the next check (bounded, explainable).
 */
export function incidentMemoryDecisionCue(inc: Incident): string | null {
  const intel = inc.intelligence
  if (!intel) return null

  const mem = intel.action_outcome_memory ?? []
  const gov = intel.governance_memory ?? []
  const hist = intel.historically_used_actions ?? []

  if (inc.reopened_from_incident_id) {
    return 'Reopened — re-verify replay and outcomes before repeating the same control pattern.'
  }
  if (gov.some((g) => g.rejected_count > 0)) {
    return 'Governance memory shows rejections on this action family — confirm policy / approver posture before re-proposing.'
  }
  if (mem.some((m) => (m.inspect_before_reuse?.length ?? 0) > 0)) {
    return 'Outcome memory flags inspect-before-reuse — open operational memory before copying a prior mitigation.'
  }
  const weakOutcomes = new Set([
    'deterioration_observed',
    'insufficient_evidence',
    'no_clear_post_action_signal',
    'mixed_historical_evidence',
  ])
  if (mem.some((m) => weakOutcomes.has(m.outcome_framing))) {
    return 'Historical outcomes skew inconclusive or negative — do not treat past association as a green light for aggressive action.'
  }
  if (
    intel.evidence_strength === 'sparse' &&
    ((intel.signature_match_count ?? 0) > 1 || (intel.similar_incidents?.length ?? 0) > 0)
  ) {
    return 'Sparse evidence but recurring pattern signals — widen replay/topology context before trusting similarity.'
  }
  if (intel.evidence_strength === 'sparse' && hist.length > 0) {
    return 'Sparse evidence with repeated action-type history — treat reuse as experimental; justify with live observations.'
  }
  if (mem.some((m) => (m.sample_size ?? 0) > 0 && (m.sample_size ?? 0) < 3)) {
    return 'Thin sample sizes in outcome memory — patterns are weakly supported statistically.'
  }
  return null
}
