/**
 * Canonical Incident Decision Pack helpers — prefer backend `decision_pack` over page-local re-derivation.
 */
import type { Incident } from '@/types/api'

/** Workbench/dashboard “why this row” line: uses pack when API provides it. */
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
