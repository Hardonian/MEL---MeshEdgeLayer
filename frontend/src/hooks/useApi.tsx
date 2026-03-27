import { createContext, useContext, useState, useCallback, useEffect, ReactNode } from 'react'
import type {
  StatusResponse,
  Message,
  DeadLetter,
  PrivacyFinding,
  Recommendation,
  AuditLog,
  NodeInfo,
  Finding,
  ControlStatusResponse,
  ControlHistoryResponse,
  ControlOperationalStateResponse,
  ControlRealityMatrixItem,
  MeshNodeControlAction,
} from '@/types/api'

// API Base URL - uses relative path for proxy
const API_BASE = '/api'

function isRecord(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === 'object' && !Array.isArray(v)
}

function parseRealityMatrixItem(raw: unknown): ControlRealityMatrixItem | null {
  if (!isRecord(raw)) return null
  const action_type = raw.action_type
  if (typeof action_type !== 'string') return null
  const blast_radius_class = raw.blast_radius_class
  const notes = raw.notes
  return {
    action_type,
    actuator_exists: raw.actuator_exists === true,
    reversible: raw.reversible === true,
    blast_radius_class: typeof blast_radius_class === 'string' ? blast_radius_class : 'unknown',
    safe_for_guarded_auto: raw.safe_for_guarded_auto === true,
    advisory_only: raw.advisory_only === true,
    notes: typeof notes === 'string' ? notes : '',
  }
}

function parseMeshNodeControlAction(raw: unknown): MeshNodeControlAction | null {
  if (!isRecord(raw)) return null
  const id = raw.id
  const result = raw.result
  if (typeof id !== 'string' || typeof result !== 'string') return null

  let operator_view: MeshNodeControlAction['operator_view']
  const ovRaw = raw.operator_view
  if (isRecord(ovRaw)) {
    operator_view = {
      target_summary: typeof ovRaw.target_summary === 'string' ? ovRaw.target_summary : undefined,
      approval_status: typeof ovRaw.approval_status === 'string' ? ovRaw.approval_status : undefined,
      execution_status: typeof ovRaw.execution_status === 'string' ? ovRaw.execution_status : undefined,
      queue_status: typeof ovRaw.queue_status === 'string' ? ovRaw.queue_status : undefined,
      second_operator_note:
        typeof ovRaw.second_operator_note === 'string' ? ovRaw.second_operator_note : undefined,
      sod_blocks_self: ovRaw.sod_blocks_self === true,
      break_glass_in_history: ovRaw.break_glass_in_history === true,
      linked_incident_id: typeof ovRaw.linked_incident_id === 'string' ? ovRaw.linked_incident_id : undefined,
    }
  }

  let details: Record<string, unknown> | undefined
  const d = raw.details
  if (isRecord(d)) {
    details = d
  }

  return {
    id,
    result,
    command: typeof raw.command === 'string' ? raw.command : undefined,
    action_type: typeof raw.action_type === 'string' ? raw.action_type : undefined,
    target_node: typeof raw.target_node === 'string' ? raw.target_node : undefined,
    target_segment: typeof raw.target_segment === 'string' ? raw.target_segment : undefined,
    target_node_id: typeof raw.target_node_id === 'string' ? raw.target_node_id : undefined,
    target_transport: typeof raw.target_transport === 'string' ? raw.target_transport : undefined,
    transport_name: typeof raw.transport_name === 'string' ? raw.transport_name : undefined,
    denial_reason: typeof raw.denial_reason === 'string' ? raw.denial_reason : undefined,
    created_at: typeof raw.created_at === 'string' ? raw.created_at : undefined,
    executed_at: typeof raw.executed_at === 'string' ? raw.executed_at : undefined,
    outcome_detail: typeof raw.outcome_detail === 'string' ? raw.outcome_detail : undefined,
    advisory_only: raw.advisory_only === true,
    lifecycle_state: typeof raw.lifecycle_state === 'string' ? raw.lifecycle_state : undefined,
    execution_mode: typeof raw.execution_mode === 'string' ? raw.execution_mode : undefined,
    proposed_by: typeof raw.proposed_by === 'string' ? raw.proposed_by : undefined,
    approved_by: typeof raw.approved_by === 'string' ? raw.approved_by : undefined,
    evidence_bundle_id: typeof raw.evidence_bundle_id === 'string' ? raw.evidence_bundle_id : undefined,
    operator_view,
    details,
  }
}

function parseControlStatusJson(raw: unknown): ControlStatusResponse {
  if (!isRecord(raw)) return {}
  const matrixRaw = raw.reality_matrix
  const reality_matrix = Array.isArray(matrixRaw)
    ? matrixRaw.map(parseRealityMatrixItem).filter((x): x is ControlRealityMatrixItem => x !== null)
    : undefined
  return {
    mode: typeof raw.mode === 'string' ? raw.mode : undefined,
    reality_matrix,
    queue_depth: typeof raw.queue_depth === 'number' ? raw.queue_depth : undefined,
    queue_capacity: typeof raw.queue_capacity === 'number' ? raw.queue_capacity : undefined,
    active_actions: typeof raw.active_actions === 'number' ? raw.active_actions : undefined,
    policy_summary: typeof raw.policy_summary === 'string' ? raw.policy_summary : undefined,
    emergency_disable: typeof raw.emergency_disable === 'boolean' ? raw.emergency_disable : undefined,
  }
}

function parseControlHistoryJson(raw: unknown): ControlHistoryResponse {
  if (!isRecord(raw)) return {}
  const actionsRaw = raw.actions
  const actions = Array.isArray(actionsRaw)
    ? actionsRaw.map(parseMeshNodeControlAction).filter((x): x is MeshNodeControlAction => x !== null)
    : undefined
  return { actions }
}

function parseOperationalStateJson(raw: unknown): ControlOperationalStateResponse {
  if (!isRecord(raw)) return {}
  const pendingRaw = raw.pending_approvals
  const pending_approvals = Array.isArray(pendingRaw)
    ? pendingRaw.map(parseMeshNodeControlAction).filter((x): x is MeshNodeControlAction => x !== null)
    : undefined
  return { pending_approvals }
}

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
      const data = await res.json()
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
  const [data, setData] = useState<ControlStatusResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/control/status')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const json: unknown = await res.json()
      setData(parseControlStatusJson(json))
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
  const [data, setData] = useState<ControlHistoryResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/control/history')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const json: unknown = await res.json()
      setData(parseControlHistoryJson(json))
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
  const [data, setData] = useState<ControlOperationalStateResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/control/operational-state')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const json: unknown = await res.json()
      setData(parseOperationalStateJson(json))
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
