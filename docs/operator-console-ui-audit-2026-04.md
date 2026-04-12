# Operator Console UI Audit — 2026-04-12

Scope: `frontend/src/pages/*.tsx` route surfaces used by the MEL operator console.

## Audit method
- Reviewed each routed page for hierarchy, scannability, state communication, keyboard affordances, and truth boundary wording.
- Prioritized upgrades that improve all pages through shared components first, then improved high-traffic surfaces (`Dashboard`, `Status`).

## Page-by-page findings

### Dashboard (`/`)
- **Current strength**: rich evidence + workflow context exists.
- **Problem**: dense above-the-fold text and weak first-glance state anchors.
- **Upgrade applied**: header now includes explicit status chips for transport posture, open incidents, and evidence completeness.

### Status (`/status`)
- **Current strength**: detailed per-transport cards and truthful caveats.
- **Problem**: top summary did not clearly separate stable vs degraded vs unavailable at a glance.
- **Upgrade applied**: header status chips + stronger stat-card semantic variants (`partial`, `unavailable`) for clearer state differentiation.

### Nodes (`/nodes`)
- **Finding**: mostly strong, but dense metric blocks rely on repeated prose.
- **Next pass**: add compact evidence legend and sticky filters for small screens.

### Topology (`/topology`)
- **Finding**: good control surface; still text-heavy in legend/explanations.
- **Next pass**: convert legend copy into concise grouped evidence modules.

### Planning (`/planning`)
- **Finding**: deep insights but high cognitive load from long paragraphs.
- **Next pass**: convert explanation-heavy areas into progressive disclosure rows.

### Messages (`/messages`)
- **Finding**: clear truth boundaries; list readability acceptable.
- **Next pass**: stronger empty/degraded list treatment and quick filters.

### Dead Letters (`/dead-letters`)
- **Finding**: strong deterministic semantics.
- **Next pass**: add compact error taxonomy chips to reduce scanning time.

### Incidents (`/incidents`)
- **Finding**: dense but operationally rich.
- **Next pass**: stronger visual grouping by urgency + evidence sufficiency.

### Incident Detail (`/incidents/:id`)
- **Finding**: feature-rich with many sections and long explanatory copy.
- **Next pass**: section-level scannability pass (collapsible rationale clusters).

### Control Actions (`/control-actions`)
- **Finding**: lifecycle truth is clear.
- **Next pass**: tighter keyboard workflow hints for queue triage.

### Events (`/events`)
- **Finding**: audit semantics are strong.
- **Next pass**: severity timeline anchors and faster row parsing.

### Diagnostics (`/diagnostics`)
- **Finding**: good truth messaging.
- **Next pass**: stronger differentiator between unavailable API vs degraded runtime checks.

### Privacy (`/privacy`)
- **Finding**: severity model is understandable.
- **Next pass**: compress repetitive explanatory copy into semantic legends.

### Recommendations (`/recommendations`)
- **Finding**: assistive/non-canonical framing is good.
- **Next pass**: sharper actionable-vs-informational grouping.

### Operational Review (`/operational-review`)
- **Finding**: high-value operational memory surface.
- **Next pass**: improve card hierarchy for shift handoff readability.

### Settings (`/settings`)
- **Finding**: useful reference density; visually flat.
- **Next pass**: improve config section wayfinding with grouped anchors.

## Highest-leverage upgrades selected in this pass
1. **Shared PageHeader redesign** for hierarchy, grouping, and status chips across pages.
2. **Semantic StatCard variants** for `partial`, `stale`, and `unavailable` differentiation.
3. **Global focus-visible accessibility baseline** for keyboard operators.
4. **Dashboard + Status adoption** of new high-signal header chips for top-level scannability.

## Truth and performance guardrails
- No fake metrics or inferred runtime claims were added.
- Existing data sources and query cadence were preserved.
- Enhancements are presentational and state-labeling only.
