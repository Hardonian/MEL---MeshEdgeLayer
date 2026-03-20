import { ReactNode, useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { clsx } from 'clsx'
import {
  LayoutDashboard,
  Activity,
  Radio,
  MessageSquare,
  Shield,
  AlertTriangle,
  Lightbulb,
  ScrollText,
  Settings,
  Menu,
  RefreshCw,
  Wifi,
  WifiOff,
  ChevronRight,
} from 'lucide-react'
import { useApi } from '@/hooks/useApi'

interface LayoutProps {
  children: ReactNode
}

const navItems = [
  { path: '/', label: 'Dashboard', icon: LayoutDashboard, description: 'System overview' },
  { path: '/status', label: 'Status', icon: Activity, description: 'Transport health' },
  { path: '/nodes', label: 'Nodes', icon: Radio, description: 'Device inventory' },
  { path: '/messages', label: 'Messages', icon: MessageSquare, description: 'Mesh messages' },
  { path: '/privacy', label: 'Privacy', icon: Shield, description: 'Security posture' },
  { path: '/dead-letters', label: 'Dead Letters', icon: AlertTriangle, description: 'Failed messages' },
  { path: '/recommendations', label: 'Recommendations', icon: Lightbulb, description: 'Action items' },
  { path: '/events', label: 'Events', icon: ScrollText, description: 'Audit logs' },
  { path: '/settings', label: 'Settings', icon: Settings, description: 'Configuration' },
]

export function Layout({ children }: LayoutProps) {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const location = useLocation()
  const { refreshAll, status } = useApi()

  const isConnected = status.data?.transports?.some(t => t.effective_state === 'connected') ?? false
  const currentPage = navItems.find(n => n.path === location.pathname)

  return (
    <div className="min-h-screen bg-background">
      {/* Mobile sidebar backdrop */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-background/80 backdrop-blur-sm lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={clsx(
          'fixed top-0 left-0 z-50 h-full w-64 border-r bg-card transition-transform duration-200 ease-in-out lg:translate-x-0',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        {/* Logo */}
        <div className="flex h-16 items-center gap-2 border-b px-4">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary">
            <Radio className="h-5 w-5 text-primary-foreground" />
          </div>
          <div className="flex flex-col">
            <span className="font-semibold text-foreground">MEL</span>
            <span className="text-xs text-muted-foreground">MeshEdgeLayer</span>
          </div>
        </div>

        {/* Navigation */}
        <nav className="space-y-1 p-3">
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              onClick={() => setSidebarOpen(false)}
              className={({ isActive }) =>
                clsx(
                  'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all',
                  isActive
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                )
              }
            >
              <item.icon className="h-5 w-5 shrink-0" />
              <span className="flex-1">{item.label}</span>
            </NavLink>
          ))}
        </nav>

        {/* Connection status */}
        <div className="absolute bottom-0 left-0 right-0 border-t bg-card/50 p-4 backdrop-blur">
          <div className="flex items-center gap-3">
            <div className={clsx(
              'flex h-8 w-8 shrink-0 items-center justify-center rounded-full',
              isConnected ? 'bg-success/10' : 'bg-muted'
            )}>
              {isConnected ? (
                <Wifi className="h-4 w-4 text-success" />
              ) : (
                <WifiOff className="h-4 w-4 text-muted-foreground" />
              )}
            </div>
            <div className="flex-1 min-w-0">
              <p className={clsx(
                'text-sm font-medium',
                isConnected ? 'text-success' : 'text-muted-foreground'
              )}>
                {isConnected ? 'Connected' : 'Disconnected'}
              </p>
              <p className="text-xs text-muted-foreground truncate">
                {status.data?.transports?.length || 0} transport{status.data?.transports?.length !== 1 ? 's' : ''} configured
              </p>
            </div>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="lg:pl-64">
        {/* Top header */}
        <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b bg-background/95 px-4 backdrop-blur supports-[backdrop-filter]:bg-background/60 lg:px-6">
          {/* Mobile menu button */}
          <div className="flex items-center gap-3">
            <button
              onClick={() => setSidebarOpen(true)}
              className="rounded-lg p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground lg:hidden"
              aria-label="Open menu"
            >
              <Menu className="h-5 w-5" />
            </button>

            {/* Breadcrumb */}
            <nav className="hidden sm:flex items-center gap-1.5 text-sm">
              {location.pathname !== '/' && (
                <>
                  <NavLink 
                    to="/" 
                    className="text-muted-foreground hover:text-foreground transition-colors"
                  >
                    Home
                  </NavLink>
                  <ChevronRight className="h-4 w-4 text-muted-foreground/50" />
                  <span className="font-medium text-foreground">
                    {currentPage?.label || 'Page'}
                  </span>
                </>
              )}
              {location.pathname === '/' && (
                <span className="font-medium text-foreground">Dashboard</span>
              )}
            </nav>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-2">
            <span className="hidden md:block text-xs text-muted-foreground mr-2">
              Last updated: {status.lastUpdated ? new Date(status.lastUpdated).toLocaleTimeString() : '—'}
            </span>
            <button
              onClick={() => refreshAll()}
              className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              title="Refresh all data"
            >
              <RefreshCw className="h-4 w-4" />
              <span className="hidden sm:inline">Refresh</span>
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className="p-4 lg:p-6">
          {children}
        </main>
      </div>
    </div>
  )
}
