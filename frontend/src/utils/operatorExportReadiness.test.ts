import { describe, expect, it } from 'vitest'
import { operatorExportReadinessFromVersion } from './operatorExportReadiness'
import type { VersionResponse } from '@/types/api'

describe('operatorExportReadinessFromVersion', () => {
  it('flags unknown when version fetch failed', () => {
    const r = operatorExportReadinessFromVersion(undefined, 'network')
    expect(r.semantic).toBe('unknown_partial')
    expect(r.artifactStrength).toBe('weaker_check_runtime')
  })

  it('flags policy block when export disabled', () => {
    const v = {
      version: 'test',
      platform_posture: {
        evidence_export_delete: { export_enabled: false, delete_enabled: false, delete_scope: [] },
      },
    } as unknown as VersionResponse
    const r = operatorExportReadinessFromVersion(v, null)
    expect(r.semantic).toBe('policy_limited')
    expect(r.artifactStrength).toBe('blocked')
  })

  it('available when export enabled', () => {
    const v = {
      version: 'test',
      platform_posture: {
        evidence_export_delete: { export_enabled: true, delete_enabled: false, delete_scope: [] },
      },
    } as unknown as VersionResponse
    const r = operatorExportReadinessFromVersion(v, null)
    expect(r.semantic).toBe('available')
    expect(r.artifactStrength).toBe('useful_now')
  })
})
