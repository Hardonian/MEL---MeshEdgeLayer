import { useCallback, useEffect, useState } from 'react'
import type { PanelResponse, TrustUIHints } from '@/types/api'

export function useOperatorContext() {
  const [trustUI, setTrustUI] = useState<TrustUIHints | null>(null)
  const [capabilities, setCapabilities] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/v1/panel')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data: PanelResponse = await res.json()
      setTrustUI(data.trust_ui ?? null)
      setCapabilities(Array.isArray(data.capabilities) ? data.capabilities : [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load operator context')
      setTrustUI(null)
      setCapabilities([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return { trustUI, capabilities, loading, error, refresh }
}
