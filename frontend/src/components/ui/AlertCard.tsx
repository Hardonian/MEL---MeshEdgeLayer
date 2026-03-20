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

const variantStyles: Record<AlertVariant, { container: string; icon: string; border: string }> = {
  info: {
    container: 'bg-blue-50 border-blue-200 dark:bg-blue-950 dark:border-blue-800',
    icon: 'text-blue-600 dark:text-blue-400',
    border: 'border-l-blue-500',
  },
  warning: {
    container: 'bg-amber-50 border-amber-200 dark:bg-amber-950 dark:border-amber-800',
    icon: 'text-amber-600 dark:text-amber-400',
    border: 'border-l-amber-500',
  },
  success: {
    container: 'bg-emerald-50 border-emerald-200 dark:bg-emerald-950 dark:border-emerald-800',
    icon: 'text-emerald-600 dark:text-emerald-400',
    border: 'border-l-emerald-500',
  },
  error: {
    container: 'bg-red-50 border-red-200 dark:bg-red-950 dark:border-red-800',
    icon: 'text-red-600 dark:text-red-400',
    border: 'border-l-red-500',
  },
  critical: {
    container: 'bg-red-50 border-red-300 dark:bg-red-950/50 dark:border-red-900',
    icon: 'text-red-700 dark:text-red-400',
    border: 'border-l-red-600',
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
  const defaultIcon = defaultIcons[variant]

  return (
    <div
      className={clsx(
        'rounded-lg border p-4',
        styles.container,
        styles.border,
        'border-l-4',
        className
      )}
    >
      <div className="flex items-start gap-3">
        <div className={clsx('mt-0.5 shrink-0', styles.icon)}>
          {icon || defaultIcon}
        </div>
        <div className="flex-1 min-w-0">
          {title && (
            <h4 className="text-sm font-semibold text-foreground">{title}</h4>
          )}
          {(description || children) && (
            <div className="mt-1">
              {description && (
                <p className="text-sm text-muted-foreground">{description}</p>
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

// Compact inline alert for use in lists
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
        'flex items-center gap-2 rounded-md border-l-4 px-3 py-2 text-sm',
        styles.container,
        styles.border,
        className
      )}
    >
      <span className={clsx('shrink-0', styles.icon)}>
        {defaultIcons[variant]}
      </span>
      <span className="flex-1">{children}</span>
    </div>
  )
}
