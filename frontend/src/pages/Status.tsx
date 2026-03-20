import { useStatus } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge, HealthBadge } from '@/components/ui/Badge'
import { Loading, ErrorView, EmptyState } from '@/components/ui/StateViews'
import { getHealthState, formatRelativeTime, TransportHealth } from '@/types/api'
import { Wifi, WifiOff, AlertCircle } from 'lucide-react'
import { clsx } from 'clsx'

export function Status() {
  const { data, loading, error, refresh } = useStatus()

  if (loading && !data) {
    return <Loading message="Loading status..." />
  }

  if (error && !data) {
    return <ErrorView message={error} onRetry={refresh} />
  }

  const transports = data?.transports || []

  return (
    <div className="space-y-8">
      {/* Page Header */}
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Status</h1>
        <p className="text-muted-foreground">
          Detailed view of transport health and system status.
        </p>
      </div>

      {/* Configured Transport Modes */}
      <Card>
        <CardHeader>
          <CardTitle>System Overview</CardTitle>
          <CardDescription>Current MEL runtime configuration and status</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            <div className="rounded-lg border p-4">
              <p className="text-sm text-muted-foreground">Configured Modes</p>
              <div className="mt-2 flex flex-wrap gap-2">
                {data?.configured_transport_modes?.map(mode => (
                  <Badge key={mode} variant="outline">{mode}</Badge>
                )) || <span className="text-muted-foreground">None</span>}
              </div>
            </div>
            <div className="rounded-lg border p-4">
              <p className="text-sm text-muted-foreground">Runtime Messages</p>
              <p className="mt-1 text-2xl font-semibold">{data?.messages || 0}</p>
            </div>
            <div className="rounded-lg border p-4">
              <p className="text-sm text-muted-foreground">Active Transports</p>
              <p className="mt-1 text-2xl font-semibold">
                {transports.filter(t => t.effective_state === 'connected').length} / {transports.length}
              </p>
            </div>
            <div className="rounded-lg border p-4">
              <p className="text-sm text-muted-foreground">Healthy Transports</p>
              <p className="mt-1 text-2xl font-semibold">
                {transports.filter(t => getHealthState(t.health) === 'healthy').length}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Transport Details */}
      <Card>
        <CardHeader>
          <CardTitle>Transport Health</CardTitle>
          <CardDescription>
            Detailed health information for each configured transport
          </CardDescription>
        </CardHeader>
        <CardContent>
          {transports.length === 0 ? (
            <EmptyState
              icon={<WifiOff className="h-10 w-10" />}
              title="No transports configured"
              description="Add transport configuration to your MEL config file to start receiving mesh data."
            />
          ) : (
            <div className="space-y-6">
              {transports.map((transport) => (
                <TransportCard key={transport.name} transport={transport} />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function TransportCard({ transport }: { transport: TransportHealth }) {
  const healthState = getHealthState(transport.health)

  return (
    <div className={clsx(
      'rounded-lg border p-4',
      healthState === 'healthy' ? 'border-success/20 bg-success/5' :
      healthState === 'degraded' ? 'border-warning/20 bg-warning/5' :
      'border-critical/20 bg-critical/5'
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
            {transport.effective_state === 'connected' ? (
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
            <h3 className="font-semibold">{transport.name}</h3>
            <p className="text-sm text-muted-foreground">{transport.type}</p>
          </div>
        </div>
        <HealthBadge health={healthState} />
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <p className="text-xs text-muted-foreground">Effective State</p>
          <p className="font-mono text-sm mt-1">{transport.effective_state || 'unknown'}</p>
          <p className="text-xs text-muted-foreground">runtime: {transport.runtime_state || 'unknown'}</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Messages</p>
          <p className="font-mono text-sm mt-1">{transport.total_messages} runtime</p>
          <p className="text-xs text-muted-foreground">{transport.persisted_messages} persisted</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Timeouts</p>
          <p className="font-mono text-sm mt-1">{transport.consecutive_timeouts}</p>
          <p className="text-xs text-muted-foreground">consecutive</p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Last Activity</p>
          <p className="font-mono text-sm mt-1">
            {transport.last_heartbeat_at ? formatRelativeTime(transport.last_heartbeat_at) : 'Never'}
          </p>
          <p className="text-xs text-muted-foreground">heartbeat</p>
        </div>
      </div>

      {/* Health Details */}
      {transport.health && (
        <div className="mt-4 pt-4 border-t">
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <p className="text-xs text-muted-foreground">Health Score</p>
              <p className="font-mono text-sm mt-1">{transport.health.score} / 100</p>
              <p className="text-xs text-muted-foreground">primary: {transport.health.primary_reason || 'none'}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Active Alerts</p>
              {transport.active_alerts && transport.active_alerts.length > 0 ? (
                <div className="mt-1 space-y-1">
                  {transport.active_alerts.map((alert, i) => (
                    <Badge key={i} variant="warning" className="text-xs">{alert}</Badge>
                  ))}
                </div>
              ) : (
                <p className="text-sm mt-1 text-success">None</p>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Error Info */}
      {transport.last_error && (
        <div className="mt-4 pt-4 border-t">
          <div className="flex items-start gap-2">
            <AlertCircle className="h-4 w-4 text-critical mt-0.5" />
            <div>
              <p className="text-xs text-critical font-medium">Last Error</p>
              <p className="text-sm font-mono mt-1">{transport.last_error}</p>
            </div>
          </div>
        </div>
      )}

      {/* Guidance */}
      {transport.guidance && (
        <div className="mt-4 pt-4 border-t">
          <p className="text-xs text-muted-foreground">Guidance</p>
          <p className="text-sm mt-1">{transport.guidance}</p>
        </div>
      )}
    </div>
  )
}
