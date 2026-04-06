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
    icon: 'border-neon/20 text-neon',
    value: 'text-neon',
    accent: 'bg-neon',
  },
  success: {
    icon: 'border-neon/20 text-neon',
    value: 'text-neon',
    accent: 'bg-neon',
  },
  warning: {
    icon: 'border-neon-warn/20 text-neon-warn',
    value: 'text-neon-warn',
    accent: 'bg-neon-warn',
  },
  critical: {
    icon: 'border-neon-hot/20 text-neon-hot',
    value: 'text-neon-hot',
    accent: 'bg-neon-hot',
  },
  info: {
    icon: 'border-neon-alt/20 text-neon-alt',
    value: 'text-neon-alt',
    accent: 'bg-neon-alt',
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
    <div className={clsx('surface-panel interactive-lift overflow-hidden p-3', className)}>
      {/* Neon accent top line */}
      <div className={clsx('absolute inset-x-0 top-0 h-px', v.accent, 'opacity-40')} aria-hidden />

      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="mel-label">{title}</p>
          <p className={clsx('mel-metric mt-1.5 font-data text-mel-metric', v.value)}>{value}</p>
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
