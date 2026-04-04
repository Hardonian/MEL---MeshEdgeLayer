import { useEffect, useState } from 'react'
import { OperatorReadinessPanel } from '@/components/diagnostics/OperatorReadinessPanel'
import { SupportBundleExport } from '@/components/diagnostics/SupportBundleExport'
import { OperatorEmptyState } from '@/components/states/OperatorEmptyState'
import {
  parseDiagnosticsFindingsFromApi,
  safeArray,
  type DiagnosticsApiFinding,
} from '@/utils/apiResilience'
import { PageHeader } from '@/components/ui/PageHeader'
import { Loading } from '@/components/ui/StateViews'
import { AlertCard } from '@/components/ui/AlertCard'
import { Card } from '@/components/ui/Card'
import { clsx } from 'clsx'

type DiagnosticsPageState = 'loading' | 'unreachable' | 'disabled' | 'ready'

export function Diagnostics() {
  const [findings, setFindings] = useState<DiagnosticsApiFinding[]>([])
  const [pageState, setPageState] = useState<DiagnosticsPageState>('loading')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchDiagnostics = async () => {
      try {
        const res = await fetch('/api/v1/diagnostics')
        if (res.status === 404 || res.status === 501) {
          setPageState('disabled')
          return
        }
        if (!res.ok) throw new Error(`HTTP ${res.status}: Failed to fetch diagnostics`)
        const data = await res.json()
        setFindings(parseDiagnosticsFindingsFromApi(data))
        setPageState('ready')
      } catch (err) {
        setPageState('unreachable')
        setError(
          err instanceof TypeError
            ? 'Backend is unreachable (Network Error). Is MEL running?'
            : (err as Error).message
        )
      }
    }
    void fetchDiagnostics()
  }, [])

  if (pageState === 'loading') {
    return (
      <div className="space-y-6">
        <PageHeader
          title="System Diagnostics"
          subtitle="Mesh operations cockpit"
          description="Liveness vs readiness vs deep checks — wired to live API responses."
        />
        <Loading message="Running system diagnostics…" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="System Diagnostics"
        subtitle="Mesh operations cockpit"
        description="Liveness vs readiness vs deep checks — wired to live API responses."
      />

      <OperatorReadinessPanel />

      <div className="mb-2">
        <SupportBundleExport />
      </div>

      {pageState === 'unreachable' && error && (
        <AlertCard variant="critical" title="Connection failure" description={error} />
      )}

      {pageState === 'disabled' && (
        <AlertCard
          variant="info"
          title="Diagnostics API unavailable"
          description="The running MEL build does not expose GET /api/v1/diagnostics. Use mel doctor on the host for deep checks."
        />
      )}

      <div className="space-y-4">
        {pageState === 'ready' && findings.length === 0 ? (
          <OperatorEmptyState
            title="No diagnostic findings"
            description="MEL reports zero active diagnostic findings from this endpoint."
          />
        ) : (
          pageState === 'ready' &&
          findings.map((f, idx) => {
            const sev =
              f.severity === 'critical' || f.severity === 'warning' || f.severity === 'info'
                ? f.severity
                : 'info'
            return (
            <Card
              key={idx}
              className={clsx(
                'p-5',
                sev === 'critical' && 'border-critical/30 bg-critical/5',
                sev === 'warning' && 'border-warning/30 bg-warning/5',
                sev === 'info' && 'border-info/25 bg-info/5'
              )}
            >
              <div className="mb-2 flex items-start justify-between gap-3">
                <h4
                  className={clsx(
                    'text-lg font-semibold',
                    sev === 'critical' && 'text-critical',
                    sev === 'warning' && 'text-warning',
                    sev === 'info' && 'text-info'
                  )}
                >
                  {f.title}
                </h4>
                <span className="rounded-full border border-border bg-muted/50 px-2 py-1 font-mono text-xs uppercase text-muted-foreground">
                  {f.code}
                </span>
              </div>
              <p className="mb-4 text-sm text-muted-foreground">{f.explanation}</p>
              <div className="mb-1 text-sm font-medium text-foreground">Recommended action</div>
              <ul className="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
                {safeArray(f.recommended_steps).map((step, sIdx) => (
                  <li key={sIdx}>{step}</li>
                ))}
              </ul>
            </Card>
            )
          })
        )}
      </div>
    </div>
  )
}
