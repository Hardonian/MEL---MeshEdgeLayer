import { describe, expect, it } from 'vitest'
import { parseMeshNodeControlAction, parseOperationalStateJson } from './useApi'

describe('useApi parsers', () => {
  it('preserves enriched control action approval and governance fields', () => {
    const parsed = parseMeshNodeControlAction({
      id: 'act-1',
      result: 'pending_approval',
      action_type: 'suppress_source',
      required_approvals: 2,
      collected_approvals: 1,
      approval_mode: 'multi_party',
      approval_basis: ['blast_radius_high', 'sod'],
      approval_policy_source: 'control.require_approval_for_high_blast_radius',
      high_blast_radius: true,
      approval_escalated_due_to_blast_radius: true,
      incident_id: 'inc-7',
      requires_separate_approver: true,
    })

    expect(parsed).not.toBeNull()
    expect(parsed?.required_approvals).toBe(2)
    expect(parsed?.collected_approvals).toBe(1)
    expect(parsed?.approval_mode).toBe('multi_party')
    expect(parsed?.approval_basis).toEqual(['blast_radius_high', 'sod'])
    expect(parsed?.approval_policy_source).toContain('high_blast_radius')
    expect(parsed?.high_blast_radius).toBe(true)
    expect(parsed?.approval_escalated_due_to_blast_radius).toBe(true)
    expect(parsed?.incident_id).toBe('inc-7')
    expect(parsed?.requires_separate_approver).toBe(true)
  })

  it('parses operational state snapshot fields instead of dropping canon', () => {
    const parsed = parseOperationalStateJson({
      automation_mode: 'frozen',
      freeze_count: 1,
      approval_backlog: 3,
      snapshot_at: '2026-04-02T12:00:00Z',
      queue_metrics: { queue_depth: 2, queue_capacity: 50 },
      executor: { present: false, reason: 'heartbeat_stale' },
      active_freezes: [{ id: 'frz-1', reason: 'operator freeze' }],
      active_maintenance: [{ id: 'mw-1', reason: 'window' }],
      pending_approvals: [{ id: 'act-1', result: 'pending_approval' }],
    })

    expect(parsed.automation_mode).toBe('frozen')
    expect(parsed.freeze_count).toBe(1)
    expect(parsed.approval_backlog).toBe(3)
    expect(parsed.snapshot_at).toBe('2026-04-02T12:00:00Z')
    expect(parsed.queue_metrics).toEqual({ queue_depth: 2, queue_capacity: 50 })
    expect(parsed.executor).toEqual({ present: false, reason: 'heartbeat_stale' })
    expect(parsed.active_freezes?.length).toBe(1)
    expect(parsed.active_maintenance?.length).toBe(1)
    expect(parsed.pending_approvals?.length).toBe(1)
  })
})
