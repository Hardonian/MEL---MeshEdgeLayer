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
    container: 'bg-info/5 border-info/20',
    icon: 'text-info',
    border: 'border-l-info',
  },
  warning: {
    container: 'bg-warning/10 border-warning/25',
    icon: 'text-warning',
    border: 'border-l-warning',
  },
  success: {
    container: 'bg-success/10 border-success/25',
    icon: 'text-success',
    border: 'border-l-success',
  },
  error: {
    container: 'bg-critical/10 border-critical/25',
    icon: 'text-critical',
    border: 'border-l-critical',
  },
  critical: {
    container: 'bg-critical/10 border-critical/30',
    icon: 'text-critical',
    border: 'border-l-critical',
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
