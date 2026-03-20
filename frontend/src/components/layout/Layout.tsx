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
} from 'lucide-react'
import { useApi } from '@/hooks/useApi'

interface LayoutProps {
  children: ReactNode
}

const navItems = [
  { path: '/', label: 'Dashboard', icon: LayoutDashboard },
  { path: '/status', label: 'Status', icon: Activity },
  { path: '/nodes', label: 'Nodes', icon: Radio },
  { path: '/messages', label: 'Messages', icon: MessageSquare },
  { path: '/privacy', label: 'Privacy', icon: Shield },
  { path: '/dead-letters', label: 'Dead Letters', icon: AlertTriangle },
  { path: '/recommendations', label: 'Recommendations', icon: Lightbulb },
  { path: '/events', label: 'Events', icon: ScrollText },
  { path: '/settings', label: 'Settings', icon: Settings },
]

export function Layout({ children }: LayoutProps) {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const location = useLocation()
  const { refreshAll, status } = useApi()

  const isConnected = status.data?.transports?.some(t => t.effective_state === 'connected') ?? false

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
        <div className="flex h-16 items-center gap-2 border-b px-6">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
            <Radio className="h-5 w-5 text-primary-foreground" />
          </div>
          <div className="flex flex-col">
            <span className="font-semibold text-foreground">MEL</span>
            <span className="text-xs text-muted-foreground">MeshEdgeLayer</span>
          </div>
        </div>

        {/* Navigation */}
        <nav className="space-y-1 p-4">
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              onClick={() => setSidebarOpen(false)}
              className={({ isActive }) =>
                clsx(
                  'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                )
              }
            >
              <item.icon className="h-5 w-5" />
              {item.label}
            </NavLink>
          ))}
        </nav>

        {/* Connection status */}
        <div className="absolute bottom-0 left-0 right-0 border-t p-4">
          <div className="flex items-center gap-2 text-sm">
            {isConnected ? (
              <>
                <Wifi className="h-4 w-4 text-success" />
                <span className="text-success">Connected</span>
              </>
            ) : (
              <>
                <WifiOff className="h-4 w-4 text-muted-foreground" />
                <span className="text-muted-foreground">Disconnected</span>
              </>
            )}
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="lg:pl-64">
        {/* Top header */}
        <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b bg-background/95 px-4 backdrop-blur supports-[backdrop-filter]:bg-background/60 lg:px-8">
          {/* Mobile menu button */}
          <button
            onClick={() => setSidebarOpen(true)}
            className="rounded-lg p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground lg:hidden"
            aria-label="Open menu"
          >
            <Menu className="h-5 w-5" />
          </button>

          {/* Breadcrumb / Page title area */}
          <div className="flex items-center gap-2 text-sm">
            {location.pathname !== '/' && (
              <>
                <NavLink to="/" className="text-muted-foreground hover:text-foreground">
                  Home
                </NavLink>
                <span className="text-muted-foreground">/</span>
                <span className="font-medium text-foreground">
                  {navItems.find(n => n.path === location.pathname)?.label || 'Page'}
                </span>
              </>
            )}
          </div>

          {/* Actions */}
          <div className="flex items-center gap-2">
            <button
              onClick={() => refreshAll()}
              className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            >
              <RefreshCw className="h-4 w-4" />
              <span className="hidden sm:inline">Refresh</span>
            </button>
          </div>
        </header>

        {/* Page content */}
        <main className="p-4 lg:p-8">
          {children}
        </main>
      </div>
    </div>
  )
}
