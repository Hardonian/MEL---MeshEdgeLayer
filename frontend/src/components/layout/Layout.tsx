import { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { clsx } from 'clsx'
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
  HelpCircle,
  Activity,
  GitBranch,
  AlertTriangle,
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
  { label: 'Messages', href: '/messages', icon: MessageSquare },
  { label: 'Dead Letters', href: '/dead-letters', icon: Inbox },
  { label: 'Incidents', href: '/incidents', icon: AlertTriangle },
  { label: 'Events', href: '/events', icon: FileText },
  { label: 'Diagnostics', href: '/diagnostics', icon: Shield },
  { label: 'Privacy', href: '/privacy', icon: Shield },
  { label: 'Recommendations', href: '/recommendations', icon: Activity },
  { label: 'Settings', href: '/settings', icon: Settings },
]

export function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)
  const [isMobile, setIsMobile] = useState(false)

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

  return (
    <div className="min-h-screen bg-background">
      {/* Top navigation */}
      <header className="sticky top-0 z-50 glass dark:glass-dark">
        <div className="flex h-16 items-center justify-between px-6">
          <div className="flex items-center gap-4">
            <button
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              className="rounded-md p-2 hover:bg-muted md:hidden"
              aria-label="Toggle menu"
            >
              {isMobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
            <Link to="/" className="flex items-center gap-2 group">
              <div className="bg-primary text-primary-foreground p-1.5 rounded-lg shadow-inner group-hover:scale-110 transition-transform">
                <Radio className="h-5 w-5" />
              </div>
              <span className="font-outfit font-bold text-xl tracking-tight hidden sm:inline">MEL</span>
            </Link>
          </div>

          <div className="flex items-center gap-4">
            <div className="hidden md:flex items-center gap-2 text-xs font-medium text-muted-foreground uppercase tracking-widest bg-muted/50 px-3 py-1.5 rounded-full border">
              <span className="h-1.5 w-1.5 rounded-full bg-success animate-pulse" />
              MeshEdgeLayer Active
            </div>
            <button 
              className="p-2 rounded-full hover:bg-muted transition-colors text-muted-foreground"
              aria-label="Help and Support"
              title="Help and Support"
            >
              <HelpCircle className="h-5 w-5" />
            </button>
          </div>
        </div>
      </header>

      <div className="flex">
        {/* Sidebar navigation - desktop */}
        <aside className={clsx(
          "fixed inset-y-0 left-0 z-40 w-64 transform border-r bg-background transition-transform duration-200 ease-in-out md:translate-x-0",
          isMobile && isMobileMenuOpen ? "translate-x-0" : "-translate-x-full"
        )}>
          <nav className="flex flex-col gap-1 p-4 pt-16">
            {navItems.map((item) => {
              const isActive = location.pathname === item.href || 
                (item.href !== '/' && location.pathname.startsWith(item.href))
              return (
                <Link
                  key={item.href}
                  to={item.href}
                  className={clsx(
                    "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
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
          <div className="absolute bottom-0 left-0 right-0 border-t p-4">
            <Link
              to="/settings"
              className="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              <HelpCircle className="h-4 w-4" />
              Help & Support
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
        <main className="flex-1 p-4 md:p-6 md:ml-64">
          {children}
        </main>
      </div>
    </div>
  )
}
