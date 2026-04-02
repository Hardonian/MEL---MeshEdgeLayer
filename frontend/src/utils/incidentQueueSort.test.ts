import { describe, expect, it } from 'vitest'
import type { Incident } from '@/types/api'
import { compareIncidentsByServerQueueOrder, sortOpenIncidentsByServerQueue } from './incidentQueueSort'

describe('incidentQueueSort', () => {
  it('orders by queue_sort_key_lex ascending', () => {
    const a = {
      id: 'a',
      triage_signals: {
        tier: 4,
        queue_ordering_contract: 'open_incident_workbench_v2',
        queue_sort_key_lex: '4.00000000000000000002.0000000000000001',
      },
    } as Incident
    const b = {
      id: 'b',
      triage_signals: {
        tier: 4,
        queue_ordering_contract: 'open_incident_workbench_v2',
        queue_sort_key_lex: '4.00000000000000000001.0000000000000001',
      },
    } as Incident
    const sorted = sortOpenIncidentsByServerQueue([a, b])
    expect(sorted.map((x) => x.id)).toEqual(['b', 'a'])
  })

  it('compareIncidentsByServerQueueOrder is stable fallback without lex key', () => {
    const x = { id: 'x', triage_signals: { tier: 2, queue_sort_primary: 2 }, updated_at: '2020-01-01T00:00:00Z' } as Incident
    const y = { id: 'y', triage_signals: { tier: 4, queue_sort_primary: 4 }, updated_at: '2025-01-01T00:00:00Z' } as Incident
    expect(compareIncidentsByServerQueueOrder(x, y)).toBeLessThan(0)
  })
})
