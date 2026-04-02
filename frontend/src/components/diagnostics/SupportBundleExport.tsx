import React, { useState } from 'react'
import { Card } from '@/components/ui/Card'
import { AlertCard } from '@/components/ui/AlertCard'
import { clsx } from 'clsx'
import { useVersionInfo } from '@/hooks/useVersionInfo'
import { operatorExportReadinessFromVersion } from '@/utils/operatorExportReadiness'

export const SupportBundleExport: React.FC = () => {
  const versionInfo = useVersionInfo()
  const exportReadiness = operatorExportReadinessFromVersion(versionInfo.data, versionInfo.error ?? null)
  const [status, setStatus] = useState<'idle' | 'generating' | 'success' | 'error' | 'unavailable'>('idle')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const handleExport = async () => {
    setStatus('generating')
    setErrorMessage(null)

    try {
      const response = await fetch('/api/v1/support-bundle')
      if (response.status === 404 || response.status === 501) {
        setStatus('unavailable')
        setErrorMessage('Support bundle export API is currently disabled or not implemented in this backend version.')
        return
      }
      if (response.status === 401 || response.status === 403) {
        throw new Error(
          `${response.status}: authentication or capability required (export_support_bundle). Use mel support bundle --config … from the host if the UI cannot authorize.`
        )
      }
      if (!response.ok) {
        throw new Error(`Export failed: ${response.status} ${response.statusText}`)
      }

      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `mel-support-bundle-${new Date().toISOString().replace(/[:.]/g, '-')}.zip`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)

      setStatus('success')
      setTimeout(() => setStatus('idle'), 5000)
    } catch (err) {
      setStatus('error')
      setErrorMessage(
        err instanceof TypeError
          ? 'MEL backend unreachable (Network Error).'
          : err instanceof Error
            ? err.message
            : 'Unknown network error'
      )
    }
  }

  return (
    <Card className="p-6">
      <h3 className="mb-2 text-lg font-medium text-foreground">Export support bundle</h3>
      <p className="mb-4 text-sm text-muted-foreground">
        Downloads a ZIP with <code className="rounded bg-muted px-1 font-mono text-xs">bundle.json</code> (redacted config, nodes, messages sample,
        control evidence) and <code className="rounded bg-muted px-1 font-mono text-xs">doctor.json</code> (same checks as{' '}
        <code className="rounded bg-muted px-1 font-mono text-xs">mel doctor</code>, bundle-redacted). Review before sharing externally.
      </p>
      {!versionInfo.loading && (
        <div
          className={clsx(
            'mb-4 rounded-lg border px-3 py-2 text-xs',
            exportReadiness.semantic === 'available'
              ? 'border-success/25 bg-success/5 text-muted-foreground'
              : exportReadiness.semantic === 'policy_limited'
                ? 'border-critical/30 bg-critical/5 text-foreground'
                : exportReadiness.semantic === 'degraded'
                  ? 'border-warning/30 bg-warning/5 text-foreground'
                  : 'border-warning/25 bg-warning/5 text-foreground',
          )}
          role="status"
          aria-live="polite"
          data-testid="support-bundle-export-readiness"
        >
          <span className="font-semibold text-foreground">Export / artifact readiness (same as proofpack): </span>
          {exportReadiness.summary}
          {exportReadiness.blockers.length > 0 && (
            <ul className="mt-1.5 list-disc pl-4 text-[11px] text-muted-foreground space-y-0.5">
              {exportReadiness.blockers.map((b) => (
                <li key={b.code}>
                  <span className="font-mono text-foreground/80">{b.code}</span>
                  {b.summary ? ` — ${b.summary}` : ''}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
      {exportReadiness.semantic === 'unknown_partial' && (
        <div className="mb-4">
          <AlertCard
            variant="warning"
            title="Instance export posture not fully loaded"
            description={`${exportReadiness.summary} You can still download this host bundle; confirm GET /api/v1/version if proofpack parity matters.`}
          />
        </div>
      )}
      {exportReadiness.semantic === 'policy_limited' && (
        <div className="mb-4">
          <AlertCard
            variant="critical"
            title="Instance policy blocks evidence-class exports"
            description={`${exportReadiness.summary} This ZIP is host/support continuity — do not treat it as a substitute for incident proofpack when policy blocks export.`}
          />
        </div>
      )}
      <button
        type="button"
        onClick={() => void handleExport()}
        disabled={status === 'generating' || status === 'unavailable'}
        className={clsx(
          'inline-flex items-center rounded-md border border-transparent px-4 py-2 text-sm font-medium text-primary-foreground shadow-sm transition-colors',
          'bg-primary hover:bg-primary/90',
          'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
          'disabled:cursor-not-allowed disabled:opacity-50'
        )}
      >
        {status === 'generating' ? 'Generating bundle…' : 'Download redacted bundle'}
      </button>
      {(status === 'error' || status === 'unavailable') && errorMessage && (
        <div className="mt-3">
          <AlertCard
            variant={status === 'unavailable' ? 'info' : 'critical'}
            title={status === 'unavailable' ? 'Export unavailable' : 'Export failed'}
            description={errorMessage}
          />
        </div>
      )}
      {status === 'success' && (
        <p className="mt-3 text-sm text-success" role="status">
          Download started. Check your browser downloads folder.
        </p>
      )}
    </Card>
  )
}
