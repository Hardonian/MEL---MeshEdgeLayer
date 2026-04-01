/**
 * Canonical operator-facing language for evidence and uncertainty across MEL surfaces.
 * Maps API enums/strings to consistent labels — does not invent stronger claims than the backend.
 */

export type EvidenceStrength = 'sparse' | 'moderate' | 'strong'

/** Badge variant aligned with existing Badge component (subset used here). */
export type SemanticBadgeVariant = 'success' | 'warning' | 'secondary' | 'outline' | 'critical'

export function evidenceStrengthLabel(s: EvidenceStrength | string | undefined): string {
  switch (s) {
    case 'strong':
      return 'Strong evidence (for this incident)'
    case 'moderate':
      return 'Moderate evidence'
    case 'sparse':
      return 'Sparse evidence'
    default:
      return s ? String(s).replace(/_/g, ' ') : 'Evidence strength unknown'
  }
}

export function evidenceStrengthBadgeVariant(s: EvidenceStrength | string | undefined): SemanticBadgeVariant {
  if (s === 'strong') return 'success'
  if (s === 'moderate') return 'warning'
  if (s === 'sparse') return 'secondary'
  return 'outline'
}

export function wirelessEvidencePostureLabel(
  posture: string | undefined,
): { label: string; variant: SemanticBadgeVariant } {
  const p = (posture || '').toLowerCase()
  switch (p) {
    case 'live':
      return { label: 'Live observations (as stored)', variant: 'success' }
    case 'historical':
      return { label: 'Historical / not proven live now', variant: 'secondary' }
    case 'partial':
      return { label: 'Partial visibility', variant: 'warning' }
    case 'sparse':
      return { label: 'Sparse observations', variant: 'warning' }
    case 'imported':
      return { label: 'Imported / offline bundle context', variant: 'outline' }
    case 'unsupported':
      return { label: 'Unsupported domain for automated ingest', variant: 'secondary' }
    default:
      return { label: posture ? posture.replace(/_/g, ' ') : 'Unknown evidence posture', variant: 'outline' }
  }
}

export function wirelessConfidencePostureLabel(
  posture: string | undefined,
): { label: string; variant: SemanticBadgeVariant } {
  const p = (posture || '').toLowerCase()
  switch (p) {
    case 'evidence_backed':
      return { label: 'Claims tied to stored evidence', variant: 'success' }
    case 'mixed':
      return { label: 'Mixed evidence quality', variant: 'warning' }
    case 'sparse':
      return { label: 'Sparse — verify before acting', variant: 'warning' }
    case 'inconclusive':
      return { label: 'Inconclusive — no strong directional read', variant: 'secondary' }
    default:
      return { label: posture ? posture.replace(/_/g, ' ') : 'Unknown confidence posture', variant: 'outline' }
  }
}

/** Topology node: observed recency vs inferred graph health (no RF implication). */
export function topologyNodeTruthSummary(args: {
  stale: boolean
  health_state?: string
  last_seen_at?: string
}): string {
  if (args.stale || args.health_state === 'stale') {
    return 'Stale: last observation older than configured window — not “offline proof”.'
  }
  if (args.health_state === 'inferred_only' || args.health_state === 'weakly_observed') {
    return 'Graph health partly from inferred or weakly observed edges; drill down for factors.'
  }
  return 'Recent observations within stale window; health from stored scoring factors.'
}

/** Link edge: observed packet-derived vs inferred path. */
export function topologyLinkTruthSummary(args: { observed: boolean; stale: boolean; relay_dependent: boolean }): string {
  const parts: string[] = []
  parts.push(args.observed ? 'Observed path (packet fields)' : 'Inferred edge (not direct observation)')
  if (args.stale) parts.push('stale')
  if (args.relay_dependent) parts.push('relay-dependent visibility')
  return parts.join(' · ')
}

export function runbookStrengthOperatorLabel(strength: string | undefined): string {
  const s = (strength || '').toLowerCase()
  switch (s) {
    case 'historically_proven':
    case 'proven_historically':
      return 'Historically seen on this instance (not causal proof)'
    case 'historically_promising':
      return 'Historically promising — small sample or mixed outcomes'
    case 'plausible':
      return 'Plausible — verify preconditions'
    case 'weakly_supported':
      return 'Weakly supported by history'
    case 'unsupported':
      return 'Unsupported by local history — treat as manual judgment'
    default:
      return strength ? strength.replace(/_/g, ' ') : 'Unknown strength'
  }
}

export function guidanceConfidenceLabel(c: 'low' | 'medium' | undefined): string {
  if (c === 'medium') return 'Medium — still verify against live transport and incident state'
  if (c === 'low') return 'Low — checklist only; confirm with evidence'
  return 'Unspecified confidence'
}
