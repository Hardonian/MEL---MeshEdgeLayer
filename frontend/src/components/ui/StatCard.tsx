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
    icon: 'border-border/60 text-foreground',
    value: 'text-foreground',
  },
  success: {
    icon: 'border-signal-live/25 text-signal-live',
    value: 'text-signal-live',
  },
  warning: {
    icon: 'border-signal-degraded/25 text-signal-degraded',
    value: 'text-signal-degraded',
  },
  critical: {
    icon: 'border-signal-critical/25 text-signal-critical',
    value: 'text-signal-critical',
  },
  info: {
    icon: 'border-signal-observed/25 text-signal-observed',
    value: 'text-signal-observed',
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
  const v = variantStyles[variant]

  return (
    <div className={clsx('surface-panel interactive-lift relative overflow-hidden p-3', className)}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="mel-label">{title}</p>
          <p className={clsx('mt-1.5 font-data text-lg font-bold tabular-nums tracking-tight', v.value)}>{value}</p>
          {description && (
            <p className="mt-1 text-mel-xs text-muted-foreground">{description}</p>
          )}
          {trend && (
            <p className={clsx('mt-1.5 text-mel-xs font-bold', v.value)}>
              {trend.value > 0 ? '+' : ''}{trend.value}% {trend.label}
            </p>
          )}
        </div>
        {icon && (
          <div className={clsx('flex h-7 w-7 shrink-0 items-center justify-center border', v.icon)}>
            {icon}
          </div>
        )}
      </div>
    </div>
  )
}
