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
import { PageHeader } from '@/components/ui/PageHeader'
import { StaleDataBanner } from '@/components/states/StaleDataBanner'
import { EmptyState, SystemHealthy, NoTransportsConfigured } from '@/components/ui/EmptyState'
import { useStatus, useNodes, useMessages, usePrivacyFindings, useRecommendations, useDeadLetters } from '@/hooks/useApi'
import { getHealthState, formatRelativeTime, TransportHealth, NodeInfo } from '@/types/api'

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
              className="button-danger"
            >
              Retry connection
            </button>
          }
        />
      </div>
    )
  }

  const transports = status.data?.transports || []
  const connectedTransport = transports.find((t) => t.effective_state === 'connected')
  const healthyTransports = transports.filter((t) => getHealthState(t.health) === 'healthy').length
  const totalTransports = transports.length
  const hasTransports = totalTransports > 0

  const activePrivacyFindings = privacy.data?.filter((p) => p.severity === 'critical' || p.severity === 'high') || []
  const pendingRecommendations = recommendations.data?.filter((r) => r.actionable) || []

  const newestHeartbeat = transports.reduce((max, t) => {
    if (!t.last_heartbeat_at) return max
    const ts = new Date(t.last_heartbeat_at).getTime()
    return ts > max ? ts : max
  }, 0)
  const dashboardStaleTs = newestHeartbeat ? new Date(newestHeartbeat).toISOString() : undefined

  return (
    <div className="space-y-6 pb-10 md:space-y-8">
      <PageHeader
        title="Dashboard"
        description="Operational snapshot from the last API refresh. Values reflect what MEL has observed, not guaranteed live mesh state. The console re-polls every 30 seconds."
        action={<Badge variant="outline" className="uppercase tracking-[0.18em]">Auto refresh 30s</Badge>}
      />

      <StaleDataBanner lastSuccessfulIngest={dashboardStaleTs} componentName="Dashboard / Transports" />

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4 stagger-children">
        <StatCard
          title="Connection Status"
          value={connectedTransport ? 'Connected' : 'Disconnected'}
          description={connectedTransport ? connectedTransport.name : 'No active transport'}
          icon={connectedTransport ? <CheckCircle2 className="h-5 w-5" /> : <XCircle className="h-5 w-5" />}
          variant={connectedTransport ? 'success' : 'warning'}
        />
        <StatCard
          title="Nodes"
          value={nodes.data?.length || 0}
          description={nodes.data?.length === 0 ? 'Awaiting mesh observations' : 'Mesh devices detected'}
          icon={<Radio className="h-5 w-5" />}
          variant="default"
        />
        <StatCard
          title="Messages"
          value={messagesData?.length || status.data?.messages || 0}
          description="Runtime message count"
          icon={<MessageSquare className="h-5 w-5" />}
          variant="info"
        />
        <StatCard
          title="Transport Health"
          value={`${healthyTransports}/${totalTransports}`}
          description={hasTransports ? 'Healthy transport connections' : 'No transports configured'}
          icon={<TrendingUp className="h-5 w-5" />}
          variant={healthyTransports === totalTransports && hasTransports ? 'success' : healthyTransports > 0 ? 'warning' : 'default'}
        />
      </div>

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
              className="inline-flex items-center gap-1 text-sm font-semibold text-foreground transition-colors hover:text-primary"
            >
              View all <ArrowRight className="h-4 w-4" />
            </Link>
          }
        />
      )}

      <div className="grid gap-5 xl:grid-cols-2">
        <Card className="overflow-hidden">
          <CardHeader className="border-b border-border/50 pb-4">
            <SectionCardHeader
              icon={<Activity className="h-4 w-4" />}
              iconClassName="border-primary/16 bg-primary/12 text-primary"
              title="Transport Status"
              description="Health status of configured transports"
              href="/status"
            />
          </CardHeader>
          <CardContent className="pt-5">
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

        <Card className="overflow-hidden">
          <CardHeader className="border-b border-border/50 pb-4">
            <SectionCardHeader
              icon={<Shield className="h-4 w-4" />}
              iconClassName="border-success/16 bg-success/12 text-success"
              title="Privacy Posture"
              description="Security and privacy assessment"
              href="/privacy"
            />
          </CardHeader>
          <CardContent className="pt-5">
            {privacy.loading ? (
              <Loading message="Scanning..." />
            ) : activePrivacyFindings.length === 0 ? (
              <SystemHealthy message="No critical privacy issues detected" />
            ) : (
              <div className="space-y-3">
                {activePrivacyFindings.slice(0, 3).map((finding, i) => (
                  <InlineAlert key={i} variant={finding.severity === 'critical' ? 'critical' : 'warning'}>
                    <div className="flex items-center justify-between gap-2">
                      <span className="truncate text-sm">{finding.message}</span>
                      <SeverityBadge severity={finding.severity} />
                    </div>
                  </InlineAlert>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="overflow-hidden">
          <CardHeader className="border-b border-border/50 pb-4">
            <SectionCardHeader
              icon={<Radio className="h-4 w-4" />}
              iconClassName="border-border/70 bg-secondary text-secondary-foreground"
              title="Recent Nodes"
              description="Recently observed mesh devices"
              href="/nodes"
            />
          </CardHeader>
          <CardContent className="pt-5">
            {nodes.loading ? (
              <Loading message="Loading nodes..." />
            ) : nodes.data?.length === 0 ? (
              <EmptyState
                type="no-data"
                title="No nodes yet"
                description="Nodes will appear here once mesh traffic is observed."
              />
            ) : (
              <div className="space-y-2.5">
                {nodes.data?.slice(0, 5).map((node) => (
                  <NodeListItem key={node.node_id} node={node} />
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="overflow-hidden">
          <CardHeader className="border-b border-border/50 pb-4">
            <SectionCardHeader
              icon={<Zap className="h-4 w-4" />}
              iconClassName="border-warning/16 bg-warning/12 text-warning"
              title="Recommendations"
              description="Actions to improve your setup"
              href="/recommendations"
            />
          </CardHeader>
          <CardContent className="pt-5">
            {recommendations.loading ? (
              <Loading message="Analyzing..." />
            ) : pendingRecommendations.length === 0 ? (
              <SystemHealthy message="No pending recommendations" />
            ) : (
              <div className="space-y-2.5">
                {pendingRecommendations.slice(0, 4).map((rec, i) => (
                  <RecommendationListItem key={i} recommendation={rec} />
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {deadLetters.data && deadLetters.data.length > 0 && (
        <Card className="overflow-hidden border-warning/20">
          <CardHeader className="border-b border-border/50 pb-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-2xl border border-warning/18 bg-warning/12 text-warning shadow-inset">
                <AlertTriangle className="h-5 w-5" />
              </div>
              <div>
                <CardTitle className="text-base">Recent Dead Letters</CardTitle>
                <CardDescription className="mt-1 text-xs uppercase tracking-[0.16em] text-muted-foreground">
                  Messages that failed to be processed
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="pt-5">
            <div className="space-y-2.5">
              {deadLetters.data.slice(0, 3).map((dl, i) => (
                <div key={i} className="list-row px-4 py-3">
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-semibold text-foreground">{dl.transport_name}</p>
                    <p className="truncate text-xs leading-relaxed text-muted-foreground">{dl.reason}</p>
                  </div>
                  <div className="ml-4 shrink-0 text-xs uppercase tracking-[0.16em] text-muted-foreground">
                    {formatRelativeTime(dl.created_at)}
                  </div>
                </div>
              ))}
            </div>
            <Link
              to="/dead-letters"
              className="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-warning transition-colors hover:text-foreground"
            >
              View all dead letters <ArrowRight className="h-4 w-4" />
            </Link>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function SectionCardHeader({
  icon,
  iconClassName,
  title,
  description,
  href,
}: {
  icon: React.ReactNode
  iconClassName: string
  title: string
  description: string
  href: string
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex items-center gap-3">
        <div className={clsx('flex h-10 w-10 items-center justify-center rounded-2xl border shadow-inset', iconClassName)}>
          {icon}
        </div>
        <div>
          <CardTitle className="text-base">{title}</CardTitle>
          <CardDescription className="mt-1 text-xs uppercase tracking-[0.16em] text-muted-foreground">
            {description}
          </CardDescription>
        </div>
      </div>
      <Link
        to={href}
        className="inline-flex items-center gap-1 text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground transition-colors hover:text-foreground"
      >
        View all <ArrowRight className="h-3.5 w-3.5" />
      </Link>
    </div>
  )
}

function TransportListItem({ transport }: { transport: TransportHealth }) {
  const healthState = getHealthState(transport.health)

  return (
    <div className="list-row justify-between gap-3 px-4 py-3">
      <div className="flex min-w-0 items-center gap-3">
        <div
          className={clsx(
            'flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border shadow-inset',
            healthState === 'healthy'
              ? 'border-success/18 bg-success/12 text-success'
              : healthState === 'degraded'
                ? 'border-warning/18 bg-warning/12 text-warning'
                : 'border-critical/18 bg-critical/12 text-critical'
          )}
        >
          <Activity className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold text-foreground">{transport.name}</p>
          <p className="truncate text-xs uppercase tracking-[0.16em] text-muted-foreground">{transport.type}</p>
        </div>
      </div>
      <div className="ml-3 flex flex-col items-end gap-1.5">
        <HealthBadge health={healthState} />
        <div className="flex items-center gap-1 text-xs uppercase tracking-[0.14em] text-muted-foreground">
          <Clock className="h-3 w-3" />
          {transport.last_heartbeat_at ? formatRelativeTime(transport.last_heartbeat_at) : 'Never'}
        </div>
      </div>
    </div>
  )
}

function NodeListItem({ node }: { node: NodeInfo }) {
  return (
    <div className="list-row justify-between gap-3 px-4 py-3">
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-border/70 bg-secondary text-secondary-foreground shadow-inset">
          <Radio className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold text-foreground">{node.long_name || 'Unknown Node'}</p>
          <p className="truncate font-mono text-xs text-muted-foreground">{node.node_id}</p>
        </div>
      </div>
      <div className="ml-3 shrink-0 text-xs uppercase tracking-[0.16em] text-muted-foreground">
        {formatRelativeTime(node.last_seen)}
      </div>
    </div>
  )
}

function RecommendationListItem({
  recommendation,
}: {
  recommendation: { message: string; category?: string; actionable: boolean }
}) {
  return (
    <div className="list-row items-start gap-3 px-4 py-3">
      <div
        className={clsx(
          'mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl border shadow-inset',
          recommendation.actionable
            ? 'border-warning/18 bg-warning/12 text-warning'
            : 'border-border/70 bg-card/75 text-muted-foreground'
        )}
      >
        {recommendation.actionable ? <AlertCircle className="h-4 w-4" /> : <Zap className="h-4 w-4" />}
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm leading-relaxed text-foreground">{recommendation.message}</p>
        {recommendation.category && (
          <Badge variant="outline" className="mt-2">
            {recommendation.category}
          </Badge>
        )}
      </div>
    </div>
  )
}
