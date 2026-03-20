
import { useTransports } from '@/hooks/useApi'
import { PageHeader } from '@/components/ui/PageHeader'
import { Loading } from '@/components/ui/StateViews'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'

export function Transports() {
  const { data, loading, error } = useTransports()

  if (loading && !data) {
    return <Loading message="Loading transports..." />
  }

  if (error) {
    return <div>Error: {error}</div>
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Transports"
        description="Monitor the health and status of your transports."
      />

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {data?.transports.map((transport) => (
          <Card key={transport.name}>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span>{transport.name}</span>
                <Badge variant={transport.health.state === 'healthy' ? 'success' : 'destructive'}>
                  {transport.health.state}
                </Badge>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <p>Type: {transport.type}</p>
                <p>State: {transport.runtime_state}</p>
                <p>Messages: {transport.total_messages}</p>
                <p>Last Ingest: {transport.last_ingest_at || 'N/A'}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
