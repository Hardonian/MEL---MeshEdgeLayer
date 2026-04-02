/**
 * Incident list / workbench grouping — aligns sort with Command surface (operatorWorkflow).
 */
import type { Incident } from '@/types/api'
import { incidentDecisionPackNeedsAttention } from './incidentDecisionPack'
import {
  openIncidentShiftPriority,
  sortOpenIncidentsForShiftStart,
  type IncidentWorkQueueWhyContext,
} from './operatorWorkflow'

/** Open incidents that should surface before the rest (follow-up, control gates, sparse/degraded). */
export function partitionOpenIncidentsForWorkbench(
  incidents: Incident[],
  ctx?: IncidentWorkQueueWhyContext,
): {
  needsAttention: Incident[]
  backlog: Incident[]
} {
  const open = incidents.filter((i) => {
    const s = (i.state || '').toLowerCase()
    return s !== 'resolved' && s !== 'closed'
  })
  const sorted = sortOpenIncidentsForShiftStart(open, ctx)
  const needsAttention: Incident[] = []
  const backlog: Incident[] = []
  for (const inc of sorted) {
    const canonical = incidentDecisionPackNeedsAttention(inc)
    if (canonical != null) {
      if (canonical) needsAttention.push(inc)
      else backlog.push(inc)
      continue
    }
    if (openIncidentShiftPriority(inc, ctx) <= 2) needsAttention.push(inc)
    else backlog.push(inc)
  }
  return { needsAttention, backlog }
}
