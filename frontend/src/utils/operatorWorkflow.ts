/**
 * Shared operator-workflow ordering and labels for shift-start surfaces.
 * Single-operator honest: no team queues or implied multi-user coordination.
 */
import type { Incident } from '@/types/api'
import { incidentDecisionPackWhyLine } from './incidentDecisionPack'
import { resolvedIncidentActionVisibility } from './incidentOperatorTruth'
import { compareIncidentsByServerQueueOrder, hasServerQueueOrdering } from './incidentQueueSort'

const FOLLOW_UP_REVIEW = new Set(['follow_up_needed', 'pending_review', 'mitigated'])

export function openIncidentExplicitFollowUp(inc: Incident): boolean {
  return FOLLOW_UP_REVIEW.has((inc.review_state || '').toLowerCase())
}

export function countOpenIncidentsExplicitFollowUp(incidents: Incident[]): number {
  return incidents.filter(openIncidentExplicitFollowUp).length
}

function ts(s: string | undefined): number {
  if (!s) return 0
  const t = new Date(s).getTime()
  return Number.isFinite(t) ? t : 0
}

function linkedActionsAwaitingApproval(inc: Incident): number {
  const linked = inc.linked_control_actions ?? []
  return linked.filter((a) => (a.lifecycle_state || '').toLowerCase() === 'pending_approval').length
}

function referencedPendingActionCount(inc: Incident): number {
  return inc.pending_actions?.filter(Boolean).length ?? 0
}

export interface IncidentWorkQueueWhyContext {
  /** When false, incident evidence export is disabled by policy — proofpack/escalation likely blocked. */
  exportEnabled?: boolean
  /** Version/policy endpoint failed — export gates unknown. */
  exportPolicyUnknown?: boolean
  /** When false, FK-linked control rows are not shown — list view may look “empty” while queue has work. */
  canReadLinkedActions?: boolean
}

function triageTierFromIncident(inc: Incident): number | null {
  const sig = inc.triage_signals
  const t = sig?.tier
  if (typeof t !== 'number' || Number.isNaN(t)) return null
  if (t < 0 || t > 4) return null
  const rs = (inc.review_state || '').toLowerCase()
  const isFollowUp = FOLLOW_UP_REVIEW.has(rs)
  if (isFollowUp && t !== 0) return null
  if (!isFollowUp && t === 0) return null
  if (!sig?.queue_ordering_contract) return null
  // v2: server owns ordering; trust tier when lex key present.
  if (hasServerQueueOrdering(sig)) {
    return t
  }
  if (sig.queue_sort_primary !== t) return null
  return t
}

/** Lower = more urgent for shift-start and incident workbench ordering among open incidents. */
export function openIncidentShiftPriority(inc: Incident, ctx?: IncidentWorkQueueWhyContext): number {
  const fromAPI = triageTierFromIncident(inc)
  if (fromAPI != null) {
    return fromAPI
  }
  const rs = (inc.review_state || '').toLowerCase()
  if (FOLLOW_UP_REVIEW.has(rs)) return 0
  const canRead = ctx?.canReadLinkedActions !== false
  const vis = resolvedIncidentActionVisibility(inc, { canReadLinkedActions: canRead })
  const awaiting = canRead ? linkedActionsAwaitingApproval(inc) : 0
  const pend = referencedPendingActionCount(inc)
  if (awaiting > 0 || pend > 0) return 1
  if (vis.kind === 'visibility_limited' || vis.kind === 'action_context_degraded') return 2
  if (vis.kind === 'no_linked_historical_signals') return 3
  if (inc.intelligence?.evidence_strength === 'sparse' || inc.intelligence?.degraded === true) return 2
  if ((inc.intelligence?.signature_match_count ?? 0) > 1) return 3
  return 4
}

export function sortOpenIncidentsForShiftStart(incidents: Incident[], ctx?: IncidentWorkQueueWhyContext): Incident[] {
  const anyV2 = incidents.some((i) => hasServerQueueOrdering(i.triage_signals))
  if (anyV2) {
    return [...incidents].sort(compareIncidentsByServerQueueOrder)
  }
  return [...incidents].sort((a, b) => {
    const pa = openIncidentShiftPriority(a, ctx)
    const pb = openIncidentShiftPriority(b, ctx)
    if (pa !== pb) return pa - pb
    return ts(b.updated_at) - ts(a.updated_at)
  })
}

export function openIncidentShiftWhyLine(inc: Incident, ctx?: IncidentWorkQueueWhyContext): string {
  const packWhy = incidentDecisionPackWhyLine(inc)
  if (packWhy) return packWhy
  const rs = (inc.review_state || '').toLowerCase()
  if (FOLLOW_UP_REVIEW.has(rs)) {
    return `Review state “${rs.replace(/_/g, ' ')}” — explicit follow-up or review posture in MEL.`
  }
  const sig = inc.triage_signals
  if (sig?.codes?.length) {
    const pick = ['governance_friction_memory', 'mitigation_durability_weak_in_family', 'sparse_or_degraded_intel']
    for (const code of pick) {
      if (sig.codes.includes(code)) {
        const idx = sig.codes.indexOf(code)
        const line = sig.rationale_lines?.[idx]
        if (line) {
          return `${line} (triage code: ${code.replace(/_/g, ' ')} — inspect incident.triage_signals in API for full basis).`
        }
      }
    }
  }
  const md = inc.intelligence?.mitigation_durability_memory
  if (md?.summary && (md.posture === 'reopened_after_resolution_in_family' || md.posture === 'deterioration_or_mixed_in_outcome_memory')) {
    return `${md.summary} (${md.uncertainty.replace(/_/g, ' ')} — see incident.intelligence.mitigation_durability_memory).`
  }
  const canRead = ctx?.canReadLinkedActions !== false
  const vis = resolvedIncidentActionVisibility(inc, { canReadLinkedActions: canRead })
  if (vis.kind === 'visibility_limited') {
    return vis.explanation
  }
  if (vis.kind === 'action_context_degraded') {
    return vis.explanation
  }
  if (vis.kind === 'no_linked_historical_signals') {
    return vis.explanation
  }
  const awaiting = canRead ? linkedActionsAwaitingApproval(inc) : 0
  if (awaiting > 0) {
    return `${awaiting} linked control action${awaiting > 1 ? 's' : ''} awaiting approval — approval ≠ execution; check the control queue.`
  }
  const pend = referencedPendingActionCount(inc)
  if (pend > 0) {
    return `${pend} referenced pending action ID${pend > 1 ? 's' : ''} on the incident record — verify they match the queue and linkage.`
  }
  if (inc.intelligence?.evidence_strength === 'sparse' || inc.intelligence?.degraded === true) {
    return 'Sparse or degraded intelligence — conclusions stay bounded; gather replay/topology/control context.'
  }
  if ((inc.intelligence?.signature_match_count ?? 0) > 1) {
    return 'Recurring signature on this instance — pattern memory, not proof of repeating root cause.'
  }
  if (inc.reopened_from_incident_id) {
    return 'Reopened incident — prior case id is in record; compare replay and outcomes before reusing the same control pattern.'
  }
  if (ctx?.exportEnabled === false) {
    return 'Instance policy disables evidence export — plan plain handoff and runtime/diagnostics continuity, not proofpack-first.'
  }
  if (ctx?.exportPolicyUnknown) {
    return 'Export policy not loaded — confirm Settings/runtime truth before choosing proofpack or escalation.'
  }
  if (vis.kind === 'no_linked_observed') {
    return `${vis.explanation} Verify replay and topology before stronger claims.`
  }
  return 'Open in workflow — verify state against replay and exports before stronger claims.'
}

export interface ShiftStartAttentionRow {
  key: string
  /** Lower sorts first */
  priority: number
  title: string
  why: string
  href: string
}

export function buildShiftStartAttentionRows(args: {
  openIncidents: Incident[]
  unhealthyTransportCount: number
  degradedTransportCount: number
  criticalDiagCount: number
  criticalPrivacyCount: number
  pendingApprovalCount: number
  deadLetterCount: number
  deadLettersIncreasedSinceBaseline: boolean
  sparseOpenCount: number
  /** When set, incident rows use the same export-gate hints as the incident workbench. */
  incidentWhyContext?: IncidentWorkQueueWhyContext
}): ShiftStartAttentionRow[] {
  const rows: ShiftStartAttentionRow[] = []

  if (args.unhealthyTransportCount > 0) {
    rows.push({
      key: 'transport-unhealthy',
      priority: 0,
      title: `${args.unhealthyTransportCount} unhealthy transport${args.unhealthyTransportCount > 1 ? 's' : ''}`,
      why: 'Ingest or broker path may be failing — verify before trusting live counters.',
      href: '/status',
    })
  }
  if (args.criticalDiagCount > 0) {
    rows.push({
      key: 'diagnostics-critical',
      priority: 0,
      title: `${args.criticalDiagCount} critical/high diagnostic finding${args.criticalDiagCount > 1 ? 's' : ''}`,
      why: 'Runtime or config posture flagged — may affect evidence quality or exports.',
      href: '/diagnostics',
    })
  }
  if (args.criticalPrivacyCount > 0) {
    rows.push({
      key: 'privacy-critical',
      priority: 0,
      title: `${args.criticalPrivacyCount} critical/high privacy finding${args.criticalPrivacyCount > 1 ? 's' : ''}`,
      why: 'Policy or data-handling risk surfaced — review before wider handoff.',
      href: '/privacy',
    })
  }
  if (args.pendingApprovalCount > 0) {
    rows.push({
      key: 'approvals',
      priority: 1,
      title: `${args.pendingApprovalCount} control action${args.pendingApprovalCount > 1 ? 's' : ''} awaiting approval`,
      why: 'Trusted control is gated — approval ≠ execution; check queue for incident linkage.',
      href: '/control-actions',
    })
  }
  if (args.deadLetterCount > 0) {
    rows.push({
      key: 'dead-letters',
      priority: args.deadLettersIncreasedSinceBaseline ? 1 : 2,
      title: `${args.deadLetterCount} dead letter${args.deadLetterCount > 1 ? 's' : ''}`,
      why: args.deadLettersIncreasedSinceBaseline
        ? 'Count increased since your saved baseline — processing failures deserve a pass.'
        : 'Failed message processing — may explain gaps in timeline or intelligence.',
      href: '/dead-letters',
    })
  }
  if (args.degradedTransportCount > 0 && args.unhealthyTransportCount === 0) {
    rows.push({
      key: 'transport-degraded',
      priority: 2,
      title: `${args.degradedTransportCount} degraded transport${args.degradedTransportCount > 1 ? 's' : ''}`,
      why: 'Connected but impaired — evidence may be partial or delayed.',
      href: '/status',
    })
  }

  const sorted = sortOpenIncidentsForShiftStart(args.openIncidents, args.incidentWhyContext)
  for (const inc of sorted.slice(0, 12)) {
    const pri = 4 + openIncidentShiftPriority(inc, args.incidentWhyContext)
    const workbenchReturn = `/incidents?focus=${encodeURIComponent(inc.id)}&section=open`
    rows.push({
      key: `incident-${inc.id}`,
      priority: pri,
      title: inc.title || `Incident ${inc.id.slice(0, 10)}…`,
      why: openIncidentShiftWhyLine(inc, args.incidentWhyContext),
      href: `/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(workbenchReturn)}`,
    })
  }

  rows.sort((a, b) => {
    if (a.priority !== b.priority) return a.priority - b.priority
    return a.title.localeCompare(b.title)
  })

  return rows
}

/** Open incidents with the strongest deterministic recurrence / memory signals for operator home. */
export interface RecurrenceHomeTeaser {
  id: string
  title: string
  why: string
}

export function buildRecurrenceHomeTeasers(incidents: Incident[], limit = 4): RecurrenceHomeTeaser[] {
  const open = incidents.filter((i) => {
    const s = (i.state || '').toLowerCase()
    return s !== 'resolved' && s !== 'closed'
  })
  const scored = open
    .map((i) => {
      const intel = i.intelligence
      const sig = intel?.signature_match_count ?? 0
      const sim = intel?.similar_incidents?.length ?? 0
      const mem = intel?.action_outcome_memory?.length ?? 0
      const gov = intel?.governance_memory?.length ?? 0
      const fam = i.intelligence?.signature_family_resolved_history
    const reopenStress =
      fam && fam.resolved_peer_count >= 2 && fam.reopened_peer_count >= 1 ? 400 : 0
    const score = (sig > 1 ? sig * 1000 : 0) + sim * 50 + mem * 30 + gov * 20 + reopenStress
      return { i, score, sig, sim, mem }
    })
    .filter((x) => x.score > 0)
    .sort((a, b) => {
      if (b.score !== a.score) return b.score - a.score
      return ts(b.i.updated_at) - ts(a.i.updated_at)
    })

  const out: RecurrenceHomeTeaser[] = []
  for (const { i, sig, sim, mem } of scored.slice(0, limit)) {
    const parts: string[] = []
    if (sig > 1) parts.push(`signature seen ${sig}× on this instance (bucket, not root cause)`)
    if (sim > 0) parts.push(`${sim} similar prior case${sim > 1 ? 's' : ''} linked in intelligence`)
    if (mem > 0) parts.push(`${mem} historical action outcome row${mem > 1 ? 's' : ''}`)
    const md = i.intelligence?.mitigation_durability_memory
    if (md?.posture === 'reopened_after_resolution_in_family' || md?.posture === 'deterioration_or_mixed_in_outcome_memory') {
      parts.push(`mitigation durability: ${md.posture.replace(/_/g, ' ')}`)
    }
    out.push({
      id: i.id,
      title: i.title || i.id.slice(0, 12),
      why: parts.join(' · ') || 'Pattern memory surfaced — open detail for rationale.',
    })
  }
  return out
}

/** Graph focus node derived from incident resource / implicated domains — not RF certainty. */
export function incidentTopologyFocusNodeNum(inc: Incident): number | null {
  const rt = (inc.resource_type || '').toLowerCase()
  const rid = (inc.resource_id || '').trim()
  if (rt === 'mesh_node' || rt === 'node') {
    const n = parseInt(rid.replace(/\D/g, '') || '0', 10)
    if (Number.isFinite(n) && n > 0) return n
  }
  for (const d of inc.intelligence?.implicated_domains ?? []) {
    if ((d.domain || '').toLowerCase() !== 'mesh_topology') continue
    for (const ref of d.evidence_refs ?? []) {
      const m = /^node[:_]?(\d+)$/i.exec(ref.trim())
      if (m) {
        const n = parseInt(m[1], 10)
        if (Number.isFinite(n) && n > 0) return n
      }
    }
  }
  return null
}
