import { ReactNode } from 'react'
import { clsx } from 'clsx'

type BadgeVariant = 'default' | 'success' | 'warning' | 'critical' | 'secondary' | 'outline' | 'info'

interface BadgeProps {
  children: ReactNode
  variant?: BadgeVariant
  className?: string
}

const variantStyles: Record<BadgeVariant, string> = {
  default: 'border-neon/30 text-neon',
  success: 'border-neon/30 text-neon',
  warning: 'border-neon-warn/30 text-neon-warn',
  critical: 'border-neon-hot/30 text-neon-hot',
  secondary: 'border-border text-muted-foreground',
  outline: 'border-border text-foreground',
  info: 'border-neon-alt/30 text-neon-alt',
}

function BadgeDot({ className }: { className?: string }) {
  return <span aria-hidden className={clsx('h-1.5 w-1.5 rounded-full', className)} style={{ boxShadow: '0 0 6px currentColor' }} />
}

export function Badge({ children, variant = 'default', className }: BadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex items-center gap-1 border px-1.5 py-0.5 text-[9px] font-bold uppercase leading-none tracking-[0.1em]',
        variantStyles[variant],
        className
      )}
    >
      [{children}]
    </span>
  )
}

type HealthVariant = 'healthy' | 'degraded' | 'unhealthy' | 'unknown'

export function HealthBadge({ health, showLabel = true }: { health: HealthVariant; showLabel?: boolean }) {
  const variantMap: Record<HealthVariant, BadgeVariant> = {
    healthy: 'success',
    degraded: 'warning',
    unhealthy: 'critical',
    unknown: 'secondary',
  }

  const label = {
    healthy: 'OK',
    degraded: 'DEGRADED',
    unhealthy: 'CRIT',
    unknown: '???',
  }[health]

  const dotClass = {
    healthy: 'bg-neon text-neon',
    degraded: 'bg-neon-warn text-neon-warn',
    unhealthy: 'bg-neon-hot text-neon-hot',
    unknown: 'bg-muted-foreground text-muted-foreground',
  }[health]

  return (
    <Badge variant={variantMap[health]}>
      {showLabel && <BadgeDot className={dotClass} />}
      {label}
    </Badge>
  )
}

type SeverityVariant = 'critical' | 'high' | 'medium' | 'low'

export function SeverityBadge({ severity }: { severity: SeverityVariant }) {
  const variantMap: Record<SeverityVariant, BadgeVariant> = {
    critical: 'critical',
    high: 'warning',
    medium: 'warning',
    low: 'secondary',
  }

  return (
    <Badge variant={variantMap[severity]}>
      {severity === 'critical' && <BadgeDot className="bg-neon-hot text-neon-hot" />}
      {severity.toUpperCase()}
    </Badge>
  )
}

type ConnectionState = 'connected' | 'disconnected' | 'connecting' | 'error'

export function ConnectionBadge({ state }: { state: ConnectionState }) {
  const variantMap: Record<ConnectionState, BadgeVariant> = {
    connected: 'success',
    disconnected: 'secondary',
    connecting: 'info',
    error: 'critical',
  }

  const dotClass = {
    connected: 'bg-neon text-neon',
    disconnected: 'bg-muted-foreground text-muted-foreground',
    connecting: 'bg-neon-alt text-neon-alt animate-pulse-slow',
    error: 'bg-neon-hot text-neon-hot',
  }[state]

  return (
    <Badge variant={variantMap[state]}>
      <BadgeDot className={dotClass} />
      {state.toUpperCase()}
    </Badge>
  )
}

type PriorityVariant = 'urgent' | 'high' | 'normal' | 'low'

export function PriorityBadge({ priority }: { priority: PriorityVariant }) {
  const variantMap: Record<PriorityVariant, BadgeVariant> = {
    urgent: 'critical',
    high: 'warning',
    normal: 'default',
    low: 'secondary',
  }

  return (
    <Badge variant={variantMap[priority]}>
      {priority === 'urgent' ? '!!' : ''}{priority.toUpperCase()}
    </Badge>
  )
}

export function TransportBadge({ type }: { type: string }) {
  const variantMap: Record<string, BadgeVariant> = {
    mqtt: 'info',
    tcp: 'default',
    serial: 'secondary',
    http: 'outline',
    bluetooth: 'default',
  }

  return (
    <Badge variant={variantMap[type.toLowerCase()] || 'outline'}>
      {type.toUpperCase()}
    </Badge>
  )
}
