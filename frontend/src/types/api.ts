// MEL API Types - Derived from Go backend structure

export interface StatusResponse {
  configured_transport_modes: string[]
  messages: number
  nodes: NodeInfo[]
  transports: TransportHealth[]
}

export interface NodeInfo {
  num: number
  id: string
  long_name: string
  short_name: string
  last_seen: string
  gateway_id: string
  // Additional fields from mesh endpoint
  user?: {
    id: string
    long_name: string
    short_name: string
    macaddr: string
    hw_model: string
  }
  position?: {
    latitude_i: number
    longitude_i: number
    altitude: number
    time: string
  }
}

export interface TransportHealth {
  name: string
  source?: string
  type: string
  effective_state: string
  runtime_state: string
  health: HealthScore
  active_alerts: string[]
  status_scope: string
  detail: string
  guidance: string
  total_messages: number
  persisted_messages: number
  last_heartbeat_at: string
  consecutive_timeouts: number
  retry_status: string
  dead_letters: number
  observation_drops: number
  last_attempt_at: string
  last_ingest_at: string
  last_error: string
}

export interface HealthScore {
  score: number
  state: string
  primary_reason: string
  explanation: string[]
}

export interface Message {
  transport_name: string
  packet_id: string
  from_node: string
  to_node: string
  portnum: string
  payload_text: string
  rx_time: string
}

export interface DeadLetter {
  transport_name: string
  transport_type: string
  topic: string
  reason: string
  created_at: string
}

export interface Finding {
  component: string;
  severity: string;
  message: string;
  guidance: string;
}

export interface PrivacyFinding {
  severity: 'critical' | 'high' | 'medium' | 'low'
  message: string
  remediation: string
}

export interface Recommendation {
  category: string
  priority: string
  message: string
  actionable: boolean
  action?: string
}

export interface AuditLog {
  category: string
  level: string
  message: string
  created_at: string
}

export interface PanelData {
  operator_state: string
  summary: string
  short_commands: string[]
  device_menu: string
}

export interface MeshState {
  nodes: NodeInfo[]
  messages: Message[]
  connected: boolean
}

// Self-Observability Types (Phase 10)

export interface InternalComponentHealth {
  name: string
  health: 'healthy' | 'degraded' | 'failing' | 'unknown'
  last_success: string
  last_failure: string
  error_count: number
  success_count: number
  error_rate: number
}

export interface InternalHealthResponse {
  overall_health: 'healthy' | 'degraded' | 'failing' | 'unknown'
  components: InternalComponentHealth[]
}

export interface FreshnessMarker {
  component: string
  last_update: string
  age_seconds: number
  is_fresh: boolean
  is_stale: boolean
  expected_interval: number
  stale_threshold: number
}

export interface FreshnessResponse {
  markers: FreshnessMarker[]
  stale_components: string[]
}

export interface SLOStatus {
  name: string
  description: string
  current_value: number
  target: number
  status: 'healthy' | 'at_risk' | 'breached' | 'unknown'
  budget_used: number
  unit: string
  window: string
  window_start: string
  window_end: string
  evaluated_at: string
}

export interface SLOResponse {
  slos: SLOStatus[]
}

export interface InternalMetricsResponse {
  timestamp: string
  pipeline_latency: Record<string, number>
  worker_heartbeats: Record<string, string>
  queue_depths: Record<string, number>
  error_rates: Record<string, number>
  resource_usage: {
    memory_used_bytes: number
    goroutines: number
    num_gc: number
  }
  operation_counts: Record<string, number>
}

// Utility types
export type HealthState = 'healthy' | 'degraded' | 'unhealthy' | 'unknown'

export function getHealthState(health: HealthScore | undefined): HealthState {
  if (!health) return 'unknown'
  const state = health.state?.toLowerCase()
  if (state === 'healthy' || state === 'ok') return 'healthy'
  if (state === 'degraded' || state === 'warning') return 'degraded'
  if (state === 'unhealthy' || state === 'critical' || state === 'error') return 'unhealthy'
  return 'unknown'
}

export function formatTimestamp(timestamp: string | undefined): string {
  if (!timestamp) return '—'
  try {
    const date = new Date(timestamp)
    return date.toLocaleString()
  } catch {
    return timestamp
  }
}

export function formatRelativeTime(timestamp: string | undefined): string {
  if (!timestamp) return 'never'
  try {
    const date = new Date(timestamp)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMins / 60)
    const diffDays = Math.floor(diffHours / 24)

    if (diffMins < 1) return 'just now'
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    if (diffDays < 7) return `${diffDays}d ago`
    return date.toLocaleDateString()
  } catch {
    return timestamp
  }
}
