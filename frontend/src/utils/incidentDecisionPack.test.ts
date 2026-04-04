import { describe, expect, it } from 'vitest'
import type { Incident } from '@/types/api'
import {
  guidanceActionPostureLabel,
  guidanceDegradedReasonLabel,
  guidanceEscalationPostureLabel,
  guidanceEvidencePostureLabel,
  guidanceSupportPostureLabel,
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

describe('guidanceEvidencePostureLabel', () => {
  it('maps strong to success variant', () => {
    const r = guidanceEvidencePostureLabel('strong')
    expect(r.variant).toBe('success')
    expect(r.label).toMatch(/strong/i)
  })

  it('maps moderate to warning variant', () => {
    const r = guidanceEvidencePostureLabel('moderate')
    expect(r.variant).toBe('warning')
    expect(r.label).toMatch(/moderate/i)
  })

  it('maps sparse to warning variant', () => {
    const r = guidanceEvidencePostureLabel('sparse')
    expect(r.variant).toBe('warning')
    expect(r.label).toMatch(/sparse/i)
  })

  it('maps degraded to critical variant', () => {
    const r = guidanceEvidencePostureLabel('degraded')
    expect(r.variant).toBe('critical')
    expect(r.label).toMatch(/degraded/i)
  })

  it('maps unknown to secondary variant', () => {
    const r = guidanceEvidencePostureLabel('unknown')
    expect(r.variant).toBe('secondary')
  })

  it('falls back for undefined', () => {
    const r = guidanceEvidencePostureLabel(undefined)
    expect(r.variant).toBe('secondary')
  })
})

describe('guidanceActionPostureLabel', () => {
  it('maps available to success', () => {
    const r = guidanceActionPostureLabel('available')
    expect(r.variant).toBe('success')
    expect(r.label).toMatch(/available/i)
  })

  it('maps guarded to warning', () => {
    const r = guidanceActionPostureLabel('guarded')
    expect(r.variant).toBe('warning')
    expect(r.label).toMatch(/guarded/i)
  })

  it('maps unsupported to secondary', () => {
    const r = guidanceActionPostureLabel('unsupported')
    expect(r.variant).toBe('secondary')
    expect(r.label).toMatch(/limited/i)
  })

  it('maps verify_linkage to outline', () => {
    const r = guidanceActionPostureLabel('verify_linkage')
    expect(r.variant).toBe('outline')
    expect(r.label).toMatch(/verify/i)
  })

  it('falls back for unknown values', () => {
    const r = guidanceActionPostureLabel(undefined)
    expect(r.variant).toBe('outline')
  })
})

describe('guidanceSupportPostureLabel', () => {
  it('maps ready to success', () => {
    const r = guidanceSupportPostureLabel('ready')
    expect(r.variant).toBe('success')
    expect(r.label).toMatch(/ready/i)
  })

  it('maps partial to warning', () => {
    const r = guidanceSupportPostureLabel('partial')
    expect(r.variant).toBe('warning')
    expect(r.label).toMatch(/partial/i)
  })

  it('maps blocked to critical', () => {
    const r = guidanceSupportPostureLabel('blocked')
    expect(r.variant).toBe('critical')
    expect(r.label).toMatch(/blocked/i)
  })

  it('maps unknown to secondary', () => {
    const r = guidanceSupportPostureLabel('unknown')
    expect(r.variant).toBe('secondary')
  })
})

describe('guidanceEscalationPostureLabel', () => {
  it('returns replay_first sentence', () => {
    const s = guidanceEscalationPostureLabel('replay_first')
    expect(s).toMatch(/replay/i)
    expect(s.length).toBeGreaterThan(0)
  })

  it('returns follow_up sentence', () => {
    const s = guidanceEscalationPostureLabel('follow_up')
    expect(s).toMatch(/follow/i)
  })

  it('returns bounded_review sentence', () => {
    const s = guidanceEscalationPostureLabel('bounded_review')
    expect(s).toMatch(/bounded/i)
  })

  it('returns empty string for undefined', () => {
    expect(guidanceEscalationPostureLabel(undefined)).toBe('')
  })
})

describe('guidanceDegradedReasonLabel', () => {
  it('maps known codes to readable labels', () => {
    expect(guidanceDegradedReasonLabel('no_intelligence')).toMatch(/no intelligence/i)
    expect(guidanceDegradedReasonLabel('incident_intelligence_degraded')).toMatch(/intelligence degraded/i)
    expect(guidanceDegradedReasonLabel('action_visibility_limited')).toMatch(/action visibility/i)
    expect(guidanceDegradedReasonLabel('export_policy_degraded')).toMatch(/export policy/i)
  })

  it('strips replay_ prefix for replay reason codes', () => {
    const label = guidanceDegradedReasonLabel('replay_no_history')
    expect(label).toMatch(/^Replay:/i)
    expect(label).toMatch(/no history/i)
  })

  it('falls back gracefully for unknown codes', () => {
    const label = guidanceDegradedReasonLabel('some_unknown_code')
    expect(label).toBe('some unknown code')
  })
})
