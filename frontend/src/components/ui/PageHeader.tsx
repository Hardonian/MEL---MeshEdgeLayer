import { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { clsx } from 'clsx'

interface PageHeaderProps {
  title: string
  subtitle?: string
  description?: string
  action?: ReactNode
  breadcrumbs?: {
    label: string
    href?: string
  }[]
  className?: string
}

export function PageHeader({
  title,
  subtitle,
  description,
  action,
  breadcrumbs,
  className,
}: PageHeaderProps) {
  return (
    <div
      className={clsx(
        'mb-5 border-b border-border/50 pb-4 sm:mb-6 sm:pb-5',
        className
      )}
    >
      <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0 flex-1">
          {breadcrumbs && breadcrumbs.length > 0 && (
            <nav className="mb-2 flex flex-wrap items-center gap-1.5 font-mono text-mel-xs text-muted-foreground">
              {breadcrumbs.map((crumb, index) => (
                <span key={`${crumb.label}-${index}`} className="flex items-center gap-1.5">
                  {index > 0 && <span className="text-muted-foreground/40">/</span>}
                  {crumb.href ? (
                    <Link
                      to={crumb.href}
                      className="transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      {crumb.label}
                    </Link>
                  ) : (
                    <span className="text-foreground">{crumb.label}</span>
                  )}
                </span>
              ))}
            </nav>
          )}
          <h1 className="font-display text-xl font-bold tracking-tight text-foreground sm:text-2xl">
            {title}
          </h1>
          {subtitle && (
            <p className="mt-1 mel-label text-muted-foreground/70">
              {subtitle}
            </p>
          )}
          {description && (
            <p className="mt-1.5 max-w-3xl prose-body text-mel-sm text-muted-foreground">
              {description}
            </p>
          )}
        </div>
        {action && (
          <div className="flex shrink-0 items-center gap-2 lg:justify-end">
            {action}
          </div>
        )}
      </div>
    </div>
  )
}
