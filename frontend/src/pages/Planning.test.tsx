import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { Planning } from './Planning'

describe('Planning', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo) => {
        const url = typeof input === 'string' ? input : input.url
        if (url.includes('/api/v1/planning/bundle')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({
              evidence_model:
                'Directional validation only — not RF coverage; before/after can be inconclusive if baseline is missing or confounded.',
              transport_connected: false,
              topology_enabled: true,
              resilience: {
                resilience_score: 0.5,
                redundancy_score: 0.4,
                partition_risk_score: 0.3,
                fragility_explanation: [],
                next_best_move_summary: 'observe',
                confidence: { level: 'medium', score: 0.5 },
              },
              node_profiles: [],
              ranked_next_plans: [],
              playbooks: [],
              limits: ['No RF map'],
              computed_at: '2026-01-01T00:00:00Z',
            }),
          } as Response)
        }
        if (url.includes('/api/v1/planning/advisory-alerts')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({ alerts: [] }),
          } as Response)
        }
        return Promise.reject(new Error(`unexpected fetch: ${url}`))
      }),
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders evidence-model uncertainty banner', async () => {
    render(<Planning />)
    await waitFor(() => {
      expect(screen.getByTestId('planning-evidence-banner')).toBeTruthy()
    })
    expect(screen.getByTestId('planning-evidence-banner').textContent).toMatch(/inconclusive|Directional/i)
  })

  it('empty advisories state explains uncertainty', async () => {
    render(<Planning />)
    await waitFor(() => {
      expect(screen.getByTestId('planning-advisories-empty')).toBeTruthy()
    })
    expect(screen.getByTestId('planning-advisories-empty').textContent).toMatch(/does not prove/)
  })
})
