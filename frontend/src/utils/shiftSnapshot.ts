/**
 * Browser-local "since last visit" snapshot for the operator command surface.
 * Single-browser / single-profile; does not coordinate across operators or devices.
 */
import type { AuditLog, Incident } from '@/types/api'

const STORAGE_KEY = 'mel_operator_shift_snapshot_v1'

export interface ShiftSnapshot {
  savedAt: string
  openIncidentIds: string[]
  nodeLastSeen: Record<string, string>
  transportHeartbeatMax: string | null
  eventMaxTime: string | null
  messageCountApprox: number
  deadLetterCount: number
}

export function readShiftSnapshot(): ShiftSnapshot | null {
  if (typeof window === 'undefined') return null
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const p = JSON.parse(raw) as Partial<ShiftSnapshot>
    if (!p || typeof p.savedAt !== 'string') return null
    return {
      savedAt: p.savedAt,
      openIncidentIds: Array.isArray(p.openIncidentIds) ? p.openIncidentIds.filter((x) => typeof x === 'string') : [],
      nodeLastSeen:
        p.nodeLastSeen && typeof p.nodeLastSeen === 'object' && !Array.isArray(p.nodeLastSeen)
          ? (p.nodeLastSeen as Record<string, string>)
          : {},
      transportHeartbeatMax: typeof p.transportHeartbeatMax === 'string' ? p.transportHeartbeatMax : null,
      eventMaxTime: typeof p.eventMaxTime === 'string' ? p.eventMaxTime : null,
      messageCountApprox: typeof p.messageCountApprox === 'number' ? p.messageCountApprox : 0,
      deadLetterCount: typeof p.deadLetterCount === 'number' ? p.deadLetterCount : 0,
    }
  } catch {
    return null
  }
}

export function writeShiftSnapshot(snapshot: ShiftSnapshot): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(snapshot))
  } catch {
    /* quota / private mode */
  }
}

export function buildShiftSnapshotFromConsole(args: {
  incidents: Incident[]
  nodes: Array<{ node_id: string; last_seen: string }>
  transports: Array<{ last_heartbeat_at?: string }>
  events: Pick<AuditLog, 'created_at'>[]
  messageCount: number
  deadLetterCount: number
}): ShiftSnapshot {
  const openIncidentIds = args.incidents
    .filter((i) => {
      const s = (i.state || '').toLowerCase()
      return s !== 'resolved' && s !== 'closed'
    })
    .map((i) => i.id)

  const nodeLastSeen: Record<string, string> = {}
  for (const n of args.nodes) {
    if (n.node_id && n.last_seen) nodeLastSeen[n.node_id] = n.last_seen
  }

  let transportHeartbeatMax: string | null = null
  let maxTs = 0
  for (const t of args.transports) {
    if (!t.last_heartbeat_at) continue
    const x = new Date(t.last_heartbeat_at).getTime()
    if (x > maxTs) {
      maxTs = x
      transportHeartbeatMax = t.last_heartbeat_at
    }
  }

  let eventMaxTime: string | null = null
  let eventMax = 0
  for (const e of args.events) {
    const ts = e.created_at ? new Date(e.created_at).getTime() : 0
    if (ts > eventMax) {
      eventMax = ts
      eventMaxTime = e.created_at ?? null
    }
  }

  return {
    savedAt: new Date().toISOString(),
    openIncidentIds,
    nodeLastSeen,
    transportHeartbeatMax,
    eventMaxTime,
    messageCountApprox: args.messageCount,
    deadLetterCount: args.deadLetterCount,
  }
}

export interface ShiftDelta {
  incidentsTouchedSince: Incident[]
  nodesWithNewerLastSeen: Array<{ node_id: string; long_name?: string; short_name?: string }>
  transportHeartbeatAdvanced: boolean
  newAuditEvents: number
  messagesIncreased: boolean
  deadLettersIncreased: boolean
}

function ts(s: string | undefined): number {
  if (!s) return 0
  const t = new Date(s).getTime()
  return Number.isFinite(t) ? t : 0
}

export function computeShiftDelta(
  prev: ShiftSnapshot | null,
  args: {
    incidents: Incident[]
    nodes: Array<{ node_id: string; long_name?: string; short_name?: string; last_seen: string }>
    transports: Array<{ last_heartbeat_at?: string }>
    events: Pick<AuditLog, 'created_at'>[]
    messageCount: number
    deadLetterCount: number
  },
): ShiftDelta {
  if (!prev) {
    return {
      incidentsTouchedSince: [],
      nodesWithNewerLastSeen: [],
      transportHeartbeatAdvanced: false,
      newAuditEvents: 0,
      messagesIncreased: false,
      deadLettersIncreased: false,
    }
  }
  const anchor = ts(prev.savedAt)

  const incidentsTouchedSince = args.incidents.filter((inc) => {
    const touch = Math.max(ts(inc.updated_at), ts(inc.occurred_at), ts(inc.resolved_at))
    return touch > anchor
  })

  const nodesWithNewerLastSeen = args.nodes.filter((n) => {
    const prevSeen = prev.nodeLastSeen[n.node_id]
    if (!prevSeen) return ts(n.last_seen) > anchor
    return ts(n.last_seen) > ts(prevSeen)
  })

  let transportHeartbeatAdvanced = false
  if (prev.transportHeartbeatMax) {
    const prevHb = ts(prev.transportHeartbeatMax)
    let max = 0
    for (const t of args.transports) {
      const x = ts(t.last_heartbeat_at)
      if (x > max) max = x
    }
    transportHeartbeatAdvanced = max > prevHb
  }

  const eventCut = ts(prev.eventMaxTime || '')
  let newAuditEvents = 0
  for (const e of args.events) {
    if (ts(e.created_at) > eventCut) newAuditEvents++
  }

  return {
    incidentsTouchedSince,
    nodesWithNewerLastSeen,
    transportHeartbeatAdvanced,
    newAuditEvents,
    messagesIncreased: args.messageCount > prev.messageCountApprox,
    deadLettersIncreased: args.deadLetterCount > prev.deadLetterCount,
  }
}
