/**
 * Presentation-only helpers for dense operator surfaces (IDs, payloads).
 * Does not interpret domain semantics beyond string shaping.
 */

const ELLIPSIS = '…'

export function truncateMiddle(text: string, maxLen: number): string {
  if (maxLen < 8) return text.length <= maxLen ? text : text.slice(0, maxLen)
  if (text.length <= maxLen) return text
  const head = Math.ceil((maxLen - 1) / 2)
  const tail = Math.floor((maxLen - 1) / 2)
  return `${text.slice(0, head)}${ELLIPSIS}${text.slice(text.length - tail)}`
}

export function extractTransportsFromStatusJson(data: unknown): Array<{
  name?: string
  effective_state?: string
  detail?: string
}> {
  if (!data || typeof data !== 'object') return []
  const o = data as Record<string, unknown>
  if (Array.isArray(o.transports)) {
    return o.transports as Array<{ name?: string; effective_state?: string; detail?: string }>
  }
  const inner = o.status
  if (inner && typeof inner === 'object') {
    const st = inner as Record<string, unknown>
    if (Array.isArray(st.transports)) {
      return st.transports as Array<{ name?: string; effective_state?: string; detail?: string }>
    }
  }
  return []
}
