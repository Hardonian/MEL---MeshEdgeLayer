import { ReactNode } from 'react'
import { Loader2, AlertCircle, Inbox } from 'lucide-react'
import { clsx } from 'clsx'

interface LoadingProps {
  message?: string
  className?: string
}

export function Loading({ message = 'Loading...', className }: LoadingProps) {
  return (
    <div className={clsx('flex flex-col items-center justify-center gap-4 p-8 text-center', className)}>
      <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      <p className="text-sm text-muted-foreground">{message}</p>
    </div>
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
  className
}: ErrorViewProps) {
  return (
    <div className={clsx('flex flex-col items-center justify-center gap-4 p-8 text-center', className)}>
      <AlertCircle className="h-10 w-10 text-critical" />
      <div>
        <h3 className="font-semibold text-foreground">{title}</h3>
        <p className="mt-1 text-sm text-muted-foreground">{message}</p>
      </div>
      {onRetry && (
        <button
          onClick={onRetry}
          className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          Try again
        </button>
      )}
    </div>
  )
}

interface EmptyStateProps {
  icon?: ReactNode
  title: string
  description?: string
  action?: ReactNode
  className?: string
}

export function EmptyState({
  icon,
  title,
  description,
  action,
  className
}: EmptyStateProps) {
  return (
    <div className={clsx('flex flex-col items-center justify-center gap-4 p-8 text-center', className)}>
      {icon || <Inbox className="h-10 w-10 text-muted-foreground" />}
      <div>
        <h3 className="font-semibold text-foreground">{title}</h3>
        {description && (
          <p className="mt-1 text-sm text-muted-foreground">{description}</p>
        )}
      </div>
      {action && <div>{action}</div>}
    </div>
  )
}

interface SkeletonProps {
  className?: string
}

export function Skeleton({ className }: SkeletonProps) {
  return (
    <div className={clsx('animate-pulse rounded-md bg-muted', className)} />
  )
}

export function CardSkeleton() {
  return (
    <div className="rounded-xl border bg-card p-6">
      <Skeleton className="h-4 w-1/3 mb-4" />
      <Skeleton className="h-8 w-1/2 mb-2" />
      <Skeleton className="h-4 w-1/4" />
    </div>
  )
}

export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="rounded-xl border bg-card">
      <div className="border-b p-4">
        <Skeleton className="h-4 w-full" />
      </div>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="border-b p-4 last:border-0">
          <Skeleton className="h-4 w-full" />
        </div>
      ))}
    </div>
  )
}
