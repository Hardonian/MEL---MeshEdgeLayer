import { createContext, useContext, useState, useCallback, useEffect, ReactNode } from 'react'
import type { StatusResponse, Message, DeadLetter, PrivacyFinding, Recommendation, AuditLog, NodeInfo, Finding } from '@/types/api'

// API Base URL - uses relative path for proxy
const API_BASE = '/api'

interface ApiState<T> {
  data: T | null
  loading: boolean
  error: string | null
  lastUpdated: Date | null
}

interface ApiContextValue {
  // Status
  status: ApiState<StatusResponse>
  refreshStatus: () => Promise<void>
  
  // Nodes
  nodes: ApiState<NodeInfo[]>
  refreshNodes: () => Promise<void>
  
  // Messages
  messages: ApiState<Message[]>
  refreshMessages: () => Promise<void>
  
  // Dead Letters
  deadLetters: ApiState<DeadLetter[]>
  refreshDeadLetters: () => Promise<void>
  
  // Privacy
  privacyFindings: ApiState<PrivacyFinding[]>
  refreshPrivacy: () => Promise<void>
  
  // Recommendations
  recommendations: ApiState<Recommendation[]>
  refreshRecommendations: () => Promise<void>
  
  // Events/Logs
  events: ApiState<AuditLog[]>
  refreshEvents: () => Promise<void>

  // Diagnostics
  diagnostics: ApiState<Finding[]>
  refreshDiagnostics: () => Promise<void>
  
  // Global refresh
  refreshAll: () => Promise<void>
}

const ApiContext = createContext<ApiContextValue | null>(null)

export function ApiProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<ApiState<StatusResponse>>({ data: null, loading: true, error: null, lastUpdated: null })
  const [nodes, setNodes] = useState<ApiState<NodeInfo[]>>({ data: null, loading: true, error: null, lastUpdated: null })
  const [messages, setMessages] = useState<ApiState<Message[]>>({ data: null, loading: true, error: null, lastUpdated: null })
  const [deadLetters, setDeadLetters] = useState<ApiState<DeadLetter[]>>({ data: null, loading: true, error: null, lastUpdated: null })
  const [privacyFindings, setPrivacyFindings] = useState<ApiState<PrivacyFinding[]>>({ data: null, loading: true, error: null, lastUpdated: null })
  const [recommendations, setRecommendations] = useState<ApiState<Recommendation[]>>({ data: null, loading: true, error: null, lastUpdated: null })
  const [events, setEvents] = useState<ApiState<AuditLog[]>>({ data: null, loading: true, error: null, lastUpdated: null })
  const [diagnostics, setDiagnostics] = useState<ApiState<Finding[]>>({ data: null, loading: true, error: null, lastUpdated: null })

  const refreshStatus = useCallback(async () => {
    setStatus(s => ({ ...s, loading: true, error: null }))
    try {
      const res = await fetch(`${API_BASE}/v1/status`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const raw = await res.json()
      const data =
        raw && typeof raw === 'object' && 'status' in raw && raw.status && typeof raw.status === 'object'
          ? { ...(raw.status as StatusResponse), _apiEnvelope: raw }
          : (raw as StatusResponse)
      setStatus({ data, loading: false, error: null, lastUpdated: new Date() })
    } catch (e) {
      setStatus(s => ({ ...s, loading: false, error: e instanceof Error ? e.message : 'Failed to fetch status' }))
    }
  }, [])

  const refreshNodes = useCallback(async () => {
    setNodes(s => ({ ...s, loading: true, error: null }))
    try {
      const res = await fetch(`${API_BASE}/v1/nodes`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setNodes({ data: data.nodes || [], loading: false, error: null, lastUpdated: new Date() })
    } catch (e) {
      setNodes(s => ({ ...s, loading: false, error: e instanceof Error ? e.message : 'Failed to fetch nodes' }))
    }
  }, [])

  const refreshMessages = useCallback(async () => {
    setMessages(s => ({ ...s, loading: true, error: null }))
    try {
      const res = await fetch(`${API_BASE}/v1/messages?limit=50`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setMessages({ data: data.messages || [], loading: false, error: null, lastUpdated: new Date() })
    } catch (e) {
      setMessages(s => ({ ...s, loading: false, error: e instanceof Error ? e.message : 'Failed to fetch messages' }))
    }
  }, [])

  const refreshDeadLetters = useCallback(async () => {
    setDeadLetters(s => ({ ...s, loading: true, error: null }))
    try {
      const res = await fetch(`${API_BASE}/v1/dead-letters?limit=50`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setDeadLetters({ data: data.dead_letters || [], loading: false, error: null, lastUpdated: new Date() })
    } catch (e) {
      setDeadLetters(s => ({ ...s, loading: false, error: e instanceof Error ? e.message : 'Failed to fetch dead letters' }))
    }
  }, [])

  const refreshPrivacy = useCallback(async () => {
    setPrivacyFindings(s => ({ ...s, loading: true, error: null }))
    try {
      const res = await fetch(`${API_BASE}/privacy/audit`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setPrivacyFindings({ data: data.findings || [], loading: false, error: null, lastUpdated: new Date() })
    } catch (e) {
      setPrivacyFindings(s => ({ ...s, loading: false, error: e instanceof Error ? e.message : 'Failed to fetch privacy findings' }))
    }
  }, [])

  const refreshRecommendations = useCallback(async () => {
    setRecommendations(s => ({ ...s, loading: true, error: null }))
    try {
      const res = await fetch(`${API_BASE}/recommendations`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setRecommendations({ data: data.recommendations || [], loading: false, error: null, lastUpdated: new Date() })
    } catch (e) {
      setRecommendations(s => ({ ...s, loading: false, error: e instanceof Error ? e.message : 'Failed to fetch recommendations' }))
    }
  }, [])

  const refreshEvents = useCallback(async () => {
    setEvents(s => ({ ...s, loading: true, error: null }))
    try {
      const res = await fetch(`${API_BASE}/logs?limit=50`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setEvents({ data: data.logs || data.events || [], loading: false, error: null, lastUpdated: new Date() })
    } catch (e) {
      setEvents(s => ({ ...s, loading: false, error: e instanceof Error ? e.message : 'Failed to fetch events' }))
    }
  }, [])

  const refreshDiagnostics = useCallback(async () => {
    setDiagnostics(s => ({ ...s, loading: true, error: null }))
    try {
      const res = await fetch(`${API_BASE}/v1/diagnostics`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setDiagnostics({ data: data || [], loading: false, error: null, lastUpdated: new Date() })
    } catch (e) {
      setDiagnostics(s => ({ ...s, loading: false, error: e instanceof Error ? e.message : 'Failed to fetch diagnostics' }))
    }
  }, [])

  const refreshAll = useCallback(async () => {
    await Promise.all([
      refreshStatus(),
      refreshNodes(),
      refreshMessages(),
      refreshDeadLetters(),
      refreshPrivacy(),
      refreshRecommendations(),
      refreshEvents(),
      refreshDiagnostics(),
    ])
  }, [refreshStatus, refreshNodes, refreshMessages, refreshDeadLetters, refreshPrivacy, refreshRecommendations, refreshEvents, refreshDiagnostics])

  // Initial load
  useEffect(() => {
    refreshAll()
  }, [refreshAll])

  // Auto-refresh every 30 seconds
  useEffect(() => {
    const interval = setInterval(refreshAll, 30000)
    return () => clearInterval(interval)
  }, [refreshAll])

  return (
    <ApiContext.Provider value={{
      status,
      refreshStatus,
      nodes,
      refreshNodes,
      messages,
      refreshMessages,
      deadLetters,
      refreshDeadLetters,
      privacyFindings,
      refreshPrivacy,
      recommendations,
      refreshRecommendations,
      events,
      refreshEvents,
      diagnostics,
      refreshDiagnostics,
      refreshAll,
    }}>
      {children}
    </ApiContext.Provider>
  )
}

export function useApi() {
  const context = useContext(ApiContext)
  if (!context) {
    throw new Error('useApi must be used within ApiProvider')
  }
  return context
}

export function useStatus() {
  const { status, refreshStatus } = useApi()
  return { ...status, refresh: refreshStatus }
}

export function useNodes() {
  const { nodes, refreshNodes } = useApi()
  return { ...nodes, refresh: refreshNodes }
}

export function useMessages() {
  const { messages, refreshMessages } = useApi()
  return { ...messages, refresh: refreshMessages }
}

export function useDeadLetters() {
  const { deadLetters, refreshDeadLetters } = useApi()
  return { ...deadLetters, refresh: refreshDeadLetters }
}

export function usePrivacyFindings() {
  const { privacyFindings, refreshPrivacy } = useApi()
  return { ...privacyFindings, refresh: refreshPrivacy }
}

export function useRecommendations() {
  const { recommendations, refreshRecommendations } = useApi()
  return { ...recommendations, refresh: refreshRecommendations }
}

export function useEvents() {
  const { events, refreshEvents } = useApi()
  return { ...events, refresh: refreshEvents }
}

export function useDiagnostics() {
    const { diagnostics, refreshDiagnostics } = useApi()
    return { ...diagnostics, refresh: refreshDiagnostics }
}

export function useControlStatus() {
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/control/status')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const json = await res.json()
      setData(json)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to fetch control status')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { data, loading, error, refresh }
}

export function useControlHistory() {
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/control/history')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const json = await res.json()
      setData(json)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to fetch control history')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { data, loading, error, refresh }
}

export function useOperationalState() {
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/control/operational-state')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const json = await res.json()
      setData(json)
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to fetch operational state')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  return { data, loading, error, refresh }
}
