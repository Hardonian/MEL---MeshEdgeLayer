/**
 * Canonical Incident Decision Pack helpers — prefer backend `decision_pack` over page-local re-derivation.
 */
import type { Incident } from '@/types/api'

/** Workbench/dashboard “why this row” line: uses pack when API provides it. */
export function incidentDecisionPackWhyLine(inc: Incident): string | undefined {
  const line = inc.decision_pack?.queue?.why_surfaced_one_liner?.trim()
  return line || undefined
}
