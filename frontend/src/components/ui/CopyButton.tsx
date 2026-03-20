import { useState } from 'react'
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
  successDuration = 2000 
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      setTimeout(() => setCopied(false), successDuration)
    } catch (error) {
      // Fallback for older browsers
      const textArea = document.createElement('textarea')
      textArea.value = value
      textArea.style.position = 'fixed'
      textArea.style.left = '-9999px'
      document.body.appendChild(textArea)
      textArea.select()
      try {
        document.execCommand('copy')
        setCopied(true)
        setTimeout(() => setCopied(false), successDuration)
      } catch (e) {
        // Copy failed
      }
      document.body.removeChild(textArea)
    }
  }

  return (
    <button
      onClick={handleCopy}
      className={clsx(
        'inline-flex items-center gap-1.5 px-2 py-1 text-xs rounded transition-colors',
        'text-muted-foreground hover:text-foreground hover:bg-muted',
        copied && 'text-success bg-success/10',
        className
      )}
      aria-label={copied ? 'Copied!' : label}
      title={label}
    >
      {copied ? (
        <>
          <Check className="h-3.5 w-3.5" />
          <span>Copied</span>
        </>
      ) : (
        <>
          <Copy className="h-3.5 w-3.5" />
          <span>Copy</span>
        </>
      )}
    </button>
  )
}

// For inline text that should be copyable
interface CopyableTextProps {
  value: string
  className?: string
  truncate?: boolean
}

export function CopyableText({ value, className, truncate = false }: CopyableTextProps) {
  return (
    <div className={clsx('inline-flex items-center gap-2', className)}>
      <code className={clsx(
        'font-mono text-sm bg-muted px-1.5 py-0.5 rounded',
        truncate && 'max-w-[200px] truncate'
      )}>
        {value}
      </code>
      <CopyButton value={value} />
    </div>
  )
}

// For displaying IDs, node IDs, message hashes in tables
interface CopyableCellProps {
  value: string
  truncate?: boolean
}

export function CopyableCell({ value, truncate = true }: CopyableCellProps) {
  return (
    <div className="flex items-center gap-2">
      <span className={clsx(
        'font-mono text-sm',
        truncate && 'max-w-[150px] truncate'
      )} title={value}>
        {value}
      </span>
      <CopyButton value={value} />
    </div>
  )
}
