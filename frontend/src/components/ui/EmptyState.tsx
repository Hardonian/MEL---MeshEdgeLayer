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

const typeConfig: Record<EmptyStateType, { icon: ReactNode; prefix: string }> = {
  default: { icon: <Inbox className="h-5 w-5" />, prefix: 'EMPTY' },
  'no-data': { icon: <Inbox className="h-5 w-5" />, prefix: 'NO_DATA' },
  'not-found': { icon: <Search className="h-5 w-5" />, prefix: '404' },
  disconnected: { icon: <Wifi className="h-5 w-5" />, prefix: 'OFFLINE' },
  'not-configured': { icon: <Settings className="h-5 w-5" />, prefix: 'UNCONFIGURED' },
  error: { icon: <AlertCircle className="h-5 w-5" />, prefix: 'ERROR' },
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
        'surface-panel surface-panel-muted flex flex-col items-center justify-center gap-3 border-dashed p-6 text-center',
        className
      )}
    >
      <div className="text-muted-foreground/40">{config.icon}</div>
      <div className="max-w-md space-y-1">
        <p className="text-mel-xs font-bold text-muted-foreground/50">[{config.prefix}]</p>
        <h3 className="text-mel-sm font-bold uppercase text-foreground">{title}</h3>
        {description && (
          <p className="text-mel-xs text-muted-foreground">{description}</p>
        )}
        {details && <div className="pt-1">{details}</div>}
      </div>
      {action && <div className="flex items-center gap-2">{action}</div>}
    </div>
  )
}

export function NoTransportsConfigured({ onConfigure }: { onConfigure?: () => void }) {
  return (
    <EmptyState
      type="not-configured"
      title="No transport configured"
      description="Add a supported transport in config before expecting live ingest evidence."
      action={
        onConfigure ? (
          <button onClick={onConfigure} className="button-primary">Configure transport</button>
        ) : undefined
      }
      details={
        <div className="space-y-0.5 text-mel-xs text-muted-foreground/60">
          <p># supported: MQTT, TCP, Serial</p>
          <p># review: Configuration Guide</p>
        </div>
      }
    />
  )
}

export function NoNodesYet() {
  return (
    <EmptyState
      type="no-data"
      title="No nodes observed"
      description="Node inventory empty — no live mesh observations stored. Expected when transports idle."
    />
  )
}

export function NoMessagesYet() {
  return (
    <EmptyState
      type="no-data"
      title="No messages"
      description="No message observations stored. Expected when transports idle or disconnected."
    />
  )
}

export function SystemHealthy({ message = 'No active findings' }: { message?: string }) {
  return (
    <div className="surface-inset flex items-start gap-3 border-neon/15 bg-neon/4 p-3">
      <div className="flex h-6 w-6 shrink-0 items-center justify-center border border-neon/20 text-neon">
        <Info className="h-3.5 w-3.5" aria-hidden />
      </div>
      <div className="min-w-0">
        <p className="font-mono text-mel-sm font-bold text-neon">[OK] {message}</p>
        <p className="text-mel-xs text-muted-foreground">No action required from current evidence.</p>
      </div>
    </div>
  )
}
