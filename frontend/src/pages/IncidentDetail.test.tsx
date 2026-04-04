import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { IncidentDetail } from './IncidentDetail'
import type { Incident } from '@/types/api'
import { ToastProvider } from '@/components/ui/Toast'

function renderIncidentDetail(path = '/incidents/inc-test') {
  return render(
    <ToastProvider>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/incidents/:id" element={<IncidentDetail />} />
        </Routes>
      </MemoryRouter>
    </ToastProvider>,
  )
}

function setupFetch(incidentFactory: () => Incident) {
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo) => {
      const url = typeof input === 'string' ? input : input.url
      if (url.includes('/api/v1/incidents/inc-test/replay')) {
        return Promise.resolve({ ok: false, status: 503 } as Response)
      }
      if (url.includes('/api/v1/incidents/inc-test')) {
        return Promise.resolve({ ok: true, json: async () => incidentFactory() } as Response)
      }
      if (url.includes('/api/v1/panel')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ trust_ui: { incident_mutate: true }, capabilities: ['read_incidents', 'incident_update'] }),
        } as Response)
      }
      if (url.includes('/api/v1/version')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            version: 'test',
            platform_posture: { evidence_export_delete: { export_enabled: true, delete_enabled: false, delete_scope: [] } },
          }),
        } as Response)
      }
      if (url.includes('/api/v1/control/status')) {
        return Promise.resolve({ ok: true, json: async () => ({ mode: 'advisory' }) } as Response)
      }
      return Promise.resolve({ ok: true, json: async () => ({}) } as Response)
    }),
  )
}

describe('IncidentDetail decision-pack rationale rendering', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders degraded rationale reasons when backend guidance is degraded', async () => {
    setupFetch(() => ({
      id: 'inc-test',
      title: 'Transport pressure',
      state: 'open',
      occurred_at: '2026-03-31T10:00:00Z',
      decision_pack: {
        schema_version: '1',
        incident_id: 'inc-test',
        generated_at: '2026-03-31T10:10:00Z',
        queue: { why_surfaced_one_liner: 'Queue surfaced from transport anomaly pattern.' },
        guidance: {
          why_now: 'Recent event volume increased against baseline.',
          degraded: true,
          degraded_reasons: ['incident_intelligence_degraded'],
        },
      },
      replay_summary: { delta_total: 4, recent_count: 6, prior_count: 2 },
    }))

    renderIncidentDetail()
    await waitFor(() => {
      expect(screen.getByText(/Incident Decision Pack/i)).toBeTruthy()
    })

    expect(screen.getByText(/Queue surfaced from transport anomaly pattern/i)).toBeTruthy()
    expect(screen.getByText(/Recent event volume increased against baseline/i)).toBeTruthy()
    expect(screen.getByText(/Pack guidance degraded:/i)).toBeTruthy()
  })

  it('shows explicit unavailable rationale state when decision-pack why/change fields are absent', async () => {
    setupFetch(() => ({
      id: 'inc-test',
      title: 'Sparse incident',
      state: 'open',
      occurred_at: '2026-03-31T10:00:00Z',
      decision_pack: {
        schema_version: '1',
        incident_id: 'inc-test',
        generated_at: '2026-03-31T10:10:00Z',
      },
    }))

    renderIncidentDetail()
    await waitFor(() => {
      expect(screen.getByText(/Incident Decision Pack/i)).toBeTruthy()
    })

    expect(screen.getByText(/Backend rationale unavailable/i)).toBeTruthy()
    expect(screen.getByText(/Treat ordering as partial/i)).toBeTruthy()
  })
})
