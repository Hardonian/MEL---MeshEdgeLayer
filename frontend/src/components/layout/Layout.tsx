import { useState, useEffect, useCallback } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { clsx } from 'clsx'
import { useApi, useStatus } from '@/hooks/useApi'
import { HelpMenu } from '@/components/ui/HelpMenu'
import { truncateMiddle } from '@/utils/presentation'
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
  const instanceId = status.data?.instance?.instance_id
  const productScope = status.data?.product?.product_scope

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < 768)
    }
    checkMobile()
    window.addEventListener('resize', checkMobile)
    return () => window.removeEventListener('resize', checkMobile)
  }, [])

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

  const transportStatusLabel = status.loading
    ? 'Loading transport state'
    : !hasTransports
      ? 'No transport configured'
      : hasConnectedTransport
        ? 'Transport connected'
        : 'No active transport'

  const transportStatusClasses = clsx(
    'status-pill hidden md:inline-flex',
    status.loading && 'border-border/80 text-muted-foreground',
    !status.loading && !hasTransports && 'border-border/80 text-muted-foreground',
    !status.loading && hasTransports && hasConnectedTransport && 'border-success/30 text-foreground',
    !status.loading && hasTransports && !hasConnectedTransport && 'border-warning/30 text-foreground'
  )

  const transportDotClasses = clsx(
    'status-dot',
    status.loading && 'bg-muted-foreground text-muted-foreground',
    !status.loading && !hasTransports && 'bg-muted-foreground text-muted-foreground',
    !status.loading && hasTransports && hasConnectedTransport && 'animate-pulse-slow bg-success text-success',
    !status.loading && hasTransports && !hasConnectedTransport && 'bg-warning text-warning'
  )

  return (
    <div className="min-h-screen">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[100] focus:rounded-xl focus:bg-background focus:px-3 focus:py-2 focus:text-sm focus:shadow-panel focus:outline-none focus:ring-2 focus:ring-ring"
      >
        Skip to main content
      </a>

      <header className="sticky top-0 z-50 px-3 pt-3 md:px-4">
        <div className="surface-toolbar mx-auto flex h-16 max-w-[min(100%,96rem)] items-center justify-between px-3 sm:px-4">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              className="icon-button md:hidden"
              aria-label="Toggle menu"
            >
              {isMobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>

            <Link
              to="/"
              className="group flex items-center gap-3 rounded-xl outline-none transition-opacity hover:opacity-95 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            >
              <div className="flex h-10 w-10 items-center justify-center rounded-2xl border border-primary/20 bg-primary/14 text-primary shadow-[0_14px_28px_-20px_hsl(var(--primary)/0.72)] transition-transform duration-200 group-hover:-translate-y-0.5">
                <Radio className="h-5 w-5" />
              </div>
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-outfit text-xl font-semibold tracking-[-0.03em] text-foreground">MEL</span>
                  <span className="hidden rounded-full border border-border/70 bg-card/70 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground sm:inline-flex">
                    Operator Console
                  </span>
                </div>
                <p className="hidden text-xs text-muted-foreground sm:block">
                  MeshEdgeLayer control plane
                </p>
              </div>
            </Link>
          </div>

          <div className="flex items-center gap-2">
            <div className={transportStatusClasses}>
              <span className={transportDotClasses} aria-hidden />
              {transportStatusLabel}
            </div>

            <button
              type="button"
              onClick={() => void handleRefresh()}
              disabled={refreshBusy}
              className="icon-button"
              aria-label={refreshBusy ? 'Refreshing data' : 'Refresh console data from API'}
              title={refreshBusy ? 'Refreshing…' : 'Re-fetch dashboard data from the API'}
            >
              <RefreshCw className={clsx('h-4 w-4', refreshBusy && 'animate-spin')} aria-hidden />
            </button>

            <HelpMenu />
          </div>
        </div>
      </header>

      <div className="flex">
        <aside
          className={clsx(
            'fixed inset-y-0 left-0 z-40 w-72 transform px-3 pb-3 pt-24 transition-transform duration-200 ease-out md:translate-x-0 md:px-4',
            isMobile && isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'
          )}
        >
          <div className="surface-panel surface-panel-strong flex h-full flex-col overflow-hidden">
            <div className="border-b border-border/60 px-4 pb-3 pt-4">
              <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
                Console
              </p>
              <p className="mt-1 text-sm text-foreground">Operational surfaces</p>
            </div>

            <nav className="flex-1 space-y-1 overflow-y-auto p-3" aria-label="Primary">
              {navItems.map((item) => {
                const isActive =
                  location.pathname === item.href ||
                  (item.href !== '/' && location.pathname.startsWith(item.href))

                return (
                  <Link
                    key={item.href}
                    to={item.href}
                    className={clsx(
                      'group flex items-center gap-3 rounded-2xl border px-3 py-2.5 text-sm font-medium outline-none transition-all duration-150 focus-visible:ring-2 focus-visible:ring-ring',
                      isActive
                        ? 'border-primary/22 bg-primary/12 text-foreground shadow-[0_18px_28px_-24px_hsl(var(--primary)/0.65)]'
                        : 'border-transparent text-muted-foreground hover:border-border/60 hover:bg-accent/65 hover:text-foreground'
                    )}
                    aria-current={isActive ? 'page' : undefined}
                  >
                    <span
                      className={clsx(
                        'flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border transition-colors',
                        isActive
                          ? 'border-primary/20 bg-primary/16 text-primary'
                          : 'border-border/70 bg-card/70 text-muted-foreground group-hover:border-primary/12 group-hover:text-foreground'
                      )}
                    >
                      <item.icon className="h-4 w-4" />
                    </span>
                    <span className="truncate">{item.label}</span>
                  </Link>
                )
              })}
            </nav>

            <div className="border-t border-border/60 p-3">
              <Link
                to="/settings"
                className="group flex items-center gap-3 rounded-2xl border border-transparent px-3 py-2.5 text-sm font-medium text-muted-foreground outline-none transition-all duration-150 hover:border-border/60 hover:bg-accent/65 hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
              >
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-border/70 bg-card/70 text-muted-foreground transition-colors group-hover:border-primary/12 group-hover:text-foreground">
                  <Settings className="h-4 w-4" aria-hidden />
                </span>
                Settings &amp; reference
              </Link>
            </div>
          </div>
        </aside>

        {isMobile && isMobileMenuOpen && (
          <div
            className="fixed inset-0 z-30 bg-background/72 backdrop-blur-sm md:hidden"
            onClick={() => setIsMobileMenuOpen(false)}
          />
        )}

        <main
          id="main-content"
          tabIndex={-1}
          className="flex-1 px-4 pb-10 pt-4 outline-none md:ml-72 md:px-6 md:pt-5"
        >
          <div className="mx-auto w-full max-w-8xl page-enter">{children}</div>
        </main>
      </div>

      {!status.loading && (instanceId || productScope) && (
        <footer className="px-4 pb-5 md:pl-[21rem] md:pr-6">
          <div className="surface-toolbar mx-auto flex max-w-8xl flex-wrap items-center gap-x-4 gap-y-2 px-4 py-2 text-[11px] text-muted-foreground">
            {productScope && (
              <span title={status.data?.product?.notes}>
                Scope:{' '}
                <code className="rounded-md border border-border/60 bg-card/75 px-1.5 py-0.5 font-mono text-[10px] text-foreground">
                  {productScope}
                </code>
              </span>
            )}
            {instanceId && (
              <span>
                Instance:{' '}
                <code className="rounded-md border border-border/60 bg-card/75 px-1.5 py-0.5 font-mono text-[10px] text-foreground" title={instanceId}>
                  {truncateMiddle(instanceId, 36)}
                </code>
              </span>
            )}
          </div>
        </footer>
      )}
    </div>
  )
}
