import { useCallback } from 'react'
import { Link } from 'react-router-dom'
import { ClipboardList, Download } from 'lucide-react'
import { PageHeader } from '@/components/ui/PageHeader'
import { OperatorTruthRibbon } from '@/components/ui/OperatorTruthRibbon'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Loading } from '@/components/ui/StateViews'
import { InlineAlert } from '@/components/ui/AlertCard'
import { useOperatorBriefing, useOperatorDigest } from '@/hooks/useApi'
import { formatRelativeTime } from '@/types/api'

export function OperationalReview() {
  const briefing = useOperatorBriefing()
  const digest = useOperatorDigest()

  const downloadDigest = useCallback(() => {
    if (!digest.data) return
    const blob = new Blob([JSON.stringify(digest.data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `mel-operational-digest-${digest.data.generated_at.replace(/[:]/g, '-')}.json`
    a.click()
    URL.revokeObjectURL(url)
  }, [digest.data])

  if (briefing.loading && !briefing.data && digest.loading && !digest.data) {
    return <Loading message="Loading operational review…" />
  }

  const top = (briefing.data?.top_priorities ?? []).slice(0, 8)

  return (
    <div className="space-y-5 pb-10">
      <PageHeader
        title="Operational review"
        subtitle="Evidence-backed snapshot"
        description="Deterministic counts from this MEL instance database plus a ranked issue briefing from diagnostics and incidents. Not fleet-wide truth and not proof of live RF or routing."
      />
      <OperatorTruthRibbon summary="Digest counts are SQL aggregates on this gateway’s SQLite store. Briefing priorities use bounded heuristics over the same persisted evidence — treat both as operator aids, not autonomous diagnosis." />

      {(briefing.error || digest.error) && (
        <InlineAlert variant="warning">
          <span className="font-semibold text-foreground">Partial data.</span>{' '}
          {[briefing.error, digest.error].filter(Boolean).join(' ')}
        </InlineAlert>
      )}

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader className="flex flex-row flex-wrap items-start justify-between gap-2">
            <div>
              <CardTitle className="text-base">Instance digest</CardTitle>
              <CardDescription>
                Schema {digest.data?.schema_version ?? '—'} · window {digest.data?.window_hours ?? 24}h activity
              </CardDescription>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {digest.lastUpdated && (
                <span className="text-mel-xs text-muted-foreground">
                  {formatRelativeTime(digest.lastUpdated.toISOString())}
                </span>
              )}
              <button
                type="button"
                onClick={() => void digest.refresh()}
                className="rounded-sm border border-border/70 px-2 py-1 text-mel-sm font-semibold hover:bg-muted/50"
              >
                Refresh
              </button>
              <button
                type="button"
                disabled={!digest.data}
                onClick={downloadDigest}
                className="inline-flex items-center gap-1 rounded-sm border border-border/70 px-2 py-1 text-mel-sm font-semibold hover:bg-muted/50 disabled:opacity-50"
              >
                <Download className="h-3 w-3" aria-hidden />
                JSON
              </button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4 text-sm">
            {digest.data?.instance_id && (
              <p className="text-xs text-muted-foreground font-mono break-all">{digest.data.instance_id}</p>
            )}
            {digest.data && (
              <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-xs sm:grid-cols-3">
                <div>
                  <dt className="text-muted-foreground">Open incidents</dt>
                  <dd className="font-semibold tabular-nums">{digest.data.counts.open_incidents}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Critical / high open</dt>
                  <dd className="font-semibold tabular-nums">
                    {digest.data.counts.critical_open_incidents} / {digest.data.counts.high_open_incidents}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Resolved (7d)</dt>
                  <dd className="font-semibold tabular-nums">{digest.data.counts.resolved_last_7_days}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Control actions (total)</dt>
                  <dd className="font-semibold tabular-nums">{digest.data.counts.control_actions_total}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Pending approval</dt>
                  <dd className="font-semibold tabular-nums">{digest.data.counts.pending_approval_actions}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Approved, awaiting executor</dt>
                  <dd className="font-semibold tabular-nums">{digest.data.counts.awaiting_executor_actions}</dd>
                </div>
                <div className="col-span-2 sm:col-span-3 border-t border-border/40 pt-2 mt-1">
                  <p className="text-mel-xs font-semibold uppercase tracking-[0.12em] text-muted-foreground mb-1">
                    Last {digest.data.window_hours}h (primary timestamps)
                  </p>
                  <div className="grid grid-cols-3 gap-2">
                    <div>
                      <dt className="text-muted-foreground">Incidents opened</dt>
                      <dd className="font-semibold tabular-nums">{digest.data.window_counts.incidents_opened}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Actions created</dt>
                      <dd className="font-semibold tabular-nums">{digest.data.window_counts.control_actions_created}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Operator notes</dt>
                      <dd className="font-semibold tabular-nums">{digest.data.window_counts.operator_notes_created}</dd>
                    </div>
                  </div>
                </div>
              </dl>
            )}
            {digest.data?.truth_notes && digest.data.truth_notes.length > 0 && (
              <ul className="text-mel-sm text-muted-foreground list-disc list-inside space-y-1">
                {digest.data.truth_notes.map((n, i) => (
                  <li key={i}>{n}</li>
                ))}
              </ul>
            )}
            <div className="flex flex-wrap gap-3 pt-1 text-xs font-semibold">
              <Link to="/incidents" className="text-primary hover:underline">
                Incidents
              </Link>
              <Link to="/control-actions" className="text-primary hover:underline">
                Control actions
              </Link>
              <Link to="/events" className="text-primary hover:underline">
                Events / audit
              </Link>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row flex-wrap items-start justify-between gap-2">
            <div>
              <CardTitle className="text-base">Intelligence briefing</CardTitle>
              <CardDescription>Ranked operational issues from persisted state</CardDescription>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {briefing.data && (
                <Badge
                  variant={
                    briefing.data.overall_status === 'Critical'
                      ? 'critical'
                      : briefing.data.overall_status === 'Degraded'
                        ? 'warning'
                        : 'secondary'
                  }
                >
                  {briefing.data.overall_status}
                </Badge>
              )}
              {briefing.lastUpdated && (
                <span className="text-mel-xs text-muted-foreground">
                  {formatRelativeTime(briefing.lastUpdated.toISOString())}
                </span>
              )}
              <button
                type="button"
                onClick={() => void briefing.refresh()}
                className="rounded-sm border border-border/70 px-2 py-1 text-mel-sm font-semibold hover:bg-muted/50"
              >
                Refresh
              </button>
            </div>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            {briefing.data?.blast_radius_estimate && (
              <p className="text-xs text-muted-foreground">{briefing.data.blast_radius_estimate}</p>
            )}
            {top.length === 0 ? (
              <p className="text-xs text-muted-foreground flex items-center gap-2">
                <ClipboardList className="h-4 w-4 shrink-0" aria-hidden />
                No ranked priorities in this response — calm posture or sparse diagnostics.
              </p>
            ) : (
              <ol className="space-y-2 list-decimal list-inside text-sm">
                {top.map((p) => (
                  <li key={p.id} className="marker:text-mel-sm marker:text-muted-foreground">
                    <span className="font-medium text-foreground">{p.title}</span>
                    <span className="text-muted-foreground text-xs block sm:inline sm:ml-1"> — {p.summary}</span>
                    <span className="text-mel-xs text-muted-foreground block">
                      {p.severity} · evidence {p.evidence_freshness}
                      {p.is_actionable ? ' · actionable' : ''}
                    </span>
                  </li>
                ))}
              </ol>
            )}
            {(briefing.data?.uncertainty_notes?.length ?? 0) > 0 && (
              <ul className="text-mel-sm text-muted-foreground list-disc list-inside space-y-1 border-t border-border/40 pt-2">
                {briefing.data!.uncertainty_notes!.map((n, i) => (
                  <li key={i}>{n}</li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
