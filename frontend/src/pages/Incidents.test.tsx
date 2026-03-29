import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { Incidents } from './Incidents'

function setupFetch() {
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo) => {
      const url = typeof input === 'string' ? input : input.url
      if (url.includes('/api/v1/incidents')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            recent_incidents: [
              {
                id: 'inc-1',
                title: 'Transport timeout',
                state: 'open',
                severity: 'warning',
                occurred_at: '2026-03-29T00:00:00Z',
                updated_at: '2026-03-29T01:00:00Z',
                intelligence: {
                  signature_label: 'transport/transport pattern (timeout_stall)',
                  signature_match_count: 3,
                  evidence_strength: 'moderate',
                  similar_incidents: [{ incident_id: 'inc-old-1' }],
                  investigate_next: [
                    {
                      id: 'g-1',
                      title: 'Inspect transport evidence surfaces',
                      rationale: 'Association only.',
                      confidence: 'low',
                    },
                  ],
                },
              },
              {
                id: 'inc-2',
                title: 'Sparse evidence incident',
                state: 'open',
                intelligence: {
                  evidence_strength: 'sparse',
                  degraded: true,
                  degraded_reasons: ['limited_correlated_evidence'],
                },
              },
            ],
          }),
        } as Response)
      }
      if (url.includes('/api/v1/panel')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            trust_ui: {
              incident_handoff_write: true,
            },
            capabilities: ['read_incidents'],
          }),
        } as Response)
      }
      return Promise.reject(new Error(`unexpected fetch ${url}`))
    }),
  )
}

describe('Incidents intelligence rendering', () => {
  beforeEach(() => {
    setupFetch()
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders signature and similarity summary when incident intelligence is available', async () => {
    render(<Incidents />)
    await waitFor(() => {
      expect(screen.getAllByText(/Incident intelligence/i).length).toBeGreaterThan(0)
    })
    expect(screen.getByText(/seen 3 times/i)).toBeTruthy()
    expect(screen.getByText(/Similar prior incidents: inc-old-1/i)).toBeTruthy()
  })

  it('renders degraded warning when evidence is sparse', async () => {
    render(<Incidents />)
    await waitFor(() => {
      expect(screen.getByText(/Intelligence is limited by available evidence/i)).toBeTruthy()
    })
  })
})

