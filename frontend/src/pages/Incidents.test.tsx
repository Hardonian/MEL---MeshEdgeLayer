import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { Incidents } from './Incidents'

function renderIncidents() {
  return render(
    <BrowserRouter>
      <Incidents />
    </BrowserRouter>,
  )
}

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
                  wireless_context: {
                    classification: 'wifi_backhaul_instability',
                    primary_domain: 'wifi',
                    observed_domains: ['wifi', 'lora'],
                    evidence_posture: 'partial',
                    confidence_posture: 'evidence_backed',
                    summary: 'Wireless context suggests Wi-Fi/backhaul instability association from transport evidence; this is not root-cause proof.',
                    reasons: [
                      { code: 'wifi_backhaul_terms_present', statement: 'Wi-Fi/backhaul terms appear in incident/evidence text; inspect transport continuity and dead letters.' },
                    ],
                    evidence_gaps: [],
                  },
                  similar_incidents: [{ incident_id: 'inc-old-1' }],
                  action_outcome_memory: [
                    {
                      action_type: 'trigger_health_recheck',
                      action_label: 'trigger health recheck',
                      occurrence_count: 4,
                      sample_size: 4,
                      outcome_framing: 'mixed_historical_evidence',
                      evidence_strength: 'moderate',
                      observed_post_action_status: 'mixed_signals',
                      improvement_observed_count: 2,
                      deterioration_observed_count: 1,
                      inconclusive_count: 1,
                      caveats: ['Historical association only; check concurrent transport changes.'],
                      inspect_before_reuse: ['Confirm signature match before reuse.'],
                    },
                    {
                      action_type: 'restart_transport',
                      action_label: 'restart transport',
                      occurrence_count: 2,
                      sample_size: 2,
                      outcome_framing: 'insufficient_evidence',
                      evidence_strength: 'sparse',
                      observed_post_action_status: 'inconclusive',
                      improvement_observed_count: 1,
                      deterioration_observed_count: 0,
                      inconclusive_count: 1,
                      inspect_before_reuse: ['Verify broker disconnect evidence before reuse.'],
                    },
                  ],
                  action_outcome_snapshots: [
                    {
                      snapshot_id: 'aos-1',
                      signature_key: 'sig-1',
                      incident_id: 'inc-old-1',
                      action_id: 'act-1',
                      action_type: 'trigger_health_recheck',
                      derived_classification: 'mixed_historical_evidence',
                      evidence_sufficiency: 'partial',
                      window_start: '2026-03-28T22:00:00Z',
                      window_end: '2026-03-28T23:00:00Z',
                      pre_action_evidence: { dead_letters_count: 4, transport_alerts_count: 2 },
                      post_action_evidence: { dead_letters_count: 1, transport_alerts_count: 1 },
                      observed_signal_count: 2,
                      caveats: ['partial evidence window'],
                      association_only: true,
                      derived_at: '2026-03-28T23:01:00Z',
                    },
                  ],
                  action_outcome_trace: {
                    expected_snapshot_writes: 2,
                    snapshot_write_failures: 1,
                    snapshot_retrieval_status: 'available',
                    persisted_snapshot_count: 1,
                    completeness: 'partial',
                  },
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
                  degraded: false,
                  sparsity_markers: ['limited_correlated_evidence'],
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
              incident_mutate: true,
            },
            capabilities: ['read_incidents', 'incident_update'],
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
    renderIncidents()
    // Wait for incident intelligence section to appear (open incidents are expanded by default)
    await waitFor(() => {
      expect(screen.getAllByText(/Incident intelligence/i).length).toBeGreaterThan(0)
    })
    // "seen Nx" badge in the header
    expect(screen.getByText(/seen 3x/i)).toBeTruthy()
    // Similar prior incidents in the intelligence section
    expect(screen.getByText(/inc-old-1/i)).toBeTruthy()
    // Historical action outcomes section
    expect(screen.getByText(/Historical action outcomes/i)).toBeTruthy()
    expect(screen.getByText(/does not establish causality/i)).toBeTruthy()
    expect(screen.getByText(/trigger health recheck/i)).toBeTruthy()
    expect(screen.getByText(/n=4/i)).toBeTruthy()
    expect(screen.getByText(/2 improved/i)).toBeTruthy()
    expect(screen.getByText(/Historical association only/i)).toBeTruthy()
    // Snapshot traceability
    expect(screen.getByText(/Snapshot traceability/i)).toBeTruthy()
    expect(screen.getByText(/write failures: 1/i)).toBeTruthy()
    // Wireless context
    expect(screen.getAllByText(/Wireless context/i).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/Wi-Fi backhaul instability/i).length).toBeGreaterThan(0)
    expect(screen.getByText(/Observed domains: wifi, lora/i)).toBeTruthy()
  })

  it('shows sparsity markers without falsely marking intelligence degraded', async () => {
    renderIncidents()
    await waitFor(() => {
      expect(screen.getByText(/Intelligence limited by available evidence/i)).toBeTruthy()
    })
    expect(screen.queryByText(/Intelligence is limited by available evidence/i)).toBeNull()
  })
})
