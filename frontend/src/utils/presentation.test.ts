import { describe, it, expect } from 'vitest'
import { truncateMiddle, extractTransportsFromStatusJson } from './presentation'

describe('truncateMiddle', () => {
  it('returns short strings unchanged', () => {
    expect(truncateMiddle('abc', 10)).toBe('abc')
  })

  it('truncates long strings in the middle', () => {
    const s = 'abcdefghijklmnopqrstuvwxyz'
    const out = truncateMiddle(s, 12)
    expect(out.length).toBeLessThanOrEqual(12)
    expect(out).toContain('…')
    expect(out.startsWith('abcd')).toBe(true)
    expect(out.endsWith('wxyz')).toBe(true)
  })
})

describe('extractTransportsFromStatusJson', () => {
  it('reads top-level transports', () => {
    expect(
      extractTransportsFromStatusJson({
        transports: [{ name: 'a', effective_state: 'connected' }],
      }),
    ).toEqual([{ name: 'a', effective_state: 'connected' }])
  })

  it('reads nested status.transports', () => {
    expect(
      extractTransportsFromStatusJson({
        status: { transports: [{ name: 'b', effective_state: 'disconnected' }] },
      }),
    ).toEqual([{ name: 'b', effective_state: 'disconnected' }])
  })

  it('returns empty for invalid input', () => {
    expect(extractTransportsFromStatusJson(null)).toEqual([])
    expect(extractTransportsFromStatusJson({})).toEqual([])
  })
})
