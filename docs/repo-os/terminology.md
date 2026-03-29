# MEL Canonical Terminology & Language Rules

This is MEL's canonical language guide for operator UI, API docs, runbooks, README text, architecture docs, and contributor guidance.

Use this file to keep wording simple, precise, and truthful.

## 1) Core language policy

### Default rule
Use simple operator-facing terms when they preserve truth:
- mesh
- node
- link
- signal
- frequency
- LoRa
- Wi-Fi
- Bluetooth
- action
- evidence
- proofpack
- incident
- health
- degraded

### Precision rule
Use deeper engineering terms only when needed to prevent implementation ambiguity, support-truth drift, security/control mistakes, or verification ambiguity.

Examples: `ingesting`, `connected_no_ingest`, `pending_approval`, `approval_expired`, `dead_letter`.

### Mixed audience rule
When both are useful, lead with the simple term and include the precise term secondarily.

Example:
- Preferred: "Live ingest evidence (`ingesting`) is present."
- Avoid: "Transport operability state is ingest-positive."

## 2) Scientific observation language rules

Always separate:
- **Observed**: directly measured/persisted evidence.
- **Inferred**: deterministic derivation from observed evidence.
- **Estimated**: bounded estimate with declared assumptions.
- **Unknown**: insufficient evidence for safe classification.

Never collapse inferred/estimated/unknown into observed.
Never use fake precision when measurement resolution is unavailable.

## 3) MEL truth-state vocabulary (canonical)

| Term | Canonical meaning | Use notes |
|---|---|---|
| **Live** | Current ingest evidence exists within freshness window. | Never claim from config presence alone. |
| **Stale** | Last ingest evidence is older than freshness window. | Must be explicit in UI/API copy. |
| **Historical** | Persisted past records; not current runtime proof. | Use for replay/history views. |
| **Imported / Offline** | Externally sourced evidence brought into MEL. | Never present as live fleet truth by default. |
| **Partial** | Some expected evidence is missing for known reasons. | Keep machine-visible in response fields when possible. |
| **Degraded** | Known quality gap (disconnects, dead letters, scope gaps, backlog). | Must be explicit and actionable. |
| **Unsupported** | Feature/path not implemented in MEL. | Never imply partial support. |
| **Unknown** | Not enough evidence to classify state safely. | Prefer over optimistic wording. |

## 4) Transport and mixed-channel support language rules

Use this wording consistently in docs/UI/API references:
- **Supported ingest transports:** direct serial/TCP, MQTT ingest.
- **Unsupported ingest transports:** BLE ingest, HTTP ingest.
- **Out-of-scope claim:** MEL is not a mesh routing/transmit stack and does not prove RF propagation success by itself.

For mixed-channel context, prefer explicit fields/phrasing:
- channel: LoRa / Bluetooth / Wi-Fi / mixed / unknown;
- path: direct / relayed / bridged / local-only / backhaul-assisted;
- support: supported / unsupported / sparse / degraded / unknown.

## 5) Control and action language rules

Use lifecycle-specific terms. Do not collapse state boundaries.

- **Submitted** != **Approved** != **Dispatched** != **Executed** != **Audited**.
- Use "recommendation" for advisory-only outputs with no actuator.
- Use "action" for lifecycle-tracked control records.
- Use "action history" for prior actions and outcomes.
- Use "action-outcome memory" when prior actions are used for future guidance.

## 6) Evidence and proofpack language rules

- **Evidence**: persisted observations, diagnostics, and audit records.
- **Evidence gap**: explicit missing data needed to raise confidence.
- **Proofpack**: curated export bundle that captures evidence context and provenance boundaries.
- **Snapshot completeness**: complete / partial / unavailable.

Never use certainty language stronger than available evidence.

## 7) Local inference language rules

- "Assistive" means optional, non-canonical output.
- Do not phrase LLM/local model output as system truth.
- If inference runtime is unavailable, use explicit fallback language and keep base truth contracts intact.

## 8) Preferred vs discouraged wording

| Prefer | Avoid | Why |
|---|---|---|
| "node" | "endpoint entity" | Plain operator wording. |
| "link" | "inter-node transport adjacency" | Same meaning, clearer. |
| "degraded" | "suboptimal but healthy" | Avoid mixed/confusing status. |
| "unsupported" | "planned / coming soon" (without evidence) | Prevents scope overclaiming. |
| "imported/offline evidence" | "federated live data" | Preserve trust boundary. |
| "action pending approval" | "queued for execution" (if approval not done) | Prevent control-path ambiguity. |
| "no live evidence yet" | "live soon" | Keep claims bounded. |

## 9) Surface-specific style guidance

### Operator surfaces (UI, quickstarts, runbooks)
- Lead with operational meaning and next step.
- Keep terms compact and scannable.
- Make degraded/unknown conditions explicit.

### Engineering surfaces (architecture, contributor docs)
- Keep exact contract/state names where implementation depends on them.
- Add operator-facing translation for dense sections where useful.

## 10) Canonical alignment checklist

Before merging documentation changes:
1. Does wording overclaim support, precision, or runtime truth?
2. Are live/stale/historical/imported/partial/degraded/unknown states clearly distinguished?
3. Are observed/inferred/estimated/unknown distinctions explicit where needed?
4. Are control lifecycle states uncollapsed?
5. Is operator-facing wording simple where precision is not lost?
6. Does terminology align with this file across README/docs/runbooks/UI/API descriptions?
