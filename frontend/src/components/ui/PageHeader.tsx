import { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { clsx } from 'clsx'
import { Badge, type BadgeVariant } from '@/components/ui/Badge'

interface PageHeaderProps {
  title: string
  subtitle?: string
  description?: string
  action?: ReactNode
  statusChips?: Array<{
    label: string
    value: string
    variant?: BadgeVariant
  }>
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
  statusChips,
  breadcrumbs,
  className,
}: PageHeaderProps) {
  return (
    <header className={clsx('mb-4 surface-panel surface-panel-strong p-3 md:p-4', className)}>
      <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0 flex-1">
          {breadcrumbs && breadcrumbs.length > 0 && (
            <nav className="mb-2 flex flex-wrap items-center gap-1 text-mel-xs text-muted-foreground" aria-label="Breadcrumb">
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
          <h1 className="mel-prompt text-base font-bold uppercase tracking-[0.04em] text-foreground sm:text-lg">
            {title}
          </h1>
          {subtitle && (
            <p className="mt-1 text-mel-xs text-muted-foreground/70"># {subtitle}</p>
          )}
          {description && (
            <p className="mt-1.5 max-w-4xl text-mel-sm text-muted-foreground">{description}</p>
          )}
          {statusChips && statusChips.length > 0 && (
            <ul className="mt-3 flex flex-wrap items-center gap-1.5" aria-label="Header status">
              {statusChips.map((chip) => (
                <li key={`${chip.label}-${chip.value}`} className="inline-flex items-center gap-1">
                  <span className="text-mel-xs text-muted-foreground">{chip.label}:</span>
                  <Badge variant={chip.variant ?? 'secondary'}>{chip.value}</Badge>
                </li>
              ))}
            </ul>
          )}
        </div>
        {action && (
          <div className="flex shrink-0 items-center gap-2 lg:justify-end lg:self-start">
            {action}
          </div>
        )}
      </div>
    </header>
  )
}
