import { ReactNode } from 'react'
import { clsx } from 'clsx'

type BadgeVariant = 'default' | 'success' | 'warning' | 'critical' | 'secondary' | 'outline'

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

export function HealthBadge({ health }: { health: HealthVariant }) {
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

  return <Badge variant={variant}>{label}</Badge>
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

  return (
    <Badge variant={variantMap[severity]}>
      {severity.toUpperCase()}
    </Badge>
  )
}
