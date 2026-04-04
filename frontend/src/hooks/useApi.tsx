import { createContext, useContext, useState, useCallback, useEffect, ReactNode } from 'react'
import type {
  StatusResponse,
  Message,
  DeadLetter,
  PrivacyFinding,
  Recommendation,
  AuditLog,
  NodeInfo,
  ControlStatusResponse,
  ControlHistoryResponse,
  ControlOperationalStateResponse,
  ControlQueueMetrics,
  ControlExecutorPresence,
  ActiveFreezeWindow,
  ActiveMaintenanceWindow,
  ControlRealityMatrixItem,
  MeshNodeControlAction,
  OperatorBriefingResponse,
  OperatorBriefingPriority,
  OperatorBriefingRecoveryStep,
} from '@/types/api'
import { parseDiagnosticsFindingsFromApi, type DiagnosticsApiFinding } from '@/utils/apiResilience'

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

export function parseMeshNodeControlAction(raw: unknown): MeshNodeControlAction | null {
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
    approval_expires_at: typeof raw.approval_expires_at === 'string' ? raw.approval_expires_at : undefined,
    blast_radius_class: typeof raw.blast_radius_class === 'string' ? raw.blast_radius_class : undefined,
    requires_separate_approver: raw.requires_separate_approver === true,
    incident_id: typeof raw.incident_id === 'string' ? raw.incident_id : undefined,
    execution_started_at: typeof raw.execution_started_at === 'string' ? raw.execution_started_at : undefined,
    sod_bypass: raw.sod_bypass === true,
    sod_bypass_actor: typeof raw.sod_bypass_actor === 'string' ? raw.sod_bypass_actor : undefined,
    sod_bypass_reason: typeof raw.sod_bypass_reason === 'string' ? raw.sod_bypass_reason : undefined,
    approval_mode: typeof raw.approval_mode === 'string' ? raw.approval_mode : undefined,
    required_approvals: typeof raw.required_approvals === 'number' ? raw.required_approvals : undefined,
    collected_approvals: typeof raw.collected_approvals === 'number' ? raw.collected_approvals : undefined,
    approval_basis: Array.isArray(raw.approval_basis)
      ? raw.approval_basis.filter((x): x is string => typeof x === 'string')
      : undefined,
    approval_policy_source: typeof raw.approval_policy_source === 'string' ? raw.approval_policy_source : undefined,
    high_blast_radius: raw.high_blast_radius === true,
    approval_escalated_due_to_blast_radius: raw.approval_escalated_due_to_blast_radius === true,
    execution_source: typeof raw.execution_source === 'string' ? raw.execution_source : undefined,
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

export function parseOperatorBriefingJson(raw: unknown): OperatorBriefingResponse {
  if (!isRecord(raw)) {
    return {
      api_version: '',
      truth_basis: [],
      overall_status: 'unknown',
      top_priorities: [],
      likely_causes: [],
      recommended_sequence: [],
      blast_radius_estimate: '',
      uncertainty_notes: [],
      generated_at: '',
    }
  }
  const prioritiesRaw = raw.top_priorities
  const top_priorities: OperatorBriefingPriority[] = Array.isArray(prioritiesRaw)
    ? prioritiesRaw
        .map((p): OperatorBriefingPriority | null => {
          if (!isRecord(p)) return null
          return {
            id: typeof p.id === 'string' ? p.id : '',
            category: typeof p.category === 'string' ? p.category : '',
            severity: typeof p.severity === 'string' ? p.severity : '',
            title: typeof p.title === 'string' ? p.title : '',
            summary: typeof p.summary === 'string' ? p.summary : '',
            rank: typeof p.rank === 'number' ? p.rank : 0,
            confidence: typeof p.confidence === 'number' ? p.confidence : 0,
            evidence_freshness: typeof p.evidence_freshness === 'string' ? p.evidence_freshness : '',
            is_actionable: p.is_actionable === true,
            blocks_recovery: p.blocks_recovery === true,
            resource_kind: typeof p.resource_kind === 'string' ? p.resource_kind : undefined,
            metadata: isRecord(p.metadata) ? p.metadata : undefined,
          }
        })
        .filter((x): x is OperatorBriefingPriority => x !== null)
    : []

  const seqRaw = raw.recommended_sequence
  const recommended_sequence: OperatorBriefingRecoveryStep[] = Array.isArray(seqRaw)
    ? seqRaw
        .map((s): OperatorBriefingRecoveryStep | null => {
          if (!isRecord(s)) return null
          return {
            stage: typeof s.stage === 'number' ? s.stage : 0,
            action: typeof s.action === 'string' ? s.action : '',
            justification: typeof s.justification === 'string' ? s.justification : '',
            status: typeof s.status === 'string' ? s.status : '',
            unsafe_early: s.unsafe_early === true,
            dependencies: Array.isArray(s.dependencies)
              ? s.dependencies.filter((d): d is string => typeof d === 'string')
              : undefined,
          }
        })
        .filter((x): x is OperatorBriefingRecoveryStep => x !== null)
    : []

  const truthRaw = raw.truth_basis
  const truth_basis = Array.isArray(truthRaw)
    ? truthRaw.filter((x): x is string => typeof x === 'string')
    : []

  const likelyRaw = raw.likely_causes
  const likely_causes = Array.isArray(likelyRaw)
    ? likelyRaw.filter((x): x is string => typeof x === 'string')
    : []

  const uncRaw = raw.uncertainty_notes
  const uncertainty_notes = Array.isArray(uncRaw)
    ? uncRaw.filter((x): x is string => typeof x === 'string')
    : []

  return {
    api_version: typeof raw.api_version === 'string' ? raw.api_version : '',
    truth_basis,
    overall_status: typeof raw.overall_status === 'string' ? raw.overall_status : 'unknown',
    top_priorities,
    likely_causes,
    recommended_sequence,
    blast_radius_estimate:
      typeof raw.blast_radius_estimate === 'string' ? raw.blast_radius_estimate : '',
    uncertainty_notes,
    generated_at: typeof raw.generated_at === 'string' ? raw.generated_at : '',
  }
}

export function parseOperationalStateJson(raw: unknown): ControlOperationalStateResponse {
  if (!isRecord(raw)) return {}
  const pendingRaw = raw.pending_approvals
  const pending_approvals = Array.isArray(pendingRaw)
    ? pendingRaw.map(parseMeshNodeControlAction).filter((x): x is MeshNodeControlAction => x !== null)
    : undefined
  const parseQueueMetrics = (value: unknown): ControlQueueMetrics | undefined => {
    if (!isRecord(value)) return undefined
    return {
      queued_lifecycle_pending_count:
        typeof value.queued_lifecycle_pending_count === 'number' ? value.queued_lifecycle_pending_count : 0,
      awaiting_approval_count: typeof value.awaiting_approval_count === 'number' ? value.awaiting_approval_count : 0,
      approved_waiting_executor_count:
        typeof value.approved_waiting_executor_count === 'number' ? value.approved_waiting_executor_count : 0,
      oldest_queued_pending_created_at:
        typeof value.oldest_queued_pending_created_at === 'string' ? value.oldest_queued_pending_created_at : '',
      oldest_approved_waiting_executor_created_at:
        typeof value.oldest_approved_waiting_executor_created_at === 'string'
          ? value.oldest_approved_waiting_executor_created_at
          : '',
    }
  }

  const parseExecutorPresence = (value: unknown): ControlExecutorPresence | undefined => {
    if (!isRecord(value)) return undefined
    return {
      executor_activity: typeof value.executor_activity === 'string' ? value.executor_activity : 'unknown',
      executor_last_heartbeat_at:
        typeof value.executor_last_heartbeat_at === 'string' ? value.executor_last_heartbeat_at : '',
      executor_last_reported_kind:
        typeof value.executor_last_reported_kind === 'string' ? value.executor_last_reported_kind : '',
      executor_heartbeat_basis:
        typeof value.executor_heartbeat_basis === 'string' ? value.executor_heartbeat_basis : '',
      executor_presence_note: typeof value.executor_presence_note === 'string' ? value.executor_presence_note : '',
      backlog_requires_active_executor: value.backlog_requires_active_executor === true,
    }
  }

  const parseFreezeWindow = (value: unknown): ActiveFreezeWindow | null => {
    if (!isRecord(value)) return null
    const id = value.id
    if (typeof id !== 'string' || id.trim() === '') return null
    return {
      id,
      reason: typeof value.reason === 'string' ? value.reason : '',
      actor: typeof value.created_by === 'string' ? value.created_by : '',
      scope: [typeof value.scope_type === 'string' ? value.scope_type : '', typeof value.scope_value === 'string' ? value.scope_value : '']
        .filter(Boolean)
        .join(':'),
      created_at: typeof value.created_at === 'string' ? value.created_at : '',
    }
  }

  const parseMaintenanceWindow = (value: unknown): ActiveMaintenanceWindow | null => {
    if (!isRecord(value)) return null
    const id = value.id
    if (typeof id !== 'string' || id.trim() === '') return null
    return {
      id,
      reason: typeof value.reason === 'string' ? value.reason : '',
      actor: typeof value.created_by === 'string' ? value.created_by : '',
      starts_at: typeof value.starts_at === 'string' ? value.starts_at : '',
      ends_at: typeof value.ends_at === 'string' ? value.ends_at : '',
    }
  }

  const active_freezes = Array.isArray(raw.active_freezes)
    ? raw.active_freezes.map(parseFreezeWindow).filter((x): x is ActiveFreezeWindow => x !== null)
    : undefined
  const active_maintenance = Array.isArray(raw.active_maintenance)
    ? raw.active_maintenance.map(parseMaintenanceWindow).filter((x): x is ActiveMaintenanceWindow => x !== null)
    : undefined
  return {
    automation_mode: typeof raw.automation_mode === 'string' ? raw.automation_mode : undefined,
    freeze_count: typeof raw.freeze_count === 'number' ? raw.freeze_count : undefined,
    approval_backlog: typeof raw.approval_backlog === 'number' ? raw.approval_backlog : undefined,
    snapshot_at: typeof raw.snapshot_at === 'string' ? raw.snapshot_at : undefined,
    queue_metrics: parseQueueMetrics(raw.queue_metrics),
    executor: parseExecutorPresence(raw.executor),
    active_freezes,
    active_maintenance,
    pending_approvals,
  }
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
  diagnostics: ApiState<DiagnosticsApiFinding[]>
  refreshDiagnostics: () => Promise<void>

  // Operator briefing (incidents + diagnostics rank; GET /api/v1/intelligence/briefing)
  operatorBriefing: ApiState<OperatorBriefingResponse>
  refreshOperatorBriefing: () => Promise<void>
  
  // Global refresh
  refreshAll: () => Promise<void>
  refreshMeta: {
    mode: 'near_live_polling' | 'background_paused'
    intervalMs: number
    lastAttemptAt: Date | null
  }
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
  const [diagnostics, setDiagnostics] = useState<ApiState<DiagnosticsApiFinding[]>>({
    data: null,
    loading: true,
    error: null,
    lastUpdated: null,
  })
  const [operatorBriefing, setOperatorBriefing] = useState<ApiState<OperatorBriefingResponse>>({
    data: null,
    loading: true,
    error: null,
    lastUpdated: null,
  })
  const [refreshMode, setRefreshMode] = useState<'near_live_polling' | 'background_paused'>(
    typeof document !== 'undefined' && document.visibilityState === 'visible'
      ? 'near_live_polling'
      : 'background_paused'
  )
  const [lastAttemptAt, setLastAttemptAt] = useState<Date | null>(null)

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
      const raw = await res.json()
      const list = parseDiagnosticsFindingsFromApi(raw)
      setDiagnostics({ data: list, loading: false, error: null, lastUpdated: new Date() })
    } catch (e) {
      setDiagnostics(s => ({ ...s, loading: false, error: e instanceof Error ? e.message : 'Failed to fetch diagnostics' }))
    }
  }, [])

  const refreshOperatorBriefing = useCallback(async () => {
    setOperatorBriefing((s) => ({ ...s, loading: true, error: null }))
    try {
      const res = await fetch(`${API_BASE}/v1/intelligence/briefing`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const raw: unknown = await res.json()
      setOperatorBriefing({
        data: parseOperatorBriefingJson(raw),
        loading: false,
        error: null,
        lastUpdated: new Date(),
      })
    } catch (e) {
      setOperatorBriefing((s) => ({
        ...s,
        loading: false,
        error: e instanceof Error ? e.message : 'Failed to fetch operator briefing',
      }))
    }
  }, [])

  const refreshAll = useCallback(async () => {
    setLastAttemptAt(new Date())
    await Promise.all([
      refreshStatus(),
      refreshNodes(),
      refreshMessages(),
      refreshDeadLetters(),
      refreshPrivacy(),
      refreshRecommendations(),
      refreshEvents(),
      refreshDiagnostics(),
      refreshOperatorBriefing(),
    ])
  }, [refreshStatus, refreshNodes, refreshMessages, refreshDeadLetters, refreshPrivacy, refreshRecommendations, refreshEvents, refreshDiagnostics, refreshOperatorBriefing])

  // Initial load
  useEffect(() => {
    refreshAll()
  }, [refreshAll])

  // Visibility-aware polling cadence.
  useEffect(() => {
    const computeInterval = () => (document.visibilityState === 'visible' ? 10000 : 60000)
    setRefreshMode(document.visibilityState === 'visible' ? 'near_live_polling' : 'background_paused')

    let intervalMs = computeInterval()
    let interval = setInterval(refreshAll, intervalMs)

    const onVisibilityChange = () => {
      setRefreshMode(document.visibilityState === 'visible' ? 'near_live_polling' : 'background_paused')
      const next = computeInterval()
      if (next === intervalMs) return
      intervalMs = next
      clearInterval(interval)
      interval = setInterval(refreshAll, intervalMs)
      if (document.visibilityState === 'visible') {
        void refreshAll()
      }
    }

    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('focus', onVisibilityChange)
    window.addEventListener('online', onVisibilityChange)

    return () => {
      clearInterval(interval)
      document.removeEventListener('visibilitychange', onVisibilityChange)
      window.removeEventListener('focus', onVisibilityChange)
      window.removeEventListener('online', onVisibilityChange)
    }
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
      operatorBriefing,
      refreshOperatorBriefing,
      refreshAll,
      refreshMeta: {
        mode: refreshMode,
        intervalMs: refreshMode === 'near_live_polling' ? 10000 : 60000,
        lastAttemptAt,
      },
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

export function useOperatorBriefing() {
  const { operatorBriefing, refreshOperatorBriefing } = useApi()
  return { ...operatorBriefing, refresh: refreshOperatorBriefing }
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
