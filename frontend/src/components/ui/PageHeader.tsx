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
    <div className={clsx('mb-4 border-b border-border/50 pb-3', className)}>
      <div className="flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0 flex-1">
          {breadcrumbs && breadcrumbs.length > 0 && (
            <nav className="mb-1.5 flex flex-wrap items-center gap-1 text-mel-xs text-muted-foreground">
              {breadcrumbs.map((crumb, index) => (
                <span key={`${crumb.label}-${index}`} className="flex items-center gap-1">
                  {index > 0 && <span className="text-muted-foreground/30">/</span>}
                  {crumb.href ? (
                    <Link to={crumb.href} className="hover:text-neon focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring">
                      {crumb.label}
                    </Link>
                  ) : (
                    <span className="text-foreground">{crumb.label}</span>
                  )}
                </span>
              ))}
            </nav>
          )}
          <h1 className="mel-prompt text-base font-bold uppercase tracking-[0.04em] text-foreground">
            {title}
          </h1>
          {subtitle && (
            <p className="mt-0.5 text-mel-xs text-muted-foreground/60"># {subtitle}</p>
          )}
          {description && (
            <p className="mt-1 max-w-3xl text-mel-sm text-muted-foreground">{description}</p>
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
