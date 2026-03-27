import { ReactNode } from 'react'
import { clsx } from 'clsx'

type BadgeVariant = 'default' | 'success' | 'warning' | 'critical' | 'secondary' | 'outline' | 'info'

interface BadgeProps {
  children: ReactNode
  variant?: BadgeVariant
  className?: string
}

const variantStyles: Record<BadgeVariant, string> = {
  default: 'border-primary/18 bg-primary/10 text-primary',
  success: 'border-success/20 bg-success/10 text-success',
  warning: 'border-warning/24 bg-warning/10 text-warning',
  critical: 'border-critical/24 bg-critical/10 text-critical',
  secondary: 'border-border/70 bg-muted/70 text-muted-foreground',
  outline: 'border-border/75 bg-card/65 text-foreground',
  info: 'border-info/20 bg-info/10 text-info',
}

function BadgeDot({ className }: { className?: string }) {
  return <span aria-hidden className={clsx('h-1.5 w-1.5 rounded-full', className)} />
}

export function Badge({ children, variant = 'default', className }: BadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex min-h-6 items-center gap-1.5 rounded-full border px-2.5 py-1 text-[10.5px] font-semibold leading-none shadow-inset transition-colors sm:text-[11px]',
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
    <Badge variant={variantMap[health]} className="uppercase tracking-[0.16em]">
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
    <Badge variant={variantMap[severity]} className="uppercase tracking-[0.16em]">
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
    <Badge variant={variantMap[state]} className="uppercase tracking-[0.16em]">
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
    <Badge variant={variantMap[priority]} className="uppercase tracking-[0.16em]">
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
    <Badge variant={variant} className="uppercase tracking-[0.16em]">
      {type.toUpperCase()}
    </Badge>
  )
}
