import { describe, expect, it } from 'vitest'
import type { Incident } from '@/types/api'
import { partitionOpenIncidentsForWorkbench } from './incidentWorkbench'

function mk(partial: Partial<Incident> & { id: string }): Incident {
  return {
    id: partial.id,
    state: partial.state ?? 'open',
    review_state: partial.review_state,
    updated_at: partial.updated_at,
    intelligence: partial.intelligence,
    linked_control_actions: partial.linked_control_actions,
    pending_actions: partial.pending_actions,
  } as Incident
}

describe('partitionOpenIncidentsForWorkbench', () => {
  it('puts follow-up and control gates ahead of sparse-only backlog', () => {
    const sparse = mk({
      id: 'sparse',
      updated_at: '2026-01-02T00:00:00Z',
      intelligence: { evidence_strength: 'sparse' } as Incident['intelligence'],
    })
    const follow = mk({
      id: 'follow',
      review_state: 'follow_up_needed',
      updated_at: '2020-01-01T00:00:00Z',
    })
    const approval = mk({
      id: 'appr',
      updated_at: '2019-01-01T00:00:00Z',
      linked_control_actions: [{ id: 'a1', action_type: 'x', lifecycle_state: 'pending_approval' }],
    } as Incident)
    const { needsAttention, backlog } = partitionOpenIncidentsForWorkbench([sparse, follow, approval])
    expect(needsAttention.map((x) => x.id)).toEqual(['follow', 'appr', 'sparse'])
    expect(backlog).toHaveLength(0)
  })
})
