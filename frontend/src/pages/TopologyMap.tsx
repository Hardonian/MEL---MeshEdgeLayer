import { useState, useEffect, useRef, useCallback } from 'react'
import { MapPin, RefreshCw, Layers, AlertTriangle, Radio, Wifi } from 'lucide-react'

interface TopoNode {
  node_num: number
  node_id: string
  long_name: string
  short_name: string
  health_state: string
  health_score: number
  lat_redacted: number
  lon_redacted: number
  stale: boolean
  quarantined: boolean
  last_snr: number
  last_rssi: number
  trust_class: string
}

interface TopoLink {
  edge_id: string
  src_node_num: number
  dst_node_num: number
  quality_score: number
  stale: boolean
  observed: boolean
  observation_count: number
}

interface TopoSnapshot {
  snapshot_id: string
  created_at: string
  node_count: number
  edge_count: number
  healthy_nodes: number
  degraded_nodes: number
  stale_nodes: number
  isolated_nodes: number
}

const healthColors: Record<string, string> = {
  healthy: '#22c55e',
  degraded: '#f59e0b',
  unstable: '#ef4444',
  stale: '#6b7280',
  weakly_observed: '#a855f7',
  inferred_only: '#8b5cf6',
  isolated: '#dc2626',
  bridge_critical: '#f97316',
  flapping: '#eab308',
  quarantined: '#991b1b',
  unknown: '#9ca3af',
}

export function TopologyMap() {
  const [nodes, setNodes] = useState<TopoNode[]>([])
  const [links, setLinks] = useState<TopoLink[]>([])
  const [snapshot, setSnapshot] = useState<TopoSnapshot | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedNode, setSelectedNode] = useState<TopoNode | null>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const [viewTransform, setViewTransform] = useState({ x: 0, y: 0, scale: 1 })
  const dragRef = useRef<{ dragging: boolean; lastX: number; lastY: number }>({ dragging: false, lastX: 0, lastY: 0 })

  const fetchData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [nodesRes, linksRes, snapRes] = await Promise.all([
        fetch('/api/topology/nodes?limit=500'),
        fetch('/api/topology/links?limit=2000'),
        fetch('/api/topology/snapshots?limit=1'),
      ])
      if (!nodesRes.ok || !linksRes.ok) throw new Error('Failed to fetch topology data')
      const nodesData = await nodesRes.json()
      const linksData = await linksRes.json()
      setNodes(Array.isArray(nodesData) ? nodesData : [])
      setLinks(Array.isArray(linksData) ? linksData : [])
      if (snapRes.ok) {
        const snapData = await snapRes.json()
        setSnapshot(Array.isArray(snapData) && snapData.length > 0 ? snapData[0] : null)
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchData() }, [fetchData])

  // Build node position map (use lat/lon if available, else force-directed layout)
  const geoNodes = nodes.filter(n => n.lat_redacted !== 0 || n.lon_redacted !== 0)
  const hasGeo = geoNodes.length > nodes.length * 0.3

  const nodePositions = useRef<Map<number, { x: number; y: number }>>(new Map())

  useEffect(() => {
    if (nodes.length === 0) return
    const positions = new Map<number, { x: number; y: number }>()

    if (hasGeo) {
      // Use geographic positions
      let minLat = Infinity, maxLat = -Infinity, minLon = Infinity, maxLon = -Infinity
      for (const n of nodes) {
        if (n.lat_redacted !== 0 || n.lon_redacted !== 0) {
          minLat = Math.min(minLat, n.lat_redacted)
          maxLat = Math.max(maxLat, n.lat_redacted)
          minLon = Math.min(minLon, n.lon_redacted)
          maxLon = Math.max(maxLon, n.lon_redacted)
        }
      }
      const latRange = Math.max(maxLat - minLat, 0.001)
      const lonRange = Math.max(maxLon - minLon, 0.001)
      const padding = 80
      const w = 800 - padding * 2
      const h = 600 - padding * 2

      for (const n of nodes) {
        if (n.lat_redacted !== 0 || n.lon_redacted !== 0) {
          positions.set(n.node_num, {
            x: padding + ((n.lon_redacted - minLon) / lonRange) * w,
            y: padding + ((maxLat - n.lat_redacted) / latRange) * h,
          })
        } else {
          // Random position for nodes without geo
          positions.set(n.node_num, {
            x: padding + Math.random() * w,
            y: padding + Math.random() * h,
          })
        }
      }
    } else {
      // Force-directed layout for nodes without geo
      const w = 800, h = 600
      for (const n of nodes) {
        positions.set(n.node_num, {
          x: w * 0.2 + Math.random() * w * 0.6,
          y: h * 0.2 + Math.random() * h * 0.6,
        })
      }
      // Simple force-directed iterations
      const adj = new Map<number, Set<number>>()
      for (const l of links) {
        if (!adj.has(l.src_node_num)) adj.set(l.src_node_num, new Set())
        if (!adj.has(l.dst_node_num)) adj.set(l.dst_node_num, new Set())
        adj.get(l.src_node_num)!.add(l.dst_node_num)
        adj.get(l.dst_node_num)!.add(l.src_node_num)
      }
      for (let iter = 0; iter < 50; iter++) {
        for (const n of nodes) {
          const pos = positions.get(n.node_num)!
          let fx = 0, fy = 0
          // Repulsion from all nodes
          for (const m of nodes) {
            if (m.node_num === n.node_num) continue
            const mpos = positions.get(m.node_num)!
            const dx = pos.x - mpos.x
            const dy = pos.y - mpos.y
            const dist = Math.max(Math.sqrt(dx * dx + dy * dy), 1)
            const force = 5000 / (dist * dist)
            fx += (dx / dist) * force
            fy += (dy / dist) * force
          }
          // Attraction from linked nodes
          const neighbors = adj.get(n.node_num)
          if (neighbors) {
            for (const m of neighbors) {
              const mpos = positions.get(m)
              if (!mpos) continue
              const dx = mpos.x - pos.x
              const dy = mpos.y - pos.y
              const dist = Math.sqrt(dx * dx + dy * dy)
              const force = dist * 0.01
              fx += (dx / Math.max(dist, 1)) * force
              fy += (dy / Math.max(dist, 1)) * force
            }
          }
          // Center gravity
          fx += (w / 2 - pos.x) * 0.001
          fy += (h / 2 - pos.y) * 0.001
          pos.x = Math.max(40, Math.min(w - 40, pos.x + fx * 0.5))
          pos.y = Math.max(40, Math.min(h - 40, pos.y + fy * 0.5))
        }
      }
    }
    nodePositions.current = positions
  }, [nodes, links, hasGeo])

  // Canvas rendering
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || nodes.length === 0) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const dpr = window.devicePixelRatio || 1
    const rect = canvas.getBoundingClientRect()
    canvas.width = rect.width * dpr
    canvas.height = rect.height * dpr
    ctx.scale(dpr, dpr)

    ctx.clearRect(0, 0, rect.width, rect.height)
    ctx.save()
    ctx.translate(viewTransform.x, viewTransform.y)
    ctx.scale(viewTransform.scale, viewTransform.scale)

    // Draw links
    for (const link of links) {
      const srcPos = nodePositions.current.get(link.src_node_num)
      const dstPos = nodePositions.current.get(link.dst_node_num)
      if (!srcPos || !dstPos) continue

      ctx.beginPath()
      ctx.moveTo(srcPos.x, srcPos.y)
      ctx.lineTo(dstPos.x, dstPos.y)
      ctx.strokeStyle = link.stale ? '#374151' : `rgba(59, 130, 246, ${0.3 + link.quality_score * 0.5})`
      ctx.lineWidth = link.stale ? 0.5 : 1 + link.quality_score
      if (link.stale) ctx.setLineDash([4, 4])
      else ctx.setLineDash([])
      ctx.stroke()
    }
    ctx.setLineDash([])

    // Draw nodes
    const nodeMap = new Map(nodes.map(n => [n.node_num, n]))
    for (const n of nodes) {
      const pos = nodePositions.current.get(n.node_num)
      if (!pos) continue

      const color = healthColors[n.health_state] || healthColors.unknown
      const radius = n.stale ? 4 : 6 + n.health_score * 4
      const isSelected = selectedNode?.node_num === n.node_num

      // Glow for selected
      if (isSelected) {
        ctx.beginPath()
        ctx.arc(pos.x, pos.y, radius + 6, 0, Math.PI * 2)
        ctx.fillStyle = 'rgba(59, 130, 246, 0.3)'
        ctx.fill()
      }

      // Node circle
      ctx.beginPath()
      ctx.arc(pos.x, pos.y, radius, 0, Math.PI * 2)
      ctx.fillStyle = color
      ctx.fill()
      ctx.strokeStyle = isSelected ? '#3b82f6' : '#1f2937'
      ctx.lineWidth = isSelected ? 2 : 1
      ctx.stroke()

      // Label
      if (viewTransform.scale > 0.5 || isSelected) {
        const label = n.short_name || n.long_name || `${n.node_num}`
        ctx.font = '10px ui-monospace, monospace'
        ctx.fillStyle = '#d1d5db'
        ctx.textAlign = 'center'
        ctx.fillText(label, pos.x, pos.y + radius + 12)
      }
    }

    ctx.restore()
  }, [nodes, links, viewTransform, selectedNode])

  // Pan & zoom handlers
  const handleMouseDown = (e: React.MouseEvent) => {
    dragRef.current = { dragging: true, lastX: e.clientX, lastY: e.clientY }
  }
  const handleMouseMove = (e: React.MouseEvent) => {
    if (!dragRef.current.dragging) return
    const dx = e.clientX - dragRef.current.lastX
    const dy = e.clientY - dragRef.current.lastY
    dragRef.current.lastX = e.clientX
    dragRef.current.lastY = e.clientY
    setViewTransform(v => ({ ...v, x: v.x + dx, y: v.y + dy }))
  }
  const handleMouseUp = () => { dragRef.current.dragging = false }
  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault()
    const delta = e.deltaY > 0 ? 0.9 : 1.1
    setViewTransform(v => ({
      ...v,
      scale: Math.min(5, Math.max(0.1, v.scale * delta)),
    }))
  }
  const handleCanvasClick = (e: React.MouseEvent) => {
    const canvas = canvasRef.current
    if (!canvas) return
    const rect = canvas.getBoundingClientRect()
    const mx = (e.clientX - rect.left - viewTransform.x) / viewTransform.scale
    const my = (e.clientY - rect.top - viewTransform.y) / viewTransform.scale

    let closest: TopoNode | null = null
    let closestDist = 20
    for (const n of nodes) {
      const pos = nodePositions.current.get(n.node_num)
      if (!pos) continue
      const dx = pos.x - mx
      const dy = pos.y - my
      const dist = Math.sqrt(dx * dx + dy * dy)
      if (dist < closestDist) {
        closest = n
        closestDist = dist
      }
    }
    setSelectedNode(closest)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <MapPin className="h-6 w-6 text-primary" />
          Topology Map
        </h1>
        <button
          onClick={fetchData}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          disabled={loading}
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-sm text-red-400 flex items-center gap-2">
          <AlertTriangle className="h-4 w-4" />
          {error}
        </div>
      )}

      {/* Stats bar */}
      {snapshot && (
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-8 gap-3">
          <StatCard label="Nodes" value={snapshot.node_count} />
          <StatCard label="Edges" value={snapshot.edge_count} />
          <StatCard label="Healthy" value={snapshot.healthy_nodes} color="text-green-400" />
          <StatCard label="Degraded" value={snapshot.degraded_nodes} color="text-yellow-400" />
          <StatCard label="Stale" value={snapshot.stale_nodes} color="text-gray-400" />
          <StatCard label="Isolated" value={snapshot.isolated_nodes} color="text-red-400" />
          <div className="col-span-2 rounded-lg border bg-card p-3">
            <div className="text-xs text-muted-foreground">Last Snapshot</div>
            <div className="text-sm font-mono">{snapshot.created_at ? new Date(snapshot.created_at).toLocaleTimeString() : 'N/A'}</div>
          </div>
        </div>
      )}

      <div className="flex gap-4">
        {/* Canvas map */}
        <div ref={containerRef} className="flex-1 rounded-lg border bg-card overflow-hidden" style={{ minHeight: '600px' }}>
          {nodes.length === 0 && !loading ? (
            <div className="flex items-center justify-center h-full text-muted-foreground">
              <div className="text-center space-y-2">
                <Radio className="h-12 w-12 mx-auto opacity-50" />
                <p>No topology data available yet.</p>
                <p className="text-xs">Nodes will appear as packets are ingested.</p>
              </div>
            </div>
          ) : (
            <canvas
              ref={canvasRef}
              className="w-full cursor-grab active:cursor-grabbing"
              style={{ height: '600px' }}
              onMouseDown={handleMouseDown}
              onMouseMove={handleMouseMove}
              onMouseUp={handleMouseUp}
              onMouseLeave={handleMouseUp}
              onWheel={handleWheel}
              onClick={handleCanvasClick}
            />
          )}
        </div>

        {/* Detail panel */}
        {selectedNode && (
          <div className="w-72 rounded-lg border bg-card p-4 space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="font-semibold text-lg">{selectedNode.short_name || selectedNode.long_name || `Node ${selectedNode.node_num}`}</h3>
              <button onClick={() => setSelectedNode(null)} className="text-muted-foreground hover:text-foreground text-xs">Close</button>
            </div>
            <div className="space-y-2 text-sm">
              <DetailRow label="Node Num" value={selectedNode.node_num} />
              <DetailRow label="Node ID" value={selectedNode.node_id || 'N/A'} />
              <DetailRow label="Health">
                <span
                  className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium"
                  style={{ backgroundColor: `${healthColors[selectedNode.health_state]}20`, color: healthColors[selectedNode.health_state] }}
                >
                  {selectedNode.health_state}
                </span>
              </DetailRow>
              <DetailRow label="Score" value={`${(selectedNode.health_score * 100).toFixed(0)}%`} />
              <DetailRow label="Trust" value={selectedNode.trust_class} />
              <DetailRow label="SNR" value={`${selectedNode.last_snr.toFixed(1)} dB`} />
              <DetailRow label="RSSI" value={`${selectedNode.last_rssi} dBm`} />
              {selectedNode.lat_redacted !== 0 && (
                <DetailRow label="Position" value={`${selectedNode.lat_redacted.toFixed(2)}, ${selectedNode.lon_redacted.toFixed(2)}`} />
              )}
              {selectedNode.stale && (
                <div className="rounded bg-yellow-500/10 border border-yellow-500/20 px-2 py-1 text-xs text-yellow-400">Stale</div>
              )}
              {selectedNode.quarantined && (
                <div className="rounded bg-red-500/10 border border-red-500/20 px-2 py-1 text-xs text-red-400">Quarantined</div>
              )}
            </div>
            {/* Connected links */}
            <div>
              <h4 className="text-xs font-medium text-muted-foreground mb-1 flex items-center gap-1"><Wifi className="h-3 w-3" /> Links</h4>
              <div className="space-y-1 max-h-40 overflow-y-auto">
                {links.filter(l => l.src_node_num === selectedNode.node_num || l.dst_node_num === selectedNode.node_num).slice(0, 20).map(l => {
                  const peer = l.src_node_num === selectedNode.node_num ? l.dst_node_num : l.src_node_num
                  const peerNode = nodes.find(n => n.node_num === peer)
                  return (
                    <div key={l.edge_id} className="flex items-center justify-between text-xs bg-muted/50 rounded px-2 py-1">
                      <span>{peerNode?.short_name || peer}</span>
                      <span className={l.stale ? 'text-gray-500' : 'text-blue-400'}>{(l.quality_score * 100).toFixed(0)}%</span>
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="flex flex-wrap gap-3 text-xs text-muted-foreground">
        <span className="flex items-center gap-1"><Layers className="h-3 w-3" /> Legend:</span>
        {Object.entries(healthColors).slice(0, 6).map(([state, color]) => (
          <span key={state} className="flex items-center gap-1">
            <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: color }} />
            {state}
          </span>
        ))}
      </div>
    </div>
  )
}

function StatCard({ label, value, color }: { label: string; value: number; color?: string }) {
  return (
    <div className="rounded-lg border bg-card p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`text-xl font-bold ${color || ''}`}>{value}</div>
    </div>
  )
}

function DetailRow({ label, value, children }: { label: string; value?: string | number; children?: React.ReactNode }) {
  return (
    <div className="flex justify-between items-center">
      <span className="text-muted-foreground">{label}</span>
      {children || <span className="font-mono">{value}</span>}
    </div>
  )
}
