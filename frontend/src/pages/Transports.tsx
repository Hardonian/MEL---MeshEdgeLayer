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
          {transports.map((transport) => (
            <Card key={transport.name}>
              <CardHeader>
                <CardTitle className="flex items-center justify-between text-base">
                  <span>{transport.name}</span>
                  <Badge variant={
                    transport.health?.state === 'healthy' ? 'success' : 
                    transport.health?.state === 'degraded' ? 'warning' : 'critical'
                  }>
                    {transport.health?.state || 'unknown'}
                  </Badge>
                </CardTitle>
                <CardDescription className="text-xs">
                  {transport.type} transport module
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Runtime State:</span>
                    <span className="font-medium">{transport.runtime_state || 'unknown'}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Messages:</span>
                    <span className="font-medium">{transport.total_messages ?? 0}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Last Ingest:</span>
                    <span className="font-medium">{transport.last_ingest_at || 'Never'}</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
