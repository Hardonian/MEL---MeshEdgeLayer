import { describe, expect, it, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { FirstRunHintBanner } from './FirstRunHintBanner'

describe('FirstRunHintBanner', () => {
  afterEach(() => {
    localStorage.clear()
  })

  it('renders truthful first-run links including transports route', () => {
    render(
      <MemoryRouter>
        <FirstRunHintBanner visible />
      </MemoryRouter>,
    )

    expect(screen.getByRole('link', { name: 'Transports' }).getAttribute('href')).toBe('/transports')
    expect(screen.getByRole('link', { name: 'Status health' }).getAttribute('href')).toBe('/status#transport-health')
  })

  it('shows clipboard warning when clipboard API is unavailable', () => {
    Object.defineProperty(globalThis.navigator, 'clipboard', {
      configurable: true,
      value: { writeText: undefined },
    })

    render(
      <MemoryRouter>
        <FirstRunHintBanner visible />
      </MemoryRouter>,
    )

    fireEvent.click(screen.getByRole('button', { name: /copy quickstart commands/i }))

    expect(screen.getByText(/Clipboard unavailable/i)).toBeTruthy()
  })
})
