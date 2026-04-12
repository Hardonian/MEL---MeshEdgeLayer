import { useLayoutEffect } from 'react'
import { useStatus } from '@/hooks/useApi'
import { ProgressBar, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui'
import { MelPageSection, MelPanel } from '@/components/ui/operator'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge, HealthBadge, ConnectionBadge, TransportBadge } from '@/components/ui/Badge'
import { AlertCard, InlineAlert } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { NoTransportsConfigured } from '@/components/ui/EmptyState'
import { getHealthState, formatRelativeTime, TransportHealth } from '@/types/api'
import { Wifi, WifiOff, AlertCircle, Activity, Clock, MessageSquare, TrendingUp, CheckCircle2, HelpCircle, RefreshCw } from 'lucide-react'
import { clsx } from 'clsx'

function transportSectionId(name: string) {
  return 'mel-transport-' + name.replace(/[^a-zA-Z0-9_-]/g, '_')
}

export function Status() {
  const { data, loading, error, refresh } = useStatus()

  useLayoutEffect(() => {
    const id = window.location.hash.replace(/^#/, '')
    if (!id || !id.startsWith('mel-transport-')) return
    const el = document.getElementById(id)
    if (!el) return
    requestAnimationFrame(() => {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }, [data?.transports])

  if (loading && !data) {
    return <Loading message="Loading status..." />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load status"
          description={error}
          action={
            <button
              onClick={refresh}
              className="button-danger"
            >
              Retry
            </button>
          }
        />
      </div>
    )
  }

  const transports = data?.transports || []
  const hasTransports = transports.length > 0

  const connectedCount = transports.filter((t) => t.effective_state === 'connected').length
  const healthyCount = transports.filter((t) => getHealthState(t.health) === 'healthy').length
  const degradedCount = transports.filter((t) => getHealthState(t.health) === 'degraded').length
  const unhealthyCount = transports.filter((t) => getHealthState(t.health) === 'unhealthy').length

  return (
    <div className="space-y-6">
      <PageHeader
        title="System status"
        description="Runtime and transport posture for this MEL instance. Metrics reflect what the API reports now — not historical mesh proof or fleet-wide state."
        statusChips={[
          { label: 'connected', value: `${connectedCount}/${transports.length || 0}`, variant: connectedCount > 0 ? 'complete' : 'stale' },
          {
            label: 'health',
            value: unhealthyCount > 0 ? 'unhealthy present' : degradedCount > 0 ? 'degraded present' : 'stable',
            variant: unhealthyCount > 0 ? 'critical' : degradedCount > 0 ? 'degraded' : 'complete',
          },
          { label: 'runtime', value: loading ? 'refreshing' : 'observed', variant: loading ? 'partial' : 'observed' },
        ]}
        action={
          <button
            onClick={refresh}
            className="button-secondary"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh status
          </button>
        }
      />

      <MelPanel className="overflow-hidden">
        <CardHeader className="border-b border-border/50 pb-4 px-4 pt-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-md border border-primary/16 bg-primary/12 text-primary">
              <Activity className="h-5 w-5" />
            </div>
            <div>
              <CardTitle className="text-base">System overview</CardTitle>
              <CardDescription className="mt-1 text-sm">
                Runtime configuration and transport posture reported by this instance.
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-5 px-4 pb-4">
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <StatCard
              title="Transport Modes"
              value={data?.configured_transport_modes?.length || 0}
              description={data?.configured_transport_modes?.join(', ') || 'None configured'}
              icon={<TrendingUp className="h-4 w-4" />}
              rhythm="console"
            />
            <StatCard
              title="Runtime Messages"
              value={data?.messages || 0}
              description="Messages processed"
              icon={<MessageSquare className="h-4 w-4" />}
              variant="info"
              rhythm="console"
            />
            <StatCard
              title="Connected"
              value={`${connectedCount}/${transports.length}`}
              description="Active transport connections"
              icon={connectedCount > 0 ? <CheckCircle2 className="h-4 w-4" /> : <WifiOff className="h-4 w-4" />}
              variant={connectedCount > 0 ? 'success' : hasTransports ? 'warning' : 'unavailable'}
              rhythm="console"
            />
            <StatCard
              title="Healthy"
              value={healthyCount}
              description={
                unhealthyCount > 0
                  ? `${unhealthyCount} unhealthy`
                  : degradedCount > 0
                    ? `${degradedCount} degraded`
                    : 'All configured transports operational'
              }
              icon={<Activity className="h-4 w-4" />}
              variant={
                healthyCount === transports.length && transports.length > 0
                  ? 'success'
                  : unhealthyCount > 0
                    ? 'critical'
                    : degradedCount > 0
                      ? 'partial'
                      : 'unavailable'
              }
              rhythm="console"
            />
          </div>
        </CardContent>
      </MelPanel>

      <MelPanel className="overflow-hidden">
        <CardHeader className="border-b border-border/50 pb-4 px-4 pt-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-md border border-info/16 bg-info/12 text-info">
                <Wifi className="h-5 w-5" />
              </div>
              <div>
                <CardTitle className="text-base">Transport health</CardTitle>
                <CardDescription className="mt-1 text-sm">
                  Per-transport evidence from the status API — verify live behavior on the wire when it matters.
                </CardDescription>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={connectedCount > 0 ? 'success' : 'secondary'}>{connectedCount} connected</Badge>
              <Badge variant={unhealthyCount > 0 ? 'critical' : degradedCount > 0 ? 'warning' : 'success'}>
                {unhealthyCount > 0 ? `${unhealthyCount} unhealthy` : degradedCount > 0 ? `${degradedCount} degraded` : 'stable'}
              </Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-5 px-4 pb-4">
          {!hasTransports ? (
            <NoTransportsConfigured />
          ) : (
            <div className="space-y-4">
              {transports.map((transport) => (
                <TransportDetailCard key={transport.name} transport={transport} />
              ))}
            </div>
          )}
        </CardContent>
      </MelPanel>

      <MelPageSection
        eyebrow="Reference"
        title="Health semantics"
        description="Labels follow API-reported evidence. UI emphasis does not strengthen the underlying signal."
      >
        <MelPanel className="overflow-hidden">
          <CardHeader className="border-b border-border/50 pb-4 px-4 pt-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-md border border-border/70 bg-card/75 text-muted-foreground">
                <HelpCircle className="h-5 w-5" />
              </div>
              <div>
                <CardTitle className="text-base">Reading transport health</CardTitle>
                <CardDescription className="mt-1 text-sm">
                  Use badges and scores as reported by this instance — not as proof of RF coverage or mesh routing success.
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="pt-5 px-4 pb-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <HealthExplanation
                status="healthy"
                description="Connected ingest path reporting normal operation for this gateway’s view."
              />
              <HealthExplanation
                status="degraded"
                description="Intermittent connection, elevated errors, or latency pressure reported — monitor and verify on the transport."
              />
              <HealthExplanation
                status="unhealthy"
                description="Critical failure on the reported path — messages may be dropped until the transport recovers."
              />
            </div>
          </CardContent>
        </MelPanel>
      </MelPageSection>
    </div>
  )
}

function TransportDetailCard({ transport }: { transport: TransportHealth }) {
  const healthState = getHealthState(transport.health)
  const isConnected = transport.effective_state === 'connected'

  return (
    <div
      id={transportSectionId(transport.name)}
      className={clsx(
        'surface-panel surface-panel-muted interactive-lift overflow-hidden rounded-md p-4 sm:p-5 scroll-mt-24',
        healthState === 'healthy'
          ? 'border-success/20'
          : healthState === 'degraded'
            ? 'border-warning/22'
            : 'border-critical/22'
      )}
    >
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="flex min-w-0 items-start gap-3">
          <div
            className={clsx(
              'flex h-11 w-11 shrink-0 items-center justify-center rounded-md border',
              healthState === 'healthy'
                ? 'border-success/18 bg-success/12 text-success'
                : healthState === 'degraded'
                  ? 'border-warning/18 bg-warning/12 text-warning'
                  : 'border-critical/18 bg-critical/12 text-critical'
            )}
          >
            {isConnected ? <Wifi className="h-5 w-5" /> : <WifiOff className="h-5 w-5 text-muted-foreground" />}
          </div>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="font-mono text-lg font-semibold tracking-[-0.03em] text-foreground">{transport.name}</h3>
              <TransportBadge type={transport.type} />
            </div>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <ConnectionBadge
                state={isConnected ? 'connected' : transport.effective_state === 'error' ? 'error' : 'disconnected'}
              />
              <HealthBadge health={healthState} />
              <Badge variant="outline">Runtime {transport.runtime_state || 'unknown'}</Badge>
            </div>
          </div>
        </div>

        <div className="surface-inset flex flex-wrap items-center gap-3 rounded-md px-3 py-2">
          <div>
            <p className="text-mel-sm font-semibold uppercase tracking-[0.16em] text-muted-foreground">Last activity</p>
            <p className="mt-1 text-sm font-semibold text-foreground">
              {transport.last_heartbeat_at ? formatRelativeTime(transport.last_heartbeat_at) : 'Never'}
            </p>
          </div>
          <div className="hidden h-8 w-px bg-border/60 sm:block" />
          <div>
            <p className="text-mel-sm font-semibold uppercase tracking-[0.16em] text-muted-foreground">Guidance</p>
            <p className="mt-1 text-sm text-foreground">{transport.guidance || 'No additional guidance reported.'}</p>
          </div>
        </div>
      </div>

      <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatBox
          label="Effective State"
          value={transport.effective_state || 'unknown'}
          subValue={`runtime: ${transport.runtime_state || 'unknown'}`}
          icon={<Activity className="h-3.5 w-3.5" />}
        />
        <StatBox
          label="Messages"
          value={(transport.total_messages ?? 0).toString()}
          subValue={`${transport.persisted_messages ?? 0} persisted`}
          icon={<MessageSquare className="h-3.5 w-3.5" />}
        />
        <StatBox
          label="Timeouts"
          value={(transport.consecutive_timeouts ?? 0).toString()}
          subValue="consecutive"
          icon={<AlertCircle className="h-3.5 w-3.5" />}
          highlight={(transport.consecutive_timeouts ?? 0) > 3}
        />
        <StatBox
          label="Last Activity"
          value={transport.last_heartbeat_at ? formatRelativeTime(transport.last_heartbeat_at) : 'Never'}
          subValue="heartbeat"
          icon={<Clock className="h-3.5 w-3.5" />}
        />
      </div>

      {transport.health && (
        <div className="mt-4 grid gap-4 border-t border-border/50 pt-4 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
          <div className="space-y-3">
            <div>
              <p className="text-mel-sm font-semibold uppercase tracking-[0.16em] text-muted-foreground">Health score</p>
              <div className="mt-2 flex items-center gap-3">
                <ProgressBar
                  value={transport.health.score}
                  variant={healthState === 'healthy' ? 'success' : healthState === 'degraded' ? 'warning' : 'critical'}
                  className="mt-1"
                />
                <span className="font-mono text-sm font-semibold text-foreground">{transport.health.score}</span>
              </div>
              {transport.health.primary_reason && (
                <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                  Primary reason: {transport.health.primary_reason}
                </p>
              )}
            </div>

            {transport.health.explanation && transport.health.explanation.length > 0 && (
              <div className="raw-block px-4 py-3">
                <p className="text-sm leading-relaxed text-muted-foreground">
                  {transport.health.explanation.join(' ')}
                </p>
              </div>
            )}
          </div>

          <div>
            <p className="text-mel-sm font-semibold uppercase tracking-[0.16em] text-muted-foreground">Active alerts</p>
            {transport.active_alerts && transport.active_alerts.length > 0 ? (
              <div className="mt-2 flex flex-wrap gap-2">
                {transport.active_alerts.map((alert, i) => (
                  <Badge key={i} variant="warning">
                    {alert}
                  </Badge>
                ))}
              </div>
            ) : (
              <div className="surface-inset mt-2 rounded-md border-success/18 bg-success/10 px-3 py-3 text-sm font-medium text-success">
                None
              </div>
            )}
          </div>
        </div>
      )}

      {transport.last_error && (
        <div className="mt-4 border-t border-border/50 pt-4">
          <InlineAlert variant="critical">
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0">
                <p className="text-mel-sm font-semibold uppercase tracking-[0.16em] text-muted-foreground">Last error</p>
                <p className="mt-1 break-all font-mono text-xs leading-relaxed text-foreground">{transport.last_error}</p>
              </div>
            </div>
          </InlineAlert>
        </div>
      )}
    </div>
  )
}

function StatBox({
  label,
  value,
  subValue,
  icon,
  highlight,
}: {
  label: string
  value: string
  subValue: string
  icon: React.ReactNode
  highlight?: boolean
}) {
  return (
    <div
      className={clsx(
        'surface-inset rounded-md px-3 py-3',
        highlight && 'border-critical/22 bg-critical/10'
      )}
    >
      <div className="mb-1 flex items-center gap-1.5 text-mel-sm font-semibold uppercase tracking-[0.16em] text-muted-foreground">
        {icon}
        {label}
      </div>
      <p className={clsx('font-mono text-base font-semibold', highlight ? 'text-critical' : 'text-foreground')}>
        {value}
      </p>
      <p className="mt-1 text-xs uppercase tracking-[0.12em] text-muted-foreground">{subValue}</p>
    </div>
  )
}

function HealthExplanation({
  status,
  description,
}: {
  status: 'healthy' | 'degraded' | 'unhealthy'
  description: string
}) {
  const tones = {
    healthy: 'border-success/20 bg-success/10',
    degraded: 'border-warning/20 bg-warning/10',
    unhealthy: 'border-critical/20 bg-critical/10',
  } as const

  return (
    <div className={clsx('surface-inset rounded-md p-4', tones[status])}>
      <div className="flex items-center gap-2">
        <HealthBadge health={status} />
      </div>
      <p className="mt-3 text-sm leading-relaxed text-muted-foreground">{description}</p>
    </div>
  )
}
