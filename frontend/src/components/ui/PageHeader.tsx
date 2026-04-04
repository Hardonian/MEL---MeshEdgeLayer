import { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { clsx } from 'clsx'

interface PageHeaderProps {
  title: string
  /** Short cockpit label (e.g. mesh operations) — keep optional to avoid churn on every page. */
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
        'mb-6 border-b border-border/60 pb-5 sm:mb-8 sm:pb-6',
        className
      )}
    >
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0 flex-1">
          {breadcrumbs && breadcrumbs.length > 0 && (
            <nav className="mb-3 flex flex-wrap items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
              {breadcrumbs.map((crumb, index) => (
                <span key={`${crumb.label}-${index}`} className="flex items-center gap-2">
                  {index > 0 && <span className="text-muted-foreground/50">/</span>}
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
          <h1 className="font-outfit text-[2rem] font-semibold tracking-[-0.04em] text-foreground sm:text-[2.5rem]">
            {title}
          </h1>
          {subtitle && (
            <p className="mt-1 text-[11px] font-semibold uppercase tracking-[0.2em] text-muted-foreground/80">
              {subtitle}
            </p>
          )}
          {description && (
            <p className="mt-2 max-w-3xl text-sm leading-relaxed text-muted-foreground sm:text-[0.95rem]">
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
