import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { Planning } from './Planning'

interface PlanningScenario {
  evidenceModel?: string
  meshAssessmentId?: string
  advisories?: Array<{ id: string; severity: string; reason: string; summary: string }>
  limits?: string[]
  confidenceLevel?: string
  evidenceClassification?: string
  uncertaintyNotes?: string[]
  evidenceFlags?: Record<string, boolean>
  advisoryEvidenceFlags?: Record<string, boolean>
}

function setupPlanningFetch(scenario: PlanningScenario = {}) {
  const bundle = {
    evidence_model:
      scenario.evidenceModel ??
      'Directional validation only — not RF coverage; before/after can be inconclusive if baseline is missing or confounded.',
    mesh_assessment_id: scenario.meshAssessmentId,
    transport_connected: false,
    topology_enabled: true,
    evidence_flags: scenario.evidenceFlags ?? {},
    resilience: {
      resilience_score: 0.5,
      redundancy_score: 0.4,
      partition_risk_score: 0.3,
      fragility_explanation: [],
      next_best_move_summary: 'observe',
      confidence: { level: scenario.confidenceLevel ?? 'medium', score: 0.5 },
    },
    best_next_move: {
      title: 'Hold rollout',
      summary_lines: ['Need cleaner before/after separation'],
      primary_verdict: 'hold',
      evidence_classification: scenario.evidenceClassification ?? 'directional_only',
      uncertainty_notes: scenario.uncertaintyNotes ?? [],
    },
    node_profiles: [],
    ranked_next_plans: [],
    playbooks: [],
    limits: scenario.limits ?? ['No RF map'],
    computed_at: '2026-01-01T00:00:00Z',
  }

  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo) => {
      const url = typeof input === 'string' ? input : input.url
      if (url.includes('/api/v1/planning/bundle')) {
        return Promise.resolve({ ok: true, json: async () => bundle } as Response)
      }
      if (url.includes('/api/v1/planning/advisory-alerts')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ alerts: scenario.advisories ?? [], evidence_flags: scenario.advisoryEvidenceFlags ?? {} }),
        } as Response)
      }
      return Promise.reject(new Error(`unexpected fetch: ${url}`))
    }),
  )
}

describe('Planning', () => {
  beforeEach(() => {
    setupPlanningFetch()
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

  it('shows baseline-missing and directional caution semantics', async () => {
    setupPlanningFetch({
      evidenceModel: 'Planning evidence model copy changed with no baseline wording.',
      evidenceFlags: { baseline_missing: true, directional_only: true },
    })

    render(<Planning />)
    await waitFor(() => {
      expect(screen.getByTestId('planning-evidence-signals')).toBeTruthy()
    })

    const text = screen.getByTestId('planning-evidence-signals').textContent ?? ''
    expect(text).toContain('Baseline evidence is missing or unavailable')
    expect(text).toContain('directional/inconclusive')
  })

  it('shows confounded caution when same assessment context is reported', async () => {
    setupPlanningFetch({
      meshAssessmentId: 'assessment-123',
      evidenceModel: 'Confounding wording is intentionally absent from prose.',
      evidenceFlags: { confounded_same_assessment_context: true },
    })

    render(<Planning />)
    await waitFor(() => {
      expect(screen.getByTestId('planning-evidence-signals')).toBeTruthy()
    })

    expect(screen.getByTestId('planning-evidence-signals').textContent).toContain(
      'potentially confounded (same or concurrent assessment context)',
    )
  })

  it('shows topology drift warning text when graph drift evidence exists', async () => {
    setupPlanningFetch({
      evidenceModel: 'No graph drift wording in fallback prose.',
      evidenceFlags: { topology_or_graph_drift_detected: true },
    })

    render(<Planning />)
    await waitFor(() => {
      expect(screen.getByTestId('planning-evidence-signals')).toBeTruthy()
    })

    expect(screen.getByTestId('planning-evidence-signals').textContent).toContain('Graph/topology drift is present')
  })

  it('keeps empty advisories and uncertainty visible together (not all-clear)', async () => {
    setupPlanningFetch({
      evidenceModel: 'No uncertainty wording in prose.',
      advisories: [],
      advisoryEvidenceFlags: { no_advisories: true },
      uncertaintyNotes: ['Short observation horizon'],
      evidenceFlags: { limited_confidence: true },
    })

    render(<Planning />)
    await waitFor(() => {
      expect(screen.getByTestId('planning-advisories-empty')).toBeTruthy()
      expect(screen.getByTestId('planning-evidence-signals')).toBeTruthy()
    })

    expect(screen.getByTestId('planning-advisories-empty').textContent).toMatch(/does not prove/)
    expect(screen.getByTestId('planning-evidence-signals').textContent).toContain('not an all-clear')
    expect(screen.getByTestId('planning-evidence-signals').textContent).toContain('Recommendations exist, but confidence is limited')
  })

  it('surfaces limited-confidence semantics when advisory/recommendation exists', async () => {
    setupPlanningFetch({
      evidenceClassification: 'topology_only',
      uncertaintyNotes: [],
      evidenceFlags: { recommendation_present_with_uncertain_evidence: true },
      advisories: [{ id: 'a-1', severity: 'warning', reason: 'partition-risk', summary: 'Potential fragility increase' }],
    })

    render(<Planning />)
    await waitFor(() => {
      expect(screen.getByTestId('planning-evidence-signals')).toBeTruthy()
    })

    expect(screen.getByTestId('planning-evidence-signals').textContent).toContain(
      'Recommendations exist, but confidence is limited',
    )
  })

  it('falls back to legacy phrase parsing when evidence flags are absent', async () => {
    setupPlanningFetch({
      evidenceFlags: {},
      evidenceModel: 'No baseline mesh assessment id was recorded and validation is directional.',
    })
    render(<Planning />)
    await waitFor(() => {
      expect(screen.getByTestId('planning-evidence-signals')).toBeTruthy()
    })
    expect(screen.getByTestId('planning-evidence-signals').textContent).toContain('Baseline evidence is missing or unavailable')
  })
})
