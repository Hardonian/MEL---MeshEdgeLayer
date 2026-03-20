import { useEvents } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Loading, ErrorView, EmptyState } from '@/components/ui/StateViews'
import { formatTimestamp, AuditLog } from '@/types/api'
import { ScrollText, Clock } from 'lucide-react'
import { clsx } from 'clsx'

export function Events() {
  const { data, loading, error, refresh } = useEvents()

  if (loading && !data) {
    return <Loading message="Loading events..." />
  }

  if (error && !data) {
    return <ErrorView message={error} onRetry={refresh} />
  }

  const events = data || []

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Events</h1>
        <p className="text-muted-foreground">
          System audit logs and events.
        </p>
      </div>

      {/* Events List */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Events</CardTitle>
          <CardDescription>
            Audit logs and system events from MEL operations
          </CardDescription>
        </CardHeader>
        <CardContent>
          {events.length === 0 ? (
            <EmptyState
              icon={<ScrollText className="h-10 w-10" />}
              title="No events yet"
              description="No audit logs have been recorded yet. Events will appear as MEL processes mesh data."
            />
          ) : (
            <div className="space-y-3">
              {events.map((event, i) => (
                <EventRow key={i} event={event} />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function EventRow({ event }: { event: AuditLog }) {
  const levelColors = {
    info: 'border-l-primary',
    warning: 'border-l-warning',
    error: 'border-l-critical',
    debug: 'border-l-muted',
  }

  const levelBadges = {
    info: 'default',
    warning: 'warning',
    error: 'critical',
    debug: 'secondary',
  } as const

  return (
    <div className={clsx(
      'rounded-lg border border-l-4 p-4 pl-6',
      levelColors[event.level?.toLowerCase() as keyof typeof levelColors] || 'border-l-muted'
    )}>
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <Badge variant={levelBadges[event.level?.toLowerCase() as keyof typeof levelBadges] || 'secondary'}>
              {event.level || 'unknown'}
            </Badge>
            <span className="text-xs text-muted-foreground">{event.category || 'system'}</span>
          </div>
          <p className="text-sm">{event.message}</p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground whitespace-nowrap">
          <Clock className="h-3 w-3" />
          {formatTimestamp(event.created_at)}
        </div>
      </div>
    </div>
  )
}
