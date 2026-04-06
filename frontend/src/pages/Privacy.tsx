import { usePrivacyFindings } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge, SeverityBadge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { EmptyState, SystemHealthy } from '@/components/ui/EmptyState'
import { PrivacyFinding } from '@/types/api'
import type { PlatformPosture } from '@/types/api'
import { AlertTriangle, Info, CheckCircle2, HelpCircle, Lock, Eye, Server, Database } from 'lucide-react'
import { clsx } from 'clsx'
import { Link } from 'react-router-dom'
import { useEffect, useState } from 'react'

export function Privacy() {
  const { data, loading, error, refresh } = usePrivacyFindings()
  const [posture, setPosture] = useState<PlatformPosture | null>(null)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const res = await fetch('/api/v1/platform/posture')
        if (!res.ok) return
        const raw = await res.json()
        if (!cancelled && raw?.platform_posture) {
          setPosture(raw.platform_posture as PlatformPosture)
        }
      } catch {
        // Keep page functional even when posture endpoint is not available.
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  if (loading && !data) {
    return <Loading message="Scanning privacy posture..." />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load privacy findings"
          description={error}
          action={
            <button
              onClick={refresh}
              className="rounded-sm bg-critical px-4 py-2 text-sm font-medium text-white hover:bg-critical/90"
            >
              Retry
            </button>
          }
        />
      </div>
    )
  }

  const findings = data || []
  const critical = findings.filter(f => f.severity === 'critical')
  const high = findings.filter(f => f.severity === 'high')
  const medium = findings.filter(f => f.severity === 'medium')
  const low = findings.filter(f => f.severity === 'low')

  const hasCriticalOrHigh = critical.length > 0 || high.length > 0

  return (
    <div className="space-y-6">
      <PageHeader
        title="Privacy"
        description="Security and privacy posture assessment for your MEL configuration."
      />

      {/* Summary */}
      <div className="grid gap-4 sm:grid-cols-4">
        <StatCard
          title="Critical"
          value={critical.length}
          description="Requires immediate attention"
          icon={<AlertTriangle className="h-5 w-5" />}
          variant={critical.length > 0 ? 'critical' : 'default'}
        />
        <StatCard
          title="High"
          value={high.length}
          description="Should be addressed soon"
          icon={<AlertTriangle className="h-5 w-5" />}
          variant={high.length > 0 ? 'warning' : 'default'}
        />
        <StatCard
          title="Medium"
          value={medium.length}
          description="Consider addressing"
          icon={<Info className="h-5 w-5" />}
          variant="default"
        />
        <StatCard
          title="Low"
          value={low.length}
          description="Informational only"
          icon={<CheckCircle2 className="h-5 w-5" />}
          variant="success"
        />
      </div>

      {/* Overall Status */}
      {findings.length === 0 ? (
        <Card>
          <CardContent className="pt-6">
            <SystemHealthy message="No privacy issues detected" />
            <p className="text-sm text-muted-foreground mt-3 text-center">
              Your MEL configuration passes all privacy checks.
            </p>
          </CardContent>
        </Card>
      ) : hasCriticalOrHigh ? (
        <AlertCard
          variant="critical"
          title={`${critical.length + high.length} privacy finding${critical.length + high.length > 1 ? 's' : ''} require attention`}
          description="Review and address these findings to maintain your privacy posture."
          action={
            <Link
              to="/settings"
              className="text-sm font-medium hover:underline"
            >
              Review settings
            </Link>
          }
        />
      ) : null}

      {/* Privacy Categories Explanation */}
      <Card className="bg-muted/30">
        <CardHeader className="pb-3">
          <div className="flex items-center gap-2">
            <HelpCircle className="h-5 w-5 text-muted-foreground" />
            <CardTitle className="text-base">Understanding Privacy Findings</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="pt-0">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <PrivacyCategory
              icon={<Lock className="h-4 w-4" />}
              title="Encryption"
              description="Data encryption at rest and in transit"
            />
            <PrivacyCategory
              icon={<Eye className="h-4 w-4" />}
              title="Data Collection"
              description="What data MEL stores and why"
            />
            <PrivacyCategory
              icon={<Server className="h-4 w-4" />}
              title="Network"
              description="Network exposure and access controls"
            />
            <PrivacyCategory
              icon={<Database className="h-4 w-4" />}
              title="Storage"
              description="Data retention and cleanup policies"
            />
          </div>
        </CardContent>
      </Card>

      {/* Findings */}
      {posture && (
        <Card>
          <CardHeader className="pb-4">
            <CardTitle>Runtime & Export Posture</CardTitle>
            <CardDescription>Machine-visible privacy/runtime controls currently enforced by backend policy.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <p>Telemetry outbound: <strong>{posture.telemetry_enabled && posture.telemetry_outbound ? 'enabled' : 'disabled'}</strong>.</p>
            <p>Evidence export: <strong>{posture.evidence_export_delete.export_enabled ? 'enabled' : 'disabled'}</strong>; delete APIs: <strong>{posture.evidence_export_delete.delete_enabled ? 'enabled' : 'disabled'}</strong>.</p>
            <p>Assist runtime: <strong>{posture.inference_enabled ? 'configured' : 'disabled'}</strong>; outputs remain non-canonical.</p>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <CardTitle>Privacy Findings</CardTitle>
            <Badge variant="outline">{findings.length} total</Badge>
          </div>
          <CardDescription>
            Active security and privacy concerns for your current configuration
          </CardDescription>
        </CardHeader>
        <CardContent>
          {findings.length === 0 ? (
            <EmptyState
              type="default"
              title="No privacy issues"
              description="All privacy checks have passed."
            />
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
    critical: 'border-critical/30 bg-critical/5',
    high: 'border-warning/30 bg-warning/5',
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
      'rounded-sm border p-4',
      severityColors[finding.severity]
    )}>
      <div className="flex items-start gap-3">
        <div className={clsx(
          'mt-0.5 shrink-0',
          finding.severity === 'critical' ? 'text-critical' :
          finding.severity === 'high' ? 'text-warning' :
          'text-muted-foreground'
        )}>
          <Icon className="h-5 w-5" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2">
            <SeverityBadge severity={finding.severity} />
          </div>
          <p className="text-sm font-medium">{finding.message}</p>
          {finding.remediation && (
            <div className="mel-panel-inset mt-3 border-signal-observed/20 bg-signal-observed/5">
              <p className="text-xs font-medium text-foreground">Recommended Action</p>
              <p className="mt-1 text-sm text-muted-foreground">
                {finding.remediation}
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function PrivacyCategory({ 
  icon, 
  title, 
  description 
}: { 
  icon: React.ReactNode
  title: string
  description: string 
}) {
  return (
    <div className="flex items-start gap-3 p-3 rounded-sm bg-muted/30">
      <div className="shrink-0 text-muted-foreground">
        {icon}
      </div>
      <div>
        <p className="text-sm font-medium">{title}</p>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
    </div>
  )
}
