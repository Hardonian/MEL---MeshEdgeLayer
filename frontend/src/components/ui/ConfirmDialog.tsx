import { useState, useRef, useEffect, ReactNode } from 'react'
import { AlertTriangle, X } from 'lucide-react'
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
  onConfirm: () => void
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
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    if (open) {
      cancelRef.current?.focus()
    }
  }, [open])

  const handleConfirm = async () => {
    setIsLoading(true)
    try {
      await onConfirm()
      onOpenChange(false)
    } catch (error) {
      // Error handling - dialog stays open
    } finally {
      setIsLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onOpenChange(false)
    }
  }

  if (!open) return null

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="dialog-title"
      onKeyDown={handleKeyDown}
    >
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-background/80 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
        aria-hidden="true"
      />

      {/* Dialog */}
      <div className="relative w-full max-w-md animate-expand-in bg-card rounded-xl border shadow-xl">
        {/* Header */}
        <div className="flex items-start gap-4 p-6">
          <div className={clsx(
            'flex h-10 w-10 shrink-0 items-center justify-center rounded-full',
            variant === 'danger' ? 'bg-critical/10' : 'bg-primary/10'
          )}>
            <AlertTriangle className={clsx(
              'h-5 w-5',
              variant === 'danger' ? 'text-critical' : 'text-primary'
            )} />
          </div>
          <div className="flex-1">
            <h2 id="dialog-title" className="text-lg font-semibold text-foreground">
              {title}
            </h2>
            {description && (
              <p className="mt-1 text-sm text-muted-foreground">{description}</p>
            )}
          </div>
        </div>

        {/* Message */}
        <div className="px-6 pb-2">
          <div className="text-sm text-foreground bg-muted rounded-lg p-4">
            {message}
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center justify-end gap-3 p-6 pt-4">
          <button
            ref={cancelRef}
            onClick={() => onOpenChange(false)}
            className="px-4 py-2 text-sm font-medium rounded-lg border bg-background hover:bg-accent transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
          >
            {cancelLabel}
          </button>
          <button
            onClick={handleConfirm}
            disabled={isLoading}
            className={clsx(
              'px-4 py-2 text-sm font-medium rounded-lg text-white transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
              variant === 'danger' 
                ? 'bg-critical hover:bg-critical/90' 
                : 'bg-primary hover:bg-primary/90',
              isLoading && 'opacity-50 cursor-not-allowed'
            )}
          >
            {isLoading ? 'Processing...' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}

// Destructive action button with built-in confirmation
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
        className="text-sm text-critical hover:text-critical/80 transition-colors"
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
