import { AlertCircle } from 'lucide-react'
import { clsx } from 'clsx'

interface FadeInProps {
  children: React.ReactNode
  className?: string
  delay?: number
}

const delayClasses: Record<number, string> = {
  0: 'delay-0', 50: 'delay-50', 100: 'delay-100', 150: 'delay-150',
  200: 'delay-200', 250: 'delay-250', 300: 'delay-300', 400: 'delay-400', 500: 'delay-500',
}

export function FadeIn({ children, className, delay = 0 }: FadeInProps) {
  return <div className={clsx('animate-fade-in', delayClasses[delay] || 'delay-0', className)}>{children}</div>
}

interface SlideInProps {
  children: React.ReactNode
  className?: string
  delay?: number
}

export function SlideIn({ children, className, delay = 0 }: SlideInProps) {
  return <div className={clsx('animate-slide-up', delayClasses[delay] || 'delay-0', className)}>{children}</div>
}

interface ExpandInProps {
  children: React.ReactNode
  className?: string
}

export function ExpandIn({ children, className }: ExpandInProps) {
  return <div className={clsx('animate-expand-in', className)}>{children}</div>
}

interface LoadingProps {
  message?: string
  className?: string
}

export function Loading({ message = 'Loading...', className }: LoadingProps) {
  return (
    <div
      className={clsx(
        'surface-panel surface-panel-muted flex min-h-[8rem] flex-col items-center justify-center gap-2 p-6 text-center',
        className
      )}
      role="status"
      aria-live="polite"
    >
      <div className="flex items-center gap-2 text-neon">
        <span className="animate-terminal-blink">█</span>
        <span className="text-mel-sm font-bold">{message}</span>
      </div>
      <p className="text-mel-xs text-muted-foreground/50"># working with current API evidence</p>
    </div>
  )
}

export function InlineLoading({ className }: { className?: string }) {
  return <span className={clsx('animate-terminal-blink text-neon', className)}>█</span>
}

interface ErrorViewProps {
  title?: string
  message?: string
  onRetry?: () => void
  className?: string
}

export function ErrorView({
  title = 'Something went wrong',
  message = 'An error occurred while loading this data.',
  onRetry,
  className,
}: ErrorViewProps) {
  return (
    <div className={clsx('surface-panel flex flex-col items-center justify-center gap-3 p-6 text-center', className)}>
      <div className="text-neon-hot">
        <AlertCircle className="h-6 w-6" />
      </div>
      <div className="space-y-1">
        <p className="text-mel-sm font-bold text-neon-hot neon-glow-hot">[ERR] {title}</p>
        <p className="text-mel-xs text-muted-foreground">{message}</p>
      </div>
      {onRetry && (
        <button onClick={onRetry} className="button-primary">retry</button>
      )}
    </div>
  )
}

interface SkeletonProps {
  className?: string
}

export function Skeleton({ className }: SkeletonProps) {
  return <div className={clsx('skeleton-shimmer', className)} />
}

export function CardSkeleton() {
  return (
    <div className="surface-panel p-3">
      <Skeleton className="mb-2 h-2 w-1/3" />
      <Skeleton className="mb-2 h-5 w-1/2" />
      <Skeleton className="h-2 w-1/4" />
    </div>
  )
}

export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="surface-panel overflow-hidden">
      <div className="border-b border-border px-3 py-2">
        <div className="flex gap-3">
          <Skeleton className="h-2 flex-1" />
          <Skeleton className="h-2 flex-1" />
          <Skeleton className="h-2 flex-1" />
          <Skeleton className="h-2 flex-1" />
        </div>
      </div>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="border-b border-border/30 px-3 py-2 last:border-0">
          <Skeleton className="h-2.5 w-full" />
        </div>
      ))}
    </div>
  )
}

export function StatSkeleton() {
  return (
    <div className="surface-panel p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1">
          <Skeleton className="mb-2 h-2 w-14" />
          <Skeleton className="mb-1 h-5 w-10" />
          <Skeleton className="h-2 w-18" />
        </div>
        <Skeleton className="h-7 w-7" />
      </div>
    </div>
  )
}

export function StaleBanner({ timestamp, message = 'Data may be stale' }: { timestamp?: string; message?: string }) {
  if (!timestamp) return null

  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()

  if (diffMs < 300000) return null

  let timeStr = 'never'
  try {
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMins / 60)
    const diffDays = Math.floor(diffHours / 24)
    if (diffMins < 60) timeStr = `${diffMins}m ago`
    else if (diffHours < 24) timeStr = `${diffHours}h ago`
    else if (diffDays < 7) timeStr = `${diffDays}d ago`
    else timeStr = date.toLocaleDateString()
  } catch { timeStr = timestamp }

  return (
    <div className="surface-inset flex flex-wrap items-center gap-2 border-neon-warn/20 bg-neon-warn/4 px-3 py-1.5 text-mel-sm">
      <AlertCircle className="h-3 w-3 shrink-0 text-neon-warn" aria-hidden />
      <span className="font-bold text-neon-warn">[STALE]</span>
      <span className="text-foreground">{message}</span>
      <span className="ml-auto text-mel-xs text-muted-foreground">last_update: {timeStr}</span>
    </div>
  )
}
