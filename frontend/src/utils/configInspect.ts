import type { ConfigInspectResponse, ConfigInspectValues, ConfigSafetyViolation } from '@/types/api'

function isRecord(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === 'object' && !Array.isArray(v)
}

function parseSafetyViolation(raw: unknown): ConfigSafetyViolation | null {
  if (!isRecord(raw)) return null
  const field = raw.field
  const issue = raw.issue
  const current = raw.current
  const safe = raw.safe
  if (typeof field !== 'string' || typeof issue !== 'string' || typeof current !== 'string' || typeof safe !== 'string') {
    return null
  }
  return { field, issue, current, safe }
}

function parseConfigInspectValues(raw: unknown): ConfigInspectValues | undefined {
  if (!isRecord(raw)) return undefined

  const bind = isRecord(raw.bind)
    ? {
        api: typeof raw.bind.api === 'string' ? raw.bind.api : undefined,
        metrics: typeof raw.bind.metrics === 'string' ? raw.bind.metrics : undefined,
      }
    : undefined

  const auth = isRecord(raw.auth)
    ? {
        enabled: typeof raw.auth.enabled === 'boolean' ? raw.auth.enabled : undefined,
        ui_user: typeof raw.auth.ui_user === 'string' ? raw.auth.ui_user : undefined,
      }
    : undefined

  const storage = isRecord(raw.storage)
    ? {
        database_path: typeof raw.storage.database_path === 'string' ? raw.storage.database_path : undefined,
      }
    : undefined

  const privacy = isRecord(raw.privacy)
    ? {
        redact_exports: typeof raw.privacy.redact_exports === 'boolean' ? raw.privacy.redact_exports : undefined,
        map_reporting_allowed:
          typeof raw.privacy.map_reporting_allowed === 'boolean' ? raw.privacy.map_reporting_allowed : undefined,
      }
    : undefined

  const features = isRecord(raw.features)
    ? {
        google_maps_in_topology_ui:
          typeof raw.features.google_maps_in_topology_ui === 'boolean' ? raw.features.google_maps_in_topology_ui : undefined,
        google_maps_api_key_env:
          typeof raw.features.google_maps_api_key_env === 'string' ? raw.features.google_maps_api_key_env : undefined,
        metrics: typeof raw.features.metrics === 'boolean' ? raw.features.metrics : undefined,
      }
    : undefined

  return { bind, auth, storage, privacy, features }
}

export function parseConfigInspectResponse(raw: unknown): ConfigInspectResponse | null {
  if (!isRecord(raw)) return null

  const violationsRaw = raw.violations
  const violations: ConfigSafetyViolation[] | undefined = Array.isArray(violationsRaw)
    ? violationsRaw.map(parseSafetyViolation).filter((x): x is ConfigSafetyViolation => x !== null)
    : undefined

  return {
    fingerprint: typeof raw.fingerprint === 'string' ? raw.fingerprint : undefined,
    canonical_fingerprint: typeof raw.canonical_fingerprint === 'string' ? raw.canonical_fingerprint : undefined,
    values: parseConfigInspectValues(raw.values),
    violations,
  }
}
