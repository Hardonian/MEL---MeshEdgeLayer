/**
 * Server-owned open workbench ordering — use triage_signals.queue_sort_key_lex when present.
 * Falls back only when API omits contract (legacy clients).
 */
import type { Incident, IncidentTriageSignals } from '@/types/api'

function ts(s: string | undefined): number {
  if (!s) return 0
  const t = new Date(s).getTime()
  return Number.isFinite(t) ? t : 0
}

function hasV2Contract(inc: Incident): boolean {
  return (inc.triage_signals?.queue_ordering_contract || '').startsWith('open_incident_workbench_v2')
}

function hasLexSortKey(inc: Incident): boolean {
  return !!inc.triage_signals?.queue_sort_key_lex
}

/** Ascending: earlier in queue = higher priority (lower tier, more recent updated_at, stable tie-break). */
export function compareIncidentsByServerQueueOrder(a: Incident, b: Incident): number {
  const ka = a.triage_signals?.queue_sort_key_lex
  const kb = b.triage_signals?.queue_sort_key_lex
  const eitherV2 = hasV2Contract(a) || hasV2Contract(b)
  if (eitherV2) {
    const aHasLex = hasLexSortKey(a)
    const bHasLex = hasLexSortKey(b)
    if (aHasLex && !bHasLex) return -1
    if (!aHasLex && bHasLex) return 1
  }
  if (ka && kb) {
    return ka.localeCompare(kb)
  }
  return fallbackOpenIncidentCompare(a, b)
}

function fallbackOpenIncidentCompare(a: Incident, b: Incident): number {
  const pa = a.triage_signals?.queue_sort_primary ?? a.triage_signals?.tier ?? 4
  const pb = b.triage_signals?.queue_sort_primary ?? b.triage_signals?.tier ?? 4
  if (pa !== pb) return pa - pb
  return ts(b.updated_at) - ts(a.updated_at)
}

export function sortOpenIncidentsByServerQueue(incidents: Incident[]): Incident[] {
  return [...incidents].sort(compareIncidentsByServerQueueOrder)
}

/** True when list response includes v2 ordering — callers may skip local tier re-derivation. */
export function hasServerQueueOrdering(sig: IncidentTriageSignals | undefined): boolean {
  return !!sig?.queue_ordering_contract?.startsWith('open_incident_workbench_v2') && !!sig.queue_sort_key_lex
}

/** v2 contract rows missing lex key indicate partial queue semantics and should be surfaced as degraded. */
export function countV2QueueRowsMissingLex(incidents: Incident[]): number {
  return incidents.filter((inc) => hasV2Contract(inc) && !hasLexSortKey(inc)).length
}
