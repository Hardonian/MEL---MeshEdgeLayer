/**
 * Incident list / workbench grouping — aligns sort with Command surface (operatorWorkflow).
 */
import type { Incident } from '@/types/api'
import { openIncidentShiftPriority, sortOpenIncidentsForShiftStart } from './operatorWorkflow'

/** Open incidents that should surface before the rest (follow-up, control gates, sparse/degraded). */
export function partitionOpenIncidentsForWorkbench(incidents: Incident[]): {
  needsAttention: Incident[]
  backlog: Incident[]
} {
  const open = incidents.filter((i) => {
    const s = (i.state || '').toLowerCase()
    return s !== 'resolved' && s !== 'closed'
  })
  const sorted = sortOpenIncidentsForShiftStart(open)
  const needsAttention: Incident[] = []
  const backlog: Incident[] = []
  for (const inc of sorted) {
    if (openIncidentShiftPriority(inc) <= 2) needsAttention.push(inc)
    else backlog.push(inc)
  }
  return { needsAttention, backlog }
}
