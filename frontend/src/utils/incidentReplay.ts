function isRecord(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === 'object' && !Array.isArray(v)
}

export interface ReplaySegment {
  event_time: string
  event_type: string
  event_id?: string
  summary: string
  knowledge_posture: string
  event_class?: string
  actor_id?: string
  severity?: string
  scope_posture?: string
  timing_posture?: string
  resource_id?: string
  details?: Record<string, unknown>
  evidence_refs?: string[]
}

export interface ReplayView {
  kind: string
  incident_id: string
  replay_segments?: ReplaySegment[]
  knowledge_timeline?: ReplaySegment[]
  replay_meta?: {
    schema_version?: string
    window_from?: string
    window_to?: string
    timeline_event_count?: number
    recommendation_outcome_count?: number
    combined_segment_count?: number
    sparse_timeline?: boolean
    window_truncated?: boolean
    interpretation_posture?: string
    linked_control_redacted?: boolean
    visibility_note?: string
    ordering_posture_note?: string
    delta_last_10m?: {
      window_minutes: number
      anchor_time: string
      cutoff_time: string
      recent_segment_count: number
      prior_segment_count: number
      recent_control_events: number
      prior_control_events: number
      recent_workflow_events: number
      prior_workflow_events: number
      recent_operator_events: number
      prior_operator_events: number
      recent_evidence_events: number
      prior_evidence_events: number
      recent_incident_events: number
      prior_incident_events: number
      delta_total: number
      delta_control: number
      delta_workflow: number
      delta_operator: number
      delta_evidence: number
      delta_incident: number
      activity_posture: string
      sparse_evidence?: boolean
      uncertainty?: string
      interpretation_posture_note?: string
    }
  }
  recommendation_outcomes?: Array<{
    id: string
    recommendation_id: string
    outcome: string
    actor_id?: string
    note?: string
    created_at: string
  }>
  bounded_counterfactual_ranking?: {
    statement: string
    top?: Array<{ id: string; rank_score: number; strength: string }>
    second?: Array<{ id: string; rank_score: number; strength: string }>
  }
  truth_note?: string
  generated_at?: string
}

function parseReplaySegment(raw: unknown): ReplaySegment | null {
  if (!isRecord(raw)) return null
  if (typeof raw.event_type !== 'string' || typeof raw.summary !== 'string' || typeof raw.knowledge_posture !== 'string') {
    return null
  }

  return {
    event_time: typeof raw.event_time === 'string' ? raw.event_time : '',
    event_type: raw.event_type,
    event_id: typeof raw.event_id === 'string' ? raw.event_id : undefined,
    summary: raw.summary,
    knowledge_posture: raw.knowledge_posture,
    event_class: typeof raw.event_class === 'string' ? raw.event_class : undefined,
    actor_id: typeof raw.actor_id === 'string' ? raw.actor_id : undefined,
    severity: typeof raw.severity === 'string' ? raw.severity : undefined,
    scope_posture: typeof raw.scope_posture === 'string' ? raw.scope_posture : undefined,
    timing_posture: typeof raw.timing_posture === 'string' ? raw.timing_posture : undefined,
    resource_id: typeof raw.resource_id === 'string' ? raw.resource_id : undefined,
    details: isRecord(raw.details) ? raw.details : undefined,
    evidence_refs: Array.isArray(raw.evidence_refs)
      ? raw.evidence_refs.filter((x): x is string => typeof x === 'string' && x.trim().length > 0)
      : undefined,
  }
}

function parseReplaySegments(value: unknown): ReplaySegment[] | undefined {
  if (!Array.isArray(value)) return undefined
  return value.map(parseReplaySegment).filter((s): s is ReplaySegment => s !== null)
}

function parseReplayMeta(value: unknown): ReplayView['replay_meta'] | undefined {
  if (!isRecord(value)) return undefined
  type ReplayDelta = NonNullable<NonNullable<ReplayView['replay_meta']>['delta_last_10m']>
  let delta_last_10m: ReplayDelta | undefined
  if (isRecord(value.delta_last_10m)) {
    const d = value.delta_last_10m
    const activity_posture = typeof d.activity_posture === 'string' ? d.activity_posture : 'unknown'
    delta_last_10m = {
      window_minutes: typeof d.window_minutes === 'number' ? d.window_minutes : 0,
      anchor_time: typeof d.anchor_time === 'string' ? d.anchor_time : '',
      cutoff_time: typeof d.cutoff_time === 'string' ? d.cutoff_time : '',
      recent_segment_count: typeof d.recent_segment_count === 'number' ? d.recent_segment_count : 0,
      prior_segment_count: typeof d.prior_segment_count === 'number' ? d.prior_segment_count : 0,
      recent_control_events: typeof d.recent_control_events === 'number' ? d.recent_control_events : 0,
      prior_control_events: typeof d.prior_control_events === 'number' ? d.prior_control_events : 0,
      recent_workflow_events: typeof d.recent_workflow_events === 'number' ? d.recent_workflow_events : 0,
      prior_workflow_events: typeof d.prior_workflow_events === 'number' ? d.prior_workflow_events : 0,
      recent_operator_events: typeof d.recent_operator_events === 'number' ? d.recent_operator_events : 0,
      prior_operator_events: typeof d.prior_operator_events === 'number' ? d.prior_operator_events : 0,
      recent_evidence_events: typeof d.recent_evidence_events === 'number' ? d.recent_evidence_events : 0,
      prior_evidence_events: typeof d.prior_evidence_events === 'number' ? d.prior_evidence_events : 0,
      recent_incident_events: typeof d.recent_incident_events === 'number' ? d.recent_incident_events : 0,
      prior_incident_events: typeof d.prior_incident_events === 'number' ? d.prior_incident_events : 0,
      delta_total: typeof d.delta_total === 'number' ? d.delta_total : 0,
      delta_control: typeof d.delta_control === 'number' ? d.delta_control : 0,
      delta_workflow: typeof d.delta_workflow === 'number' ? d.delta_workflow : 0,
      delta_operator: typeof d.delta_operator === 'number' ? d.delta_operator : 0,
      delta_evidence: typeof d.delta_evidence === 'number' ? d.delta_evidence : 0,
      delta_incident: typeof d.delta_incident === 'number' ? d.delta_incident : 0,
      activity_posture,
      sparse_evidence: d.sparse_evidence === true,
      uncertainty: typeof d.uncertainty === 'string' ? d.uncertainty : undefined,
      interpretation_posture_note:
        typeof d.interpretation_posture_note === 'string' ? d.interpretation_posture_note : undefined,
    }
  }
  return {
    schema_version: typeof value.schema_version === 'string' ? value.schema_version : undefined,
    window_from: typeof value.window_from === 'string' ? value.window_from : undefined,
    window_to: typeof value.window_to === 'string' ? value.window_to : undefined,
    timeline_event_count: typeof value.timeline_event_count === 'number' ? value.timeline_event_count : undefined,
    recommendation_outcome_count:
      typeof value.recommendation_outcome_count === 'number' ? value.recommendation_outcome_count : undefined,
    combined_segment_count: typeof value.combined_segment_count === 'number' ? value.combined_segment_count : undefined,
    sparse_timeline: value.sparse_timeline === true,
    window_truncated: value.window_truncated === true,
    interpretation_posture: typeof value.interpretation_posture === 'string' ? value.interpretation_posture : undefined,
    linked_control_redacted: value.linked_control_redacted === true,
    visibility_note: typeof value.visibility_note === 'string' ? value.visibility_note : undefined,
    ordering_posture_note: typeof value.ordering_posture_note === 'string' ? value.ordering_posture_note : undefined,
    delta_last_10m,
  }
}

function parseCounterfactualRows(value: unknown): Array<{ id: string; rank_score: number; strength: string }> | undefined {
  if (!Array.isArray(value)) return undefined
  return value
    .map((row) => {
      if (!isRecord(row) || typeof row.id !== 'string' || typeof row.rank_score !== 'number' || typeof row.strength !== 'string') {
        return null
      }
      return { id: row.id, rank_score: row.rank_score, strength: row.strength }
    })
    .filter((x): x is NonNullable<typeof x> => x !== null)
}

function parseCounterfactualRanking(value: unknown): ReplayView['bounded_counterfactual_ranking'] | undefined {
  if (!isRecord(value) || typeof value.statement !== 'string') return undefined
  return {
    statement: value.statement,
    top: parseCounterfactualRows(value.top),
    second: parseCounterfactualRows(value.second),
  }
}

function parseRecommendationOutcomes(value: unknown): ReplayView['recommendation_outcomes'] | undefined {
  if (!Array.isArray(value)) return undefined
  const rows = value
    .map((row) => {
      if (!isRecord(row)) return null
      if (
        typeof row.id !== 'string' ||
        typeof row.recommendation_id !== 'string' ||
        typeof row.outcome !== 'string' ||
        typeof row.created_at !== 'string'
      ) {
        return null
      }
      return {
        id: row.id,
        recommendation_id: row.recommendation_id,
        outcome: row.outcome,
        created_at: row.created_at,
        actor_id: typeof row.actor_id === 'string' ? row.actor_id : undefined,
        note: typeof row.note === 'string' ? row.note : undefined,
      }
    })
    .filter((x): x is NonNullable<typeof x> => x !== null)
  return rows
}

export function parseReplayViewResponse(raw: unknown): ReplayView | null {
  if (!isRecord(raw) || typeof raw.kind !== 'string' || typeof raw.incident_id !== 'string') return null

  return {
    kind: raw.kind,
    incident_id: raw.incident_id,
    replay_segments: parseReplaySegments(raw.replay_segments),
    knowledge_timeline: parseReplaySegments(raw.knowledge_timeline),
    replay_meta: parseReplayMeta(raw.replay_meta),
    recommendation_outcomes: parseRecommendationOutcomes(raw.recommendation_outcomes),
    bounded_counterfactual_ranking: parseCounterfactualRanking(raw.bounded_counterfactual_ranking),
    truth_note: typeof raw.truth_note === 'string' ? raw.truth_note : undefined,
    generated_at: typeof raw.generated_at === 'string' ? raw.generated_at : undefined,
  }
}
