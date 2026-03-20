import { Link } from 'react-router-dom'
import { ChevronRight, Home } from 'lucide-react'
import { clsx } from 'clsx'

interface BreadcrumbItem {
  label: string
  href?: string
}

interface BreadcrumbProps {
  items: BreadcrumbItem[]
  className?: string
}

export function Breadcrumb({ items, className }: BreadcrumbProps) {
  return (
    <nav 
      aria-label="Breadcrumb" 
      className={clsx('flex items-center gap-1 text-sm', className)}
    >
      <ol className="flex items-center gap-1 list-none p-0 m-0">
        <li className="flex items-center">
          <Link 
            to="/" 
            className="flex items-center gap-1 text-muted-foreground hover:text-foreground transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 rounded"
            aria-label="Home"
          >
            <Home className="h-4 w-4" />
            <span className="sr-only">Home</span>
          </Link>
        </li>
        {items.map((item, index) => (
          <li 
            key={index} 
            className="flex items-center gap-1"
          >
            <ChevronRight className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
            {item.href ? (
              <Link 
                to={item.href}
                className="text-muted-foreground hover:text-foreground transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 rounded"
              >
                {item.label}
              </Link>
            ) : (
              <span 
                className="text-foreground font-medium"
                aria-current="page"
              >
                {item.label}
              </span>
            )}
          </li>
        ))}
      </ol>
    </nav>
  )
}

// Quick link for returning to overview pages
interface BackLinkProps {
  href: string
  label: string
}

export function BackLink({ href, label }: BackLinkProps) {
  return (
    <Link 
      to={href}
      className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 rounded"
    >
      <ChevronRight className="h-4 w-4 rotate-180" aria-hidden="true" />
      {label}
    </Link>
  )
}
