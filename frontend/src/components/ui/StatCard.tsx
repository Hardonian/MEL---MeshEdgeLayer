import { ReactNode } from 'react'
import { clsx } from 'clsx'

interface StatCardProps {
  title: string
  value: string | number
  description?: string
  icon?: ReactNode
  trend?: {
    value: number
    label: string
  }
  variant?: 'default' | 'success' | 'warning' | 'critical' | 'info'
  className?: string
}

const variantStyles = {
  default: {
    icon: 'border-primary/20 bg-primary/10 text-primary',
    accent: 'bg-primary',
    trend: 'text-muted-foreground',
  },
  success: {
    icon: 'border-success/20 bg-success/10 text-success',
    accent: 'bg-success',
    trend: 'text-success',
  },
  warning: {
    icon: 'border-warning/25 bg-warning/10 text-warning',
    accent: 'bg-warning',
    trend: 'text-warning',
  },
  critical: {
    icon: 'border-critical/25 bg-critical/10 text-critical',
    accent: 'bg-critical',
    trend: 'text-critical',
  },
  info: {
    icon: 'border-info/25 bg-info/10 text-info',
    accent: 'bg-info',
    trend: 'text-info',
  },
} as const

export function StatCard({
  title,
  value,
  description,
  icon,
  trend,
  variant = 'default',
  className,
}: StatCardProps) {
  const variantStyle = variantStyles[variant]

  return (
    <div
      className={clsx(
        'surface-panel interactive-lift group overflow-hidden p-4',
        className
      )}
    >
      {/* Top accent line */}
      <div className={clsx('absolute inset-x-0 top-0 h-px', variantStyle.accent, 'opacity-30')} aria-hidden />

      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="mel-label">{title}</p>
          <p className="mel-metric mt-2 text-mel-metric text-foreground">{value}</p>
          {description && (
            <p className="mt-1 prose-body text-mel-xs text-muted-foreground">{description}</p>
          )}
          {trend && (
            <p className={clsx('mt-2 font-mono text-mel-xs font-semibold', variantStyle.trend)}>
              {trend.value > 0 ? '+' : ''}{trend.value}% {trend.label}
            </p>
          )}
        </div>
        {icon && (
          <div
            className={clsx(
              'flex h-9 w-9 shrink-0 items-center justify-center rounded-md border',
              variantStyle.icon
            )}
          >
            {icon}
          </div>
        )}
      </div>
    </div>
  )
}
