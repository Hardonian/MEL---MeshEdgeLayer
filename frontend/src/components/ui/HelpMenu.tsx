import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
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
    href: 'https://github.com/meshtastic/MEL',
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
    href: 'https://github.com/meshtastic/MEL/blob/main/CHANGELOG.md',
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

function isEditableTarget(target: EventTarget | null): boolean {
  if (!target || !(target instanceof HTMLElement)) return false
  const tag = target.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
  if (target.isContentEditable) return true
  return false
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

  return (
    <div className="relative">
      <button
        ref={buttonRef}
        id="mel-help-menu-trigger"
        type="button"
        onClick={() => setIsOpen((o) => !o)}
        className={clsx(
          'flex items-center gap-2 rounded-lg px-2 py-2 text-sm transition-colors sm:px-3',
          'text-muted-foreground hover:bg-muted hover:text-foreground',
          'outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
          isOpen && 'bg-muted text-foreground'
        )}
        aria-expanded={isOpen}
        aria-haspopup="true"
        aria-controls={isOpen ? menuId : undefined}
      >
        <HelpCircle className="h-5 w-5 shrink-0" aria-hidden />
        <span className="hidden sm:inline">Help</span>
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
            className="absolute right-0 z-50 mt-2 w-[min(100vw-2rem,18rem)] rounded-lg border border-border bg-card shadow-xl"
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
                      'flex items-start gap-3 rounded-lg p-3 transition-colors',
                      'hover:bg-accent hover:text-accent-foreground',
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
                  'flex w-full items-start gap-3 rounded-lg p-3 text-left transition-colors',
                  'hover:bg-accent hover:text-accent-foreground',
                  'outline-none focus-visible:ring-2 focus-visible:ring-ring'
                )}
                onClick={openShortcuts}
              >
                <Keyboard className="h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
                <span className="font-medium text-foreground">Keyboard shortcuts</span>
              </button>
            </div>
            <p className="border-t border-border px-3 py-2 text-[11px] leading-snug text-muted-foreground">
              Press <kbd className="rounded bg-muted px-1 font-mono text-[10px]">?</kbd> outside fields to open shortcuts.
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
              role="dialog"
              aria-modal="true"
              aria-labelledby={shortcutsTitleId}
              aria-describedby={shortcutsDescId}
              className="relative z-[61] w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-xl"
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
                  className="rounded-lg border border-border bg-background px-4 py-2 text-sm font-medium transition-colors hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
  const shortcuts: { keys: string; description: string }[] = [
    { keys: '?', description: 'Open this shortcuts panel (when focus is not in an input)' },
    { keys: 'Escape', description: 'Close open menus or this panel' },
  ]

  return (
    <div className="text-xs">
      <ul className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-3">
        {shortcuts.map((shortcut) => (
          <li key={shortcut.keys} className="contents">
            <kbd className="rounded bg-muted px-1.5 py-0.5 font-mono text-[11px]">{shortcut.keys}</kbd>
            <span className="text-muted-foreground">{shortcut.description}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
