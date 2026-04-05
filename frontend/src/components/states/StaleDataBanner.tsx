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
      className="surface-inset mb-3 flex items-start gap-2 border-neon-warn/20 bg-neon-warn/4 p-3"
      role="alert"
    >
      <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0 text-neon-warn" aria-hidden />
      <div className="min-w-0">
        <p className="text-mel-sm font-bold text-foreground">
          <span className="text-neon-warn">[STALE]</span> {componentName}
        </p>
        <p className="mt-0.5 text-mel-xs text-muted-foreground">
          No recent updates. last_verified: <strong className="text-foreground">{formatOperatorTime(lastSuccessfulIngest)}</strong>
        </p>
      </div>
    </div>
  )
}
