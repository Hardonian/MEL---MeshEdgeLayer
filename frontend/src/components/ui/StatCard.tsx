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
    icon: 'bg-primary/10 text-primary',
    trend: 'text-muted-foreground',
  },
  success: {
    icon: 'bg-success/10 text-success',
    trend: 'text-success',
  },
  warning: {
    icon: 'bg-warning/10 text-warning',
    trend: 'text-warning',
  },
  critical: {
    icon: 'bg-critical/10 text-critical',
    trend: 'text-critical',
  },
  info: {
    icon: 'bg-info/10 text-info',
    trend: 'text-info',
  },
}

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
        'rounded-xl border border-border/80 bg-card p-4 transition-shadow duration-200 hover:shadow-sm sm:p-5',
        className
      )}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          <p className="mt-1.5 text-2xl font-bold tracking-tight">{value}</p>
          {description && (
            <p className="mt-1 text-xs text-muted-foreground">{description}</p>
          )}
          {trend && (
            <p className={clsx('mt-2 text-xs font-medium', variantStyle.trend)}>
              {trend.value > 0 ? '+' : ''}{trend.value}% {trend.label}
            </p>
          )}
        </div>
        {icon && (
          <div
            className={clsx(
              'flex h-10 w-10 items-center justify-center rounded-lg',
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
