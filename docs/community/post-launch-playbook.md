# Post-launch playbook (first weeks)

MEL is evidence-first and anti-theatre. This playbook keeps public intake fast without weakening truth boundaries.

## Signals to watch

- **Truth drift:** UI/docs language implies certainty the evidence does not support.
- **Control safety drift:** lifecycle separation becomes ambiguous.
- **Setup friction:** first-run commands fail on clean environments.
- **Transport confusion:** unsupported surfaces read as “almost supported.”

## Issue classes (route quickly)

| Class | Intake template | First maintainer response |
| --- | --- | --- |
| Setup / bootstrap | `.github/ISSUE_TEMPLATE/install_setup.md` | Reproduce from first failing command + `mel doctor` output. |
| Bug / truth mismatch | `.github/ISSUE_TEMPLATE/bug_report.md` | Compare claim vs persisted API/CLI evidence. |
| Docs drift | `.github/ISSUE_TEMPLATE/docs.md` | Patch wording/links or narrow claim immediately. |
| Field observation | `.github/ISSUE_TEMPLATE/field_report.md` | Confirm privacy-safe details and label evidence posture. |
| Hardware compatibility | `.github/ISSUE_TEMPLATE/hardware_compatibility.md` | Treat as anecdotal evidence, not certification. |
| Security | GitHub advisories + `SECURITY.md` | Keep details private until triaged. |

Label policy: apply `needs:triage` + one `kind:*` label on intake, then one `area:*` label if needed.

## Maintainer rhythm

- **Daily:** route issues, close duplicates with canonical links, protect truth semantics first.
- **Pre-release:** run release checklists and `make product-verify` where applicable.
- **Hotfixes:** ship the smallest safe correction and add verification evidence.

## Triage examples

1. **Status reads live but evidence is stale**
   - Template: `bug_report.md`
   - Action: compare UI/API timestamps; patch semantics or docs.
2. **First-run setup fails**
   - Template: `install_setup.md`
   - Action: isolate env mismatch vs docs drift; fix smallest source of truth.
3. **Field report suggests edge-case behavior**
   - Template: `field_report.md`
   - Action: classify as known limitation, bug, or documentation clarification.

## What feedback is most useful

- Exact commands + first failure.
- Version/commit + transport context.
- Redacted evidence snippets (`mel doctor`, `/api/v1/status`, screenshots with captions).

## Related

- [Adoption guide](adoption-guide.md)
- [Claims vs reality](claims-vs-reality.md)
- [Launch and demo runbook](../runbooks/launch-and-demo.md)
