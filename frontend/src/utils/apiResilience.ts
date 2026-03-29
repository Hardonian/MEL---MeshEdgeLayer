/**
 * apiResilience.ts
 * 
 * Edge-safe contract handling for operator consoles. 
 * Defends against missing fields, nullable timestamps, and unknown enum values
 * without hiding the fact that data is missing.
 */

export const safeArray = <T>(arr: T[] | null | undefined): T[] => {
  return Array.isArray(arr) ? arr : [];
};

/** GET /api/v1/diagnostics returns { findings: DiagnosticFinding[] } */
export interface DiagnosticsApiFinding {
  code: string
  severity: string
  component: string
  title: string
  explanation: string
  likely_causes?: string[]
  recommended_steps?: string[]
  evidence?: Record<string, unknown>
  can_auto_recover?: boolean
  operator_action_required?: boolean
  affected_transport?: string
  generated_at?: string
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === 'object' && !Array.isArray(v)
}

export function parseDiagnosticsFindingsFromApi(raw: unknown): DiagnosticsApiFinding[] {
  if (!isRecord(raw)) return []
  const findings = raw.findings
  if (!Array.isArray(findings)) return []
  const out: DiagnosticsApiFinding[] = []
  for (const item of findings) {
    if (!isRecord(item)) continue
    const code = item.code
    const severity = item.severity
    const component = item.component
    const title = item.title
    const explanation = item.explanation
    if (
      typeof code !== 'string' ||
      typeof severity !== 'string' ||
      typeof component !== 'string' ||
      typeof title !== 'string' ||
      typeof explanation !== 'string'
    ) {
      continue
    }
    const f: DiagnosticsApiFinding = {
      code,
      severity,
      component,
      title,
      explanation,
    }
    if (Array.isArray(item.likely_causes)) {
      f.likely_causes = item.likely_causes.filter((x): x is string => typeof x === 'string')
    }
    if (Array.isArray(item.recommended_steps)) {
      f.recommended_steps = item.recommended_steps.filter((x): x is string => typeof x === 'string')
    }
    if (isRecord(item.evidence)) {
      f.evidence = item.evidence as Record<string, unknown>
    }
    if (typeof item.can_auto_recover === 'boolean') f.can_auto_recover = item.can_auto_recover
    if (typeof item.operator_action_required === 'boolean')
      f.operator_action_required = item.operator_action_required
    if (typeof item.affected_transport === 'string') f.affected_transport = item.affected_transport
    if (typeof item.generated_at === 'string') f.generated_at = item.generated_at
    out.push(f)
  }
  return out
}

export const normalizeEnum = <T extends string>(
  value: string | null | undefined, 
  validValues: readonly T[], 
  fallback: T
): T => {
  if (!value) return fallback;
  return validValues.includes(value as T) ? (value as T) : fallback;
};

export type TimeState = 'fresh' | 'stale' | 'never_seen' | 'invalid';

export const evaluateTimeState = (timestamp: string | null | undefined, thresholdMs = 300000): TimeState => {
  if (!timestamp) return 'never_seen';
  const time = new Date(timestamp).getTime();
  if (isNaN(time)) return 'invalid';
  return Date.now() - time > thresholdMs ? 'stale' : 'fresh';
};

export const formatOperatorTime = (timestamp: string | null | undefined): string => {
  const state = evaluateTimeState(timestamp);
  
  switch (state) {
    case 'never_seen': return 'Never (Omitted)';
    case 'invalid': return `Invalid Timestamp (${timestamp})`;
  }
  
  try {
    const d = new Date(timestamp as string);
    return isNaN(d.getTime()) ? 'Invalid Date' : d.toISOString();
  } catch {
    return 'Invalid Date';
  }
};

export const safeDenialReason = (reason: string | null | undefined): string => {
  if (!reason) return 'reason_omitted';
  const knownReasons = ['mode', 'policy', 'cooldown', 'budget', 'low_confidence', 'transient', 'missing_actuator', 'irreversible', 'conflict', 'override'];
  return knownReasons.includes(reason) ? reason : `unknown_reason_code:${reason}`;
};