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
    icon: 'border-primary/16 bg-primary/12 text-primary',
    accent: 'from-primary/18 via-primary/5 to-transparent',
    trend: 'text-muted-foreground',
  },
  success: {
    icon: 'border-success/18 bg-success/12 text-success',
    accent: 'from-success/20 via-success/6 to-transparent',
    trend: 'text-success',
  },
  warning: {
    icon: 'border-warning/22 bg-warning/12 text-warning',
    accent: 'from-warning/20 via-warning/6 to-transparent',
    trend: 'text-warning',
  },
  critical: {
    icon: 'border-critical/22 bg-critical/12 text-critical',
    accent: 'from-critical/22 via-critical/6 to-transparent',
    trend: 'text-critical',
  },
  info: {
    icon: 'border-info/22 bg-info/12 text-info',
    accent: 'from-info/22 via-info/6 to-transparent',
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
        'surface-panel interactive-lift group overflow-hidden p-4 sm:p-5',
        className
      )}
    >
      <div className={clsx('absolute inset-x-0 top-0 h-px bg-gradient-to-r', variantStyle.accent)} aria-hidden />
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <p className="font-mono text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
            {title}
          </p>
          <p className="mt-3 font-mono text-[1.9rem] font-semibold tracking-[-0.04em] text-foreground sm:text-[2.15rem]">
            {value}
          </p>
          {description && (
            <p className="mt-1.5 max-w-[24ch] text-sm leading-relaxed text-muted-foreground">
              {description}
            </p>
          )}
          {trend && (
            <p className={clsx('mt-3 text-xs font-semibold uppercase tracking-[0.16em]', variantStyle.trend)}>
              {trend.value > 0 ? '+' : ''}
              {trend.value}% {trend.label}
            </p>
          )}
        </div>
        {icon && (
          <div
            className={clsx(
              'flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border shadow-inset',
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
