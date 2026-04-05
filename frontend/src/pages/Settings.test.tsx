import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { SettingsPage } from './Settings'

vi.mock('@/hooks/useApi', () => ({
  useStatus: () => ({ data: { transports: [] }, loading: false, error: null }),
}))

vi.mock('@/hooks/useVersionInfo', () => ({
  useVersionInfo: () => ({
    data: {
      version: 'test',
      go_version: 'go1.25',
      schema_matches_binary: true,
      db_actual_version: '35',
    },
    loading: false,
    error: null,
  }),
}))

vi.mock('@/hooks/useConsoleThemePreference', () => ({
  useConsoleThemePreference: () => ({
    preference: 'system' as const,
    setPreference: vi.fn(),
  }),
}))

function renderSettings() {
  return render(
    <MemoryRouter>
      <SettingsPage />
    </MemoryRouter>,
  )
}

describe('Settings config inspect truth rendering', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('keeps partial nested config payload unreadable instead of treating it as runtime-loaded', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        fingerprint: 'fp-1',
        values: {
          bind: 'invalid-shape',
          auth: { enabled: true },
        },
      }),
    } as Response)

    renderSettings()

    await waitFor(() => {
      expect(screen.getByText('Config fingerprint:')).toBeTruthy()
    })

    const unreadableBadges = screen.getAllByText(/\[unreadable\]/i)
    expect(unreadableBadges.length).toBeGreaterThan(0)
    expect(screen.getAllByText('true').length).toBeGreaterThan(0)
  })

  it('renders safe-default violations from backend inspect payload', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        fingerprint: 'fp-2',
        canonical_fingerprint: 'canon-2',
        values: {},
        violations: [
          {
            field: 'control.mode',
            issue: 'non-advisory control mode enabled',
            current: 'enabled',
            safe: 'advisory or disabled',
          },
        ],
      }),
    } as Response)

    renderSettings()

    await waitFor(() => {
      expect(screen.getByText('Safe-default violations from config inspect')).toBeTruthy()
    })

    expect(screen.getByText(/control\.mode/)).toBeTruthy()
    expect(screen.getByText(/current=enabled/)).toBeTruthy()
    expect(screen.getByText(/safe=advisory or disabled/)).toBeTruthy()
  })

  it('shows explicit unavailable warning when config inspect payload is invalid', async () => {
    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: async () => 'not-an-object',
    } as Response)

    renderSettings()

    await waitFor(() => {
      expect(screen.getByText('Effective config unavailable')).toBeTruthy()
    })

    expect(screen.getByText(/invalid config inspect payload/)).toBeTruthy()
  })
})
