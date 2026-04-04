import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Activity,
  Radio,
  MessageSquare,
  Shield,
  AlertTriangle,
  ArrowRight,
  CheckCircle2,
  XCircle,
  TrendingUp,
  AlertCircle,
  Zap,
  Inbox,
  FileText,
  Compass,
  ClipboardList,
  Clock,
  HelpCircle,
} from 'lucide-react'
import { clsx } from 'clsx'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { Badge, HealthBadge, SeverityBadge } from '@/components/ui/Badge'
import { AlertCard, InlineAlert } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { PageHeader } from '@/components/ui/PageHeader'
import { OperatorTruthRibbon } from '@/components/ui/OperatorTruthRibbon'
import { StaleDataBanner } from '@/components/states/StaleDataBanner'
import { NoTransportsConfigured } from '@/components/ui/EmptyState'
import { ActivityFeed, eventsToFeedItems, type FeedItem } from '@/components/ui/ActivityFeed'
import { useStatus, useNodes, useMessages, usePrivacyFindings, useRecommendations, useDeadLetters, useEvents, useDiagnostics, useOperationalState } from '@/hooks/useApi'
import { useIncidents } from '@/hooks/useIncidents'
import { getHealthState, formatRelativeTime, TransportHealth, NodeInfo } from '@/types/api'
import type { ShiftSnapshot } from '@/utils/shiftSnapshot'
import {
  buildShiftSnapshotFromConsole,
  computeShiftDelta,
  readShiftSnapshot,
  writeShiftSnapshot,
} from '@/utils/shiftSnapshot'
import {
  buildRecurrenceHomeTeasers,
  buildShiftStartAttentionRows,
  countOpenIncidentsExplicitFollowUp,
  sortOpenIncidentsForShiftStart,
} from '@/utils/operatorWorkflow'
import { useVersionInfo } from '@/hooks/useVersionInfo'
import { useOperatorContext } from '@/hooks/useOperatorContext'
import type { IncidentWorkQueueWhyContext } from '@/utils/operatorWorkflow'
import { operatorCanReadLinkedControlRows } from '@/utils/incidentOperatorTruth'
import { operatorExportReadinessFromVersion } from '@/utils/operatorExportReadiness'
import { FirstRunHintBanner } from '@/components/onboarding/FirstRunHintBanner'

export function Dashboard() {
  const status = useStatus()
  const nodes = useNodes()
  const { data: messagesData, loading: messagesLoading, error: messagesError } = useMessages()
  const privacy = usePrivacyFindings()
  const recommendations = useRecommendations()
  const deadLetters = useDeadLetters()
  const events = useEvents()
  const diagnostics = useDiagnostics()
  const incidents = useIncidents()
  const operational = useOperationalState()
  const versionInfo = useVersionInfo()
  const operatorCtx = useOperatorContext()

  const [refreshCount, setRefreshCount] = useState(0)
  const prevAttemptRef = useRef<Date | null>(null)
  const [shiftBaseline, setShiftBaseline] = useState<ShiftSnapshot | null>(() => readShiftSnapshot())
  const [topologyNodesForShift, setTopologyNodesForShift] = useState<
    Array<{ node_num: number; last_seen_at?: string; short_name?: string; long_name?: string }>
  >([])

  const isLoading = status.loading || nodes.loading || messagesLoading
  const hasError = status.error || nodes.error || messagesError

  const transports = useMemo(() => status.data?.transports ?? [], [status.data?.transports])

  const sparseIncidents = useMemo(() => {
    const list = incidents.data ?? []
    return list.filter((i) => {
      const s = (i.state || '').toLowerCase()
      if (s === 'resolved' || s === 'closed') return false
      return (
        i.intelligence?.evidence_strength === 'sparse' ||
        (i.intelligence?.degraded === true && (i.intelligence?.sparsity_markers?.length ?? 0) > 0)
      )
    })
  }, [incidents.data])

  const recurringOpenIncidents = useMemo(() => {
    return (incidents.data ?? []).filter((i) => {
      const s = (i.state || '').toLowerCase()
      if (s === 'resolved' || s === 'closed') return false
      return (i.intelligence?.signature_match_count ?? 0) > 1
    })
  }, [incidents.data])

  const shiftDelta = useMemo(() => {
    const incList = incidents.data ?? []
    const nodeList = nodes.data ?? []
    const ev = events.data ?? []
    const msgCount =
      typeof status.data?.messages === 'number'
        ? status.data.messages
        : messagesData?.length ?? 0
    return computeShiftDelta(shiftBaseline, {
      incidents: incList,
      nodes: nodeList,
      topologyNodes: topologyNodesForShift,
      transports,
      events: ev,
      messageCount: msgCount,
      deadLetterCount: deadLetters.data?.length ?? 0,
    })
  }, [
    shiftBaseline,
    incidents.data,
    nodes.data,
    topologyNodesForShift,
    transports,
    events.data,
    status.data?.messages,
    messagesData?.length,
    deadLetters.data?.length,
  ])

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const res = await fetch('/api/v1/topology/nodes?limit=500')
        if (!res.ok) return
        const j = (await res.json()) as {
          nodes?: Array<{ node_num: number; last_seen_at?: string; short_name?: string; long_name?: string }>
        }
        if (!cancelled) setTopologyNodesForShift(j.nodes ?? [])
      } catch {
        if (!cancelled) setTopologyNodesForShift([])
      }
    })()
    return () => {
      cancelled = true
    }
  }, [refreshCount])

  useEffect(() => {
    const t = status.lastUpdated
    if (!t) return
    if (prevAttemptRef.current && t.getTime() === prevAttemptRef.current.getTime()) return
    prevAttemptRef.current = t
    setRefreshCount((c) => c + 1)
  }, [status.lastUpdated])

  const openIncidents = useMemo(
    () =>
      (incidents.data ?? []).filter((inc) => {
        const s = (inc.state || '').toLowerCase()
        return s !== 'resolved' && s !== 'closed'
      }),
    [incidents.data],
  )

  const transportStats = useMemo(() => {
    const connectedTransport = transports.find((t) => t.effective_state === 'connected')
    const healthyTransports = transports.filter((t) => getHealthState(t.health) === 'healthy').length
    const totalTransports = transports.length
    const hasTransports = totalTransports > 0
    const degradedTransports = transports.filter((t) => getHealthState(t.health) === 'degraded').length
    const unhealthyTransports = transports.filter((t) => getHealthState(t.health) === 'unhealthy').length
    return {
      connectedTransport,
      healthyTransports,
      totalTransports,
      hasTransports,
      degradedTransports,
      unhealthyTransports,
    }
  }, [transports])

  const activePrivacyFindings = useMemo(
    () => privacy.data?.filter((p) => p.severity === 'critical' || p.severity === 'high') || [],
    [privacy.data],
  )
  const pendingRecommendations = useMemo(
    () => recommendations.data?.filter((r) => r.actionable) || [],
    [recommendations.data],
  )
  const criticalDiags = useMemo(
    () => diagnostics.data?.filter((d) => d.severity === 'critical' || d.severity === 'high') || [],
    [diagnostics.data],
  )
  const deadLetterCount = deadLetters.data?.length ?? 0

  const pendingApprovals = operational.data?.pending_approvals ?? []

  const exportPosture = versionInfo.data?.platform_posture?.evidence_export_delete
  const exportReadiness = useMemo(
    () => operatorExportReadinessFromVersion(versionInfo.data, versionInfo.error ?? null),
    [versionInfo.data, versionInfo.error],
  )

  const incidentQueueCtx: IncidentWorkQueueWhyContext = useMemo(
    () => ({
      exportEnabled: exportPosture?.export_enabled,
      exportPolicyUnknown: !versionInfo.loading && versionInfo.error != null && exportPosture == null,
      canReadLinkedActions: operatorCanReadLinkedControlRows({
        loading: operatorCtx.loading,
        error: operatorCtx.error,
        trustUI: operatorCtx.trustUI,
        capabilities: operatorCtx.capabilities ?? [],
      }),
    }),
    [exportPosture, versionInfo.loading, versionInfo.error, operatorCtx.trustUI, operatorCtx.capabilities, operatorCtx.loading, operatorCtx.error],
  )

  const openIncidentsShiftOrder = useMemo(
    () => sortOpenIncidentsForShiftStart(openIncidents, incidentQueueCtx),
    [openIncidents, incidentQueueCtx],
  )

  const shiftAttentionRows = useMemo(
    () =>
      buildShiftStartAttentionRows({
        openIncidents: openIncidentsShiftOrder,
        unhealthyTransportCount: transportStats.unhealthyTransports,
        degradedTransportCount: transportStats.degradedTransports,
        criticalDiagCount: criticalDiags.length,
        criticalPrivacyCount: activePrivacyFindings.length,
        pendingApprovalCount: pendingApprovals.length,
        deadLetterCount,
        deadLettersIncreasedSinceBaseline: shiftDelta.deadLettersIncreased,
        sparseOpenCount: sparseIncidents.length,
        incidentWhyContext: incidentQueueCtx,
      }),
    [
      openIncidentsShiftOrder,
      transportStats.unhealthyTransports,
      transportStats.degradedTransports,
      criticalDiags.length,
      activePrivacyFindings.length,
      pendingApprovals.length,
      deadLetterCount,
      shiftDelta.deadLettersIncreased,
      sparseIncidents.length,
      incidentQueueCtx,
    ],
  )

  const retentionDays =
    versionInfo.data?.platform_posture?.retention?.audit_days ??
    versionInfo.data?.platform_posture?.retention_default_days

  const explicitFollowUpOpenCount = useMemo(
    () => countOpenIncidentsExplicitFollowUp(openIncidents),
    [openIncidents],
  )

  const recurrenceTeasers = useMemo(
    () => buildRecurrenceHomeTeasers(incidents.data ?? [], 5),
    [incidents.data],
  )

  if (isLoading && !status.data) {
    return <Loading message="Loading system status..." />
  }

  if (hasError && !status.data) {
    return (
      <div className="p-8 animate-fade-in">
        <AlertCard
          variant="critical"
          title="Unable to connect to MEL backend"
          description={status.error || 'Failed to connect to MEL backend. Please ensure MEL is running.'}
          action={
            <button
              onClick={() => window.location.reload()}
              className="button-danger"
            >
              Retry connection
            </button>
          }
        />
      </div>
    )
  }

  const {
    connectedTransport,
    healthyTransports,
    totalTransports,
    hasTransports,
    degradedTransports,
    unhealthyTransports,
  } = transportStats

  function markShiftBaseline() {
    const incList = incidents.data ?? []
    const nodeList = nodes.data ?? []
    const ev = events.data ?? []
    const msgCount =
      typeof status.data?.messages === 'number'
        ? status.data.messages
        : messagesData?.length ?? 0
    const snap = buildShiftSnapshotFromConsole({
      incidents: incList,
      nodes: nodeList,
      topologyNodes: topologyNodesForShift,
      transports,
      events: ev,
      messageCount: msgCount,
      deadLetterCount: deadLetters.data?.length ?? 0,
    })
    writeShiftSnapshot(snap)
    setShiftBaseline(snap)
  }

  const newestHeartbeat = transports.reduce((max, t) => {
    if (!t.last_heartbeat_at) return max
    const ts = new Date(t.last_heartbeat_at).getTime()
    return ts > max ? ts : max
  }, 0)
  const dashboardStaleTs = newestHeartbeat ? new Date(newestHeartbeat).toISOString() : undefined

  // Build unified feed items
  const feedItems: FeedItem[] = [
    ...eventsToFeedItems(events.data ?? []),
    ...(openIncidents.map((inc) => ({
      id: `inc-${inc.id}`,
      type: 'incident' as const,
      level: (inc.severity === 'critical' ? 'critical' : inc.severity === 'high' ? 'warning' : 'info') as FeedItem['level'],
      title: inc.title || `Incident ${inc.id.slice(0, 8)}`,
      detail: inc.summary,
      timestamp: inc.occurred_at ?? inc.updated_at ?? '',
      href: `/incidents/${encodeURIComponent(inc.id)}`,
      category: inc.category,
    }))),
    ...(deadLetters.data ?? []).slice(0, 5).map((dl, i) => ({
      id: `dl-${i}-${dl.created_at}`,
      type: 'dead_letter' as const,
      level: 'warning' as const,
      title: `Dead letter: ${dl.transport_name}`,
      detail: dl.reason,
      timestamp: dl.created_at,
      href: '/dead-letters',
    })),
  ]

  // Count attention items for the system pulse
  const attentionCount =
    openIncidents.length +
    activePrivacyFindings.length +
    criticalDiags.length +
    (unhealthyTransports > 0 ? 1 : 0) +
    pendingApprovals.length

  const hasShiftBaseline = shiftBaseline !== null
  const deltaSummaryParts: string[] = []
  if (shiftDelta.openIncidentsChangedSince.length > 0) {
    deltaSummaryParts.push(
      `${shiftDelta.openIncidentsChangedSince.length} open incident(s) updated since baseline`,
    )
  } else if (shiftDelta.incidentsTouchedSince.length > 0) {
    deltaSummaryParts.push(`${shiftDelta.incidentsTouchedSince.length} incident touch(es) (incl. resolved/closed)`)
  }
  if (shiftDelta.stillOpenSinceBaseline.length > 0) {
    deltaSummaryParts.push(`${shiftDelta.stillOpenSinceBaseline.length} still open from baseline list`)
  }
  if (shiftDelta.noLongerOpenSinceBaseline.length > 0) {
    deltaSummaryParts.push(
      `${shiftDelta.noLongerOpenSinceBaseline.length} incident(s) no longer open since baseline (state changed — verify in list)`,
    )
  }
  if (shiftDelta.nodesWithNewerLastSeen.length > 0) {
    deltaSummaryParts.push(`${shiftDelta.nodesWithNewerLastSeen.length} node(s) with newer last_seen`)
  }
  if (shiftDelta.topologyNodesWithNewerLastSeen.length > 0) {
    deltaSummaryParts.push(
      `${shiftDelta.topologyNodesWithNewerLastSeen.length} topology node(s) with newer last_seen (graph store)`,
    )
  }
  if (shiftDelta.newAuditEvents > 0) {
    deltaSummaryParts.push(`${shiftDelta.newAuditEvents} new audit event(s)`)
  }
  if (shiftDelta.transportHeartbeatAdvanced) deltaSummaryParts.push('transport heartbeat advanced')
  if (shiftDelta.messagesIncreased) deltaSummaryParts.push('message counter increased')
  if (shiftDelta.deadLettersIncreased) deltaSummaryParts.push('dead letter count increased')

  return (
    <div className="space-y-5 pb-10 md:space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <PageHeader
          title="Command surface"
          subtitle="Mesh operations cockpit"
          description="Shift-start overview: attention, evidence posture, and where to go next. Data refreshes on a short poll while this tab is visible."
        />
        <div className="flex flex-col items-end gap-1.5">
          <div className="flex items-center gap-2">
            <Badge variant="outline" className="uppercase tracking-[0.18em]">Near-live poll</Badge>
            {status.lastUpdated && (
              <span className="text-[11px] text-muted-foreground/60">
                {formatRelativeTime(status.lastUpdated.toISOString())}
              </span>
            )}
          </div>
          <span className="text-[10px] text-muted-foreground/70 max-w-[280px] text-right">
            Refreshed {refreshCount} time{refreshCount === 1 ? '' : 's'} this session (browser tab).
          </span>
        </div>
      </div>

      <OperatorTruthRibbon summary="This surface summarizes persisted ingest, incidents, and audit signals. It does not prove RF coverage, routing success, or live paths beyond what the API exposes." />

      <FirstRunHintBanner visible={!hasTransports} />

      {/* Shift baseline — local browser only */}
      <div className="rounded-2xl border border-border/60 bg-card/40 p-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex gap-3 min-w-0">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-border/60 bg-muted/30 text-muted-foreground">
              <ClipboardList className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <p className="text-sm font-semibold text-foreground">Since last visit (this browser)</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                {hasShiftBaseline && shiftBaseline
                  ? `Baseline saved ${formatRelativeTime(shiftBaseline.savedAt)}. Comparison is local to this profile — not shared across operators or devices.`
                  : 'No baseline yet. Mark baseline after you have reviewed the console so “what changed” has a truthful anchor.'}
              </p>
              {hasShiftBaseline && deltaSummaryParts.length > 0 && (
                <ul className="mt-2 text-xs text-foreground space-y-1 list-disc list-inside">
                  {deltaSummaryParts.map((line, i) => (
                    <li key={i}>{line}</li>
                  ))}
                </ul>
              )}
              {hasShiftBaseline && deltaSummaryParts.length === 0 && (
                <p className="mt-2 text-xs text-muted-foreground">No deltas detected against your saved baseline on this load.</p>
              )}
            </div>
          </div>
          <button
            type="button"
            onClick={markShiftBaseline}
            className="shrink-0 rounded-xl border border-border/70 bg-background px-3 py-2 text-xs font-semibold hover:bg-muted/50 transition-colors"
          >
            Mark “caught up” baseline
          </button>
        </div>
      </div>

      {/* Open work vs baseline — operational continuity (browser-local) */}
      {hasShiftBaseline &&
        (shiftDelta.stillOpenSinceBaseline.length > 0 ||
          shiftDelta.openIncidentsChangedSince.length > 0 ||
          shiftDelta.noLongerOpenSinceBaseline.length > 0) && (
          <div
            className="rounded-2xl border border-border/60 bg-card/30 p-4"
            aria-label="Open work relative to your saved baseline"
          >
            <div className="flex flex-wrap items-center gap-2 mb-3">
              <Clock className="h-4 w-4 text-muted-foreground shrink-0" />
              <p className="text-sm font-semibold text-foreground">Open work vs your baseline</p>
              <span className="text-[10px] text-muted-foreground/80">
                Same-browser anchor only — not shared across operators.
              </span>
            </div>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {shiftDelta.stillOpenSinceBaseline.length > 0 && (
                <div className="min-w-0">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground mb-1.5">
                    Still open (was open at baseline)
                  </p>
                  <ul className="space-y-1">
                    {shiftDelta.stillOpenSinceBaseline.slice(0, 5).map((inc) => (
                      <li key={inc.id}>
                        <Link
                          to={`/incidents/${encodeURIComponent(inc.id)}`}
                          className="text-sm font-medium text-primary hover:underline truncate block"
                        >
                          {inc.title || inc.id.slice(0, 12)}
                        </Link>
                      </li>
                    ))}
                  </ul>
                  {shiftDelta.stillOpenSinceBaseline.length > 5 && (
                    <Link to="/incidents" className="text-xs text-muted-foreground hover:underline mt-1 inline-block">
                      +{shiftDelta.stillOpenSinceBaseline.length - 5} more on incidents list
                    </Link>
                  )}
                </div>
              )}
              {shiftDelta.openIncidentsChangedSince.length > 0 && (
                <div className="min-w-0">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-warning mb-1.5">
                    Open + touched since baseline
                  </p>
                  <p className="text-[11px] text-muted-foreground mb-1.5">
                    Updated while still open — prioritize review and handoff notes.
                  </p>
                  <ul className="space-y-1">
                    {shiftDelta.openIncidentsChangedSince.slice(0, 5).map((inc) => (
                      <li key={inc.id}>
                        <Link
                          to={`/incidents/${encodeURIComponent(inc.id)}?replay=1`}
                          className="text-sm font-medium text-foreground hover:underline truncate block"
                        >
                          {inc.title || inc.id.slice(0, 12)}
                        </Link>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              {shiftDelta.noLongerOpenSinceBaseline.length > 0 && (
                <div className="min-w-0 sm:col-span-2 lg:col-span-1">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground mb-1.5">
                    No longer open (since baseline)
                  </p>
                  <p className="text-[11px] text-muted-foreground mb-1.5">
                    State changed to resolved/closed in MEL — not proof the underlying mesh issue ended; verify in
                    incident record.
                  </p>
                  <Link to="/incidents" className="text-sm font-medium text-primary hover:underline">
                    Review incidents list →
                  </Link>
                </div>
              )}
            </div>
          </div>
        )}

      <StaleDataBanner lastSuccessfulIngest={dashboardStaleTs} componentName="Dashboard / Transports" />

      {/* Shift order — ranked attention with honest “why now” */}
      {recurrenceTeasers.length > 0 && (
        <section
          className="rounded-2xl border border-border/60 bg-card/30 p-4"
          aria-label="Open incidents with pattern or case memory"
        >
          <div className="flex flex-wrap items-center gap-2 mb-3">
            <TrendingUp className="h-4 w-4 text-warning shrink-0" />
            <h2 className="text-sm font-semibold text-foreground">Case memory on open work</h2>
            <span className="text-[10px] text-muted-foreground/80">
              Deterministic signals from this instance — not ML or causal proof.
            </span>
          </div>
          <ul className="space-y-2">
            {recurrenceTeasers.map((t) => (
              <li key={t.id} className="text-sm">
                <Link
                  to={`/incidents/${encodeURIComponent(t.id)}`}
                  className="font-medium text-primary hover:underline"
                >
                  {t.title}
                </Link>
                <span className="text-muted-foreground text-xs block sm:inline sm:ml-1 sm:before:content-['—_'] sm:before:text-muted-foreground/50">
                  {t.why}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {shiftAttentionRows.length > 0 && (
        <section
          className="rounded-2xl border border-border/60 bg-card/35 p-4"
          aria-label="Shift-start attention order"
        >
          <div className="flex flex-wrap items-center gap-2 mb-3">
            <Compass className="h-4 w-4 text-muted-foreground shrink-0" />
            <h2 className="text-sm font-semibold text-foreground">Shift order (work this pass)</h2>
            <span className="text-[10px] text-muted-foreground/80">
              Single-operator ordering from live posture — not a team queue.
            </span>
          </div>
          <ol className="space-y-2 list-decimal list-inside marker:text-[11px] marker:text-muted-foreground/70">
            {shiftAttentionRows.slice(0, 10).map((row) => (
              <li key={row.key} className="text-sm">
                <Link
                  to={row.href}
                  className="font-medium text-primary hover:underline align-middle"
                >
                  {row.title}
                </Link>
                <span className="text-muted-foreground text-xs block sm:inline sm:ml-1 sm:before:content-['—_'] sm:before:text-muted-foreground/50">
                  {row.why}
                </span>
              </li>
            ))}
          </ol>
          {explicitFollowUpOpenCount > 0 && (
            <p className="mt-3 text-[11px] text-muted-foreground border-t border-border/40 pt-2">
              {explicitFollowUpOpenCount} open incident{explicitFollowUpOpenCount > 1 ? 's' : ''} in follow-up / review workflow
              states — see incidents list for full set.
            </p>
          )}
        </section>
      )}

      {/* Retention / export posture (instance truth) */}
      {!versionInfo.loading && versionInfo.data?.platform_posture && (
        <div
          className="rounded-xl border border-border/50 bg-muted/10 px-3 py-2 text-[11px] text-muted-foreground"
          role="region"
          aria-label="Retention and export policy from this instance"
        >
          <span className="font-semibold text-foreground">Instance policy cues: </span>
          {retentionDays != null && (
            <span>default retention window ~{retentionDays} day audit horizon (see Settings for full matrix). </span>
          )}
          <span className={exportReadiness.semantic === 'policy_limited' ? 'text-warning' : ''}>
            {exportReadiness.summary}{' '}
          </span>
          <Link to="/settings" className="text-primary font-medium hover:underline ml-1">
            Settings → runtime truth
          </Link>
        </div>
      )}

      {/* System Pulse — what needs attention */}
      {attentionCount > 0 && (
        <div className="rounded-2xl border border-warning/25 bg-warning/5 p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl border border-warning/25 bg-warning/12 text-warning">
              <AlertTriangle className="h-4 w-4" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-semibold text-foreground">
                {attentionCount} item{attentionCount > 1 ? 's' : ''} need{attentionCount === 1 ? 's' : ''} attention
              </p>
              <div className="mt-1 flex flex-wrap gap-2">
                {openIncidents.length > 0 && (
                  <Link to="/incidents" className="inline-flex items-center gap-1 text-xs text-warning hover:text-foreground transition-colors">
                    <AlertTriangle className="h-3 w-3" />
                    {openIncidents.length} open incident{openIncidents.length > 1 ? 's' : ''}
                  </Link>
                )}
                {recurringOpenIncidents.length > 0 && (
                  <span className="inline-flex items-center gap-1 text-xs text-muted-foreground" title="Same signature bucket seen before on this instance — not causal proof">
                    <TrendingUp className="h-3 w-3 text-warning" />
                    {recurringOpenIncidents.length} recurring pattern
                    {recurringOpenIncidents.length > 1 ? 's' : ''} (open)
                  </span>
                )}
                {unhealthyTransports > 0 && (
                  <Link to="/status" className="inline-flex items-center gap-1 text-xs text-critical hover:text-foreground transition-colors">
                    <Activity className="h-3 w-3" />
                    {unhealthyTransports} unhealthy transport{unhealthyTransports > 1 ? 's' : ''}
                  </Link>
                )}
                {criticalDiags.length > 0 && (
                  <Link to="/diagnostics" className="inline-flex items-center gap-1 text-xs text-critical hover:text-foreground transition-colors">
                    <Shield className="h-3 w-3" />
                    {criticalDiags.length} diagnostic finding{criticalDiags.length > 1 ? 's' : ''}
                  </Link>
                )}
                {activePrivacyFindings.length > 0 && (
                  <Link to="/privacy" className="inline-flex items-center gap-1 text-xs text-critical hover:text-foreground transition-colors">
                    <Shield className="h-3 w-3" />
                    {activePrivacyFindings.length} privacy finding{activePrivacyFindings.length > 1 ? 's' : ''}
                  </Link>
                )}
                {pendingApprovals.length > 0 && (
                  <Link to="/control-actions" className="inline-flex items-center gap-1 text-xs text-warning hover:text-foreground transition-colors">
                    <Zap className="h-3 w-3" />
                    {pendingApprovals.length} control action{pendingApprovals.length > 1 ? 's' : ''} awaiting approval
                  </Link>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Calm state — when everything is quiet */}
      {attentionCount === 0 && !isLoading && hasTransports && (
        <div className="rounded-2xl border border-border/60 bg-muted/10 p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl border border-border/50 bg-card/60 text-muted-foreground">
              <Compass className="h-4 w-4" />
            </div>
            <div>
              <p className="text-sm font-semibold text-foreground">Attention strip is empty — stay watchful</p>
              <p className="text-xs text-muted-foreground">
                No open incidents, transports report healthy, and operational state shows no pending approvals. That is{' '}
                <span className="font-medium text-foreground/90">not</span> proof the mesh is fully observed, stable, or
                risk-free — only that this console’s current signals are quiet.
                {pendingRecommendations.length > 0 && (
                  <> &middot;{' '}
                    <Link to="/recommendations" className="text-primary hover:underline">
                      {pendingRecommendations.length} recommendation{pendingRecommendations.length > 1 ? 's' : ''} available
                    </Link>
                  </>
                )}
              </p>
            </div>
          </div>
          <div className="border-t border-border/40 pt-3">
            <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground mb-2">
              Still worth a pass (quiet console)
            </p>
            <ul className="text-xs text-muted-foreground space-y-1.5">
              <li>
                <Link
                  to={
                    shiftDelta.topologyNodesWithNewerLastSeen.length > 0
                      ? '/topology?filter=changed_since_visit'
                      : '/topology?filter=degraded'
                  }
                  className="font-medium text-foreground hover:underline"
                >
                  Topology drift / graph evidence
                </Link>
                {shiftDelta.topologyNodesWithNewerLastSeen.length > 0
                  ? ` — ${shiftDelta.topologyNodesWithNewerLastSeen.length} node(s) newer than your baseline.`
                  : ' — open degraded/sparse graph view when nothing moved vs baseline.'}
              </li>
              <li>
                <Link to="/events" className="font-medium text-foreground hover:underline">
                  Events / audit trail
                </Link>
                {' — '}continuity for what changed on the instance.
              </li>
              {deadLetterCount > 0 && (
                <li>
                  <Link to="/dead-letters" className="font-medium text-warning hover:underline">
                    {deadLetterCount} dead letter{deadLetterCount > 1 ? 's' : ''}
                  </Link>
                  {' — '}processing failures still deserve attention.
                </li>
              )}
              <li>
                <Link to="/diagnostics" className="font-medium text-foreground hover:underline">
                  Diagnostics
                </Link>
                {' — '}
                runtime truth and{' '}
                <span className="text-foreground/90">support bundle</span> export when you need host-level continuity.
              </li>
            </ul>
          </div>
        </div>
      )}

      {/* Evidence gaps + next steps */}
      {(sparseIncidents.length > 0 || !connectedTransport || pendingApprovals.length > 0) && (
        <div className="grid gap-3 md:grid-cols-2">
          {sparseIncidents.length > 0 && (
            <div className="rounded-2xl border border-warning/25 bg-warning/5 p-4">
              <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.14em] text-warning mb-2">
                <HelpCircle className="h-3.5 w-3.5" />
                Sparse or degraded incident evidence
              </div>
              <p className="text-xs text-muted-foreground mb-2">
                Open incidents where intelligence is sparse or explicitly degraded — treat conclusions as bounded.
              </p>
              <ul className="space-y-1.5">
                {sparseIncidents.slice(0, 4).map((inc) => (
                  <li key={inc.id}>
                    <Link
                      to={`/incidents/${encodeURIComponent(inc.id)}`}
                      className="text-sm text-foreground hover:underline font-medium truncate block"
                    >
                      {inc.title || inc.id.slice(0, 12)}
                    </Link>
                    <span className="text-[11px] text-muted-foreground">
                      {inc.intelligence?.evidence_strength ?? 'unknown'} evidence
                      {inc.intelligence?.degraded ? ' · degraded intel' : ''}
                    </span>
                  </li>
                ))}
              </ul>
              {sparseIncidents.length > 4 && (
                <Link to="/incidents" className="text-xs font-semibold text-primary mt-2 inline-block hover:underline">
                  View all incidents →
                </Link>
              )}
            </div>
          )}
          <div className="rounded-2xl border border-border/60 bg-card/50 p-4">
            <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground mb-2">
              <Clock className="h-3.5 w-3.5" />
              Suggested next checks
            </div>
            <ul className="text-sm text-muted-foreground space-y-2">
              {!connectedTransport && (
                <li>
                  <Link to="/status" className="text-foreground font-medium hover:underline">Status</Link>
                  {' — '}no active transport; ingest may be idle.
                </li>
              )}
              {pendingApprovals.length > 0 && (
                <li>
                  <Link to="/control-actions" className="text-foreground font-medium hover:underline">Control actions</Link>
                  {' — '}
                  {pendingApprovals.length} pending approval{pendingApprovals.length > 1 ? 's' : ''}.
                </li>
              )}
              {openIncidentsShiftOrder.length > 0 && (
                <li>
                  <Link
                    to={`/incidents/${encodeURIComponent(openIncidentsShiftOrder[0].id)}`}
                    className="text-foreground font-medium hover:underline"
                  >
                    Highest-priority open incident (this pass)
                  </Link>
                  {' — '}
                  {openIncidentsShiftOrder[0].title || openIncidentsShiftOrder[0].id.slice(0, 12)}
                </li>
              )}
              <li>
                <Link
                  to={
                    shiftDelta.topologyNodesWithNewerLastSeen.length > 0
                      ? '/topology?filter=changed_since_visit'
                      : '/topology'
                  }
                  className="text-foreground font-medium hover:underline"
                >
                  Topology
                </Link>
                {' — '}
                {shiftDelta.topologyNodesWithNewerLastSeen.length > 0
                  ? `${shiftDelta.topologyNodesWithNewerLastSeen.length} node(s) changed since your baseline — open with “changed since visit” focus.`
                  : 'compare stale vs healthy nodes from stored graph evidence.'}
              </li>
              <li>
                <Link to="/planning" className="text-foreground font-medium hover:underline">Planning</Link>
                {' — '}resilience posture from topology bounds (not RF simulation).
              </li>
            </ul>
          </div>
        </div>
      )}

      {/* KPI cards */}
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4 stagger-children">
        <StatCard
          title="Connection"
          value={connectedTransport ? 'Connected' : 'Disconnected'}
          description={connectedTransport ? connectedTransport.name : 'No active transport'}
          icon={connectedTransport ? <CheckCircle2 className="h-5 w-5" /> : <XCircle className="h-5 w-5" />}
          variant={connectedTransport ? 'success' : 'warning'}
        />
        <StatCard
          title="Nodes"
          value={nodes.data?.length || 0}
          description={nodes.data?.length === 0 ? 'Awaiting mesh observations' : 'Mesh devices detected'}
          icon={<Radio className="h-5 w-5" />}
          variant="default"
        />
        <StatCard
          title="Messages"
          value={messagesData?.length || status.data?.messages || 0}
          description="Runtime message count"
          icon={<MessageSquare className="h-5 w-5" />}
          variant="info"
        />
        <StatCard
          title="Transport health"
          value={`${healthyTransports}/${totalTransports}`}
          description={
            !hasTransports
              ? 'No transports configured'
              : degradedTransports > 0
                ? `${degradedTransports} degraded`
                : unhealthyTransports > 0
                  ? `${unhealthyTransports} unhealthy`
                  : 'All transports healthy'
          }
          icon={<TrendingUp className="h-5 w-5" />}
          variant={healthyTransports === totalTransports && hasTransports ? 'success' : healthyTransports > 0 ? 'warning' : 'default'}
        />
      </div>

      {/* Main content grid: Activity feed left, key surfaces right */}
      <div className="grid gap-5 xl:grid-cols-5">
        {/* Activity feed — left column, takes more space */}
        <div className="xl:col-span-3">
          <Card>
            <CardHeader className="border-b border-border/50 pb-3">
              <SectionCardHeader
                icon={<FileText className="h-4 w-4" />}
                iconClassName="border-primary/16 bg-primary/12 text-primary"
                title="Recent activity"
                description="Events, incidents, and changes across the mesh"
                href="/events"
              />
            </CardHeader>
            <CardContent className="pt-3">
              <ActivityFeed
                items={feedItems}
                maxItems={10}
                viewAllHref="/events"
                emptyMessage="No recent activity. The system is quiet."
              />
            </CardContent>
          </Card>
        </div>

        {/* Right column: key operational surfaces */}
        <div className="space-y-4 xl:col-span-2">
          {/* Open incidents quick view */}
          <Card>
            <CardHeader className="border-b border-border/50 pb-3">
              <SectionCardHeader
                icon={<AlertTriangle className="h-4 w-4" />}
                iconClassName={openIncidents.length > 0 ? 'border-warning/18 bg-warning/12 text-warning' : 'border-success/16 bg-success/10 text-success'}
                title="Incidents"
                description={openIncidents.length > 0 ? `${openIncidents.length} open` : 'None open'}
                href="/incidents"
              />
            </CardHeader>
            <CardContent className="pt-3">
              {incidents.loading && !incidents.data ? (
                <Loading message="Loading..." />
              ) : openIncidents.length === 0 ? (
                <div className="flex items-center gap-3 py-3">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  <p className="text-sm text-muted-foreground">
                    No open incidents in MEL right now — not proof nothing needs follow-up elsewhere.
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  {openIncidentsShiftOrder.slice(0, 3).map((inc) => (
                    <Link
                      key={inc.id}
                      to={`/incidents/${encodeURIComponent(inc.id)}`}
                      className="block rounded-lg border border-border/60 bg-card/50 p-3 transition-colors hover:bg-accent/50"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0">
                          <p className="truncate text-sm font-medium text-foreground">
                            {inc.title || inc.id.slice(0, 8)}
                          </p>
                          {inc.intelligence?.signature_match_count && inc.intelligence.signature_match_count > 1 && (
                            <p className="mt-0.5 text-[11px] text-warning">
                              Seen {inc.intelligence.signature_match_count} times before
                            </p>
                          )}
                        </div>
                        <div className="flex gap-1.5">
                          {inc.severity && <Badge variant={inc.severity === 'critical' ? 'critical' : inc.severity === 'high' ? 'warning' : 'secondary'}>{inc.severity}</Badge>}
                          <Badge variant="secondary">{inc.intelligence?.evidence_strength ?? 'unknown'}</Badge>
                        </div>
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Transport status */}
          <Card>
            <CardHeader className="border-b border-border/50 pb-3">
              <SectionCardHeader
                icon={<Activity className="h-4 w-4" />}
                iconClassName="border-primary/16 bg-primary/12 text-primary"
                title="Transports"
                description="Health of configured transports"
                href="/status"
              />
            </CardHeader>
            <CardContent className="pt-3">
              {!hasTransports ? (
                <NoTransportsConfigured />
              ) : (
                <div className="space-y-2">
                  {transports.slice(0, 4).map((transport) => (
                    <TransportListItem key={transport.name} transport={transport} />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Recommendations quick view */}
          {pendingRecommendations.length > 0 && (
            <Card>
              <CardHeader className="border-b border-border/50 pb-3">
                <SectionCardHeader
                  icon={<Compass className="h-4 w-4" />}
                  iconClassName="border-warning/16 bg-warning/12 text-warning"
                  title="Recommendations"
                  description={`${pendingRecommendations.length} actionable`}
                  href="/recommendations"
                />
              </CardHeader>
              <CardContent className="pt-3">
                <div className="space-y-2">
                  {pendingRecommendations.slice(0, 3).map((rec, i) => (
                    <RecommendationListItem key={i} recommendation={rec} />
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Dead letters alert */}
          {deadLetterCount > 0 && (
            <Link to="/dead-letters" className="block">
              <div className="flex items-center gap-3 rounded-2xl border border-warning/20 bg-warning/5 p-3 transition-colors hover:bg-warning/8">
                <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-warning/12 text-warning">
                  <Inbox className="h-4 w-4" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-foreground">
                    {deadLetterCount} dead letter{deadLetterCount > 1 ? 's' : ''}
                  </p>
                  <p className="text-xs text-muted-foreground">Messages that failed processing</p>
                </div>
                <ArrowRight className="h-4 w-4 text-muted-foreground" />
              </div>
            </Link>
          )}

          {/* Privacy posture */}
          {activePrivacyFindings.length > 0 && (
            <Card>
              <CardHeader className="border-b border-border/50 pb-3">
                <SectionCardHeader
                  icon={<Shield className="h-4 w-4" />}
                  iconClassName="border-critical/16 bg-critical/10 text-critical"
                  title="Privacy"
                  description={`${activePrivacyFindings.length} critical/high`}
                  href="/privacy"
                />
              </CardHeader>
              <CardContent className="pt-3">
                <div className="space-y-2">
                  {activePrivacyFindings.slice(0, 3).map((finding, i) => (
                    <InlineAlert key={i} variant={finding.severity === 'critical' ? 'critical' : 'warning'}>
                      <div className="flex items-center justify-between gap-2">
                        <span className="truncate text-sm">{finding.message}</span>
                        <SeverityBadge severity={finding.severity} />
                      </div>
                    </InlineAlert>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Bottom row: nodes quick view when nothing dramatic is happening */}
      {openIncidents.length === 0 && (nodes.data?.length ?? 0) > 0 && (
        <Card>
          <CardHeader className="border-b border-border/50 pb-3">
            <SectionCardHeader
              icon={<Radio className="h-4 w-4" />}
              iconClassName="border-border/70 bg-secondary text-secondary-foreground"
              title="Recent nodes"
              description={`${nodes.data?.length ?? 0} devices observed`}
              href="/nodes"
            />
          </CardHeader>
          <CardContent className="pt-3">
            <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
              {nodes.data?.slice(0, 6).map((node) => (
                <NodeCompactItem key={node.node_id} node={node} />
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function SectionCardHeader({
  icon,
  iconClassName,
  title,
  description,
  href,
}: {
  icon: React.ReactNode
  iconClassName: string
  title: string
  description: string
  href: string
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <div className="flex items-center gap-2.5">
        <div className={clsx('flex h-8 w-8 items-center justify-center rounded-xl border shadow-inset', iconClassName)}>
          {icon}
        </div>
        <div>
          <CardTitle className="text-[14px]">{title}</CardTitle>
          <CardDescription className="mt-0.5 text-[11px] text-muted-foreground">
            {description}
          </CardDescription>
        </div>
      </div>
      <Link
        to={href}
        className="inline-flex items-center gap-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-muted-foreground transition-colors hover:text-foreground"
      >
        View <ArrowRight className="h-3 w-3" />
      </Link>
    </div>
  )
}

function TransportListItem({ transport }: { transport: TransportHealth }) {
  const healthState = getHealthState(transport.health)

  return (
    <Link to="/status" className="block">
      <div className="list-row justify-between gap-3 px-3 py-2.5">
        <div className="flex min-w-0 items-center gap-2.5">
          <div
            className={clsx(
              'flex h-8 w-8 shrink-0 items-center justify-center rounded-xl border shadow-inset',
              healthState === 'healthy'
                ? 'border-success/18 bg-success/12 text-success'
                : healthState === 'degraded'
                  ? 'border-warning/18 bg-warning/12 text-warning'
                  : 'border-critical/18 bg-critical/12 text-critical'
            )}
          >
            <Activity className="h-3.5 w-3.5" />
          </div>
          <div className="min-w-0">
            <p className="truncate text-[13px] font-semibold text-foreground">{transport.name}</p>
            <p className="truncate text-[11px] uppercase tracking-[0.14em] text-muted-foreground">{transport.type}</p>
          </div>
        </div>
        <div className="ml-2 flex items-center gap-2">
          <HealthBadge health={healthState} />
        </div>
      </div>
    </Link>
  )
}

function NodeCompactItem({ node }: { node: NodeInfo }) {
  return (
    <div className="flex items-center gap-2.5 rounded-xl border border-border/60 bg-card/50 px-3 py-2.5">
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-border/60 bg-secondary text-secondary-foreground">
        <Radio className="h-3.5 w-3.5" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium text-foreground">{node.long_name || 'Unknown'}</p>
        <p className="text-[11px] text-muted-foreground">
          {formatRelativeTime(node.last_seen)}
        </p>
      </div>
    </div>
  )
}

function RecommendationListItem({
  recommendation,
}: {
  recommendation: { message: string; category?: string; actionable: boolean }
}) {
  return (
    <div className="flex items-start gap-2.5 rounded-lg border border-border/50 bg-card/40 px-3 py-2.5">
      <div
        className={clsx(
          'mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-lg border shadow-inset',
          recommendation.actionable
            ? 'border-warning/18 bg-warning/12 text-warning'
            : 'border-border/70 bg-card/75 text-muted-foreground'
        )}
      >
        {recommendation.actionable ? <AlertCircle className="h-3 w-3" /> : <Zap className="h-3 w-3" />}
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-[13px] leading-relaxed text-foreground">{recommendation.message}</p>
      </div>
    </div>
  )
}
