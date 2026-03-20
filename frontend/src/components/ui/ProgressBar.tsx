import { clsx } from 'clsx'

interface ProgressBarProps {
  value: number
  max?: number
  variant?: 'success' | 'warning' | 'critical' | 'default'
  className?: string
}

export function ProgressBar({ 
  value, 
  max = 100, 
  variant = 'default',
  className 
}: ProgressBarProps) {
  const percentage = Math.min(Math.max((value / max) * 100, 0), 100)
  
  const variantStyles = {
    default: 'bg-primary',
    success: 'bg-success',
    warning: 'bg-warning',
    critical: 'bg-critical',
  }

  const fillStyle: React.CSSProperties = { width: `${percentage}%` }

  return (
    <div className={clsx('h-2 w-full overflow-hidden rounded-full bg-muted', className)}>
      <div
        className={clsx(
          'h-full rounded-full transition-all duration-500 ease-in-out',
          variantStyles[variant]
        )}
        style={fillStyle}
        role="progressbar"
        aria-valuenow={Math.round(value)}
        aria-valuemin={0}
        aria-valuemax={max}
        aria-label={`${value}/${max}`}
      />
    </div>
  )
}
