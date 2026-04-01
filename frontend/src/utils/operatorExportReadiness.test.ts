import { describe, expect, it } from 'vitest'
import { operatorExportReadinessFromVersion } from './operatorExportReadiness'
import type { VersionResponse } from '@/types/api'

describe('operatorExportReadinessFromVersion', () => {
  it('flags unknown when version fetch failed', () => {
    const r = operatorExportReadinessFromVersion(undefined, 'network')
    expect(r.semantic).toBe('unknown_partial')
    expect(r.artifactStrength).toBe('weaker_until_runtime_checked')
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

  it('prefers operator_readiness from version when present', () => {
    const v = {
      version: 'test',
      operator_readiness: {
        semantic: 'degraded',
        summary: 'Backend canonical summary',
        artifact_strength: 'usable_degraded',
        blockers: [{ code: 'x', summary: 'y' }],
      },
      platform_posture: {
        evidence_export_delete: { export_enabled: true, delete_enabled: false, delete_scope: [] },
      },
    } as unknown as VersionResponse
    const r = operatorExportReadinessFromVersion(v, null)
    expect(r.source).toBe('operator_readiness')
    expect(r.summary).toBe('Backend canonical summary')
    expect(r.semantic).toBe('degraded')
    expect(r.artifactStrength).toBe('usable_but_degraded')
    expect(r.blockers).toHaveLength(1)
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
    expect(r.source).toBe('platform_posture_fallback')
  })
})
