import { Link } from 'react-router-dom'
import { clsx } from 'clsx'

type OperatorTruthRibbonProps = {
  summary: string
  className?: string
}

export function OperatorTruthRibbon({ summary, className }: OperatorTruthRibbonProps) {
  return (
    <div
      className={clsx(
        'flex flex-col gap-1 border border-border bg-panel-muted px-3 py-1.5 sm:flex-row sm:items-center sm:gap-2',
        className,
      )}
      role="note"
      aria-label="Operator truth contract for this view"
    >
      <div className="flex min-w-0 flex-1 items-start gap-2">
        <span className="text-mel-xs font-bold text-neon shrink-0">[TRUTH]</span>
        <p className="text-mel-xs text-muted-foreground">{summary}</p>
      </div>
      <Link
        to="/settings#effective-config"
        className="shrink-0 text-mel-xs font-bold text-neon hover:underline"
      >
        → runtime posture
      </Link>
    </div>
  )
}
