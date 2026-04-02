import { describe, expect, it } from 'vitest'
import type { ControlActionRecord } from '@/types/api'
import { controlActionExecPhase } from './controlActionPhase'

describe('controlActionExecPhase', () => {
  it('labels pending without approved result as pre-approval queue', () => {
    const a = { lifecycle_state: 'pending', result: '' } as ControlActionRecord
    expect(controlActionExecPhase(a).label).toBe('Queued (pre-approval)')
  })

  it('maps failed lifecycle to Failed', () => {
    const a = { lifecycle_state: 'failed' } as ControlActionRecord
    expect(controlActionExecPhase(a).label).toBe('Failed')
  })
})
