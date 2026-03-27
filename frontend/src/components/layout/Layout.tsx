import { useState, useEffect, useCallback } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { clsx } from 'clsx'
import { useApi, useStatus } from '@/hooks/useApi'
import { HelpMenu } from '@/components/ui/HelpMenu'
import {
  LayoutDashboard,
  Radio,
  MessageSquare,
  Inbox,
  Settings,
  Shield,
  FileText,
  Menu,
  X,
  Activity,
  GitBranch,
  Compass,
  AlertTriangle,
  Zap,
  RefreshCw,
} from 'lucide-react'

interface NavItem {
  label: string
  href: string
  icon: React.ElementType
}

const navItems: NavItem[] = [
  { label: 'Dashboard', href: '/', icon: LayoutDashboard },
  { label: 'Status', href: '/status', icon: Activity },
  { label: 'Nodes', href: '/nodes', icon: Radio },
  { label: 'Topology', href: '/topology', icon: GitBranch },
  { label: 'Planning', href: '/planning', icon: Compass },
  { label: 'Messages', href: '/messages', icon: MessageSquare },
  { label: 'Dead Letters', href: '/dead-letters', icon: Inbox },
  { label: 'Incidents', href: '/incidents', icon: AlertTriangle },
  { label: 'Control actions', href: '/control-actions', icon: Zap },
  { label: 'Events', href: '/events', icon: FileText },
  { label: 'Diagnostics', href: '/diagnostics', icon: Shield },
  { label: 'Privacy', href: '/privacy', icon: Shield },
  { label: 'Recommendations', href: '/recommendations', icon: Activity },
  { label: 'Settings', href: '/settings', icon: Settings },
]

export function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const status = useStatus()
  const { refreshAll } = useApi()
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)
  const [isMobile, setIsMobile] = useState(false)
  const [refreshBusy, setRefreshBusy] = useState(false)

  const transports = status.data?.transports ?? []
  const hasTransports = transports.length > 0
  const hasConnectedTransport = transports.some((t) => t.effective_state === 'connected')

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 768)
    }
    checkMobile()
    window.addEventListener('resize', checkMobile)
    return () => window.removeEventListener('resize', checkMobile)
  }, [])

  // Close mobile menu on route change
  useEffect(() => {
    setIsMobileMenuOpen(false)
  }, [location.pathname])

  const handleRefresh = useCallback(async () => {
    setRefreshBusy(true)
    try {
      await refreshAll()
    } finally {
      setRefreshBusy(false)
    }
  }, [refreshAll])

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      setIsMobileMenuOpen(false)
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [])

  return (
    <div className="min-h-screen bg-background">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[100] focus:rounded-md focus:bg-background focus:px-3 focus:py-2 focus:text-sm focus:shadow focus:outline-none focus:ring-2 focus:ring-ring"
      >
        Skip to main content
      </a>
      {/* Top navigation */}
      <header className="sticky top-0 z-50 border-b border-border/60 bg-background/80 backdrop-blur-md supports-[backdrop-filter]:bg-background/70">
        <div className="flex h-14 items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-4">
            <button
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              className="rounded-md p-2 hover:bg-muted md:hidden"
              aria-label="Toggle menu"
            >
              {isMobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
            <Link
              to="/"
              className="flex items-center gap-2 rounded-md outline-none ring-offset-background transition-transform focus-visible:ring-2 focus-visible:ring-ring group hover:opacity-90"
            >
              <div className="rounded-lg bg-primary p-1.5 text-primary-foreground shadow-inner transition-transform group-hover:scale-[1.02]">
                <Radio className="h-5 w-5" />
              </div>
              <span className="hidden font-outfit text-xl font-bold tracking-tight sm:inline">MEL</span>
            </Link>
          </div>

          <div className="flex items-center gap-1 sm:gap-2">
            <div
              className={clsx(
                'hidden items-center gap-2 rounded-full border px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wider md:flex',
                !status.loading && !hasTransports && 'border-border bg-muted/40 text-muted-foreground',
                !status.loading && hasTransports && hasConnectedTransport && 'border-success/25 bg-success/5 text-foreground',
                !status.loading && hasTransports && !hasConnectedTransport && 'border-warning/30 bg-warning/10 text-foreground',
                status.loading && 'border-border bg-muted/30 text-muted-foreground'
              )}
            >
              <span
                className={clsx(
                  'h-1.5 w-1.5 shrink-0 rounded-full',
                  status.loading && 'bg-muted-foreground',
                  !status.loading && !hasTransports && 'bg-muted-foreground',
                  !status.loading && hasTransports && hasConnectedTransport && 'bg-success animate-pulse',
                  !status.loading && hasTransports && !hasConnectedTransport && 'bg-warning'
                )}
                aria-hidden
              />
              {status.loading
                ? 'Loading transport state…'
                : !hasTransports
                  ? 'No transport configured'
                  : hasConnectedTransport
                    ? 'Transport connected'
                    : 'No active transport'}
            </div>
            <button
              type="button"
              onClick={() => void handleRefresh()}
              disabled={refreshBusy}
              className="rounded-full p-2 text-muted-foreground transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
              aria-label={refreshBusy ? 'Refreshing data' : 'Refresh console data from API'}
              title="Re-fetch dashboard data (same 30s polling cadence still applies)"
            >
              <RefreshCw className={clsx('h-5 w-5', refreshBusy && 'animate-spin')} aria-hidden />
            </button>
            <HelpMenu />
          </div>
        </div>
      </header>

      <div className="flex">
        {/* Sidebar navigation - desktop */}
        <aside className={clsx(
          "fixed inset-y-0 left-0 z-40 w-64 transform border-r border-border/80 bg-background transition-transform duration-200 ease-in-out md:translate-x-0",
          isMobile && isMobileMenuOpen ? "translate-x-0" : "-translate-x-full"
        )}>
          <nav className="flex flex-col gap-0.5 p-3 pt-14" aria-label="Primary">
            {navItems.map((item) => {
              const isActive = location.pathname === item.href || 
                (item.href !== '/' && location.pathname.startsWith(item.href))
              return (
                <Link
                  key={item.href}
                  to={item.href}
                  className={clsx(
                    "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring",
                    isActive
                      ? "bg-primary/10 text-primary"
                      : "text-muted-foreground hover:bg-muted hover:text-foreground"
                  )}
                >
                  <item.icon className="h-4 w-4" />
                  {item.label}
                </Link>
              )
            })}
          </nav>

          {/* Bottom section */}
          <div className="absolute bottom-0 left-0 right-0 border-t border-border/80 p-3">
            <Link
              to="/settings"
              className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground outline-none transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
            >
              <Settings className="h-4 w-4 shrink-0" aria-hidden />
              Settings &amp; reference
            </Link>
          </div>
        </aside>

        {/* Mobile menu overlay */}
        {isMobile && isMobileMenuOpen && (
          <div
            className="fixed inset-0 z-30 bg-black/50 md:hidden"
            onClick={() => setIsMobileMenuOpen(false)}
          />
        )}

        {/* Main content */}
        <main
          id="main-content"
          tabIndex={-1}
          className="flex-1 px-4 py-5 outline-none md:ml-64 md:max-w-[min(100%,88rem)] md:px-8 md:py-6"
        >
          {children}
        </main>
      </div>
    </div>
  )
}
