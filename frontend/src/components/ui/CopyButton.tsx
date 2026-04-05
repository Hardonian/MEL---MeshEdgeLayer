import { useId, useState } from 'react'
import { Copy, Check } from 'lucide-react'
import { clsx } from 'clsx'

interface CopyButtonProps {
  value: string
  className?: string
  label?: string
  successDuration?: number
}

export function CopyButton({
  value,
  className,
  label = 'Copy to clipboard',
  successDuration = 2000,
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false)
  const statusId = useId()

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      window.setTimeout(() => setCopied(false), successDuration)
    } catch {
      const textArea = document.createElement('textarea')
      textArea.value = value
      textArea.style.position = 'fixed'
      textArea.style.left = '-9999px'
      document.body.appendChild(textArea)
      textArea.select()
      try {
        document.execCommand('copy')
        setCopied(true)
        window.setTimeout(() => setCopied(false), successDuration)
      } finally {
        document.body.removeChild(textArea)
      }
    }
  }

  return (
    <>
      <button
        type="button"
        onClick={handleCopy}
        className={clsx(
          'inline-flex items-center gap-1.5 rounded-md border border-border/70 bg-card/75 px-2 py-1.5 text-xs font-medium text-muted-foreground shadow-inset outline-none transition-all duration-150 hover:border-primary/16 hover:bg-accent/60 hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
          copied && 'border-success/18 bg-success/10 text-success',
          className
        )}
        aria-label={copied ? 'Copied to clipboard' : label}
        aria-describedby={statusId}
        title={label}
      >
        {copied ? (
          <>
            <Check className="h-3.5 w-3.5" aria-hidden />
            <span>Copied</span>
          </>
        ) : (
          <>
            <Copy className="h-3.5 w-3.5" aria-hidden />
            <span>Copy</span>
          </>
        )}
      </button>
      <span id={statusId} className="sr-only" aria-live="polite" aria-atomic="true">
        {copied ? 'Copied to clipboard' : ''}
      </span>
    </>
  )
}

interface CopyableTextProps {
  value: string
  className?: string
  truncate?: boolean
}

export function CopyableText({ value, className, truncate = false }: CopyableTextProps) {
  return (
    <div className={clsx('inline-flex items-center gap-2', className)}>
      <code
        className={clsx(
          'raw-block px-2 py-1 font-mono text-sm',
          truncate && 'max-w-[200px] truncate'
        )}
      >
        {value}
      </code>
      <CopyButton value={value} />
    </div>
  )
}

interface CopyableCellProps {
  value: string
  truncate?: boolean
}

export function CopyableCell({ value, truncate = true }: CopyableCellProps) {
  return (
    <div className="flex items-center gap-2">
      <span
        className={clsx(
          'font-mono text-sm text-foreground',
          truncate && 'max-w-[150px] truncate'
        )}
        title={value}
      >
        {value}
      </span>
      <CopyButton value={value} />
    </div>
  )
}
