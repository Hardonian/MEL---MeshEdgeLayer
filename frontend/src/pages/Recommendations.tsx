import { useRecommendations } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { PageHeader } from '@/components/ui/PageHeader'
import { Badge } from '@/components/ui/Badge'
import { AlertCard, InlineAlert } from '@/components/ui/AlertCard'
import { Loading, EmptyState, SystemHealthy } from '@/components/ui/EmptyState'
import { Recommendation } from '@/types/api'
import { Lightbulb, CheckCircle2, ArrowRight, Zap, BookOpen, Settings, AlertTriangle } from 'lucide-react'
import { clsx } from 'clsx'
import { Link } from 'react-router-dom'

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

  const recommendations = data || []
  const actionable = recommendations.filter(r => r.actionable)
  const informational = recommendations.filter(r => !r.actionable)

  return (
    <div className="space-y-6">
      <PageHeader
        title="Recommendations"
        description="Suggestions to improve your MEL deployment based on your configuration and system state."
      />

      {/* Summary */}
      <div className="grid gap-4 sm:grid-cols-3">
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

      {/* Actionable Alert */}
      {actionable.length > 0 && (
        <AlertCard
          variant="warning"
          title={`${actionable.length} actionable recommendation${actionable.length > 1 ? 's' : ''}`}
          description="These items require your attention to improve your MEL deployment."
        />
      )}

      {/* Recommendations Categories */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <CardTitle>All Recommendations</CardTitle>
            <Badge variant="outline">{recommendations.length} total</Badge>
          </div>
          <CardDescription>
            Configuration suggestions based on your current setup
          </CardDescription>
        </CardHeader>
        <CardContent>
          {recommendations.length === 0 ? (
            <SystemHealthy message="No recommendations at this time" />
          ) : (
            <div className="space-y-4">
              {recommendations.map((rec, i) => (
                <RecommendationCard key={i} recommendation={rec} />
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
    configuration: <Settings className="h-4 w-4" />,
    security: <AlertTriangle className="h-4 w-4" />,
    performance: <Zap className="h-4 w-4" />,
    storage: <Lightbulb className="h-4 w-4" />,
    network: <Lightbulb className="h-4 w-4" />,
  }

  const defaultIcon = recommendation.actionable ? <ArrowRight className="h-4 w-4" /> : <Lightbulb className="h-4 w-4" />

  return (
    <div className={clsx(
      'rounded-lg border p-4',
      recommendation.actionable 
        ? 'border-warning/30 bg-warning/5' 
        : 'border-muted bg-muted/30'
    )}>
      <div className="flex items-start gap-3">
        <div className={clsx(
          'mt-0.5 shrink-0',
          recommendation.actionable ? 'text-warning' : 'text-muted-foreground'
        )}>
          {categoryIcons[recommendation.category?.toLowerCase() || ''] || defaultIcon}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2 flex-wrap">
            <Badge variant={recommendation.actionable ? 'warning' : 'outline'}>
              {recommendation.category || 'General'}
            </Badge>
            <Badge variant="secondary">
              {recommendation.priority || 'Normal'}
            </Badge>
            {recommendation.actionable && (
              <Badge variant="warning">Actionable</Badge>
            )}
          </div>
          <p className="text-sm font-medium">{recommendation.message}</p>
          {recommendation.action && (
            <div className="mt-3 p-3 rounded-md bg-muted/50">
              <p className="text-xs font-medium text-muted-foreground mb-1">Suggested Command</p>
              <code className="text-xs font-mono block text-foreground">
                {recommendation.action}
              </code>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
