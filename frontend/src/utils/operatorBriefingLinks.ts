import type { OperatorBriefingPriority } from '@/types/api'

function encodeTransportHash(name: string) {
  return `#mel-transport-${name.replace(/[^a-zA-Z0-9_-]/g, '_')}`
}

/** Deep link for a briefing priority row — bounded by resource_kind + known metadata keys. */
export function hrefForBriefingPriority(p: OperatorBriefingPriority): string {
  const meta = p.metadata ?? {}
  const transportName =
    typeof meta.resource_id === 'string' && meta.resource_id.trim() !== ''
      ? meta.resource_id
      : typeof meta.affected_transport === 'string' && meta.affected_transport.trim() !== ''
        ? meta.affected_transport
        : ''

  if (p.resource_kind === 'incident' && p.id) {
    return `/incidents/${encodeURIComponent(p.id)}`
  }
  if ((p.resource_kind === 'transport' || p.resource_kind === 'diagnostic') && transportName) {
    return `/status${encodeTransportHash(transportName)}`
  }
  if (p.resource_kind === 'control') {
    return '/control-actions'
  }
  if (p.resource_kind === 'diagnostic') {
    return '/diagnostics'
  }
  return '/diagnostics'
}
