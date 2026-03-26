import { useEffect, useState } from 'react'
import { PageHeader } from '@/components/ui/PageHeader'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { EmptyState } from '@/components/ui/EmptyState'

interface PlanningBundle {
  evidence_model: string
  graph_hash?: string
  mesh_assessment_id?: string
  transport_connected: boolean
  topology_enabled: boolean
  resilience: {
    resilience_score: number
    redundancy_score: number
    partition_risk_score: number
    fragility_explanation: string[]
    next_best_move_summary: string
    confidence: { level: string; score: number; missing_inputs?: string[] }
  }
  node_profiles: Array<{
    node_num: number
    short_name?: string
    critical_node_score: number
    partition_risk_score: number
    resilience_score: number
    spof_class: string
    recovery_priority: number
  }>
  ranked_next_plans: Array<{
    rank: number
    id: string
    title: string
    verdict: string
    benefit_band: string
    lines: string[]
  }>
  playbooks: Array<{
    class: string
    title: string
    summary: string
    minimum_viable_milestone: string
    steps: Array<{ order: number; title: string; rationale: string; observe_hours: number }>
    limits: string[]
  }>
  limits: string[]
  computed_at: string
}

export function Planning() {
  const [bundle, setBundle] = useState<PlanningBundle | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const res = await fetch('/api/v1/planning/bundle')
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        const data = (await res.json()) as PlanningBundle
        if (!cancelled) {
          setBundle(data)
          setError(null)
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load planning bundle')
          setBundle(null)
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  if (loading) {
    return (
      <div className="p-6">
        <PageHeader title="Deployment planning" description="Loading…" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6 space-y-4">
        <PageHeader title="Deployment planning" description="Bounded what-if and resilience (topology evidence, not RF maps)" />
        <EmptyState title="Planning unavailable" description={error} />
      </div>
    )
  }

  if (!bundle) {
    return null
  }

  return (
    <div className="p-6 space-y-6 max-w-6xl mx-auto">
      <PageHeader
        title="Deployment planning"
        description="Next-step moves, resilience, and playbooks from observed mesh evidence. Not RF coverage simulation."
      />

      <Card className="p-4 border-amber-500/30 bg-amber-500/5">
        <p className="text-sm text-muted-foreground leading-relaxed">{bundle.evidence_model}</p>
        <div className="flex flex-wrap gap-2 mt-3">
          <Badge variant={bundle.transport_connected ? 'default' : 'secondary'}>
            Transport: {bundle.transport_connected ? 'connected' : 'not connected'}
          </Badge>
          <Badge variant={bundle.topology_enabled ? 'default' : 'secondary'}>
            Topology: {bundle.topology_enabled ? 'enabled' : 'disabled'}
          </Badge>
          {bundle.mesh_assessment_id && (
            <Badge variant="outline">Assessment {bundle.mesh_assessment_id}</Badge>
          )}
        </div>
      </Card>

      <div className="grid gap-4 md:grid-cols-2">
        <Card className="p-4">
          <h3 className="font-semibold mb-2">Resilience summary</h3>
          <dl className="grid grid-cols-2 gap-2 text-sm">
            <dt className="text-muted-foreground">Resilience</dt>
            <dd>{bundle.resilience.resilience_score.toFixed(2)}</dd>
            <dt className="text-muted-foreground">Redundancy</dt>
            <dd>{bundle.resilience.redundancy_score.toFixed(2)}</dd>
            <dt className="text-muted-foreground">Partition risk</dt>
            <dd>{bundle.resilience.partition_risk_score.toFixed(2)}</dd>
            <dt className="text-muted-foreground">Confidence</dt>
            <dd>{bundle.resilience.confidence.level}</dd>
          </dl>
          <p className="mt-3 text-sm">{bundle.resilience.next_best_move_summary}</p>
          <ul className="mt-2 list-disc list-inside text-sm text-muted-foreground space-y-1">
            {bundle.resilience.fragility_explanation.map((x, i) => (
              <li key={i}>{x}</li>
            ))}
          </ul>
        </Card>

        <Card className="p-4">
          <h3 className="font-semibold mb-2">Ranked next moves</h3>
          {bundle.ranked_next_plans.length === 0 ? (
            <p className="text-sm text-muted-foreground">No mesh recommendations in this snapshot.</p>
          ) : (
            <ul className="space-y-3">
              {bundle.ranked_next_plans.slice(0, 6).map((r) => (
                <li key={r.id} className="text-sm border-b border-border/50 pb-2 last:border-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-medium">{r.rank}. {r.title}</span>
                    <Badge variant="outline">{r.verdict}</Badge>
                    <Badge variant="secondary">{r.benefit_band}</Badge>
                  </div>
                  {r.lines.slice(0, 2).map((line, i) => (
                    <p key={i} className="text-muted-foreground mt-1">{line}</p>
                  ))}
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>

      <Card className="p-4">
        <h3 className="font-semibold mb-3">Playbooks (field-guide)</h3>
        {bundle.playbooks.length === 0 ? (
          <p className="text-sm text-muted-foreground">No playbooks for this state.</p>
        ) : (
          <div className="space-y-4">
            {bundle.playbooks.map((pb) => (
              <div key={pb.class} className="rounded-lg border border-border p-3">
                <div className="flex items-start justify-between gap-2 flex-wrap">
                  <h4 className="font-medium">{pb.title}</h4>
                  <Badge variant="outline">{pb.class}</Badge>
                </div>
                <p className="text-sm text-muted-foreground mt-1">{pb.summary}</p>
                <p className="text-sm mt-2"><span className="text-muted-foreground">Milestone:</span> {pb.minimum_viable_milestone}</p>
                <ol className="mt-2 space-y-2 list-decimal list-inside text-sm">
                  {pb.steps.map((s) => (
                    <li key={s.order}>
                      <span className="font-medium">{s.title}</span>
                      <span className="text-muted-foreground"> — {s.rationale} (observe ~{s.observe_hours}h)</span>
                    </li>
                  ))}
                </ol>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card className="p-4">
        <h3 className="font-semibold mb-2">Node criticality (observed graph)</h3>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-muted-foreground border-b">
                <th className="py-2 pr-4">#</th>
                <th className="py-2 pr-4">Node</th>
                <th className="py-2 pr-4">Critical</th>
                <th className="py-2 pr-4">Partition risk</th>
                <th className="py-2 pr-4">SPOF</th>
              </tr>
            </thead>
            <tbody>
              {bundle.node_profiles.slice(0, 15).map((n) => (
                <tr key={n.node_num} className="border-b border-border/40">
                  <td className="py-2 pr-4">{n.recovery_priority}</td>
                  <td className="py-2 pr-4">{n.short_name || n.node_num}</td>
                  <td className="py-2 pr-4">{n.critical_node_score.toFixed(2)}</td>
                  <td className="py-2 pr-4">{n.partition_risk_score.toFixed(2)}</td>
                  <td className="py-2 pr-4">{n.spof_class}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <Card className="p-4">
        <h3 className="font-semibold mb-2">Limits</h3>
        <ul className="list-disc list-inside text-sm text-muted-foreground space-y-1">
          {bundle.limits.map((x, i) => (
            <li key={i}>{x}</li>
          ))}
        </ul>
        <p className="text-xs text-muted-foreground mt-3">Computed {bundle.computed_at}</p>
      </Card>
    </div>
  )
}
