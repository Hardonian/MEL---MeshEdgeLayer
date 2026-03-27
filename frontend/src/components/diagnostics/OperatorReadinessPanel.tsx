import { useEffect, useState } from 'react'
import { clsx } from 'clsx'
import { Card } from '@/components/ui/Card'
import { Loading } from '@/components/ui/StateViews'
import { AlertCard } from '@/components/ui/AlertCard'
import { extractTransportsFromStatusJson } from '@/utils/presentation'

type ReadyReasonCode = string

interface ReadyComponent {
  name: string
  state: string
  detail?: string
}

interface ReadyzPayload {
  api_version?: string
  ready?: boolean
  status?: string
  reason_codes?: ReadyReasonCode[]
  summary?: string
  checked_at?: string
  process_ready?: boolean
  ingest_ready?: boolean
  stale_ingest_evidence?: boolean
  operator_state?: string
  mesh_state?: string
  operator_next_steps?: string[]
  components?: ReadyComponent[]
  error_class?: string
  message?: string
}

function pillClass(ok: boolean | undefined, warn?: boolean) {
  if (ok === true) return 'border-success/30 bg-success/10 text-foreground'
  if (warn) return 'border-warning/30 bg-warning/10 text-foreground'
  return 'border-critical/30 bg-critical/10 text-foreground'
}

export function OperatorReadinessPanel() {
  const [phase, setPhase] = useState<'loading' | 'ok' | 'error'>('loading')
  const [err, setErr] = useState<string | null>(null)
  const [readyz, setReadyz] = useState<ReadyzPayload | null>(null)
  const [statusJson, setStatusJson] = useState<unknown>(null)

  useEffect(() => {
    const run = async () => {
      try {
        const [rRes, sRes] = await Promise.all([fetch('/api/v1/readyz'), fetch('/api/v1/status')])
        const rJson = (await rRes.json()) as ReadyzPayload
        setReadyz(rJson)
        if (sRes.ok) {
          setStatusJson(await sRes.json())
        }
        setPhase('ok')
      } catch (e) {
        setPhase('error')
        setErr(e instanceof TypeError ? 'API unreachable — is MEL running and is this page same-origin?' : String(e))
      }
    }
    void run()
  }, [])

  if (phase === 'loading') {
    return (
      <Card className="p-6">
        <Loading message="Loading readiness and status from the API…" />
      </Card>
    )
  }

  if (phase === 'error') {
    return (
      <AlertCard
        variant="critical"
        title="Diagnostics API unreachable"
        description={err ?? 'Unknown error'}
      >
        <p className="mt-2 text-sm text-muted-foreground">
          Quick process liveness only: <code className="rounded bg-muted px-1 font-mono text-xs">GET /healthz</code> (or{' '}
          <code className="rounded bg-muted px-1 font-mono text-xs">mel preflight</code>). Readiness needs a running API:{' '}
          <code className="rounded bg-muted px-1 font-mono text-xs">GET /api/v1/readyz</code>.
        </p>
      </AlertCard>
    )
  }

  const httpReady = readyz?.ready === true
  const idle =
    readyz?.operator_state === 'idle' || (readyz?.reason_codes?.includes('TRANSPORT_IDLE') ?? false)
  const degraded = readyz?.operator_state === 'degraded'
  const transports = extractTransportsFromStatusJson(statusJson)

  return (
    <Card className="space-y-4 p-6">
      <div>
        <h3 className="text-lg font-semibold text-foreground">Setup &amp; readiness</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          Signals come from <code className="rounded bg-muted px-1 font-mono text-xs">/api/v1/readyz</code> and{' '}
          <code className="rounded bg-muted px-1 font-mono text-xs">/api/v1/status</code> — not invented UI state.
        </p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <div className={clsx('rounded-md border p-3 text-sm', pillClass(true))}>
          <div className="font-medium text-foreground">Liveness (process up)</div>
          <p className="mt-1 opacity-90">
            <code className="rounded bg-background/50 px-1 font-mono text-xs">GET /healthz</code> — JSON{' '}
            <code className="rounded bg-background/50 px-1 font-mono text-xs">{'{ ok: true }'}</code> only proves the HTTP process answers.
            Preflight uses this for a quick probe; it does not prove MQTT, serial, or ingest.
          </p>
        </div>
        <div className={clsx('rounded-md border p-3 text-sm', pillClass(httpReady, idle && httpReady))}>
          <div className="font-medium text-foreground">Readiness (subsystem + ingest contract)</div>
          <p className="mt-1 opacity-90">
            <code className="rounded bg-background/50 px-1 font-mono text-xs">GET /readyz</code> and{' '}
            <code className="rounded bg-background/50 px-1 font-mono text-xs">GET /api/v1/readyz</code> share semantics: HTTP <strong>200</strong>{' '}
            when ready, <strong>503</strong> when not (or when the status snapshot cannot be built). Idle (no enabled transports) returns 200 with
            explicit idle reasons.
          </p>
        </div>
      </div>

      <div className="rounded-md border border-border/80 bg-muted/30 p-4 text-sm text-foreground">
        <div className="flex flex-wrap items-center gap-2">
          <span className={clsx('rounded px-2 py-0.5 font-mono text-xs', pillClass(httpReady, idle))}>
            ready={String(readyz?.ready)} · status={readyz?.status ?? '—'}
          </span>
          {readyz?.ingest_ready === false && !idle && (
            <span className="text-xs text-warning">Ingest not proven on any transport</span>
          )}
          {idle && <span className="text-xs text-muted-foreground">Transport idle (no enabled transports)</span>}
          {degraded && <span className="text-xs text-warning">Degraded transport evidence — see status</span>}
          {readyz?.stale_ingest_evidence && <span className="text-xs text-warning">Stale persisted ingest timestamp</span>}
        </div>
        {readyz?.summary && <p className="mt-2 text-muted-foreground">{readyz.summary}</p>}
        {readyz?.reason_codes && readyz.reason_codes.length > 0 && (
          <p className="mt-2 font-mono text-xs text-muted-foreground">reason_codes: {readyz.reason_codes.join(', ')}</p>
        )}
        {readyz?.error_class === 'snapshot_unavailable' && (
          <p className="mt-2 text-critical">
            Readiness evidence unavailable (503). Process is up; fix DB/migrations or paths, then retry.
          </p>
        )}
      </div>

      {transports.length > 0 && (
        <div className="text-sm">
          <div className="mb-2 font-medium text-foreground">Transport truth (from /api/v1/status)</div>
          <ul className="space-y-1 text-muted-foreground">
            {transports.slice(0, 8).map((tr, i) => (
              <li key={i} className="flex flex-wrap gap-2">
                <span className="font-mono text-xs">{tr.name}</span>
                <span className="rounded bg-muted px-1 text-xs">{tr.effective_state}</span>
                {tr.detail && <span className="text-xs">{tr.detail}</span>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {readyz?.components && readyz.components.length > 0 && (
        <div className="text-sm">
          <div className="mb-2 font-medium text-foreground">Components</div>
          <ul className="grid gap-1 sm:grid-cols-2">
            {readyz.components.map((c, i) => (
              <li key={i} className="rounded border border-border/80 bg-card px-2 py-1 text-xs">
                <span className="font-mono">{c.name}</span>: {c.state}
                {c.detail && <span className="text-muted-foreground"> — {c.detail}</span>}
              </li>
            ))}
          </ul>
        </div>
      )}

      {readyz?.operator_next_steps && readyz.operator_next_steps.length > 0 && (
        <div>
          <div className="mb-1 text-sm font-medium text-foreground">Suggested next steps</div>
          <ol className="list-decimal space-y-1 pl-5 text-sm text-muted-foreground">
            {readyz.operator_next_steps.map((s, i) => (
              <li key={i}>{s}</li>
            ))}
          </ol>
        </div>
      )}

      <div className="space-y-1 border-t border-border/80 pt-3 text-xs text-muted-foreground">
        <p>
          <strong className="text-foreground">Deeper diagnosis:</strong> run <code className="font-mono">mel doctor</code> or{' '}
          <code className="font-mono">mel preflight</code> on the host (paths, DB, serial/TCP checks).
        </p>
        <p>
          <strong className="text-foreground">Authoritative transport view:</strong>{' '}
          <code className="font-mono">GET /api/v1/status</code> and <code className="font-mono">mel status --config …</code>.
        </p>
      </div>
    </Card>
  )
}
