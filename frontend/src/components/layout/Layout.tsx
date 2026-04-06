import { useState, useEffect, useCallback } from 'react'
import { Link, useLocation, matchPath } from 'react-router-dom'
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

  const deadLetterCount = api.deadLetters.data?.length ?? 0
  const diagCount = api.diagnostics.data?.filter((d) => d.severity === 'critical' || d.severity === 'high').length ?? 0
  const privacyCount = api.privacyFindings.data?.filter((p) => p.severity === 'critical' || p.severity === 'high').length ?? 0
  const recsCount = api.recommendations.data?.filter((r) => r.actionable).length ?? 0

  const navGroups: NavGroup[] = [
    {
      label: 'overview',
      icon: Eye,
      items: [
        { label: 'console', href: '/', icon: LayoutDashboard },
        { label: 'status', href: '/status', icon: Activity },
        { label: 'events', href: '/events', icon: FileText },
      ],
    },
    {
      label: 'mesh',
      icon: Radio,
      items: [
        { label: 'nodes', href: '/nodes', icon: Radio },
        { label: 'topology', href: '/topology', icon: GitBranch },
        { label: 'messages', href: '/messages', icon: MessageSquare },
        {
          label: 'dead-letters',
          href: '/dead-letters',
          icon: Inbox,
          badge: deadLetterCount > 0 ? deadLetterCount : undefined,
          badgeVariant: 'warning' as const,
        },
      ],
    },
    {
      label: 'ops',
      icon: Wrench,
      items: [
        { label: 'incidents', href: '/incidents', icon: AlertTriangle },
        { label: 'control', href: '/control-actions', icon: Zap },
        { label: 'planning', href: '/planning', icon: Compass },
        { label: 'review', href: '/operational-review', icon: ClipboardList },
        {
          label: 'recommendations',
          href: '/recommendations',
          icon: Activity,
          badge: recsCount > 0 ? recsCount : undefined,
        },
      ],
    },
    {
      label: 'trust',
      icon: Shield,
      items: [
        {
          label: 'diagnostics',
          href: '/diagnostics',
          icon: Shield,
          badge: diagCount > 0 ? diagCount : undefined,
          badgeVariant: diagCount > 0 ? 'critical' as const : undefined,
        },
        {
          label: 'privacy',
          href: '/privacy',
          icon: Shield,
          badge: privacyCount > 0 ? privacyCount : undefined,
          badgeVariant: privacyCount > 0 ? 'critical' as const : undefined,
        },
      ],
    },
  ]

  useEffect(() => {
    const checkMobile = () => setIsMobile(window.innerWidth < 768)
    checkMobile()
    window.addEventListener('resize', checkMobile)
    return () => window.removeEventListener('resize', checkMobile)
  }, [])

  useEffect(() => {
    setIsMobileMenuOpen(false)
  }, [location.pathname])

  const handleRefresh = useCallback(async () => {
    setRefreshBusy(true)
    try { await refreshAll() } finally { setRefreshBusy(false) }
  }, [refreshAll])

  useGlobalKeyboardShortcuts(handleRefresh)

  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === 'visible') void refreshAll()
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => document.removeEventListener('visibilitychange', onVisible)
  }, [refreshAll])

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { setIsMobileMenuOpen(false); setCommandOpen(false) }
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') { e.preventDefault(); setCommandOpen((p) => !p) }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [])

  const transportLabel = status.loading
    ? 'LOADING'
    : !hasTransports
      ? 'NO_TRANSPORT'
      : hasConnectedTransport
        ? 'CONNECTED'
        : 'DISCONNECTED'

  const transportColor = status.loading
    ? 'text-muted-foreground'
    : !hasTransports
      ? 'text-muted-foreground'
      : hasConnectedTransport
        ? 'text-neon'
        : 'text-neon-warn'

  return (
    <div className="min-h-screen bg-background mel-grid-bg">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[100] focus:bg-background focus:px-2 focus:py-1 focus:text-mel-sm focus:outline-none focus:ring-1 focus:ring-ring"
      >
        Skip to main content
      </a>

      {/* ╔══ TOP STATUS BAR — tmux style ══╗ */}
      <header className="sticky top-0 z-50 border-b border-chrome-border bg-chrome-bg">
        <div className="mx-auto flex h-8 max-w-[96rem] items-center justify-between px-3">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              className="icon-button md:hidden"
              aria-label="Toggle menu"
            >
              {isMobileMenuOpen ? <X className="h-3 w-3" /> : <Menu className="h-3 w-3" />}
            </button>

            <Link
              to="/"
              className="group flex items-center gap-2 outline-none focus-visible:ring-1 focus-visible:ring-ring"
            >
              <span className="font-data text-neon font-bold tracking-tight text-sm">MEL</span>
              <span className="text-muted-foreground/40">│</span>
              <span className="text-mel-xs text-muted-foreground/50 hidden sm:inline">mesh::edge::layer</span>
            </Link>
          </div>

          <div className="flex items-center gap-2">
            {/* Transport status */}
            <span className={clsx('text-mel-xs font-bold hidden md:inline', transportColor)}>
              [{transportLabel}]
            </span>

            {/* Poll rate */}
            <span className="text-mel-xs text-muted-foreground hidden lg:inline">
              {api.refreshMeta.mode === 'near_live_polling' ? 'poll:10s' : 'poll:60s'}
            </span>

            <span className="text-muted-foreground/30 hidden md:inline">│</span>

            {/* Command palette */}
            <button
              type="button"
              onClick={() => setCommandOpen(true)}
              className="hidden md:inline-flex items-center gap-1 text-mel-xs text-muted-foreground hover:text-neon transition-colors"
              title="Search console routes (Ctrl+K)"
            >
              <Search className="h-3 w-3" />
              <span>ctrl+k · search</span>
            </button>

            <button
              type="button"
              onClick={() => void handleRefresh()}
              disabled={refreshBusy}
              className="icon-button"
              aria-label="Refresh"
            >
              <RefreshCw className={clsx('h-3 w-3', refreshBusy && 'animate-spin')} aria-hidden />
            </button>

            <HelpMenu />
          </div>
        </div>
      </header>

      <div className="flex">
        {/* ╔══ SIDE NAV — file tree style ══╗ */}
        <aside
          className={clsx(
            'fixed inset-y-0 left-0 z-40 w-[13rem] transform border-r border-chrome-border bg-chrome-bg pt-8 transition-transform duration-150 ease-out md:translate-x-0',
            isMobile && isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'
          )}
        >
          <nav className="flex h-full flex-col overflow-y-auto px-1.5 pb-2 pt-2" aria-label="Primary">
            <div className="flex-1 space-y-3">
              {navGroups.map((group) => (
                <div key={group.label}>
                  <div className="mb-0.5 px-2 text-mel-xs font-bold uppercase tracking-[0.16em] text-muted-foreground/50">
                    <span className="text-neon/40">▸</span> {group.label}/
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
                            'group flex items-center gap-1.5 border-l-2 px-2 py-1 text-mel-sm font-medium outline-none transition-colors duration-75 focus-visible:ring-1 focus-visible:ring-ring',
                            isActive
                              ? 'border-neon bg-neon/6 text-neon'
                              : 'border-transparent text-muted-foreground hover:border-muted-foreground/30 hover:text-foreground'
                          )}
                          aria-current={isActive ? 'page' : undefined}
                        >
                          <item.icon className={clsx(
                            'h-3 w-3 shrink-0',
                            isActive ? 'text-neon' : 'text-muted-foreground/40'
                          )} />
                          <span className="flex-1 truncate">{item.label}</span>
                          {item.badge !== undefined && (
                            <span
                              className={clsx(
                                'text-mel-xs font-bold',
                                item.badgeVariant === 'critical'
                                  ? 'text-neon-hot'
                                  : item.badgeVariant === 'warning'
                                    ? 'text-neon-warn'
                                    : 'text-neon'
                              )}
                            >
                              [{item.badge}]
                            </span>
                          )}
                        </Link>
                      )
                    })}
                  </div>
                </div>
              ))}
            </div>

            <div className="border-t border-border/50 pt-1.5">
              <Link
                to="/settings"
                className={clsx(
                  'group flex items-center gap-1.5 border-l-2 px-2 py-1 text-mel-sm font-medium outline-none transition-colors duration-75 focus-visible:ring-1 focus-visible:ring-ring',
                  location.pathname === '/settings'
                    ? 'border-neon bg-neon/6 text-neon'
                    : 'border-transparent text-muted-foreground hover:border-muted-foreground/30 hover:text-foreground'
                )}
              >
                <Settings className="h-3 w-3 shrink-0 text-muted-foreground/40" aria-hidden />
                settings
              </Link>
              {!status.loading && instanceId && (
                <div className="mt-1 px-2 text-[8px] text-muted-foreground/30 truncate" title={instanceId}>
                  {truncateMiddle(instanceId, 24)}
                </div>
              )}
            </div>
          </nav>
        </aside>

        {isMobile && isMobileMenuOpen && (
          <div
            className="fixed inset-0 z-30 bg-background/90 md:hidden"
            onClick={() => setIsMobileMenuOpen(false)}
          />
        )}

        {/* ╔══ MAIN CONTENT AREA ══╗ */}
        <main
          id="main-content"
          tabIndex={-1}
          className="flex-1 px-3 pb-6 pt-3 outline-none md:ml-[13rem] md:px-4 md:pt-3"
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

      {/* ╔══ BOTTOM STATUS LINE ══╗ */}
      {!status.loading && productScope && (
        <footer className="fixed bottom-0 left-0 right-0 z-40 border-t border-chrome-border bg-chrome-bg px-3 md:pl-[13.5rem]">
          <div className="mx-auto flex h-6 max-w-8xl items-center gap-3 text-[9px] text-muted-foreground/40">
            <span className="text-neon/40">$</span>
            <code>{productScope}</code>
            <span className="ml-auto">MEL::MeshEdgeLayer</span>
            <span className="text-neon/50" aria-hidden>
              ▌
            </span>
          </div>
        </footer>
      )}

      {commandOpen && <CommandPalette onClose={() => setCommandOpen(false)} />}
    </div>
  )
}

function TruthContractStrip() {
  return (
    <div
      className="mb-3 border border-border bg-panel-muted px-3 py-1.5 text-mel-xs text-muted-foreground flex flex-col gap-1 sm:flex-row sm:flex-wrap sm:items-center sm:gap-x-2"
      role="note"
      aria-label="Operator truth contract"
    >
      <span className="text-neon font-bold">[TRUTH]</span>
      <span>
        <strong className="text-foreground/80">Live</strong> = recent persisted ingest.{' '}
        <strong className="text-foreground/80">Stale/partial/degraded</strong> stay visible.
      </span>
      <span className="text-muted-foreground/30 hidden sm:inline">│</span>
      <span>
        Topology = <strong className="text-foreground/80">observed context</strong>, not RF proof.
      </span>
      <Link
        to="/settings#operator-truth-contract"
        className="text-mel-xs font-bold text-neon hover:underline sm:ml-auto"
      >
        → full contract
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
      className="mb-3 flex flex-wrap items-center gap-2 border border-neon/20 bg-neon/4 px-3 py-1.5 text-mel-sm"
      role="region"
      aria-label="Current workspace focus"
    >
      <Crosshair className="h-3 w-3 shrink-0 text-neon" aria-hidden />
      <span className="font-bold text-neon">[FOCUS]</span>
      <Link
        to={`/incidents/${encodeURIComponent(focus.incidentId)}`}
        className="font-bold text-neon hover:underline truncate"
      >
        {focus.incidentTitle?.trim() || focus.incidentId.slice(0, 14)}
      </Link>
      <span className="text-mel-xs text-muted-foreground/50 hidden sm:inline">
        set {formatRelativeTime(focus.savedAt)}
      </span>
      <div className="flex flex-wrap items-center gap-1 sm:ml-auto">
        {(['Replay', 'Topology', 'Handoff'] as const).map((action) => {
          const hrefs = {
            Replay: `/incidents/${encodeURIComponent(focus.incidentId)}?replay=1`,
            Topology: `/topology?incident=${encodeURIComponent(focus.incidentId)}&filter=incident_focus`,
            Handoff: `/incidents/${encodeURIComponent(focus.incidentId)}#shift-continuity-handoff`,
          }
          return (
            <Link
              key={action}
              to={hrefs[action]}
              className="border border-border bg-card px-1.5 py-0.5 text-mel-xs font-bold text-muted-foreground hover:text-neon hover:border-neon/30 transition-colors"
            >
              {action.toLowerCase()}
            </Link>
          )
        })}
        <button
          type="button"
          onClick={onDismiss}
          className="inline-flex items-center gap-0.5 border border-border px-1.5 py-0.5 text-mel-xs font-bold text-muted-foreground hover:text-neon-hot hover:border-neon-hot/30 transition-colors"
          aria-label="Clear workspace focus"
        >
          <X className="h-2.5 w-2.5" aria-hidden />
          clear
        </button>
      </div>
    </div>
  )
}

type PaletteLink = {
  group: string
  label: string
  href: string
  keywords: string
}

function paletteHrefMatchesLocation(href: string): boolean {
  try {
    const u = new URL(href, window.location.origin)
    const cur = new URL(window.location.href)
    return u.pathname === cur.pathname && u.search === cur.search && u.hash === cur.hash
  } catch {
    return false
  }
}

function CommandPalette({ onClose }: { onClose: () => void }) {
  const [query, setQuery] = useState('')
  const location = useLocation()

  const incidentMatch = matchPath({ path: '/incidents/:id', end: true }, location.pathname)
  const incidentId = incidentMatch?.params.id

  const contextLinks: PaletteLink[] = incidentId
    ? [
        {
          group: 'This incident',
          label: 'Replay / timeline',
          href: `/incidents/${encodeURIComponent(incidentId)}?replay=1`,
          keywords: 'replay timeline chronology history',
        },
        {
          group: 'This incident',
          label: 'Operational summary (anchor)',
          href: `/incidents/${encodeURIComponent(incidentId)}#incident-operational-summary`,
          keywords: 'summary header severity state',
        },
        {
          group: 'This incident',
          label: 'Decision pack (anchor)',
          href: `/incidents/${encodeURIComponent(incidentId)}#mel-incident-decision-pack`,
          keywords: 'decision pack adjudication guidance',
        },
        {
          group: 'This incident',
          label: 'Handoff & export (anchor)',
          href: `/incidents/${encodeURIComponent(incidentId)}#shift-continuity-handoff`,
          keywords: 'handoff escalation continuity export json',
        },
        {
          group: 'This incident',
          label: 'Linked control actions (anchor)',
          href: `/incidents/${encodeURIComponent(incidentId)}#linked-control-actions`,
          keywords: 'control queue approval execution linked',
        },
        {
          group: 'This incident',
          label: 'Topology (incident focus)',
          href: `/topology?incident=${encodeURIComponent(incidentId)}&filter=incident_focus`,
          keywords: 'topology graph map focus',
        },
        {
          group: 'This incident',
          label: 'Control queue (filtered)',
          href: `/control-actions?incident=${encodeURIComponent(incidentId)}`,
          keywords: 'approve reject pending',
        },
      ]
    : []

  const allPages: PaletteLink[] = [
    { group: 'Console', label: 'console', href: '/', keywords: 'home overview operator workspace workbench' },
    { group: 'Console', label: 'briefing', href: '/#mel-instance-briefing', keywords: 'briefing intelligence priorities' },
    { group: 'Console', label: 'status', href: '/status', keywords: 'transport health' },
    { group: 'Console', label: 'nodes', href: '/nodes', keywords: 'devices mesh radio' },
    { group: 'Console', label: 'topology', href: '/topology', keywords: 'graph network map' },
    { group: 'Console', label: 'planning', href: '/planning', keywords: 'resilience playbook' },
    { group: 'Console', label: 'operational-review', href: '/operational-review', keywords: 'digest shift' },
    { group: 'Console', label: 'messages', href: '/messages', keywords: 'packets traffic' },
    { group: 'Console', label: 'dead-letters', href: '/dead-letters', keywords: 'failed errors' },
    { group: 'Console', label: 'incidents', href: '/incidents', keywords: 'alerts disruptions queue' },
    { group: 'Console', label: 'control-actions', href: '/control-actions', keywords: 'approve reject' },
    { group: 'Console', label: 'recommendations', href: '/recommendations', keywords: 'suggestions' },
    { group: 'Console', label: 'events', href: '/events', keywords: 'audit log' },
    { group: 'Console', label: 'diagnostics', href: '/diagnostics', keywords: 'health checks support bundle' },
    { group: 'Console', label: 'privacy', href: '/privacy', keywords: 'security audit' },
    { group: 'Console', label: 'settings', href: '/settings', keywords: 'config prefs' },
  ]

  const combined = [...contextLinks, ...allPages]

  const lowerQuery = query.toLowerCase().trim()
  const filtered = lowerQuery
    ? combined.filter(
        (p) =>
          p.label.toLowerCase().includes(lowerQuery) ||
          p.keywords.includes(lowerQuery) ||
          p.href.toLowerCase().includes(lowerQuery) ||
          p.group.toLowerCase().includes(lowerQuery),
      )
    : combined

  useEffect(() => {
    onClose()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname])

  return (
    <div className="fixed inset-0 z-[200] flex items-start justify-center pt-[12vh]" onClick={onClose}>
      <div className="fixed inset-0 bg-background/85" />
      <div
        className="relative z-10 w-full max-w-lg animate-slide-up overflow-hidden border border-border/80 bg-card shadow-float"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-label="Command palette"
      >
        <div className="flex items-center gap-2 border-b border-border px-3 py-2">
          <Search className="h-3.5 w-3.5 shrink-0 text-muted-foreground/50" aria-hidden />
          <input
            type="text"
            placeholder="Jump to a surface or incident anchor"
            className="flex-1 bg-transparent text-mel-sm text-foreground outline-none placeholder:text-muted-foreground/50 caret-primary"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            autoFocus
            aria-label="Command palette filter"
          />
          <span className="text-mel-xs text-muted-foreground/40">esc</span>
        </div>
        <div className="max-h-[40vh] overflow-y-auto p-1">
          {filtered.length === 0 ? (
            <div className="px-3 py-4 text-center text-mel-sm text-muted-foreground">
              No matching jump target.
            </div>
          ) : (
            filtered.map((page, i) => (
              <Link
                key={`${page.group}:${page.href}:${i}`}
                to={page.href}
                className="flex items-center gap-2 px-2 py-1.5 text-mel-sm text-foreground transition-colors hover:bg-muted/60 hover:text-foreground focus:bg-muted/60 focus:text-foreground focus:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                onClick={onClose}
              >
                <span className="text-muted-foreground/40">→</span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate">{page.label}</span>
                  {page.group !== 'Console' && (
                    <span className="block truncate text-mel-xs text-muted-foreground/70">{page.group}</span>
                  )}
                </span>
                {paletteHrefMatchesLocation(page.href) ? (
                  <span className="text-[8px] shrink-0 font-bold uppercase tracking-wide text-primary">here</span>
                ) : null}
              </Link>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
