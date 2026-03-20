import { useState } from 'react'
import { 
  HelpCircle, 
  ExternalLink, 
  BookOpen, 
  MessageSquare, 
  Github, 
  FileText,
  ChevronDown 
} from 'lucide-react'
import { clsx } from 'clsx'

interface HelpLink {
  label: string
  href: string
  description?: string
  icon?: 'docs' | 'api' | 'github' | 'community' | 'changelog'
}

const helpLinks: HelpLink[] = [
  { 
    label: 'Documentation', 
    href: '/docs/ops/first-10-minutes', 
    description: 'Get started with MEL',
    icon: 'docs' 
  },
  { 
    label: 'API Reference', 
    href: '/docs/ops/api-reference',
    description: 'REST API endpoints',
    icon: 'api' 
  },
  { 
    label: 'GitHub', 
    href: 'https://github.com/meshtastic/MEL',
    description: 'Source code & issues',
    icon: 'github',
  },
  { 
    label: 'Community', 
    href: 'https://meshtastic.org/',
    description: 'Discussion forums',
    icon: 'community',
  },
  { 
    label: 'Changelog', 
    href: '/CHANGELOG.md',
    description: 'Release notes',
    icon: 'changelog',
  },
]

const icons = {
  docs: BookOpen,
  api: FileText,
  github: Github,
  community: MessageSquare,
  changelog: FileText,
}

export function HelpMenu() {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={clsx(
          'flex items-center gap-2 px-3 py-2 text-sm rounded-lg transition-colors',
          'text-muted-foreground hover:text-foreground hover:bg-accent',
          isOpen && 'bg-accent'
        )}
        aria-expanded={isOpen ? 'true' : 'false'}
        aria-haspopup="true"
      >
        <HelpCircle className="h-4 w-4" />
        <span className="hidden sm:inline">Help</span>
        <ChevronDown className={clsx(
          'h-4 w-4 transition-transform',
          isOpen && 'rotate-180'
        )} />
      </button>

      {isOpen && (
        <>
          {/* Backdrop */}
          <div 
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
            aria-hidden="true"
          />

          {/* Menu */}
          <div 
            className="absolute right-0 mt-2 w-64 z-50 bg-card rounded-lg border shadow-xl animate-fade-in"
            role="menu"
          >
            <div className="p-2">
              {helpLinks.map((link, index) => {
                const Icon = icons[link.icon || 'docs']
                const isExternal = link.href.startsWith('http')
                
                return (
                  <a
                    key={index}
                    href={link.href}
                    target={isExternal ? '_blank' : undefined}
                    rel={isExternal ? 'noopener noreferrer' : undefined}
                    className={clsx(
                      'flex items-start gap-3 p-3 rounded-lg transition-colors',
                      'hover:bg-accent hover:text-accent-foreground',
                      'focus:outline-none focus:ring-2 focus:ring-ring'
                    )}
                    role="menuitem"
                    onClick={() => setIsOpen(false)}
                  >
                    <Icon className="h-5 w-5 shrink-0 text-muted-foreground" />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-1">
                        <span className="font-medium text-foreground">{link.label}</span>
                        {isExternal && (
                          <ExternalLink className="h-3 w-3 text-muted-foreground" />
                        )}
                      </div>
                      {link.description && (
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {link.description}
                        </p>
                      )}
                    </div>
                  </a>
                )
              })}
            </div>
          </div>
        </>
      )}
    </div>
  )
}

// Version info component for footer/settings
export function VersionInfo() {
  return (
    <div className="text-xs text-muted-foreground">
      <span>MEL v0.1.0</span>
      <span className="mx-1.5">·</span>
      <a 
        href="/CHANGELOG" 
        className="hover:text-foreground transition-colors"
      >
        Changelog
      </a>
      <span className="mx-1.5">·</span>
      <a 
        href="https://github.com/meshtastic/MEL"
        target="_blank"
        rel="noopener noreferrer"
        className="hover:text-foreground transition-colors"
      >
        GitHub
      </a>
    </div>
  )
}

// Keyboard shortcuts help
export function KeyboardShortcuts() {
  const shortcuts = [
    { keys: ['?'], description: 'Show keyboard shortcuts' },
    { keys: ['r'], description: 'Refresh data' },
    { keys: ['/'], description: 'Search' },
    { keys: ['Esc'], description: 'Close dialog/menu' },
  ]

  return (
    <div className="text-xs">
      <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
        {shortcuts.map((shortcut, index) => (
          <div key={index} className="contents">
            <kbd className="px-1.5 py-0.5 bg-muted rounded text-xs font-mono">
              {shortcut.keys.join('+')}
            </kbd>
            <span className="text-muted-foreground">{shortcut.description}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
