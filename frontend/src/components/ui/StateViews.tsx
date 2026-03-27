import { Loader2, AlertCircle } from 'lucide-react'
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
        'surface-panel surface-panel-muted flex min-h-[12rem] flex-col items-center justify-center gap-4 p-10 text-center',
        className
      )}
      role="status"
      aria-live="polite"
    >
      <div className="flex h-14 w-14 items-center justify-center rounded-[1.25rem] border border-primary/16 bg-primary/10 shadow-inset">
        <Loader2 className="h-7 w-7 animate-spin text-primary" aria-hidden />
      </div>
      <div className="space-y-1">
        <p className="text-sm font-medium text-foreground">{message}</p>
        <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Working with current API evidence</p>
      </div>
    </div>
  )
}

export function InlineLoading({ className }: { className?: string }) {
  return (
    <Loader2 className={clsx('h-5 w-5 animate-spin text-muted-foreground', className)} />
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
    <div className={clsx('surface-panel flex flex-col items-center justify-center gap-4 p-10 text-center', className)}>
      <div className="flex h-16 w-16 items-center justify-center rounded-[1.3rem] border border-critical/18 bg-critical/10 shadow-inset">
        <AlertCircle className="h-8 w-8 text-critical" />
      </div>
      <div className="space-y-1.5">
        <h3 className="font-outfit text-lg font-semibold text-foreground">{title}</h3>
        <p className="text-sm leading-relaxed text-muted-foreground">{message}</p>
      </div>
      {onRetry && (
        <button
          onClick={onRetry}
          className="button-primary"
        >
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
    <div className={clsx('skeleton-shimmer rounded-md', className)} />
  )
}

export function CardSkeleton() {
  return (
    <div className="surface-panel p-6">
      <Skeleton className="mb-4 h-4 w-1/3 rounded-full" />
      <Skeleton className="mb-2 h-8 w-1/2 rounded-full" />
      <Skeleton className="h-4 w-1/4 rounded-full" />
    </div>
  )
}

export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="surface-panel overflow-hidden">
      <div className="border-b border-border/60 px-4 py-4">
        <div className="flex gap-4">
          <Skeleton className="h-4 flex-1 rounded-full" />
          <Skeleton className="h-4 flex-1 rounded-full" />
          <Skeleton className="h-4 flex-1 rounded-full" />
          <Skeleton className="h-4 flex-1 rounded-full" />
        </div>
      </div>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="border-b border-border/50 px-4 py-4 last:border-0">
          <Skeleton className="h-4 w-full rounded-full" />
        </div>
      ))}
    </div>
  )
}

export function StatSkeleton() {
  return (
    <div className="surface-panel p-5">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1">
          <Skeleton className="mb-3 h-3 w-20 rounded-full" />
          <Skeleton className="mb-2 h-8 w-16 rounded-full" />
          <Skeleton className="h-3 w-24 rounded-full" />
        </div>
        <Skeleton className="h-11 w-11 rounded-2xl" />
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
    <div className="surface-inset flex flex-wrap items-center gap-2 rounded-xl border-warning/22 bg-warning/10 px-3 py-2 text-sm text-foreground">
      <AlertCircle className="h-4 w-4 shrink-0 text-warning" aria-hidden />
      <span className="font-semibold text-warning">{message}</span>
      <span className="ml-auto text-xs uppercase tracking-[0.16em] text-muted-foreground">Last updated {timeStr}</span>
    </div>
  )
}
