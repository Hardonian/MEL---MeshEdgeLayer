import { useState, useMemo } from 'react'
import { useEvents } from '@/hooks/useApi'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatRelativeTime, AuditLog } from '@/types/api'
import { ScrollText, AlertTriangle, AlertCircle, FileText, Clock, RefreshCw, Filter } from 'lucide-react'
import { clsx } from 'clsx'

const LEVEL_FILTERS = ['all', 'error', 'warning', 'info', 'debug'] as const
type LevelFilter = (typeof LEVEL_FILTERS)[number]

export function Events() {
  const { data, loading, error, refresh } = useEvents()
  const [levelFilter, setLevelFilter] = useState<LevelFilter>('all')

  const events = data || []

  const errorCount = events.filter(e => e.level?.toLowerCase() === 'error').length
  const warningCount = events.filter(e => e.level?.toLowerCase() === 'warning').length
  const categories = [...new Set(events.map(e => e.category).filter(Boolean))]

  const filtered = useMemo(() => {
    if (levelFilter === 'all') return events
    return events.filter(e => e.level?.toLowerCase() === levelFilter)
  }, [events, levelFilter])

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
          action={<button onClick={refresh} className="button-danger">Retry</button>}
        />
      </div>
    )
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <PageHeader
          title="Events"
          description="System audit logs. Track what happens in your MEL instance over time."
        />
        <button onClick={refresh} className="button-secondary">
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      <div className="grid gap-3 sm:grid-cols-4">
        <StatCard
          title="Total"
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

      {/* Level filter */}
      <div className="flex flex-wrap items-center gap-2" role="radiogroup" aria-label="Filter by level">
        <Filter className="h-3.5 w-3.5 text-muted-foreground" />
        {LEVEL_FILTERS.map((level) => (
          <button
            key={level}
            type="button"
            role="radio"
            aria-checked={levelFilter === level}
            onClick={() => setLevelFilter(level)}
            className={clsx(
              'rounded-full border px-3 py-1 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
              levelFilter === level
                ? 'border-primary bg-primary/10 text-primary'
                : 'border-border text-muted-foreground hover:bg-muted'
            )}
          >
            {level === 'all' ? `All (${events.length})` : `${level} (${events.filter(e => e.level?.toLowerCase() === level).length})`}
          </button>
        ))}
      </div>

      <Card>
        <CardHeader className="border-b border-border/50 pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-[14px]">
              {levelFilter === 'all' ? 'All events' : `${levelFilter} events`}
            </CardTitle>
            <Badge variant="outline">{filtered.length} shown</Badge>
          </div>
        </CardHeader>
        <CardContent className="pt-3">
          {filtered.length === 0 ? (
            <EmptyState
              type="no-data"
              title="No events match"
              description={events.length === 0
                ? 'No audit logs have been recorded yet. Events appear as MEL processes mesh data.'
                : `No ${levelFilter}-level events in current data.`
              }
            />
          ) : (
            <div className="space-y-1">
              {filtered.map((event, i) => (
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
  const level = event.level?.toLowerCase() || 'info'

  const dotClass = {
    error: 'bg-critical',
    warning: 'bg-warning',
    info: 'bg-info',
    debug: 'bg-muted-foreground/40',
  }[level] || 'bg-muted-foreground/40'

  const borderClass = {
    error: 'border-l-critical/60',
    warning: 'border-l-warning/60',
    info: 'border-l-info/40',
    debug: 'border-l-border/40',
  }[level] || 'border-l-border/40'

  return (
    <div className={clsx('flex items-start gap-3 rounded-lg border-l-2 px-3 py-2.5 transition-colors hover:bg-accent/40', borderClass)}>
      <div className="mt-1.5 flex items-center gap-2">
        <span className={clsx('h-1.5 w-1.5 rounded-full', dotClass)} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <p className="text-[13px] text-foreground">{event.message}</p>
        </div>
        <div className="mt-0.5 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground/70">
          <span>{event.category || 'system'}</span>
          <span>&middot;</span>
          <span>{event.level || 'info'}</span>
        </div>
      </div>
      <div className="ml-2 flex shrink-0 items-center gap-1 text-[11px] text-muted-foreground/60">
        <Clock className="h-3 w-3" />
        {formatRelativeTime(event.created_at)}
      </div>
    </div>
  )
}
