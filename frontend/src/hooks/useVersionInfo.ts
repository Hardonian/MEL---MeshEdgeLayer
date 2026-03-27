import { useCallback, useEffect, useState } from 'react'
import type { VersionResponse } from '@/types/api'

interface VersionInfoState {
  data: VersionResponse | null
  loading: boolean
  error: string | null
}

/**
 * Fetches build/version metadata from the running MEL instance.
 * Does not change backend behavior; read-only GET /api/v1/version.
 */
export function useVersionInfo(): VersionInfoState & { refresh: () => Promise<void> } {
  const [data, setData] = useState<VersionResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/v1/version')
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`)
      }
      const json: unknown = await res.json()
      if (json && typeof json === 'object' && 'version' in json && typeof (json as VersionResponse).version === 'string') {
        setData(json as VersionResponse)
      } else {
        setData(null)
        setError('Unexpected version response shape')
      }
    } catch (e) {
      setData(null)
      setError(e instanceof Error ? e.message : 'Failed to load version')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return { data, loading, error, refresh }
}
