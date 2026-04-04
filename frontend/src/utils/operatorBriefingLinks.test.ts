import { describe, expect, it } from 'vitest'
import { hrefForBriefingPriority } from './operatorBriefingLinks'

describe('hrefForBriefingPriority', () => {
  it('links incidents by id', () => {
    expect(
      hrefForBriefingPriority({
        id: 'inc-abc',
        category: 'transport',
        severity: 'high',
        title: 't',
        summary: 's',
        rank: 1,
        confidence: 1,
        evidence_freshness: 'High',
        is_actionable: true,
        blocks_recovery: false,
        resource_kind: 'incident',
      }),
    ).toBe('/incidents/inc-abc')
  })

  it('links transport diagnostics to status hash', () => {
    expect(
      hrefForBriefingPriority({
        id: 'x:y:z',
        category: 'transport',
        severity: 'high',
        title: 't',
        summary: 's',
        rank: 1,
        confidence: 0.9,
        evidence_freshness: 'High',
        is_actionable: true,
        blocks_recovery: false,
        resource_kind: 'diagnostic',
        metadata: { affected_transport: 'mqtt-primary' },
      }),
    ).toBe('/status#mel-transport-mqtt-primary')
  })

  it('falls back to diagnostics when diagnostic has no transport', () => {
    expect(
      hrefForBriefingPriority({
        id: 'code:system:',
        category: 'system',
        severity: 'high',
        title: 't',
        summary: 's',
        rank: 1,
        confidence: 0.9,
        evidence_freshness: 'High',
        is_actionable: true,
        blocks_recovery: true,
        resource_kind: 'diagnostic',
      }),
    ).toBe('/diagnostics')
  })
})
