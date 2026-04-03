import { describe, expect, it } from 'vitest'
import { parseReplayViewResponse } from './incidentReplay'

describe('parseReplayViewResponse', () => {
  it('keeps valid replay rows and drops malformed nested rows', () => {
    const parsed = parseReplayViewResponse({
      kind: 'incident_replay',
      incident_id: 'inc-1',
      replay_segments: [
        {
          event_time: '2026-04-03T00:00:00Z',
          event_type: 'incident_created',
          summary: 'created',
          knowledge_posture: 'observed_persisted_event',
          details: { source: 'db' },
          evidence_refs: ['ev-1', 10, 'ev-2'],
        },
        {
          event_time: '2026-04-03T00:01:00Z',
          event_type: 10,
          summary: 'bad',
          knowledge_posture: 'observed_persisted_event',
        },
      ],
      recommendation_outcomes: [
        {
          id: 'o-1',
          recommendation_id: 'r-1',
          outcome: 'improvement_observed',
          created_at: '2026-04-03T00:02:00Z',
        },
        {
          id: 'o-2',
          recommendation_id: 'r-2',
          outcome: 'ignored',
        },
      ],
    })

    expect(parsed).not.toBeNull()
    expect(parsed?.replay_segments).toHaveLength(1)
    expect(parsed?.replay_segments?.[0].evidence_refs).toEqual(['ev-1', 'ev-2'])
    expect(parsed?.recommendation_outcomes).toHaveLength(1)
  })

  it('returns null for non-object and missing canonical fields', () => {
    expect(parseReplayViewResponse('bad')).toBeNull()
    expect(parseReplayViewResponse({ kind: 'incident_replay' })).toBeNull()
  })
})
