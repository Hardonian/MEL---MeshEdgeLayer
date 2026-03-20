import { Loader2, AlertCircle, RefreshCw } from 'lucide-react'
import { clsx } from 'clsx'

// Fade-in wrapper for content transitions
interface FadeInProps {
  children: React.ReactNode
  className?: string
  delay?: number
}

export function FadeIn({ children, className, delay = 0 }: FadeInProps) {
  return (
    <div 
      className={clsx('animate-fade-in', className)}
      style={{ animationDelay: `${delay}ms` }}
    >
      {children}
    </div>
  )
}

// Slide-in from bottom for panels
interface SlideInProps {
  children: React.ReactNode
  className?: string
  delay?: number
}

export function SlideIn({ children, className, delay = 0 }: SlideInProps) {
  return (
    <div 
      className={clsx('animate-slide-up', className)}
      style={{ animationDelay: `${delay}ms` }}
    >
      {children}
    </div>
  )
}

// Expand animation for cards/panels
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
    <div className={clsx('flex flex-col items-center justify-center gap-4 p-12 text-center', className)}>
      <div className="relative">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
        <div className="absolute inset-0 h-10 w-10 animate-ping rounded-full bg-primary/20 opacity-75" />
      </div>
      <p className="text-sm text-muted-foreground">{message}</p>
    </div>
  )
}

// Inline loading spinner for smaller areas
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
  className
}: ErrorViewProps) {
  return (
    <div className={clsx('flex flex-col items-center justify-center gap-4 p-12 text-center', className)}>
      <div className="flex h-16 w-16 items-center justify-center rounded-full bg-critical/10">
        <AlertCircle className="h-8 w-8 text-critical" />
      </div>
      <div>
        <h3 className="font-semibold text-foreground">{title}</h3>
        <p className="mt-1 text-sm text-muted-foreground">{message}</p>
      </div>
      {onRetry && (
        <button
          onClick={onRetry}
          className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
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
    <div className="rounded-xl border bg-card overflow-hidden">
      <div className="border-b p-4">
        <div className="flex gap-4">
          <Skeleton className="h-4 flex-1" />
          <Skeleton className="h-4 flex-1" />
          <Skeleton className="h-4 flex-1" />
          <Skeleton className="h-4 flex-1" />
        </div>
      </div>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="border-b p-4 last:border-0">
          <Skeleton className="h-4 w-full" />
        </div>
      ))}
    </div>
  )
}

export function StatSkeleton() {
  return (
    <div className="rounded-xl border bg-card p-5">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <Skeleton className="h-4 w-20 mb-3" />
          <Skeleton className="h-8 w-16 mb-2" />
          <Skeleton className="h-3 w-24" />
        </div>
        <Skeleton className="h-10 w-10 rounded-lg" />
      </div>
    </div>
  )
}
