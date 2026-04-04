import type { Incident } from '@/types/api'

function toWords(v: string | undefined): string {
  return (v || '').replace(/_/g, ' ').trim()
}

function replayChangeLine(inc: Incident): string | null {
  const replay = inc.replay_summary
  if (!replay) return null
  if (typeof replay.delta_total !== 'number') return replay.summary || null
  const delta = `${replay.delta_total >= 0 ? '+' : ''}${replay.delta_total}`
  return `Δ ${delta} events (${replay.recent_count ?? 0} recent / ${replay.prior_count ?? 0} prior)`
}

export function IncidentRationaleSummary({
  incident,
  fallbackWhy,
  className,
}: {
  incident: Incident
  fallbackWhy?: string
  className?: string
}) {
  const queue = incident.decision_pack?.queue
  const guidance = incident.decision_pack?.guidance

  const whySurfaced = queue?.why_surfaced_one_liner?.trim() || fallbackWhy?.trim()
  const whyNow = guidance?.why_now?.trim()
  const changedLine = replayChangeLine(incident)
  const replaySummary = guidance?.replay_summary?.trim() || incident.replay_summary?.summary?.trim()
  const degradedReasons = guidance?.degraded ? (guidance.degraded_reasons ?? []) : []

  if (!whySurfaced && !whyNow && !changedLine && !replaySummary && degradedReasons.length === 0) {
    return (
      <div className={`rounded-lg border border-warning/30 bg-warning/5 px-3 py-2 text-xs text-muted-foreground ${className || ''}`} role="status">
        <span className="font-semibold text-foreground">Backend rationale unavailable: </span>
        This row has no decision-pack why/change fields in this response. Treat ordering as partial and open detail for evidence basis.
      </div>
    )
  }

  return (
    <section className={`rounded-lg border border-border/50 bg-muted/15 px-3 py-2 text-xs space-y-1.5 ${className || ''}`} data-testid="incident-rationale-summary">
      {whySurfaced && (
        <p>
          <span className="font-semibold text-foreground">Why surfaced: </span>
          <span className="text-muted-foreground">{whySurfaced}</span>
        </p>
      )}
      {whyNow && (
        <p>
          <span className="font-semibold text-foreground">Why this matters now: </span>
          <span className="text-muted-foreground">{whyNow}</span>
        </p>
      )}
      {(changedLine || replaySummary) && (
        <p>
          <span className="font-semibold text-foreground">What changed: </span>
          <span className="text-muted-foreground">{changedLine || replaySummary}</span>
        </p>
      )}
      {degradedReasons.length > 0 && (
        <p className="text-warning">
          <span className="font-semibold">Guidance degraded: </span>
          {degradedReasons.slice(0, 3).map(toWords).join(', ')}
        </p>
      )}
      <p className="text-[10px] text-muted-foreground/80">Backend-computed rationale and replay posture; bounded guidance, not delivery/path proof.</p>
    </section>
  )
}
