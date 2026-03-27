import { createContext, useContext, useState, useCallback, ReactNode } from 'react'
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react'
import { clsx } from 'clsx'

type ToastType = 'success' | 'error' | 'warning' | 'info'

interface Toast {
  id: string
  type: ToastType
  title: string
  message?: string
  duration?: number
}

interface ToastContextType {
  toasts: Toast[]
  addToast: (toast: Omit<Toast, 'id'>) => void
  removeToast: (id: string) => void
}

const ToastContext = createContext<ToastContextType | null>(null)

export function useToast() {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider')
  }
  return context
}

interface ToastProviderProps {
  children: ReactNode
}

export function ToastProvider({ children }: ToastProviderProps) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const addToast = useCallback((toast: Omit<Toast, 'id'>) => {
    const id = crypto.randomUUID()
    setToasts((prev) => [...prev, { ...toast, id }])

    const duration = toast.duration ?? 5000
    if (duration > 0) {
      window.setTimeout(() => {
        setToasts((prev) => prev.filter((t) => t.id !== id))
      }, duration)
    }
  }, [])

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  return (
    <ToastContext.Provider value={{ toasts, addToast, removeToast }}>
      {children}
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </ToastContext.Provider>
  )
}

function ToastContainer({ toasts, onRemove }: { toasts: Toast[]; onRemove: (id: string) => void }) {
  if (toasts.length === 0) return null

  return (
    <div
      className="fixed bottom-4 right-4 z-50 flex max-w-sm flex-col gap-2"
      role="region"
      aria-label="Notifications"
    >
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} onRemove={onRemove} />
      ))}
    </div>
  )
}

function ToastItem({ toast, onRemove }: { toast: Toast; onRemove: (id: string) => void }) {
  const icons = {
    success: CheckCircle,
    error: AlertCircle,
    warning: AlertTriangle,
    info: Info,
  }

  const colors = {
    success: {
      rail: 'from-success/55 via-success/20 to-transparent',
      icon: 'border-success/16 bg-success/12 text-success',
    },
    error: {
      rail: 'from-critical/55 via-critical/22 to-transparent',
      icon: 'border-critical/18 bg-critical/12 text-critical',
    },
    warning: {
      rail: 'from-warning/55 via-warning/22 to-transparent',
      icon: 'border-warning/18 bg-warning/12 text-warning',
    },
    info: {
      rail: 'from-info/55 via-info/22 to-transparent',
      icon: 'border-info/18 bg-info/12 text-info',
    },
  } as const

  const Icon = icons[toast.type]
  const tone = colors[toast.type]

  return (
    <div
      className="surface-panel animate-toast-in relative overflow-hidden rounded-[1.1rem] p-4"
      role="status"
      aria-live="polite"
    >
      <div className={clsx('absolute inset-y-0 left-0 w-1 bg-gradient-to-b', tone.rail)} aria-hidden />
      <div className="flex items-start gap-3">
        <div className={clsx('flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border shadow-inset', tone.icon)}>
          <Icon className="h-5 w-5" aria-hidden="true" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="font-semibold text-foreground">{toast.title}</p>
          {toast.message && (
            <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{toast.message}</p>
          )}
        </div>
        <button
          onClick={() => onRemove(toast.id)}
          className="icon-button h-8 min-h-8 w-8 min-w-8 rounded-xl"
          aria-label="Dismiss notification"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  )
}

export function useToastHelpers() {
  const { addToast } = useToast()

  return {
    success: (title: string, message?: string) => addToast({ type: 'success', title, message }),
    error: (title: string, message?: string) => addToast({ type: 'error', title, message }),
    warning: (title: string, message?: string) => addToast({ type: 'warning', title, message }),
    info: (title: string, message?: string) => addToast({ type: 'info', title, message }),
  }
}
