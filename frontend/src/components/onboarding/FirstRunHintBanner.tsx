import { useCallback, useState } from 'react'
import { Link } from 'react-router-dom'
import { BookOpen, X } from 'lucide-react'

const STORAGE_KEY = 'mel.dismissFirstRunHint'

function readDismissed(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

/**
 * Shown when no transports are configured — points operators at honest next steps without implying ingest is live.
 */
export function FirstRunHintBanner({ visible }: { visible: boolean }) {
  const [dismissed, setDismissed] = useState(readDismissed)

  const dismiss = useCallback(() => {
    try {
      localStorage.setItem(STORAGE_KEY, '1')
    } catch {
      /* ignore quota / private mode */
    }
    setDismissed(true)
  }, [])

  if (!visible || dismissed) return null

  return (
    <div
      className="rounded-md border border-primary/30 bg-primary/[0.06] px-4 py-3 sm:flex sm:items-start sm:justify-between sm:gap-4 sm:px-5 sm:py-4"
      role="status"
      aria-label="First-run setup hint"
    >
      <div className="flex gap-3 min-w-0">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-primary/25 bg-card/60 text-primary">
          <BookOpen className="h-4 w-4" aria-hidden />
        </div>
        <div className="min-w-0 space-y-1">
          <p className="text-sm font-semibold text-foreground">Configure an ingest transport to unlock live mesh evidence</p>
          <p className="text-xs leading-relaxed text-muted-foreground">
            MEL only shows what your configured serial, TCP, or MQTT paths persist. Unsupported paths (BLE, HTTP ingest) stay
            explicitly out of scope — see project docs before assuming coverage.
          </p>
          <p className="text-xs text-muted-foreground">
            <Link to="/transports" className="font-medium text-primary hover:underline">
              Transports
            </Link>
            {' · '}
            <span className="text-muted-foreground/90">
              Quickstart: <code className="rounded bg-muted/50 px-1 py-0.5 font-mono text-mel-xs">docs/getting-started/QUICKSTART.md</code>
            </span>
            {' · '}
            <span className="text-muted-foreground/90">
              Or seed demo data: <code className="rounded bg-muted/50 px-1 py-0.5 font-mono text-mel-xs">make demo-seed</code>
            </span>
          </p>
        </div>
      </div>
      <button
        type="button"
        onClick={dismiss}
        className="mt-3 inline-flex shrink-0 items-center gap-1 rounded-sm border border-border/60 bg-card/40 px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground sm:mt-0"
        aria-label="Dismiss first-run hint"
      >
        <X className="h-3.5 w-3.5" aria-hidden />
        Dismiss
      </button>
    </div>
  )
}
