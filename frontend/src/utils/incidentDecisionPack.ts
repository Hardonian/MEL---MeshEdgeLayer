/**
 * Canonical Incident Decision Pack helpers — prefer backend `decision_pack` over page-local re-derivation.
 */
import type { Incident, IncidentDecisionPackGuidance } from '@/types/api'

export type SemanticBadgeVariant = 'success' | 'warning' | 'critical' | 'secondary' | 'outline'

/** Workbench/dashboard "why this row" line: uses pack when API provides it. */
export function incidentDecisionPackWhyLine(inc: Incident): string | undefined {
  const guidanceWhy = inc.decision_pack?.guidance?.why_now?.trim()
  if (guidanceWhy) return guidanceWhy
  const line = inc.decision_pack?.queue?.why_surfaced_one_liner?.trim()
  return line || undefined
}

/** Canonical deterministic queue tier from decision_pack guidance when present. */
export function incidentDecisionPackPriorityTier(inc: Incident): number | undefined {
  const tier = inc.decision_pack?.guidance?.priority_tier
  if (typeof tier !== 'number' || Number.isNaN(tier)) return undefined
  if (tier < 0 || tier > 4) return undefined
  return tier
}

/** Canonical "needs attention" flag from backend guidance contract. */
export function incidentDecisionPackNeedsAttention(inc: Incident): boolean | undefined {
  const v = inc.decision_pack?.guidance?.needs_attention
  return typeof v === 'boolean' ? v : undefined
}

/** Label and badge variant for backend-computed action posture. */
export function guidanceActionPostureLabel(posture: IncidentDecisionPackGuidance['action_posture'] | undefined): {
  label: string
  variant: SemanticBadgeVariant
} {
  switch (posture) {
    case 'available':
      return { label: 'Actions available', variant: 'success' }
    case 'guarded':
      return { label: 'Guarded — verify before acting', variant: 'warning' }
    case 'unsupported':
      return { label: 'Action visibility limited', variant: 'secondary' }
    case 'verify_linkage':
      return { label: 'Verify action linkage', variant: 'outline' }
    default:
      return { label: 'Action posture unknown', variant: 'outline' }
  }
}

/** Label and badge variant for backend-computed evidence posture. */
export function guidanceEvidencePostureLabel(posture: IncidentDecisionPackGuidance['evidence_posture'] | undefined): {
  label: string
  variant: SemanticBadgeVariant
} {
  switch (posture) {
    case 'strong':
      return { label: 'Strong evidence', variant: 'success' }
    case 'moderate':
      return { label: 'Moderate evidence', variant: 'warning' }
    case 'sparse':
      return { label: 'Sparse evidence', variant: 'warning' }
    case 'degraded':
      return { label: 'Evidence degraded', variant: 'critical' }
    case 'unknown':
      return { label: 'Evidence unknown', variant: 'secondary' }
    default: {
      const p = posture as string | undefined
      return { label: p ? p.replace(/_/g, ' ') : 'Evidence unknown', variant: 'secondary' }
    }
  }
}

/** Label and badge variant for backend-computed support/export posture. */
export function guidanceSupportPostureLabel(posture: IncidentDecisionPackGuidance['support_posture'] | undefined): {
  label: string
  variant: SemanticBadgeVariant
} {
  switch (posture) {
    case 'ready':
      return { label: 'Export/support ready', variant: 'success' }
    case 'partial':
      return { label: 'Partial export/support', variant: 'warning' }
    case 'blocked':
      return { label: 'Export/support blocked', variant: 'critical' }
    case 'unknown':
      return { label: 'Export posture unknown', variant: 'secondary' }
    default: {
      const p = posture as string | undefined
      return { label: p ? p.replace(/_/g, ' ') : 'Support posture unknown', variant: 'outline' }
    }
  }
}

/**
 * Human-readable escalation posture sentence — empty string when not actionable.
 * Returns operator-truth framing, not a command.
 */
export function guidanceEscalationPostureLabel(
  posture: IncidentDecisionPackGuidance['escalation_posture'] | undefined,
): string {
  switch (posture) {
    case 'replay_first':
      return 'Escalation posture: replay first — compare timeline before deciding next action'
    case 'follow_up':
      return 'Escalation posture: follow-up (reopened or prior case linked)'
    case 'bounded_review':
      return 'Escalation posture: bounded review'
    default:
      return ''
  }
}

/** Degraded reason codes mapped to operator-readable strings. */
export function guidanceDegradedReasonLabel(code: string): string {
  const map: Record<string, string> = {
    no_intelligence: 'No intelligence assembled',
    incident_intelligence_degraded: 'Intelligence degraded',
    action_visibility_limited: 'Action visibility limited',
    action_context_degraded: 'Action context degraded',
    export_policy_degraded: 'Export policy degraded',
    export_policy_unknown_partial: 'Export policy unknown or partial',
  }
  if (map[code]) return map[code]
  if (code.startsWith('replay_')) return `Replay: ${code.slice(7).replace(/_/g, ' ')}`
  return code.replace(/_/g, ' ')
}
