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

const variantStyles: Record<AlertVariant, { container: string; prefix: string; rail: string }> = {
  info: { container: 'border-neon-alt/20 bg-neon-alt/4', prefix: 'INFO', rail: 'bg-neon-alt' },
  warning: { container: 'border-neon-warn/20 bg-neon-warn/4', prefix: 'WARN', rail: 'bg-neon-warn' },
  success: { container: 'border-neon/20 bg-neon/4', prefix: 'OK', rail: 'bg-neon' },
  error: { container: 'border-neon-hot/20 bg-neon-hot/4', prefix: 'ERR', rail: 'bg-neon-hot' },
  critical: { container: 'border-neon-hot/25 bg-neon-hot/6', prefix: '!!CRIT', rail: 'bg-neon-hot' },
}

const defaultIcons: Record<AlertVariant, ReactNode> = {
  info: <Info className="h-4 w-4" />,
  warning: <AlertTriangle className="h-4 w-4" />,
  success: <CheckCircle2 className="h-4 w-4" />,
  error: <XCircle className="h-4 w-4" />,
  critical: <AlertTriangle className="h-4 w-4" />,
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
    <div className={clsx('surface-panel relative overflow-hidden p-3', styles.container, className)}>
      <div className={clsx('absolute inset-y-0 left-0 w-0.5', styles.rail, 'opacity-60')} aria-hidden />
      <div className="flex items-start gap-2 pl-2">
        <div className="mt-0.5 shrink-0 text-inherit">{icon || defaultIcons[variant]}</div>
        <div className="min-w-0 flex-1">
          {title && (
            <h4 className="text-mel-sm font-bold text-foreground">
              <span className="text-inherit">[{styles.prefix}]</span> {title}
            </h4>
          )}
          {(description || children) && (
            <div className="mt-1 space-y-1">
              {description && <p className="text-mel-xs text-muted-foreground">{description}</p>}
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
    <div className={clsx('surface-inset flex items-center gap-2 px-3 py-2 text-mel-sm', styles.container, className)}>
      <span className="font-bold">[{styles.prefix}]</span>
      <span className="min-w-0 flex-1">{children}</span>
    </div>
  )
}
