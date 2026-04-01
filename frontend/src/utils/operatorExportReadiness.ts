/**
 * Canonical export / bundle readiness from GET /api/v1/version (platform_posture).
 * Single source for incident list, detail, and shift-start surfaces — no silent "clean" when policy is unknown.
 */
import type { VersionResponse } from '@/types/api'

export type OperatorReadinessSemantic =
  | 'available'
  | 'degraded'
  | 'gated'
  | 'unsupported'
  | 'unavailable'
  | 'unknown_partial'
  | 'sparse'
  | 'partial'
  | 'capability_limited'
  | 'policy_limited'

export type OperatorArtifactStrength = 'useful_now' | 'usable_degraded' | 'weaker_check_runtime' | 'blocked'

export interface OperatorExportReadiness {
  semantic: OperatorReadinessSemantic
  summary: string
  artifactStrength: OperatorArtifactStrength
}

export function operatorExportReadinessFromVersion(
  v: VersionResponse | null | undefined,
  versionError: string | null | undefined,
): OperatorExportReadiness {
  const exp = v?.platform_posture?.evidence_export_delete
  if (versionError && !exp) {
    return {
      semantic: 'unknown_partial',
      summary: `Export policy not loaded (${versionError}) — confirm Settings before proofpack or escalation.`,
      artifactStrength: 'weaker_check_runtime',
    }
  }
  if (exp?.export_enabled === false) {
    return {
      semantic: 'policy_limited',
      summary:
        'Instance policy disables evidence export — proofpack and escalation bundles are blocked; use plain handoff where allowed.',
      artifactStrength: 'blocked',
    }
  }
  if (exp?.export_enabled === true) {
    return {
      semantic: 'available',
      summary:
        'Evidence export allowed by policy — proofpack still reflects assembly-time gaps; review completeness before external handoff.',
      artifactStrength: 'useful_now',
    }
  }
  return {
    semantic: 'unknown_partial',
    summary: 'Export policy unknown — confirm Settings/runtime before relying on proofpack.',
    artifactStrength: 'weaker_check_runtime',
  }
}
