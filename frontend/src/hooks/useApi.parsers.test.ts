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
      queue_metrics: {
        queued_lifecycle_pending_count: 2,
        awaiting_approval_count: 3,
        approved_waiting_executor_count: 1,
        oldest_queued_pending_created_at: '2026-04-02T11:00:00Z',
        oldest_approved_waiting_executor_created_at: '2026-04-02T11:30:00Z',
      },
      executor: {
        executor_activity: 'inactive',
        executor_last_heartbeat_at: '2026-04-02T11:59:00Z',
        executor_last_reported_kind: 'serve_loop',
        executor_heartbeat_basis: 'control_plane_state',
        executor_presence_note: 'heartbeat stale',
        backlog_requires_active_executor: true,
      },
      active_freezes: [{ id: 'frz-1', reason: 'operator freeze', scope_type: 'global', scope_value: '', created_by: 'ops', created_at: '2026-04-02T10:00:00Z' }],
      active_maintenance: [{ id: 'mw-1', reason: 'window', created_by: 'ops', starts_at: '2026-04-02T12:00:00Z', ends_at: '2026-04-02T13:00:00Z' }],
      pending_approvals: [{ id: 'act-1', result: 'pending_approval' }],
    })

    expect(parsed.automation_mode).toBe('frozen')
    expect(parsed.freeze_count).toBe(1)
    expect(parsed.approval_backlog).toBe(3)
    expect(parsed.snapshot_at).toBe('2026-04-02T12:00:00Z')
    expect(parsed.queue_metrics?.approved_waiting_executor_count).toBe(1)
    expect(parsed.queue_metrics?.awaiting_approval_count).toBe(3)
    expect(parsed.executor?.executor_activity).toBe('inactive')
    expect(parsed.executor?.backlog_requires_active_executor).toBe(true)
    expect(parsed.active_freezes?.[0]?.scope).toBe('global')
    expect(parsed.active_freezes?.length).toBe(1)
    expect(parsed.active_maintenance?.length).toBe(1)
    expect(parsed.pending_approvals?.length).toBe(1)
  })

  it('keeps degraded/unknown posture explicit when nested fields are partial', () => {
    const parsed = parseOperationalStateJson({
      queue_metrics: {},
      executor: { executor_presence_note: 'no heartbeat recorded' },
      active_freezes: [{ reason: 'missing id should be dropped' }],
      active_maintenance: [{ id: 'mw-1' }],
    })

    expect(parsed.queue_metrics).toEqual({
      queued_lifecycle_pending_count: 0,
      awaiting_approval_count: 0,
      approved_waiting_executor_count: 0,
      oldest_queued_pending_created_at: '',
      oldest_approved_waiting_executor_created_at: '',
    })
    expect(parsed.executor).toEqual({
      executor_activity: 'unknown',
      executor_last_heartbeat_at: '',
      executor_last_reported_kind: '',
      executor_heartbeat_basis: '',
      executor_presence_note: 'no heartbeat recorded',
      backlog_requires_active_executor: false,
    })
    expect(parsed.active_freezes).toEqual([])
    expect(parsed.active_maintenance).toEqual([
      { id: 'mw-1', reason: '', actor: '', starts_at: '', ends_at: '' },
    ])
  })
})
