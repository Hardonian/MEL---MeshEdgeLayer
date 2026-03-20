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
} from 'lucide-react'
import { clsx } from 'clsx'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card'
import { Badge, HealthBadge, SeverityBadge } from '@/components/ui/Badge'
import { Loading, ErrorView, EmptyState } from '@/components/ui/StateViews'
import { useStatus, useNodes, useMessages, usePrivacyFindings, useRecommendations, useDeadLetters } from '@/hooks/useApi'
import { getHealthState, formatRelativeTime } from '@/types/api'

export function Dashboard() {
  const status = useStatus()
  const nodes = useNodes()
  const messages = useMessages()
  const privacy = usePrivacyFindings()
  const recommendations = useRecommendations()
  const deadLetters = useDeadLetters()

  const isLoading = status.loading || nodes.loading || messages.loading
  const hasError = status.error || nodes.error || messages.error

  if (isLoading && !status.data) {
    return <Loading message="Loading system status..." />
  }

  if (hasError && !status.data) {
    return <ErrorView message={status.error || 'Failed to connect to MEL backend'} />
  }

  const transports = status.data?.transports || []
  const connectedTransport = transports.find(t => t.effective_state === 'connected')
  const healthyTransports = transports.filter(t => getHealthState(t.health) === 'healthy').length
  const totalTransports = transports.length

  const activePrivacyFindings = privacy.data?.filter(p => p.severity === 'critical' || p.severity === 'high') || []
  const pendingRecommendations = recommendations.data?.filter(r => r.actionable) || []

  return (
    <div className="space-y-8">
      {/* Page Header */}
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          Real-time overview of your MeshEdgeLayer mesh observability system.
        </p>
      </div>

      {/* Quick Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 stagger-children">
        {/* Connection Status */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Connection</CardTitle>
            {connectedTransport ? (
              <CheckCircle2 className="h-4 w-4 text-success" />
            ) : (
              <XCircle className="h-4 w-4 text-critical" />
            )}
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {connectedTransport ? 'Connected' : 'Disconnected'}
            </div>
            <p className="text-xs text-muted-foreground">
              {connectedTransport ? connectedTransport.name : 'No active transport'}
            </p>
          </CardContent>
        </Card>

        {/* Nodes */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Nodes</CardTitle>
            <Radio className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{nodes.data?.length || 0}</div>
            <p className="text-xs text-muted-foreground">
              {nodes.data?.length === 0 ? 'No nodes observed yet' : 'Mesh devices detected'}
            </p>
          </CardContent>
        </Card>

        {/* Messages */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Messages</CardTitle>
            <MessageSquare className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{status.data?.messages || 0}</div>
            <p className="text-xs text-muted-foreground">Runtime message count</p>
          </CardContent>
        </Card>

        {/* Transport Health */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Transports</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {healthyTransports}/{totalTransports}
            </div>
            <p className="text-xs text-muted-foreground">Healthy transport connections</p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Transport Status */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Transport Status</CardTitle>
              <Link
                to="/status"
                className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
              >
                View all <ArrowRight className="h-4 w-4" />
              </Link>
            </div>
            <CardDescription>Health status of configured transports</CardDescription>
          </CardHeader>
          <CardContent>
            {transports.length === 0 ? (
              <EmptyState
                title="No transports configured"
                description="Configure a transport in your MEL config to start receiving mesh data."
              />
            ) : (
              <div className="space-y-3">
                {transports.slice(0, 4).map((transport) => (
                  <div
                    key={transport.name}
                    className="flex items-center justify-between rounded-lg border p-3"
                  >
                    <div className="flex items-center gap-3">
                      <div className={clsx(
                        'flex h-8 w-8 items-center justify-center rounded-full',
                        getHealthState(transport.health) === 'healthy' ? 'bg-success/10' :
                        getHealthState(transport.health) === 'degraded' ? 'bg-warning/10' :
                        'bg-critical/10'
                      )}>
                        <Activity className={clsx(
                          'h-4 w-4',
                          getHealthState(transport.health) === 'healthy' ? 'text-success' :
                          getHealthState(transport.health) === 'degraded' ? 'text-warning' :
                          'text-critical'
                        )} />
                      </div>
                      <div>
                        <p className="font-medium">{transport.name}</p>
                        <p className="text-xs text-muted-foreground">{transport.type}</p>
                      </div>
                    </div>
                    <div className="text-right">
                      <HealthBadge health={getHealthState(transport.health)} />
                      <p className="mt-1 text-xs text-muted-foreground">
                        {transport.last_heartbeat_at ? formatRelativeTime(transport.last_heartbeat_at) : 'Never'}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Privacy Findings */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Privacy Findings</CardTitle>
              <Link
                to="/privacy"
                className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
              >
                View all <ArrowRight className="h-4 w-4" />
              </Link>
            </div>
            <CardDescription>Security and privacy issues detected</CardDescription>
          </CardHeader>
          <CardContent>
            {privacy.loading ? (
              <Loading message="Scanning..." />
            ) : activePrivacyFindings.length === 0 ? (
              <div className="flex items-center gap-2 text-success">
                <CheckCircle2 className="h-5 w-5" />
                <span>No critical privacy issues detected</span>
              </div>
            ) : (
              <div className="space-y-3">
                {activePrivacyFindings.slice(0, 4).map((finding, i) => (
                  <div key={i} className="flex items-start gap-3 rounded-lg border border-critical/20 bg-critical/5 p-3">
                    <Shield className="h-5 w-5 text-critical mt-0.5" />
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <SeverityBadge severity={finding.severity} />
                      </div>
                      <p className="mt-1 text-sm">{finding.message}</p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent Nodes */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Recent Nodes</CardTitle>
              <Link
                to="/nodes"
                className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
              >
                View all <ArrowRight className="h-4 w-4" />
              </Link>
            </div>
            <CardDescription>Recently observed mesh nodes</CardDescription>
          </CardHeader>
          <CardContent>
            {nodes.loading ? (
              <Loading message="Loading nodes..." />
            ) : nodes.data?.length === 0 ? (
              <EmptyState
                title="No nodes yet"
                description="Nodes will appear here once mesh traffic is observed."
              />
            ) : (
              <div className="space-y-3">
                {nodes.data?.slice(0, 5).map((node) => (
                  <div key={node.id} className="flex items-center justify-between rounded-lg border p-3">
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-full bg-secondary">
                        <Radio className="h-4 w-4 text-secondary-foreground" />
                      </div>
                      <div>
                        <p className="font-medium">{node.long_name || node.id}</p>
                        <p className="text-xs text-muted-foreground font-mono">{node.id}</p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-xs text-muted-foreground">
                        {formatRelativeTime(node.last_seen)}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recommendations */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Recommendations</CardTitle>
              <Link
                to="/recommendations"
                className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
              >
                View all <ArrowRight className="h-4 w-4" />
              </Link>
            </div>
            <CardDescription>Suggested actions to improve your setup</CardDescription>
          </CardHeader>
          <CardContent>
            {recommendations.loading ? (
              <Loading message="Analyzing..." />
            ) : pendingRecommendations.length === 0 ? (
              <div className="flex items-center gap-2 text-success">
                <CheckCircle2 className="h-5 w-5" />
                <span>No pending recommendations</span>
              </div>
            ) : (
              <div className="space-y-3">
                {pendingRecommendations.slice(0, 4).map((rec, i) => (
                  <div key={i} className="flex items-start gap-3 rounded-lg border p-3">
                    <AlertTriangle className="h-5 w-5 text-warning mt-0.5" />
                    <div>
                      <p className="font-medium">{rec.message}</p>
                      {rec.category && (
                        <Badge variant="outline" className="mt-2">{rec.category}</Badge>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Dead Letters Alert */}
      {deadLetters.data && deadLetters.data.length > 0 && (
        <Card className="border-warning/50">
          <CardHeader>
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-warning" />
              <CardTitle>Recent Dead Letters</CardTitle>
            </div>
            <CardDescription>Messages that failed to be processed</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {deadLetters.data.slice(0, 3).map((dl, i) => (
                <div key={i} className="flex items-center justify-between rounded-lg border border-warning/20 bg-warning/5 p-3">
                  <div>
                    <p className="font-medium">{dl.transport_name}</p>
                    <p className="text-sm text-muted-foreground">{dl.reason}</p>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {formatRelativeTime(dl.created_at)}
                  </p>
                </div>
              ))}
            </div>
            <Link
              to="/dead-letters"
              className="mt-4 flex items-center gap-1 text-sm text-warning hover:underline"
            >
              View all dead letters <ArrowRight className="h-4 w-4" />
            </Link>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
