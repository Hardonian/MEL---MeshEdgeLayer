import { useEvents } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatTimestamp, AuditLog } from '@/types/api'
import { ScrollText, Clock, AlertTriangle, Info, AlertCircle, FileText, HelpCircle } from 'lucide-react'
import { clsx } from 'clsx'

export function Events() {
  const { data, loading, error, refresh } = useEvents()

  if (loading && !data) {
    return <Loading message="Loading events..." />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load events"
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

  const events = data || []

  // Calculate stats
  const errorCount = events.filter(e => e.level?.toLowerCase() === 'error').length
  const warningCount = events.filter(e => e.level?.toLowerCase() === 'warning').length
  const categories = [...new Set(events.map(e => e.category).filter(Boolean))]

  return (
    <div className="space-y-6">
      <PageHeader
        title="Events"
        description="System audit logs and events. Track what happens in your MEL instance over time."
      />

      {/* Summary */}
      <div className="grid gap-4 sm:grid-cols-4">
        <StatCard
          title="Total Events"
          value={events.length}
          description="Events in current view"
          icon={<ScrollText className="h-5 w-5" />}
          variant="default"
        />
        <StatCard
          title="Errors"
          value={errorCount}
          description="Error-level events"
          icon={<AlertCircle className="h-5 w-5" />}
          variant={errorCount > 0 ? 'critical' : 'success'}
        />
        <StatCard
          title="Warnings"
          value={warningCount}
          description="Warning-level events"
          icon={<AlertTriangle className="h-5 w-5" />}
          variant={warningCount > 0 ? 'warning' : 'default'}
        />
        <StatCard
          title="Categories"
          value={categories.length}
          description="Unique event sources"
          icon={<FileText className="h-5 w-5" />}
          variant="info"
        />
      </div>

      {/* Explanation */}
      <Card className="bg-muted/30">
        <CardHeader className="pb-3">
          <div className="flex items-center gap-2">
            <HelpCircle className="h-5 w-5 text-muted-foreground" />
            <CardTitle className="text-base">About Events</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="pt-0">
          <p className="text-sm text-muted-foreground">
            Events are recorded audit logs of MEL operations. They help you understand:
          </p>
          <ul className="mt-2 list-disc pl-4 text-sm text-muted-foreground space-y-1">
            <li>System startup and shutdown sequences</li>
            <li>Transport connection state changes</li>
            <li>Configuration changes</li>
            <li>Error conditions and recovery actions</li>
            <li>Message processing milestones</li>
          </ul>
        </CardContent>
      </Card>

      {/* Events List */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <CardTitle>Recent Events</CardTitle>
            <Badge variant="outline">{events.length} events</Badge>
          </div>
          <CardDescription>
            Audit logs and system events from MEL operations
          </CardDescription>
        </CardHeader>
        <CardContent>
          {events.length === 0 ? (
            <EmptyState
              type="no-data"
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
    info: 'border-l-primary bg-primary/5',
    warning: 'border-l-amber-500 bg-amber-50 dark:bg-amber-950/20',
    error: 'border-l-red-500 bg-red-50 dark:bg-red-950/20',
    debug: 'border-l-muted bg-muted/30',
  }

  const levelIcons = {
    info: Info,
    warning: AlertTriangle,
    error: AlertCircle,
    debug: FileText,
  }

  const levelBadge = {
    info: 'default',
    warning: 'warning',
    error: 'critical',
    debug: 'secondary',
  } as const

  const icon = levelIcons[event.level?.toLowerCase() as keyof typeof levelIcons] || Info

  return (
    <div className={clsx(
      'rounded-lg border border-l-4 p-4 transition-colors hover:bg-muted/30',
      levelColors[event.level?.toLowerCase() as keyof typeof levelColors] || levelColors.info
    )}>
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1 flex-wrap">
            <Badge variant={levelBadge[event.level?.toLowerCase() as keyof typeof levelBadge] || 'secondary'}>
              {event.level || 'unknown'}
            </Badge>
            <span className="text-xs text-muted-foreground">{event.category || 'system'}</span>
          </div>
          <p className="text-sm">{event.message}</p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground whitespace-nowrap shrink-0">
          <Clock className="h-3 w-3" />
          {formatTimestamp(event.created_at)}
        </div>
      </div>
    </div>
  )
}
