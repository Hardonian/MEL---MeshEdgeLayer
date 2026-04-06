import { useRecommendations } from '@/hooks/useApi'
import { MelDenseRow, MelPanel, MelPanelInset, MelStat } from '@/components/ui/operator'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { AlertCard } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { SystemHealthy } from '@/components/ui/EmptyState'
import { CopyButton } from '@/components/ui/CopyButton'
import { Recommendation } from '@/types/api'
import { Lightbulb, ArrowRight, Zap, Settings, AlertTriangle, RefreshCw } from 'lucide-react'
import { clsx } from 'clsx'
import { MelPanelInset } from '@/components/ui/operator'


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
          description="Assistive posture cues from this instance — bounded heuristics, not canonical transport or incident truth."
        />
        <button onClick={refresh} className="button-secondary">
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      <MelPanel className="p-4">
        <p className="mel-label mb-3">Queue snapshot</p>
        <div className="grid gap-4 sm:grid-cols-3">
          <MelStat label="Total" value={recommendations.length} description="All cues returned" />
          <MelStat
            label="Actionable"
            value={actionable.length}
            description={actionable.length > 0 ? 'Needs a decision or change' : 'Nothing blocking'}
          />
          <MelStat label="Informational" value={informational.length} description="Context / reference only" />
        </div>
      </MelPanel>

      {/* Actionable recommendations — surface first */}
      {actionable.length > 0 && (
        <MelPanel className="overflow-hidden">
          <div className="mel-chrome-title border-b border-border/50 px-3 py-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <span
                  className="flex h-8 w-8 items-center justify-center rounded-md border border-warning/18 bg-warning/12 text-warning"
                  aria-hidden
                >
                  <AlertTriangle className="h-4 w-4" />
                </span>
                <h3 className="text-sm font-semibold text-foreground">Actionable</h3>
              </div>
              <Badge variant="warning">{actionable.length}</Badge>
            </div>
          </div>
          <div className="space-y-2 px-3 py-3">
            {actionable.map((rec, i) => (
              <RecommendationCard key={`a-${i}`} recommendation={rec} />
            ))}
          </div>
        </MelPanel>
      )}

      {/* Informational / all */}
      <MelPanel className="overflow-hidden">
        <div className="mel-chrome-title border-b border-border/50 px-3 py-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h3 className="text-sm font-semibold text-foreground">
              {actionable.length === 0 ? 'All recommendations' : 'Informational'}
            </h3>
            <Badge variant="outline">
              {actionable.length === 0 ? recommendations.length : informational.length}
            </Badge>
          </div>
        </div>
        <div className="px-3 py-3">
          {(actionable.length === 0 ? recommendations : informational).length === 0 ? (
            <SystemHealthy message="No active recommendations" />
          ) : (
            <div className="space-y-2">
              {(actionable.length === 0 ? recommendations : informational).map((rec, i) => (
                <RecommendationCard key={`i-${i}`} recommendation={rec} />
              ))}
            </div>
          )}
        </div>
      </MelPanel>
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
    <MelDenseRow className="p-3" tone={recommendation.actionable ? 'warning' : 'default'}>
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
            <MelPanelInset className="mt-2 flex items-center gap-2 border-border/50 bg-muted/30 py-2">
              <code className="flex-1 text-xs font-mono text-foreground">{recommendation.action}</code>
              <CopyButton value={recommendation.action} label="Copy command" />
            </MelPanelInset>
          )}
        </div>
      </div>
    </MelDenseRow>
  )
}
