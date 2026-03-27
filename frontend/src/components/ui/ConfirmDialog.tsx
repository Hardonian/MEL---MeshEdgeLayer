import { useState, useRef, useEffect, useId, ReactNode } from 'react'
import { AlertTriangle } from 'lucide-react'
import { clsx } from 'clsx'

interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  message: ReactNode
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'default' | 'danger'
  onConfirm: () => void | Promise<void>
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'default',
  onConfirm,
}: ConfirmDialogProps) {
  const cancelRef = useRef<HTMLButtonElement>(null)
  const dialogRef = useRef<HTMLDivElement>(null)
  const [isLoading, setIsLoading] = useState(false)
  const titleId = useId()
  const descriptionId = useId()

  useEffect(() => {
    if (open) {
      cancelRef.current?.focus()
    }
  }, [open])

  useEffect(() => {
    if (!open) return

    const root = dialogRef.current
    if (!root) return

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onOpenChange(false)
        return
      }

      if (e.key !== 'Tab') return

      const focusables = Array.from(
        root.querySelectorAll<HTMLElement>(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        )
      ).filter((el) => !el.hasAttribute('disabled'))

      if (focusables.length === 0) return

      const first = focusables[0]
      const last = focusables[focusables.length - 1]

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
  }, [open, onOpenChange])

  const handleConfirm = async () => {
    setIsLoading(true)
    try {
      await onConfirm()
      onOpenChange(false)
    } finally {
      setIsLoading(false)
    }
  }

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      role="presentation"
    >
      <div
        className="absolute inset-0 bg-background/82 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
        aria-hidden="true"
      />

      <div
        ref={dialogRef}
        className="surface-panel animate-expand-in relative w-full max-w-md overflow-hidden rounded-[1.25rem]"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description || typeof message === 'string' ? descriptionId : undefined}
      >
        <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-primary/18 via-white/10 to-warning/18" aria-hidden />

        <div className="flex items-start gap-4 p-6">
          <div
            className={clsx(
              'flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border shadow-inset',
              variant === 'danger'
                ? 'border-critical/18 bg-critical/12 text-critical'
                : 'border-primary/18 bg-primary/12 text-primary'
            )}
          >
            <AlertTriangle className="h-5 w-5" />
          </div>
          <div className="min-w-0 flex-1">
            <h2 id={titleId} className="font-outfit text-xl font-semibold tracking-[-0.03em] text-foreground">
              {title}
            </h2>
            {description && (
              <p id={descriptionId} className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
                {description}
              </p>
            )}
          </div>
        </div>

        <div className="px-6 pb-2">
          <div
            id={!description && typeof message === 'string' ? descriptionId : undefined}
            className="raw-block px-4 py-4 text-sm leading-relaxed text-foreground"
          >
            {message}
          </div>
        </div>

        <div className="flex items-center justify-end gap-3 p-6 pt-5">
          <button
            ref={cancelRef}
            onClick={() => onOpenChange(false)}
            className="button-secondary"
          >
            {cancelLabel}
          </button>
          <button
            onClick={handleConfirm}
            disabled={isLoading}
            className={variant === 'danger' ? 'button-danger' : 'button-primary'}
          >
            {isLoading ? 'Processing...' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}

interface DangerousActionButtonProps {
  onClick: () => void
  label: string
  description: string
  confirmLabel?: string
}

export function DangerousActionButton({
  onClick,
  label,
  description,
  confirmLabel = 'Delete',
}: DangerousActionButtonProps) {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <>
      <button
        onClick={() => setIsOpen(true)}
        className="text-sm font-medium text-critical transition-colors hover:text-critical/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        {label}
      </button>
      <ConfirmDialog
        open={isOpen}
        onOpenChange={setIsOpen}
        title={label}
        description="This action cannot be undone."
        message={description}
        confirmLabel={confirmLabel}
        variant="danger"
        onConfirm={onClick}
      />
    </>
  )
}
