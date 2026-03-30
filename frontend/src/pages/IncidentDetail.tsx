import { useEffect, useMemo, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { AlertCard } from '@/components/ui/AlertCard'
import { Badge } from '@/components/ui/Badge'
import { Loading, ErrorView } from '@/components/ui/StateViews'
import { ProgressBar } from '@/components/ui/ProgressBar'
import { formatRelativeTime, formatTimestamp, type Incident } from '@/types/api'

type ReplayEvent = {
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
}

type IncidentReplayResponse = {
  incident_id?: string
  generated_at?: string
  timeline?: ReplayEvent[]
  evidence_gaps?: string[]
  unsupported?: string[]
  degraded?: boolean
  degraded_reasons?: string[]
  known_limitations?: string[]
}

function computeProofpackCompleteness(incident: Incident): { percent: number; state: string; gaps: string[] } {
  const gaps = incident.intelligence?.wireless_context?.evidence_gaps ?? []
  const completeness = incident.intelligence?.action_outcome_trace?.completeness
  const hasSparse = incident.intelligence?.evidence_strength === 'sparse'

  if (completeness === 'complete' && gaps.length === 0 && !hasSparse) {
    return { percent: 100, state: 'review_ready', gaps: [] }
  }
  if (completeness === 'partial' || gaps.length > 0 || hasSparse) {
    return { percent: 55, state: 'partial', gaps }
  }
  return { percent: 25, state: 'unknown', gaps }
}

export function IncidentDetail() {
  const { id } = useParams<{ id: string }>()
  const [incident, setIncident] = useState<Incident | null>(null)
  const [replay, setReplay] = useState<IncidentReplayResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    let cancelled = false
    ;(async () => {
      setLoading(true)
      try {
        const [incRes, replayRes] = await Promise.all([
          fetch(`/api/v1/incidents/${encodeURIComponent(id)}`),
          fetch(`/api/v1/incidents/${encodeURIComponent(id)}/replay`),
        ])
        if (!incRes.ok) throw new Error(`Incident HTTP ${incRes.status}`)
        const incData = (await incRes.json()) as Incident
        const replayData = replayRes.ok ? ((await replayRes.json()) as IncidentReplayResponse) : null
        if (!cancelled) {
          setIncident(incData)
          setReplay(replayData)
          setError(null)
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load incident')
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [id])

  const proofpack = useMemo(() => (incident ? computeProofpackCompleteness(incident) : null), [incident])
  const replayTimeline = replay?.timeline ?? []

  if (loading) return <Loading message="Loading incident review…" />
  if (error || !incident) return <ErrorView message={error || 'Incident unavailable'} />

  return (
    <div className="space-y-5">
      <PageHeader
        title={incident.title || `Incident ${incident.id}`}
        description="Bookmarkable incident review surface. Evidence is presented as observed records plus bounded derived guidance."
        action={
          <div className="flex items-center gap-2">
            <Link to="/incidents" className="button-secondary">Back to incidents</Link>
            <a className="button-secondary" href={`/api/v1/incidents/${encodeURIComponent(incident.id)}/proofpack?download=true`}>
              Export proofpack
            </a>
          </div>
        }
      />

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Incident truth state</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex flex-wrap gap-2">
              <Badge variant="outline">ID {incident.id}</Badge>
              {incident.state && <Badge variant="secondary">{incident.state}</Badge>}
              {incident.severity && <Badge variant="warning">{incident.severity}</Badge>}
              {incident.intelligence?.evidence_strength && (
                <Badge variant="outline">{incident.intelligence.evidence_strength} evidence</Badge>
              )}
            </div>
            {incident.summary && <p className="text-muted-foreground">{incident.summary}</p>}
            <div className="grid gap-2 sm:grid-cols-2 text-xs text-muted-foreground">
              <div>Occurred: {incident.occurred_at ? formatTimestamp(incident.occurred_at) : 'Unknown'}</div>
              <div>Updated: {incident.updated_at ? formatRelativeTime(incident.updated_at) : 'Unknown'}</div>
              <div>Owner: {incident.owner_actor_id || 'Unassigned'}</div>
              <div>Review state: {incident.review_state || 'Not set'}</div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Proofpack completeness</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {proofpack && (
              <>
                <ProgressBar value={proofpack.percent} max={100} />
                <p className="text-xs font-medium">{proofpack.state.replace(/_/g, ' ')}</p>
                <p className="text-xs text-muted-foreground">
                  {proofpack.state === 'review_ready'
                    ? 'Complete enough for internal review. Export still reflects redaction policy and request-time snapshot.'
                    : proofpack.state === 'partial'
                      ? 'Partial evidence. Treat conclusions as bounded and review evidence gaps before export/share.'
                      : 'Completeness unknown from current payload.'}
                </p>
                {proofpack.gaps.length > 0 && (
                  <ul className="list-disc pl-4 text-xs text-muted-foreground space-y-1">
                    {proofpack.gaps.slice(0, 4).map((gap) => (
                      <li key={gap}>{gap}</li>
                    ))}
                  </ul>
                )}
              </>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Replay timeline</CardTitle>
        </CardHeader>
        <CardContent>
          {replayTimeline.length === 0 ? (
            <AlertCard
              variant="warning"
              title="Replay data unavailable"
              description="No timeline was returned for this incident. This may indicate sparse history, capability limits, or temporary backend unavailability."
            />
          ) : (
            <div className="space-y-2">
              {replayTimeline.map((event, idx) => (
                <div key={`${event.occurred_at || event.observed_at || idx}-${idx}`} className="rounded-lg border p-3 text-xs">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant="outline">{event.event_type || event.kind || 'event'}</Badge>
                    <span className="text-muted-foreground">{event.occurred_at || event.observed_at || 'time unknown'}</span>
                    {event.basis && <Badge variant="secondary">basis: {event.basis}</Badge>}
                    {event.stale && <Badge variant="warning">stale segment</Badge>}
                  </div>
                  <p className="mt-1 text-muted-foreground">{event.summary || event.statement || 'No summary provided.'}</p>
                </div>
              ))}
            </div>
          )}

          {(replay?.evidence_gaps?.length ?? 0) > 0 && (
            <div className="mt-3 rounded-lg border border-warning/30 bg-warning/5 p-3 text-xs">
              <p className="font-medium">Evidence gaps</p>
              <ul className="mt-1 list-disc pl-4 text-muted-foreground">
                {replay!.evidence_gaps!.slice(0, 5).map((gap) => (
                  <li key={gap}>{gap}</li>
                ))}
              </ul>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
