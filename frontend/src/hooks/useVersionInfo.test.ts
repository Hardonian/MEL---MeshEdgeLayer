import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { useVersionInfo } from './useVersionInfo'

describe('useVersionInfo', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('loads version from GET /api/v1/version', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        version: '1.2.3',
        go_version: 'go1.24.0',
        schema_matches_binary: true,
        db_actual_version: '42',
      }),
    }) as unknown as typeof fetch

    const { result } = renderHook(() => useVersionInfo())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBeNull()
    expect(result.current.data?.version).toBe('1.2.3')
    expect(result.current.data?.go_version).toBe('go1.24.0')
    expect(result.current.data?.schema_matches_binary).toBe(true)
  })

  it('surfaces HTTP errors', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
    }) as unknown as typeof fetch

    const { result } = renderHook(() => useVersionInfo())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.data).toBeNull()
    expect(result.current.error).toContain('503')
  })
})
