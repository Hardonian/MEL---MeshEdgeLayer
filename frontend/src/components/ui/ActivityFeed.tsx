import { Link } from 'react-router-dom'
import { clsx } from 'clsx'
import {
  AlertTriangle,
  Zap,
  FileText,
  Shield,
  Activity,
  ArrowRight,
  Clock,
  Radio,
  Inbox,
} from 'lucide-react'
import { formatRelativeTime, type AuditLog } from '@/types/api'
import { Badge } from '@/components/ui/Badge'

export interface FeedItem {
  id: string
  type: 'event' | 'incident' | 'action' | 'recommendation' | 'dead_letter' | 'privacy' | 'transport'
  level: 'critical' | 'warning' | 'info' | 'neutral'
  title: string
  detail?: string
  timestamp: string
  href?: string
  category?: string
}

const typeConfig: Record<
  FeedItem['type'],
  { icon: React.ElementType; label: string }
> = {
  event: { icon: FileText, label: 'Event' },
  incident: { icon: AlertTriangle, label: 'Incident' },
  action: { icon: Zap, label: 'Action' },
  recommendation: { icon: Activity, label: 'Rec' },
  dead_letter: { icon: Inbox, label: 'Dead letter' },
  privacy: { icon: Shield, label: 'Privacy' },
  transport: { icon: Radio, label: 'Transport' },
}

const levelStyles: Record<FeedItem['level'], string> = {
  critical: 'border-l-critical/60',
  warning: 'border-l-warning/60',
  info: 'border-l-info/50',
  neutral: 'border-l-border/50',
}

const levelDotStyles: Record<FeedItem['level'], string> = {
  critical: 'bg-critical',
  warning: 'bg-warning',
  info: 'bg-info',
  neutral: 'bg-muted-foreground/40',
}

interface ActivityFeedProps {
  items: FeedItem[]
  maxItems?: number
  showViewAll?: boolean
  viewAllHref?: string
  emptyMessage?: string
  className?: string
}

export function ActivityFeed({
  items,
  maxItems = 12,
  showViewAll = true,
  viewAllHref = '/events',
  emptyMessage = 'No recent activity. The system is quiet.',
  className,
}: ActivityFeedProps) {
  const sorted = [...items]
    .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
    .slice(0, maxItems)

  if (sorted.length === 0) {
    return (
      <div className={clsx('flex flex-col items-center gap-2 py-8 text-center', className)}>
        <div className="flex h-10 w-10 items-center justify-center rounded-2xl border border-success/20 bg-success/8 text-success">
          <Activity className="h-5 w-5" />
        </div>
        <p className="text-sm text-muted-foreground">{emptyMessage}</p>
        <p className="text-xs text-muted-foreground/60">
          Activity appears here when MEL observes changes, incidents, or actions.
        </p>
      </div>
    )
  }

  return (
    <div className={clsx('space-y-1', className)}>
      {sorted.map((item) => {
        const config = typeConfig[item.type]
        const Icon = config.icon
        const inner = (
          <div
            className={clsx(
              'group flex items-start gap-3 rounded-lg border-l-2 px-3 py-2.5 transition-colors',
              levelStyles[item.level],
              item.href
                ? 'cursor-pointer hover:bg-accent/50'
                : 'bg-transparent'
            )}
          >
            <div className="mt-0.5 flex items-center gap-2">
              <span className={clsx('h-1.5 w-1.5 rounded-full', levelDotStyles[item.level])} />
              <Icon className="h-3.5 w-3.5 text-muted-foreground" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-baseline gap-2">
                <p className="truncate text-[13px] font-medium text-foreground">
                  {item.title}
                </p>
                {item.category && (
                  <Badge variant="outline" className="hidden shrink-0 sm:inline-flex">
                    {item.category}
                  </Badge>
                )}
              </div>
              {item.detail && (
                <p className="mt-0.5 truncate text-xs text-muted-foreground">
                  {item.detail}
                </p>
              )}
            </div>
            <div className="ml-2 flex shrink-0 items-center gap-1 text-[11px] text-muted-foreground/70">
              <Clock className="h-3 w-3" />
              {formatRelativeTime(item.timestamp)}
            </div>
          </div>
        )

        return item.href ? (
          <Link key={item.id} to={item.href} className="block outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-lg">
            {inner}
          </Link>
        ) : (
          <div key={item.id}>{inner}</div>
        )
      })}

      {showViewAll && items.length > maxItems && (
        <Link
          to={viewAllHref}
          className="mt-2 flex items-center gap-1 px-3 py-1.5 text-xs font-semibold text-muted-foreground transition-colors hover:text-foreground"
        >
          View all activity <ArrowRight className="h-3 w-3" />
        </Link>
      )}
    </div>
  )
}

/** Build FeedItem[] from audit logs */
export function eventsToFeedItems(events: AuditLog[]): FeedItem[] {
  return events.map((e, i) => ({
    id: `evt-${i}-${e.created_at}`,
    type: 'event' as const,
    level:
      e.level === 'error' ? 'critical' as const
        : e.level === 'warning' ? 'warning' as const
          : e.level === 'info' ? 'info' as const
            : 'neutral' as const,
    title: e.message,
    category: e.category,
    timestamp: e.created_at,
    href: '/events',
  }))
}
