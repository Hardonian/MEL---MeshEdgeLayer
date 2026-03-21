import { Link } from 'react-router-dom'
import {
  Activity,
  Radio,
  MessageSquare,
  Shield,
  AlertTriangle,
  ArrowRight,
  CheckCircle2,
  XCircle,
  TrendingUp,
  AlertCircle,
  Zap,
  Clock,
} from 'lucide-react'
import { clsx } from 'clsx'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { StatCard } from '@/components/ui/StatCard'
import { Badge, HealthBadge, SeverityBadge } from '@/components/ui/Badge'
import { AlertCard, InlineAlert } from '@/components/ui/AlertCard'
import { Loading } from '@/components/ui/StateViews'
import { StaleDataBanner } from '@/components/states/StaleDataBanner'
import { EmptyState, SystemHealthy, NoTransportsConfigured } from '@/components/ui/EmptyState'
import { useStatus, useNodes, useMessages, usePrivacyFindings, useRecommendations, useDeadLetters } from '@/hooks/useApi'
import { getHealthState, formatRelativeTime, TransportHealth } from '@/types/api'

export function Dashboard() {
  const status = useStatus()
  const nodes = useNodes()
  const { data: messagesData, loading: messagesLoading, error: messagesError } = useMessages()
  const privacy = usePrivacyFindings()
  const recommendations = useRecommendations()
  const deadLetters = useDeadLetters()

  const isLoading = status.loading || nodes.loading || messagesLoading
  const hasError = status.error || nodes.error || messagesError

  if (isLoading && !status.data) {
    return <Loading message="Loading system status..." />
  }

  if (hasError && !status.data) {
    return (
      <div className="p-8 animate-fade-in">
        <AlertCard
          variant="critical"
          title="Unable to connect to MEL backend"
          description={status.error || 'Failed to connect to MEL backend. Please ensure MEL is running.'}
          action={
            <button
              onClick={() => window.location.reload()}
              className="rounded-lg bg-critical px-4 py-2 text-sm font-medium text-white hover:bg-critical/90 transition-colors shadow-lg shadow-critical/20"
            >
              Retry Connection
            </button>
          }
        />
      </div>
    )
  }

  const transports = status.data?.transports || []
  const connectedTransport = transports.find(t => t.effective_state === 'connected')
  const healthyTransports = transports.filter(t => getHealthState(t.health) === 'healthy').length
  const totalTransports = transports.length
  const hasTransports = totalTransports > 0

  const activePrivacyFindings = privacy.data?.filter(p => p.severity === 'critical' || p.severity === 'high') || []
  const pendingRecommendations = recommendations.data?.filter(r => r.actionable) || []

  const newestHeartbeat = transports.reduce((max, t) => {
    if (!t.last_heartbeat_at) return max
    const ts = new Date(t.last_heartbeat_at).getTime()
    return ts > max ? ts : max
  }, 0)
  const dashboardStaleTs = newestHeartbeat ? new Date(newestHeartbeat).toISOString() : undefined

  return (
    <div className="space-y-8 animate-fade-in pb-12">
      <div className="flex flex-col gap-1">
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">Real-time overview of your MeshEdgeLayer observability system.</p>
      </div>

      <StaleDataBanner lastSuccessfulIngest={dashboardStaleTs} componentName="Dashboard / Transports" />

      {/* Quick Stats - Enhanced Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 stagger-children">
        <div className="card-hover">
          <StatCard
            title="Connection Status"
            value={connectedTransport ? 'Connected' : 'Disconnected'}
            description={connectedTransport ? connectedTransport.name : 'No active transport'}
            icon={connectedTransport ? <CheckCircle2 className="h-5 w-5" /> : <XCircle className="h-5 w-5" />}
            variant={connectedTransport ? 'success' : 'warning'}
          />
        </div>

        <div className="card-hover">
          <StatCard
            title="Nodes"
            value={nodes.data?.length || 0}
            description={nodes.data?.length === 0 ? 'Awaiting mesh observations' : 'Mesh devices detected'}
            icon={<Radio className="h-5 w-5" />}
            variant="default"
          />
        </div>

        <div className="card-hover">
          <StatCard
            title="Messages"
            value={messagesData?.length || status.data?.messages || 0}
            description="Runtime message count"
            icon={<MessageSquare className="h-5 w-5" />}
            variant="info"
          />
        </div>

        <div className="card-hover">
          <StatCard
            title="Transport Health"
            value={`${healthyTransports}/${totalTransports}`}
            description={hasTransports ? 'Healthy transport connections' : 'No transports configured'}
            icon={<TrendingUp className="h-5 w-5" />}
            variant={healthyTransports === totalTransports && hasTransports ? 'success' : healthyTransports > 0 ? 'warning' : 'default'}
          />
        </div>
      </div>

      {/* System Status Alert - shows if there's something to alert about */}
      {(activePrivacyFindings.length > 0 || pendingRecommendations.length > 0 || (deadLetters.data && deadLetters.data.length > 0)) && (
        <AlertCard
          variant={activePrivacyFindings.length > 0 ? 'critical' : pendingRecommendations.length > 0 ? 'warning' : 'info'}
          title={
            activePrivacyFindings.length > 0
              ? `${activePrivacyFindings.length} critical privacy finding${activePrivacyFindings.length > 1 ? 's' : ''} require attention`
              : pendingRecommendations.length > 0
              ? `${pendingRecommendations.length} recommendation${pendingRecommendations.length > 1 ? 's' : ''} pending`
              : 'System status'
          }
          description={
            activePrivacyFindings.length > 0
              ? 'Review and address these findings to maintain your privacy posture.'
              : pendingRecommendations.length > 0
              ? 'Review recommendations to optimize your MEL deployment.'
              : undefined
          }
          action={
            <Link
              to="/recommendations"
              className="flex items-center gap-1 text-sm font-medium hover:underline"
            >
              View all <ArrowRight className="h-4 w-4" />
            </Link>
          }
        />
      )}

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Transport Status */}
        <Card className="overflow-hidden">
          <CardHeader className="pb-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
                  <Activity className="h-4 w-4 text-primary" />
                </div>
                <div>
                  <CardTitle className="text-base">Transport Status</CardTitle>
                  <CardDescription className="text-xs mt-0.5">Health status of configured transports</CardDescription>
                </div>
              </div>
              <Link
                to="/status"
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                View all <ArrowRight className="h-3 w-3" />
              </Link>
            </div>
          </CardHeader>
          <CardContent className="pt-0">
            {!hasTransports ? (
              <NoTransportsConfigured />
            ) : (
              <div className="space-y-3">
                {transports.slice(0, 4).map((transport) => (
                  <TransportListItem key={transport.name} transport={transport} />
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Privacy Findings */}
        <Card className="overflow-hidden">
          <CardHeader className="pb-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-success/10">
                  <Shield className="h-4 w-4 text-success" />
                </div>
                <div>
                  <CardTitle className="text-base">Privacy Posture</CardTitle>
                  <CardDescription className="text-xs mt-0.5">Security and privacy assessment</CardDescription>
                </div>
              </div>
              <Link
                to="/privacy"
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                View all <ArrowRight className="h-3 w-3" />
              </Link>
            </div>
          </CardHeader>
          <CardContent className="pt-0">
            {privacy.loading ? (
              <Loading message="Scanning..." />
            ) : activePrivacyFindings.length === 0 ? (
              <SystemHealthy message="No critical privacy issues detected" />
            ) : (
              <div className="space-y-3">
                {activePrivacyFindings.slice(0, 3).map((finding, i) => (
                  <InlineAlert key={i} variant={finding.severity === 'critical' ? 'critical' : 'warning'}>
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-sm truncate flex-1">{finding.message}</span>
                      <SeverityBadge severity={finding.severity} />
                    </div>
                  </InlineAlert>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent Nodes */}
        <Card className="overflow-hidden">
          <CardHeader className="pb-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-secondary">
                  <Radio className="h-4 w-4 text-secondary-foreground" />
                </div>
                <div>
                  <CardTitle className="text-base">Recent Nodes</CardTitle>
                  <CardDescription className="text-xs mt-0.5">Recently observed mesh devices</CardDescription>
                </div>
              </div>
              <Link
                to="/nodes"
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                View all <ArrowRight className="h-3 w-3" />
              </Link>
            </div>
          </CardHeader>
          <CardContent className="pt-0">
            {nodes.loading ? (
              <Loading message="Loading nodes..." />
            ) : nodes.data?.length === 0 ? (
              <EmptyState
                type="no-data"
                title="No nodes yet"
                description="Nodes will appear here once mesh traffic is observed."
              />
            ) : (
              <div className="space-y-2">
                {nodes.data?.slice(0, 5).map((node) => (
                  <NodeListItem key={node.id} node={node} />
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recommendations */}
        <Card className="overflow-hidden">
          <CardHeader className="pb-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-50 dark:bg-amber-950">
                  <Zap className="h-4 w-4 text-amber-600 dark:text-amber-400" />
                </div>
                <div>
                  <CardTitle className="text-base">Recommendations</CardTitle>
                  <CardDescription className="text-xs mt-0.5">Actions to improve your setup</CardDescription>
                </div>
              </div>
              <Link
                to="/recommendations"
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                View all <ArrowRight className="h-3 w-3" />
              </Link>
            </div>
          </CardHeader>
          <CardContent className="pt-0">
            {recommendations.loading ? (
              <Loading message="Analyzing..." />
            ) : pendingRecommendations.length === 0 ? (
              <SystemHealthy message="No pending recommendations" />
            ) : (
              <div className="space-y-2">
                {pendingRecommendations.slice(0, 4).map((rec, i) => (
                  <RecommendationListItem key={i} recommendation={rec} />
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Dead Letters Alert - Conditional */}
      {deadLetters.data && deadLetters.data.length > 0 && (
        <Card className="border-amber-200 bg-amber-50/50 dark:border-amber-800 dark:bg-amber-950/30">
          <CardHeader className="pb-4">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-400" />
              <div>
                <CardTitle className="text-base">Recent Dead Letters</CardTitle>
                <CardDescription className="text-xs mt-0.5">Messages that failed to be processed</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="pt-0">
            <div className="space-y-2">
              {deadLetters.data.slice(0, 3).map((dl, i) => (
                <div key={i} className="flex items-center justify-between rounded-lg border border-amber-200/50 bg-white/50 p-3 dark:border-amber-800/50 dark:bg-black/20">
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{dl.transport_name}</p>
                    <p className="truncate text-xs text-muted-foreground">{dl.reason}</p>
                  </div>
                  <div className="ml-4 shrink-0 text-xs text-muted-foreground">
                    {formatRelativeTime(dl.created_at)}
                  </div>
                </div>
              ))}
            </div>
            <Link
              to="/dead-letters"
              className="mt-4 flex items-center gap-1 text-sm font-medium text-amber-700 hover:underline dark:text-amber-400"
            >
              View all dead letters <ArrowRight className="h-4 w-4" />
            </Link>
          </CardContent>
        </Card>
      )}
    </div>
  )
}



function TransportListItem({ transport }: { transport: TransportHealth }) {
  const healthState = getHealthState(transport.health)
  
  return (
    <div className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-muted/50">
      <div className="flex items-center gap-3 min-w-0">
        <div className={clsx(
          'flex h-9 w-9 shrink-0 items-center justify-center rounded-full',
          healthState === 'healthy' ? 'bg-success/10' :
          healthState === 'degraded' ? 'bg-warning/10' :
          'bg-critical/10'
        )}>
          <Activity className={clsx(
            'h-4 w-4',
            healthState === 'healthy' ? 'text-success' :
            healthState === 'degraded' ? 'text-warning' :
            'text-critical'
          )} />
        </div>
        <div className="min-w-0">
          <p className="truncate font-medium text-sm">{transport.name}</p>
          <p className="truncate text-xs text-muted-foreground">{transport.type}</p>
        </div>
      </div>
      <div className="ml-3 flex flex-col items-end gap-1">
        <HealthBadge health={healthState} />
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <Clock className="h-3 w-3" />
          {transport.last_heartbeat_at ? formatRelativeTime(transport.last_heartbeat_at) : 'Never'}
        </div>
      </div>
    </div>
  )
}

function NodeListItem({ node }: { node: { id: string; long_name?: string; short_name?: string; last_seen?: string } }) {
  return (
    <div className="flex items-center justify-between rounded-lg border p-2.5 transition-colors hover:bg-muted/50">
      <div className="flex items-center gap-3 min-w-0">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-secondary">
          <Radio className="h-4 w-4 text-secondary-foreground" />
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">{node.long_name || 'Unknown Node'}</p>
          <p className="truncate text-xs font-mono text-muted-foreground">{node.id}</p>
        </div>
      </div>
      <div className="ml-3 text-xs text-muted-foreground shrink-0">
        {formatRelativeTime(node.last_seen)}
      </div>
    </div>
  )
}

function RecommendationListItem({ recommendation }: { recommendation: { message: string; category?: string; actionable: boolean } }) {
  return (
    <div className="flex items-start gap-3 rounded-lg border p-3">
      <div className="mt-0.5">
        {recommendation.actionable ? (
          <AlertCircle className="h-4 w-4 text-amber-500" />
        ) : (
          <Zap className="h-4 w-4 text-muted-foreground" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm line-clamp-2">{recommendation.message}</p>
        {recommendation.category && (
          <Badge variant="outline" className="mt-2 text-xs">{recommendation.category}</Badge>
        )}
      </div>
    </div>
  )
}
