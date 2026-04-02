import { useEffect, useMemo, useRef, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { PageHeader } from '@/components/ui/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { AlertCard } from '@/components/ui/AlertCard'
import { Badge } from '@/components/ui/Badge'
import { Loading, ErrorView } from '@/components/ui/StateViews'
import { ProgressBar } from '@/components/ui/ProgressBar'
import { formatRelativeTime, formatTimestamp, type Incident } from '@/types/api'
import { type ReplayEventClass, normalizeReplayEvents, type ReplayEventRaw } from '@/types/replay'
import { usePageHotkeys } from '@/hooks/usePageHotkeys'

type IncidentReplayResponse = {
  incident_id?: string
  generated_at?: string
  timeline?: ReplayEventRaw[]
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

const ALL_CLASSES: ReplayEventClass[] = ['state', 'action', 'transport', 'evidence', 'system', 'unknown']

function classVariant(eventClass: ReplayEventClass): 'outline' | 'secondary' | 'warning' {
  if (eventClass === 'transport' || eventClass === 'evidence') return 'warning'
  if (eventClass === 'unknown') return 'secondary'
  return 'outline'
}

export function IncidentDetail() {
  const { id } = useParams<{ id: string }>()
  const [incident, setIncident] = useState<Incident | null>(null)
  const [replay, setReplay] = useState<IncidentReplayResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [eventQuery, setEventQuery] = useState('')
  const [classFilter, setClassFilter] = useState<ReplayEventClass | 'all'>('all')
  const [originFilter, setOriginFilter] = useState<'all' | 'observed' | 'derived' | 'imported' | 'unknown'>('all')
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())
  const sectionRefs = useRef<Record<string, HTMLElement | null>>({})
  const replaySearchRef = useRef<HTMLInputElement>(null)

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
  const normalizedReplay = useMemo(() => normalizeReplayEvents(replay?.timeline ?? []), [replay?.timeline])

  const filteredReplay = useMemo(() => {
    return normalizedReplay.filter((event) => {
      if (classFilter !== 'all' && event.eventClass !== classFilter) return false
      if (originFilter !== 'all' && event.origin !== originFilter) return false
      if (eventQuery.trim().length > 0) {
        const q = eventQuery.toLowerCase()
        const haystack = `${event.summary} ${event.eventType} ${event.basis || ''} ${event.source || ''}`.toLowerCase()
        if (!haystack.includes(q)) return false
      }
      return true
    })
  }, [normalizedReplay, classFilter, originFilter, eventQuery])

  usePageHotkeys([
    {
      key: '/',
      description: 'Focus replay filter',
      handler: () => replaySearchRef.current?.focus(),
    },
    {
      key: '1',
      description: 'Jump to incident truth state',
      handler: () => sectionRefs.current.truth?.scrollIntoView({ behavior: 'smooth', block: 'start' }),
    },
    {
      key: '2',
      description: 'Jump to replay timeline',
      handler: () => sectionRefs.current.replay?.scrollIntoView({ behavior: 'smooth', block: 'start' }),
    },
    {
      key: '3',
      description: 'Jump to replay caveats',
      handler: () => sectionRefs.current.caveats?.scrollIntoView({ behavior: 'smooth', block: 'start' }),
    },
    {
      key: 'o',
      description: 'Expand replay details',
      handler: () => setExpandedIds(new Set(filteredReplay.map((e) => e.id))),
    },
    {
      key: 'c',
      description: 'Collapse replay details',
      handler: () => setExpandedIds(new Set()),
    },
  ])

  if (loading) return <Loading message="Loading incident review…" />
  if (error || !incident) return <ErrorView message={error || 'Incident unavailable'} />

  return (
    <div className="space-y-4 md:space-y-5">
      <PageHeader
        title={incident.title || `Incident ${incident.id}`}
        description="Bookmarkable incident review surface. Evidence is shown as observed records plus bounded derived guidance."
        action={
          <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:justify-end">
            <Link to="/incidents" className="button-secondary">Back to incidents</Link>
            <a className="button-secondary" href={`/api/v1/incidents/${encodeURIComponent(incident.id)}/proofpack?download=true`}>
              Export proofpack
            </a>
          </div>
        }
      />

      <div className="grid gap-4 lg:grid-cols-3" ref={(el) => (sectionRefs.current.truth = el)}>
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

      <div ref={(el) => (sectionRefs.current.replay = el)}><Card>
        <CardHeader>
          <CardTitle>Replay timeline</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid gap-2 sm:grid-cols-3">
            <input
              ref={replaySearchRef}
              value={eventQuery}
              onChange={(e) => setEventQuery(e.target.value)}
              placeholder="Filter replay events…"
              className="rounded border border-border bg-background px-2 py-1.5 text-xs"
            />
            <select
              className="rounded border border-border bg-background px-2 py-1.5 text-xs"
              value={classFilter}
              onChange={(e) => setClassFilter(e.target.value as ReplayEventClass | 'all')}
            >
              <option value="all">All classes</option>
              {ALL_CLASSES.map((cls) => (
                <option key={cls} value={cls}>{cls}</option>
              ))}
            </select>
            <select
              className="rounded border border-border bg-background px-2 py-1.5 text-xs"
              value={originFilter}
              onChange={(e) => setOriginFilter(e.target.value as 'all' | 'observed' | 'derived' | 'imported' | 'unknown')}
            >
              <option value="all">All origins</option>
              <option value="observed">Observed</option>
              <option value="derived">Derived</option>
              <option value="imported">Imported</option>
              <option value="unknown">Unknown</option>
            </select>
          </div>
          <p className="text-[11px] text-muted-foreground">
            Event classes are frontend semantics only. They improve scan speed but do not upgrade backend evidence certainty.
          </p>

          {normalizedReplay.length === 0 ? (
            <AlertCard
              variant="warning"
              title="Replay data unavailable"
              description="No timeline was returned for this incident. This may indicate sparse history, capability limits, or temporary backend unavailability."
            />
          ) : filteredReplay.length === 0 ? (
            <p className="text-xs text-muted-foreground">No replay events match the current filters.</p>
          ) : (
            <div className="space-y-2">
              {filteredReplay.map((event) => {
                const expanded = expandedIds.has(event.id)
                return (
                  <button
                    type="button"
                    key={event.id}
                    className="w-full rounded-lg border p-3 text-left text-xs"
                    onClick={() => {
                      setExpandedIds((prev) => {
                        const next = new Set(prev)
                        if (next.has(event.id)) {
                          next.delete(event.id)
                        } else {
                          next.add(event.id)
                        }
                        return next
                      })
                    }}
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant={classVariant(event.eventClass)}>{event.eventClass}</Badge>
                      <Badge variant="outline">{event.eventType}</Badge>
                      <Badge variant={event.origin === 'observed' ? 'outline' : 'secondary'}>{event.origin}</Badge>
                      <Badge variant={event.confidenceLabel === 'low' ? 'warning' : 'secondary'}>confidence: {event.confidenceLabel}</Badge>
                      <span className="text-muted-foreground">{event.timestamp || 'time unknown'}</span>
                      {event.stale && <Badge variant="warning">stale segment</Badge>}
                    </div>
                    <p className="mt-1 text-muted-foreground">{event.summary}</p>
                    {expanded && (
                      <div className="mt-2 grid gap-1 text-[11px] text-muted-foreground sm:grid-cols-2">
                        <div>Source: {event.source || 'unknown'}</div>
                        <div>Basis: {event.basis || 'not provided'}</div>
                        <div>Actor: {event.actorId || 'unknown'}</div>
                        {event.confidenceValue != null && <div>Confidence score: {event.confidenceValue.toFixed(2)}</div>}
                      </div>
                    )}
                  </button>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card></div>

      <div ref={(el) => (sectionRefs.current.caveats = el)}><Card>
        <CardHeader>
          <CardTitle>Replay caveats and gaps</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-xs">
          {(replay?.evidence_gaps?.length ?? 0) > 0 && (
            <div className="rounded-lg border border-warning/30 bg-warning/5 p-3">
              <p className="font-medium">Evidence gaps</p>
              <ul className="mt-1 list-disc pl-4 text-muted-foreground">
                {replay!.evidence_gaps!.slice(0, 6).map((gap) => (
                  <li key={gap}>{gap}</li>
                ))}
              </ul>
            </div>
          )}
          {(replay?.known_limitations?.length ?? 0) > 0 && (
            <div>
              <p className="font-medium">Known replay limitations</p>
              <ul className="mt-1 list-disc pl-4 text-muted-foreground">
                {replay!.known_limitations!.slice(0, 6).map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          )}
          {(replay?.unsupported?.length ?? 0) > 0 && (
            <div>
              <p className="font-medium">Unsupported in this replay payload</p>
              <ul className="mt-1 list-disc pl-4 text-muted-foreground">
                {replay!.unsupported!.slice(0, 6).map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          )}
          {replay?.degraded && (
            <AlertCard
              variant="warning"
              title="Replay degraded"
              description={(replay.degraded_reasons ?? []).join(' · ') || 'Replay degraded with no reason text from API.'}
            />
          )}
        </CardContent>
      </Card></div>
    </div>
  )
}
