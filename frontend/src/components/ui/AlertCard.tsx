import { ReactNode } from 'react'
import { clsx } from 'clsx'
import { AlertTriangle, Info, CheckCircle2, XCircle } from 'lucide-react'

type AlertVariant = 'info' | 'warning' | 'success' | 'error' | 'critical'

interface AlertCardProps {
  title?: string
  description?: string
  variant?: AlertVariant
  icon?: ReactNode
  action?: ReactNode
  className?: string
  children?: ReactNode
}

const variantStyles: Record<AlertVariant, { container: string; icon: string; rail: string }> = {
  info: {
    container: 'border-info/20 bg-info/8',
    icon: 'border-info/16 bg-info/12 text-info',
    rail: 'from-info/40 via-info/16 to-transparent',
  },
  warning: {
    container: 'border-warning/24 bg-warning/10',
    icon: 'border-warning/18 bg-warning/12 text-warning',
    rail: 'from-warning/46 via-warning/16 to-transparent',
  },
  success: {
    container: 'border-success/24 bg-success/10',
    icon: 'border-success/18 bg-success/12 text-success',
    rail: 'from-success/42 via-success/16 to-transparent',
  },
  error: {
    container: 'border-critical/24 bg-critical/10',
    icon: 'border-critical/18 bg-critical/12 text-critical',
    rail: 'from-critical/42 via-critical/16 to-transparent',
  },
  critical: {
    container: 'border-critical/28 bg-critical/12',
    icon: 'border-critical/20 bg-critical/14 text-critical',
    rail: 'from-critical/52 via-critical/20 to-transparent',
  },
}

const defaultIcons: Record<AlertVariant, ReactNode> = {
  info: <Info className="h-5 w-5" />,
  warning: <AlertTriangle className="h-5 w-5" />,
  success: <CheckCircle2 className="h-5 w-5" />,
  error: <XCircle className="h-5 w-5" />,
  critical: <AlertTriangle className="h-5 w-5" />,
}

export function AlertCard({
  title,
  description,
  variant = 'info',
  icon,
  action,
  className,
  children,
}: AlertCardProps) {
  const styles = variantStyles[variant]

  return (
    <div
      className={clsx(
        'surface-panel relative overflow-hidden rounded-[1.1rem] p-4 sm:p-5',
        styles.container,
        className
      )}
    >
      <div className={clsx('absolute inset-y-0 left-0 w-1 bg-gradient-to-b', styles.rail)} aria-hidden />
      <div className="flex items-start gap-3">
        <div className={clsx('mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border shadow-inset', styles.icon)}>
          {icon || defaultIcons[variant]}
        </div>
        <div className="min-w-0 flex-1">
          {title && (
            <h4 className="text-sm font-semibold text-foreground">{title}</h4>
          )}
          {(description || children) && (
            <div className="mt-1.5 space-y-2">
              {description && (
                <p className="text-sm leading-relaxed text-muted-foreground">{description}</p>
              )}
              {children}
            </div>
          )}
        </div>
        {action && <div className="shrink-0">{action}</div>}
      </div>
    </div>
  )
}

interface InlineAlertProps {
  variant?: AlertVariant
  children: ReactNode
  className?: string
}

export function InlineAlert({ variant = 'info', children, className }: InlineAlertProps) {
  const styles = variantStyles[variant]

  return (
    <div
      className={clsx(
        'surface-inset flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm',
        styles.container,
        className
      )}
    >
      <span className={clsx('flex h-8 w-8 shrink-0 items-center justify-center rounded-xl border shadow-inset', styles.icon)}>
        {defaultIcons[variant]}
      </span>
      <span className="min-w-0 flex-1">{children}</span>
    </div>
  )
}
