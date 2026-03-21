import { useStatus } from '@/hooks/useApi'
import { PageHeader } from '@/components/ui/PageHeader'
import { Loading } from '@/components/ui/StateViews'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
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

  return (
    <div className="space-y-6">
      <PageHeader
        title="Transports"
        description="Monitor the health and status of your active transports."
      />

      {transports.length === 0 ? (
        <NoTransportsConfigured />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {transports.map((transport) => {
            let derivedState = 'Unknown';
            let stateVariant: 'default' | 'success' | 'warning' | 'critical' | 'info' = 'default';
            
            const isDisconnected = transport.effective_state === 'disconnected';
            const hbDate = transport.last_heartbeat_at ? new Date(transport.last_heartbeat_at) : null;
            const now = new Date();
            const heartbeatMs = hbDate ? now.getTime() - hbDate.getTime() : Infinity;
            
            if (isDisconnected) {
              derivedState = 'Disconnected';
              stateVariant = 'critical';
            } else if (transport.consecutive_timeouts > 0 || heartbeatMs > 5 * 60 * 1000) {
              derivedState = 'Stalled';
              stateVariant = 'warning';
            } else {
              const ingestDate = transport.last_ingest_at ? new Date(transport.last_ingest_at) : null;
              const ingestMs = ingestDate ? now.getTime() - ingestDate.getTime() : Infinity;
              if (ingestMs > 5 * 60 * 1000) {
                derivedState = 'Idle';
                stateVariant = 'default';
              } else {
                derivedState = 'Active';
                stateVariant = 'success';
              }
            }

            return (
              <Card key={transport.name}>
                <CardHeader>
                  <CardTitle className="flex items-center justify-between text-base">
                    <span>{transport.name}</span>
                    <Badge variant={stateVariant}>
                      {derivedState}
                    </Badge>
                  </CardTitle>
                  <CardDescription className="text-xs">
                    {transport.type} transport module — Runtime: {transport.runtime_state || 'unknown'}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2 text-sm">
                    {transport.health?.state && transport.health.state !== 'healthy' && (
                       <div className="flex justify-between text-amber-600">
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
                      <div className="mt-2 text-xs text-red-600 bg-red-50 p-2 rounded border border-red-100">
                        {transport.last_error || `Timeouts: ${transport.consecutive_timeouts}`}
                      </div>
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
