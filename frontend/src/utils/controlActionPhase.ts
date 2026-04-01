import type { ControlActionRecord } from '@/types/api'

export function controlActionExecPhase(a: ControlActionRecord): {
  label: string
  variant: 'warning' | 'info' | 'success' | 'critical' | 'secondary'
} {
  const ls = (a.lifecycle_state || '').toLowerCase()
  const res = (a.result || '').toLowerCase()
  if (ls === 'pending_approval') return { label: 'Awaiting approval', variant: 'warning' }
  if (ls === 'pending' && res === 'approved') return { label: 'Approved, queued', variant: 'info' }
  if (ls === 'running') return { label: 'Executing', variant: 'info' }
  if (ls === 'completed') {
    if (res === 'rejected') return { label: 'Rejected', variant: 'critical' }
    if (res.includes('failed')) return { label: 'Failed', variant: 'critical' }
    return { label: 'Finished', variant: 'success' }
  }
  return { label: a.lifecycle_state || 'Unknown', variant: 'secondary' }
}
