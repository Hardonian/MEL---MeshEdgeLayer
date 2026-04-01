import { describe, expect, it } from 'vitest'
import type { Incident } from '@/types/api'
import {
  incidentActionVisibility,
  incidentMemoryDecisionCue,
  operatorCanReadLinkedControlRows,
  resolvedIncidentActionVisibility,
} from './incidentOperatorTruth'

function inc(partial: Partial<Incident> & { id: string }): Incident {
  return {
    state: 'open',
    ...partial,
  } as Incident
}

describe('resolvedIncidentActionVisibility', () => {
  it('prefers server action_visibility when present', () => {
    const v = resolvedIncidentActionVisibility(
      inc({
        id: 'srv',
        action_visibility: {
          action_visibility_kind: 'linked_observed',
          action_visibility_summary: 'Server says linked.',
          action_context_should_open_control_queue: false,
          action_context_has_material_prior_attempts: true,
          action_context_has_pending_related_work: false,
          action_context_is_partial: false,
          linked_control_row_count: 2,
        },
      }),
      { canReadLinkedActions: true },
    )
    expect(v.kind).toBe('linked_observed')
    expect(v.explanation).toBe('Server says linked.')
    expect(v.linkedCount).toBe(2)
  })
})

describe('incidentActionVisibility', () => {
  it('flags visibility_limited when read_actions is off', () => {
    const v = incidentActionVisibility(inc({ id: 'a' }), { canReadLinkedActions: false })
    expect(v.kind).toBe('visibility_limited')
    expect(v.suggestControlQueue).toBe(true)
    expect(v.explanation).toMatch(/read_actions|credentials/i)
  })

  it('still reports ref counts when visibility is limited', () => {
    const v = incidentActionVisibility(
      inc({ id: 'a', pending_actions: ['x'] }),
      { canReadLinkedActions: false },
    )
    expect(v.pendingRefCount).toBe(1)
    expect(v.kind).toBe('visibility_limited')
  })

  it('linked_observed when FK rows present', () => {
    const v = incidentActionVisibility(
      inc({
        id: 'a',
        linked_control_actions: [
          { id: '1', action_type: 't', lifecycle_state: 'pending_approval' },
        ],
      }),
      { canReadLinkedActions: true },
    )
    expect(v.kind).toBe('linked_observed')
    expect(v.awaitingApproval).toBe(1)
    expect(v.shortLabel).toMatch(/approval/)
  })

  it('references_only when pending IDs but no linked rows', () => {
    const v = incidentActionVisibility(
      inc({ id: 'a', pending_actions: ['act-1'] }),
      { canReadLinkedActions: true },
    )
    expect(v.kind).toBe('references_only')
    expect(v.suggestControlQueue).toBe(true)
  })

  it('action_context_degraded when trace is broken', () => {
    const v = incidentActionVisibility(
      inc({
        id: 'a',
        intelligence: {
          evidence_strength: 'moderate',
          action_outcome_trace: {
            expected_snapshot_writes: 1,
            snapshot_write_failures: 1,
            snapshot_retrieval_status: 'error',
            persisted_snapshot_count: 0,
            completeness: 'partial',
          },
        } as Incident['intelligence'],
      }),
    )
    expect(v.kind).toBe('action_context_degraded')
  })

  it('no_linked_historical_signals when memory exists without linkage', () => {
    const v = incidentActionVisibility(
      inc({
        id: 'a',
        intelligence: {
          evidence_strength: 'strong',
          action_outcome_memory: [
            {
              action_type: 'x',
              outcome_framing: 'improvement_observed',
              evidence_strength: 'moderate',
              sample_size: 2,
            },
          ],
        } as Incident['intelligence'],
      }),
    )
    expect(v.kind).toBe('no_linked_historical_signals')
  })

  it('no_linked_observed when empty signals', () => {
    const v = incidentActionVisibility(
      inc({
        id: 'a',
        intelligence: { evidence_strength: 'moderate' } as Incident['intelligence'],
      }),
    )
    expect(v.kind).toBe('no_linked_observed')
  })
})

describe('operatorCanReadLinkedControlRows', () => {
  it('returns true while panel is loading', () => {
    expect(
      operatorCanReadLinkedControlRows({
        loading: true,
        error: null,
        trustUI: null,
        capabilities: [],
      }),
    ).toBe(true)
  })

  it('returns false on panel error', () => {
    expect(
      operatorCanReadLinkedControlRows({
        loading: false,
        error: 'network',
        trustUI: { read_actions: true } as import('@/types/api').TrustUIHints,
        capabilities: [],
      }),
    ).toBe(false)
  })

  it('honors read_actions in trust_ui', () => {
    expect(
      operatorCanReadLinkedControlRows({
        loading: false,
        error: null,
        trustUI: { read_actions: true } as import('@/types/api').TrustUIHints,
        capabilities: [],
      }),
    ).toBe(true)
  })
})

describe('incidentMemoryDecisionCue', () => {
  it('prioritizes reopen framing', () => {
    const cue = incidentMemoryDecisionCue(
      inc({
        id: 'a',
        reopened_from_incident_id: 'prior',
        intelligence: {
          evidence_strength: 'strong',
          governance_memory: [{ action_type: 't', summary: 'x', linked_action_count: 1, approved_or_passed_count: 0, rejected_count: 2 }],
        } as Incident['intelligence'],
      }),
    )
    expect(cue).toMatch(/Reopened/i)
  })

  it('surfaces governance rejections', () => {
    const cue = incidentMemoryDecisionCue(
      inc({
        id: 'a',
        intelligence: {
          evidence_strength: 'strong',
          governance_memory: [
            {
              action_type: 'restart',
              summary: 'often blocked',
              linked_action_count: 2,
              approved_or_passed_count: 0,
              rejected_count: 1,
            },
          ],
        } as Incident['intelligence'],
      }),
    )
    expect(cue).toMatch(/Governance|rejections/i)
  })

  it('returns null when no strong memory cue', () => {
    expect(
      incidentMemoryDecisionCue(
        inc({ id: 'a', intelligence: { evidence_strength: 'strong' } as Incident['intelligence'] }),
      ),
    ).toBeNull()
  })
})
