import { useRecommendations } from '@/hooks/useApi'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { Loading, ErrorView, EmptyState } from '@/components/ui/StateViews'
import { Recommendation } from '@/types/api'
import { Lightbulb, CheckCircle2, ArrowRight } from 'lucide-react'
import { clsx } from 'clsx'

export function Recommendations() {
  const { data, loading, error, refresh } = useRecommendations()

  if (loading && !data) {
    return <Loading message="Analyzing configuration..." />
  }

  if (error && !data) {
    return <ErrorView message={error} onRetry={refresh} />
  }

  const recommendations = data || []
  const actionable = recommendations.filter(r => r.actionable)
  const informational = recommendations.filter(r => !r.actionable)

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Recommendations</h1>
        <p className="text-muted-foreground">
          Suggestions to improve your MEL deployment.
        </p>
      </div>

      {/* Summary */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10">
                <Lightbulb className="h-6 w-6 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold">{recommendations.length}</p>
                <p className="text-sm text-muted-foreground">Total</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-warning/10">
                <ArrowRight className="h-6 w-6 text-warning" />
              </div>
              <div>
                <p className="text-2xl font-bold">{actionable.length}</p>
                <p className="text-sm text-muted-foreground">Actionable</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-secondary">
                <CheckCircle2 className="h-6 w-6 text-secondary-foreground" />
              </div>
              <div>
                <p className="text-2xl font-bold">{informational.length}</p>
                <p className="text-sm text-muted-foreground">Informational</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Recommendations List */}
      <Card>
        <CardHeader>
          <CardTitle>All Recommendations</CardTitle>
          <CardDescription>
            Configuration suggestions based on your current setup
          </CardDescription>
        </CardHeader>
        <CardContent>
          {recommendations.length === 0 ? (
            <EmptyState
              icon={<CheckCircle2 className="h-10 w-10" />}
              title="No recommendations"
              description="Your configuration looks good. No recommendations at this time."
            />
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
  return (
    <div className={clsx(
      'rounded-lg border p-4',
      recommendation.actionable ? 'border-warning/20 bg-warning/5' : 'border-muted bg-muted/30'
    )}>
      <div className="flex items-start gap-3">
        {recommendation.actionable ? (
          <ArrowRight className="h-5 w-5 text-warning mt-0.5" />
        ) : (
          <Lightbulb className="h-5 w-5 text-muted-foreground mt-0.5" />
        )}
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <Badge variant={recommendation.actionable ? 'warning' : 'outline'}>
              {recommendation.category || 'General'}
            </Badge>
            <Badge variant="secondary">
              {recommendation.priority || 'Normal'}
            </Badge>
          </div>
          <p className="text-sm font-medium">{recommendation.message}</p>
          {recommendation.action && (
            <p className="mt-2 text-sm font-mono bg-muted p-2 rounded">
              {recommendation.action}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}
