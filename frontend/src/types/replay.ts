export type ReplayEventClass = 'state' | 'action' | 'transport' | 'evidence' | 'system' | 'unknown'
export type ReplayOrigin = 'observed' | 'derived' | 'imported' | 'unknown'

export interface ReplayEventRaw {
  occurred_at?: string
  observed_at?: string
  kind?: string
  event_type?: string
  source?: string
  basis?: string
  summary?: string
  statement?: string
  actor_id?: string
  stale?: boolean
  confidence?: number | string
  confidence_label?: string
  confidence_posture?: string
  evidence_strength?: string
  inferred?: boolean
  imported?: boolean
}

export interface ReplayEventNormalized {
  id: string
  timestamp?: string
  eventType: string
  summary: string
  source?: string
  basis?: string
  actorId?: string
  stale: boolean
  eventClass: ReplayEventClass
  origin: ReplayOrigin
  confidenceLabel: string
  confidenceValue?: number
  raw: ReplayEventRaw
}

function parseConfidence(raw: ReplayEventRaw): { label: string; value?: number } {
  if (typeof raw.confidence === 'number') {
    const bounded = Math.max(0, Math.min(1, raw.confidence))
    const label = bounded >= 0.8 ? 'high' : bounded >= 0.5 ? 'medium' : 'low'
    return { label, value: bounded }
  }
  if (typeof raw.confidence === 'string') return { label: raw.confidence }
  if (typeof raw.confidence_label === 'string') return { label: raw.confidence_label }
  if (typeof raw.confidence_posture === 'string') return { label: raw.confidence_posture }
  if (typeof raw.evidence_strength === 'string') return { label: raw.evidence_strength }
  return { label: 'unknown' }
}

function classifyEventClass(value: string): ReplayEventClass {
  const t = value.toLowerCase()
  if (t.includes('state') || t.includes('status')) return 'state'
  if (t.includes('action') || t.includes('approval') || t.includes('dispatch')) return 'action'
  if (t.includes('transport') || t.includes('mqtt') || t.includes('ingest') || t.includes('link')) return 'transport'
  if (t.includes('evidence') || t.includes('proof') || t.includes('snapshot')) return 'evidence'
  if (t.includes('system') || t.includes('runtime') || t.includes('service')) return 'system'
  return 'unknown'
}

function deriveOrigin(raw: ReplayEventRaw): ReplayOrigin {
  const source = `${raw.source || ''} ${raw.basis || ''}`.toLowerCase()
  if (raw.imported || source.includes('import')) return 'imported'
  if (raw.inferred || source.includes('derived') || source.includes('inferred')) return 'derived'
  if (source.includes('observed') || source.includes('ingest') || source.includes('audit')) return 'observed'
  return 'unknown'
}

export function normalizeReplayEvents(timeline: ReplayEventRaw[]): ReplayEventNormalized[] {
  return timeline.map((raw, idx) => {
    const eventType = raw.event_type || raw.kind || 'event'
    const confidence = parseConfidence(raw)
    return {
      id: `${raw.occurred_at || raw.observed_at || 'unknown'}-${eventType}-${idx}`,
      timestamp: raw.occurred_at || raw.observed_at,
      eventType,
      summary: raw.summary || raw.statement || 'No summary provided.',
      source: raw.source,
      basis: raw.basis,
      actorId: raw.actor_id,
      stale: Boolean(raw.stale),
      eventClass: classifyEventClass(eventType),
      origin: deriveOrigin(raw),
      confidenceLabel: confidence.label,
      confidenceValue: confidence.value,
      raw,
    }
  })
}
