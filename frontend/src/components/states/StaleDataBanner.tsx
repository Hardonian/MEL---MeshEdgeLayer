import { AlertCircle } from 'lucide-react'
import { evaluateTimeState, formatOperatorTime } from '@/utils/apiResilience'

interface StaleDataBannerProps {
  lastSuccessfulIngest: string | null | undefined
  thresholdMinutes?: number
  componentName: string
}

export function StaleDataBanner({
  lastSuccessfulIngest,
  thresholdMinutes = 5,
  componentName,
}: StaleDataBannerProps) {
  const thresholdMs = thresholdMinutes * 60 * 1000
  const state = evaluateTimeState(lastSuccessfulIngest, thresholdMs)

  if (state !== 'stale') return null

  return (
    <div
      className="surface-inset mb-4 flex items-start gap-3 rounded-[1rem] border-warning/24 bg-warning/10 p-4"
      role="alert"
    >
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-warning/18 bg-warning/12 text-warning shadow-inset">
        <AlertCircle className="h-5 w-5" aria-hidden />
      </div>
      <div className="min-w-0">
        <h3 className="text-sm font-semibold text-foreground">
          Stale data: {componentName}
        </h3>
        <p className="mt-1 text-sm leading-relaxed text-muted-foreground">
          No recent updates received. Last verified activity:{' '}
          <strong className="font-medium text-foreground">{formatOperatorTime(lastSuccessfulIngest)}</strong>.
        </p>
      </div>
    </div>
  )
}
