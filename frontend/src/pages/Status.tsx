import { useStatus } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge, HealthBadge, ConnectionBadge, TransportBadge } from '@/components/ui/Badge'
import { AlertCard, InlineAlert } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { NoTransportsConfigured } from '@/components/ui/EmptyState'
import { getHealthState, formatRelativeTime, TransportHealth } from '@/types/api'
import { Wifi, WifiOff, AlertCircle, Activity, Clock, MessageSquare, TrendingUp, CheckCircle2, HelpCircle } from 'lucide-react'
import { clsx } from 'clsx'

export function Status() {
  const { data, loading, error, refresh } = useStatus()

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
              className="rounded-lg bg-critical px-4 py-2 text-sm font-medium text-white hover:bg-critical/90"
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

  const connectedCount = transports.filter(t => t.effective_state === 'connected').length
  const healthyCount = transports.filter(t => getHealthState(t.health) === 'healthy').length
  const degradedCount = transports.filter(t => getHealthState(t.health) === 'degraded').length
  const unhealthyCount = transports.filter(t => getHealthState(t.health) === 'unhealthy').length

  return (
    <div className="space-y-6">
      <PageHeader
        title="System Status"
        description="Detailed view of transport health, connection status, and system metrics."
      />

      {/* System Overview */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center gap-2">
            <Activity className="h-5 w-5 text-muted-foreground" />
            <CardTitle className="text-base">System Overview</CardTitle>
          </div>
          <CardDescription>Current MEL runtime configuration and status</CardDescription>
        </CardHeader>
        <CardContent className="pt-0">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              title="Transport Modes"
              value={data?.configured_transport_modes?.length || 0}
              description={data?.configured_transport_modes?.join(', ') || 'None configured'}
              icon={<TrendingUp className="h-4 w-4" />}
            />
            <StatCard
              title="Runtime Messages"
              value={data?.messages || 0}
              description="Messages processed"
              icon={<MessageSquare className="h-4 w-4" />}
            />
            <StatCard
              title="Connected"
              value={`${connectedCount}/${transports.length}`}
              description="Active transport connections"
              icon={connectedCount > 0 ? <CheckCircle2 className="h-4 w-4" /> : <WifiOff className="h-4 w-4" />}
              variant={connectedCount > 0 ? 'success' : 'warning'}
            />
            <StatCard
              title="Healthy"
              value={healthyCount}
              description={unhealthyCount > 0 ? `${unhealthyCount} unhealthy` : degradedCount > 0 ? `${degradedCount} degraded` : 'All systems operational'}
              icon={<Activity className="h-4 w-4" />}
              variant={healthyCount === transports.length && transports.length > 0 ? 'success' : unhealthyCount > 0 ? 'critical' : 'warning'}
            />
          </div>
        </CardContent>
      </Card>

      {/* Transport Health Details */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center gap-2">
            <Wifi className="h-5 w-5 text-muted-foreground" />
            <CardTitle className="text-base">Transport Health</CardTitle>
          </div>
          <CardDescription>
            Detailed health information for each configured transport
          </CardDescription>
        </CardHeader>
        <CardContent className="pt-0">
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
      </Card>

      {/* Health Explanation */}
      <Card className="bg-muted/30">
        <CardHeader className="pb-3">
          <div className="flex items-center gap-2">
            <HelpCircle className="h-5 w-5 text-muted-foreground" />
            <CardTitle className="text-base">Understanding Transport Health</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="pt-0">
          <div className="grid gap-4 sm:grid-cols-3">
            <HealthExplanation
              status="healthy"
              description="Transport is connected and functioning normally. Messages are being received and processed without issues."
            />
            <HealthExplanation
              status="degraded"
              description="Transport is experiencing issues such as intermittent connections, high latency, or elevated error rates. Monitor for resolution."
            />
            <HealthExplanation
              status="unhealthy"
              description="Transport has critical issues preventing normal operation. Messages may be lost. Immediate attention recommended."
            />
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function TransportDetailCard({ transport }: { transport: TransportHealth }) {
  const healthState = getHealthState(transport.health)
  const isConnected = transport.effective_state === 'connected'

  return (
    <div className={clsx(
      'rounded-lg border p-4 transition-all',
      healthState === 'healthy' ? 'border-success/20 bg-success/5' :
      healthState === 'degraded' ? 'border-warning/20 bg-warning/5' :
      'border-critical/10 bg-critical/5'
    )}>
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className={clsx(
            'flex h-10 w-10 items-center justify-center rounded-lg',
            healthState === 'healthy' ? 'bg-success/10' :
            healthState === 'degraded' ? 'bg-warning/10' :
            'bg-critical/10'
          )}>
            {isConnected ? (
              <Wifi className={clsx(
                'h-5 w-5',
                healthState === 'healthy' ? 'text-success' :
                healthState === 'degraded' ? 'text-warning' :
                'text-critical'
              )} />
            ) : (
              <WifiOff className="h-5 w-5 text-muted-foreground" />
            )}
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="font-semibold">{transport.name}</h3>
              <TransportBadge type={transport.type} />
            </div>
            <div className="flex items-center gap-2 mt-0.5">
              <ConnectionBadge state={isConnected ? 'connected' : transport.effective_state === 'error' ? 'error' : 'disconnected'} />
              <HealthBadge health={healthState} />
            </div>
          </div>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <StatBox
          label="Effective State"
          value={transport.effective_state || 'unknown'}
          subValue={`runtime: ${transport.runtime_state || 'unknown'}`}
          icon={<Activity className="h-3.5 w-3.5" />}
        />
        <StatBox
          label="Messages"
          value={transport.total_messages.toString()}
          subValue={`${transport.persisted_messages} persisted`}
          icon={<MessageSquare className="h-3.5 w-3.5" />}
        />
        <StatBox
          label="Timeouts"
          value={transport.consecutive_timeouts.toString()}
          subValue="consecutive"
          icon={<AlertCircle className="h-3.5 w-3.5" />}
          highlight={transport.consecutive_timeouts > 3}
        />
        <StatBox
          label="Last Activity"
          value={transport.last_heartbeat_at ? formatRelativeTime(transport.last_heartbeat_at) : 'Never'}
          subValue="heartbeat"
          icon={<Clock className="h-3.5 w-3.5" />}
        />
      </div>

      {/* Health Details */}
      {transport.health && (
        <div className="mt-4 pt-4 border-t border-border/50">
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <p className="text-xs text-muted-foreground mb-1">Health Score</p>
              <div className="flex items-center gap-2">
                <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden">
                  <div 
                    className={clsx(
                      'h-full rounded-full',
                      healthState === 'healthy' ? 'bg-success' :
                      healthState === 'degraded' ? 'bg-warning' :
                      'bg-critical'
                    )}
                    style={{ width: `${transport.health.score}%` }}
                  />
                </div>
                <span className="text-sm font-mono font-medium">{transport.health.score}</span>
              </div>
              {transport.health.primary_reason && (
                <p className="text-xs text-muted-foreground mt-1">
                  Primary: {transport.health.primary_reason}
                </p>
              )}
            </div>
            <div>
              <p className="text-xs text-muted-foreground mb-1">Active Alerts</p>
              {transport.active_alerts && transport.active_alerts.length > 0 ? (
                <div className="flex flex-wrap gap-1">
                  {transport.active_alerts.map((alert, i) => (
                    <Badge key={i} variant="warning" className="text-xs">{alert}</Badge>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-success">None</p>
              )}
            </div>
          </div>
          {transport.health.explanation && transport.health.explanation.length > 0 && (
            <div className="mt-3 p-2 rounded bg-muted/50">
              <p className="text-xs text-muted-foreground">
                {transport.health.explanation.join(' ')}
              </p>
            </div>
          )}
        </div>
      )}

      {/* Error Info */}
      {transport.last_error && (
        <div className="mt-4 pt-4 border-t border-border/50">
          <InlineAlert variant="critical">
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="text-xs font-medium">Last Error</p>
                <p className="text-xs font-mono mt-0.5">{transport.last_error}</p>
              </div>
            </div>
          </InlineAlert>
        </div>
      )}

      {/* Guidance */}
      {transport.guidance && (
        <div className="mt-4 pt-4 border-t border-border/50">
          <p className="text-xs text-muted-foreground mb-1">Guidance</p>
          <p className="text-sm">{transport.guidance}</p>
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
  highlight 
}: { 
  label: string
  value: string
  subValue: string
  icon: React.ReactNode
  highlight?: boolean
}) {
  return (
    <div className={clsx(
      'rounded-md p-2.5',
      highlight ? 'bg-critical/5 border border-critical/20' : 'bg-muted/30'
    )}>
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
        {icon}
        {label}
      </div>
      <p className={clsx(
        'text-sm font-mono font-medium',
        highlight ? 'text-critical' : 'text-foreground'
      )}>
        {value}
      </p>
      <p className="text-xs text-muted-foreground">{subValue}</p>
    </div>
  )
}

function HealthExplanation({ status, description }: { status: string; description: string }) {
  const colors = {
    healthy: 'text-success',
    degraded: 'text-warning',
    unhealthy: 'text-critical',
  }
  
  return (
    <div className="space-y-1">
      <p className={clsx('text-sm font-medium', colors[status as keyof typeof colors])}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </p>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
  )
}
