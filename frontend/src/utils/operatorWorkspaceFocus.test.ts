import { describe, expect, it, beforeEach } from 'vitest'
import {
  clearOperatorWorkspaceFocus,
  readOperatorWorkspaceFocus,
  writeOperatorWorkspaceFocus,
} from './operatorWorkspaceFocus'

describe('operatorWorkspaceFocus', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('round-trips focus', () => {
    writeOperatorWorkspaceFocus({
      incidentId: 'inc-abc',
      incidentTitle: 'Test',
      savedAt: '2026-01-01T00:00:00Z',
    })
    const r = readOperatorWorkspaceFocus()
    expect(r?.incidentId).toBe('inc-abc')
    expect(r?.incidentTitle).toBe('Test')
  })

  it('clearOperatorWorkspaceFocus removes key', () => {
    writeOperatorWorkspaceFocus({
      incidentId: 'x',
      savedAt: '2026-01-01T00:00:00Z',
    })
    clearOperatorWorkspaceFocus()
    expect(readOperatorWorkspaceFocus()).toBeNull()
  })
})
