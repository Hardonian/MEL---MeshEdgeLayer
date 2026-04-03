import { describe, expect, it } from 'vitest'
import { parseConfigInspectResponse } from './configInspect'

describe('parseConfigInspectResponse', () => {
  it('parses known nested posture fields', () => {
    const parsed = parseConfigInspectResponse({
      fingerprint: 'fp',
      canonical_fingerprint: 'canon',
      values: {
        bind: { api: '127.0.0.1:8080', metrics: '' },
        auth: { enabled: true, ui_user: 'admin' },
        storage: { database_path: './data/mel.db' },
        privacy: { redact_exports: true, map_reporting_allowed: false },
        features: { google_maps_in_topology_ui: false, google_maps_api_key_env: '', metrics: true },
      },
      violations: [{ field: 'x', issue: 'i', current: 'c', safe: 's' }],
    })

    expect(parsed?.values?.bind?.api).toBe('127.0.0.1:8080')
    expect(parsed?.values?.auth?.enabled).toBe(true)
    expect(parsed?.violations?.length).toBe(1)
  })

  it('treats wrong nested types as unavailable instead of coercing', () => {
    const parsed = parseConfigInspectResponse({
      values: {
        bind: 'bad',
        auth: { enabled: 'true' },
      },
      violations: [{ field: 'missing-safe' }],
    })

    expect(parsed?.values?.bind).toBeUndefined()
    expect(parsed?.values?.auth?.enabled).toBeUndefined()
    expect(parsed?.violations).toEqual([])
  })

  it('returns null for non-object payload', () => {
    expect(parseConfigInspectResponse('bad')).toBeNull()
  })
})
