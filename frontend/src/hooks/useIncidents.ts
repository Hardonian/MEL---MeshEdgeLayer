import { useCallback, useEffect, useState } from 'react'
import type { Incident } from '@/types/api'

export function useIncidents() {
  const [data, setData] = useState<Incident[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/v1/incidents')
      if (res.status === 403) {
        throw new Error('Forbidden — missing read_incidents or read_status capability')
      }
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const json = await res.json()
      const rows = (json.recent_incidents as Incident[]) || []
      setData(rows)
    } catch (e) {
      setData(null)
      setError(e instanceof Error ? e.message : 'Failed to load incidents')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return { data, loading, error, refresh }
}
