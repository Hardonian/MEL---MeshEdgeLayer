import { useCallback, useEffect, useState } from 'react'
import type { ControlActionRecord } from '@/types/api'

export function useControlActions(lifecycleFilter: string) {
  const [data, setData] = useState<ControlActionRecord[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams({ limit: '100' })
      if (lifecycleFilter.trim()) {
        params.set('lifecycle_state', lifecycleFilter.trim())
      }
      const res = await fetch(`/api/v1/control/actions?${params.toString()}`)
      if (res.status === 403) {
        throw new Error('Forbidden — missing read_actions or read_status capability')
      }
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const json = await res.json()
      const rows = (json.actions as ControlActionRecord[]) || []
      setData(rows)
    } catch (e) {
      setData(null)
      setError(e instanceof Error ? e.message : 'Failed to load control actions')
    } finally {
      setLoading(false)
    }
  }, [lifecycleFilter])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return { data, loading, error, refresh }
}
