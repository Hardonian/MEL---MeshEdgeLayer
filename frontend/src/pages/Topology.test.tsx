import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { Topology } from './Topology'

function renderTopology() {
  return render(
    <BrowserRouter>
      <Topology />
    </BrowserRouter>,
  )
}

describe('Topology truth boundary', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo) => {
        const url = typeof input === 'string' ? input : input.url
        if (url.endsWith('/api/v1/topology')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({ topology_enabled: true, transport_connected: true }),
          } as Response)
        }
        if (url.includes('/api/v1/topology/nodes?')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({
              nodes: [
                { node_num: 1, node_id: 'n1', long_name: 'Node 1', short_name: 'N1', health_state: 'healthy', health_score: 0.9, stale: false },
                { node_num: 2, node_id: 'n2', long_name: 'Node 2', short_name: 'N2', health_state: 'degraded', health_score: 0.4, stale: true },
              ],
            }),
          } as Response)
        }
        if (url.includes('/api/v1/topology/links?')) {
          return Promise.resolve({
            ok: true,
            json: async () => ({
              links: [
                { edge_id: 'e1', src_node_num: 1, dst_node_num: 2, observed: true, stale: true, quality_score: 0.4, relay_dependent: false },
                { edge_id: 'e2', src_node_num: 2, dst_node_num: 1, observed: false, stale: false, quality_score: 0.8, relay_dependent: false },
              ],
            }),
          } as Response)
        }
        if (url.includes('/api/v1/topology/nodes/')) {
          return Promise.resolve({ ok: false, status: 404 } as Response)
        }
        return Promise.reject(new Error(`Unexpected fetch ${url}`))
      }),
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('shows persistent topology truth boundary with inferred/stale edge counts', async () => {
    renderTopology()
    await waitFor(() => {
      expect(screen.getByTestId('topology-truth-boundary')).toBeTruthy()
    })
    expect(screen.getByText(/Not RF\/path proof or delivery proof/i)).toBeTruthy()
    expect(screen.getByText(/includes 1 inferred edge/i)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Stale' }))
    expect(screen.getByTestId('topology-truth-boundary')).toBeTruthy()
  })
})
