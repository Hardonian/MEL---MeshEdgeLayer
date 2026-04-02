import { describe, expect, it } from 'vitest'
import type { Incident } from '@/types/api'
import {
  incidentDecisionPackNeedsAttention,
  incidentDecisionPackPriorityTier,
  incidentDecisionPackWhyLine,
} from './incidentDecisionPack'
import { openIncidentShiftWhyLine } from './operatorWorkflow'

describe('incidentDecisionPack', () => {
  it('exposes why line from decision_pack', () => {
    const inc = {
      id: 'i1',
      review_state: 'open',
      decision_pack: {
        schema_version: '1',
        incident_id: 'i1',
        generated_at: '2026-01-01T00:00:00Z',
        queue: { why_surfaced_one_liner: 'Pack canonical why line.' },
      },
    } as Incident
    expect(incidentDecisionPackWhyLine(inc)).toBe('Pack canonical why line.')
    expect(openIncidentShiftWhyLine(inc)).toBe('Pack canonical why line.')
  })

  it('prefers guidance why/tier/attention flags when provided', () => {
    const inc = {
      id: 'i2',
      decision_pack: {
        schema_version: '1',
        incident_id: 'i2',
        generated_at: '2026-01-01T00:00:00Z',
        guidance: { why_now: 'Canonical guidance reason', priority_tier: 1, needs_attention: true },
        queue: { why_surfaced_one_liner: 'Legacy queue why' },
      },
    } as Incident
    expect(incidentDecisionPackWhyLine(inc)).toBe('Canonical guidance reason')
    expect(incidentDecisionPackPriorityTier(inc)).toBe(1)
    expect(incidentDecisionPackNeedsAttention(inc)).toBe(true)
  })
})
