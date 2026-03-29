import { useIncidents } from '@/hooks/useIncidents'
import { useOperatorContext } from '@/hooks/useOperatorContext'
import { PageHeader } from '@/components/ui/PageHeader'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatTimestamp, type Incident } from '@/types/api'
import { ClipboardCopy, Download, RefreshCw } from 'lucide-react'
import { clsx } from 'clsx'
import { useState } from 'react'

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
      return 'Improvement observed in similar history'
    case 'deterioration_observed':
      return 'Deterioration observed in similar history'
    case 'mixed_historical_evidence':
      return 'Mixed historical evidence'
    case 'insufficient_evidence':
      return 'Insufficient historical evidence'
    case 'no_clear_post_action_signal':
      return 'No clear post-action signal'
    default:
      return toWords(value) || 'Historical signal unavailable'
  }
}

function observedStatusLabel(value: string | undefined): string {
  switch (value) {
    case 'mixed_signals':
      return 'Observed status: mixed signals'
    case 'inconclusive':
      return 'Observed status: inconclusive'
    case 'improvement_observed':
      return 'Observed status: improvement observed'
    case 'deterioration_observed':
      return 'Observed status: deterioration observed'
    default:
      return `Observed status: ${toWords(value) || 'unavailable'}`
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
      return 'Sparse evidence incident'
    case 'unsupported_wireless_domain_observed':
      return 'Unsupported wireless domain observed'
    case 'recurring_unknown_pattern':
      return 'Recurring unknown wireless pattern'
    default:
      return toWords(value) || 'Wireless classification unavailable'
  }
}

function humanizeReasonCode(value: string | undefined): string {
  const text = toWords(value)
  if (!text) return 'No additional context'
  return text.charAt(0).toUpperCase() + text.slice(1)
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


export function Incidents() {
  const { data, loading, error, refresh } = useIncidents()
  const ctx = useOperatorContext()

  if (loading && !data) {
    return <Loading message="Loading incidents…" />
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
              className="rounded-lg bg-critical px-4 py-2 text-sm font-medium text-white hover:bg-critical/90"
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
  const canHandoff = ctx.trustUI?.incident_handoff_write === true

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <PageHeader
          title="Incidents"
          description="Mesh / link / transport disruptions with durable handoff context. Pending action IDs are operator references only — approve or reject via mel action or the HTTP API, not by editing this list."
        />
        <button
          type="button"
          onClick={() => {
            void refresh()
            void ctx.refresh()
          }}
          className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-sm font-medium hover:bg-muted"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      {ctx.error && (
        <AlertCard variant="warning" title="Operator context unavailable" description={ctx.error} />
      )}

      {!canHandoff && !ctx.loading && (
        <AlertCard
          variant="info"
          title="Read-only incident view"
          description="Your credentials do not include incident_handoff_write. You can list incidents but cannot record handoff via the API from this session."
        />
      )}

      {openIncidents.length === 0 ? (
        <EmptyState
          type="no-data"
          title="No open incidents"
          description={
            incidents.length === 0
              ? 'There are no incidents in the recent list. When transport or system incidents are raised, they appear here with owner and handoff fields when recorded.'
              : 'All recent incidents are resolved or closed. Expand history below if needed.'
          }
        />
      ) : (
        <div className="grid gap-4">
          {openIncidents.map((inc) => (
            <IncidentCard key={inc.id} incident={inc} />
          ))}
        </div>
      )}

      {incidents.length > openIncidents.length && (
        <section className="space-y-2">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
            Other recent incidents
          </h2>
          <div className="grid gap-3">
            {incidents
              .filter((i) => !isOpenIncident(i))
              .map((inc) => (
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
    <div className="flex items-center gap-2">
      <div className="space-y-1">
        <button
          type="button"
          onClick={() => void download()}
          disabled={state === 'loading'}
          className="inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1.5 text-xs font-medium hover:bg-muted disabled:opacity-50"
          title="Download incident evidence proofpack (JSON)"
        >
          <Download className="h-3.5 w-3.5" />
          {state === 'loading' ? 'Assembling…' : 'Export proofpack'}
        </button>
        <p className="text-[11px] text-muted-foreground">
          Snapshot at request-time only; always review <code>evidence_gaps</code>.
        </p>
      </div>
      {state === 'error' && errorMsg && <span className="text-xs text-critical">{errorMsg}</span>}
    </div>
  )
}

function IncidentCard({ incident: inc, muted = false }: { incident: Incident; muted?: boolean }) {
  const pending = inc.pending_actions?.filter(Boolean) ?? []
  const hasHandoffText = !!(inc.handoff_summary && inc.handoff_summary.trim())
  const owner = inc.owner_actor_id?.trim()

  return (
    <Card
      className={clsx(
        muted && 'border-dashed opacity-90',
        'transition-shadow hover:shadow-sm'
      )}
    >
      <CardHeader className="pb-2">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div>
            <CardTitle className="text-lg font-semibold">{inc.title || inc.id}</CardTitle>
            <CardDescription className="font-mono text-xs">{inc.id}</CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            {inc.state && <Badge variant="outline">{inc.state}</Badge>}
            {inc.severity && <Badge variant="secondary">{inc.severity}</Badge>}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        {inc.summary && <p className="text-muted-foreground">{inc.summary}</p>}
        <dl className="grid gap-1 sm:grid-cols-2">
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Owner</dt>
            <dd className="font-mono text-xs">{owner || '—'}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Updated</dt>
            <dd>{formatTimestamp(inc.updated_at)}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-muted-foreground">Created</dt>
            <dd>{formatTimestamp(inc.occurred_at)}</dd>
          </div>
        </dl>
        <div>
          <div className="text-xs uppercase text-muted-foreground">Handoff summary</div>
          <div className="mt-1 rounded-md border border-border bg-muted/30 p-2 text-sm">
            {hasHandoffText ? inc.handoff_summary : 'No handoff summary recorded.'}
          </div>
        </div>
        <ProofpackDownloadButton incidentId={inc.id} />

        <div>
          <div className="mb-1 text-xs uppercase text-muted-foreground">Referenced mesh / node action IDs</div>
          {pending.length === 0 ? (
            <p className="text-muted-foreground">None recorded for this incident.</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {pending.map((id) => (
                <li
                  key={id}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border bg-background px-2 py-1.5"
                >
                  <code className="text-xs break-all">{id}</code>
                  <button
                    type="button"
                    onClick={() => copyText(id)}
                    className="inline-flex shrink-0 items-center gap-1 rounded border border-border px-2 py-0.5 text-xs hover:bg-muted"
                    title="Copy action id"
                  >
                    <ClipboardCopy className="h-3 w-3" />
                    Copy
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
        {inc.intelligence && (
          <div className="space-y-2 rounded-md border border-border bg-muted/20 p-3">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs uppercase text-muted-foreground">Incident intelligence</span>
              {inc.intelligence.signature_label && <Badge variant="outline">{inc.intelligence.signature_label}</Badge>}
              <Badge variant="secondary">evidence {inc.intelligence.evidence_strength}</Badge>
              {(inc.intelligence.signature_match_count || 0) > 1 && (
                <Badge variant="outline">seen {inc.intelligence.signature_match_count} times</Badge>
              )}
            </div>
            {inc.intelligence.similar_incidents && inc.intelligence.similar_incidents.length > 0 && (
              <p className="text-xs text-muted-foreground">
                Similar prior incidents:{' '}
                {inc.intelligence.similar_incidents
                  .map((s) => s.incident_id)
                  .filter(Boolean)
                  .join(', ')}
              </p>
            )}
            {inc.intelligence.wireless_context && (
              <div className="space-y-2 rounded border border-border/80 bg-background px-2 py-2 text-xs">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium text-foreground">Mixed wireless context</span>
                  <Badge variant="outline">{wirelessClassificationLabel(inc.intelligence.wireless_context.classification)}</Badge>
                  <Badge variant="secondary">confidence {toWords(inc.intelligence.wireless_context.confidence_posture)}</Badge>
                  <Badge variant="outline">posture {toWords(inc.intelligence.wireless_context.evidence_posture)}</Badge>
                </div>
                <p className="text-muted-foreground">{inc.intelligence.wireless_context.summary}</p>
                {(inc.intelligence.wireless_context.observed_domains || []).length > 0 && (
                  <p className="text-muted-foreground">
                    Observed domains: {(inc.intelligence.wireless_context.observed_domains || []).join(', ')}.
                  </p>
                )}
                {(inc.intelligence.wireless_context.reasons || []).length > 0 && (
                  <ul className="list-disc space-y-1 pl-4 text-muted-foreground">
                    {inc.intelligence.wireless_context.reasons?.slice(0, 2).map((reason) => (
                      <li key={reason.code}>{reason.statement}</li>
                    ))}
                  </ul>
                )}
                {(inc.intelligence.wireless_context.evidence_gaps || []).length > 0 && (
                  <p className="rounded border border-amber-300/60 bg-amber-50 px-2 py-1 text-amber-900 dark:border-amber-800/70 dark:bg-amber-950/30 dark:text-amber-100">
                    Evidence gaps: {inc.intelligence.wireless_context.evidence_gaps?.slice(0, 3).join(', ')}
                  </p>
                )}
                {(inc.intelligence.wireless_context.unsupported || []).length > 0 && (
                  <p className="rounded border border-border/80 bg-muted/30 px-2 py-1 text-foreground">
                    Unsupported scope: {inc.intelligence.wireless_context.unsupported?.map((u) => `${u.domain} ${u.scope}`).join(', ')}.
                  </p>
                )}
              </div>
            )}
            {inc.intelligence.investigate_next && inc.intelligence.investigate_next.length > 0 && (
              <ul className="list-disc space-y-1 pl-4 text-xs text-muted-foreground">
                {inc.intelligence.investigate_next.slice(0, 2).map((g) => (
                  <li key={g.id}>
                    <span className="font-medium text-foreground">{g.title}:</span> {g.rationale}
                  </li>
                ))}
              </ul>
            )}
            {inc.intelligence.action_outcome_memory && inc.intelligence.action_outcome_memory.length > 0 && (
              <div className="space-y-2">
                <div className="text-xs uppercase text-muted-foreground">Historical action-outcome memory (association only)</div>
                <p className="text-xs text-muted-foreground">
                  Historical observations from similar incidents. This does not recommend execution or establish causality.
                </p>
                <ul className="space-y-2">
                  {inc.intelligence.action_outcome_memory.map((m) => (
                    <li key={m.action_type} className="space-y-2 rounded border border-border bg-background px-2 py-2 text-xs">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium text-foreground">{m.action_label || m.action_type}</span>
                        <Badge variant="outline">occurrences {m.occurrence_count}</Badge>
                        <Badge variant="outline">sample n={m.sample_size}</Badge>
                        <Badge variant="secondary">{outcomeFramingLabel(m.outcome_framing)}</Badge>
                        {m.sample_size < 3 && <Badge variant="warning">Sparse history</Badge>}
                      </div>
                      <p className="text-muted-foreground">
                        {observedStatusLabel(m.observed_post_action_status)} • evidence strength {m.evidence_strength}
                      </p>
                      <p className="text-muted-foreground">
                        Observed outcomes: improved {m.improvement_observed_count} • deteriorated {m.deterioration_observed_count} • inconclusive {m.inconclusive_count}
                      </p>
                      {(m.caveats || []).length > 0 && (
                        <p className="rounded border border-amber-300/60 bg-amber-50 px-2 py-1 text-amber-900 dark:border-amber-800/70 dark:bg-amber-950/30 dark:text-amber-100">
                          Caveat: {(m.caveats || []).join('; ')}
                        </p>
                      )}
                      {(m.inspect_before_reuse || []).length > 0 && (
                        <p className="rounded border border-border/80 bg-muted/30 px-2 py-1 text-foreground">
                          Inspect before reuse: {m.inspect_before_reuse?.slice(0, 1).join(', ')}
                        </p>
                      )}
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {inc.intelligence.degraded && (
              <div className="space-y-1 text-xs text-amber-700">
                <p>
                  Intelligence is limited by available evidence. Treat this as investigative guidance, not causal proof.
                </p>
                {inc.intelligence.degraded_reasons && inc.intelligence.degraded_reasons.length > 0 && (
                  <ul className="list-disc pl-4">
                    {inc.intelligence.degraded_reasons.map((reason) => (
                      <li key={reason}>
                        <code>{reason}</code> — {humanizeReasonCode(reason)}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}
            {inc.intelligence.generated_at && (
              <p className="text-[11px] text-muted-foreground">
                Intelligence generated at {formatTimestamp(inc.intelligence.generated_at)}.
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
