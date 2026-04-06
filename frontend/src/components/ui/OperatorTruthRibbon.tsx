import { Link } from 'react-router-dom'
import { clsx } from 'clsx'
import { MelPanelInset } from '@/components/ui/operator'

type OperatorTruthRibbonProps = {
  summary: string
  className?: string
}

export function OperatorTruthRibbon({ summary, className }: OperatorTruthRibbonProps) {
  return (
    <MelPanelInset
      tone="observed"
      className={clsx('flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-3', className)}
      role="note"
      aria-label="Operator truth contract for this view"
    >
      <div className="flex min-w-0 flex-1 items-start gap-2">
        <span className="text-mel-xs font-bold text-signal-observed shrink-0 font-mono uppercase tracking-wide" aria-hidden>
          [TRUTH]
        </span>
        <p className="text-mel-xs text-muted-foreground">{summary}</p>
      </div>
      <Link
        to="/settings#effective-config"
        className="shrink-0 text-mel-xs font-bold text-signal-observed underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background"
      >
        Runtime posture →
      </Link>
    </MelPanelInset>
  )
}
