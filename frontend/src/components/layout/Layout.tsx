import { useState, useEffect, useCallback } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { clsx } from 'clsx'
import { useApi, useStatus } from '@/hooks/useApi'
import { HelpMenu, useGlobalKeyboardShortcuts } from '@/components/ui/HelpMenu'
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
  Search,
  Eye,
  Wrench,
} from 'lucide-react'

interface NavItem {
  label: string
  href: string
  icon: React.ElementType
  badge?: number | string
  badgeVariant?: 'default' | 'warning' | 'critical'
}

interface NavGroup {
  label: string
  icon: React.ElementType
  items: NavItem[]
}

export function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const navigate = useNavigate()
  const status = useStatus()
  const api = useApi()
  const { refreshAll } = api
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)
  const [isMobile, setIsMobile] = useState(false)
  const [refreshBusy, setRefreshBusy] = useState(false)
  const [commandOpen, setCommandOpen] = useState(false)

  const transports = status.data?.transports ?? []
  const hasTransports = transports.length > 0
  const hasConnectedTransport = transports.some((t) => t.effective_state === 'connected')
  const instanceId = status.data?.instance?.instance_id
  const productScope = status.data?.product?.product_scope

  // Compute live counts for nav badges
  const deadLetterCount = api.deadLetters.data?.length ?? 0
  const diagCount = api.diagnostics.data?.filter((d) => d.severity === 'critical' || d.severity === 'high').length ?? 0
  const privacyCount = api.privacyFindings.data?.filter((p) => p.severity === 'critical' || p.severity === 'high').length ?? 0
  const recsCount = api.recommendations.data?.filter((r) => r.actionable).length ?? 0

  const navGroups: NavGroup[] = [
    {
      label: 'Overview',
      icon: Eye,
      items: [
        { label: 'Dashboard', href: '/', icon: LayoutDashboard },
        { label: 'Status', href: '/status', icon: Activity },
        { label: 'Events', href: '/events', icon: FileText },
      ],
    },
    {
      label: 'Mesh',
      icon: Radio,
      items: [
        { label: 'Nodes', href: '/nodes', icon: Radio },
        { label: 'Topology', href: '/topology', icon: GitBranch },
        { label: 'Messages', href: '/messages', icon: MessageSquare },
        {
          label: 'Dead letters',
          href: '/dead-letters',
          icon: Inbox,
          badge: deadLetterCount > 0 ? deadLetterCount : undefined,
          badgeVariant: 'warning' as const,
        },
      ],
    },
    {
      label: 'Operations',
      icon: Wrench,
      items: [
        { label: 'Incidents', href: '/incidents', icon: AlertTriangle },
        { label: 'Control actions', href: '/control-actions', icon: Zap },
        { label: 'Planning', href: '/planning', icon: Compass },
        {
          label: 'Recommendations',
          href: '/recommendations',
          icon: Activity,
          badge: recsCount > 0 ? recsCount : undefined,
        },
      ],
    },
    {
      label: 'Trust & safety',
      icon: Shield,
      items: [
        {
          label: 'Diagnostics',
          href: '/diagnostics',
          icon: Shield,
          badge: diagCount > 0 ? diagCount : undefined,
          badgeVariant: diagCount > 0 ? 'critical' as const : undefined,
        },
        {
          label: 'Privacy',
          href: '/privacy',
          icon: Shield,
          badge: privacyCount > 0 ? privacyCount : undefined,
          badgeVariant: privacyCount > 0 ? 'critical' as const : undefined,
        },
      ],
    },
  ]

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

  // Global keyboard shortcuts: g+letter nav, r=refresh
  useGlobalKeyboardShortcuts(handleRefresh)

  // Page Visibility API: refresh when tab becomes visible after being hidden
  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === 'visible') {
        void refreshAll()
      }
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => document.removeEventListener('visibilitychange', onVisible)
  }, [refreshAll])

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null
      const inEditable =
        !!target &&
        (target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.tagName === 'SELECT' ||
          target.isContentEditable)

      if (e.key === 'Escape') {
        setIsMobileMenuOpen(false)
        setCommandOpen(false)
      }
      // Cmd+K or Ctrl+K for command palette
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setCommandOpen((prev) => !prev)
      }
      if (inEditable) return
      if (e.key === 'g') {
        const onSecond = (ev: KeyboardEvent) => {
          if (ev.key === 'i') navigate('/incidents')
          if (ev.key === 't') navigate('/topology')
          if (ev.key === 'p') navigate('/planning')
          if (ev.key === 's') navigate('/status')
          document.removeEventListener('keydown', onSecond)
        }
        document.addEventListener('keydown', onSecond, { once: true })
      }
      if (e.key.toLowerCase() === 'r') {
        e.preventDefault()
        void handleRefresh()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [navigate, handleRefresh])

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
        <div className="surface-toolbar mx-auto flex h-14 max-w-[min(100%,96rem)] items-center justify-between px-3 sm:px-4">
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
              <div className="flex h-9 w-9 items-center justify-center rounded-2xl border border-primary/20 bg-primary/14 text-primary shadow-[0_14px_28px_-20px_hsl(var(--primary)/0.72)] transition-transform duration-200 group-hover:-translate-y-0.5">
                <Radio className="h-4 w-4" />
              </div>
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-outfit text-lg font-semibold tracking-[-0.03em] text-foreground">MEL</span>
                  <span className="hidden rounded-full border border-border/70 bg-card/70 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground sm:inline-flex">
                    Console
                  </span>
                </div>
              </div>
            </Link>
          </div>

          <div className="flex items-center gap-1.5">
            <div className={transportStatusClasses}>
              <span className={transportDotClasses} aria-hidden />
              {transportStatusLabel}
            </div>
            <div className="status-pill hidden lg:inline-flex">
              {api.refreshMeta.mode === 'near_live_polling' ? 'Near-live polling (10s)' : 'Background polling (60s)'}
            </div>

            <button
              type="button"
              onClick={() => setCommandOpen(true)}
              className="hidden items-center gap-2 rounded-xl border border-border/70 bg-card/60 px-3 py-1.5 text-xs text-muted-foreground shadow-inset transition-all hover:border-primary/18 hover:bg-accent/72 hover:text-foreground md:inline-flex"
              title="Search commands (Ctrl+K)"
            >
              <Search className="h-3.5 w-3.5" />
              <span>Search...</span>
              <kbd className="ml-1 rounded border border-border/80 bg-muted/60 px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
                {'\u2318'}K
              </kbd>
            </button>

            <button
              type="button"
              onClick={() => void handleRefresh()}
              disabled={refreshBusy}
              className="icon-button"
              aria-label={refreshBusy ? 'Refreshing data' : 'Refresh console data from API'}
              title={refreshBusy ? 'Refreshing...' : 'Re-fetch dashboard data from the API'}
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
            'fixed inset-y-0 left-0 z-40 w-[17rem] transform px-3 pb-3 pt-[4.5rem] transition-transform duration-200 ease-out md:translate-x-0 md:px-3',
            isMobile && isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'
          )}
        >
          <div className="surface-panel surface-panel-strong flex h-full flex-col overflow-hidden">
            <nav className="flex-1 space-y-5 overflow-y-auto px-3 pb-3 pt-4" aria-label="Primary">
              {navGroups.map((group) => (
                <div key={group.label}>
                  <div className="mb-1.5 flex items-center gap-2 px-2">
                    <group.icon className="h-3 w-3 text-muted-foreground/60" aria-hidden />
                    <span className="text-[10px] font-semibold uppercase tracking-[0.2em] text-muted-foreground/70">
                      {group.label}
                    </span>
                  </div>
                  <div className="space-y-0.5">
                    {group.items.map((item) => {
                      const isActive =
                        location.pathname === item.href ||
                        (item.href !== '/' && location.pathname.startsWith(item.href))

                      return (
                        <Link
                          key={item.href}
                          to={item.href}
                          className={clsx(
                            'group flex items-center gap-2.5 rounded-xl border px-2.5 py-2 text-[13px] font-medium outline-none transition-all duration-150 focus-visible:ring-2 focus-visible:ring-ring',
                            isActive
                              ? 'border-primary/22 bg-primary/12 text-foreground shadow-[0_14px_24px_-20px_hsl(var(--primary)/0.55)]'
                              : 'border-transparent text-muted-foreground hover:border-border/60 hover:bg-accent/65 hover:text-foreground'
                          )}
                          aria-current={isActive ? 'page' : undefined}
                        >
                          <span
                            className={clsx(
                              'flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border transition-colors',
                              isActive
                                ? 'border-primary/20 bg-primary/16 text-primary'
                                : 'border-border/60 bg-card/60 text-muted-foreground group-hover:border-primary/12 group-hover:text-foreground'
                            )}
                          >
                            <item.icon className="h-3.5 w-3.5" />
                          </span>
                          <span className="flex-1 truncate">{item.label}</span>
                          {item.badge !== undefined && (
                            <span
                              className={clsx(
                                'flex h-5 min-w-5 items-center justify-center rounded-full px-1.5 text-[10px] font-bold leading-none',
                                item.badgeVariant === 'critical'
                                  ? 'bg-critical/15 text-critical'
                                  : item.badgeVariant === 'warning'
                                    ? 'bg-warning/15 text-warning'
                                    : 'bg-primary/15 text-primary'
                              )}
                            >
                              {item.badge}
                            </span>
                          )}
                        </Link>
                      )
                    })}
                  </div>
                </div>
              ))}
            </nav>

            <div className="border-t border-border/60 p-2.5">
              <Link
                to="/settings"
                className={clsx(
                  'group flex items-center gap-2.5 rounded-xl border px-2.5 py-2 text-[13px] font-medium outline-none transition-all duration-150 focus-visible:ring-2 focus-visible:ring-ring',
                  location.pathname === '/settings'
                    ? 'border-primary/22 bg-primary/12 text-foreground'
                    : 'border-transparent text-muted-foreground hover:border-border/60 hover:bg-accent/65 hover:text-foreground'
                )}
              >
                <span
                  className={clsx(
                    'flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border transition-colors',
                    location.pathname === '/settings'
                      ? 'border-primary/20 bg-primary/16 text-primary'
                      : 'border-border/60 bg-card/60 text-muted-foreground group-hover:border-primary/12 group-hover:text-foreground'
                  )}
                >
                  <Settings className="h-3.5 w-3.5" aria-hidden />
                </span>
                Settings &amp; reference
              </Link>
              {!status.loading && instanceId && (
                <div className="mt-2 px-2.5 text-[10px] text-muted-foreground/60" title={instanceId}>
                  <span className="font-mono">{truncateMiddle(instanceId, 28)}</span>
                </div>
              )}
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
          className="flex-1 px-4 pb-10 pt-4 outline-none md:ml-[17rem] md:px-6 md:pt-5"
        >
          <div className="mx-auto w-full max-w-8xl page-enter">{children}</div>
        </main>
      </div>

      {!status.loading && productScope && (
        <footer className="px-4 pb-4 md:pl-[18.5rem] md:pr-6">
          <div className="mx-auto flex max-w-8xl items-center gap-3 px-1 text-[10px] text-muted-foreground/50">
            {productScope && (
              <span>
                <code className="font-mono">{productScope}</code>
              </span>
            )}
          </div>
        </footer>
      )}

      {/* Command Palette */}
      {commandOpen && (
        <CommandPalette onClose={() => setCommandOpen(false)} />
      )}
    </div>
  )
}

function CommandPalette({ onClose }: { onClose: () => void }) {
  const [query, setQuery] = useState('')
  const location = useLocation()

  const allPages = [
    { label: 'Dashboard', href: '/', keywords: 'home overview' },
    { label: 'Status', href: '/status', keywords: 'transport health connection' },
    { label: 'Nodes', href: '/nodes', keywords: 'devices mesh radio' },
    { label: 'Topology', href: '/topology', keywords: 'graph network map' },
    { label: 'Planning', href: '/planning', keywords: 'resilience playbook' },
    { label: 'Messages', href: '/messages', keywords: 'packets traffic' },
    { label: 'Dead letters', href: '/dead-letters', keywords: 'failed errors' },
    { label: 'Incidents', href: '/incidents', keywords: 'alerts disruptions' },
    { label: 'Control actions', href: '/control-actions', keywords: 'approve reject execute' },
    { label: 'Recommendations', href: '/recommendations', keywords: 'suggestions optimize' },
    { label: 'Events', href: '/events', keywords: 'audit log history' },
    { label: 'Diagnostics', href: '/diagnostics', keywords: 'health checks findings' },
    { label: 'Privacy', href: '/privacy', keywords: 'security audit compliance' },
    { label: 'Settings', href: '/settings', keywords: 'config preferences reference' },
  ]

  const lowerQuery = query.toLowerCase()
  const filtered = query
    ? allPages.filter(
        (p) =>
          p.label.toLowerCase().includes(lowerQuery) ||
          p.keywords.includes(lowerQuery) ||
          p.href.includes(lowerQuery)
      )
    : allPages

  useEffect(() => {
    onClose()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname])

  return (
    <div className="fixed inset-0 z-[200] flex items-start justify-center pt-[15vh]" onClick={onClose}>
      <div className="fixed inset-0 bg-background/60 backdrop-blur-sm" />
      <div
        className="relative z-10 w-full max-w-lg animate-slide-up overflow-hidden rounded-2xl border border-border/80 bg-card shadow-[0_24px_64px_-16px_hsl(var(--shell-shadow)/0.7)]"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-label="Command palette"
      >
        <div className="flex items-center gap-3 border-b border-border/60 px-4 py-3">
          <Search className="h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Go to page..."
            className="flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground/60"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            autoFocus
          />
          <kbd className="rounded border border-border/80 bg-muted/50 px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
            Esc
          </kbd>
        </div>
        <div className="max-h-[40vh] overflow-y-auto p-2">
          {filtered.length === 0 ? (
            <div className="px-3 py-6 text-center text-sm text-muted-foreground">
              No matching pages
            </div>
          ) : (
            filtered.map((page) => (
              <Link
                key={page.href}
                to={page.href}
                className="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm text-foreground transition-colors hover:bg-accent/70 focus:bg-accent/70 focus:outline-none"
                onClick={onClose}
              >
                <span className="flex-1">{page.label}</span>
                {location.pathname === page.href && (
                  <span className="text-[10px] text-muted-foreground">current</span>
                )}
              </Link>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
