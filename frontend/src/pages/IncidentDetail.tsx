import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams, Link, useNavigate, useSearchParams } from 'react-router-dom'
import {
  ArrowLeft,
  AlertTriangle,
  Clock,
  User,
  RefreshCw,
  Download,
  Shield,
  Activity,
  HelpCircle,
  CheckCircle2,
  XCircle,
  Eye,
  FileText,
  Zap,
  Link2,
  ChevronDown,
  ChevronUp,
  BookOpen,
  GitBranch,
  Circle,
  History,
  ArrowRight,
} from 'lucide-react'
import { clsx } from 'clsx'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { CopyButton } from '@/components/ui/CopyButton'
import { useToast } from '@/components/ui/Toast'
import { useOperatorContext } from '@/hooks/useOperatorContext'
import { useOperatorWorkspaceFocus } from '@/hooks/useOperatorWorkspaceFocus'
import { useControlStatus } from '@/hooks/useApi'
import { useVersionInfo } from '@/hooks/useVersionInfo'
import { formatTimestamp, formatRelativeTime, type ControlActionRecord, type Incident, type IncidentDecisionPack } from '@/types/api'
import {
  evidenceStrengthLabel,
  guidanceConfidenceLabel,
  runbookStrengthOperatorLabel,
  wirelessConfidencePostureLabel,
  wirelessEvidencePostureLabel,
} from '@/utils/evidenceSemantics'
import { controlActionExecPhase } from '@/utils/controlActionPhase'
import { incidentTopologyFocusNodeNum } from '@/utils/operatorWorkflow'
import {
  resolvedIncidentActionVisibility,
  incidentMemoryDecisionCue,
  operatorCanReadLinkedControlRows,
} from '@/utils/incidentOperatorTruth'
import { operatorExportReadinessFromVersion } from '@/utils/operatorExportReadiness'

// ─── Types ────────────────────────────────────────────────────────────────────

interface ReplaySegment {
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

interface ReplayView {
  kind: string
  incident_id: string
  incident: Incident
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

// ─── Helpers ──────────────────────────────────────────────────────────────────

function toWords(v: string | undefined) {
  return (v || '').replace(/_/g, ' ').trim()
}

function postureColor(posture: string): string {
  if (posture.includes('operator') || posture.includes('adjudicat')) return 'text-info'
  if (posture.includes('control')) return 'text-warning'
  if (posture.includes('observed')) return 'text-success'
  return 'text-muted-foreground'
}

function postureLabel(posture: string): string {
  switch (posture) {
    case 'observed_persisted_event': return 'Observed'
    case 'observed_operator_or_system_event': return 'Operator / system'
    case 'observed_control_plane_event': return 'Control plane'
    case 'observed_control_lifecycle_event': return 'Control lifecycle'
    default: return toWords(posture) || 'Unknown'
  }
}

/** Groups segments for replay filter chips (driven by backend event_class when present). */
function replayFilterBucket(seg: ReplaySegment): 'incident' | 'control' | 'workflow' | 'operator' | 'evidence' {
  const c = (seg.event_class || '').trim()
  switch (c) {
    case 'control_action':
    case 'control_lifecycle':
      return 'control'
    case 'workflow':
    case 'handoff':
      return 'workflow'
    case 'operator_annotation':
    case 'operator_adjudication':
      return 'operator'
    case 'evidence_export':
    case 'imported_evidence':
      return 'evidence'
    default:
      return 'incident'
  }
}

const REPLAY_FILTER_OPTIONS = [
  { id: 'all' as const, label: 'All' },
  { id: 'incident' as const, label: 'Incident' },
  { id: 'control' as const, label: 'Control' },
  { id: 'workflow' as const, label: 'Workflow / handoff' },
  { id: 'operator' as const, label: 'Notes / outcomes' },
  { id: 'evidence' as const, label: 'Evidence / import' },
]

type ReplayFilterId = typeof REPLAY_FILTER_OPTIONS[number]['id']

function eventClassShortLabel(seg: ReplaySegment): string {
  const c = (seg.event_class || '').trim()
  if (!c) return toWords(seg.event_type)
  return toWords(c)
}

function severityVariant(s?: string): 'critical' | 'warning' | 'secondary' {
  if (s === 'critical') return 'critical'
  if (s === 'high') return 'warning'
  return 'secondary'
}

function stateVariant(s?: string): 'success' | 'outline' {
  if (s === 'resolved' || s === 'closed') return 'success'
  return 'outline'
}

const WORKFLOW_REVIEW_OPTIONS = [
  { value: 'open', label: 'Open' },
  { value: 'acknowledged', label: 'Acknowledged' },
  { value: 'investigating', label: 'Investigating' },
  { value: 'mitigated', label: 'Mitigated' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'follow_up_needed', label: 'Follow-up needed' },
  { value: 'pending_review', label: 'Pending review' },
  { value: 'resolved_review', label: 'Resolved (review)' },
  { value: 'closed_review', label: 'Closed (review)' },
] as const

function evidenceStrengthVariant(s?: string): 'success' | 'warning' | 'secondary' {
  if (s === 'strong') return 'success'
  if (s === 'moderate') return 'warning'
  return 'secondary'
}

function outcomeVariant(o: string): 'success' | 'critical' | 'secondary' {
  if (o === 'improvement_observed') return 'success'
  if (o === 'deterioration_observed') return 'critical'
  return 'secondary'
}

function proofpackCompletenessVariant(completeness: string): 'success' | 'warning' | 'secondary' {
  if (completeness === 'complete') return 'success'
  if (completeness === 'partial') return 'warning'
  return 'secondary'
}

function defaultProofpackFilename(id: string) {
  return `proofpack-${id || 'incident'}.json`
}

type InvestigationPathStep = {
  label: string
  detail: string
  href: string
  samePage: boolean
  emphasize?: boolean
}

function OperationalMemoryPanel({ inc }: { inc: Incident }) {
  const intel = inc.intelligence
  if (!intel) return null

  const memoryDecisionCue = incidentMemoryDecisionCue(inc)

  const sig = intel.signature_match_count ?? 0
  const sim = intel.similar_incidents ?? []
  const mem = intel.action_outcome_memory ?? []
  const gov = intel.governance_memory ?? []
  const hist = intel.historically_used_actions ?? []
  const drift = intel.drift_fingerprints ?? []
  const corr = intel.correlation_groups ?? []

  const fam = intel.signature_family_resolved_history

  const hasBody =
    sig > 1 ||
    sim.length > 0 ||
    mem.length > 0 ||
    gov.length > 0 ||
    hist.length > 0 ||
    drift.length > 0 ||
    corr.length > 0 ||
    !!inc.reopened_from_incident_id ||
    !!fam

  if (!hasBody) return null

  return (
    <Card data-testid="incident-operational-memory">
      <CardHeader className="pb-2">
        <CardTitle className="text-base inline-flex items-center gap-2">
          <History className="h-4 w-4 text-muted-foreground" aria-hidden />
          Operational memory (this instance)
        </CardTitle>
        <p className="text-xs text-muted-foreground mt-1">
          Counts and links from stored history — bounded, explainable, not causal proof. Weak matches stay labeled in similar cases.
        </p>
      </CardHeader>
      <CardContent className="space-y-3 pt-0 text-xs">
        {memoryDecisionCue && (
          <div
            className="rounded-lg border border-warning/25 bg-warning/5 px-3 py-2 text-[11px] text-foreground"
            role="status"
            data-testid="incident-memory-decision-cue"
          >
            <span className="font-semibold">What history changes next: </span>
            {memoryDecisionCue}
          </div>
        )}
        {inc.reopened_from_incident_id && (
          <div className="rounded-lg border border-border/50 bg-card/40 px-3 py-2">
            <span className="font-semibold text-foreground">Reopened from </span>
            <Link
              to={`/incidents/${encodeURIComponent(inc.reopened_from_incident_id)}`}
              className="font-mono text-primary hover:underline"
            >
              {inc.reopened_from_incident_id.slice(0, 14)}…
            </Link>
            {inc.reopened_at && (
              <span className="text-muted-foreground"> · {formatRelativeTime(inc.reopened_at)}</span>
            )}
          </div>
        )}
        {fam && fam.family_match_total > 0 && (
          <div className="rounded-lg border border-border/50 bg-card/40 px-3 py-2" data-testid="signature-family-history">
            <p className="font-semibold text-foreground">Signature family (resolved peers on this instance)</p>
            <p className="text-muted-foreground mt-0.5">
              <span className="text-foreground font-medium">{fam.resolved_peer_count}</span> other resolved/closed peer
              {fam.resolved_peer_count === 1 ? '' : 's'} ·{' '}
              <span className="text-foreground font-medium">{fam.reopened_peer_count}</span> with reopen marker on record —{' '}
              <span className="text-warning/90">chronology only, not causal proof.</span>
            </p>
            {fam.peer_sample_incident_id && (
              <Link
                to={`/incidents/${encodeURIComponent(fam.peer_sample_incident_id)}`}
                className="mt-1.5 inline-block text-[11px] font-semibold text-primary hover:underline"
              >
                Open sample peer →
              </Link>
            )}
          </div>
        )}
        {sig > 1 && (
          <div className="rounded-lg border border-border/50 bg-card/40 px-3 py-2">
            <p className="font-semibold text-foreground">Signature recurrence</p>
            <p className="text-muted-foreground mt-0.5">
              Same signature bucket seen <span className="text-foreground font-medium">{sig}</span> times — structural repeat, not verified root-cause repeat.
              {intel.signature_label && (
                <>
                  {' '}
                  <span className="text-foreground/90">({intel.signature_label})</span>
                </>
              )}
            </p>
          </div>
        )}
        {sim.length > 0 && (
          <div className="rounded-lg border border-border/50 bg-card/40 px-3 py-2">
            <p className="font-semibold text-foreground">Similar prior incidents</p>
            <p className="text-muted-foreground mt-0.5">
              {sim.length} linked row{sim.length > 1 ? 's' : ''} in intelligence — open each for rationale and weak-match flags.
            </p>
            <a href="#similar-prior-incidents" className="mt-1.5 inline-block text-[11px] font-semibold text-primary hover:underline">
              Jump to list →
            </a>
          </div>
        )}
        {mem.length > 0 && (
          <div className="rounded-lg border border-border/50 bg-card/40 px-3 py-2 space-y-1.5">
            <p className="font-semibold text-foreground">Historical action outcomes (association)</p>
            {mem.slice(0, 4).map((m) => (
              <div key={m.action_type} className="text-muted-foreground border-t border-border/30 pt-1.5 first:border-0 first:pt-0">
                <span className="text-foreground font-medium">{m.action_label || toWords(m.action_type)}</span>
                {' — '}
                {toWords(m.outcome_framing)} · evidence {m.evidence_strength} · n={m.sample_size}
                {m.caveats?.length ? (
                  <span className="block mt-0.5 text-[10px]">{m.caveats!.slice(0, 2).join(' · ')}</span>
                ) : null}
                {(m.inspect_before_reuse?.length ?? 0) > 0 && (
                  <span className="block mt-1 text-[10px] text-warning/90">
                    Before reusing: {m.inspect_before_reuse!.slice(0, 2).join(' · ')}
                  </span>
                )}
              </div>
            ))}
            {mem.length > 4 && (
              <p className="text-[10px] text-muted-foreground/80">+{mem.length - 4} more in detailed section below.</p>
            )}
          </div>
        )}
        {gov.length > 0 && (
          <div className="rounded-lg border border-border/50 bg-card/40 px-3 py-2 space-y-1">
            <p className="font-semibold text-foreground">Governance memory (control plane)</p>
            {gov.slice(0, 3).map((g) => (
              <p key={g.action_type} className="text-muted-foreground">
                <span className="text-foreground font-medium">{toWords(g.action_type)}</span>: {g.summary}
                <span className="text-[10px] text-muted-foreground/80">
                  {' '}
                  ({g.linked_action_count} linked, {g.approved_or_passed_count} approved/passed, {g.rejected_count} rejected)
                </span>
              </p>
            ))}
          </div>
        )}
        {hist.length > 0 && (
          <div className="rounded-lg border border-border/50 bg-card/40 px-3 py-2">
            <p className="font-semibold text-foreground">Historically used action types (this signature family)</p>
            <ul className="mt-1 list-disc pl-4 text-muted-foreground space-y-0.5">
              {hist.slice(0, 6).map((h) => (
                <li key={h.action_type}>
                  {toWords(h.action_type)} — count {h.count}
                </li>
              ))}
            </ul>
          </div>
        )}
        {(drift.length > 0 || corr.length > 0) && (
          <p className="text-[11px] text-muted-foreground border-t border-border/40 pt-2">
            {drift.length > 0 && (
              <span>
                {drift.length} topology drift fingerprint{drift.length > 1 ? 's' : ''} on record (graph / observation bounds, not RF proof).{' '}
              </span>
            )}
            {corr.length > 0 && (
              <span>
                {corr.length} correlation group{corr.length > 1 ? 's' : ''} — see intelligence payload for membership.
              </span>
            )}
          </p>
        )}
        <div className="rounded-lg border border-border/60 bg-muted/15 px-3 py-2 text-[11px] text-muted-foreground space-y-1 border-t border-border/40 pt-3 mt-1">
          <p className="font-semibold text-foreground">What this changes in your next step</p>
          <ul className="list-disc pl-4 space-y-0.5">
            {sig > 1 && (
              <li>
                Treat as a <span className="text-foreground">repeat bucket</span> — compare replay windows and outcomes before assuming the same fix.
              </li>
            )}
            {sim.length > 0 && (
              <li>
                Open similar cases for <span className="text-foreground">bounded pattern context</span> — weak matches stay labeled in detail.
              </li>
            )}
            {mem.some((m) => (m.sample_size ?? 0) < 3 || m.evidence_strength === 'sparse') && (
              <li>
                Sparse outcome history — <span className="text-foreground">do not treat past association as a reliable predictor</span> for this run.
              </li>
            )}
            {gov.some((g) => g.rejected_count > 0) && (
              <li>
                Governance memory shows rejections on this action family — <span className="text-foreground">verify policy / approver posture</span> before re-proposing.
              </li>
            )}
            {inc.reopened_from_incident_id && (
              <li>
                Reopened lineage — <span className="text-foreground">re-verify</span> what changed since the prior incident closed or mitigated.
              </li>
            )}
            {sig <= 1 &&
              sim.length === 0 &&
              mem.length === 0 &&
              gov.length === 0 &&
              hist.length === 0 &&
              drift.length === 0 &&
              corr.length === 0 &&
              !inc.reopened_from_incident_id && (
              <li>No strong historical signals on this row yet — prioritize live replay, topology, and transport evidence.</li>
            )}
          </ul>
        </div>
      </CardContent>
    </Card>
  )
}

function MeshRoutingCompanionStrip({ inc }: { inc: Incident }) {
  const c = inc.intelligence?.mesh_routing_companion
  if (!c?.applicable) return null
  return (
    <div
      className="rounded-xl border border-border/60 bg-muted/10 px-3 py-2.5 text-xs text-muted-foreground"
      role="region"
      aria-label="Mesh routing pressure companion"
    >
      <p className="font-semibold text-foreground mb-1">Mesh ingest routing pressure (companion)</p>
      <p className="leading-snug mb-2">
        From latest mesh intelligence snapshot — observability proxies only, not RF or live path proof. Assessment at{' '}
        <span className="font-mono text-foreground">{c.assessment_computed_at || '—'}</span>
        {c.transport_connected === false && ' · transport was disconnected when computed (may be stale).'}
      </p>
      {(c.suspected_relay_hotspot || c.weak_onward_propagation_suspected || c.hop_budget_stress_suspected) && (
        <ul className="list-disc pl-4 space-y-0.5 mb-2 text-foreground">
          {c.suspected_relay_hotspot && <li>Suspected relay / duplicate-forward hotspot (message-field proxy)</li>}
          {c.weak_onward_propagation_suspected && <li>Weak onward propagation in observed graph edges (proxy)</li>}
          {c.hop_budget_stress_suspected && <li>Hop-limit stress in recent message rollup (proxy)</li>}
        </ul>
      )}
      {c.routing_summary_lines && c.routing_summary_lines.length > 0 && (
        <ul className="list-disc pl-4 space-y-0.5">
          {c.routing_summary_lines.slice(0, 4).map((line, i) => (
            <li key={i}>{line}</li>
          ))}
        </ul>
      )}
      {c.suggested_topology_search && (
        <p className="mt-2">
          <Link
            to={`/topology?${c.suggested_topology_search}`}
            className="font-semibold text-primary hover:underline"
          >
            Open topology with same incident focus →
          </Link>
        </p>
      )}
    </div>
  )
}

function OperatorSuggestedActionsPanel({ inc }: { inc: Incident }) {
  const acts = inc.intelligence?.operator_suggested_actions
  if (!acts?.length) return null
  return (
    <Card id="mel-operator-suggested-actions" className="scroll-mt-20" data-testid="operator-suggested-actions">
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Deterministic next checks</CardTitle>
        <p className="text-xs text-muted-foreground mt-1">
          Reviewable links from persisted evidence — not ranked black-box recommendations. Turn off inference in Settings if you want assist
          disabled; deterministic incident intelligence remains on the server.
        </p>
      </CardHeader>
      <CardContent className="pt-0 space-y-2">
        <ol className="space-y-2">
          {acts.map((a, idx) => (
            <li key={a.id} className="rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-sm">
              <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                <span className="text-[11px] font-mono text-muted-foreground">{idx + 1}.</span>
                <span className="font-medium text-foreground">{a.title}</span>
                <Badge variant="outline" className="text-[10px]">
                  {a.kind.replace(/_/g, ' ')}
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground mt-1 leading-snug">{a.rationale}</p>
              {a.uncertainty && (
                <p className="text-[11px] text-warning mt-1 border-l-2 border-warning/30 pl-2">{a.uncertainty}</p>
              )}
              {a.evidence_refs && a.evidence_refs.length > 0 && (
                <p className="text-[10px] font-mono text-muted-foreground/80 mt-1 break-all">
                  {a.evidence_refs.join(' · ')}
                </p>
              )}
              {a.href && (
                <div className="mt-2">
                  <Link
                    to={a.href}
                    className="text-xs font-semibold text-primary hover:underline inline-flex items-center gap-1 min-h-[44px] sm:min-h-0"
                  >
                    Open
                    <ArrowRight className="h-3 w-3" aria-hidden />
                  </Link>
                </div>
              )}
            </li>
          ))}
        </ol>
      </CardContent>
    </Card>
  )
}

function InvestigationPathPanel({ inc, returnTo }: { inc: Incident; returnTo: string }) {
  const ctx = useOperatorContext()
  const canReadLinked = operatorCanReadLinkedControlRows({
    loading: ctx.loading,
    error: ctx.error,
    trustUI: ctx.trustUI,
    capabilities: ctx.capabilities ?? [],
  })
  const actionVis = resolvedIncidentActionVisibility(inc, { canReadLinkedActions: canReadLinked })
  const topoNum = incidentTopologyFocusNodeNum(inc)
  const intel = inc.intelligence
  const gaps =
    (intel?.wireless_context?.evidence_gaps?.length ?? 0) + (intel?.sparsity_markers?.length ?? 0) > 0
  const rt = encodeURIComponent(returnTo)
  const here = `/incidents/${encodeURIComponent(inc.id)}?return=${rt}`

  const controlStepDetail =
    actionVis.kind === 'linked_observed'
      ? `${actionVis.linkedCount} linked — ${actionVis.awaitingApproval} awaiting approval, ${actionVis.inFlight} queued or executing.`
      : actionVis.explanation

  const steps: InvestigationPathStep[] = [
    {
      label: 'Operational picture',
      detail: 'Severity, state, and evidence strength at top of this page.',
      href: '#incident-operational-summary',
      samePage: true,
      emphasize: false,
    },
    {
      label: 'Linked control actions',
      detail: controlStepDetail,
      href: '#linked-control-actions',
      samePage: true,
      emphasize:
        actionVis.awaitingApproval > 0 ||
        actionVis.inFlight > 0 ||
        actionVis.kind === 'references_only' ||
        actionVis.kind === 'visibility_limited' ||
        actionVis.kind === 'action_context_degraded' ||
        actionVis.kind === 'no_linked_historical_signals',
    },
    {
      label: 'Replay / timeline',
      detail: 'Merged chronology — filter by control, workflow, or evidence classes.',
      href: `${here}&replay=1`,
      samePage: false,
      emphasize: gaps,
    },
    {
      label: 'Topology focus',
      detail:
        topoNum != null
          ? `Graph around node ${topoNum} from resource / implicated domains — not an RF map.`
          : 'Incident-scoped nodes when the record references topology evidence.',
      href: `/topology?incident=${encodeURIComponent(inc.id)}&filter=incident_focus${topoNum != null ? `&select=${topoNum}` : ''}&return=${rt}`,
      samePage: false,
      emphasize: topoNum != null,
    },
    {
      label: 'Planning board',
      detail: 'Resilience bounds from stored topology — same incident context.',
      href: `/planning?incident=${encodeURIComponent(inc.id)}&return=${rt}`,
      samePage: false,
      emphasize: false,
    },
    {
      label: 'Handoff → proof → support',
      detail: 'Continuity text/JSON and escalation bundle; proofpack for bundled evidence; diagnostics for host/runtime.',
      href: '#shift-continuity-handoff',
      samePage: true,
      emphasize: false,
    },
  ]

  return (
    <Card id="mel-investigation-path" data-testid="incident-investigation-path" className="scroll-mt-20">
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Investigation path (in-product)</CardTitle>
        <p className="text-xs text-muted-foreground mt-1">
          One pass through MEL’s surfaces — you still verify against transports and exports. No implied automation or root-cause
          certainty.
        </p>
      </CardHeader>
      <CardContent className="pt-0">
        <ol className="space-y-2">
          {steps.map((s, i) => (
            <li key={s.label} className="flex gap-3 text-sm">
              <span
                className={clsx(
                  'flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border text-[11px] font-semibold',
                  s.emphasize ? 'border-warning/30 bg-warning/8 text-warning' : 'border-border/60 bg-muted/20 text-muted-foreground',
                )}
                aria-hidden
              >
                {i + 1}
              </span>
              <div className="min-w-0 flex-1">
                {s.samePage ? (
                  <a
                    href={s.href}
                    className={clsx('font-medium hover:underline', s.emphasize ? 'text-warning' : 'text-primary')}
                  >
                    {s.label}
                  </a>
                ) : (
                  <Link
                    to={s.href}
                    className={clsx('font-medium hover:underline', s.emphasize ? 'text-warning' : 'text-primary')}
                  >
                    {s.label}
                  </Link>
                )}
                <p className="text-xs text-muted-foreground mt-0.5 leading-snug">{s.detail}</p>
              </div>
            </li>
          ))}
        </ol>
      </CardContent>
    </Card>
  )
}

function InvestigationGuidePanel({ inc, returnTo }: { inc: Incident; returnTo: string }) {
  const intel = inc.intelligence
  if (!intel) return null

  const topoNum = incidentTopologyFocusNodeNum(inc)
  const evPosture = intel.wireless_context ? wirelessEvidencePostureLabel(intel.wireless_context.evidence_posture) : null
  const confPosture = intel.wireless_context ? wirelessConfidencePostureLabel(intel.wireless_context.confidence_posture) : null
  const gaps = [...(intel.sparsity_markers ?? []), ...(intel.wireless_context?.evidence_gaps ?? [])]
  const inspectNext = intel.wireless_context?.inspect_next ?? []

  return (
    <Card data-testid="incident-investigation-guide">
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Investigation guide (bounded)</CardTitle>
        <p className="text-xs text-muted-foreground mt-1">
          Deterministic checklist from stored intelligence — not automation or root-cause AI. Verify against transports, replay, and topology
          before control actions.
        </p>
      </CardHeader>
      <CardContent className="space-y-4 pt-0">
        <div className="flex flex-wrap gap-2 items-start">
          <span className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wide w-full sm:w-auto">Evidence posture</span>
          <span title={evidenceStrengthLabel(intel.evidence_strength)} className="inline-flex">
            <Badge variant={evidenceStrengthVariant(intel.evidence_strength)}>{intel.evidence_strength ?? 'unknown'} strength</Badge>
          </span>
          {intel.degraded && <Badge variant="warning">Degraded intel</Badge>}
          {evPosture && <Badge variant={evPosture.variant}>{evPosture.label}</Badge>}
          {confPosture && <Badge variant={confPosture.variant}>{confPosture.label}</Badge>}
        </div>

        {(intel.degraded_reasons?.length ?? 0) > 0 && (
          <div className="rounded-lg border border-warning/25 bg-warning/5 px-3 py-2 text-xs">
            <p className="font-medium text-foreground">What not to assume</p>
            <ul className="mt-1 list-disc pl-4 text-muted-foreground space-y-0.5">
              {intel.degraded_reasons!.map((r, i) => (
                <li key={i}>{r}</li>
              ))}
            </ul>
          </div>
        )}

        {gaps.length > 0 && (
          <div>
            <p className="text-xs font-semibold text-foreground mb-1">What remains missing / sparse</p>
            <ul className="text-xs text-muted-foreground list-disc pl-4 space-y-0.5">
              {gaps.slice(0, 10).map((g, i) => (
                <li key={i}>{g}</li>
              ))}
            </ul>
          </div>
        )}

        {inspectNext.length > 0 && (
          <div>
            <p className="text-xs font-semibold text-foreground mb-1">Verify first (from wireless context)</p>
            <ul className="text-xs text-muted-foreground list-disc pl-4 space-y-0.5">
              {inspectNext.slice(0, 6).map((x, i) => (
                <li key={i}>{x}</li>
              ))}
            </ul>
          </div>
        )}

        <div className="flex flex-wrap gap-2 pt-1 border-t border-border/40">
          <Link
            to={`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnTo)}&replay=1`}
            className="inline-flex items-center gap-1 rounded-lg border border-border/70 bg-card/50 px-2.5 py-1.5 text-xs font-medium text-primary hover:bg-muted/40"
          >
            <Activity className="h-3.5 w-3.5" />
            Replay / timeline
          </Link>
          <Link
            to={`/topology?incident=${encodeURIComponent(inc.id)}&filter=incident_focus${topoNum != null ? `&select=${topoNum}` : ''}&return=${encodeURIComponent(`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnTo)}`)}`}
            className="inline-flex items-center gap-1 rounded-lg border border-border/70 bg-card/50 px-2.5 py-1.5 text-xs font-medium text-primary hover:bg-muted/40"
          >
            <GitBranch className="h-3.5 w-3.5" />
            Topology{topoNum != null ? ` (node ${topoNum})` : ''}
          </Link>
          <Link
            to={`/planning?incident=${encodeURIComponent(inc.id)}&return=${encodeURIComponent(`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnTo)}`)}`}
            className="inline-flex items-center gap-1 rounded-lg border border-border/70 bg-card/50 px-2.5 py-1.5 text-xs font-medium text-primary hover:bg-muted/40"
          >
            Planning
          </Link>
          <Link
            to={`/control-actions?incident=${encodeURIComponent(inc.id)}&return=${encodeURIComponent(`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnTo)}`)}`}
            className="inline-flex items-center gap-1 rounded-lg border border-border/70 bg-card/50 px-2.5 py-1.5 text-xs font-medium text-primary hover:bg-muted/40"
          >
            Control queue
          </Link>
          <Link
            to="/diagnostics"
            className="inline-flex items-center gap-1 rounded-lg border border-border/70 bg-card/50 px-2.5 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted/40 hover:text-foreground"
          >
            Support bundle
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}

function filenameFromDisposition(cd: string | null, fallback: string): string {
  if (!cd) return fallback
  const m = cd.match(/filename\*?=(?:UTF-8''|")?([^";]+)/i)
  if (!m?.[1]) return fallback
  try { return decodeURIComponent(m[1].replace(/"/g, '').trim()) } catch { return fallback }
}

// ─── Proofpack download button ────────────────────────────────────────────────

function ProofpackButton({
  incidentId,
  exportBlocked,
  exportBlockedReason,
}: {
  incidentId: string
  exportBlocked?: boolean
  exportBlockedReason?: string
}) {
  const [state, setState] = useState<'idle' | 'loading' | 'error'>('idle')
  const [err, setErr] = useState('')

  async function download() {
    if (exportBlocked) return
    setState('loading')
    setErr('')
    try {
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(incidentId)}/proofpack?download=true`)
      if (!res.ok) {
        setErr(res.status === 403 ? 'Insufficient permissions.' : res.status === 404 ? 'Not found.' : `HTTP ${res.status}`)
        setState('error')
        return
      }
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filenameFromDisposition(res.headers.get('content-disposition'), defaultProofpackFilename(incidentId))
      document.body.appendChild(a); a.click(); a.remove()
      URL.revokeObjectURL(url)
      setState('idle')
    } catch {
      setErr('Network error.')
      setState('error')
    }
  }

  return (
    <div className="flex flex-col gap-1.5">
      {exportBlocked && (
        <p className="text-xs text-warning border border-warning/25 rounded-lg px-2 py-1.5 bg-warning/5" role="status">
          {exportBlockedReason || 'Evidence export disabled or unavailable — proofpack request would likely fail.'}
        </p>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => void download()}
          disabled={state === 'loading' || exportBlocked}
          className="button-secondary text-xs min-h-[44px] sm:min-h-0 touch-manipulation"
        >
          <Download className="h-3.5 w-3.5" />
          {state === 'loading' ? 'Assembling…' : 'Export proofpack'}
        </button>
        <span className="text-[10px] text-muted-foreground/60">Snapshot at request-time. Review evidence_gaps.</span>
        {state === 'error' && <span className="text-xs text-critical">{err}</span>}
      </div>
    </div>
  )
}

// ─── Proofpack completeness panel ─────────────────────────────────────────────

function LinkedControlActionsPanel({ inc, returnTo }: { inc: Incident; returnTo: string }) {
  const ctx = useOperatorContext()
  const { data: ctrlData, refresh: refreshCtrl } = useControlStatus()
  const linked = useMemo(() => inc.linked_control_actions ?? [], [inc.linked_control_actions])
  const canReadActions = operatorCanReadLinkedControlRows({
    loading: ctx.loading,
    error: ctx.error,
    trustUI: ctx.trustUI,
    capabilities: ctx.capabilities ?? [],
  })
  const emergencyOff = ctrlData?.emergency_disable === true
  const matrix = ctrlData?.reality_matrix ?? []

  useEffect(() => {
    void refreshCtrl()
  }, [inc.id, refreshCtrl])

  function matrixRowFor(type: string) {
    return matrix.find((m) => m.action_type === type)
  }

  const sortedLinked = useMemo(() => {
    return [...linked].sort((a, b) => {
      const ta = new Date(
        a.completed_at || a.executed_at || a.approved_at || a.created_at || 0,
      ).getTime()
      const tb = new Date(
        b.completed_at || b.executed_at || b.approved_at || b.created_at || 0,
      ).getTime()
      return tb - ta
    })
  }, [linked])

  const groupedSorted = useMemo(() => {
    const awaiting: typeof linked = []
    const inFlight: typeof linked = []
    const done: typeof linked = []
    for (const a of sortedLinked) {
      const ls = (a.lifecycle_state || '').toLowerCase()
      if (ls === 'pending_approval') awaiting.push(a)
      else if (ls === 'pending' || ls === 'running') inFlight.push(a)
      else done.push(a)
    }
    return { awaiting, inFlight, done }
  }, [sortedLinked])

  const matrixCoverage =
    linked.length > 0
      ? linked.filter((a) => {
          const row = matrix.find((m) => m.action_type === a.action_type)
          return row && row.actuator_exists === false
        }).length
      : 0

  if (!canReadActions && linked.length === 0) {
    return (
      <div
        id="linked-control-actions"
        className="rounded-xl border border-border/60 bg-muted/10 p-4 scroll-mt-20"
      >
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-1">
          <Zap className="h-3.5 w-3.5" />
          Linked control actions
        </div>
        <p className="text-xs text-muted-foreground">
          {ctx.error
            ? `Operator context failed to load (${ctx.error}) — cannot confirm read_actions; open the control queue to verify linkage.`
            : 'Your session may lack read_actions — open the control queue with appropriate credentials to see incident-linked rows.'}
        </p>
        <Link
          to={`/control-actions?incident=${encodeURIComponent(inc.id)}&return=${encodeURIComponent(`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnTo)}`)}`}
          className="mt-2 inline-flex text-xs font-semibold text-primary hover:underline"
        >
          Control queue (filtered) →
        </Link>
      </div>
    )
  }

  const showLinkedDespiteGate = !canReadActions && linked.length > 0

  return (
    <div
      id="linked-control-actions"
      className="rounded-xl border border-border/60 bg-muted/10 p-4 space-y-3 scroll-mt-20"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
          <Zap className="h-3.5 w-3.5" />
          Control actions linked to this incident
        </div>
        <Link
          to={`/control-actions?incident=${encodeURIComponent(inc.id)}&return=${encodeURIComponent(`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnTo)}`)}`}
          className="text-[11px] font-semibold text-primary hover:underline"
        >
          Full queue →
        </Link>
      </div>
      <p className="text-[11px] text-muted-foreground leading-snug">
        Rows where <code className="font-mono text-[10px]">incident_id</code> matches this incident. Approval, queue, and execution remain
        separate states — see lifecycle on each row.
      </p>
      {showLinkedDespiteGate && (
        <p className="text-xs text-warning border border-warning/25 rounded-lg px-3 py-2 bg-warning/5" role="status">
          {ctx.error
            ? `Operator context unavailable (${ctx.error}) — rows below came with the incident payload; capability to open the full queue may still be limited.`
            : 'read_actions is off for this session — rows below came with the incident payload; verify sensitive actions in the queue with appropriate credentials.'}
        </p>
      )}
      {emergencyOff && (
        <p className="text-xs text-warning border border-warning/25 rounded-lg px-3 py-2 bg-warning/5">
          Control emergency disable is on for this instance — new execution may be blocked regardless of approval state.
        </p>
      )}
      {ctrlData == null && canReadActions && (
        <p className="text-xs text-muted-foreground border border-border/40 rounded-lg px-2 py-1.5">
          Control capability matrix not loaded — reversibility and advisory cues may be incomplete until status returns.
        </p>
      )}
      {matrixCoverage > 0 && (
        <p className="text-xs text-warning border border-warning/20 rounded-lg px-2 py-1.5 bg-warning/5">
          {matrixCoverage} linked action type{matrixCoverage > 1 ? 's' : ''} report no actuator on this instance — execution may be
          blocked by capability, not only approval.
        </p>
      )}
      {linked.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No linked control rows yet. If you expect actions, check the queue; linkage requires{' '}
          <code className="font-mono text-xs">incident_id</code> on the action record.
        </p>
      ) : (
        <div className="space-y-3">
          {groupedSorted.awaiting.length > 0 && (
            <div>
              <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-warning mb-1.5">Awaiting approval</p>
              <ul className="space-y-2">
                {groupedSorted.awaiting.map((a) => (
                  <LinkedActionRow
                    key={a.id}
                    incidentId={inc.id}
                    action={a}
                    matrixRow={matrixRowFor(a.action_type)}
                    returnAfterQueue={`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnTo)}`}
                  />
                ))}
              </ul>
            </div>
          )}
          {groupedSorted.inFlight.length > 0 && (
            <div>
              <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-info mb-1.5">Queued / executing</p>
              <ul className="space-y-2">
                {groupedSorted.inFlight.map((a) => (
                  <LinkedActionRow
                    key={a.id}
                    incidentId={inc.id}
                    action={a}
                    matrixRow={matrixRowFor(a.action_type)}
                    returnAfterQueue={`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnTo)}`}
                  />
                ))}
              </ul>
            </div>
          )}
          {groupedSorted.done.length > 0 && (
            <div>
              <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground mb-1.5">Completed / terminal</p>
              <ul className="space-y-2">
                {groupedSorted.done.map((a) => (
                  <LinkedActionRow
                    key={a.id}
                    incidentId={inc.id}
                    action={a}
                    matrixRow={matrixRowFor(a.action_type)}
                    returnAfterQueue={`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnTo)}`}
                  />
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function LinkedActionRow({
  incidentId,
  action: a,
  matrixRow,
  returnAfterQueue,
}: {
  incidentId: string
  action: ControlActionRecord
  matrixRow?: {
    reversible?: boolean
    blast_radius_class?: string
    notes?: string
    advisory_only?: boolean
    actuator_exists?: boolean
  }
  returnAfterQueue: string
}) {
  const phase = controlActionExecPhase(a)
  const rev = matrixRow?.reversible === true ? 'Reversible (policy matrix)' : matrixRow?.reversible === false ? 'Treat as hard to reverse' : null
  const blast = matrixRow?.blast_radius_class && matrixRow.blast_radius_class !== 'unknown' ? matrixRow.blast_radius_class : null

  return (
    <li className="rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="font-medium text-foreground">{a.action_type}</p>
          <p className="font-mono text-[10px] text-muted-foreground/80 truncate mt-0.5">{a.id}</p>
        </div>
        <Badge variant={phase.variant}>{phase.label}</Badge>
      </div>
      <div className="mt-1.5 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
        {a.result && <span>result: <span className="text-foreground">{a.result}</span></span>}
        {a.reason && (
          <span className="max-w-full">
            reason: <span className="text-foreground break-words">{a.reason}</span>
          </span>
        )}
        {blast && (
          <span title={matrixRow?.notes || undefined}>
            blast: <span className="text-foreground">{blast}</span>
          </span>
        )}
        {rev && <span>{rev}</span>}
        {matrixRow?.advisory_only && <span className="text-warning">advisory-only type</span>}
        {matrixRow && matrixRow.actuator_exists === false && (
          <span className="text-warning">no actuator on instance (capability)</span>
        )}
        {a.requires_separate_approver && <span className="text-warning">separate approver required</span>}
        {a.approval_mode && a.approval_mode !== 'unknown' && (
          <span>
            approval mode: <span className="text-foreground">{toWords(a.approval_mode)}</span>
          </span>
        )}
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        <Link
          to={`/control-actions?incident=${encodeURIComponent(incidentId)}&return=${encodeURIComponent(returnAfterQueue)}`}
          className="text-[11px] font-semibold text-primary hover:underline"
        >
          Open in queue
        </Link>
        <a
          href={`/api/v1/control/actions/${encodeURIComponent(a.id)}/inspect`}
          className="text-[11px] font-semibold text-muted-foreground hover:text-foreground"
          target="_blank"
          rel="noopener noreferrer"
        >
          Inspect API (new tab)
        </a>
      </div>
    </li>
  )
}

function ProofpackCompletenessPanel({ inc }: { inc: Incident }) {
  const versionInfo = useVersionInfo()
  const er = operatorExportReadinessFromVersion(versionInfo.data, versionInfo.error ?? null)
  const exportBlocked = er.semantic === 'policy_limited' || er.semantic === 'unknown_partial'
  const exportBlockedReason = er.summary

  const trace = inc.intelligence?.action_outcome_trace
  const wirelessGaps = inc.intelligence?.wireless_context?.evidence_gaps ?? []
  const sparsityMarkers = inc.intelligence?.sparsity_markers ?? []
  const isDegraded = inc.intelligence?.degraded === true
  const degradedReasons = inc.intelligence?.degraded_reasons ?? []

  const completeness = trace?.completeness ?? (isDegraded ? 'partial' : 'unavailable')
  const variant = proofpackCompletenessVariant(completeness)

  const allGaps: string[] = [
    ...wirelessGaps,
    ...sparsityMarkers,
    ...degradedReasons.map((r) => toWords(r)),
  ].filter(Boolean)

  const snapshotTotal = trace?.expected_snapshot_writes ?? 0
  const snapshotPersisted = trace?.persisted_snapshot_count ?? 0
  const snapshotFailed = trace?.snapshot_write_failures ?? 0
  const pct = snapshotTotal > 0 ? Math.round((snapshotPersisted / snapshotTotal) * 100) : null

  return (
    <div className="rounded-xl border border-border/60 bg-muted/10 p-4 space-y-3">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
          <Shield className="h-3.5 w-3.5" />
          Proofpack completeness
        </div>
        <Badge variant={variant}>{toWords(completeness)}</Badge>
      </div>

      {trace && (
        <div className="space-y-2">
          {pct !== null && (
            <div>
              <div className="flex justify-between text-[11px] text-muted-foreground mb-1">
                <span>Snapshot coverage</span>
                <span>{snapshotPersisted}/{snapshotTotal} ({pct}%)</span>
              </div>
              <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                <div
                  className={clsx('h-full rounded-full transition-all', pct === 100 ? 'bg-success' : pct > 50 ? 'bg-warning' : 'bg-critical')}
                  style={{ width: `${pct}%` }}
                />
              </div>
            </div>
          )}
          <div className="flex flex-wrap gap-2 text-xs">
            <span className="text-muted-foreground">Retrieval: <span className="text-foreground">{trace.snapshot_retrieval_status}</span></span>
            {snapshotFailed > 0 && <Badge variant="warning">write failures: {snapshotFailed}</Badge>}
          </div>
          {trace.snapshot_retrieval_error && (
            <p className="text-xs text-warning">{trace.snapshot_retrieval_error}</p>
          )}
        </div>
      )}

      {allGaps.length > 0 && (
        <div className="rounded-lg border border-warning/25 bg-warning/5 px-3 py-2 text-xs">
          <p className="font-medium text-foreground mb-1">Evidence gaps / sparsity</p>
          <ul className="space-y-0.5">
            {allGaps.slice(0, 6).map((g, i) => (
              <li key={i} className="flex items-start gap-1.5 text-muted-foreground">
                <AlertTriangle className="h-3 w-3 shrink-0 mt-0.5 text-warning" />
                {g}
              </li>
            ))}
          </ul>
        </div>
      )}

      {completeness === 'unavailable' && !trace && (
        <p className="text-xs text-muted-foreground">
          No snapshot traceability data available for this incident. Proofpack may still export available evidence.
        </p>
      )}

      <ProofpackButton
        incidentId={inc.id}
        exportBlocked={exportBlocked}
        exportBlockedReason={exportBlockedReason}
      />
      <p className="text-[11px] text-muted-foreground">
        Pair with{' '}
        <a href="#shift-continuity-handoff" className="font-medium text-primary hover:underline">
          handoff / escalation
        </a>{' '}
        for continuity narrative; proofpack remains the stronger incident evidence bundle when policy allows.
      </p>
      <p className="text-[11px] text-muted-foreground">
        For host/runtime continuity (not incident proof), use{' '}
        <Link to="/diagnostics" className="font-medium text-primary hover:underline">
          Diagnostics → support bundle
        </Link>
        .
      </p>
    </div>
  )
}

// ─── Replay timeline ──────────────────────────────────────────────────────────

function replayInterpretationHuman(posture: string | undefined): string {
  switch (posture) {
    case 'timeline_query_capped':
      return 'Timeline query hit the row cap — older or concurrent events may be missing; this strip is not a complete history.'
    case 'no_timeline_rows_in_window':
      return 'No DB timeline rows in the bounded window — quiet here can mean pruning, wrong ref, or an empty slice; it is not proof nothing happened elsewhere.'
    case 'sparse_evidence_window':
      return 'Very few persisted events in-window — treat the sequence as weak evidence; widen time or check transports/diagnostics.'
    case 'bounded_persistence_view':
      return 'Bounded persistence view — sequence is what MEL stored for this window, not guaranteed completeness.'
    default:
      return posture ? posture.replace(/_/g, ' ') : ''
  }
}

function ReplayTimeline({ segments, truthNote, generatedAt, replayMeta, incidentId }: {
  segments: ReplaySegment[]
  truthNote?: string
  generatedAt?: string
  replayMeta?: ReplayView['replay_meta']
  incidentId?: string
}) {
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const [filter, setFilter] = useState<ReplayFilterId>('all')
  const [newestFirst, setNewestFirst] = useState(false)

  const filtered = useMemo(() => {
    if (filter === 'all') return segments
    return segments.filter((s) => replayFilterBucket(s) === filter)
  }, [segments, filter])

  const ordered = useMemo(() => {
    if (!newestFirst) return filtered
    return [...filtered].reverse()
  }, [filtered, newestFirst])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement || e.target instanceof HTMLSelectElement) return
      if (!e.altKey) return
      const n = Number.parseInt(e.key, 10)
      if (n >= 0 && n < REPLAY_FILTER_OPTIONS.length) {
        e.preventDefault()
        setFilter(REPLAY_FILTER_OPTIONS[n]!.id)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  function toggle(i: number) {
    setExpanded((prev) => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }

  const interpLine = replayInterpretationHuman(replayMeta?.interpretation_posture)

  if (segments.length === 0) {
    return (
      <div className="rounded-xl border border-border/60 bg-muted/10 p-4">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2">
          <GitBranch className="h-3.5 w-3.5" />
          Incident replay / timeline
        </div>
        <p className="text-sm text-muted-foreground">
          No timeline events in the replay window ({replayMeta?.window_from ?? '—'} → {replayMeta?.window_to ?? '—'}). Evidence may have been pruned, notes may use a different ref, or the incident is outside the bounded window.
        </p>
        {replayMeta?.window_truncated && (
          <p className="mt-2 text-xs text-warning font-medium" role="status">
            Window query capped — timeline may be truncated at the fetch limit.
          </p>
        )}
        {interpLine && (
          <p className="mt-2 text-xs text-foreground/90 border-l-2 border-warning/50 pl-2.5" role="status">
            {interpLine}
          </p>
        )}
        {replayMeta?.linked_control_redacted && replayMeta?.visibility_note && (
          <p className="mt-2 text-[11px] text-muted-foreground border-l-2 border-border pl-2.5">{replayMeta.visibility_note}</p>
        )}
        {incidentId && (
          <p className="mt-3 text-[11px] text-muted-foreground flex flex-wrap gap-x-3 gap-y-1">
            <Link to={`/diagnostics`} className="font-medium text-primary hover:underline">
              Diagnostics / support bundle
            </Link>
            <span className="text-muted-foreground/50">·</span>
            <Link to={`/topology?incident=${encodeURIComponent(incidentId)}&filter=incident_focus`} className="font-medium text-primary hover:underline">
              Topology (incident focus)
            </Link>
            <span className="text-muted-foreground/50">·</span>
            <Link to={`/incidents/${encodeURIComponent(incidentId)}`} className="font-medium text-primary hover:underline">
              Incident detail
            </Link>
          </p>
        )}
        {replayMeta?.ordering_posture_note && (
          <p className="mt-2 text-[11px] text-muted-foreground/70 border-l-2 border-muted pl-2.5">{replayMeta.ordering_posture_note}</p>
        )}
      </div>
    )
  }

  return (
    <div className="rounded-xl border border-border/60 bg-muted/10 p-4 space-y-3" role="region" aria-label="Incident replay timeline">
      <div className="flex flex-wrap items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
        <GitBranch className="h-3.5 w-3.5" />
        Incident replay / timeline
        <span className="ml-auto font-normal normal-case tracking-normal">
          {ordered.length}{filter !== 'all' ? ` / ${segments.length}` : ''} events
        </span>
      </div>

      {replayMeta && (
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-[10px] text-muted-foreground/80 font-mono">
          {replayMeta.window_from && replayMeta.window_to && (
            <span>Window: {replayMeta.window_from.slice(0, 16)}… → {replayMeta.window_to.slice(0, 16)}…</span>
          )}
          {replayMeta.timeline_event_count != null && (
            <span>DB timeline rows: {replayMeta.timeline_event_count}</span>
          )}
          {replayMeta.sparse_timeline && (
            <span className="text-warning">Sparse timeline</span>
          )}
          {replayMeta.window_truncated && <span className="text-warning">Query capped</span>}
        </div>
      )}
      {interpLine && (
        <p className="text-xs text-foreground/90 border-l-2 border-warning/40 pl-2.5 leading-snug" role="status">
          {interpLine}
          {filter !== 'all' && ' Filtered view is not representative of the full window.'}
        </p>
      )}
      {replayMeta?.linked_control_redacted && replayMeta?.visibility_note && (
        <p className="text-[11px] text-muted-foreground border-l-2 border-border pl-2.5">{replayMeta.visibility_note}</p>
      )}
      {incidentId && (
        <p className="text-[11px] text-muted-foreground flex flex-wrap gap-x-3 gap-y-1">
          <span className="text-muted-foreground/70">If replay is weak for decisions:</span>
          <Link to="/diagnostics" className="font-medium text-primary hover:underline">
            Diagnostics
          </Link>
          <span className="text-muted-foreground/40">·</span>
          <Link to={`/topology?incident=${encodeURIComponent(incidentId)}&filter=incident_focus`} className="font-medium text-primary hover:underline">
            Topology
          </Link>
        </p>
      )}

      <div className="flex flex-wrap items-center gap-1.5" role="toolbar" aria-label="Replay filters">
        <span className="sr-only">
          Filter shortcuts: hold Alt and press 0 through 6 to select a filter when focus is not in a text field.
        </span>
        {REPLAY_FILTER_OPTIONS.map((o, idx) => (
          <button
            key={o.id}
            type="button"
            onClick={() => setFilter(o.id)}
            title={`Alt+${idx}`}
            aria-pressed={filter === o.id}
            className={clsx(
              'rounded-md border min-h-[36px] min-w-[36px] sm:min-h-0 sm:min-w-0 px-2 py-1.5 sm:py-1 text-[10px] font-semibold transition-colors touch-manipulation',
              filter === o.id
                ? 'border-primary/50 bg-primary/10 text-foreground'
                : 'border-border/50 bg-card/40 text-muted-foreground hover:border-border/80',
            )}
          >
            {o.label}
            <span className="ml-1 font-mono text-muted-foreground/50" aria-hidden>
              {idx}
            </span>
          </button>
        ))}
        <button
          type="button"
          onClick={() => setNewestFirst((v) => !v)}
          aria-pressed={newestFirst}
          className="ml-auto rounded-md border border-border/50 bg-card/40 min-h-[36px] px-2 py-1.5 sm:py-1 text-[10px] font-semibold text-muted-foreground hover:border-border/80 touch-manipulation"
        >
          {newestFirst ? 'Order: newest first' : 'Order: oldest first'}
        </button>
      </div>

      {truthNote && (
        <p className="text-[11px] text-muted-foreground/70 border-l-2 border-muted pl-2.5 leading-snug">
          {truthNote}
        </p>
      )}
      {replayMeta?.ordering_posture_note && (
        <p className="text-[10px] text-muted-foreground/60 border-l-2 border-border/40 pl-2.5 leading-snug">
          {replayMeta.ordering_posture_note}
        </p>
      )}

      {filter !== 'all' && ordered.length === 0 && (
        <p className="text-xs text-muted-foreground">No events in this filter; try &quot;All&quot;.</p>
      )}

      <div className="relative">
        <div className="absolute left-[6px] top-0 bottom-0 w-px bg-border/60" aria-hidden />

        <ol className="space-y-0">
          {ordered.map((seg, i) => {
            const isExp = expanded.has(i)
            const hasRefs = (seg.evidence_refs?.length ?? 0) > 0
            const hasDetails = seg.details != null && Object.keys(seg.details).length > 0
            return (
              <li key={`${seg.event_id ?? 'ev'}-${i}`} className="relative flex gap-3 pl-6 pb-4 last:pb-0">
                <Circle className={clsx('absolute left-0 h-3.5 w-3.5 shrink-0 fill-current top-0.5', postureColor(seg.knowledge_posture))} />

                <div className="flex-1 min-w-0">
                  <div className="flex flex-wrap items-start gap-x-3 gap-y-0.5">
                    <span className="text-xs font-medium text-foreground leading-snug">{seg.summary || toWords(seg.event_type)}</span>
                    <Badge variant="outline" className="text-[9px] px-1 py-0 font-mono normal-case tracking-normal">
                      {eventClassShortLabel(seg)}
                    </Badge>
                    <span className={clsx('text-[10px] font-semibold uppercase tracking-wide shrink-0', postureColor(seg.knowledge_posture))}>
                      {postureLabel(seg.knowledge_posture)}
                    </span>
                  </div>
                  <div className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[11px] text-muted-foreground">
                    <span className="inline-flex items-center gap-1" title={seg.event_time ? formatTimestamp(seg.event_time) : undefined}>
                      <Clock className="h-3 w-3" />
                      {seg.event_time ? (
                        <>
                          <span>{formatRelativeTime(seg.event_time)}</span>
                          <span className="text-muted-foreground/50 hidden sm:inline">· {formatTimestamp(seg.event_time)}</span>
                        </>
                      ) : '—'}
                    </span>
                    {seg.actor_id && <span className="text-muted-foreground/70">actor {seg.actor_id}</span>}
                    {seg.timing_posture && seg.timing_posture !== 'local_ordered' && (
                      <span className="text-warning/80 text-[10px]">timing: {seg.timing_posture}</span>
                    )}
                    {seg.scope_posture && seg.scope_posture !== 'local' && (
                      <span className="text-[10px] text-muted-foreground/70">scope: {seg.scope_posture}</span>
                    )}
                    <code className="text-muted-foreground/60">{toWords(seg.event_type)}</code>
                    {(hasRefs || hasDetails) && (
                      <button
                        type="button"
                        onClick={() => toggle(i)}
                        className="inline-flex items-center gap-0.5 text-[10px] text-muted-foreground hover:text-foreground"
                        aria-expanded={isExp}
                      >
                        {isExp ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                        details
                      </button>
                    )}
                  </div>
                  {isExp && hasRefs && (
                    <ul className="mt-1.5 space-y-0.5">
                      {seg.evidence_refs!.map((ref) => (
                        <li key={ref} className="text-[10px] font-mono text-muted-foreground/70">{ref}</li>
                      ))}
                    </ul>
                  )}
                  {isExp && hasDetails && (
                    <pre className="mt-1.5 max-h-32 overflow-auto rounded border border-border/40 bg-muted/20 p-2 text-[10px] font-mono text-muted-foreground/80 whitespace-pre-wrap">
                      {JSON.stringify(seg.details, null, 2)}
                    </pre>
                  )}
                </div>
              </li>
            )
          })}
        </ol>
      </div>

      {generatedAt && (
        <p className="text-[10px] text-muted-foreground/50">Replay assembled {formatTimestamp(generatedAt)}</p>
      )}
    </div>
  )
}

// ─── Section wrapper ──────────────────────────────────────────────────────────

function Section({ title, icon, children }: { title: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
        {icon}
        {title}
      </div>
      {children}
    </div>
  )
}

function buildHandoffStructured(inc: Incident, opts?: { canReadLinkedActions?: boolean }) {
  const intel = inc.intelligence
  const canReadLinked = opts?.canReadLinkedActions !== false
  const actionVis = resolvedIncidentActionVisibility(inc, { canReadLinkedActions: canReadLinked })
  const memoryDecisionCueCompact = incidentMemoryDecisionCue(inc)
  const gaps = [
    ...(intel?.wireless_context?.evidence_gaps ?? []),
    ...(intel?.sparsity_markers ?? []),
  ]
  const similar = (intel?.similar_incidents ?? []).slice(0, 5).map((s) => ({
    incident_id: s.incident_id,
    title: s.title,
    state: s.state,
    occurred_at: s.occurred_at,
    weighted_score: s.weighted_score,
    insufficient_evidence: s.insufficient_evidence,
    match_explanation: s.match_explanation?.slice(0, 4),
    matched_dimensions: s.matched_dimensions,
  }))
  const memoryCompact = intel
    ? {
        signature_match_count: intel.signature_match_count,
        signature_label: intel.signature_label,
        reopened_from_incident_id: inc.reopened_from_incident_id,
        reopened_at: inc.reopened_at,
        action_outcome_framings: (intel.action_outcome_memory ?? []).slice(0, 6).map((m) => ({
          action_type: m.action_type,
          outcome_framing: m.outcome_framing,
          evidence_strength: m.evidence_strength,
          sample_size: m.sample_size,
        })),
        governance_summaries: (intel.governance_memory ?? []).slice(0, 4).map((g) => ({
          action_type: g.action_type,
          summary: g.summary,
          linked_action_count: g.linked_action_count,
        })),
        historical_action_types: (intel.historically_used_actions ?? []).slice(0, 8),
        drift_fingerprint_count: intel.drift_fingerprints?.length ?? 0,
        correlation_group_count: intel.correlation_groups?.length ?? 0,
      }
    : undefined
  return {
    kind: 'mel_handoff_summary/v1',
    generated_note: 'Structured continuity snapshot — not a proofpack; use Export proofpack or escalation bundle for evidence chain.',
    incident: {
      id: inc.id,
      title: inc.title,
      state: inc.state,
      review_state: inc.review_state,
      severity: inc.severity,
      category: inc.category,
      resource: { type: inc.resource_type, id: inc.resource_id },
      occurred_at: inc.occurred_at,
      updated_at: inc.updated_at,
      resolved_at: inc.resolved_at,
      owner_actor_id: inc.owner_actor_id,
      reopened_from_incident_id: inc.reopened_from_incident_id,
      reopened_at: inc.reopened_at,
    },
    narrative: {
      summary: inc.summary,
      handoff_summary: inc.handoff_summary,
      investigation_notes: inc.investigation_notes,
      resolution_summary: inc.resolution_summary,
      lessons_learned: inc.lessons_learned,
      closeout_reason: inc.closeout_reason,
    },
    intelligence_posture: intel
      ? {
          evidence_strength: intel.evidence_strength,
          degraded: intel.degraded,
          degraded_reasons: intel.degraded_reasons,
          signature_label: intel.signature_label,
          signature_match_count: intel.signature_match_count,
        }
      : undefined,
    uncertainty: {
      evidence_and_sparsity_gaps: gaps,
    },
    operator_truth_compact: {
      action_visibility_kind: actionVis.kind,
      action_visibility_reason: inc.action_visibility?.action_visibility_reason,
      action_context_explanation: actionVis.explanation,
      memory_decision_cue: memoryDecisionCueCompact ?? undefined,
    },
    pending: {
      pending_action_ids: inc.pending_actions?.filter(Boolean) ?? [],
    },
    next_checks: (intel?.investigate_next ?? []).slice(0, 8).map((g) => ({
      id: g.id,
      title: g.title,
      rationale: g.rationale,
      confidence: g.confidence,
    })),
    similar_incidents_compact: similar,
    operational_memory_compact: memoryCompact,
    deep_links: {
      incident: `/incidents/${inc.id}`,
      control_actions: `/control-actions?incident=${encodeURIComponent(inc.id)}`,
      topology: `/topology?incident=${encodeURIComponent(inc.id)}&filter=incident_focus`,
      planning: `/planning?incident=${encodeURIComponent(inc.id)}`,
      replay: `/incidents/${encodeURIComponent(inc.id)}?replay=1`,
      diagnostics_support_bundle: '/diagnostics',
    },
  }
}

function buildHandoffExportText(inc: Incident): string {
  const intel = inc.intelligence
  const lines: string[] = [
    'MEL — incident handoff summary (paste into ticket or runbook)',
    `Incident: ${inc.title || inc.id}`,
    `ID: ${inc.id}`,
    `State (system): ${inc.state || 'unknown'}`,
    `Review / workflow: ${inc.review_state || 'open'}`,
    `Severity: ${inc.severity || '—'}`,
    `Occurred: ${inc.occurred_at || '—'}`,
    `Updated: ${inc.updated_at || '—'}`,
    `Resource: ${inc.resource_type || '—'} / ${inc.resource_id || '—'}`,
    '',
    'What we know (bounded):',
    inc.summary || '(no summary)',
    '',
  ]
  if (intel?.evidence_strength) {
    lines.push(`Evidence strength (intel): ${intel.evidence_strength}`)
  }
  if (intel?.signature_match_count != null && intel.signature_match_count > 1) {
    lines.push(`Signature recurrence (instance history): ${intel.signature_match_count} matches — structural bucket, not causal.`)
  }
  if (intel?.degraded) {
    lines.push('Intelligence degraded: yes (treat guidance as non-causal)')
    if (intel.degraded_reasons?.length) {
      lines.push(`Reasons: ${intel.degraded_reasons.join('; ')}`)
    }
  }
  if ((intel?.sparsity_markers?.length ?? 0) > 0) {
    lines.push(`Sparsity markers: ${intel!.sparsity_markers!.join('; ')}`)
  }
  const memCue = incidentMemoryDecisionCue(inc)
  if (memCue) {
    lines.push('')
    lines.push('What history changes next (deterministic cue):')
    lines.push(memCue)
  }
  lines.push('')
  lines.push('Recorded handoff narrative:')
  lines.push(inc.handoff_summary || '(none)')
  lines.push('')
  lines.push('Investigation notes:')
  lines.push(inc.investigation_notes || '(none)')
  lines.push('')
  lines.push('Resolution / closeout (if any):')
  lines.push(inc.resolution_summary || '(none)')
  if (inc.lessons_learned) lines.push(`Lessons: ${inc.lessons_learned}`)
  if (inc.closeout_reason) lines.push(`Closeout: ${inc.closeout_reason}`)
  lines.push('')
  lines.push('Pending action IDs (referenced on incident):')
  const p = inc.pending_actions?.filter(Boolean) ?? []
  if (p.length === 0) lines.push('(none listed)')
  else for (const id of p) lines.push(`- ${id}`)
  lines.push('')
  lines.push('Similar prior incidents (deterministic / explainable in UI):')
  const sim = intel?.similar_incidents ?? []
  if (sim.length === 0) {
    lines.push('(none listed — may be sparse history)')
  } else {
    for (const s of sim.slice(0, 5)) {
      const expl = (s.match_explanation?.length ? s.match_explanation.join('; ') : s.similarity_reason?.join('; ')) || 'see incident detail'
      lines.push(`- ${s.incident_id} state=${s.state ?? '?'} score=${s.weighted_score != null ? s.weighted_score.toFixed(2) : 'n/a'} weak=${s.insufficient_evidence ? 'yes' : 'no'}`)
      lines.push(`  ${expl}`)
    }
  }
  lines.push('')
  lines.push('What remains uncertain:')
  if ((intel?.wireless_context?.evidence_gaps?.length ?? 0) > 0) {
    for (const g of intel!.wireless_context!.evidence_gaps!) lines.push(`- ${g}`)
  } else {
    lines.push('- See proofpack evidence_gaps and intelligence panels in MEL.')
  }
  lines.push('')
  lines.push('Next checks (suggested):')
  const next = intel?.investigate_next?.slice(0, 5) ?? []
  if (next.length === 0) {
    lines.push('- Open replay/timeline and topology for this incident window in MEL.')
  } else {
    for (const g of next) lines.push(`- ${g.title}: ${g.rationale}`)
  }
  lines.push('')
  lines.push(`Deep link: /incidents/${inc.id}`)
  lines.push('This paste export is a snapshot; canonical evidence lives in MEL proofpack / DB. Use “Handoff JSON” or escalation bundle for machine-readable continuity.')
  return lines.join('\n')
}

function DecisionPackPanel({
  pack,
  inc,
  onSaved,
}: {
  pack: IncidentDecisionPack | undefined
  inc: Incident
  onSaved: () => void
}) {
  const ctx = useOperatorContext()
  const { addToast } = useToast()
  const adj = pack?.operator_adjudication
  const [reviewed, setReviewed] = useState(adj?.reviewed ?? false)
  const [useful, setUseful] = useState(adj?.useful === 'not_useful' ? 'not_useful' : adj?.useful === 'useful' ? 'useful' : '')
  const [note, setNote] = useState(adj?.operator_note ?? '')
  const [cueAccepted, setCueAccepted] = useState(
    adj?.cue_outcomes?.find((c) => c.cue_id === 'operator_suggested_actions')?.outcome === 'accepted',
  )
  const [cueDismissed, setCueDismissed] = useState(
    adj?.cue_outcomes?.find((c) => c.cue_id === 'operator_suggested_actions')?.outcome === 'dismissed',
  )
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const a = pack?.operator_adjudication
    setReviewed(a?.reviewed ?? false)
    setUseful(a?.useful === 'not_useful' ? 'not_useful' : a?.useful === 'useful' ? 'useful' : '')
    setNote(a?.operator_note ?? '')
    const cue = a?.cue_outcomes?.find((c) => c.cue_id === 'operator_suggested_actions')
    setCueAccepted(cue?.outcome === 'accepted')
    setCueDismissed(cue?.outcome === 'dismissed')
  }, [inc.id, pack?.generated_at, pack?.operator_adjudication])

  const canWrite = ctx.trustUI?.incident_mutate === true

  async function saveAdjudication() {
    setSaving(true)
    try {
      const cueOutcomes: { cue_id: string; outcome: string }[] = []
      if (cueAccepted) cueOutcomes.push({ cue_id: 'operator_suggested_actions', outcome: 'accepted' })
      else if (cueDismissed) cueOutcomes.push({ cue_id: 'operator_suggested_actions', outcome: 'dismissed' })
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(inc.id)}/decision-pack`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          reviewed,
          useful: useful || undefined,
          operator_note: note.trim() || undefined,
          cue_outcomes: cueOutcomes,
          replace_cue_outcomes: true,
        }),
      })
      if (res.status === 403) {
        addToast({ type: 'error', title: 'Cannot save', message: 'Missing incident update capability.' })
        return
      }
      if (!res.ok) {
        addToast({ type: 'error', title: 'Save failed', message: `HTTP ${res.status}` })
        return
      }
      addToast({ type: 'success', title: 'Saved', message: 'Decision pack feedback stored on this instance.' })
      await onSaved()
    } catch {
      addToast({ type: 'error', title: 'Save failed', message: 'Network error.' })
    } finally {
      setSaving(false)
    }
  }

  if (!pack) {
    return (
      <AlertCard
        variant="warning"
        title="Incident Decision Pack unavailable"
        description="This response did not include decision_pack — refresh or check API version."
      />
    )
  }

  const q = pack.queue
  const triage = q?.triage_signals
  const readiness = pack.readiness
  const unc = pack.uncertainty

  return (
    <Card data-testid="incident-decision-pack" id="mel-incident-decision-pack" className="scroll-mt-20">
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Incident Decision Pack</CardTitle>
        <p className="text-xs text-muted-foreground mt-1">
          Schema {pack.schema_version} · assembled {pack.generated_at ? formatRelativeTime(pack.generated_at) : '—'} — backend-owned snapshot; not reinterpreted in the UI.
        </p>
      </CardHeader>
      <CardContent className="space-y-4 pt-0">
        {q?.why_surfaced_one_liner && (
          <div className="rounded-lg border border-primary/25 bg-primary/5 px-3 py-2.5 text-sm">
            <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground mb-1">Why this is surfaced</p>
            <p className="text-foreground leading-snug">{q.why_surfaced_one_liner}</p>
            {q.ordering_note && <p className="mt-2 text-[11px] text-muted-foreground border-l-2 border-border/50 pl-2">{q.ordering_note}</p>}
          </div>
        )}

        {triage?.codes?.length ? (
          <div className="rounded-lg border border-border/50 bg-muted/10 px-3 py-2 text-xs" data-testid="decision-pack-triage">
            <p className="font-semibold text-foreground mb-1">Queue / triage basis</p>
            <p className="text-[11px] text-muted-foreground mb-2">
              Tier {triage.tier}
              {triage.queue_ordering_contract ? (
                <>
                  {' '}
                  · <span className="font-mono">{triage.queue_ordering_contract}</span>
                </>
              ) : null}
            </p>
            <ul className="space-y-1">
              {triage.codes.slice(0, 8).map((code, i) => (
                <li key={code} className="text-[11px] text-muted-foreground border-l-2 border-primary/20 pl-2">
                  <span className="font-mono text-foreground/90">{code.replace(/_/g, ' ')}</span>
                  {triage.rationale_lines?.[i] ? ` — ${triage.rationale_lines[i]}` : ''}
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        {readiness && (
          <div className="rounded-lg border border-border/50 px-3 py-2 text-xs space-y-1" data-testid="decision-pack-readiness">
            <p className="font-semibold text-foreground">Export / support posture (policy snapshot)</p>
            <p className="text-muted-foreground">{readiness.export_policy_summary}</p>
            {readiness.evidence_sufficiency_note && (
              <p className="text-[11px] text-warning border-l-2 border-warning/25 pl-2">{readiness.evidence_sufficiency_note}</p>
            )}
            <p className="text-[10px] text-muted-foreground/80 font-mono break-all">
              {readiness.proofpack_path} · {readiness.escalation_bundle_path}
            </p>
          </div>
        )}

        {unc && (unc.non_claims?.length || unc.bounded_scan_disclosures?.length) ? (
          <div className="rounded-lg border border-warning/20 bg-warning/5 px-3 py-2 text-xs space-y-2">
            <p className="font-semibold text-foreground">Uncertainty & bounded scans</p>
            {unc.bounded_scan_disclosures?.map((line) => (
              <p key={line} className="text-[11px] text-muted-foreground">{line}</p>
            ))}
            <ul className="list-disc pl-4 text-[11px] text-muted-foreground space-y-0.5">
              {(unc.non_claims ?? []).slice(0, 4).map((n) => (
                <li key={n}>{n}</li>
              ))}
            </ul>
          </div>
        ) : null}

        <div className="rounded-lg border border-border/60 bg-muted/10 px-3 py-3 space-y-3">
          <p className="text-xs font-semibold text-foreground">Operator adjudication (this pack)</p>
          <p className="text-[11px] text-muted-foreground">
            Local feedback for institutional memory — does not execute controls or imply team workflow.
          </p>
          <label className="flex items-center gap-2 text-xs cursor-pointer">
            <input
              type="checkbox"
              checked={reviewed}
              onChange={(e) => setReviewed(e.target.checked)}
              disabled={!canWrite}
            />
            Mark decision pack reviewed
          </label>
          <div className="flex flex-wrap gap-3 text-xs">
            <label className="inline-flex items-center gap-1.5">
              <input
                type="radio"
                name={`pack-useful-${inc.id}`}
                checked={useful === 'useful'}
                onChange={() => setUseful('useful')}
                disabled={!canWrite}
              />
              Useful
            </label>
            <label className="inline-flex items-center gap-1.5">
              <input
                type="radio"
                name={`pack-useful-${inc.id}`}
                checked={useful === 'not_useful'}
                onChange={() => setUseful('not_useful')}
                disabled={!canWrite}
              />
              Not useful
            </label>
          </div>
          <div>
            <label className="text-[11px] font-semibold text-muted-foreground">Note (optional)</label>
            <textarea
              className="mt-1 w-full min-h-[72px] rounded-lg border border-border/60 bg-background px-2 py-1.5 text-xs"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              disabled={!canWrite}
              placeholder="Why this pack helped or misled — stays on this instance."
            />
          </div>
          <div className="space-y-1">
            <p className="text-[11px] font-semibold text-muted-foreground">Suggested next checks cue</p>
            <label className="flex items-center gap-2 text-xs">
              <input type="checkbox" checked={cueAccepted} onChange={(e) => { setCueAccepted(e.target.checked); if (e.target.checked) setCueDismissed(false) }} disabled={!canWrite} />
              Accept cue (will use suggested actions)
            </label>
            <label className="flex items-center gap-2 text-xs">
              <input type="checkbox" checked={cueDismissed} onChange={(e) => { setCueDismissed(e.target.checked); if (e.target.checked) setCueAccepted(false) }} disabled={!canWrite} />
              Dismiss cue
            </label>
          </div>
          {!canWrite && (
            <p className="text-[11px] text-warning">Read-only session — adjudication requires incident_mutate.</p>
          )}
          <button
            type="button"
            className="button-secondary text-xs"
            disabled={!canWrite || saving}
            onClick={() => void saveAdjudication()}
          >
            {saving ? 'Saving…' : 'Save pack feedback'}
          </button>
          {adj?.reviewed_at && (
            <p className="text-[10px] text-muted-foreground">
              Last saved {formatRelativeTime(adj.reviewed_at)}
              {adj.reviewed_by_actor_id ? ` · ${adj.reviewed_by_actor_id}` : ''}
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function WorkflowPanel({ inc, onSaved, returnTo }: { inc: Incident; onSaved: () => void; returnTo: string }) {
  const ctx = useOperatorContext()
  const { addToast } = useToast()
  const [reviewState, setReviewState] = useState(inc.review_state || 'open')
  const [notes, setNotes] = useState(inc.investigation_notes || '')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setReviewState(inc.review_state || 'open')
    setNotes(inc.investigation_notes || '')
  }, [inc.id, inc.review_state, inc.investigation_notes])

  const canWrite = ctx.trustUI?.incident_mutate === true

  async function saveWorkflow() {
    setSaving(true)
    try {
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(inc.id)}/workflow`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          review_state: reviewState,
          investigation_notes: notes,
        }),
      })
      if (res.status === 403) {
        addToast({ type: 'error', title: 'Cannot save', message: 'Missing incident update capability.' })
        return
      }
      if (!res.ok) {
        addToast({ type: 'error', title: 'Save failed', message: `HTTP ${res.status}` })
        return
      }
      addToast({ type: 'success', title: 'Workflow saved', message: 'Review state and notes persisted locally on this MEL instance.' })
      await onSaved()
    } catch {
      addToast({ type: 'error', title: 'Save failed', message: 'Network error.' })
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">Workflow & investigation</CardTitle>
        <p className="text-xs text-muted-foreground mt-1">
          Review state is operator workflow on this instance (single-operator honest mode). It does not imply multi-user coordination.
        </p>
      </CardHeader>
      <CardContent className="space-y-3 pt-0">
        {!canWrite && (
          <p className="text-xs text-warning border border-warning/25 rounded-lg px-3 py-2 bg-warning/5">
            Read-only: your session cannot PATCH incident workflow. Notes and state changes require incident_mutate.
          </p>
        )}
        <div>
          <label htmlFor="mel-inc-review-state" className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            Review state
          </label>
          <select
            id="mel-inc-review-state"
            className="mt-1 w-full rounded-lg border border-border/60 bg-background px-3 py-2 text-sm"
            value={reviewState}
            onChange={(e) => setReviewState(e.target.value)}
            disabled={!canWrite}
          >
            {WORKFLOW_REVIEW_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="mel-inc-notes" className="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            Investigation notes
          </label>
          <textarea
            id="mel-inc-notes"
            className="mt-1 w-full min-h-[100px] rounded-lg border border-border/60 bg-background px-3 py-2 text-sm"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            disabled={!canWrite}
            placeholder="Observed facts, hypotheses (labeled), what you checked…"
          />
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => void saveWorkflow()}
            disabled={!canWrite || saving}
            className="button-secondary text-xs"
          >
            {saving ? 'Saving…' : 'Save workflow'}
          </button>
          <Link
            to={`/control-actions?incident=${encodeURIComponent(inc.id)}&return=${encodeURIComponent(`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnTo)}`)}`}
            className="inline-flex items-center gap-1 rounded-lg border border-border/60 px-3 py-1.5 text-xs font-semibold hover:bg-muted/50"
          >
            <Zap className="h-3.5 w-3.5" />
            Control actions for this incident
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}

function HandoffExportPanel({ inc }: { inc: Incident }) {
  const versionInfo = useVersionInfo()
  const opCtx = useOperatorContext()
  const canReadLinked = operatorCanReadLinkedControlRows({
    loading: opCtx.loading,
    error: opCtx.error,
    trustUI: opCtx.trustUI,
    capabilities: opCtx.capabilities ?? [],
  })
  const actionVis = resolvedIncidentActionVisibility(inc, { canReadLinkedActions: canReadLinked })
  const text = buildHandoffExportText(inc)
  const structured = buildHandoffStructured(inc, { canReadLinkedActions: canReadLinked })
  const jsonText = JSON.stringify(structured, null, 2)
  const [escState, setEscState] = useState<'idle' | 'loading' | 'error'>('idle')
  const [escErr, setEscErr] = useState('')
  const er = operatorExportReadinessFromVersion(versionInfo.data, versionInfo.error ?? null)
  const exportBlockedByPolicy = er.semantic === 'policy_limited'
  const policyUnknown = er.semantic === 'unknown_partial'
  const escalationLikelyBlocked = exportBlockedByPolicy || policyUnknown

  async function downloadEscalationBundle() {
    if (escalationLikelyBlocked) return
    setEscState('loading')
    setEscErr('')
    try {
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(inc.id)}/escalation-bundle`)
      if (!res.ok) {
        setEscErr(res.status === 403 ? 'Export disabled by policy or insufficient permissions.' : `HTTP ${res.status}`)
        setEscState('error')
        return
      }
      const data = await res.json()
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `escalation-bundle-${inc.id.slice(0, 12)}.json`
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      setEscState('idle')
    } catch {
      setEscErr('Network error.')
      setEscState('error')
    }
  }

  return (
    <Card id="shift-continuity-handoff" className="scroll-mt-20">
      <CardHeader className="pb-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="text-base">Shift continuity / handoff</CardTitle>
          <div className="flex flex-wrap gap-2">
            <CopyButton value={text} label="Copy plain summary" className="button-secondary text-xs" />
            <CopyButton value={jsonText} label="Copy handoff JSON" className="button-secondary text-xs" />
          </div>
        </div>
        <p className="text-xs text-muted-foreground mt-1">
          Plain text for chat/tickets; JSON is a structured continuity snapshot. Neither replaces the proofpack for canonical evidence.
        </p>
      </CardHeader>
      <CardContent className="pt-0 space-y-3">
        {versionInfo.error && (
          <p className="text-xs text-warning border border-warning/25 rounded-lg px-3 py-2 bg-warning/5" role="alert">
            Version / policy fetch failed ({versionInfo.error}). Export gates may be unknown — prefer plain handoff until Settings loads.
          </p>
        )}
        {!versionInfo.loading && (
          <div
            className="rounded-lg border border-border/40 bg-background/50 px-3 py-2 text-[11px] text-muted-foreground"
            role="status"
          >
            <span className="font-semibold text-foreground">This instance: </span>
            {er.semantic === 'policy_limited' ? (
              <span className="text-warning">
                Incident evidence export disabled by policy — escalation bundle and proofpack may be blocked; use plain handoff where
                allowed.
              </span>
            ) : er.semantic === 'unknown_partial' ? (
              <span className="text-warning">{er.summary}</span>
            ) : (
              <span>
                Export/delete policy is active — scope and caveats live under Settings (runtime truth). Review{' '}
                <code className="font-mono text-[10px]">evidence_gaps</code> before treating any bundle as complete proof.
              </span>
            )}
          </div>
        )}
        <div className="rounded-lg border border-border/50 bg-muted/15 px-3 py-2.5 text-[11px] text-muted-foreground space-y-1.5">
          <p className="font-semibold text-foreground">Decision ladder (under pressure)</p>
          <ol className="list-decimal pl-4 space-y-1.5">
            <li>
              <span className="text-foreground">Runtime, broker, or version unclear?</span>{' '}
              <Link to="/settings" className="text-primary font-medium hover:underline">
                Settings
              </Link>{' '}
              and{' '}
              <Link to="/diagnostics" className="text-primary font-medium hover:underline">
                Diagnostics
              </Link>{' '}
              first — handoff text does not fix a broken local posture.
            </li>
            <li>
              <span className="text-foreground">Need a human-readable pass-down?</span> Plain summary or handoff JSON — fastest continuity;
              still not proof.
            </li>
            <li>
              <span className="text-foreground">Need support / vendor with incident-shaped context?</span> Escalation bundle when policy
              allows — heavier than handoff; not a substitute for{' '}
              <Link to="/diagnostics" className="text-primary font-medium hover:underline">
                host/runtime diagnostics
              </Link>{' '}
              when the problem is the runtime itself.{' '}
              <span className="text-warning/90">
                Avoid leaning on it as “proof” when evidence is sparse — label gaps in your ticket.
              </span>
            </li>
            <li>
              <span className="text-foreground">Need strongest incident evidence MEL can bundle?</span> Proofpack (below in this page) —
              skip if policy blocks export or version metadata is unknown; use handoff + Settings to confirm gates first.
            </li>
            <li>
              <span className="text-foreground">Need process, build, disk, or broker truth?</span>{' '}
              <Link to="/diagnostics" className="text-primary font-medium hover:underline">
                Diagnostics → support bundle
              </Link>{' '}
              — does not replace proofpack for incident evidence chain.
            </li>
          </ol>
          <ul className="list-disc pl-4 space-y-1 border-t border-border/30 pt-2 mt-2 text-[10px]">
            <li>
              <span className="text-foreground font-medium">Avoid proofpack / escalation</span> when export is disabled or policy is unknown
              — you get predictable failure or empty legal scope; use plain handoff + runtime truth.
            </li>
            <li>
              <span className="text-foreground font-medium">Avoid handoff-only</span> when the blocker is clearly local runtime or broker
              health — diagnostics and Settings carry the failing fact, not the incident narrative.
            </li>
            <li>
              <span className="text-foreground font-medium">Control / action context partial?</span>{' '}
              <a href="#linked-control-actions" className="text-primary font-medium hover:underline">
                Linked actions panel
              </a>{' '}
              and filtered queue — {actionVis.explanation}
            </li>
          </ul>
        </div>
        {inc.intelligence?.evidence_strength === 'sparse' && (
          <p className="text-[11px] text-warning border border-warning/20 rounded-lg px-3 py-2 bg-warning/5" role="status">
            Sparse incident evidence — any bundle is weaker; prefer widening replay/topology/control context before implying completeness to
            support.
          </p>
        )}
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => void downloadEscalationBundle()}
            disabled={escState === 'loading' || escalationLikelyBlocked}
            title={
              escalationLikelyBlocked
                ? exportBlockedByPolicy
                  ? 'Disabled: instance policy blocks evidence export'
                  : 'Disabled: export policy unknown — confirm in Settings first'
                : undefined
            }
            className="button-secondary text-xs inline-flex items-center gap-1.5 min-h-[44px] sm:min-h-0 touch-manipulation"
          >
            <Download className="h-3.5 w-3.5" />
            {escState === 'loading' ? 'Downloading…' : 'Download escalation bundle'}
          </button>
          <span className="text-[10px] text-muted-foreground/70">
            {escalationLikelyBlocked
              ? exportBlockedByPolicy
                ? 'Not offered while evidence export is disabled — avoids a predictable 403.'
                : 'Held back until export policy is known — avoids silent failure.'
              : 'Includes proofpack assembly summary + linked control rows when export policy allows.'}
          </span>
        </div>
        {escState === 'error' && escErr && <p className="text-xs text-critical">{escErr}</p>}
        <pre className="max-h-48 overflow-auto rounded-lg border border-border/50 bg-muted/20 p-3 text-[11px] text-muted-foreground whitespace-pre-wrap font-mono">
          {text}
        </pre>
      </CardContent>
    </Card>
  )
}

// ─── Main page ────────────────────────────────────────────────────────────────

export function IncidentDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { setFocus: setWorkspaceFocus } = useOperatorWorkspaceFocus()
  const versionInfo = useVersionInfo()

  const returnToWorkbench = useMemo(() => {
    const raw = (searchParams.get('return') || '').trim()
    if (raw.startsWith('/')) return raw
    return id ? `/incidents?focus=${encodeURIComponent(id)}` : '/incidents'
  }, [searchParams, id])

  const exportReadiness = operatorExportReadinessFromVersion(versionInfo.data, versionInfo.error ?? null)

  const [inc, setInc] = useState<Incident | null>(null)
  const [replay, setReplay] = useState<ReplayView | null>(null)
  const [loading, setLoading] = useState(true)
  const [replayLoading, setReplayLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [replayError, setReplayError] = useState<string | null>(null)
  const [replayOpen, setReplayOpen] = useState(() => searchParams.get('replay') === '1')

  const load = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(id)}`)
      if (res.status === 404) throw new Error('Incident not found.')
      if (res.status === 403) throw new Error('Insufficient permissions.')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as Incident
      setInc(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load incident')
    } finally {
      setLoading(false)
    }
  }, [id])

  const loadReplay = useCallback(async () => {
    if (!id) return
    setReplayLoading(true)
    setReplayError(null)
    try {
      const res = await fetch(`/api/v1/incidents/${encodeURIComponent(id)}/replay`)
      if (!res.ok) {
        if (res.status === 503) {
          setReplayError('Replay not available on this instance.')
        } else {
          setReplayError(`HTTP ${res.status}`)
        }
        return
      }
      setReplay(await res.json() as ReplayView)
    } catch {
      setReplayError('Could not load replay data.')
    } finally {
      setReplayLoading(false)
    }
  }, [id])

  useEffect(() => { void load() }, [load])

  useEffect(() => {
    if (!inc) return
    setWorkspaceFocus({
      incidentId: inc.id,
      incidentTitle: inc.title?.trim() || undefined,
      savedAt: new Date().toISOString(),
    })
  }, [inc, setWorkspaceFocus])

  useEffect(() => {
    if (searchParams.get('replay') === '1') {
      setReplayOpen(true)
    }
  }, [searchParams])

  useEffect(() => {
    if (!replayOpen || !id) return
    if (replay || replayLoading || replayError) return
    void loadReplay()
  }, [replayOpen, id, replay, replayLoading, replayError, loadReplay])

  function handleReplayOpen() {
    setReplayOpen(true)
    setSearchParams((prev) => {
      const n = new URLSearchParams(prev)
      n.set('replay', '1')
      return n
    }, { replace: true })
  }

  if (loading) return <Loading message="Loading incident…" />

  if (error) {
    return (
      <div className="p-8 max-w-2xl mx-auto space-y-4">
        <Link to={returnToWorkbench} className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="h-4 w-4" /> Back to workbench
        </Link>
        <AlertCard variant="critical" title="Could not load incident" description={error} action={
          <button type="button" onClick={() => void load()} className="button-secondary text-xs">Retry</button>
        } />
      </div>
    )
  }

  if (!inc) return null

  const intel = inc.intelligence
  const hasIntel = !!intel
  const seenBefore = (intel?.signature_match_count ?? 0) > 1
  const replaySegs = replay?.replay_segments ?? replay?.knowledge_timeline ?? []
  const outcomesInTimeline = replaySegs.some((s) => s.event_type === 'recommendation_outcome')

  return (
    <div className="max-w-4xl mx-auto space-y-5 pb-12">
      {/* Back nav */}
      <div className="flex items-center gap-3 flex-wrap">
        <button
          type="button"
          onClick={() => navigate(returnToWorkbench)}
          className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to workbench
        </button>
        <Link to={returnToWorkbench} className="text-sm text-muted-foreground hover:text-foreground">
          Workbench
        </Link>
        <span className="text-muted-foreground/40">/</span>
        <span className="text-sm text-foreground font-mono truncate max-w-[200px]">{inc.id.slice(0, 16)}…</span>
        <button
          type="button"
          onClick={() => void load()}
          className="ml-auto inline-flex items-center gap-1.5 rounded-lg border border-border/60 bg-card/50 px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground"
        >
          <RefreshCw className="h-3.5 w-3.5" />
          Refresh
        </button>
      </div>

      {!versionInfo.loading && (
        <div
          className={clsx(
            'rounded-xl border px-3 py-2.5 text-xs',
            exportReadiness.semantic === 'available'
              ? 'border-success/25 bg-success/5 text-muted-foreground'
              : exportReadiness.semantic === 'policy_limited'
                ? 'border-critical/30 bg-critical/5 text-foreground'
                : exportReadiness.semantic === 'degraded'
                  ? 'border-warning/30 bg-warning/5 text-foreground'
                  : 'border-warning/25 bg-warning/5 text-foreground',
          )}
          role="status"
          aria-live="polite"
          data-testid="incident-export-readiness"
        >
          <span className="font-semibold text-foreground">Export / bundle readiness: </span>
          {exportReadiness.summary}
          {exportReadiness.blockers.length > 0 && (
            <ul className="mt-2 list-disc pl-4 text-[11px] text-muted-foreground space-y-0.5">
              {exportReadiness.blockers.map((b) => (
                <li key={b.code}>
                  <span className="font-mono text-foreground/80">{b.code}</span>
                  {b.summary ? ` — ${b.summary}` : ''}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <DecisionPackPanel pack={inc.decision_pack} inc={inc} onSaved={() => void load()} />

      {/* Header card */}
      <Card id="incident-operational-summary" className="scroll-mt-20">
        <CardHeader className="pb-3">
          <div className="flex flex-wrap items-start gap-3">
            <AlertTriangle className={clsx('h-5 w-5 shrink-0 mt-0.5', inc.severity === 'critical' ? 'text-critical' : inc.severity === 'high' ? 'text-warning' : 'text-muted-foreground')} />
            <div className="flex-1 min-w-0">
              <CardTitle className="text-lg leading-snug">{inc.title || inc.id}</CardTitle>
              <div className="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span className="font-mono">{inc.id.slice(0, 20)}</span>
                {inc.occurred_at && (
                  <span className="inline-flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    {formatRelativeTime(inc.occurred_at)}
                  </span>
                )}
                {inc.owner_actor_id && (
                  <span className="inline-flex items-center gap-1">
                    <User className="h-3 w-3" />
                    {inc.owner_actor_id}
                  </span>
                )}
              </div>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {inc.state && <Badge variant={stateVariant(inc.state)}>{inc.state}</Badge>}
              {inc.severity && <Badge variant={severityVariant(inc.severity)}>{inc.severity}</Badge>}
              {hasIntel && (
                <span title={evidenceStrengthLabel(intel.evidence_strength)} className="inline-flex">
                  <Badge variant={evidenceStrengthVariant(intel.evidence_strength)}>{intel.evidence_strength} evidence</Badge>
                </span>
              )}
              {seenBefore && <Badge variant="warning">seen {intel!.signature_match_count}x</Badge>}
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-0 space-y-4">
          {inc.summary && (
            <p className="text-sm leading-relaxed text-muted-foreground">{inc.summary}</p>
          )}
          {hasIntel && (
            <div className="flex flex-wrap gap-2">
              {intel.signature_label && (
                <Badge variant="outline"><Activity className="h-3 w-3" />{intel.signature_label}</Badge>
              )}
              {intel.wireless_context && (
                <Badge variant="outline">{toWords(intel.wireless_context.classification)}</Badge>
              )}
              {(intel.similar_incidents?.length ?? 0) > 0 && (
                <Badge variant="secondary">{intel.similar_incidents!.length} similar prior incidents</Badge>
              )}
            </div>
          )}
          <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-muted-foreground border-t border-border/40 pt-3">
            {inc.occurred_at && <span>Occurred: {formatTimestamp(inc.occurred_at)}</span>}
            {inc.updated_at && <span>Updated: {formatTimestamp(inc.updated_at)}</span>}
            {inc.resolved_at && <span>Resolved: {formatTimestamp(inc.resolved_at)}</span>}
            {inc.category && <span>Category: {inc.category}</span>}
            {inc.review_state && <span>Review: {toWords(inc.review_state)}</span>}
          </div>
        </CardContent>
      </Card>

      <InvestigationPathPanel inc={inc} returnTo={returnToWorkbench} />

      <MeshRoutingCompanionStrip inc={inc} />

      <OperatorSuggestedActionsPanel inc={inc} />

      <OperationalMemoryPanel inc={inc} />

      {/* Proofpack completeness */}
      <ProofpackCompletenessPanel inc={inc} />

      <LinkedControlActionsPanel inc={inc} returnTo={returnToWorkbench} />

      {hasIntel && <InvestigationGuidePanel inc={inc} returnTo={returnToWorkbench} />}

      <div className="grid gap-5 lg:grid-cols-2">
        <WorkflowPanel inc={inc} onSaved={() => void load()} returnTo={returnToWorkbench} />
        <HandoffExportPanel inc={inc} />
      </div>

      {/* Two-column body */}
      <div className="grid gap-5 lg:grid-cols-2">

        {/* Handoff + investigation */}
        <div className="space-y-4">
          {/* Handoff summary */}
          <Section title="Handoff summary" icon={<FileText className="h-3.5 w-3.5" />}>
            <div className={clsx(
              'rounded-lg border px-3 py-2.5 text-sm',
              inc.handoff_summary ? 'border-border/60 bg-card/50' : 'border-dashed border-border/50 bg-muted/20 text-muted-foreground'
            )}>
              {inc.handoff_summary || 'No handoff summary recorded.'}
            </div>
          </Section>

          {/* Investigation notes */}
          {inc.investigation_notes && (
            <Section title="Investigation notes" icon={<Eye className="h-3.5 w-3.5" />}>
              <div className="rounded-lg border border-border/60 bg-card/50 px-3 py-2.5 text-sm whitespace-pre-wrap">{inc.investigation_notes}</div>
            </Section>
          )}

          {/* Resolution */}
          {(inc.resolution_summary || inc.lessons_learned || inc.closeout_reason) && (
            <Section title="Resolution" icon={<CheckCircle2 className="h-3.5 w-3.5" />}>
              <div className="space-y-2 rounded-lg border border-border/60 bg-card/50 px-3 py-2.5 text-sm">
                {inc.resolution_summary && <p>{inc.resolution_summary}</p>}
                {inc.lessons_learned && (
                  <p className="text-muted-foreground"><span className="font-medium text-foreground">Lessons: </span>{inc.lessons_learned}</p>
                )}
                {inc.closeout_reason && (
                  <p className="text-muted-foreground"><span className="font-medium text-foreground">Closeout: </span>{toWords(inc.closeout_reason)}</p>
                )}
              </div>
            </Section>
          )}

          {/* Risks */}
          {(inc.risks?.length ?? 0) > 0 && (
            <Section title="Risks" icon={<AlertTriangle className="h-3.5 w-3.5" />}>
              <ul className="space-y-1">
                {inc.risks!.map((r, i) => (
                  <li key={i} className="flex items-start gap-2 text-xs text-muted-foreground">
                    <AlertTriangle className="h-3 w-3 shrink-0 mt-0.5 text-warning" />
                    {r}
                  </li>
                ))}
              </ul>
            </Section>
          )}

          {/* Referenced actions */}
          {(inc.pending_actions?.filter(Boolean).length ?? 0) > 0 && (
            <Section title="Referenced action IDs" icon={<Zap className="h-3.5 w-3.5" />}>
              <div className="space-y-1.5">
                {inc.pending_actions!.filter(Boolean).map((actionId) => (
                  <div key={actionId} className="flex items-center gap-2 rounded-lg border border-border/50 bg-card/40 px-2.5 py-1.5 text-xs">
                    <code className="flex-1 truncate font-mono text-muted-foreground">{actionId.slice(0, 24)}…</code>
                  </div>
                ))}
                <Link
                  to={`/control-actions?incident=${encodeURIComponent(inc.id)}&return=${encodeURIComponent(`/incidents/${encodeURIComponent(inc.id)}?return=${encodeURIComponent(returnToWorkbench)}`)}`}
                  className="inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline"
                >
                  View control actions for this incident →
                </Link>
              </div>
            </Section>
          )}
        </div>

        {/* Intelligence */}
        <div className="space-y-4">
          {hasIntel && (
            <>
              {/* Similar incidents */}
              {(intel.similar_incidents?.length ?? 0) > 0 && (
                <Section title="Similar prior incidents" icon={<Link2 className="h-3.5 w-3.5" />}>
                  <div id="similar-prior-incidents" className="-mt-2 mb-2 scroll-mt-24" />
                  <div className="space-y-1.5">
                    {intel.similar_incidents!.map((s) => (
                      <Link
                        key={s.incident_id}
                        to={`/incidents/${s.incident_id}`}
                        className="block rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs hover:border-border/80 hover:bg-card/70 transition-colors"
                      >
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-mono text-muted-foreground shrink-0">{s.incident_id.slice(0, 12)}</span>
                          {s.title && <span className="flex-1 min-w-0 truncate font-medium text-foreground">{s.title}</span>}
                          {s.state && <Badge variant={s.state === 'resolved' ? 'success' : 'secondary'}>{s.state}</Badge>}
                          {s.insufficient_evidence && <Badge variant="warning">weak match</Badge>}
                          {s.match_category && <Badge variant="outline">{toWords(s.match_category)}</Badge>}
                          {s.weighted_score != null && (
                            <span className="text-muted-foreground/60 shrink-0 font-mono" title="Deterministic fingerprint score; not ML confidence">
                              {(s.weighted_score * 100).toFixed(0)}%
                            </span>
                          )}
                        </div>
                        {(s.match_explanation?.length || s.similarity_reason?.length) ? (
                          <ul className="mt-1.5 space-y-0.5 text-[10px] text-muted-foreground leading-snug border-t border-border/30 pt-1.5">
                            {(s.match_explanation ?? s.similarity_reason ?? []).slice(0, 4).map((line, i) => (
                              <li key={i} className="flex gap-1.5">
                                <span className="text-muted-foreground/40 shrink-0">·</span>
                                <span>{line}</span>
                              </li>
                            ))}
                          </ul>
                        ) : null}
                      </Link>
                    ))}
                  </div>
                </Section>
              )}

              {/* Investigate next */}
              {(intel.investigate_next?.length ?? 0) > 0 && (
                <Section title="Investigate next" icon={<HelpCircle className="h-3.5 w-3.5" />}>
                  <div className="space-y-1.5">
                    {intel.investigate_next!.slice(0, 5).map((g) => (
                      <div key={g.id} className="rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="font-medium text-foreground">{g.title}</p>
                          <span title={guidanceConfidenceLabel(g.confidence)} className="inline-flex">
                            <Badge variant="outline">{g.confidence} confidence</Badge>
                          </span>
                        </div>
                        <p className="mt-0.5 text-muted-foreground">{g.rationale}</p>
                      </div>
                    ))}
                  </div>
                </Section>
              )}

              {/* Runbook recommendations */}
              {(intel.runbook_recommendations?.length ?? 0) > 0 && (
                <Section title="Runbook recommendations" icon={<BookOpen className="h-3.5 w-3.5" />}>
                  <div className="space-y-1.5">
                    {intel.runbook_recommendations!.slice(0, 4).map((r) => (
                      <div key={r.id} className="rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs">
                        <div className="flex items-center gap-2">
                          {r.rank_score != null && <span className="text-muted-foreground shrink-0">{r.rank_score.toFixed(2)}</span>}
                          <span className="font-medium text-foreground flex-1">{r.title}</span>
                          <Badge variant="outline">{toWords(r.strength)}</Badge>
                        </div>
                        <p className="mt-1 text-[10px] text-muted-foreground leading-snug">{runbookStrengthOperatorLabel(r.strength)}</p>
                        {r.rationale && <p className="mt-1 text-muted-foreground">{r.rationale}</p>}
                      </div>
                    ))}
                  </div>
                </Section>
              )}

              {/* Action outcome memory */}
              {(intel.action_outcome_memory?.length ?? 0) > 0 && (
                <Section title="Historical action outcomes" icon={<Zap className="h-3.5 w-3.5" />}>
                  <p className="text-[11px] text-muted-foreground mb-1.5">Association only — does not establish causality.</p>
                  <div className="space-y-2">
                    {intel.action_outcome_memory!.map((m) => (
                      <div key={m.action_type} className="rounded-lg border border-border/50 bg-card/40 p-3 text-xs">
                        <div className="flex flex-wrap items-center gap-1.5">
                          <span className="font-medium text-foreground">{m.action_label || toWords(m.action_type)}</span>
                          <Badge variant="outline">n={m.sample_size}</Badge>
                          <Badge variant={m.outcome_framing === 'improvement_observed' ? 'success' : m.outcome_framing === 'deterioration_observed' ? 'critical' : 'secondary'}>
                            {toWords(m.outcome_framing)}
                          </Badge>
                        </div>
                        <div className="mt-1.5 flex flex-wrap gap-3 text-muted-foreground">
                          <span className="inline-flex items-center gap-1"><CheckCircle2 className="h-3 w-3 text-success" /> {m.improvement_observed_count} improved</span>
                          <span className="inline-flex items-center gap-1"><XCircle className="h-3 w-3 text-critical" /> {m.deterioration_observed_count} deteriorated</span>
                          <span className="inline-flex items-center gap-1"><HelpCircle className="h-3 w-3" /> {m.inconclusive_count} inconclusive</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </Section>
              )}

              {/* Degraded warning */}
              {intel.degraded && (
                <div className="rounded-lg border border-warning/30 bg-warning/5 px-3 py-2.5 text-xs">
                  <div className="flex items-start gap-2">
                    <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-warning" />
                    <div>
                      <p className="font-medium text-foreground">Intelligence limited by available evidence</p>
                      <p className="mt-0.5 text-muted-foreground">Treat as investigative guidance, not causal proof.</p>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Replay timeline section */}
      <div>
        {!replayOpen ? (
          <button
            type="button"
            onClick={handleReplayOpen}
            className="inline-flex items-center gap-2 rounded-xl border border-border/60 bg-muted/20 px-4 py-2.5 text-sm font-medium text-muted-foreground hover:bg-muted/40 hover:text-foreground transition-colors"
          >
            <GitBranch className="h-4 w-4" />
            Load incident replay / timeline
          </button>
        ) : replayLoading ? (
          <div className="rounded-xl border border-border/60 bg-muted/10 p-4 text-sm text-muted-foreground">
            Assembling replay…
          </div>
        ) : replayError ? (
          <AlertCard variant="warning" title="Replay unavailable" description={replayError} />
        ) : replay ? (
          <ReplayTimeline
            segments={replaySegs}
            truthNote={replay.truth_note}
            generatedAt={replay.generated_at}
            replayMeta={replay.replay_meta}
            incidentId={id}
          />
        ) : null}
      </div>

      {/* Recommendation outcomes (from replay) */}
      {replay && (replay.recommendation_outcomes?.length ?? 0) > 0 && !outcomesInTimeline && (
        <div className="rounded-xl border border-border/60 bg-muted/10 p-4 space-y-3">
          <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
            <CheckCircle2 className="h-3.5 w-3.5" />
            Recommendation outcomes recorded
          </div>
          <div className="space-y-2">
            {replay.recommendation_outcomes!.map((o) => (
              <div key={o.id} className="flex flex-wrap items-center gap-2 rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs">
                <code className="font-mono text-muted-foreground">{o.recommendation_id.slice(0, 16)}</code>
                <Badge variant={outcomeVariant(o.outcome)}>{toWords(o.outcome)}</Badge>
                {o.actor_id && <span className="text-muted-foreground">by {o.actor_id}</span>}
                {o.note && <span className="text-muted-foreground italic">{o.note}</span>}
                <span className="ml-auto text-muted-foreground/60">{formatRelativeTime(o.created_at)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
