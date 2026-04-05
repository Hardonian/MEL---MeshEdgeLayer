import { ReactNode } from 'react'
import { clsx } from 'clsx'
import { Inbox, Info, AlertCircle, Search, Wifi, Settings, Terminal } from 'lucide-react'

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
    icon: <Inbox className="h-8 w-8" />,
    iconClass: 'text-primary',
  },
  'no-data': {
    icon: <Terminal className="h-8 w-8" />,
    iconClass: 'text-primary',
  },
  'not-found': {
    icon: <Search className="h-8 w-8" />,
    iconClass: 'text-info',
  },
  disconnected: {
    icon: <Wifi className="h-8 w-8" />,
    iconClass: 'text-muted-foreground',
  },
  'not-configured': {
    icon: <Settings className="h-8 w-8" />,
    iconClass: 'text-muted-foreground',
  },
  error: {
    icon: <AlertCircle className="h-8 w-8" />,
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
        'surface-panel surface-panel-muted flex flex-col items-center justify-center gap-3 border-dashed p-6 text-center sm:p-8',
        className
      )}
    >
      <div
        className={clsx(
          'flex h-12 w-12 items-center justify-center rounded-md border border-border/60 bg-card/60',
          config.iconClass
        )}
      >
        {config.icon}
      </div>
      <div className="max-w-md space-y-1">
        <h3 className="font-display text-sm font-bold tracking-tight text-foreground">{title}</h3>
        {description && (
          <p className="prose-body text-mel-sm text-muted-foreground">{description}</p>
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
          <button onClick={onConfigure} className="button-primary">
            Configure transports
          </button>
        ) : undefined
      }
      details={
        <div className="space-y-0.5 font-mono text-mel-xs text-muted-foreground">
          <p>Supported: MQTT, TCP, Serial</p>
          <p>See Configuration Guide for setup.</p>
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
      description="Node inventory is empty — no live mesh observations stored. Expected when transports are idle or disconnected."
    />
  )
}

export function NoMessagesYet() {
  return (
    <EmptyState
      type="no-data"
      title="No messages yet"
      description="No live message observations stored. Expected when transports are idle or disconnected."
    />
  )
}

export function SystemHealthy({ message = 'All systems operational' }: { message?: string }) {
  return (
    <div className="surface-inset flex items-start gap-3 border-success/20 bg-success/6 p-3">
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-success/20 bg-success/10">
        <Info className="h-4 w-4 text-success" aria-hidden />
      </div>
      <div className="min-w-0">
        <p className="font-mono text-mel-sm font-semibold text-success">{message}</p>
        <p className="prose-body text-mel-xs text-muted-foreground">Nothing to act on in this panel.</p>
      </div>
    </div>
  )
}
