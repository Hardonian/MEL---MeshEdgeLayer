import { describe, it, expect } from 'vitest'
import { normalizeReplayEvents } from './replay'

describe('normalizeReplayEvents', () => {
  it('classifies event class and origin without fabricating certainty', () => {
    const events = normalizeReplayEvents([
      {
        event_type: 'transport_disconnect',
        summary: 'broker dropped',
        source: 'observed ingest',
        confidence: 0.4,
      },
      {
        event_type: 'state_projection',
        statement: 'derived state',
        inferred: true,
        confidence_label: 'medium',
      },
    ])

    expect(events[0].eventClass).toBe('transport')
    expect(events[0].origin).toBe('observed')
    expect(events[0].confidenceLabel).toBe('low')

    expect(events[1].eventClass).toBe('state')
    expect(events[1].origin).toBe('derived')
    expect(events[1].summary).toBe('derived state')
  })
})
