import { Loader2, AlertCircle, Terminal } from 'lucide-react'
import { clsx } from 'clsx'

interface FadeInProps {
  children: React.ReactNode
  className?: string
  delay?: number
}

const delayClasses: Record<number, string> = {
  0: 'delay-0',
  50: 'delay-50',
  100: 'delay-100',
  150: 'delay-150',
  200: 'delay-200',
  250: 'delay-250',
  300: 'delay-300',
  400: 'delay-400',
  500: 'delay-500',
}

export function FadeIn({ children, className, delay = 0 }: FadeInProps) {
  return (
    <div className={clsx('animate-fade-in', delayClasses[delay] || 'delay-0', className)}>
      {children}
    </div>
  )
}

interface SlideInProps {
  children: React.ReactNode
  className?: string
  delay?: number
}

export function SlideIn({ children, className, delay = 0 }: SlideInProps) {
  return (
    <div className={clsx('animate-slide-up', delayClasses[delay] || 'delay-0', className)}>
      {children}
    </div>
  )
}

interface ExpandInProps {
  children: React.ReactNode
  className?: string
}

export function ExpandIn({ children, className }: ExpandInProps) {
  return (
    <div className={clsx('animate-expand-in', className)}>
      {children}
    </div>
  )
}

interface LoadingProps {
  message?: string
  className?: string
}

export function Loading({ message = 'Loading...', className }: LoadingProps) {
  return (
    <div
      className={clsx(
        'surface-panel surface-panel-muted flex min-h-[10rem] flex-col items-center justify-center gap-3 p-8 text-center',
        className
      )}
      role="status"
      aria-live="polite"
    >
      <div className="flex h-10 w-10 items-center justify-center rounded-md border border-primary/20 bg-primary/8">
        <Loader2 className="h-5 w-5 animate-spin text-primary" aria-hidden />
      </div>
      <div className="space-y-1">
        <p className="font-mono text-mel-sm font-medium text-foreground">{message}</p>
        <p className="mel-label text-muted-foreground/60">Working with current API evidence</p>
      </div>
    </div>
  )
}

export function InlineLoading({ className }: { className?: string }) {
  return (
    <Loader2 className={clsx('h-4 w-4 animate-spin text-muted-foreground', className)} />
  )
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
    <div className={clsx('surface-panel flex flex-col items-center justify-center gap-3 p-8 text-center', className)}>
      <div className="flex h-10 w-10 items-center justify-center rounded-md border border-critical/20 bg-critical/8">
        <AlertCircle className="h-5 w-5 text-critical" />
      </div>
      <div className="space-y-1">
        <h3 className="font-display text-sm font-bold text-foreground">{title}</h3>
        <p className="prose-body text-mel-sm text-muted-foreground">{message}</p>
      </div>
      {onRetry && (
        <button onClick={onRetry} className="button-primary">
          Try again
        </button>
      )}
    </div>
  )
}

interface SkeletonProps {
  className?: string
}

export function Skeleton({ className }: SkeletonProps) {
  return (
    <div className={clsx('skeleton-shimmer rounded-sm', className)} />
  )
}

export function CardSkeleton() {
  return (
    <div className="surface-panel p-4">
      <Skeleton className="mb-3 h-3 w-1/3 rounded-sm" />
      <Skeleton className="mb-2 h-6 w-1/2 rounded-sm" />
      <Skeleton className="h-3 w-1/4 rounded-sm" />
    </div>
  )
}

export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="surface-panel overflow-hidden">
      <div className="border-b border-border/50 px-3 py-3">
        <div className="flex gap-3">
          <Skeleton className="h-3 flex-1 rounded-sm" />
          <Skeleton className="h-3 flex-1 rounded-sm" />
          <Skeleton className="h-3 flex-1 rounded-sm" />
          <Skeleton className="h-3 flex-1 rounded-sm" />
        </div>
      </div>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="border-b border-border/40 px-3 py-3 last:border-0">
          <Skeleton className="h-3 w-full rounded-sm" />
        </div>
      ))}
    </div>
  )
}

export function StatSkeleton() {
  return (
    <div className="surface-panel p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1">
          <Skeleton className="mb-2 h-2.5 w-16 rounded-sm" />
          <Skeleton className="mb-2 h-6 w-12 rounded-sm" />
          <Skeleton className="h-2.5 w-20 rounded-sm" />
        </div>
        <Skeleton className="h-9 w-9 rounded-md" />
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
  } catch {
    timeStr = timestamp
  }

  return (
    <div className="surface-inset flex flex-wrap items-center gap-2 border-warning/20 bg-warning/6 px-3 py-2 font-mono text-mel-sm text-foreground">
      <AlertCircle className="h-3.5 w-3.5 shrink-0 text-warning" aria-hidden />
      <span className="font-semibold text-warning">{message}</span>
      <span className="ml-auto mel-label text-muted-foreground">Last updated {timeStr}</span>
    </div>
  )
}
