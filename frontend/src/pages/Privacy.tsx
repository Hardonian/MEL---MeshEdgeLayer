import { usePrivacyFindings } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { SeverityBadge } from '@/components/ui/Badge'
import { Loading, ErrorView } from '@/components/ui/StateViews'
import { PrivacyFinding } from '@/types/api'
import { Shield, CheckCircle2, AlertTriangle, Info } from 'lucide-react'
import { clsx } from 'clsx'

export function Privacy() {
  const { data, loading, error, refresh } = usePrivacyFindings()

  if (loading && !data) {
    return <Loading message="Scanning privacy posture..." />
  }

  if (error && !data) {
    return <ErrorView message={error} onRetry={refresh} />
  }

  const findings = data || []
  const critical = findings.filter(f => f.severity === 'critical')
  const high = findings.filter(f => f.severity === 'high')
  const medium = findings.filter(f => f.severity === 'medium')
  const low = findings.filter(f => f.severity === 'low')

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Privacy</h1>
        <p className="text-muted-foreground">
          Security and privacy posture assessment for your MEL configuration.
        </p>
      </div>

      {/* Summary */}
      <div className="grid gap-4 sm:grid-cols-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-critical/10">
                <AlertTriangle className="h-6 w-6 text-critical" />
              </div>
              <div>
                <p className="text-2xl font-bold">{critical.length}</p>
                <p className="text-sm text-muted-foreground">Critical</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-warning/10">
                <AlertTriangle className="h-6 w-6 text-warning" />
              </div>
              <div>
                <p className="text-2xl font-bold">{high.length}</p>
                <p className="text-sm text-muted-foreground">High</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-secondary">
                <Info className="h-6 w-6 text-secondary-foreground" />
              </div>
              <div>
                <p className="text-2xl font-bold">{medium.length}</p>
                <p className="text-sm text-muted-foreground">Medium</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-success/10">
                <CheckCircle2 className="h-6 w-6 text-success" />
              </div>
              <div>
                <p className="text-2xl font-bold">{low.length}</p>
                <p className="text-sm text-muted-foreground">Low</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Findings */}
      <Card>
        <CardHeader>
          <CardTitle>Privacy Findings</CardTitle>
          <CardDescription>
            Active security and privacy concerns for your current configuration
          </CardDescription>
        </CardHeader>
        <CardContent>
          {findings.length === 0 ? (
            <div className="flex flex-col items-center gap-4 py-8 text-center">
              <div className="flex h-16 w-16 items-center justify-center rounded-full bg-success/10">
                <Shield className="h-8 w-8 text-success" />
              </div>
              <div>
                <h3 className="font-semibold text-foreground">No privacy issues detected</h3>
                <p className="mt-1 text-sm text-muted-foreground">
                  Your MEL configuration passes all privacy checks.
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {findings.map((finding, i) => (
                <FindingCard key={i} finding={finding} />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function FindingCard({ finding }: { finding: PrivacyFinding }) {
  const severityColors = {
    critical: 'border-critical/20 bg-critical/5',
    high: 'border-warning/20 bg-warning/5',
    medium: 'border-warning/20 bg-warning/5',
    low: 'border-muted bg-muted/30',
  }

  const severityIcons = {
    critical: AlertTriangle,
    high: AlertTriangle,
    medium: Info,
    low: Info,
  }

  const Icon = severityIcons[finding.severity]

  return (
    <div className={clsx(
      'rounded-lg border p-4',
      severityColors[finding.severity]
    )}>
      <div className="flex items-start gap-3">
        <Icon className={clsx(
          'h-5 w-5 mt-0.5',
          finding.severity === 'critical' ? 'text-critical' :
          finding.severity === 'high' ? 'text-warning' :
          'text-muted-foreground'
        )} />
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <SeverityBadge severity={finding.severity} />
          </div>
          <p className="text-sm font-medium">{finding.message}</p>
          {finding.remediation && (
            <p className="mt-2 text-sm text-muted-foreground">
              <span className="font-medium">Remediation:</span> {finding.remediation}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}
