# MEL Architecture Decision Records

## ADR-001: Explicit Degraded States

**Status**: Accepted

### Context
Most dashboards show "live" when network is questionable. Operators need honest state.

### Decision
MEL defaults to "unknown" unless proven. Always show evidence level.

### Consequences
- Pro: Trustworthy UI
- Pro: Operators not misled
- Con: Looks "worse" than other tools

## ADR-002: Local-First

**Status**: Accepted

### Context
Field operations may lack connectivity.

### Decision
All operations work offline. Sync optional.

### Consequences
- Pro: Works in field
- Pro: Zero network dependency
- Con: More complex sync

## ADR-003: Break-Glass with SoD

**Status**: Accepted

### Context
Emergency actions need fast execution, but controls matter.

### Decision
Break-glass requires explicit `--i-understand-sod` flag. All actions audit logged.

### Consequences
- Pro: Fast emergency response
- Pro: Compliance
- Con: Requires awareness

## ADR-004: Evidence Over Inference

**Status**: Accepted

### Context
AI is useful but shouldn't generate facts.

### Decision
AI assists but never replaces evidence. All state derives from persisted data.

### Consequences
- Pro: Truth-first
- Pro: No hallucination risk
- Con: Less automation
