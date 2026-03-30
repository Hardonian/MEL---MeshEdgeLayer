import { useCallback, useEffect, useMemo, useState } from 'react'
import { GitBranch, RefreshCw, AlertCircle } from 'lucide-react'

type TopoNode = {
  node_num: number
  node_id: string
  long_name: string
  short_name: string
  health_state: string
  health_score: number
  last_seen_at?: string
  stale: boolean
  lat_redacted?: number
  lon_redacted?: number
  location_state?: string
}

type TopoLink = {
  edge_id: string
  src_node_num: number
  dst_node_num: number
  observed: boolean
  stale: boolean
  quality_score: number
  relay_dependent: boolean
  transport_path?: string
  contradiction?: boolean
  contradiction_detail?: string
}
type TopologySnapshot = {
  snapshot_id: string
  created_at: string
  graph_hash?: string
  node_count: number
  edge_count: number
}

type MeshIntelBootstrap = {
  viability: string
  lone_wolf_score: number
  bootstrap_readiness_score: number
  confidence: string
  explanation?: { top_next_action?: string; weakens_viability?: string[] }
}

type MeshIntelProtocol = {
  fit_class: string
  architecture_class: string
  confidence: string
  primary_limiting_factor?: string
}

type MeshIntelRec = {
  rank: number
  class: string
  title: string
  severity: string
  confidence: number
  evidence_summary?: string[]
  expected_benefit?: string
  downside_risk?: string
}

type MeshIntelligence = {
  assessment_id?: string
  computed_at?: string
  evidence_model?: string
  message_signals?: {
    total_messages?: number
    hop_buckets?: Array<{ key: string; count: number }>
    portnum_buckets?: Array<{ key: string; count: number }>
    rebroadcast_path_proxy?: number
    relay_max_share?: number
    distinct_relay_nodes?: number
  }
  bootstrap?: MeshIntelBootstrap
  topology?: { cluster_shape?: string; fragmentation_score?: number; infrastructure_leverage_score?: number }
  protocol_fit?: MeshIntelProtocol
  routing_pressure?: { summary_lines?: string[] }
  recommendations?: MeshIntelRec[]
}

type Intelligence = {
  generated_at?: string
  topology_enabled?: boolean
  view_mode?: string
  map_eligible_node_count?: number
  transport_connected?: boolean
  evidence_model?: string
  mesh_intelligence?: MeshIntelligence
  analysis?: {
    recommendations?: Array<{ id: string; summary: string; confidence: number; evidence?: string[] }>
    snapshot?: { explanation?: string[]; confidence_summary?: Record<string, number> }
  }
}

type NodeMeshIntel = {
  coverage_contribution_score: number
  relay_value_score: number
  placement_quality_score: number
  is_bridge_critical?: boolean
  notes?: string[]
}

type NodeDrill = {
  node: TopoNode
  scored_state: string
  scored_health: number
  score_factors?: Array<{ name: string; contribution: number; basis: string; evidence?: string }>
  next_actions?: string[]
  evidence_notes?: string[]
  links?: TopoLink[]
  freshness_age_seconds?: number
  mesh_intel?: NodeMeshIntel
}

const API = '/api/v1/topology'

export function Topology() {
  const [intel, setIntel] = useState<Intelligence | null>(null)
  const [nodes, setNodes] = useState<TopoNode[]>([])
  const [links, setLinks] = useState<TopoLink[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState<string | null>(null)
  const [selected, setSelected] = useState<NodeDrill | null>(null)
  const [selLoading, setSelLoading] = useState(false)
  const [snapshots, setSnapshots] = useState<TopologySnapshot[]>([])

  const load = useCallback(async () => {
    setLoading(true)
    setErr(null)
    try {
      const [ri, rn, rl, rs] = await Promise.all([
        fetch(`${API}`),
        fetch(`${API}/nodes?limit=500`),
        fetch(`${API}/links?limit=500`),
        fetch(`${API}/snapshots?limit=6`),
      ])
      if (!ri.ok || !rn.ok || !rl.ok || !rs.ok) {
        throw new Error(`topology API error (${ri.status}/${rn.status}/${rl.status}/${rs.status})`)
      }
      const [ji, jn, jl, js] = await Promise.all([ri.json(), rn.json(), rl.json(), rs.json()])
      setIntel(ji as Intelligence)
      setNodes(jn.nodes || [])
      setLinks(jl.links || [])
      setSnapshots((js.snapshots as TopologySnapshot[]) || [])
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'load failed')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const graphLayout = useMemo(() => {
    const ids = nodes.map((n) => n.node_num).sort((a, b) => a - b)
    const cx = 240
    const cy = 240
    const r = Math.min(200, 40 + ids.length * 6)
    const pos = new Map<number, { x: number; y: number }>()
    const n = Math.max(1, ids.length)
    ids.forEach((id, i) => {
      const ang = (2 * Math.PI * i) / n - Math.PI / 2
      pos.set(id, { x: cx + r * Math.cos(ang), y: cy + r * Math.sin(ang) })
    })
    const edges = links
      .map((l) => [l.src_node_num, l.dst_node_num] as const)
      .filter(([a, b]) => pos.has(a) && pos.has(b))

    for (let step = 0; step < 80; step += 1) {
      for (const [a, b] of edges) {
        const pa = pos.get(a)
        const pb = pos.get(b)
        if (!pa || !pb) continue
        const dx = pb.x - pa.x
        const dy = pb.y - pa.y
        const dist = Math.max(1, Math.hypot(dx, dy))
        const target = 90
        const pull = (dist - target) * 0.012
        pa.x += dx * pull
        pa.y += dy * pull
        pb.x -= dx * pull
        pb.y -= dy * pull
      }
    }

    return { pos, cx, cy, r }
  }, [nodes, links])

  const driftSummary = useMemo(() => {
    if (snapshots.length < 2) return null
    const latest = snapshots[0]
    const previous = snapshots[1]
    return {
      changed: latest.graph_hash !== '' && previous.graph_hash !== '' && latest.graph_hash !== previous.graph_hash,
      nodeDelta: latest.node_count - previous.node_count,
      edgeDelta: latest.edge_count - previous.edge_count,
      latestAt: latest.created_at,
    }
  }, [snapshots])

  const selectNode = async (num: number) => {
    setSelLoading(true)
    try {
      const res = await fetch(`${API}/nodes/${num}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const d = (await res.json()) as NodeDrill
      setSelected(d)
    } catch {
      setSelected(null)
    } finally {
      setSelLoading(false)
    }
  }

  const mapPoints = useMemo(() => {
    return nodes.filter(
      (n) =>
        (n.location_state === 'exact' || n.location_state === 'approximate') &&
        n.lat_redacted != null &&
        n.lon_redacted != null &&
        (n.lat_redacted !== 0 || n.lon_redacted !== 0)
    )
  }, [nodes])

  const intelBody =
    intel && intel.topology_enabled === false ? (
      <p className="text-muted-foreground text-sm">
        {(intel as { message?: string }).message || 'Topology model is disabled in config.'}
      </p>
    ) : null

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-primary/10 text-primary">
            <GitBranch className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">Topology</h1>
            <p className="text-sm text-muted-foreground">
              Graph from stored links (packet relay / destination fields). Not a geographic or RF map unless coordinates are present and map reporting is allowed.
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => load()}
          className="inline-flex items-center gap-2 px-3 py-2 rounded-md border bg-background hover:bg-muted text-sm"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {err && (
        <div className="flex items-center gap-2 text-destructive text-sm">
          <AlertCircle className="h-4 w-4 shrink-0" />
          {err}
        </div>
      )}

      {intelBody}

      {intel && intel.topology_enabled !== false && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <SummaryCard label="View mode" value={intel.view_mode || '—'} hint="graph unless map reporting + coordinates justify map" />
          <SummaryCard label="Nodes" value={String(nodes.length)} hint="from topology store" />
          <SummaryCard label="Links" value={String(links.length)} hint="derived from ingest" />
          <SummaryCard
            label="Transport"
            value={intel.transport_connected ? 'ingest-capable' : 'idle / disconnected'}
            hint="live or idle transport states"
          />
        </div>
      )}

      {intel?.evidence_model && (
        <p className="text-xs text-muted-foreground border-l-2 border-muted pl-3">{intel.evidence_model}</p>
      )}

      {intel?.mesh_intelligence && intel.topology_enabled !== false && (
        <div className="rounded-xl border bg-card p-4 space-y-4">
          <div>
            <h2 className="text-sm font-medium">Mesh deployment intelligence</h2>
            <p className="text-xs text-muted-foreground mt-1">
              Derived assessments from observed packets and graph — not RF proof. Advisory only; MEL does not change routing.
            </p>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <SummaryCard
              label="Bootstrap viability"
              value={(intel.mesh_intelligence.bootstrap?.viability || '—').replace(/_/g, ' ')}
              hint={`lone wolf ${(intel.mesh_intelligence.bootstrap?.lone_wolf_score ?? 0).toFixed(2)} · readiness ${(intel.mesh_intelligence.bootstrap?.bootstrap_readiness_score ?? 0).toFixed(2)}`}
            />
            <SummaryCard
              label="Confidence"
              value={intel.mesh_intelligence.bootstrap?.confidence || '—'}
              hint="raises with more nodes + message history"
            />
            <SummaryCard
              label="Topology shape"
              value={(intel.mesh_intelligence.topology?.cluster_shape || '—').replace(/_/g, ' ')}
              hint={`fragmentation ${(intel.mesh_intelligence.topology?.fragmentation_score ?? 0).toFixed(2)}`}
            />
            <SummaryCard
              label="Protocol fit (managed flood)"
              value={(intel.mesh_intelligence.protocol_fit?.fit_class || '—').replace(/_/g, ' ')}
              hint={(intel.mesh_intelligence.protocol_fit?.architecture_class || '').replace(/_/g, ' ')}
            />
          </div>
          {intel.mesh_intelligence.bootstrap?.explanation?.top_next_action && (
            <div className="text-sm border-l-2 border-primary/40 pl-3">
              <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Highest-leverage next step</span>
              <p className="mt-1">{intel.mesh_intelligence.bootstrap.explanation.top_next_action}</p>
            </div>
          )}
          {intel.mesh_intelligence.message_signals?.total_messages != null &&
            intel.mesh_intelligence.message_signals.total_messages > 0 && (
              <div className="text-xs text-muted-foreground">
                Message rollup: {intel.mesh_intelligence.message_signals.total_messages} in window
                {intel.mesh_intelligence.message_signals.rebroadcast_path_proxy != null && (
                  <> · rebroadcast path proxy {intel.mesh_intelligence.message_signals.rebroadcast_path_proxy.toFixed(2)}</>
                )}
              </div>
            )}
          {intel.mesh_intelligence.routing_pressure?.summary_lines &&
            intel.mesh_intelligence.routing_pressure.summary_lines.length > 0 && (
              <div>
                <div className="text-xs font-medium text-muted-foreground mb-1">Routing / flood pressure (suspected)</div>
                <ul className="text-xs text-muted-foreground list-disc pl-4 space-y-1">
                  {intel.mesh_intelligence.routing_pressure.summary_lines.map((line, i) => (
                    <li key={i}>{line}</li>
                  ))}
                </ul>
              </div>
            )}
          {intel.mesh_intelligence.recommendations && intel.mesh_intelligence.recommendations.length > 0 && (
            <div>
              <div className="text-xs font-medium text-muted-foreground mb-2">Ranked recommendations</div>
              <ul className="text-sm space-y-2">
                {intel.mesh_intelligence.recommendations.slice(0, 8).map((r) => (
                  <li key={r.rank} className="border-b border-border/50 pb-2 last:border-0">
                    <div>
                      <span className="text-xs text-muted-foreground mr-2">#{r.rank}</span>
                      {r.title}
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5">
                      {r.class.replace(/_/g, ' ')} · sev {r.severity} · conf {r.confidence.toFixed(2)}
                    </div>
                    {r.evidence_summary && r.evidence_summary.length > 0 && (
                      <div className="text-xs mt-1 font-mono text-muted-foreground">{r.evidence_summary.join(' · ')}</div>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          )}
          {intel.mesh_intelligence.evidence_model && (
            <p className="text-[11px] text-muted-foreground border-t pt-3">{intel.mesh_intelligence.evidence_model}</p>
          )}
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-4">
          <div className="rounded-xl border bg-card p-4">
            <h2 className="text-sm font-medium mb-2">Topology graph</h2>
            <p className="text-xs text-muted-foreground mb-4">
              Relaxed graph layout from stored links. Inferred edges are dashed; stale or contradicted links are visually de-emphasized. Click a node for drilldown.
            </p>
            {driftSummary && (
              <div className="mb-3 flex flex-wrap gap-2 text-xs">
                <span className="rounded border px-2 py-1">
                  drift {driftSummary.changed ? 'detected' : 'not-detected'} since {new Date(driftSummary.latestAt).toLocaleTimeString()}
                </span>
                <span className="rounded border px-2 py-1">Δnodes {driftSummary.nodeDelta >= 0 ? '+' : ''}{driftSummary.nodeDelta}</span>
                <span className="rounded border px-2 py-1">Δlinks {driftSummary.edgeDelta >= 0 ? '+' : ''}{driftSummary.edgeDelta}</span>
              </div>
            )}
            {nodes.length === 0 ? (
              <p className="text-sm text-muted-foreground">No nodes in topology store yet.</p>
            ) : (
              <svg viewBox="0 0 480 480" className="w-full max-h-[480px] text-foreground">
                {links.map((l) => {
                  const a = graphLayout.pos.get(l.src_node_num)
                  const b = graphLayout.pos.get(l.dst_node_num)
                  if (!a || !b) return null
                  const weak = l.stale || l.quality_score < 0.35
                  return (
                    <line
                      key={l.edge_id}
                      x1={a.x}
                      y1={a.y}
                      x2={b.x}
                      y2={b.y}
                      stroke="currentColor"
                      strokeOpacity={weak ? 0.22 : 0.55}
                      strokeWidth={l.relay_dependent ? 1 : 1.5}
                      strokeDasharray={l.observed ? undefined : '4 3'}
                      className={l.contradiction ? 'text-warning' : undefined}
                    />
                  )
                })}
                {nodes.map((n) => {
                  const p = graphLayout.pos.get(n.node_num)
                  if (!p) return null
                  const fill =
                    n.health_state === 'healthy'
                      ? 'hsl(142 70% 40%)'
                      : n.stale || n.health_state === 'stale'
                        ? 'hsl(38 90% 45%)'
                        : n.health_state === 'isolated'
                          ? 'hsl(280 50% 50%)'
                          : 'hsl(210 70% 45%)'
                  return (
                    <g key={n.node_num} className="cursor-pointer" onClick={() => selectNode(n.node_num)}>
                      <circle cx={p.x} cy={p.y} r={10} fill={fill} stroke="currentColor" strokeWidth={1} />
                      <text x={p.x + 14} y={p.y + 4} fontSize="10" fill="currentColor" className="select-none">
                        {n.short_name || n.node_num}
                      </text>
                    </g>
                  )
                })}
              </svg>
            )}
          </div>

          {(intel?.view_mode === 'map' || intel?.view_mode === 'map_partial') && mapPoints.length > 0 && (
            <div className="rounded-xl border bg-card p-4">
              <h2 className="text-sm font-medium mb-2">Coordinate scatter (redacted)</h2>
              <p className="text-xs text-muted-foreground mb-4">
                Normalized plot of lat_redacted/lon_redacted — not a surveyed map. Stale or unknown locations are excluded.
              </p>
              <MapScatter nodes={mapPoints} />
            </div>
          )}
        </div>

        <div className="rounded-xl border bg-card p-4 min-h-[200px]">
          <h2 className="text-sm font-medium mb-2">Drilldown</h2>
          {selLoading && <p className="text-sm text-muted-foreground">Loading…</p>}
          {!selLoading && !selected && <p className="text-sm text-muted-foreground">Select a node on the graph.</p>}
          {selected && (
            <div className="space-y-3 text-sm">
              <div>
                <div className="font-medium">
                  {selected.node.long_name || selected.node.short_name || selected.node.node_num}
                </div>
                <div className="text-xs text-muted-foreground font-mono">
                  #{selected.node.node_num} {selected.node.node_id}
                </div>
              </div>
              <div>
                <span className="text-muted-foreground">Scored state:</span> {selected.scored_state}{' '}
                <span className="text-muted-foreground">({selected.scored_health.toFixed(2)})</span>
              </div>
              {selected.freshness_age_seconds != null && selected.freshness_age_seconds >= 0 && (
                <div className="text-xs text-muted-foreground">
                  Last seen ≈ {Math.round(selected.freshness_age_seconds)}s ago (server clock)
                </div>
              )}
              {selected.score_factors && selected.score_factors.length > 0 && (
                <div>
                  <div className="text-xs font-medium text-muted-foreground mb-1">Score factors</div>
                  <ul className="text-xs space-y-1 max-h-40 overflow-y-auto">
                    {selected.score_factors.map((f) => (
                      <li key={f.name}>
                        <span className="font-mono">{f.name}</span>: {f.contribution.toFixed(3)} ({f.basis})
                        {f.evidence && <span className="block text-muted-foreground truncate">{f.evidence}</span>}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              {selected.next_actions && selected.next_actions.length > 0 && (
                <div>
                  <div className="text-xs font-medium text-muted-foreground mb-1">Next actions</div>
                  <ul className="text-xs list-disc pl-4 space-y-1">
                    {selected.next_actions.map((a, i) => (
                      <li key={i}>{a}</li>
                    ))}
                  </ul>
                </div>
              )}
              {selected.evidence_notes && (
                <ul className="text-xs text-muted-foreground list-disc pl-4">
                  {selected.evidence_notes.map((n, i) => (
                    <li key={i}>{n}</li>
                  ))}
                </ul>
              )}
              {selected.mesh_intel && (
                <div className="border-t pt-3 mt-2">
                  <div className="text-xs font-medium text-muted-foreground mb-2">Deployment intelligence (this node)</div>
                  <div className="text-xs space-y-1">
                    <div>
                      Coverage contribution: {selected.mesh_intel.coverage_contribution_score.toFixed(2)} · relay value:{' '}
                      {selected.mesh_intel.relay_value_score.toFixed(2)} · placement proxy:{' '}
                      {selected.mesh_intel.placement_quality_score.toFixed(2)}
                    </div>
                    {selected.mesh_intel.is_bridge_critical && (
                      <div className="text-warning">Bridge-critical in observed graph</div>
                    )}
                    {selected.mesh_intel.notes && selected.mesh_intel.notes.length > 0 && (
                      <ul className="list-disc pl-4 text-muted-foreground">
                        {selected.mesh_intel.notes.map((n, i) => (
                          <li key={i}>{n.replace(/_/g, ' ')}</li>
                        ))}
                      </ul>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {intel?.analysis?.recommendations && intel.analysis.recommendations.length > 0 && (
        <div className="rounded-xl border bg-card p-4">
          <h2 className="text-sm font-medium mb-2">Evidence-based recommendations</h2>
          <ul className="text-sm space-y-2">
            {intel.analysis.recommendations.slice(0, 12).map((r) => (
              <li key={r.id} className="border-b border-border/50 pb-2 last:border-0">
                <div>{r.summary}</div>
                <div className="text-xs text-muted-foreground">confidence {r.confidence.toFixed(2)}</div>
                {r.evidence && r.evidence.length > 0 && (
                  <div className="text-xs mt-1 font-mono text-muted-foreground">{r.evidence.join(' · ')}</div>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}

function SummaryCard({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <div className="rounded-lg border bg-card p-3">
      <div className="text-xs text-muted-foreground uppercase tracking-wide">{label}</div>
      <div className="text-lg font-semibold mt-1">{value}</div>
      <div className="text-[11px] text-muted-foreground mt-1 leading-snug">{hint}</div>
    </div>
  )
}

function MapScatter({ nodes }: { nodes: TopoNode[] }) {
  const lats = nodes.map((n) => n.lat_redacted ?? 0)
  const lons = nodes.map((n) => n.lon_redacted ?? 0)
  const minLat = Math.min(...lats)
  const maxLat = Math.max(...lats)
  const minLon = Math.min(...lons)
  const maxLon = Math.max(...lons)
  const pad = 24
  const w = 400
  const h = 280
  const xLon = (lon: number) => {
    const t = maxLon === minLon ? 0.5 : (lon - minLon) / (maxLon - minLon)
    return pad + t * (w - 2 * pad)
  }
  const yLat = (lat: number) => {
    const t = maxLat === minLat ? 0.5 : (lat - minLat) / (maxLat - minLat)
    return pad + (1 - t) * (h - 2 * pad)
  }
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="w-full max-h-[300px]">
      <rect width={w} height={h} fill="none" stroke="currentColor" strokeOpacity={0.15} />
      {nodes.map((n) => (
        <circle
          key={n.node_num}
          cx={xLon(n.lon_redacted ?? 0)}
          cy={yLat(n.lat_redacted ?? 0)}
          r={6}
          fill={n.stale ? 'hsl(38 90% 45%)' : 'hsl(142 70% 40%)'}
          stroke="currentColor"
          strokeWidth={1}
        />
      ))}
    </svg>
  )
}
