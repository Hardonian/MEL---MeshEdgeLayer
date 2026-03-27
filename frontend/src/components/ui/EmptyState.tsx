import { ReactNode } from 'react'
import { clsx } from 'clsx'
import { Inbox, Info, AlertCircle, Search, Wifi, Settings } from 'lucide-react'

export type EmptyStateType = 'default' | 'no-data' | 'not-found' | 'disconnected' | 'not-configured' | 'error'

interface EmptyStateProps {
  type?: EmptyStateType
  title: string
  description?: string
  action?: ReactNode
  details?: ReactNode
  className?: string
}

const typeConfig: Record<EmptyStateType, { icon: ReactNode; iconClass?: string }> = {
  default: {
    icon: <Inbox className="h-10 w-10" />,
    iconClass: 'text-primary',
  },
  'no-data': {
    icon: <Inbox className="h-10 w-10" />,
    iconClass: 'text-primary',
  },
  'not-found': {
    icon: <Search className="h-10 w-10" />,
    iconClass: 'text-info',
  },
  disconnected: {
    icon: <Wifi className="h-10 w-10" />,
    iconClass: 'text-muted-foreground',
  },
  'not-configured': {
    icon: <Settings className="h-10 w-10" />,
    iconClass: 'text-muted-foreground',
  },
  error: {
    icon: <AlertCircle className="h-10 w-10" />,
    iconClass: 'text-critical',
  },
}

export function EmptyState({
  type = 'default',
  title,
  description,
  action,
  details,
  className,
}: EmptyStateProps) {
  const config = typeConfig[type]

  return (
    <div
      className={clsx(
        'surface-panel surface-panel-muted flex flex-col items-center justify-center gap-4 rounded-[1.1rem] border-dashed p-8 text-center sm:p-10',
        className
      )}
    >
      <div
        className={clsx(
          'flex h-16 w-16 items-center justify-center rounded-[1.25rem] border border-border/70 bg-card/70 shadow-inset',
          config.iconClass
        )}
      >
        {config.icon}
      </div>
      <div className="max-w-md space-y-1.5">
        <h3 className="font-outfit text-lg font-semibold tracking-[-0.02em] text-foreground">{title}</h3>
        {description && (
          <p className="text-sm leading-relaxed text-muted-foreground">{description}</p>
        )}
        {details && <div className="pt-1">{details}</div>}
      </div>
      {action && <div className="flex items-center gap-2">{action}</div>}
    </div>
  )
}

export function NoTransportsConfigured({
  onConfigure,
}: {
  onConfigure?: () => void
}) {
  return (
    <EmptyState
      type="not-configured"
      title="No transports configured"
      description="Add transport configuration to your MEL config file to start receiving mesh data."
      action={
        onConfigure ? (
          <button
            onClick={onConfigure}
            className="button-primary"
          >
            Configure transports
          </button>
        ) : undefined
      }
      details={
        <div className="space-y-1 text-xs text-muted-foreground">
          <p>Supported transports: MQTT, TCP, Serial</p>
          <p>See the Configuration Guide for setup details.</p>
        </div>
      }
    />
  )
}

export function NoNodesYet() {
  return (
    <EmptyState
      type="no-data"
      title="No nodes observed yet"
      description="Node inventory is empty because no live mesh observations have been stored yet. This is expected when transports are idle or disconnected."
    />
  )
}

export function NoMessagesYet() {
  return (
    <EmptyState
      type="no-data"
      title="No messages yet"
      description="No live message observations have been stored yet. This is expected when transports are idle or disconnected."
    />
  )
}

export function SystemHealthy({ message = 'All systems operational' }: { message?: string }) {
  return (
    <div className="surface-inset flex items-start gap-3 rounded-[1rem] border-success/20 bg-success/10 p-4">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-success/16 bg-success/12 shadow-inset">
        <Info className="h-5 w-5 text-success" aria-hidden />
      </div>
      <div className="min-w-0">
        <p className="font-semibold text-success">{message}</p>
        <p className="text-sm leading-relaxed text-muted-foreground">Nothing to act on in this panel right now.</p>
      </div>
    </div>
  )
}
