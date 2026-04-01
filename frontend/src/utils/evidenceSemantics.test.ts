import { describe, it, expect } from 'vitest'
import {
  evidenceStrengthLabel,
  wirelessEvidencePostureLabel,
  topologyLinkTruthSummary,
  runbookStrengthOperatorLabel,
} from './evidenceSemantics'

describe('evidenceSemantics', () => {
  it('labels evidence strength without overstating', () => {
    expect(evidenceStrengthLabel('sparse')).toMatch(/Sparse/)
    expect(evidenceStrengthLabel('strong')).toMatch(/Strong evidence/)
  })

  it('maps wireless evidence posture to operator language', () => {
    expect(wirelessEvidencePostureLabel('imported').label).toMatch(/Imported/)
    expect(wirelessEvidencePostureLabel('unsupported').label).toMatch(/Unsupported/)
  })

  it('describes topology links as observed vs inferred', () => {
    expect(topologyLinkTruthSummary({ observed: true, stale: false, relay_dependent: false })).toContain('Observed')
    expect(topologyLinkTruthSummary({ observed: false, stale: true, relay_dependent: true })).toContain('Inferred')
  })

  it('frames runbook strength without causal claims', () => {
    expect(runbookStrengthOperatorLabel('historically_proven')).toMatch(/not causal/)
    expect(runbookStrengthOperatorLabel('unsupported')).toMatch(/manual judgment/)
  })
})
