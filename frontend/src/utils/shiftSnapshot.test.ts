import { describe, it, expect } from 'vitest'
import { computeShiftDelta, type ShiftSnapshot } from './shiftSnapshot'
import type { AuditLog, Incident } from '@/types/api'

function inc(id: string, updated: string): Incident {
  return {
    id,
    state: 'open',
    title: 't',
    updated_at: updated,
    occurred_at: '2026-01-01T00:00:00Z',
  }
}

describe('computeShiftDelta', () => {
  it('returns empty delta when no baseline', () => {
    const d = computeShiftDelta(null, {
      incidents: [inc('a', '2026-01-02T00:00:00Z')],
      nodes: [],
      transports: [],
      events: [],
      messageCount: 1,
      deadLetterCount: 0,
    })
    expect(d.incidentsTouchedSince).toHaveLength(0)
    expect(d.newAuditEvents).toBe(0)
  })

  it('detects incident updates after baseline time', () => {
    const prev: ShiftSnapshot = {
      savedAt: '2026-01-01T12:00:00Z',
      openIncidentIds: ['a'],
      nodeLastSeen: {},
      transportHeartbeatMax: null,
      eventMaxTime: null,
      messageCountApprox: 0,
      deadLetterCount: 0,
    }
    const d = computeShiftDelta(prev, {
      incidents: [inc('a', '2026-01-02T00:00:00Z')],
      nodes: [],
      transports: [],
      events: [],
      messageCount: 0,
      deadLetterCount: 0,
    })
    expect(d.incidentsTouchedSince.map((x) => x.id)).toContain('a')
  })

  it('counts audit events newer than snapshot event max', () => {
    const prev: ShiftSnapshot = {
      savedAt: '2026-01-01T00:00:00Z',
      openIncidentIds: [],
      nodeLastSeen: {},
      transportHeartbeatMax: null,
      eventMaxTime: '2026-01-01T10:00:00Z',
      messageCountApprox: 0,
      deadLetterCount: 0,
    }
    const events: Pick<AuditLog, 'created_at'>[] = [
      { created_at: '2026-01-01T11:00:00Z' },
      { created_at: '2026-01-01T12:00:00Z' },
    ]
    const d = computeShiftDelta(prev, {
      incidents: [],
      nodes: [],
      transports: [],
      events,
      messageCount: 0,
      deadLetterCount: 0,
    })
    expect(d.newAuditEvents).toBe(2)
  })
})
