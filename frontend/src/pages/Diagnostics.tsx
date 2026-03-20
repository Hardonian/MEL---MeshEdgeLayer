import { useDiagnostics } from '@/hooks/useApi'
import { PageHeader } from '@/components/ui/PageHeader'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { AlertTriangle, CheckCircle, Info, Tool } from 'lucide-react'

export function Diagnostics() {
  const { data, loading, error, refresh } = useDiagnostics()

  if (loading && !data) {
    return <Loading message="Running diagnostics..." />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to run diagnostics"
          description={error}
          action={
            <button
              onClick={refresh}
              className="rounded-lg bg-critical px-4 py-2 text-sm font-medium text-white hover:bg-critical/90"
            >
              Retry
            </button>
          }
        />
      </div>
    )
  }

  const findings = data || []
  const criticalCount = findings.filter(f => f.severity === 'critical').length
  const highCount = findings.filter(f => f.severity === 'high').length
  const mediumCount = findings.filter(f => f.severity === 'medium').length

  return (
    <div className="space-y-6">
      <PageHeader
        title="System Diagnostics"
        description="Health checks for your MEL installation."
      />

      <Card>
        <CardHeader>
          <CardTitle>Diagnostics Summary</CardTitle>
          <CardDescription>
            {findings.length} findings found.
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-3">
            <Stat
                icon={<AlertTriangle className="text-critical" />}
                label="Critical"
                value={criticalCount}
            />
            <Stat
                icon={<AlertTriangle className="text-warning" />}
                label="High"
                value={highCount}
            />
            <Stat
                icon={<Info className="text-info" />}
                label="Medium"
                value={mediumCount}
            />
        </CardContent>
      </Card>

      <div className="space-y-4">
        {findings.map((finding, i) => (
          <FindingCard key={i} finding={finding} />
        ))}
      </div>
    </div>
  )
}

function Stat({icon, label, value}: {icon: React.ReactNode, label: string, value: number}) {
    return (
        <div className="flex items-center gap-4 rounded-lg border p-4">
            {icon}
            <div>
                <p className="text-sm text-muted-foreground">{label}</p>
                <p className="text-2xl font-bold">{value}</p>
            </div>
        </div>
    )
}

function FindingCard({ finding }: { finding: { component: string, severity: string, message: string, guidance: string } }) {
  const getSeverityClass = () => {
    switch (finding.severity) {
      case 'critical':
        return 'border-critical/50 bg-critical/5'
      case 'high':
        return 'border-warning/50 bg-warning/5'
      case 'medium':
        return 'border-info/50 bg-info/5'
      default:
        return 'border-muted'
    }
  }

  const getIcon = () => {
    switch (finding.severity) {
        case 'critical':
            return <AlertTriangle className="h-5 w-5 text-critical" />
        case 'high':
            return <AlertTriangle className="h-5 w-5 text-warning" />
        case 'medium':
            return <Info className="h-5 w-5 text-info" />
        default:
            return <CheckCircle className="h-5 w-5 text-success" />
    }
  }

  return (
    <Card className={getSeverityClass()}>
      <CardHeader>
        <div className="flex items-center gap-4">
          {getIcon()}
          <div className="flex-1">
            <CardTitle className="text-base">{finding.component}</CardTitle>
            <Badge variant="outline" className="mt-1">{finding.severity}</Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <p className="mb-2">{finding.message}</p>
        <div className="flex items-start gap-2 rounded-lg bg-muted/50 p-3">
          <Tool className="h-4 w-4 flex-shrink-0 mt-1" />
          <p className="text-sm text-muted-foreground">{finding.guidance}</p>
        </div>
      </CardContent>
    </Card>
  )
}
