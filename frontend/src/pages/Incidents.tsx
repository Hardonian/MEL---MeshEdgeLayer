import { useIncidents } from '@/hooks/useIncidents'
import { useOperatorContext } from '@/hooks/useOperatorContext'
import { PageHeader } from '@/components/ui/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatTimestamp, formatRelativeTime, type Incident } from '@/types/api'
import {
  ClipboardCopy,
  Download,
  RefreshCw,
  AlertTriangle,
  Clock,
  User,
  ArrowRight,
  ChevronDown,
  ChevronUp,
  Eye,
  Shield,
  Zap,
  CheckCircle2,
  XCircle,
  HelpCircle,
  Activity,
  FileText,
  Link2,
} from 'lucide-react'
import { clsx } from 'clsx'
import { useState } from 'react'
import { Link } from 'react-router-dom'

function isOpenIncident(inc: Incident): boolean {
  const s = (inc.state || '').toLowerCase()
  return s !== 'resolved' && s !== 'closed'
}

function copyText(text: string) {
  void navigator.clipboard.writeText(text)
}

function toWords(value: string | undefined): string {
  return (value || '').replace(/_/g, ' ').trim()
}

function outcomeFramingLabel(value: string | undefined): string {
  switch (value) {
    case 'improvement_observed':
      return 'Improvement observed'
    case 'deterioration_observed':
      return 'Deterioration observed'
    case 'mixed_historical_evidence':
      return 'Mixed evidence'
    case 'insufficient_evidence':
      return 'Insufficient evidence'
    case 'no_clear_post_action_signal':
      return 'No clear signal'
    default:
      return toWords(value) || 'Unknown'
  }
}

function wirelessClassificationLabel(value: string | undefined): string {
  switch (value) {
    case 'lora_mesh_pressure':
      return 'LoRa / frequency pressure'
    case 'wifi_backhaul_instability':
      return 'Wi-Fi backhaul instability'
    case 'mixed_path_degradation':
      return 'Mixed-path degradation'
    case 'sparse_evidence_incident':
      return 'Sparse evidence'
    case 'unsupported_wireless_domain_observed':
      return 'Unsupported wireless domain'
    case 'recurring_unknown_pattern':
      return 'Recurring unknown pattern'
    default:
      return toWords(value) || 'Unclassified'
  }
}

function humanizeReasonCode(value: string | undefined): string {
  const text = toWords(value)
  if (!text) return 'No additional context'
  return text.charAt(0).toUpperCase() + text.slice(1)
}

function snapshotCompletenessTone(value: string | undefined): 'secondary' | 'warning' | 'outline' {
  if (value === 'partial') return 'warning'
  if (value === 'complete') return 'secondary'
  return 'outline'
}

function defaultProofpackFilename(incidentId: string): string {
  return `proofpack-${incidentId || 'incident'}.json`
}

function filenameFromDisposition(contentDisposition: string | null, fallback: string): string {
  if (!contentDisposition) return fallback
  const match = contentDisposition.match(/filename\*?=(?:UTF-8''|")?([^\";]+)/i)
  if (!match || !match[1]) return fallback
  const value = match[1].replace(/\"/g, '').trim()
  try {
    return decodeURIComponent(value)
  } catch {
    return value || fallback
  }
}

function evidenceStrengthVariant(strength: string | undefined): 'success' | 'warning' | 'secondary' {
  if (strength === 'strong') return 'success'
  if (strength === 'moderate') return 'warning'
  return 'secondary'
}


export function Incidents() {
  const { data, loading, error, refresh } = useIncidents()
  const ctx = useOperatorContext()

  if (loading && !data) {
    return <Loading message="Loading incidents..." />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load incidents"
          description={error}
          action={
            <button
              type="button"
              onClick={() => void refresh()}
              className="button-danger"
            >
              Retry
            </button>
          }
        />
      </div>
    )
  }

  const incidents = data || []
  const openIncidents = incidents.filter(isOpenIncident)
  const closedIncidents = incidents.filter((i) => !isOpenIncident(i))
  const canHandoff = ctx.trustUI?.incident_handoff_write === true

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <PageHeader
          title="Incidents"
          description="Mesh / link / transport disruptions with durable handoff context."
        />
        <button
          type="button"
          onClick={() => {
            void refresh()
            void ctx.refresh()
          }}
          className="button-secondary"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      {ctx.error && (
        <AlertCard variant="warning" title="Operator context unavailable" description={ctx.error} />
      )}

      {!canHandoff && !ctx.loading && (
        <div className="flex items-center gap-2 rounded-xl border border-info/20 bg-info/5 px-4 py-2.5 text-xs text-muted-foreground">
          <Eye className="h-3.5 w-3.5 text-info" />
          Read-only view. Your credentials do not include incident_handoff_write.
        </div>
      )}

      {/* Summary stats */}
      <div className="flex flex-wrap gap-3">
        <div className={clsx(
          'flex items-center gap-2 rounded-full border px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.16em]',
          openIncidents.length > 0 ? 'border-warning/25 bg-warning/8 text-warning' : 'border-success/20 bg-success/8 text-success'
        )}>
          <span className={clsx('h-1.5 w-1.5 rounded-full', openIncidents.length > 0 ? 'bg-warning' : 'bg-success')} />
          {openIncidents.length} open
        </div>
        <div className="flex items-center gap-2 rounded-full border border-border/60 bg-muted/30 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
          {closedIncidents.length} resolved
        </div>
        <div className="flex items-center gap-2 rounded-full border border-border/60 bg-muted/30 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
          {incidents.length} total
        </div>
      </div>

      {openIncidents.length === 0 ? (
        <EmptyState
          type="no-data"
          title="No open incidents"
          description={
            incidents.length === 0
              ? 'No incidents in the recent list. When transport or system disruptions are detected, they appear here with intelligence and handoff context.'
              : 'All recent incidents are resolved or closed.'
          }
        />
      ) : (
        <div className="space-y-4">
          {openIncidents.map((inc) => (
            <IncidentCard key={inc.id} incident={inc} canMutate={canMutate} />
          ))}
        </div>
      )}

      {closedIncidents.length > 0 && (
        <section className="space-y-3 pt-2">
          <h2 className="flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            <CheckCircle2 className="h-3.5 w-3.5" />
            Resolved incidents
          </h2>
          <div className="space-y-3">
            {closedIncidents.map((inc) => (
              <IncidentCard key={inc.id} incident={inc} muted />
            ))}
          </div>
        </section>
      )}
    </div>
  )
}

function ProofpackDownloadButton({ incidentId }: { incidentId: string }) {
  const [state, setState] = useState<'idle' | 'loading' | 'error'>('idle')
  const [errorMsg, setErrorMsg] = useState('')

  async function download() {
    setState('loading')
    setErrorMsg('')
    try {
      const resp = await fetch(`/api/v1/incidents/${encodeURIComponent(incidentId)}/proofpack?download=true`)
      if (!resp.ok) {
        const body = await resp.text().catch(() => '')
        if (resp.status === 401 || resp.status === 403) {
          setErrorMsg('Insufficient permissions for proofpack export.')
        } else if (resp.status === 404) {
          setErrorMsg('Incident not found.')
        } else {
          setErrorMsg(body || `HTTP ${resp.status}`)
        }
        setState('error')
        return
      }
      const blob = await resp.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filenameFromDisposition(
        resp.headers.get('content-disposition'),
        defaultProofpackFilename(incidentId),
      )
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      setState('idle')
    } catch {
      setErrorMsg('Network error — MEL backend unreachable.')
      setState('error')
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <button
        type="button"
        onClick={() => void download()}
        disabled={state === 'loading'}
        className="button-secondary text-xs"
        title="Download incident evidence proofpack (JSON)"
      >
        <Download className="h-3.5 w-3.5" />
        {state === 'loading' ? 'Assembling...' : 'Export proofpack'}
      </button>
      <span className="text-[10px] text-muted-foreground/60">
        Snapshot at request-time. Review evidence_gaps.
      </span>
      {state === 'error' && errorMsg && (
        <span className="text-xs text-critical">{errorMsg}</span>
      )}
    </div>
  )
}

function IncidentCard({ incident: inc, muted = false, canMutate = false }: { incident: Incident; muted?: boolean; canMutate?: boolean }) {
  void canMutate // reserved for future mutation controls
  const [expanded, setExpanded] = useState(!muted)
  const pending = inc.pending_actions?.filter(Boolean) ?? []
  const hasHandoffText = !!(inc.handoff_summary && inc.handoff_summary.trim())
  const owner = inc.owner_actor_id?.trim()
  const intel = inc.intelligence
  const hasIntel = !!intel
  const seenBefore = (intel?.signature_match_count ?? 0) > 1
  const hasSimilar = (intel?.similar_incidents?.length ?? 0) > 0

  const severityVariant = inc.severity === 'critical' ? 'critical' : inc.severity === 'high' ? 'warning' : 'secondary'
  const stateVariant = inc.state === 'resolved' || inc.state === 'closed' ? 'success' : 'outline'

  return (
    <Card
      className={clsx(
        muted && 'opacity-75',
        'transition-shadow hover:shadow-[0_20px_48px_-28px_hsl(var(--shell-shadow)/0.5)]'
      )}
    >
      {/* Header stripe */}
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <AlertTriangle className={clsx('h-4 w-4 shrink-0', inc.severity === 'critical' ? 'text-critical' : inc.severity === 'high' ? 'text-warning' : 'text-muted-foreground')} />
              <CardTitle className="text-base">{inc.title || inc.id}</CardTitle>
            </div>
            <div className="mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
              <span className="inline-flex items-center gap-1 font-mono">
                <Link2 className="h-3 w-3" />
                <a href={`/incidents/${encodeURIComponent(inc.id)}`} className="hover:underline">
                  {inc.id.slice(0, 12)}
                </a>
              </span>
              {inc.occurred_at && (
                <span className="inline-flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  {formatRelativeTime(inc.occurred_at)}
                </span>
              )}
              {owner && (
                <span className="inline-flex items-center gap-1">
                  <User className="h-3 w-3" />
                  {owner}
                </span>
              )}
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            {inc.state && <Badge variant={stateVariant as 'success' | 'outline'}>{inc.state}</Badge>}
            {inc.severity && <Badge variant={severityVariant as 'critical' | 'warning' | 'secondary'}>{inc.severity}</Badge>}
            {hasIntel && (
              <Badge variant={evidenceStrengthVariant(intel.evidence_strength)}>
                {intel.evidence_strength} evidence
              </Badge>
            )}
            {seenBefore && (
              <Badge variant="warning">
                seen {intel!.signature_match_count}x
              </Badge>
            )}
            <Link
              to={`/incidents/${inc.id}`}
              className="ml-1 inline-flex items-center gap-1 text-[11px] font-semibold text-primary hover:underline"
              title="Open incident detail page"
            >
              <ArrowRight className="h-3 w-3" />
              Detail
            </Link>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4 pt-0">
        {inc.summary && (
          <p className="text-sm leading-relaxed text-muted-foreground">{inc.summary}</p>
        )}

        {/* Quick intelligence snapshot — always visible */}
        {hasIntel && (
          <div className="flex flex-wrap gap-2">
            {intel.signature_label && (
              <Badge variant="outline">
                <Activity className="h-3 w-3" />
                {intel.signature_label}
              </Badge>
            )}
            {intel.wireless_context && (
              <Badge variant="outline">
                {wirelessClassificationLabel(intel.wireless_context.classification)}
              </Badge>
            )}
            {hasSimilar && (
              <Badge variant="secondary">
                {intel.similar_incidents!.length} similar prior
              </Badge>
            )}
          </div>
        )}

        {/* Expand/collapse toggle */}
        <button
          type="button"
          onClick={() => setExpanded(!expanded)}
          className="flex items-center gap-1 text-xs font-semibold text-muted-foreground transition-colors hover:text-foreground"
        >
          {expanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
          {expanded ? 'Collapse detail' : 'Expand detail'}
        </button>

        {expanded && (
          <div className="space-y-4 animate-fade-in">
            {/* Handoff summary */}
            <DetailSection title="Handoff summary" icon={<FileText className="h-3.5 w-3.5" />}>
              <div className={clsx(
                'rounded-lg border px-3 py-2 text-sm',
                hasHandoffText ? 'border-border/60 bg-card/50 text-foreground' : 'border-dashed border-border/50 bg-muted/20 text-muted-foreground'
              )}>
                {hasHandoffText ? inc.handoff_summary : 'No handoff summary recorded.'}
              </div>
            </DetailSection>

            {/* Proofpack export */}
            <DetailSection title="Evidence proofpack" icon={<Download className="h-3.5 w-3.5" />}>
              <ProofpackDownloadButton incidentId={inc.id} />
              <a href={`/incidents/${encodeURIComponent(inc.id)}`} className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline">
                Open full incident review <ArrowRight className="h-3 w-3" />
              </a>
            </DetailSection>

            {/* Referenced actions */}
            {pending.length > 0 && (
              <DetailSection title="Referenced action IDs" icon={<Zap className="h-3.5 w-3.5" />}>
                <div className="flex flex-wrap gap-2">
                  {pending.map((id) => (
                    <div
                      key={id}
                      className="flex items-center gap-2 rounded-lg border border-border/60 bg-card/50 px-2.5 py-1.5"
                    >
                      <code className="text-xs">{id.slice(0, 16)}...</code>
                      <button
                        type="button"
                        onClick={() => copyText(id)}
                        className="rounded p-0.5 text-muted-foreground hover:text-foreground"
                        title="Copy action ID"
                      >
                        <ClipboardCopy className="h-3 w-3" />
                      </button>
                    </div>
                  ))}
                </div>
                <Link to="/control-actions" className="mt-2 inline-flex items-center gap-1 text-xs font-semibold text-primary hover:underline">
                  View in control actions <ArrowRight className="h-3 w-3" />
                </Link>
              </DetailSection>
            )}

            {/* Intelligence deep dive */}
            {hasIntel && (
              <div className="space-y-3 rounded-xl border border-border/60 bg-muted/10 p-4">
                <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                  <Eye className="h-3.5 w-3.5" />
                  Incident intelligence
                </div>

                {/* Similar incidents */}
                {hasSimilar && (
                  <DetailSection title="Similar prior incidents" icon={<Link2 className="h-3.5 w-3.5" />}>
                    <div className="space-y-1.5">
                      {intel.similar_incidents!.map((s) => (
                        <div key={s.incident_id} className="flex items-center gap-3 rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs">
                          <span className="font-mono text-muted-foreground">{s.incident_id.slice(0, 12)}</span>
                          {s.title && <span className="flex-1 truncate text-foreground">{s.title}</span>}
                          {s.state && <Badge variant={s.state === 'resolved' ? 'success' : 'secondary'}>{s.state}</Badge>}
                          {s.occurred_at && <span className="text-muted-foreground/60">{formatRelativeTime(s.occurred_at)}</span>}
                        </div>
                      ))}
                    </div>
                  </DetailSection>
                )}

                {/* Wireless context */}
                {intel.wireless_context && (
                  <DetailSection title="Wireless context" icon={<Activity className="h-3.5 w-3.5" />}>
                    <div className="space-y-2 text-xs">
                      <div className="flex flex-wrap gap-1.5">
                        <Badge variant="outline">{wirelessClassificationLabel(intel.wireless_context.classification)}</Badge>
                        <Badge variant="secondary">confidence: {toWords(intel.wireless_context.confidence_posture)}</Badge>
                        <Badge variant="outline">evidence: {toWords(intel.wireless_context.evidence_posture)}</Badge>
                      </div>
                      {intel.wireless_context.summary && (
                        <p className="text-muted-foreground">{intel.wireless_context.summary}</p>
                      )}
                      {(intel.wireless_context.observed_domains?.length ?? 0) > 0 && (
                        <p className="text-muted-foreground">
                          Observed domains: {intel.wireless_context.observed_domains!.join(', ')}
                        </p>
                      )}
                      {(intel.wireless_context.reasons?.length ?? 0) > 0 && (
                        <ul className="space-y-1 pl-4">
                          {intel.wireless_context.reasons!.slice(0, 3).map((r) => (
                            <li key={r.code} className="list-disc text-muted-foreground">{r.statement}</li>
                          ))}
                        </ul>
                      )}
                      {(intel.wireless_context.evidence_gaps?.length ?? 0) > 0 && (
                        <EvidenceGapBanner gaps={intel.wireless_context.evidence_gaps!.slice(0, 3)} />
                      )}
                      {(intel.wireless_context.unsupported?.length ?? 0) > 0 && (
                        <div className="rounded-lg border border-border/60 bg-muted/20 px-3 py-2 text-muted-foreground">
                          <span className="font-medium text-foreground">Unsupported scope:</span>{' '}
                          {intel.wireless_context.unsupported!.map((u) => `${u.domain} ${u.scope}`).join(', ')}
                        </div>
                      )}
                    </div>
                  </DetailSection>
                )}

                {/* Investigate next */}
                {(intel.investigate_next?.length ?? 0) > 0 && (
                  <DetailSection title="Investigate next" icon={<HelpCircle className="h-3.5 w-3.5" />}>
                    <div className="space-y-1.5">
                      {intel.investigate_next!.slice(0, 3).map((g) => (
                        <div key={g.id} className="rounded-lg border border-border/50 bg-card/40 px-3 py-2 text-xs">
                          <p className="font-medium text-foreground">{g.title}</p>
                          <p className="mt-0.5 text-muted-foreground">{g.rationale}</p>
                        </div>
                      ))}
                    </div>
                  </DetailSection>
                )}

                {/* Action outcome memory */}
                {(intel.action_outcome_memory?.length ?? 0) > 0 && (
                  <DetailSection title="Historical action outcomes" icon={<Zap className="h-3.5 w-3.5" />}>
                    <p className="mb-2 text-[11px] text-muted-foreground">
                      Historical observations from similar incidents. Association only — does not establish causality.
                    </p>
                    <div className="space-y-2">
                      {intel.action_outcome_memory!.map((m) => (
                        <div key={m.action_type} className="rounded-lg border border-border/50 bg-card/40 p-3 text-xs">
                          <div className="flex flex-wrap items-center gap-1.5">
                            <span className="font-medium text-foreground">{m.action_label || m.action_type}</span>
                            <Badge variant="outline">n={m.sample_size}</Badge>
                            <Badge variant={m.outcome_framing === 'improvement_observed' ? 'success' : m.outcome_framing === 'deterioration_observed' ? 'critical' : 'secondary'}>
                              {outcomeFramingLabel(m.outcome_framing)}
                            </Badge>
                            {m.sample_size < 3 && <Badge variant="warning">sparse</Badge>}
                          </div>
                          <div className="mt-1.5 flex flex-wrap gap-3 text-muted-foreground">
                            <span className="inline-flex items-center gap-1">
                              <CheckCircle2 className="h-3 w-3 text-success" /> {m.improvement_observed_count} improved
                            </span>
                            <span className="inline-flex items-center gap-1">
                              <XCircle className="h-3 w-3 text-critical" /> {m.deterioration_observed_count} deteriorated
                            </span>
                            <span className="inline-flex items-center gap-1">
                              <HelpCircle className="h-3 w-3" /> {m.inconclusive_count} inconclusive
                            </span>
                          </div>
                          {(m.caveats?.length ?? 0) > 0 && (
                            <EvidenceGapBanner gaps={m.caveats!} label="Caveat" />
                          )}
                        </div>
                      ))}
                    </div>
                  </DetailSection>
                )}

                {/* Action outcome trace */}
                {intel.action_outcome_trace && (
                  <DetailSection title="Snapshot traceability" icon={<Shield className="h-3.5 w-3.5" />}>
                    <div className="flex flex-wrap items-center gap-1.5 text-xs">
                      <Badge variant={snapshotCompletenessTone(intel.action_outcome_trace.completeness)}>
                        {toWords(intel.action_outcome_trace.completeness)}
                      </Badge>
                      <Badge variant="outline">persisted: {intel.action_outcome_trace.persisted_snapshot_count}</Badge>
                      {intel.action_outcome_trace.snapshot_write_failures > 0 && (
                        <Badge variant="warning">write failures: {intel.action_outcome_trace.snapshot_write_failures}</Badge>
                      )}
                    </div>
                    {intel.action_outcome_trace.snapshot_retrieval_error && (
                      <p className="mt-1.5 text-xs text-warning">
                        Retrieval error: {intel.action_outcome_trace.snapshot_retrieval_error}
                      </p>
                    )}
                  </DetailSection>
                )}

                {/* Degraded intelligence warning */}
                {intel.degraded && (
                  <div className="rounded-lg border border-warning/30 bg-warning/5 px-3 py-2.5 text-xs">
                    <div className="flex items-start gap-2">
                      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-warning" />
                      <div>
                        <p className="font-medium text-foreground">
                          Intelligence limited by available evidence
                        </p>
                        <p className="mt-0.5 text-muted-foreground">
                          Treat as investigative guidance, not causal proof.
                        </p>
                        {(intel.degraded_reasons?.length ?? 0) > 0 && (
                          <ul className="mt-1.5 space-y-0.5">
                            {intel.degraded_reasons!.map((reason) => (
                              <li key={reason} className="text-muted-foreground">
                                <code className="rounded bg-muted/60 px-1 py-0.5 text-[10px]">{reason}</code>{' '}
                                {humanizeReasonCode(reason)}
                              </li>
                            ))}
                          </ul>
                        )}
                      </div>
                    </div>
                  </div>
                )}

                {intel.generated_at && (
                  <p className="text-[10px] text-muted-foreground/50">
                    Intelligence generated {formatTimestamp(intel.generated_at)}
                  </p>
                )}
              </div>
            )}

            {/* Metadata row */}
            <div className="flex flex-wrap gap-x-6 gap-y-2 border-t border-border/40 pt-3 text-xs text-muted-foreground">
              <span>Created: {formatTimestamp(inc.occurred_at)}</span>
              <span>Updated: {formatTimestamp(inc.updated_at)}</span>
              {inc.resolved_at && <span>Resolved: {formatTimestamp(inc.resolved_at)}</span>}
              {inc.category && <span>Category: {inc.category}</span>}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function DetailSection({
  title,
  icon,
  children,
}: {
  title: string
  icon: React.ReactNode
  children: React.ReactNode
}) {
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

function EvidenceGapBanner({ gaps, label = 'Evidence gap' }: { gaps: string[]; label?: string }) {
  return (
    <div className="rounded-lg border border-warning/25 bg-warning/5 px-3 py-2 text-xs text-muted-foreground">
      <span className="font-medium text-foreground">{label}:</span>{' '}
      {gaps.join(', ')}
    </div>
  )
}
