import { ReactNode } from 'react'
import { clsx } from 'clsx'

type BadgeVariant = 'default' | 'success' | 'warning' | 'critical' | 'secondary' | 'outline' | 'info'

interface BadgeProps {
  children: ReactNode
  variant?: BadgeVariant
  className?: string
}

const variantStyles: Record<BadgeVariant, string> = {
  default: 'border-primary/20 bg-primary/8 text-primary',
  success: 'border-success/25 bg-success/8 text-success',
  warning: 'border-warning/25 bg-warning/8 text-warning',
  critical: 'border-critical/25 bg-critical/8 text-critical',
  secondary: 'border-border/60 bg-muted/50 text-muted-foreground',
  outline: 'border-border/60 bg-card/50 text-foreground',
  info: 'border-info/20 bg-info/8 text-info',
}

function BadgeDot({ className }: { className?: string }) {
  return <span aria-hidden className={clsx('h-1.5 w-1.5 rounded-full', className)} style={{ boxShadow: '0 0 4px currentColor' }} />
}

export function Badge({ children, variant = 'default', className }: BadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex min-h-5 items-center gap-1.5 rounded-sm border px-2 py-0.5 font-mono text-[10px] font-semibold uppercase leading-none',
        'tracking-[0.08em]',
        variantStyles[variant],
        className
      )}
    >
      {children}
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
    healthy: 'Healthy',
    degraded: 'Degraded',
    unhealthy: 'Critical',
    unknown: 'Unknown',
  }[health]

  const dotClass = {
    healthy: 'bg-success',
    degraded: 'bg-warning',
    unhealthy: 'bg-critical',
    unknown: 'bg-muted-foreground',
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

  const label = {
    critical: 'Critical',
    high: 'High',
    medium: 'Medium',
    low: 'Low',
  }[severity]

  return (
    <Badge variant={variantMap[severity]}>
      {severity === 'critical' && <BadgeDot className="bg-critical" />}
      {label}
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

  const label = {
    connected: 'Connected',
    disconnected: 'Disconnected',
    connecting: 'Connecting',
    error: 'Error',
  }[state]

  const dotClass = {
    connected: 'bg-success',
    disconnected: 'bg-muted-foreground',
    connecting: 'bg-info animate-pulse-slow',
    error: 'bg-critical',
  }[state]

  return (
    <Badge variant={variantMap[state]}>
      <BadgeDot className={dotClass} />
      {label}
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

  const label = {
    urgent: 'Urgent',
    high: 'High',
    normal: 'Normal',
    low: 'Low',
  }[priority]

  return (
    <Badge variant={variantMap[priority]}>
      {label}
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

  const variant = variantMap[type.toLowerCase()] || 'outline'

  return (
    <Badge variant={variant}>
      {type.toUpperCase()}
    </Badge>
  )
}
