import { ReactNode } from 'react'
import { clsx } from 'clsx'

type BadgeVariant = 'default' | 'success' | 'warning' | 'critical' | 'secondary' | 'outline' | 'info'

interface BadgeProps {
  children: ReactNode
  variant?: BadgeVariant
  className?: string
}

const variantStyles: Record<BadgeVariant, string> = {
  default: 'border-transparent bg-primary text-primary-foreground',
  success: 'border-success/20 bg-success/10 text-success',
  warning: 'border-warning/20 bg-warning/10 text-warning',
  critical: 'border-critical/20 bg-critical/10 text-critical',
  secondary: 'border-transparent bg-secondary text-secondary-foreground',
  outline: 'border-border text-foreground',
  info: 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-300',
}

export function Badge({ children, variant = 'default', className }: BadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors',
        variantStyles[variant],
        className
      )}
    >
      {children}
    </span>
  )
}

// Health-specific badge
type HealthVariant = 'healthy' | 'degraded' | 'unhealthy' | 'unknown'

export function HealthBadge({ health, showLabel = true }: { health: HealthVariant; showLabel?: boolean }) {
  const variantMap: Record<HealthVariant, BadgeVariant> = {
    healthy: 'success',
    degraded: 'warning',
    unhealthy: 'critical',
    unknown: 'secondary',
  }
  const variant = variantMap[health]

  const label = {
    healthy: 'Healthy',
    degraded: 'Degraded',
    unhealthy: 'Critical',
    unknown: 'Unknown',
  }[health]

  const icon = {
    healthy: '●',
    degraded: '●',
    unhealthy: '●',
    unknown: '○',
  }[health]

  return (
    <Badge variant={variant}>
      {showLabel && <span className="mr-1.5">{icon}</span>}
      {label}
    </Badge>
  )
}

// Severity badge for privacy findings
type SeverityVariant = 'critical' | 'high' | 'medium' | 'low'

export function SeverityBadge({ severity }: { severity: SeverityVariant }) {
  const variantMap: Record<SeverityVariant, BadgeVariant> = {
    critical: 'critical',
    high: 'warning',
    medium: 'warning',
    low: 'secondary',
  }

  const label = {
    critical: 'CRITICAL',
    high: 'HIGH',
    medium: 'MEDIUM',
    low: 'LOW',
  }[severity]

  return (
    <Badge variant={variantMap[severity]}>
      {severity === 'critical' && <span className="mr-1">●</span>}
      {label}
    </Badge>
  )
}

// Connection state badge
type ConnectionState = 'connected' | 'disconnected' | 'connecting' | 'error'

export function ConnectionBadge({ state }: { state: ConnectionState }) {
  const variantMap: Record<ConnectionState, BadgeVariant> = {
    connected: 'success',
    disconnected: 'secondary',
    connecting: 'info',
    error: 'critical',
  }

  const icon = {
    connected: '●',
    disconnected: '○',
    connecting: '◐',
    error: '●',
  }[state]

  const label = {
    connected: 'Connected',
    disconnected: 'Disconnected',
    connecting: 'Connecting',
    error: 'Error',
  }[state]

  return (
    <Badge variant={variantMap[state]}>
      <span className="mr-1.5">{icon}</span>
      {label}
    </Badge>
  )
}

// Priority badge
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

// Transport type badge
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
