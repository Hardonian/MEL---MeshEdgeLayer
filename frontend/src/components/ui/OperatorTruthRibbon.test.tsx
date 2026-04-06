import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { OperatorTruthRibbon } from './OperatorTruthRibbon'

describe('OperatorTruthRibbon', () => {
  it('renders summary and links to effective config anchor', () => {
    render(
      <MemoryRouter>
        <OperatorTruthRibbon summary="Test truth line for the operator." />
      </MemoryRouter>,
    )
    expect(screen.getByText(/Test truth line/)).toBeTruthy()
    const link = screen.getByRole('link', { name: /Runtime posture/ })
    expect(link.getAttribute('href')).toBe('/settings#effective-config')
  })
})
