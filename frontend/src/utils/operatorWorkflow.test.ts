import { describe, expect, it } from 'vitest'
import type { Incident } from '@/types/api'
import {
  buildRecurrenceHomeTeasers,
  buildShiftStartAttentionRows,
  countOpenIncidentsExplicitFollowUp,
  openIncidentShiftPriority,
  openIncidentShiftWhyLine,
  sortOpenIncidentsForShiftStart,
} from './operatorWorkflow'

function inc(partial: Partial<Incident> & { id: string }): Incident {
  return {
    id: partial.id,
    title: partial.title,
    state: partial.state ?? 'open',
    review_state: partial.review_state,
    updated_at: partial.updated_at,
    intelligence: partial.intelligence,
    linked_control_actions: partial.linked_control_actions,
    pending_actions: partial.pending_actions,
  } as Incident
}

describe('operatorWorkflow', () => {
  it('prioritizes follow-up review states', () => {
    const a = inc({ id: 'a', review_state: 'follow_up_needed' })
    const b = inc({ id: 'b', review_state: 'investigating' })
    expect(openIncidentShiftPriority(a)).toBeLessThan(openIncidentShiftPriority(b))
  })

  it('sortOpenIncidentsForShiftStart orders by priority then updated_at', () => {
    const old = inc({
      id: 'old',
      review_state: 'investigating',
      updated_at: '2020-01-01T00:00:00Z',
    })
    const newSparse = inc({
      id: 'newSparse',
      review_state: 'investigating',
      updated_at: '2025-01-02T00:00:00Z',
      intelligence: { evidence_strength: 'sparse' } as Incident['intelligence'],
    })
    const follow = inc({
      id: 'follow',
      review_state: 'follow_up_needed',
      updated_at: '2019-01-01T00:00:00Z',
    })
    const sorted = sortOpenIncidentsForShiftStart([old, newSparse, follow])
    expect(sorted.map((x) => x.id)).toEqual(['follow', 'newSparse', 'old'])
  })

  it('openIncidentShiftWhyLine mentions recurring without causal claim', () => {
    const i = inc({
      id: 'r',
      intelligence: { signature_match_count: 3 } as Incident['intelligence'],
    })
    expect(openIncidentShiftWhyLine(i)).toContain('Recurring')
    expect(openIncidentShiftWhyLine(i)).toContain('not proof')
  })

  it('openIncidentShiftWhyLine surfaces linked approval wait before sparse intel', () => {
    const sparseOnly = inc({
      id: 's',
      intelligence: { evidence_strength: 'sparse' } as Incident['intelligence'],
    })
    const withApproval = inc({
      id: 'a',
      intelligence: { evidence_strength: 'sparse' } as Incident['intelligence'],
      linked_control_actions: [{ id: 'x', action_type: 't', lifecycle_state: 'pending_approval' }],
    } as Incident)
    expect(openIncidentShiftWhyLine(withApproval)).toMatch(/awaiting approval/i)
    expect(openIncidentShiftWhyLine(sparseOnly)).toMatch(/Sparse or degraded/i)
  })

  it('openIncidentShiftWhyLine mentions export policy when disabled', () => {
    const i = inc({ id: 'e' })
    expect(openIncidentShiftWhyLine(i, { exportEnabled: false })).toMatch(/policy disables evidence export/i)
  })

  it('openIncidentShiftWhyLine explains visibility when read_actions is off', () => {
    const i = inc({ id: 'v' })
    expect(openIncidentShiftWhyLine(i, { canReadLinkedActions: false })).toMatch(/not shown for this session/i)
  })

  it('sortOpenIncidentsForShiftStart elevates history-without-linkage before plain backlog', () => {
    const plain = inc({
      id: 'plain',
      updated_at: '2026-01-02T00:00:00Z',
      review_state: 'investigating',
    })
    const histOnly = inc({
      id: 'hist',
      updated_at: '2020-01-01T00:00:00Z',
      review_state: 'investigating',
      intelligence: {
        evidence_strength: 'strong',
        historically_used_actions: [{ action_type: 'restart', count: 2 }],
      } as Incident['intelligence'],
    })
    const sorted = sortOpenIncidentsForShiftStart([plain, histOnly])
    expect(sorted[0]?.id).toBe('hist')
  })

  it('countOpenIncidentsExplicitFollowUp counts review workflow states', () => {
    const n = countOpenIncidentsExplicitFollowUp([
      inc({ id: 'a', review_state: 'follow_up_needed' }),
      inc({ id: 'b', review_state: 'investigating' }),
    ])
    expect(n).toBe(1)
  })

  it('buildRecurrenceHomeTeasers ranks signature and similar cases', () => {
    const a = inc({
      id: 'a',
      intelligence: {
        evidence_strength: 'moderate',
        signature_match_count: 2,
        similar_incidents: [{ incident_id: 'x' }],
      } as Incident['intelligence'],
    })
    const b = inc({
      id: 'b',
      intelligence: {
        evidence_strength: 'strong',
        signature_match_count: 5,
      } as Incident['intelligence'],
    })
    const teasers = buildRecurrenceHomeTeasers([a, b], 2)
    expect(teasers[0]?.id).toBe('b')
    expect(teasers[0]?.why).toContain('signature')
  })

  it('buildShiftStartAttentionRows surfaces transports before incidents', () => {
    const rows = buildShiftStartAttentionRows({
      openIncidents: [inc({ id: 'x', title: 'X' })],
      unhealthyTransportCount: 1,
      degradedTransportCount: 0,
      criticalDiagCount: 0,
      criticalPrivacyCount: 0,
      pendingApprovalCount: 0,
      deadLetterCount: 0,
      deadLettersIncreasedSinceBaseline: false,
      sparseOpenCount: 0,
    })
    expect(rows[0]?.key).toBe('transport-unhealthy')
  })
})
