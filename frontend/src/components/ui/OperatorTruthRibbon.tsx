import { Link } from 'react-router-dom'
import { clsx } from 'clsx'
import { ShieldCheck } from 'lucide-react'

type OperatorTruthRibbonProps = {
  /** Short sentence for this surface (evidence vs inference, no RF/coverage claims). */
  summary: string
  className?: string
}

/**
 * Cross-surface reminder: UI shows stored evidence and explicit semantics — not live RF proof.
 * Links to effective config / posture in Settings for evaluator and operator drill-down.
 */
export function OperatorTruthRibbon({ summary, className }: OperatorTruthRibbonProps) {
  return (
    <div
      className={clsx(
        'flex flex-col gap-2 rounded-xl border border-border/70 bg-muted/20 px-3 py-2.5 sm:flex-row sm:items-center sm:gap-3',
        className,
      )}
      role="note"
      aria-label="Operator truth contract for this view"
    >
      <div className="flex min-w-0 flex-1 items-start gap-2">
        <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
        <p className="text-[11px] leading-relaxed text-muted-foreground sm:text-xs">
          <span className="font-semibold text-foreground">Truth contract: </span>
          {summary}
        </p>
      </div>
      <Link
        to="/settings#effective-config"
        className="shrink-0 text-[11px] font-semibold text-primary hover:underline sm:text-xs"
      >
        Runtime posture &amp; config →
      </Link>
    </div>
  )
}
