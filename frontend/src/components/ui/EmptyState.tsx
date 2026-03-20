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
    icon: <Inbox className="h-12 w-12" />,
  },
  'no-data': {
    icon: <Inbox className="h-12 w-12" />,
  },
  'not-found': {
    icon: <Search className="h-12 w-12" />,
  },
  disconnected: {
    icon: <Wifi className="h-12 w-12" />,
    iconClass: 'text-muted-foreground',
  },
  'not-configured': {
    icon: <Settings className="h-12 w-12" />,
    iconClass: 'text-muted-foreground',
  },
  error: {
    icon: <AlertCircle className="h-12 w-12" />,
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
        'flex flex-col items-center justify-center gap-4 rounded-xl border border-dashed p-8 text-center',
        className
      )}
    >
      <div
        className={clsx(
          'flex h-16 w-16 items-center justify-center rounded-full bg-muted/50',
          config.iconClass
        )}
      >
        {config.icon}
      </div>
      <div className="max-w-md space-y-1">
        <h3 className="text-base font-semibold text-foreground">{title}</h3>
        {description && (
          <p className="text-sm text-muted-foreground">{description}</p>
        )}
        {details && <div className="mt-2">{details}</div>}
      </div>
      {action && <div className="flex items-center gap-2">{action}</div>}
    </div>
  )
}

// Specialized empty states for common scenarios
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
            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            Configure transports
          </button>
        ) : undefined
      }
      details={
        <div className="text-xs text-muted-foreground">
          <p>Supported transports: MQTT, TCP, Serial, HTTP</p>
          <p className="mt-1">See the Configuration Guide for help.</p>
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

export function SystemHealthy({ message = "All systems operational" }: { message?: string }) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-success/20 bg-success/5 p-4">
      <div className="flex h-10 w-10 items-center justify-center rounded-full bg-success/10">
        <Info className="h-5 w-5 text-success" />
      </div>
      <div>
        <p className="font-medium text-success">{message}</p>
        <p className="text-sm text-muted-foreground">No action required at this time.</p>
      </div>
    </div>
  )
}
