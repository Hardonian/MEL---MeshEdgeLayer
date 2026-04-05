import { useState, useEffect, useCallback } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { clsx } from 'clsx'
import { useApi, useStatus } from '@/hooks/useApi'
import { useOperatorWorkspaceFocus } from '@/hooks/useOperatorWorkspaceFocus'
import { HelpMenu, useGlobalKeyboardShortcuts } from '@/components/ui/HelpMenu'
import { truncateMiddle } from '@/utils/presentation'
import { formatRelativeTime } from '@/types/api'
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
  Crosshair,
  ClipboardList,
  Terminal,
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
  const status = useStatus()
  const api = useApi()
  const { focus, clearFocus } = useOperatorWorkspaceFocus()
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
        { label: 'Command surface', href: '/', icon: LayoutDashboard },
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
          label: 'Op review',
          href: '/operational-review',
          icon: ClipboardList,
        },
        {
          label: 'Recommendations',
          href: '/recommendations',
          icon: Activity,
          badge: recsCount > 0 ? recsCount : undefined,
        },
      ],
    },
    {
      label: 'Trust',
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
      if (e.key === 'Escape') {
        setIsMobileMenuOpen(false)
        setCommandOpen(false)
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setCommandOpen((prev) => !prev)
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [])

  const transportStatusLabel = status.loading
    ? 'Loading...'
    : !hasTransports
      ? 'No transport'
      : hasConnectedTransport
        ? 'Connected'
        : 'Disconnected'

  const transportDotClasses = clsx(
    'status-dot',
    status.loading && 'bg-muted-foreground text-muted-foreground',
    !status.loading && !hasTransports && 'bg-muted-foreground text-muted-foreground',
    !status.loading && hasTransports && hasConnectedTransport && 'animate-pulse-slow bg-success text-success',
    !status.loading && hasTransports && !hasConnectedTransport && 'bg-warning text-warning'
  )

  return (
    <div className="min-h-screen bg-background">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[100] focus:rounded-md focus:bg-background focus:px-3 focus:py-2 focus:text-xs focus:shadow-panel focus:outline-none focus:ring-2 focus:ring-ring"
      >
        Skip to main content
      </a>

      {/* ─── Top Chrome Bar ─── */}
      <header className="sticky top-0 z-50 border-b border-chrome-border bg-chrome-bg/95 backdrop-blur-md">
        <div className="mx-auto flex h-11 max-w-[96rem] items-center justify-between px-3">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              className="icon-button md:hidden"
              aria-label="Toggle menu"
            >
              {isMobileMenuOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
            </button>

            <Link
              to="/"
              className="group flex items-center gap-2.5 outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <div className="flex h-7 w-7 items-center justify-center rounded-md border border-primary/25 bg-primary/12 text-primary">
                <Radio className="h-3.5 w-3.5" />
              </div>
              <div className="flex items-center gap-2">
                <span className="font-display text-sm font-bold tracking-tight text-foreground">MEL</span>
                <span className="hidden text-mel-xs text-muted-foreground/60 sm:inline">//</span>
                <span className="hidden font-mono text-mel-xs text-muted-foreground/60 sm:inline">MeshEdgeLayer</span>
              </div>
            </Link>
          </div>

          <div className="flex items-center gap-1.5">
            {/* Transport status */}
            <div className={clsx(
              'status-pill hidden md:inline-flex',
              status.loading && 'border-border/60 text-muted-foreground',
              !status.loading && !hasTransports && 'border-border/60 text-muted-foreground',
              !status.loading && hasTransports && hasConnectedTransport && 'border-success/25 text-success',
              !status.loading && hasTransports && !hasConnectedTransport && 'border-warning/25 text-warning'
            )}>
              <span className={transportDotClasses} aria-hidden />
              {transportStatusLabel}
            </div>

            {/* Polling mode */}
            <div className="status-pill hidden border-border/50 text-muted-foreground lg:inline-flex">
              {api.refreshMeta.mode === 'near_live_polling' ? '10s poll' : '60s poll'}
            </div>

            {/* Command palette trigger */}
            <button
              type="button"
              onClick={() => setCommandOpen(true)}
              className="hidden items-center gap-1.5 rounded-md border border-border/50 bg-card/50 px-2.5 py-1 font-mono text-mel-xs text-muted-foreground transition-colors hover:border-primary/20 hover:text-foreground md:inline-flex"
              title="Search commands (Ctrl+K)"
            >
              <Search className="h-3 w-3" />
              <span>Search</span>
              <kbd className="ml-1 rounded-sm border border-border/60 bg-muted/50 px-1 py-0.5 font-mono text-[9px]">
                {'\u2318'}K
              </kbd>
            </button>

            <button
              type="button"
              onClick={() => void handleRefresh()}
              disabled={refreshBusy}
              className="icon-button"
              aria-label={refreshBusy ? 'Refreshing data' : 'Refresh console data'}
              title={refreshBusy ? 'Refreshing...' : 'Re-fetch data from API'}
            >
              <RefreshCw className={clsx('h-3.5 w-3.5', refreshBusy && 'animate-spin')} aria-hidden />
            </button>

            <HelpMenu />
          </div>
        </div>
      </header>

      <div className="flex">
        {/* ─── Side Navigation ─── */}
        <aside
          className={clsx(
            'fixed inset-y-0 left-0 z-40 w-[15rem] transform border-r border-chrome-border bg-chrome-bg pt-11 transition-transform duration-200 ease-out md:translate-x-0',
            isMobile && isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'
          )}
        >
          <nav className="flex h-full flex-col overflow-y-auto px-2 pb-2 pt-3" aria-label="Primary">
            <div className="flex-1 space-y-4">
              {navGroups.map((group) => (
                <div key={group.label}>
                  <div className="mb-1 flex items-center gap-1.5 px-2">
                    <group.icon className="h-3 w-3 text-muted-foreground/50" aria-hidden />
                    <span className="mel-label">{group.label}</span>
                  </div>
                  <div className="space-y-px">
                    {group.items.map((item) => {
                      const isActive =
                        location.pathname === item.href ||
                        (item.href !== '/' && location.pathname.startsWith(item.href))

                      return (
                        <Link
                          key={item.href}
                          to={item.href}
                          className={clsx(
                            'group flex items-center gap-2 rounded-md border px-2 py-1.5 text-mel-sm font-medium outline-none transition-colors duration-100 focus-visible:ring-2 focus-visible:ring-ring',
                            isActive
                              ? 'border-primary/20 bg-primary/10 text-foreground'
                              : 'border-transparent text-muted-foreground hover:bg-accent/60 hover:text-foreground'
                          )}
                          aria-current={isActive ? 'page' : undefined}
                        >
                          <item.icon className={clsx(
                            'h-3.5 w-3.5 shrink-0',
                            isActive ? 'text-primary' : 'text-muted-foreground/60 group-hover:text-foreground'
                          )} />
                          <span className="flex-1 truncate">{item.label}</span>
                          {item.badge !== undefined && (
                            <span
                              className={clsx(
                                'flex h-4 min-w-4 items-center justify-center rounded-sm px-1 font-mono text-[9px] font-bold leading-none',
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
            </div>

            {/* Bottom: Settings + Instance ID */}
            <div className="border-t border-border/50 pt-2">
              <Link
                to="/settings"
                className={clsx(
                  'group flex items-center gap-2 rounded-md border px-2 py-1.5 text-mel-sm font-medium outline-none transition-colors duration-100 focus-visible:ring-2 focus-visible:ring-ring',
                  location.pathname === '/settings'
                    ? 'border-primary/20 bg-primary/10 text-foreground'
                    : 'border-transparent text-muted-foreground hover:bg-accent/60 hover:text-foreground'
                )}
              >
                <Settings className={clsx(
                  'h-3.5 w-3.5 shrink-0',
                  location.pathname === '/settings' ? 'text-primary' : 'text-muted-foreground/60'
                )} aria-hidden />
                Settings
              </Link>
              {!status.loading && instanceId && (
                <div className="mt-1.5 px-2 font-mono text-[9px] text-muted-foreground/40" title={instanceId}>
                  {truncateMiddle(instanceId, 28)}
                </div>
              )}
            </div>
          </nav>
        </aside>

        {isMobile && isMobileMenuOpen && (
          <div
            className="fixed inset-0 z-30 bg-background/80 backdrop-blur-sm md:hidden"
            onClick={() => setIsMobileMenuOpen(false)}
          />
        )}

        {/* ─── Main Content ─── */}
        <main
          id="main-content"
          tabIndex={-1}
          className="flex-1 px-4 pb-8 pt-4 outline-none md:ml-[15rem] md:px-5 md:pt-4"
        >
          <WorkspaceFocusBar
            pathname={location.pathname}
            focus={focus}
            onDismiss={clearFocus}
          />
          <TruthContractStrip />
          <div className="mx-auto w-full max-w-8xl page-enter">{children}</div>
        </main>
      </div>

      {!status.loading && productScope && (
        <footer className="px-4 pb-3 md:pl-[16rem] md:pr-5">
          <div className="mx-auto flex max-w-8xl items-center gap-2 px-1 font-mono text-[9px] text-muted-foreground/35">
            <Terminal className="h-3 w-3" aria-hidden />
            <code>{productScope}</code>
          </div>
        </footer>
      )}

      {commandOpen && (
        <CommandPalette onClose={() => setCommandOpen(false)} />
      )}
    </div>
  )
}

function TruthContractStrip() {
  return (
    <div
      className="mb-4 flex flex-col gap-2 rounded-md border border-border/50 bg-muted/20 px-3 py-2 text-mel-xs leading-relaxed text-muted-foreground sm:flex-row sm:flex-wrap sm:items-center sm:gap-x-3 sm:gap-y-1"
      role="note"
      aria-label="Operator truth contract"
    >
      <span className="mel-label text-foreground/80">Truth contract</span>
      <span className="hidden sm:inline text-muted-foreground/40" aria-hidden>//</span>
      <span className="prose-body text-mel-xs">
        <strong className="font-medium text-foreground/80">Live</strong> = recent persisted ingest.{' '}
        <strong className="font-medium text-foreground/80">Stale / partial / degraded</strong> stay visible.
      </span>
      <span className="hidden sm:inline text-muted-foreground/40" aria-hidden>//</span>
      <span className="prose-body text-mel-xs">
        Topology shows <strong className="font-medium text-foreground/80">observed context</strong>, not proof of RF path.
      </span>
      <Link
        to="/settings#operator-truth-contract"
        className="font-mono text-mel-xs font-semibold text-primary hover:underline sm:ml-auto"
      >
        Full wording →
      </Link>
    </div>
  )
}

function WorkspaceFocusBar({
  pathname,
  focus,
  onDismiss,
}: {
  pathname: string
  focus: { incidentId: string; incidentTitle?: string; savedAt: string } | null
  onDismiss: () => void
}) {
  if (!focus) return null
  const onIncidentPage =
    pathname === `/incidents/${focus.incidentId}` || pathname.startsWith(`/incidents/${focus.incidentId}/`)
  if (onIncidentPage) return null

  return (
    <div
      className="mb-4 flex flex-wrap items-center gap-2 rounded-md border border-primary/20 bg-primary/5 px-3 py-2 text-mel-sm"
      role="region"
      aria-label="Current workspace focus"
    >
      <Crosshair className="h-3.5 w-3.5 shrink-0 text-primary" aria-hidden />
      <span className="font-semibold text-foreground shrink-0">Focus:</span>
      <Link
        to={`/incidents/${encodeURIComponent(focus.incidentId)}`}
        className="min-w-0 flex-1 font-mono font-medium text-primary hover:underline truncate sm:flex-none"
      >
        {focus.incidentTitle?.trim() || focus.incidentId.slice(0, 14)}
      </Link>
      <span className="font-mono text-mel-xs text-muted-foreground/60 hidden sm:inline">
        set {formatRelativeTime(focus.savedAt)}
      </span>
      <div className="flex flex-wrap items-center gap-1 sm:ml-auto">
        <Link
          to={`/incidents/${encodeURIComponent(focus.incidentId)}?replay=1`}
          className="rounded-sm border border-border/50 bg-card/50 px-2 py-0.5 font-mono text-mel-xs font-semibold text-muted-foreground hover:text-foreground"
        >
          Replay
        </Link>
        <Link
          to={`/topology?incident=${encodeURIComponent(focus.incidentId)}&filter=incident_focus`}
          className="rounded-sm border border-border/50 bg-card/50 px-2 py-0.5 font-mono text-mel-xs font-semibold text-muted-foreground hover:text-foreground"
        >
          Topology
        </Link>
        <Link
          to={`/incidents/${encodeURIComponent(focus.incidentId)}#shift-continuity-handoff`}
          className="rounded-sm border border-border/50 bg-card/50 px-2 py-0.5 font-mono text-mel-xs font-semibold text-muted-foreground hover:text-foreground"
        >
          Handoff
        </Link>
        <button
          type="button"
          onClick={onDismiss}
          className="inline-flex items-center gap-1 rounded-sm border border-border/50 px-2 py-0.5 font-mono text-mel-xs font-semibold text-muted-foreground hover:bg-muted/40 hover:text-foreground"
          aria-label="Clear workspace focus"
        >
          <X className="h-3 w-3" aria-hidden />
          Clear
        </button>
      </div>
    </div>
  )
}

function CommandPalette({ onClose }: { onClose: () => void }) {
  const [query, setQuery] = useState('')
  const location = useLocation()

  const allPages = [
    { label: 'Command surface', href: '/', keywords: 'home overview dashboard operator workspace' },
    { label: 'Instance briefing', href: '/#mel-instance-briefing', keywords: 'briefing intelligence priorities digest' },
    { label: 'Status', href: '/status', keywords: 'transport health connection' },
    { label: 'Nodes', href: '/nodes', keywords: 'devices mesh radio' },
    { label: 'Topology', href: '/topology', keywords: 'graph network map' },
    { label: 'Planning', href: '/planning', keywords: 'resilience playbook' },
    { label: 'Operational review', href: '/operational-review', keywords: 'digest shift summary' },
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
      <div className="fixed inset-0 bg-background/70 backdrop-blur-sm" />
      <div
        className="relative z-10 w-full max-w-lg animate-slide-up overflow-hidden rounded-md border border-border bg-card shadow-float"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-label="Command palette"
      >
        <div className="flex items-center gap-2 border-b border-border/60 px-3 py-2.5">
          <Terminal className="h-3.5 w-3.5 text-primary" />
          <input
            type="text"
            placeholder="Go to page..."
            className="flex-1 bg-transparent font-mono text-mel-sm text-foreground outline-none placeholder:text-muted-foreground/50"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            autoFocus
          />
          <kbd className="rounded-sm border border-border/60 bg-muted/40 px-1.5 py-0.5 font-mono text-[9px] text-muted-foreground">
            Esc
          </kbd>
        </div>
        <div className="max-h-[40vh] overflow-y-auto p-1.5">
          {filtered.length === 0 ? (
            <div className="px-3 py-6 text-center font-mono text-mel-sm text-muted-foreground">
              No matching pages
            </div>
          ) : (
            filtered.map((page) => (
              <Link
                key={page.href}
                to={page.href}
                className="flex items-center gap-2.5 rounded-sm px-2.5 py-2 font-mono text-mel-sm text-foreground transition-colors hover:bg-accent/60 focus:bg-accent/60 focus:outline-none"
                onClick={onClose}
              >
                <span className="text-muted-foreground/50">&gt;</span>
                <span className="flex-1">{page.label}</span>
                {location.pathname === page.href && (
                  <span className="font-mono text-[9px] text-primary">current</span>
                )}
              </Link>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
