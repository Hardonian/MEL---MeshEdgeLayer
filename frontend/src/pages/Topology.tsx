import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { GitBranch, RefreshCw, AlertCircle, ZoomIn, ZoomOut, Maximize2, X, ExternalLink } from 'lucide-react'
import type { Incident } from '@/types/api'
import { readShiftSnapshot } from '@/utils/shiftSnapshot'
import { topologyLinkTruthSummary, topologyNodeTruthSummary } from '@/utils/evidenceSemantics'

type TopoNode = {
  node_num: number
  node_id: string
  long_name: string
  short_name: string
  health_state: string
  health_score: number
  first_seen_at?: string
  last_seen_at?: string
  stale: boolean
  lat_redacted?: number
  lon_redacted?: number
  location_state?: string
  trust_class?: string
  source_connector?: string
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

type TopologyCluster = {
  id: string
  node_nums: number[]
  state: string
  avg_score: number
  min_score: number
  edge_count: number
}

type TopologyStaleRegion = {
  node_nums: number[]
  stale_ratio: number
  last_fresh_at?: string
}

type TopologyAnalysisBlock = {
  weak_clusters?: TopologyCluster[]
  stale_regions?: TopologyStaleRegion[]
  isolated_nodes?: number[]
  bridge_nodes?: number[]
  bottlenecks?: Array<{ type: string; node_nums?: number[]; severity: string; explanation: string }>
  recommendations?: Array<{ id: string; summary: string; confidence: number; evidence?: string[] }>
}

type Intelligence = {
  generated_at?: string
  topology_enabled?: boolean
  view_mode?: string
  map_eligible_node_count?: number
  transport_connected?: boolean
  evidence_model?: string
  privacy_map_reporting_allowed?: boolean
  google_maps_basemap_available?: boolean
  google_maps_api_key?: string
  mesh_intelligence?: MeshIntelligence
  analysis?: TopologyAnalysisBlock & {
    snapshot?: { explanation?: string[]; confidence_summary?: Record<string, number> }
  }
  staleness?: { node_stale_minutes?: number; link_stale_minutes?: number }
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
  observations?: unknown[]
  freshness_age_seconds?: number
  mesh_intel?: NodeMeshIntel
}

// ─── Force-directed simulation ─────────────────────────────────────────────────

interface SimNode {
  id: number
  x: number
  y: number
  vx: number
  vy: number
}

function runSimulation(
  nodeIds: number[],
  edges: Array<{ src: number; dst: number }>,
  width: number,
  height: number,
  iterations = 180,
): Map<number, { x: number; y: number }> {
  const n = nodeIds.length
  if (n === 0) return new Map()
  if (n === 1) return new Map([[nodeIds[0], { x: width / 2, y: height / 2 }]])

  const indexMap = new Map(nodeIds.map((id, i) => [id, i]))
  const nodes: SimNode[] = nodeIds.map((id, i) => {
    const angle = (2 * Math.PI * i) / n - Math.PI / 2
    const r = Math.min(width, height) * 0.35
    return { id, x: width / 2 + r * Math.cos(angle), y: height / 2 + r * Math.sin(angle), vx: 0, vy: 0 }
  })

  const repulsion = Math.min(width, height) * (n < 8 ? 80 : n < 20 ? 55 : 35)
  const springLen = Math.min(width, height) * (n < 8 ? 0.35 : 0.25)
  const springK = 0.04
  const centerStrength = 0.015
  const damping = 0.82

  for (let iter = 0; iter < iterations; iter++) {
    for (let i = 0; i < n; i++) {
      for (let j = i + 1; j < n; j++) {
        const dx = nodes[i].x - nodes[j].x
        const dy = nodes[i].y - nodes[j].y
        const distSq = Math.max(1, dx * dx + dy * dy)
        const dist = Math.sqrt(distSq)
        const force = repulsion / distSq
        const fx = (dx / dist) * force
        const fy = (dy / dist) * force
        nodes[i].vx += fx
        nodes[i].vy += fy
        nodes[j].vx -= fx
        nodes[j].vy -= fy
      }
    }

    for (const edge of edges) {
      const ai = indexMap.get(edge.src)
      const bi = indexMap.get(edge.dst)
      if (ai == null || bi == null) continue
      const a = nodes[ai]
      const b = nodes[bi]
      const dx = b.x - a.x
      const dy = b.y - a.y
      const dist = Math.max(1, Math.sqrt(dx * dx + dy * dy))
      const stretch = dist - springLen
      const fx = (dx / dist) * springK * stretch
      const fy = (dy / dist) * springK * stretch
      a.vx += fx
      a.vy += fy
      b.vx -= fx
      b.vy -= fy
    }

    for (const node of nodes) {
      node.vx += (width / 2 - node.x) * centerStrength
      node.vy += (height / 2 - node.y) * centerStrength
    }

    const pad = 28
    for (const node of nodes) {
      node.vx *= damping
      node.vy *= damping
      node.x = Math.max(pad, Math.min(width - pad, node.x + node.vx))
      node.y = Math.max(pad, Math.min(height - pad, node.y + node.vy))
    }
  }

  const result = new Map<number, { x: number; y: number }>()
  for (const node of nodes) result.set(node.id, { x: node.x, y: node.y })
  return result
}

function nodeColor(n: TopoNode): string {
  if (n.stale || n.health_state === 'stale') return 'hsl(38 90% 45%)'
  if (n.health_state === 'healthy') return 'hsl(142 65% 38%)'
  if (n.health_state === 'isolated') return 'hsl(280 50% 52%)'
  if (n.health_state === 'degraded') return 'hsl(28 90% 48%)'
  if (n.health_state === 'inferred_only' || n.health_state === 'weakly_observed') return 'hsl(200 55% 42%)'
  return 'hsl(210 65% 46%)'
}

function nodeLabel(n: TopoNode): string {
  return n.short_name || String(n.node_num)
}

const API = '/api/v1/topology'

async function fetchTopologyJson<T>(path: string, signal: AbortSignal, label: string): Promise<T> {
  const res = await fetch(path, { signal })
  if (!res.ok) {
    throw new Error(`${label}: HTTP ${res.status}`)
  }
  return (await res.json()) as T
}

type NodeFilterId = 'all' | 'stale' | 'degraded' | 'risky' | 'no_map_coords' | 'changed_since_visit' | 'incident_focus' | 'transport_scoped'

function withReturnParam(targetPath: string, returnPath: string): string {
  if (!returnPath.startsWith('/')) return targetPath
  const joiner = targetPath.includes('?') ? '&' : '?'
  return `${targetPath}${joiner}return=${encodeURIComponent(returnPath)}`
}

export function Topology() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [intel, setIntel] = useState<Intelligence | null>(null)
  const [nodes, setNodes] = useState<TopoNode[]>([])
  const [links, setLinks] = useState<TopoLink[]>([])
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState<string | null>(null)
  const [selected, setSelected] = useState<NodeDrill | null>(null)
  const [selLoading, setSelLoading] = useState(false)
  const [nodeFilter, setNodeFilter] = useState<NodeFilterId>('all')
  const [incidentCtx, setIncidentCtx] = useState<Incident | null>(null)
  const [incidentErr, setIncidentErr] = useState<string | null>(null)
  const loadAbortRef = useRef<AbortController | null>(null)
  const urlSyncRef = useRef(false)

  const [vb, setVb] = useState({ x: 0, y: 0, w: 600, h: 480 })
  const svgRef = useRef<SVGSVGElement>(null)
  const dragging = useRef<{ startX: number; startY: number; vbStart: typeof vb } | null>(null)

  const load = useCallback(async () => {
    loadAbortRef.current?.abort()
    const ac = new AbortController()
    loadAbortRef.current = ac
    const { signal } = ac
    setLoading(true)
    setErr(null)
    try {
      const [ji, jn, jl] = await Promise.all([
        fetchTopologyJson<Intelligence>(`${API}`, signal, 'topology summary'),
        fetchTopologyJson<{ nodes?: TopoNode[] }>(`${API}/nodes?limit=500`, signal, 'topology nodes'),
        fetchTopologyJson<{ links?: TopoLink[] }>(`${API}/links?limit=500`, signal, 'topology links'),
      ])
      if (signal.aborted) return
      setIntel(ji)
      setNodes(jn.nodes || [])
      setLinks(jl.links || [])
    } catch (e) {
      if (e instanceof Error && e.name === 'AbortError') return
      setErr(e instanceof Error ? e.message : 'load failed')
    } finally {
      if (!signal.aborted) {
        setLoading(false)
      }
    }
  }, [])

  useEffect(() => {
    void load()
    return () => {
      loadAbortRef.current?.abort()
    }
  }, [load])

  const incidentIdParam = (searchParams.get('incident') || '').trim()
  const returnParam = (searchParams.get('return') || '').trim()
  const selectParam = searchParams.get('select')
  const selectedFromUrl = selectParam != null && selectParam !== '' ? Number(selectParam) : null

  useEffect(() => {
    const f = (searchParams.get('filter') || '').trim() as NodeFilterId
    const allowed: NodeFilterId[] = [
      'all',
      'stale',
      'degraded',
      'risky',
      'no_map_coords',
      'changed_since_visit',
      'incident_focus',
      'transport_scoped',
    ]
    if (f && allowed.includes(f)) {
      setNodeFilter(f)
    } else {
      setNodeFilter('all')
    }
  }, [searchParams])

  useEffect(() => {
    if (!incidentIdParam) {
      setIncidentCtx(null)
      setIncidentErr(null)
      return
    }
    let cancelled = false
    ;(async () => {
      try {
        const res = await fetch(`/api/v1/incidents/${encodeURIComponent(incidentIdParam)}`)
        if (!res.ok) {
          if (!cancelled) {
            setIncidentCtx(null)
            setIncidentErr(`Incident HTTP ${res.status}`)
          }
          return
        }
        const data = (await res.json()) as Incident
        if (!cancelled) {
          setIncidentCtx(data)
          setIncidentErr(null)
        }
      } catch {
        if (!cancelled) {
          setIncidentCtx(null)
          setIncidentErr('Failed to load incident')
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [incidentIdParam])

  function setFilterAndUrl(f: NodeFilterId) {
    urlSyncRef.current = true
    setNodeFilter(f)
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        if (f === 'all') next.delete('filter')
        else next.set('filter', f)
        return next
      },
      { replace: true },
    )
    window.setTimeout(() => {
      urlSyncRef.current = false
    }, 0)
  }

  const riskyNodeNums = useMemo(() => {
    const s = new Set<number>()
    for (const l of links) {
      if (l.contradiction || l.relay_dependent || l.stale || l.quality_score < 0.35) {
        s.add(l.src_node_num)
        s.add(l.dst_node_num)
      }
    }
    return s
  }, [links])

  const shiftSnap = useMemo(() => readShiftSnapshot(), [])
  const topologyChangedSinceVisit = useMemo(() => {
    const prev = shiftSnap?.topologyNodeLastSeen
    if (!prev || Object.keys(prev).length === 0) return new Set<number>()
    const s = new Set<number>()
    for (const n of nodes) {
      const key = String(n.node_num)
      const was = prev[key]
      if (!was) continue
      const cur = n.last_seen_at
      if (!cur) continue
      if (new Date(cur).getTime() > new Date(was).getTime()) s.add(n.node_num)
    }
    return s
  }, [nodes, shiftSnap])

  const incidentFocusNodeNums = useMemo(() => {
    if (!incidentCtx) return new Set<number>()
    const s = new Set<number>()
    const rt = (incidentCtx.resource_type || '').toLowerCase()
    const rid = (incidentCtx.resource_id || '').trim()
    if (rt === 'mesh_node' || rt === 'node') {
      const num = Number(rid.replace(/\D/g, '') || '0')
      if (Number.isFinite(num) && num > 0) s.add(num)
    }
    for (const d of incidentCtx.intelligence?.implicated_domains ?? []) {
      if ((d.domain || '').toLowerCase() !== 'mesh_topology') continue
      for (const ref of d.evidence_refs ?? []) {
        const m = /^node[:_]?(\d+)$/i.exec(ref.trim())
        if (m) s.add(Number(m[1]))
      }
    }
    return s
  }, [incidentCtx])

  const transportScopedNodeNums = useMemo(() => {
    const s = new Set<number>()
    const tn = (incidentCtx?.resource_type || '').toLowerCase() === 'transport' ? (incidentCtx?.resource_id || '').trim() : ''
    if (!tn) return s
    for (const l of links) {
      const path = (l.transport_path || '').toLowerCase()
      if (path.includes(tn.toLowerCase())) {
        s.add(l.src_node_num)
        s.add(l.dst_node_num)
      }
    }
    return s
  }, [links, incidentCtx])

  const filteredNodes = useMemo(() => {
    if (nodeFilter === 'all') return nodes
    return nodes.filter((n) => {
      if (nodeFilter === 'stale') return n.stale || n.health_state === 'stale'
      if (nodeFilter === 'degraded') {
        return (
          n.health_state === 'degraded' ||
          n.health_state === 'isolated' ||
          n.health_state === 'weakly_observed' ||
          n.health_state === 'inferred_only'
        )
      }
      if (nodeFilter === 'risky') return riskyNodeNums.has(n.node_num)
      if (nodeFilter === 'no_map_coords') {
        const lat = n.lat_redacted
        const lon = n.lon_redacted
        const noCoords = lat == null || lon == null || (lat === 0 && lon === 0)
        return noCoords || n.location_state === 'unknown' || !n.location_state
      }
      if (nodeFilter === 'changed_since_visit') return topologyChangedSinceVisit.has(n.node_num)
      if (nodeFilter === 'incident_focus') {
        if (incidentFocusNodeNums.size === 0) return false
        return incidentFocusNodeNums.has(n.node_num)
      }
      if (nodeFilter === 'transport_scoped') {
        if (transportScopedNodeNums.size === 0) return false
        return transportScopedNodeNums.has(n.node_num)
      }
      return true
    })
  }, [
    nodes,
    nodeFilter,
    riskyNodeNums,
    topologyChangedSinceVisit,
    incidentFocusNodeNums,
    transportScopedNodeNums,
  ])

  const filteredNodeSet = useMemo(() => new Set(filteredNodes.map((n) => n.node_num)), [filteredNodes])

  const visibleLinks = useMemo(() => {
    if (nodeFilter === 'all') return links
    return links.filter((l) => filteredNodeSet.has(l.src_node_num) && filteredNodeSet.has(l.dst_node_num))
  }, [links, nodeFilter, filteredNodeSet])

  const positions = useMemo(() => {
    const edges = visibleLinks.map((l) => ({ src: l.src_node_num, dst: l.dst_node_num }))
    return runSimulation(
      filteredNodes.map((n) => n.node_num),
      edges,
      600,
      480,
    )
  }, [filteredNodes, visibleLinks])

  useEffect(() => {
    setVb({ x: 0, y: 0, w: 600, h: 480 })
  }, [positions])

  useEffect(() => {
    if (selected && !filteredNodeSet.has(selected.node.node_num)) {
      setSelected(null)
    }
  }, [selected, filteredNodeSet])

  const syncSelectToUrl = useCallback(
    (num: number | null) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          if (num == null) next.delete('select')
          else next.set('select', String(num))
          return next
        },
        { replace: true },
      )
    },
    [setSearchParams],
  )

  const selectNode = useCallback(
    async (num: number, opts?: { syncUrl?: boolean }) => {
      const syncUrl = opts?.syncUrl !== false
      setSelLoading(true)
      try {
        const res = await fetch(`${API}/nodes/${num}`)
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        setSelected((await res.json()) as NodeDrill)
        if (syncUrl) syncSelectToUrl(num)
      } catch {
        setSelected(null)
        if (syncUrl) syncSelectToUrl(null)
      } finally {
        setSelLoading(false)
      }
    },
    [syncSelectToUrl],
  )

  useEffect(() => {
    if (selectedFromUrl == null || Number.isNaN(selectedFromUrl)) return
    if (selected?.node.node_num === selectedFromUrl) return
    void selectNode(selectedFromUrl, { syncUrl: false })
  }, [selectedFromUrl, selected?.node.node_num, selectNode])

  const selectedNodeNum = selected?.node?.node_num ?? null


  usePageHotkeys([
    { key: '/', description: 'Focus node search', handler: () => nodeSearchRef.current?.focus() },
    { key: '1', description: 'Jump to graph', handler: () => sectionRefs.current.graph?.scrollIntoView({ behavior: 'smooth' }) },
    { key: '2', description: 'Jump to drilldown', handler: () => sectionRefs.current.drill?.scrollIntoView({ behavior: 'smooth' }) },
    { key: '3', description: 'Jump to intelligence', handler: () => sectionRefs.current.intel?.scrollIntoView({ behavior: 'smooth' }) },
  ])

  const mapPoints = useMemo(() => {
    const allowed = new Set(filteredNodes.map((n) => n.node_num))
    return nodes.filter(
      (n) =>
        allowed.has(n.node_num) &&
        (n.location_state === 'exact' || n.location_state === 'approximate') &&
        n.lat_redacted != null &&
        n.lon_redacted != null &&
        (n.lat_redacted !== 0 || n.lon_redacted !== 0),
    )
  }, [nodes, filteredNodes])

  const googleMapsEligible =
    intel?.google_maps_basemap_available === true &&
    typeof intel.google_maps_api_key === 'string' &&
    intel.google_maps_api_key.length > 0

  function zoomBy(factor: number) {
    setVb((v) => {
      const cx = v.x + v.w / 2
      const cy = v.y + v.h / 2
      const nw = Math.max(100, Math.min(1200, v.w * factor))
      const nh = Math.max(80, Math.min(960, v.h * factor))
      return { x: cx - nw / 2, y: cy - nh / 2, w: nw, h: nh }
    })
  }
  function resetView() {
    setVb({ x: 0, y: 0, w: 600, h: 480 })
  }

  function onWheel(e: React.WheelEvent) {
    e.preventDefault()
    const factor = e.deltaY > 0 ? 1.12 : 0.89
    const rect = svgRef.current?.getBoundingClientRect()
    if (!rect) {
      zoomBy(factor)
      return
    }
    const mx = (e.clientX - rect.left) / rect.width
    const my = (e.clientY - rect.top) / rect.height
    setVb((v) => {
      const nw = Math.max(100, Math.min(1200, v.w * factor))
      const nh = Math.max(80, Math.min(960, v.h * factor))
      const nx = v.x + (v.w - nw) * mx
      const ny = v.y + (v.h - nh) * my
      return { x: nx, y: ny, w: nw, h: nh }
    })
  }

  function onMouseDown(e: React.MouseEvent) {
    if (e.button !== 0) return
    dragging.current = { startX: e.clientX, startY: e.clientY, vbStart: vb }
  }
  function onMouseMove(e: React.MouseEvent) {
    if (!dragging.current || !svgRef.current) return
    const rect = svgRef.current.getBoundingClientRect()
    const scaleX = vb.w / rect.width
    const scaleY = vb.h / rect.height
    const dx = (e.clientX - dragging.current.startX) * scaleX
    const dy = (e.clientY - dragging.current.startY) * scaleY
    const s = dragging.current.vbStart
    setVb({ x: s.x - dx, y: s.y - dy, w: s.w, h: s.h })
  }
  function onMouseUp() {
    dragging.current = null
  }

  const panStep = 56
  useEffect(() => {
    const el = svgRef.current
    if (!el) return
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        setSelected(null)
        syncSelectToUrl(null)
        return
      }
      if (e.key === 'ArrowLeft' || e.key === 'ArrowRight' || e.key === 'ArrowUp' || e.key === 'ArrowDown') {
        e.preventDefault()
        setVb((v) => {
          if (e.key === 'ArrowLeft') return { ...v, x: v.x - panStep }
          if (e.key === 'ArrowRight') return { ...v, x: v.x + panStep }
          if (e.key === 'ArrowUp') return { ...v, y: v.y - panStep }
          return { ...v, y: v.y + panStep }
        })
      }
    }
    el.addEventListener('keydown', onKeyDown)
    return () => el.removeEventListener('keydown', onKeyDown)
  }, [syncSelectToUrl])

  const intelDisabled = intel?.topology_enabled === false

  const FILTER_OPTIONS: Array<{ id: NodeFilterId; label: string; hint: string }> = [
    { id: 'all', label: 'All', hint: 'full graph' },
    { id: 'changed_since_visit', label: 'Changed since visit', hint: 'topology last_seen advanced vs your saved command-surface baseline (this browser)' },
    { id: 'stale', label: 'Stale', hint: 'last_seen beyond window' },
    { id: 'degraded', label: 'Degraded / sparse graph', hint: 'degraded, isolated, weakly observed, or inferred-only health states' },
    { id: 'risky', label: 'Link-risky', hint: 'touches relay-dependent, weak, stale, or contradiction edges' },
    { id: 'incident_focus', label: 'Incident focus', hint: 'requires ?incident=… — nodes referenced by incident resource or implicated_domains' },
    { id: 'transport_scoped', label: 'Transport edges', hint: 'requires transport incident — nodes on links whose transport_path mentions the resource_id' },
    { id: 'no_map_coords', label: 'No map coords', hint: 'no redacted lat/lon for scatter/basemap' },
  ]

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-3 min-w-0 flex-1">
          <div className="p-2 rounded-lg bg-primary/10 text-primary shrink-0">
            <GitBranch className="h-6 w-6" />
          </div>
          <div className="min-w-0">
            <h1 className="text-2xl font-semibold tracking-tight">Topology</h1>
            <p className="text-sm text-muted-foreground">
              Graph from stored links (packet relay / destination fields). Not a geographic or RF map unless coordinates are present.
            </p>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-3 shrink-0">
          {returnParam.startsWith('/') && (
            <Link to={returnParam} className="text-sm font-semibold text-primary hover:underline">
              ← Back
            </Link>
          )}
          <button
            type="button"
            onClick={() => void load()}
            className="inline-flex items-center gap-2 px-3 py-2 rounded-md border bg-background hover:bg-muted text-sm"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      {err && (
        <div className="flex items-center gap-2 text-destructive text-sm">
          <AlertCircle className="h-4 w-4 shrink-0" />
          {err}
        </div>
      )}

      {incidentIdParam && (
        <div
          className="rounded-lg border border-primary/25 bg-primary/5 px-3 py-2 text-xs text-muted-foreground"
          role="region"
          aria-label="Incident context for topology"
        >
          {incidentErr && (
            <p className="text-warning">
              Incident context: {incidentErr}{' '}
              <Link
                to={withReturnParam(`/incidents/${encodeURIComponent(incidentIdParam)}`, returnParam)}
                className="text-primary font-medium hover:underline"
              >
                Open incident
              </Link>
            </p>
          )}
          {!incidentErr && incidentCtx && (
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
              <span className="font-medium text-foreground">Focused for incident</span>
              <Link
                to={withReturnParam(`/incidents/${encodeURIComponent(incidentCtx.id)}`, returnParam)}
                className="inline-flex items-center gap-1 text-primary font-medium hover:underline"
              >
                {incidentCtx.title || incidentCtx.id.slice(0, 12)}
                <ExternalLink className="h-3 w-3 opacity-60" aria-hidden />
              </Link>
              <span className="text-muted-foreground">
                {incidentCtx.resource_type || '—'} / {incidentCtx.resource_id || '—'}
              </span>
              <Link
                to={withReturnParam(`/planning?incident=${encodeURIComponent(incidentCtx.id)}`, returnParam)}
                className="text-primary hover:underline font-medium"
              >
                Planning (same incident) →
              </Link>
            </div>
          )}
          {!incidentErr && !incidentCtx && <p>Loading incident context…</p>}
        </div>
      )}

      {intelDisabled && (
        <p className="text-muted-foreground text-sm">
          {(intel as { message?: string })?.message || 'Topology model is disabled in config.'}
        </p>
      )}

      {intel && !intelDisabled && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <SummaryCard
            label="View mode"
            value={intel.view_mode || '—'}
            hint="graph unless map reporting + coordinates justify map"
          />
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

      {intel?.mesh_intelligence && !intelDisabled && <MeshIntelPanel intel={intel.mesh_intelligence} />}

      {intel?.analysis && !intelDisabled && (
        <TopologyOperatorAnalysisPanel
          analysis={intel.analysis}
          staleness={intel.staleness}
          incidentId={incidentIdParam || undefined}
        />
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-4">
          <div className="rounded-xl border bg-card overflow-hidden">
            <div className="flex items-center justify-between px-4 py-2.5 border-b border-border/60 bg-muted/20">
              <div>
                <h2 className="text-sm font-medium">Topology graph</h2>
                <p className="text-xs text-muted-foreground">
                  Force-directed layout · click node to inspect · scroll/drag to navigate
                </p>
              </div>
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => zoomBy(0.85)}
                  className="rounded p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
                  title="Zoom in"
                >
                  <ZoomIn className="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  onClick={() => zoomBy(1.18)}
                  className="rounded p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
                  title="Zoom out"
                >
                  <ZoomOut className="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  onClick={resetView}
                  className="rounded p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
                  title="Reset view"
                >
                  <Maximize2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 px-4 py-2 border-b border-border/40 bg-muted/10">
              <LegendItem color="hsl(142 65% 38%)" label="Healthy" />
              <LegendItem color="hsl(38 90% 45%)" label="Stale" />
              <LegendItem color="hsl(28 90% 48%)" label="Degraded" />
              <LegendItem color="hsl(280 50% 52%)" label="Isolated" />
              <LegendItem color="hsl(210 65% 46%)" label="Unknown" />
              <LegendItem color="hsl(200 55% 42%)" label="Weak / inferred-only" />
              <span className="text-[10px] text-muted-foreground ml-auto max-w-[min(100%,280px)] sm:max-w-none sm:ml-auto">
                Solid line = packet-observed edge · dashed = inferred · thin = relay-dependent · double ring = newer last_seen vs your baseline
              </span>
            </div>

            <div
              className="flex flex-wrap items-center gap-2 px-4 py-2 border-b border-border/40 bg-card/30"
              role="toolbar"
              aria-label="Filter nodes in graph"
            >
              <span className="text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Focus</span>
              {FILTER_OPTIONS.map((f) => (
                <button
                  key={f.id}
                  type="button"
                  onClick={() => setFilterAndUrl(f.id)}
                  title={f.hint}
                  className={`rounded-full border px-2.5 py-1 text-[11px] font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                    nodeFilter === f.id
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border text-muted-foreground hover:bg-muted/60'
                  }`}
                >
                  {f.label}
                </button>
              ))}
              <span className="text-[10px] text-muted-foreground ml-auto text-right max-w-[min(100%,320px)] sm:max-w-none">
                Showing {filteredNodes.length}/{nodes.length} nodes
                {nodeFilter === 'changed_since_visit' && topologyChangedSinceVisit.size === 0 && (
                  <span className="block text-warning">No matches — mark “caught up” on the command surface to record topology baselines.</span>
                )}
                {nodeFilter === 'incident_focus' && incidentFocusNodeNums.size === 0 && incidentCtx && (
                  <span className="block text-warning">No implicated node numbers in this incident — use All or Stale, or link a mesh_node resource.</span>
                )}
                {nodeFilter === 'transport_scoped' && transportScopedNodeNums.size === 0 && (
                  <span className="block text-warning">Needs a transport-scoped incident or matching transport_path on edges.</span>
                )}
                <span className="block sm:inline sm:ml-1">· focus graph · arrows pan · Esc clears selection</span>
              </span>
            </div>

            {nodes.length === 0 ? (
              <p className="p-6 text-sm text-muted-foreground">No nodes in topology store yet.</p>
            ) : filteredNodes.length === 0 ? (
              <p className="p-6 text-sm text-muted-foreground">No nodes match this filter.</p>
            ) : (
              <svg
                ref={svgRef}
                tabIndex={0}
                viewBox={`${vb.x} ${vb.y} ${vb.w} ${vb.h}`}
                className="w-full max-h-[480px] cursor-grab active:cursor-grabbing select-none text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-b-xl"
                role="application"
                aria-label="Topology graph: nodes and inferred links from ingest. Use arrow keys to pan when focused."
                onWheel={onWheel}
                onMouseDown={onMouseDown}
                onMouseMove={onMouseMove}
                onMouseUp={onMouseUp}
                onMouseLeave={onMouseUp}
              >
                {visibleLinks.map((l) => {
                  const a = positions.get(l.src_node_num)
                  const b = positions.get(l.dst_node_num)
                  if (!a || !b) return null
                  const weak = l.stale || l.quality_score < 0.35
                  const opacity = weak ? 0.25 : l.observed ? 0.55 : 0.38
                  const width = l.relay_dependent ? 0.8 : l.quality_score > 0.7 ? 2 : 1.5
                  return (
                    <line
                      key={l.edge_id}
                      x1={a.x}
                      y1={a.y}
                      x2={b.x}
                      y2={b.y}
                      stroke="currentColor"
                      strokeOpacity={opacity}
                      strokeWidth={width}
                      strokeDasharray={l.observed ? undefined : '4 3'}
                      className={l.contradiction ? 'text-warning' : undefined}
                    />
                  )
                })}

                {filteredNodes.map((n) => {
                  const p = positions.get(n.node_num)
                  if (!p) return null
                  const fill = nodeColor(n)
                  const isSelected = selectedNodeNum === n.node_num
                  const r = isSelected ? 13 : 10
                  const label = nodeLabel(n)
                  const changedMark = topologyChangedSinceVisit.has(n.node_num)
                  return (
                    <g
                      key={n.node_num}
                      className="cursor-pointer"
                      role="button"
                      tabIndex={0}
                      aria-label={`Node ${label}, open drilldown`}
                      aria-pressed={isSelected}
                      onClick={(e) => {
                        e.stopPropagation()
                        void selectNode(n.node_num, { syncUrl: true })
                      }}
                      onKeyDown={(ev) => {
                        if (ev.key === 'Enter' || ev.key === ' ') {
                          ev.preventDefault()
                          void selectNode(n.node_num, { syncUrl: true })
                        }
                      }}
                    >
                      {isSelected && (
                        <circle
                          cx={p.x}
                          cy={p.y}
                          r={r + 4}
                          fill="none"
                          stroke="currentColor"
                          strokeWidth={1.5}
                          strokeOpacity={0.4}
                        />
                      )}
                      {changedMark && (
                        <circle
                          cx={p.x}
                          cy={p.y}
                          r={r + 6}
                          fill="none"
                          stroke="currentColor"
                          strokeWidth={1}
                          strokeOpacity={0.35}
                          strokeDasharray="2 2"
                        />
                      )}
                      <circle
                        cx={p.x}
                        cy={p.y}
                        r={r}
                        fill={fill}
                        stroke="currentColor"
                        strokeWidth={isSelected ? 1.5 : 0.8}
                        strokeOpacity={isSelected ? 0.8 : 0.4}
                        fillOpacity={n.stale ? 0.65 : 1}
                      />
                      <text
                        x={p.x + r + 3}
                        y={p.y + 4}
                        fontSize={n.stale ? '9' : '10'}
                        fill="currentColor"
                        fillOpacity={n.stale ? 0.55 : 0.9}
                        className="select-none pointer-events-none"
                      >
                        {label}
                      </text>
                    </g>
                  )
                })}
              </svg>
            )}
          </div>

          {(intel?.view_mode === 'map' || intel?.view_mode === 'map_partial') && mapPoints.length > 0 && (
            <div className="rounded-xl border bg-card p-4 space-y-3">
              <div>
                <h2 className="text-sm font-medium mb-1">Location layer (redacted coordinates)</h2>
                <p className="text-xs text-muted-foreground">
                  Points use server redacted lat/lon only. This is not RF proof or surveyed survey-grade placement.
                  {googleMapsEligible
                    ? ' Optional Google basemap loads third-party tiles when enabled in config (privacy.map_reporting_allowed + features.google_maps_in_topology_ui + API key env).'
                    : ' Default view is a local scatter plot with no third-party map provider.'}
                </p>
              </div>
              {googleMapsEligible ? (
                <MapGoogleBasemap
                  apiKey={intel!.google_maps_api_key!}
                  nodes={mapPoints}
                  selectedNodeNum={selectedNodeNum}
                  onSelectNode={(num) => void selectNode(num, { syncUrl: true })}
                />
              ) : (
                <MapScatter
                  nodes={mapPoints}
                  selectedNodeNum={selectedNodeNum}
                  onSelectNode={(num) => void selectNode(num, { syncUrl: true })}
                />
              )}
            </div>
          )}
        </div>

        <div className="rounded-xl border bg-card p-4 min-h-[200px]">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-medium">Node drilldown</h2>
            {selected && (
              <button
                type="button"
                onClick={() => {
                  setSelected(null)
                  syncSelectToUrl(null)
                }}
                className="rounded p-1 text-muted-foreground hover:text-foreground"
                aria-label="Clear node selection"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
          {selLoading && <p className="text-sm text-muted-foreground">Loading…</p>}
          {!selLoading && !selected && <p className="text-sm text-muted-foreground">Select a node on the graph.</p>}
          {selected && <NodeDrillPanel drill={selected} />}
        </div>
      </div>

      {(intel?.analysis?.recommendations?.length ?? 0) > 0 && (
        <div className="rounded-xl border bg-card p-4">
          <h2 className="text-sm font-medium mb-2">Evidence-based recommendations</h2>
          <ul className="text-sm space-y-2">
            {intel!.analysis!.recommendations!.slice(0, 12).map((r) => (
              <li key={r.id} className="border-b border-border/50 pb-2 last:border-0">
                <div>{r.summary}</div>
                <div className="text-xs text-muted-foreground">rank score {r.confidence.toFixed(2)} (not propagation proof)</div>
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

function TopologyOperatorAnalysisPanel({
  analysis,
  staleness,
  incidentId,
}: {
  analysis: NonNullable<Intelligence['analysis']>
  staleness?: Intelligence['staleness']
  incidentId?: string
}) {
  const weak = analysis.weak_clusters ?? []
  const staleRegs = analysis.stale_regions ?? []
  const bottlenecks = analysis.bottlenecks ?? []
  const isolated = analysis.isolated_nodes ?? []
  const bridges = analysis.bridge_nodes ?? []

  if (
    weak.length === 0 &&
    staleRegs.length === 0 &&
    bottlenecks.length === 0 &&
    isolated.length === 0 &&
    bridges.length === 0
  ) {
    return null
  }

  return (
    <div className="rounded-xl border bg-card p-4 space-y-3" data-testid="topology-operator-analysis">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h2 className="text-sm font-medium text-foreground">Fault domains &amp; impact (graph-bounded)</h2>
          <p className="text-[11px] text-muted-foreground mt-0.5 max-w-3xl">
            Connected components and stale regions from stored topology_links — not RF or geographic causality. Use with transport health and
            incident evidence.
            {staleness?.node_stale_minutes != null && (
              <span className="block sm:inline sm:ml-1">
                Stale thresholds: node {staleness.node_stale_minutes}m, link {staleness.link_stale_minutes ?? '—'}m (config).
              </span>
            )}
          </p>
          <p className="text-[11px] text-foreground/85 border-l-2 border-warning/30 pl-2 mt-2 max-w-3xl leading-snug" role="note">
            Pattern support only: absent edges or quiet clusters do not prove “nothing is nearby” on air; map-like layout is ingest-evidence, not
            propagation certainty.
          </p>
        </div>
        {incidentId && (
          <Link
            to={`/incidents/${encodeURIComponent(incidentId)}`}
            className="text-xs font-medium text-primary hover:underline shrink-0"
          >
            Back to incident →
          </Link>
        )}
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {weak.length > 0 && (
          <div className="rounded-lg border border-border/60 p-3">
            <p className="text-xs font-semibold text-foreground mb-1">Weaker clusters</p>
            <ul className="text-[11px] text-muted-foreground space-y-1.5 max-h-36 overflow-y-auto">
              {weak.slice(0, 6).map((c) => (
                <li key={c.id}>
                  <span className="font-mono text-foreground">{c.id}</span> · state {c.state} · nodes {c.node_nums.length} · edges{' '}
                  {c.edge_count} · min health {c.min_score.toFixed(2)}
                </li>
              ))}
            </ul>
          </div>
        )}
        {staleRegs.length > 0 && (
          <div className="rounded-lg border border-warning/25 bg-warning/5 p-3">
            <p className="text-xs font-semibold text-foreground mb-1">Stale regions</p>
            <ul className="text-[11px] text-muted-foreground space-y-1.5 max-h-36 overflow-y-auto">
              {staleRegs.slice(0, 5).map((r, i) => (
                <li key={i}>
                  {r.node_nums.length} nodes · stale ratio {(r.stale_ratio * 100).toFixed(0)}%
                  {r.last_fresh_at && <span className="block text-[10px]">last fresh hint: {r.last_fresh_at}</span>}
                </li>
              ))}
            </ul>
          </div>
        )}
        {bottlenecks.length > 0 && (
          <div className="rounded-lg border border-border/60 p-3 sm:col-span-2 lg:col-span-1">
            <p className="text-xs font-semibold text-foreground mb-1">Bottlenecks (graph shape)</p>
            <ul className="text-[11px] text-muted-foreground space-y-1.5 max-h-36 overflow-y-auto">
              {bottlenecks.slice(0, 6).map((b, i) => (
                <li key={i}>
                  <span className="font-medium text-foreground">{b.type.replace(/_/g, ' ')}</span> ({b.severity}) — {b.explanation}
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
      {(isolated.length > 0 || bridges.length > 0) && (
        <p className="text-[11px] text-muted-foreground">
          {isolated.length > 0 && (
            <span>
              Isolated in graph: {isolated.slice(0, 12).join(', ')}
              {isolated.length > 12 ? '…' : ''}.{' '}
            </span>
          )}
          {bridges.length > 0 && (
            <span>
              Bridge / articulation candidates: {bridges.slice(0, 12).join(', ')}
              {bridges.length > 12 ? '…' : ''}.
            </span>
          )}
        </p>
      )}
    </div>
  )
}

function MeshIntelPanel({ intel }: { intel: MeshIntelligence }) {
  const [open, setOpen] = useState(false)
  return (
    <div className="rounded-xl border bg-card">
      <button
        type="button"
        className="flex w-full items-center justify-between px-4 py-3 text-sm font-medium"
        onClick={() => setOpen((o) => !o)}
      >
        <span>Mesh deployment intelligence</span>
        <span className="text-xs text-muted-foreground">{open ? 'collapse' : 'expand'}</span>
      </button>
      {open && (
        <div className="px-4 pb-4 space-y-4 border-t border-border/60">
          <p className="text-xs text-muted-foreground pt-3">
            Derived from observed packets and graph — not RF proof. Advisory only; MEL does not change routing.
          </p>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <SummaryCard
              label="Bootstrap viability"
              value={(intel.bootstrap?.viability || '—').replace(/_/g, ' ')}
              hint={`lone wolf ${(intel.bootstrap?.lone_wolf_score ?? 0).toFixed(2)} · readiness ${(intel.bootstrap?.bootstrap_readiness_score ?? 0).toFixed(2)}`}
            />
            <SummaryCard label="Confidence" value={intel.bootstrap?.confidence || '—'} hint="raises with more nodes + message history" />
            <SummaryCard
              label="Topology shape"
              value={(intel.topology?.cluster_shape || '—').replace(/_/g, ' ')}
              hint={`fragmentation ${(intel.topology?.fragmentation_score ?? 0).toFixed(2)}`}
            />
            <SummaryCard
              label="Protocol fit"
              value={(intel.protocol_fit?.fit_class || '—').replace(/_/g, ' ')}
              hint={(intel.protocol_fit?.architecture_class || '').replace(/_/g, ' ')}
            />
          </div>
          {intel.bootstrap?.explanation?.top_next_action && (
            <div className="text-sm border-l-2 border-primary/40 pl-3">
              <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                Highest-leverage next step
              </span>
              <p className="mt-1">{intel.bootstrap.explanation.top_next_action}</p>
            </div>
          )}
          {(intel.routing_pressure?.summary_lines?.length ?? 0) > 0 && (
            <div>
              <div className="text-xs font-medium text-muted-foreground mb-1">Routing / flood pressure (suspected)</div>
              <ul className="text-xs text-muted-foreground list-disc pl-4 space-y-1">
                {intel.routing_pressure!.summary_lines!.map((line, i) => (
                  <li key={i}>{line}</li>
                ))}
              </ul>
            </div>
          )}
          {(intel.recommendations?.length ?? 0) > 0 && (
            <div>
              <div className="text-xs font-medium text-muted-foreground mb-2">Ranked recommendations</div>
              <ul className="text-sm space-y-2">
                {intel.recommendations!.slice(0, 8).map((r) => (
                  <li key={r.rank} className="border-b border-border/50 pb-2 last:border-0">
                    <div>
                      <span className="text-xs text-muted-foreground mr-2">#{r.rank}</span>
                      {r.title}
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5">
                      {r.class.replace(/_/g, ' ')} · sev {r.severity} · rank {r.confidence.toFixed(2)} (internal score, not RF certainty)
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          )}
          {intel.evidence_model && <p className="text-[11px] text-muted-foreground border-t pt-3">{intel.evidence_model}</p>}
        </div>
      )}
    </div>
  )
}

function NodeDrillPanel({ drill: d }: { drill: NodeDrill }) {
  const truth = topologyNodeTruthSummary({
    stale: d.node.stale,
    health_state: d.node.health_state,
    last_seen_at: d.node.last_seen_at,
  })
  return (
    <div className="space-y-3 text-sm">
      <div>
        <div className="font-medium">{d.node.long_name || d.node.short_name || d.node.node_num}</div>
        <div className="text-xs text-muted-foreground font-mono">
          #{d.node.node_num} {d.node.node_id}
        </div>
      </div>
      <div className="flex items-center gap-2 text-xs">
        <span className="h-2.5 w-2.5 rounded-full shrink-0" style={{ backgroundColor: nodeColor(d.node) }} />
        <span>{d.scored_state}</span>
        <span className="text-muted-foreground">({d.scored_health.toFixed(2)})</span>
      </div>
      <p className="text-[11px] text-muted-foreground border-l-2 border-muted pl-2">{truth}</p>
      {(d.node.first_seen_at || d.node.last_seen_at) && (
        <dl className="grid grid-cols-[auto_1fr] gap-x-2 gap-y-0.5 text-[11px] text-muted-foreground">
          {d.node.first_seen_at && (
            <>
              <dt className="font-medium text-foreground/80">First seen</dt>
              <dd className="font-mono">{d.node.first_seen_at}</dd>
            </>
          )}
          {d.node.last_seen_at && (
            <>
              <dt className="font-medium text-foreground/80">Last seen</dt>
              <dd className="font-mono">{d.node.last_seen_at}</dd>
            </>
          )}
        </dl>
      )}
      {(d.node.trust_class || d.node.source_connector) && (
        <p className="text-[11px] text-muted-foreground">
          Trust: {d.node.trust_class?.replace(/_/g, ' ') ?? '—'}
          {d.node.source_connector ? ` · connector ${d.node.source_connector}` : ''}
        </p>
      )}
      {d.freshness_age_seconds != null && d.freshness_age_seconds >= 0 && (
        <p className="text-xs text-muted-foreground">Last seen ≈ {Math.round(d.freshness_age_seconds)}s ago (server clock)</p>
      )}
      {(d.score_factors?.length ?? 0) > 0 && (
        <div>
          <div className="text-xs font-medium text-muted-foreground mb-1">Score factors</div>
          <ul className="text-xs space-y-1 max-h-44 overflow-y-auto">
            {d.score_factors!.map((f) => (
              <li key={f.name}>
                <span className="font-mono">{f.name}</span>: {f.contribution.toFixed(3)}{' '}
                <span className="text-muted-foreground">({f.basis})</span>
                {f.evidence && <span className="block text-muted-foreground truncate">{f.evidence}</span>}
              </li>
            ))}
          </ul>
        </div>
      )}
      {(d.links?.length ?? 0) > 0 && (
        <div>
          <div className="text-xs font-medium text-muted-foreground mb-1">Adjacent edges</div>
          <ul className="text-[10px] space-y-1 max-h-32 overflow-y-auto font-mono text-muted-foreground">
            {d.links!.slice(0, 8).map((l) => (
              <li key={l.edge_id}>
                →{l.dst_node_num} {topologyLinkTruthSummary(l)}
              </li>
            ))}
          </ul>
        </div>
      )}
      {(d.next_actions?.length ?? 0) > 0 && (
        <div>
          <div className="text-xs font-medium text-muted-foreground mb-1">Next checks (deterministic)</div>
          <ul className="text-xs list-disc pl-4 space-y-1">
            {d.next_actions!.map((a, i) => (
              <li key={i}>{a}</li>
            ))}
          </ul>
        </div>
      )}
      {d.mesh_intel && (
        <div className="border-t pt-3 mt-1">
          <div className="text-xs font-medium text-muted-foreground mb-2">Deployment intelligence</div>
          <div className="text-xs space-y-1">
            <div>
              Coverage: <span className="font-mono">{d.mesh_intel.coverage_contribution_score.toFixed(2)}</span>
            </div>
            <div>
              Relay value: <span className="font-mono">{d.mesh_intel.relay_value_score.toFixed(2)}</span>
            </div>
            <div>
              Placement: <span className="font-mono">{d.mesh_intel.placement_quality_score.toFixed(2)}</span>
            </div>
            {d.mesh_intel.is_bridge_critical && <div className="text-warning font-medium">Bridge-critical in observed graph</div>}
            {(d.mesh_intel.notes?.length ?? 0) > 0 && (
              <ul className="list-disc pl-4 text-muted-foreground space-y-0.5">
                {d.mesh_intel.notes!.map((note, i) => (
                  <li key={i}>{note.replace(/_/g, ' ')}</li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function SummaryCard({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <div className="rounded-lg border bg-card p-3">
      <div className="text-xs text-muted-foreground uppercase tracking-wide">{label}</div>
      <div className="text-lg font-semibold mt-1 leading-snug">{value}</div>
      <div className="text-[11px] text-muted-foreground mt-1 leading-snug">{hint}</div>
    </div>
  )
}

function LegendItem({ color, label }: { color: string; label: string }) {
  return (
    <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
      <span className="h-2.5 w-2.5 rounded-full shrink-0" style={{ backgroundColor: color }} />
      {label}
    </div>
  )
}

function MapScatter({
  nodes,
  selectedNodeNum,
  onSelectNode,
}: {
  nodes: TopoNode[]
  selectedNodeNum: number | null
  onSelectNode: (nodeNum: number) => void
}) {
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
    <svg
      viewBox={`0 0 ${w} ${h}`}
      className="w-full max-h-[300px]"
      role="img"
      aria-label="Redacted coordinate scatter plot; click a point to open node drilldown"
    >
      <rect width={w} height={h} fill="none" stroke="currentColor" strokeOpacity={0.15} />
      {nodes.map((n) => {
        const cx = xLon(n.lon_redacted ?? 0)
        const cy = yLat(n.lat_redacted ?? 0)
        const isSel = selectedNodeNum === n.node_num
        const name = n.short_name || n.long_name || `node ${n.node_num}`
        return (
          <g
            key={n.node_num}
            className="cursor-pointer"
            role="button"
            tabIndex={0}
            aria-label={`${name}, redacted position, open drilldown`}
            aria-pressed={isSel}
            onClick={() => onSelectNode(n.node_num)}
            onKeyDown={(ev) => {
              if (ev.key === 'Enter' || ev.key === ' ') {
                ev.preventDefault()
                onSelectNode(n.node_num)
              }
            }}
          >
            <title>
              {name} · redacted lat/lon · {n.stale ? 'stale' : 'recent'}
            </title>
            <circle
              cx={cx}
              cy={cy}
              r={isSel ? 8 : 6}
              fill={nodeColor(n)}
              stroke="currentColor"
              strokeWidth={isSel ? 2 : 1}
            />
          </g>
        )
      })}
    </svg>
  )
}

/** Optional Google Maps basemap: only mounted when server exposes google_maps_api_key (gated by config + env). */
function MapGoogleBasemap({
  apiKey,
  nodes,
  selectedNodeNum,
  onSelectNode,
}: {
  apiKey: string
  nodes: TopoNode[]
  selectedNodeNum: number | null
  onSelectNode: (nodeNum: number) => void
}) {
  const containerRef = useRef<HTMLDivElement>(null)
  const mapRef = useRef<google.maps.Map | null>(null)
  const markersRef = useRef<google.maps.Marker[]>([])

  useEffect(() => {
    const el = containerRef.current
    if (!el || !apiKey) return

    const mapsLib = (): typeof google.maps | undefined =>
      typeof window !== 'undefined' ? (window as unknown as { google?: { maps: typeof google.maps } }).google?.maps : undefined

    const clearMarkers = () => {
      for (const m of markersRef.current) {
        m.setMap(null)
      }
      markersRef.current = []
    }

    const buildMarkers = () => {
      const maps = mapsLib()
      const map = mapRef.current
      if (!maps || !map || nodes.length === 0) return
      clearMarkers()
      const bounds = new maps.LatLngBounds()
      for (const n of nodes) {
        const lat = n.lat_redacted ?? 0
        const lng = n.lon_redacted ?? 0
        const pos = { lat, lng }
        bounds.extend(pos)
        const marker = new maps.Marker({
          position: pos,
          map,
          title: n.short_name || n.long_name || `node ${n.node_num}`,
          opacity: n.stale ? 0.65 : 1,
        })
        marker.addListener('click', () => onSelectNode(n.node_num))
        markersRef.current.push(marker)
      }
      try {
        map.fitBounds(bounds)
      } catch {
        /* ignore */
      }
    }

    const initWhenReady = () => {
      const maps = mapsLib()
      if (!maps || !containerRef.current) return
      if (!mapRef.current) {
        mapRef.current = new maps.Map(containerRef.current, {
          mapTypeControl: false,
          streetViewControl: false,
          fullscreenControl: true,
        })
      }
      buildMarkers()
    }

    const existing = document.querySelector('script[data-mel-google-maps="1"]')
    if (mapsLib()) {
      initWhenReady()
      return () => clearMarkers()
    }

    const onLoad = () => initWhenReady()

    if (existing) {
      existing.addEventListener('load', onLoad)
      return () => {
        existing.removeEventListener('load', onLoad)
        clearMarkers()
      }
    }

    const script = document.createElement('script')
    script.src = `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(apiKey)}`
    script.async = true
    script.defer = true
    script.setAttribute('data-mel-google-maps', '1')
    script.addEventListener('load', onLoad)
    document.head.appendChild(script)

    return () => {
      script.removeEventListener('load', onLoad)
      clearMarkers()
    }
  }, [apiKey, nodes, onSelectNode])

  useEffect(() => {
    markersRef.current.forEach((marker, i) => {
      const n = nodes[i]
      if (!n) return
      marker.setZIndex(selectedNodeNum === n.node_num ? 1000 : 0)
    })
  }, [selectedNodeNum, nodes])

  return (
    <div className="space-y-2">
      <p className="text-[11px] text-warning">
        Third-party map: your browser loads Google Maps. Key is delivered to this session from the MEL API — restrict the key by HTTP referrer and
        treat remote/unauthenticated exposure as a credential leak risk.
      </p>
      <div ref={containerRef} className="w-full h-[280px] rounded-md border border-border/60 overflow-hidden bg-muted/20" />
    </div>
  )
}
