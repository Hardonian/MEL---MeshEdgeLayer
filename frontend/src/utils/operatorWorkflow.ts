/**
 * Shared operator-workflow ordering and labels for shift-start surfaces.
 * Single-operator honest: no team queues or implied multi-user coordination.
 */
import type { Incident } from '@/types/api'

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

/** Lower = more urgent for shift-start ordering among open incidents. */
export function openIncidentShiftPriority(inc: Incident): number {
  const rs = (inc.review_state || '').toLowerCase()
  if (FOLLOW_UP_REVIEW.has(rs)) return 0
  if (inc.intelligence?.evidence_strength === 'sparse' || inc.intelligence?.degraded === true) return 1
  if ((inc.intelligence?.signature_match_count ?? 0) > 1) return 2
  return 3
}

export function sortOpenIncidentsForShiftStart(incidents: Incident[]): Incident[] {
  return [...incidents].sort((a, b) => {
    const pa = openIncidentShiftPriority(a)
    const pb = openIncidentShiftPriority(b)
    if (pa !== pb) return pa - pb
    return ts(b.updated_at) - ts(a.updated_at)
  })
}

export function openIncidentShiftWhyLine(inc: Incident): string {
  const rs = (inc.review_state || '').toLowerCase()
  if (FOLLOW_UP_REVIEW.has(rs)) {
    return `Review state “${rs.replace(/_/g, ' ')}” — explicit follow-up or review posture in MEL.`
  }
  if (inc.intelligence?.evidence_strength === 'sparse' || inc.intelligence?.degraded === true) {
    return 'Sparse or degraded intelligence — conclusions stay bounded; gather replay/topology/control context.'
  }
  if ((inc.intelligence?.signature_match_count ?? 0) > 1) {
    return 'Recurring signature on this instance — pattern memory, not proof of repeating root cause.'
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

  const sorted = sortOpenIncidentsForShiftStart(args.openIncidents)
  for (const inc of sorted.slice(0, 12)) {
    const pri = 3 + openIncidentShiftPriority(inc)
    rows.push({
      key: `incident-${inc.id}`,
      priority: pri,
      title: inc.title || `Incident ${inc.id.slice(0, 10)}…`,
      why: openIncidentShiftWhyLine(inc),
      href: `/incidents/${encodeURIComponent(inc.id)}`,
    })
  }

  rows.sort((a, b) => {
    if (a.priority !== b.priority) return a.priority - b.priority
    return a.title.localeCompare(b.title)
  })

  return rows
}
