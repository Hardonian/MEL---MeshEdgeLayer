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
  return (
    <progress 
      className={clsx(
        'health-progress',
        `progress-${variant}`,
        className
      )}
      value={value} 
      max={max}
      aria-label={`${value}/${max}`}
    />
  )
}
