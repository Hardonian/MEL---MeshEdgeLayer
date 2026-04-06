import { useStatus } from '@/hooks/useApi'
import { PageHeader } from '@/components/ui/PageHeader'
import { Loading } from '@/components/ui/StateViews'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card'
import { TransportBadge } from '@/components/ui/Badge'
import { MelPanelInset, MelTruthBadge } from '@/components/ui/operator'
import { AlertCard } from '@/components/ui/AlertCard'
import { NoTransportsConfigured } from '@/components/ui/EmptyState'

export function Transports() {
  const { data, loading, error, refresh } = useStatus()

  if (loading && !data) {
    return <Loading message="Loading transports..." />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load transports"
          description={error}
          action={
            <button type="button" onClick={refresh} className="button-danger">
              Retry
            </button>
          }
        />
      </div>
    )
  }

  const transports = data?.transports || []

  return (
    <div className="space-y-6">
      <PageHeader
        title="Transports"
        subtitle="Mesh operations cockpit"
        description="View ingest evidence, transport health, and degraded conditions."
      />

      {transports.length === 0 ? (
        <NoTransportsConfigured />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {transports.map((transport) => {
            let derivedState = 'Unknown'
            let stateSemantic = 'unknown'

            const isDisconnected = transport.effective_state === 'disconnected'
            const hbDate = transport.last_heartbeat_at ? new Date(transport.last_heartbeat_at) : null
            const now = new Date()
            const heartbeatMs = hbDate ? now.getTime() - hbDate.getTime() : Infinity

            if (isDisconnected) {
              derivedState = 'Disconnected'
              stateSemantic = 'critical'
            } else if (transport.consecutive_timeouts > 0 || heartbeatMs > 5 * 60 * 1000) {
              derivedState = 'Stalled'
              stateSemantic = 'degraded'
            } else {
              const ingestDate = transport.last_ingest_at ? new Date(transport.last_ingest_at) : null
              const ingestMs = ingestDate ? now.getTime() - ingestDate.getTime() : Infinity
              if (ingestMs > 5 * 60 * 1000) {
                derivedState = 'Idle'
                stateSemantic = 'stale'
              } else {
                derivedState = 'Active'
                stateSemantic = 'live'
              }
            }

            return (
              <Card key={transport.name}>
                <CardHeader>
                  <CardTitle className="flex items-center justify-between gap-2 text-base">
                    <span className="min-w-0 truncate">{transport.name}</span>
                    <MelTruthBadge semantic={stateSemantic}>{derivedState}</MelTruthBadge>
                  </CardTitle>
                  <CardDescription className="flex flex-wrap items-center gap-2 text-xs">
                    <TransportBadge type={transport.type} />
                    <span className="text-muted-foreground">
                      runtime: <span className="font-mono text-foreground/90">{transport.runtime_state || 'unknown'}</span>
                    </span>
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2 text-sm">
                    {transport.health?.state && transport.health.state !== 'healthy' && (
                       <div className="flex justify-between text-warning">
                         <span className="font-medium">Health Check:</span>
                         <span className="font-medium">{transport.health.state}</span>
                       </div>
                    )}
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Messages (Total / Drops):</span>
                      <span className="font-medium">{transport.total_messages ?? 0} / {transport.observation_drops ?? 0}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Heartbeat:</span>
                      <span className="font-medium">{transport.last_heartbeat_at ? new Date(transport.last_heartbeat_at).toLocaleTimeString() : 'Never'}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Last Ingest:</span>
                      <span className="font-medium">{transport.last_ingest_at ? new Date(transport.last_ingest_at).toLocaleTimeString() : 'Never'}</span>
                    </div>
                    {(transport.consecutive_timeouts > 0 || transport.last_error) && (
                      <MelPanelInset tone="critical" className="mt-2 text-xs text-foreground" role="status">
                        {transport.last_error || `Timeouts: ${transport.consecutive_timeouts}`}
                      </MelPanelInset>
                    )}
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  )
}
