import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import {
  clearOperatorWorkspaceFocus,
  readOperatorWorkspaceFocus,
  writeOperatorWorkspaceFocus,
  type OperatorWorkspaceFocus,
} from '@/utils/operatorWorkspaceFocus'

interface Ctx {
  focus: OperatorWorkspaceFocus | null
  setFocus: (f: OperatorWorkspaceFocus) => void
  clearFocus: () => void
}

const OperatorWorkspaceFocusContext = createContext<Ctx | undefined>(undefined)

export function OperatorWorkspaceFocusProvider({ children }: { children: ReactNode }) {
  const [focus, setFocusState] = useState<OperatorWorkspaceFocus | null>(() => readOperatorWorkspaceFocus())

  const setFocus = useCallback((f: OperatorWorkspaceFocus) => {
    const next = { ...f, savedAt: f.savedAt || new Date().toISOString() }
    writeOperatorWorkspaceFocus(next)
    setFocusState(next)
  }, [])

  const clearFocus = useCallback(() => {
    clearOperatorWorkspaceFocus()
    setFocusState(null)
  }, [])

  const value = useMemo(() => ({ focus, setFocus, clearFocus }), [focus, setFocus, clearFocus])

  return (
    <OperatorWorkspaceFocusContext.Provider value={value}>
      {children}
    </OperatorWorkspaceFocusContext.Provider>
  )
}

/** Safe outside provider (no-op) for isolated tests; in-app always wrap with Provider. */
export function useOperatorWorkspaceFocus(): Ctx {
  const ctx = useContext(OperatorWorkspaceFocusContext)
  if (ctx) return ctx
  return {
    focus: null,
    setFocus: () => {},
    clearFocus: () => {},
  }
}
