import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { useNavigate } from 'react-router-dom'
import {
  HelpCircle,
  ExternalLink,
  BookOpen,
  MessageSquare,
  Github,
  FileText,
  ChevronDown,
  Keyboard,
} from 'lucide-react'
import { clsx } from 'clsx'
import { isEditableTarget } from '@/utils/keyboard'
import { MEL_GITHUB_REPO, melGithubFile } from '@/constants/repoLinks'

// ─── Global keyboard shortcuts hook ──────────────────────────────────────────
// Exported so Layout can mount it once at the app root.

const NAV_TARGETS: Record<string, string> = {
  h: '/',
  d: '/',
  i: '/incidents',
  t: '/topology',
  n: '/nodes',
  s: '/settings',
  e: '/events',
  p: '/planning',
  c: '/control-actions',
  m: '/messages',
  v: '/operational-review',
}

export function useGlobalKeyboardShortcuts(onRefresh?: () => void) {
  const navigate = useNavigate()
  const gPending = useRef(false)
  const gTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (isEditableTarget(e.target)) return
      if (e.repeat) return
      const key = e.key.toLowerCase()

      // g + <letter> navigation
      if (gPending.current) {
        gPending.current = false
        if (gTimer.current) clearTimeout(gTimer.current)
        const dest = NAV_TARGETS[key]
        if (dest) {
          e.preventDefault()
          navigate(dest)
        }
        return
      }

      if (key === 'g' && !e.ctrlKey && !e.metaKey && !e.altKey) {
        gPending.current = true
        // Cancel g-mode after 1.2s if no second key
        gTimer.current = setTimeout(() => { gPending.current = false }, 1200)
        return
      }

      // r = refresh
      if (key === 'r' && !e.ctrlKey && !e.metaKey && !e.altKey && onRefresh) {
        e.preventDefault()
        onRefresh()
      }
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [navigate, onRefresh])
}

interface HelpLink {
  label: string
  href: string
  description?: string
  icon?: 'docs' | 'api' | 'github' | 'community' | 'changelog'
}

const helpLinks: HelpLink[] = [
  {
    label: 'Documentation',
    href: '/docs/ops/first-10-minutes.md',
    description: 'Operator quick start (served when static docs are bundled with the UI)',
    icon: 'docs',
  },
  {
    label: 'API reference',
    href: '/docs/ops/api-reference.md',
    description: 'REST endpoints (same caveat as other /docs links)',
    icon: 'api',
  },
  {
    label: 'GitHub',
    href: MEL_GITHUB_REPO,
    description: 'Source code and issues',
    icon: 'github',
  },
  {
    label: 'Community',
    href: 'https://meshtastic.org/',
    description: 'Meshtastic project site',
    icon: 'community',
  },
  {
    label: 'Changelog',
    href: melGithubFile('CHANGELOG.md'),
    description: 'Release notes in the repository',
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
  const [shortcutsOpen, setShortcutsOpen] = useState(false)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const menuId = useId()
  const shortcutsTitleId = useId()
  const shortcutsDescId = useId()

  const closeMenu = useCallback(() => setIsOpen(false), [])
  const openShortcuts = useCallback(() => {
    setIsOpen(false)
    setShortcutsOpen(true)
  }, [])

  useEffect(() => {
    if (!isOpen) return
    const first = menuRef.current?.querySelector<HTMLElement>('[role="menuitem"]')
    window.setTimeout(() => first?.focus(), 0)
  }, [isOpen])

  useEffect(() => {
    if (!isOpen) return
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        closeMenu()
        buttonRef.current?.focus()
        return
      }
      const menu = menuRef.current
      if (!menu) return
      const items = Array.from(menu.querySelectorAll<HTMLElement>('[role="menuitem"]')).filter(
        (el) => !el.hasAttribute('disabled') && el.getAttribute('aria-disabled') !== 'true'
      )
      if (items.length === 0) return
      const active = document.activeElement
      const idx = items.indexOf(active as HTMLElement)

      const focusAt = (next: number) => {
        const i = ((next % items.length) + items.length) % items.length
        items[i]?.focus()
      }

      if (e.key === 'ArrowDown') {
        e.preventDefault()
        focusAt(idx < 0 ? 0 : idx + 1)
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        focusAt(idx < 0 ? items.length - 1 : idx - 1)
      } else if (e.key === 'Home') {
        e.preventDefault()
        items[0]?.focus()
      } else if (e.key === 'End') {
        e.preventDefault()
        items[items.length - 1]?.focus()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [isOpen, closeMenu])

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== '?' || e.repeat) return
      if (isEditableTarget(e.target)) return
      e.preventDefault()
      setShortcutsOpen(true)
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [])

  const shortcutsDialogRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!shortcutsOpen) return
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        setShortcutsOpen(false)
        buttonRef.current?.focus()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [shortcutsOpen])

  useEffect(() => {
    if (!shortcutsOpen) return
    const root = shortcutsDialogRef.current
    if (!root) return
    const focusables = root.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    )
    const list = Array.from(focusables).filter((el) => !el.hasAttribute('disabled'))
    window.setTimeout(() => list[0]?.focus(), 0)

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Tab' || list.length === 0) return
      const first = list[0]
      const last = list[list.length - 1]
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault()
          last.focus()
        }
      } else if (document.activeElement === last) {
        e.preventDefault()
        first.focus()
      }
    }
    root.addEventListener('keydown', onKeyDown)
    return () => root.removeEventListener('keydown', onKeyDown)
  }, [shortcutsOpen])

  return (
    <div className="relative">
      <button
        ref={buttonRef}
        id="mel-help-menu-trigger"
        type="button"
        onClick={() => setIsOpen((o) => !o)}
        aria-label="Help menu"
        className={clsx(
          'inline-flex h-10 items-center gap-2 rounded-md border px-3 py-2 text-sm font-medium transition-all duration-150',
          'border-border/70 bg-card/80 text-muted-foreground shadow-inset hover:border-primary/16 hover:bg-accent/65 hover:text-foreground',
          'outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
          isOpen && 'border-primary/18 bg-accent/70 text-foreground'
        )}
        aria-expanded={isOpen}
        aria-haspopup="menu"
        aria-controls={isOpen ? menuId : undefined}
      >
        <HelpCircle className="h-5 w-5 shrink-0" aria-hidden />
        <span className="hidden sm:inline" aria-hidden>
          Help
        </span>
        <ChevronDown
          className={clsx('hidden h-4 w-4 shrink-0 transition-transform sm:block', isOpen && 'rotate-180')}
          aria-hidden
        />
      </button>

      {isOpen && (
        <>
          <div
            className="fixed inset-0 z-40"
            onClick={closeMenu}
            aria-hidden="true"
          />
          <div
            ref={menuRef}
            id={menuId}
            className="surface-panel absolute right-0 z-50 mt-2 w-[min(100vw-2rem,18rem)] overflow-hidden rounded-md"
            role="menu"
            aria-labelledby="mel-help-menu-trigger"
          >
            <div className="max-h-[min(70vh,24rem)] overflow-y-auto p-2">
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
                      'flex items-start gap-3 rounded-md border border-transparent p-3 transition-all duration-150',
                      'hover:border-border/60 hover:bg-accent/65 hover:text-accent-foreground',
                      'outline-none focus-visible:ring-2 focus-visible:ring-ring'
                    )}
                    role="menuitem"
                    tabIndex={-1}
                    onClick={closeMenu}
                  >
                    <Icon className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1">
                        <span className="font-medium text-foreground">{link.label}</span>
                        {isExternal && (
                          <ExternalLink className="h-3 w-3 shrink-0 text-muted-foreground" aria-hidden />
                        )}
                      </div>
                      {link.description && (
                        <p className="mt-0.5 text-xs text-muted-foreground">{link.description}</p>
                      )}
                    </div>
                  </a>
                )
              })}
              <button
                type="button"
                role="menuitem"
                tabIndex={-1}
                className={clsx(
                  'flex w-full items-start gap-3 rounded-md border border-transparent p-3 text-left transition-all duration-150',
                  'hover:border-border/60 hover:bg-accent/65 hover:text-accent-foreground',
                  'outline-none focus-visible:ring-2 focus-visible:ring-ring'
                )}
                onClick={openShortcuts}
              >
                <Keyboard className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
                <span className="font-medium text-foreground">Keyboard shortcuts</span>
              </button>
            </div>
            <p className="border-t border-border/60 px-3 py-2 text-mel-sm leading-snug text-muted-foreground">
              Press <kbd className="rounded-md border border-border/70 bg-card/70 px-1.5 py-0.5 font-mono text-mel-xs text-foreground">?</kbd> outside fields to open shortcuts.
            </p>
          </div>
        </>
      )}

      {shortcutsOpen &&
        createPortal(
          <div
            className="fixed inset-0 z-[60] flex items-center justify-center p-4"
            role="presentation"
          >
            <div
              className="absolute inset-0 bg-background/80 backdrop-blur-sm"
              aria-hidden="true"
              onClick={() => setShortcutsOpen(false)}
            />
            <div
              ref={shortcutsDialogRef}
              role="dialog"
              aria-modal="true"
              aria-labelledby={shortcutsTitleId}
              aria-describedby={shortcutsDescId}
              className="surface-panel relative z-[61] w-full max-w-md rounded-md p-6"
            >
              <h2 id={shortcutsTitleId} className="text-lg font-semibold text-foreground">
                Keyboard shortcuts
              </h2>
              <p id={shortcutsDescId} className="mt-1 text-sm text-muted-foreground">
                These apply to the operator console only. They do not refresh mesh data by themselves.
              </p>
              <div className="mt-4">
                <KeyboardShortcuts />
              </div>
              <div className="mt-6 flex justify-end">
                <button
                  type="button"
                  className="button-secondary"
                  onClick={() => {
                    setShortcutsOpen(false)
                    buttonRef.current?.focus()
                  }}
                >
                  Close
                </button>
              </div>
            </div>
          </div>,
          document.body
        )}
    </div>
  )
}

/** Compact footer line; version should be supplied from GET /api/v1/version when available */
export function VersionInfo({
  versionLabel,
  loading,
}: {
  versionLabel?: string
  loading?: boolean
}) {
  return (
    <div className="text-xs text-muted-foreground">
      {loading ? (
        <span>Loading version…</span>
      ) : (
        <span>{versionLabel ? `MEL ${versionLabel}` : 'MEL (version unavailable)'}</span>
      )}
      <span className="mx-1.5" aria-hidden>
        ·
      </span>
      <a
        href="https://github.com/meshtastic/MEL/blob/main/CHANGELOG.md"
        className="transition-colors hover:text-foreground"
        target="_blank"
        rel="noopener noreferrer"
      >
        Changelog
      </a>
      <span className="mx-1.5" aria-hidden>
        ·
      </span>
      <a
        href="https://github.com/meshtastic/MEL"
        target="_blank"
        rel="noopener noreferrer"
        className="transition-colors hover:text-foreground"
      >
        GitHub
      </a>
    </div>
  )
}

export function KeyboardShortcuts() {
  const groups: { heading: string; shortcuts: { keys: string; description: string }[] }[] = [
    {
      heading: 'General',
      shortcuts: [
        { keys: '?', description: 'Open this shortcuts panel' },
        { keys: 'Escape', description: 'Close open menus or this panel' },
        { keys: 'r', description: 'Refresh current page data' },
      ],
    },
    {
      heading: 'Navigation (g + key)',
      shortcuts: [
        { keys: 'g h', description: 'Go to Dashboard' },
        { keys: 'g i', description: 'Go to Incidents' },
        { keys: 'g t', description: 'Go to Topology' },
        { keys: 'g n', description: 'Go to Nodes' },
        { keys: 'g p', description: 'Go to Planning' },
        { keys: 'g c', description: 'Go to Control actions' },
        { keys: 'g m', description: 'Go to Messages' },
        { keys: 'g e', description: 'Go to Events' },
        { keys: 'g v', description: 'Go to Operational review (digest + briefing)' },
        { keys: 'g s', description: 'Go to Settings' },
      ],
    },
  ]

  return (
    <div className="text-xs space-y-4">
      {groups.map((group) => (
        <div key={group.heading}>
          <p className="mb-2 font-semibold uppercase tracking-[0.14em] text-muted-foreground/70">{group.heading}</p>
          <ul className="space-y-2">
            {group.shortcuts.map((s) => (
              <li key={s.keys} className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
                <kbd className="shrink-0 rounded bg-muted px-1.5 py-0.5 font-mono text-mel-sm">{s.keys}</kbd>
                <span className="min-w-0 text-muted-foreground">{s.description}</span>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  )
}
