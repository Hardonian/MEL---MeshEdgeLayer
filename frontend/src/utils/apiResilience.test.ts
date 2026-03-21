import { describe, it, expect, vi } from 'vitest';
import { safeArray, normalizeEnum, evaluateTimeState, safeDenialReason } from './apiResilience';

describe('apiResilience', () => {
  describe('safeArray', () => {
    it('returns array for valid array', () => {
      expect(safeArray([1, 2, 3])).toEqual([1, 2, 3]);
    });
    it('returns empty array for null/undefined/non-array', () => {
      expect(safeArray(null)).toEqual([]);
      expect(safeArray(undefined)).toEqual([]);
      expect(safeArray('not-an-array' as any)).toEqual([]);
    });
  });

  describe('normalizeEnum', () => {
    it('returns valid value', () => {
      expect(normalizeEnum('active', ['active', 'inactive'], 'inactive')).toBe('active');
    });
    it('falls back on invalid value', () => {
      expect(normalizeEnum('broken', ['active', 'inactive'], 'inactive')).toBe('inactive');
    });
    it('falls back on null/undefined', () => {
      expect(normalizeEnum(null, ['active', 'inactive'], 'inactive')).toBe('inactive');
      expect(normalizeEnum(undefined, ['active', 'inactive'], 'inactive')).toBe('inactive');
    });
  });

  describe('evaluateTimeState', () => {
    it('returns never_seen for null/undefined', () => {
      expect(evaluateTimeState(null)).toBe('never_seen');
      expect(evaluateTimeState(undefined)).toBe('never_seen');
    });
    it('returns invalid for malformed dates', () => {
      expect(evaluateTimeState('not-a-date')).toBe('invalid');
    });
    
    it('returns fresh within threshold', () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-03-20T21:00:00.000Z'));
      
      const ts = new Date('2026-03-20T20:59:00.000Z').toISOString();
      expect(evaluateTimeState(ts, 5 * 60 * 1000)).toBe('fresh');
      
      vi.useRealTimers();
    });

    it('returns stale outside threshold', () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-03-20T21:00:00.000Z'));
      
      const ts = new Date('2026-03-20T20:50:00.000Z').toISOString();
      expect(evaluateTimeState(ts, 5 * 60 * 1000)).toBe('stale');
      
      vi.useRealTimers();
    });
  });

  describe('safeDenialReason', () => {
    it('returns reason_omitted on null', () => {
      expect(safeDenialReason(null)).toBe('reason_omitted');
    });
    it('passes through known reasons', () => {
      expect(safeDenialReason('cooldown')).toBe('cooldown');
    });
    it('formats unknown reasons safely', () => {
      expect(safeDenialReason('aliens')).toBe('unknown_reason_code:aliens');
    });
  });
});
