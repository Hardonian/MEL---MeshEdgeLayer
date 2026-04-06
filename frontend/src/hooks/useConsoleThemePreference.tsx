import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

export type ConsoleThemePreference = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'mel.console.theme'

function readStored(): ConsoleThemePreference | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw === 'system' || raw === 'light' || raw === 'dark') {
      return raw
    }
  } catch {
    /* ignore */
  }
  return null
}

function applyTheme(preference: ConsoleThemePreference): void {
  const root = document.documentElement
  if (preference === 'system') {
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    root.classList.toggle('dark', prefersDark)
  } else {
    root.classList.toggle('dark', preference === 'dark')
  }
}

interface ConsoleThemeContextValue {
  preference: ConsoleThemePreference
  setPreference: (next: ConsoleThemePreference) => void
}

const ConsoleThemeContext = createContext<ConsoleThemeContextValue | null>(null)

export function ConsoleThemeProvider({ children }: { children: ReactNode }) {
  /** Default dark: operator console is dark-first; "system" still available in Settings. */
  const [preference, setPreferenceState] = useState<ConsoleThemePreference>(() => readStored() ?? 'dark')

  const setPreference = useCallback((next: ConsoleThemePreference) => {
    setPreferenceState(next)
    try {
      localStorage.setItem(STORAGE_KEY, next)
    } catch {
      /* ignore */
    }
    applyTheme(next)
  }, [])

  useEffect(() => {
    applyTheme(preference)
  }, [preference])

  useEffect(() => {
    if (preference !== 'system') {
      return
    }
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = () => applyTheme('system')
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [preference])

  const value = useMemo(
    () => ({ preference, setPreference }),
    [preference, setPreference]
  )

  return <ConsoleThemeContext.Provider value={value}>{children}</ConsoleThemeContext.Provider>
}

/**
 * Local-only UI theme for the operator console (classList on the document root).
 * Does not affect MEL server configuration. Must be used within ConsoleThemeProvider.
 */
export function useConsoleThemePreference(): ConsoleThemeContextValue {
  const ctx = useContext(ConsoleThemeContext)
  if (!ctx) {
    throw new Error('useConsoleThemePreference must be used within ConsoleThemeProvider')
  }
  return ctx
}
