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
    const d = new Date(timestamp);
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