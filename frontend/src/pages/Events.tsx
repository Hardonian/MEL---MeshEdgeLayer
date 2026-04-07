import { useState, useMemo, useCallback } from 'react'
import { useEvents } from '@/hooks/useApi'
import { CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { MelPanel, MelSegment, MelSegmentItem } from '@/components/ui/operator'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatRelativeTime, AuditLog } from '@/types/api'
import { ScrollText, AlertTriangle, AlertCircle, FileText, Clock, RefreshCw } from 'lucide-react'
import { clsx } from 'clsx'

const LEVEL_FILTERS = ['all', 'error', 'warning', 'info', 'debug'] as const
type LevelFilter = (typeof LEVEL_FILTERS)[number]

export function Events() {
  const { data, loading, error, refresh } = useEvents()
  const [levelFilter, setLevelFilter] = useState<LevelFilter>('all')

  const events = useMemo(() => data ?? [], [data])

  const errorCount = useMemo(
    () => events.filter((e) => e.level?.toLowerCase() === 'error').length,
    [events],
  )
  const warningCount = useMemo(
    () => events.filter((e) => e.level?.toLowerCase() === 'warning').length,
    [events],
  )
  const categories = useMemo(
    () => [...new Set(events.map((e) => e.category).filter(Boolean))],
    [events],
  )

  const levelMatches = useCallback(
    (e: (typeof events)[number], level: LevelFilter) =>
      level === 'all' || e.level?.toLowerCase() === level,
    [],
  )

  const filtered = useMemo(() => {
    if (levelFilter === 'all') return events
    return events.filter((e) => levelMatches(e, levelFilter))
  }, [events, levelFilter, levelMatches])

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
          description="Audit log for this MEL instance. Entries reflect persisted records, not proof of live mesh or transport state."
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
          icon={<ScrollText className="h-4 w-4" />}
          variant="default"
          rhythm="console"
        />
        <StatCard
          title="Errors"
          value={errorCount}
          description="Error-level events"
          icon={<AlertCircle className="h-4 w-4" />}
          variant={errorCount > 0 ? 'critical' : 'success'}
          rhythm="console"
        />
        <StatCard
          title="Warnings"
          value={warningCount}
          description="Warning-level events"
          icon={<AlertTriangle className="h-4 w-4" />}
          variant={warningCount > 0 ? 'warning' : 'default'}
          rhythm="console"
        />
        <StatCard
          title="Categories"
          value={categories.length}
          description="Unique event sources"
          icon={<FileText className="h-4 w-4" />}
          variant="info"
          rhythm="console"
        />
      </div>

      <MelSegment label="Level" radiogroupLabel="Filter events by log level">
        {LEVEL_FILTERS.map((level) => (
          <MelSegmentItem
            key={level}
            role="radio"
            aria-checked={levelFilter === level}
            onClick={() => setLevelFilter(level)}
            active={levelFilter === level}
          >
            {level === 'all' ? `All (${events.length})` : `${level} (${events.filter((e) => e.level?.toLowerCase() === level).length})`}
          </MelSegmentItem>
        ))}
      </MelSegment>

      <MelPanel className="overflow-hidden">
        <CardHeader className="border-b border-border/50 pb-3 px-4 pt-4">
          <div className="flex items-center justify-between">
            <CardTitle className="text-[14px]">
              {levelFilter === 'all' ? 'All events' : `${levelFilter} events`}
            </CardTitle>
            <Badge variant="outline">{filtered.length} shown</Badge>
          </div>
        </CardHeader>
        <CardContent className="pt-3 px-4 pb-4">
          {filtered.length === 0 ? (
            <EmptyState
              type="no-data"
              title="No events match"
              description={events.length === 0
                ? 'No audit rows loaded for this view yet. Entries appear as the instance records activity you are allowed to read.'
                : `No ${levelFilter}-level events in the current filtered set.`
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
      </MelPanel>
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
    <div className={clsx('flex items-start gap-3 rounded-sm border-l-2 px-3 py-2.5 transition-colors hover:bg-accent/40', borderClass)}>
      <div className="mt-1.5 flex items-center gap-2">
        <span className={clsx('h-1.5 w-1.5 rounded-full', dotClass)} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <p className="text-[13px] text-foreground">{event.message}</p>
        </div>
        <div className="mt-0.5 flex flex-wrap items-center gap-2 text-mel-sm text-muted-foreground/70">
          <span>{event.category || 'system'}</span>
          <span>&middot;</span>
          <span>{event.level || 'info'}</span>
        </div>
      </div>
      <div className="ml-2 flex shrink-0 items-center gap-1 text-mel-sm text-muted-foreground/60">
        <Clock className="h-3 w-3" />
        {formatRelativeTime(event.created_at)}
      </div>
    </div>
  )
}
