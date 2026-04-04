import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { Layout } from './Layout'

vi.mock('@/hooks/useApi', () => ({
  useStatus: () => ({
    data: {
      transports: [],
      instance: { instance_id: 'inst-test' },
      product: { product_scope: 'mel.test.scope' },
    },
    loading: false,
    error: null,
  }),
  useApi: () => ({
    deadLetters: { data: [] },
    diagnostics: { data: [] },
    privacyFindings: { data: [] },
    recommendations: { data: [] },
    refreshMeta: { mode: 'near_live_polling' as const },
    refreshAll: vi.fn(),
  }),
}))

describe('Layout operator identity', () => {
  it('surfaces operator OS framing and truth contract strip', () => {
    render(
      <MemoryRouter>
        <Layout>
          <p>Child</p>
        </Layout>
      </MemoryRouter>
    )

    expect(screen.getByText('Operator OS')).toBeTruthy()
    expect(screen.getByRole('note', { name: /operator truth contract/i })).toBeTruthy()
    expect(screen.getByText(/Incident intelligence and governed control/i)).toBeTruthy()
    expect(screen.getByText('Child')).toBeTruthy()
  })
})
