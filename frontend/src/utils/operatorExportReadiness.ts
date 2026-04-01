/**
 * Canonical export / bundle / support artifact readiness.
 * Prefers GET /api/v1/version `operator_readiness` when present; falls back to platform_posture only.
 */
import type { OperatorReadinessDTO, VersionResponse } from '@/types/api'

export type OperatorReadinessSemantic = OperatorReadinessDTO['semantic']

export type OperatorArtifactStrength =
  | 'useful_now'
  | 'usable_but_degraded'
  | 'weaker_until_runtime_checked'
  | 'blocked'

export interface OperatorExportReadiness {
  semantic: OperatorReadinessSemantic
  summary: string
  artifactStrength: OperatorArtifactStrength
  blockers: Array<{ code: string; summary: string }>
  source: 'operator_readiness' | 'platform_posture_fallback' | 'version_error_fallback'
  evidenceBasis: string[]
  generatedFromNote?: string
}

function mapArtifactStrength(dto: string | undefined): OperatorArtifactStrength {
  switch (dto) {
    case 'useful_now':
      return 'useful_now'
    case 'usable_degraded':
      return 'usable_but_degraded'
    case 'weaker_until_runtime_checked':
      return 'weaker_until_runtime_checked'
    case 'blocked':
      return 'blocked'
    default:
      return 'weaker_until_runtime_checked'
  }
}

export function operatorExportReadinessFromVersion(
  v: VersionResponse | null | undefined,
  versionError: string | null | undefined,
): OperatorExportReadiness {
  const or = v?.operator_readiness
  if (or && typeof or.summary === 'string') {
    return {
      semantic: or.semantic,
      summary: or.summary,
      artifactStrength: mapArtifactStrength(or.artifact_strength),
      blockers: or.blockers ?? [],
      source: 'operator_readiness',
      evidenceBasis: or.evidence_basis ?? [],
      generatedFromNote: or.generated_from_note,
    }
  }

  const exp = v?.platform_posture?.evidence_export_delete
  if (versionError && !exp) {
    return {
      semantic: 'unknown_partial',
      summary: `Export policy not loaded (${versionError}) — confirm Settings before proofpack or escalation.`,
      artifactStrength: 'weaker_until_runtime_checked',
      blockers: [],
      source: 'version_error_fallback',
      evidenceBasis: [],
    }
  }
  if (exp?.export_enabled === false) {
    return {
      semantic: 'policy_limited',
      summary:
        'Instance policy disables evidence export — proofpack and escalation bundles are blocked; use plain handoff where allowed.',
      artifactStrength: 'blocked',
      blockers: [{ code: 'export_disabled_by_policy', summary: 'platform.retention.allow_export=false' }],
      source: 'platform_posture_fallback',
      evidenceBasis: ['platform_posture.evidence_export_delete'],
    }
  }
  if (exp?.export_enabled === true) {
    const redact = v?.platform_posture?.export_redaction_enabled === true
    return {
      semantic: redact ? 'degraded' : 'available',
      summary: redact
        ? 'Evidence export allowed by policy but privacy redaction is on — artifacts are weaker than full-fidelity; review before external handoff.'
        : 'Evidence export allowed by policy — proofpack still reflects assembly-time gaps; review completeness before external handoff.',
      artifactStrength: redact ? 'usable_but_degraded' : 'useful_now',
      blockers: redact
        ? [{ code: 'export_redaction_enabled', summary: 'platform.privacy.redact_exports=true' }]
        : [],
      source: 'platform_posture_fallback',
      evidenceBasis: ['platform_posture.evidence_export_delete', 'platform_posture.export_redaction_enabled'],
    }
  }
  return {
    semantic: 'unknown_partial',
    summary: 'Export policy unknown — confirm Settings/runtime before relying on proofpack.',
    artifactStrength: 'weaker_until_runtime_checked',
    blockers: [],
    source: 'platform_posture_fallback',
    evidenceBasis: [],
  }
}
