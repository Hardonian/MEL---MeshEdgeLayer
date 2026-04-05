import { useRecommendations } from '@/hooks/useApi'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { SystemHealthy } from '@/components/ui/EmptyState'
import { CopyButton } from '@/components/ui/CopyButton'
import { Recommendation } from '@/types/api'
import { Lightbulb, ArrowRight, Zap, BookOpen, Settings, AlertTriangle, RefreshCw } from 'lucide-react'
import { clsx } from 'clsx'


export function Recommendations() {
  const { data, loading, error, refresh } = useRecommendations()

  if (loading && !data) {
    return <Loading message="Analyzing configuration..." />
  }

  if (error && !data) {
    return (
      <div className="p-8">
        <AlertCard
          variant="critical"
          title="Unable to load recommendations"
          description={error}
          action={<button onClick={refresh} className="button-danger">Retry</button>}
        />
      </div>
    )
  }

  const recommendations = data || []
  const actionable = recommendations.filter(r => r.actionable)
  const informational = recommendations.filter(r => !r.actionable)

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <PageHeader
          title="Recommendations"
          description="Suggestions to improve your MEL deployment based on configuration and system state."
        />
        <button onClick={refresh} className="button-secondary">
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      <div className="grid gap-3 sm:grid-cols-3">
        <StatCard
          title="Total"
          value={recommendations.length}
          description="All recommendations"
          icon={<Lightbulb className="h-5 w-5" />}
          variant="default"
        />
        <StatCard
          title="Actionable"
          value={actionable.length}
          description="Requires your attention"
          icon={<ArrowRight className="h-5 w-5" />}
          variant={actionable.length > 0 ? 'warning' : 'success'}
        />
        <StatCard
          title="Informational"
          value={informational.length}
          description="For your awareness"
          icon={<BookOpen className="h-5 w-5" />}
          variant="info"
        />
      </div>

      {/* Actionable recommendations — surface first */}
      {actionable.length > 0 && (
        <Card>
          <CardHeader className="border-b border-border/50 pb-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-md border border-warning/18 bg-warning/12 text-warning">
                  <AlertTriangle className="h-4 w-4" />
                </div>
                <CardTitle className="text-[14px]">Actionable</CardTitle>
              </div>
              <Badge variant="warning">{actionable.length}</Badge>
            </div>
          </CardHeader>
          <CardContent className="pt-3">
            <div className="space-y-2">
              {actionable.map((rec, i) => (
                <RecommendationCard key={`a-${i}`} recommendation={rec} />
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Informational / all */}
      <Card>
        <CardHeader className="border-b border-border/50 pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-[14px]">
              {actionable.length === 0 ? 'All recommendations' : 'Informational'}
            </CardTitle>
            <Badge variant="outline">
              {actionable.length === 0 ? recommendations.length : informational.length}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="pt-3">
          {(actionable.length === 0 ? recommendations : informational).length === 0 ? (
            <SystemHealthy message="No recommendations at this time" />
          ) : (
            <div className="space-y-2">
              {(actionable.length === 0 ? recommendations : informational).map((rec, i) => (
                <RecommendationCard key={`i-${i}`} recommendation={rec} />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function RecommendationCard({ recommendation }: { recommendation: Recommendation }) {
  const categoryIcons: Record<string, React.ReactNode> = {
    configuration: <Settings className="h-3.5 w-3.5" />,
    security: <AlertTriangle className="h-3.5 w-3.5" />,
    performance: <Zap className="h-3.5 w-3.5" />,
    storage: <Lightbulb className="h-3.5 w-3.5" />,
    network: <Lightbulb className="h-3.5 w-3.5" />,
  }

  const defaultIcon = recommendation.actionable ? <ArrowRight className="h-3.5 w-3.5" /> : <Lightbulb className="h-3.5 w-3.5" />

  return (
    <div className={clsx(
      'rounded-md border p-3',
      recommendation.actionable
        ? 'border-warning/25 bg-warning/5'
        : 'border-border/50 bg-card/40'
    )}>
      <div className="flex items-start gap-3">
        <div className={clsx(
          'mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-sm border',
          recommendation.actionable ? 'border-warning/18 bg-warning/12 text-warning' : 'border-border/60 bg-card/60 text-muted-foreground'
        )}>
          {categoryIcons[recommendation.category?.toLowerCase() || ''] || defaultIcon}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-1.5 mb-1.5">
            <Badge variant={recommendation.actionable ? 'warning' : 'outline'}>
              {recommendation.category || 'General'}
            </Badge>
            <Badge variant="secondary">
              {recommendation.priority || 'Normal'}
            </Badge>
          </div>
          <p className="text-[13px] leading-relaxed text-foreground">{recommendation.message}</p>
          {recommendation.action && (
            <div className="mt-2 flex items-center gap-2 rounded-sm border border-border/50 bg-muted/30 px-3 py-2">
              <code className="flex-1 text-xs font-mono text-foreground">{recommendation.action}</code>
              <CopyButton value={recommendation.action} label="Copy command" />
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
