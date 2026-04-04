# Post-launch playbook (first weeks)

MEL’s doctrine is **evidence-first** and **no theatre**. This note is for maintainers routing the first wave of outside attention without diluting that bar.

## Signals to watch

- **Truth / semantics** — labels that imply live path, delivery, or RF success without ingest proof; degraded states that read as healthy.
- **Control safety** — approval bypass, ambiguous lifecycle phases, missing audit linkage.
- **Setup truth** — docs or README commands that fail on a clean machine; Node 24.x mismatch for frontend work.
- **Transport honesty** — confusion about supported vs unsupported surfaces (see README transport matrix and `docs/ops/transport-matrix.md`).

## Issue classes (route quickly)

| Class | Typical intake template | First response |
| --- | --- | --- |
| Setup / bootstrap | `.github/ISSUE_TEMPLATE/install_setup.md` | Reproduce with `mel doctor`, capture config posture (redacted). |
| Truth / degraded state | `.github/ISSUE_TEMPLATE/truth_semantics.md` | Compare API field to UI label; cite `docs/repo-os/terminology.md`. |
| Docs drift | `.github/ISSUE_TEMPLATE/docs.md` | Fix doc or narrow claim; link verification command. |
| Security | GitHub **Security** advisories + `SECURITY.md` | No public secrets or precise locations. |
| Contributor friction | `.github/ISSUE_TEMPLATE/contributor_onboarding.md` | Point to `CONTRIBUTING.md`, `docs/repo-os/README.md`, `make verify` subset. |

Label policy: apply `needs:triage` + one `kind:*` label on intake, then one `area:*` label for doctrine-sensitive reports (`area:degraded-state`, `area:control-safety`, `area:topology-or-status`, `area:environment`). See [`issue-routing-labels.md`](issue-routing-labels.md).

## Maintainer rhythm

- **Daily**: skim new issues; tag truth vs polish vs setup; close duplicates with a canonical link.
- **Pre-release**: `docs/release/RELEASE_CHECKLIST.md`, `docs/project/LAUNCH_CHECKLIST.md`, and `make product-verify` where applicable.
- **Post-release hotfix**: smallest diff that restores honest signaling; add a regression test or doc assertion when feasible.

## Example triage flows (template → labels → maintainer action)

1. **Topology overclaim report**
   - Intake template: `.github/ISSUE_TEMPLATE/truth_semantics.md`
   - Labels: `needs:triage`, `kind:truth-semantics`, `area:topology-or-status`
   - Maintainer action: compare the reported UI copy against `GET /api/v1/topology*` payload fields, then either patch UI wording or narrow docs claim with explicit non-claim language.

2. **Control-governance ambiguity**
   - Intake template: `.github/ISSUE_TEMPLATE/bug_report.md`
   - Labels: `needs:triage`, `kind:bug`, `area:control-safety`
   - Maintainer action: verify lifecycle separation (submit/approve/dispatch/execute/audit) in API and audit rows before accepting any “executed” claim.

3. **Setup/runtime mismatch**
   - Intake template: `.github/ISSUE_TEMPLATE/install_setup.md`
   - Labels: `needs:triage`, `kind:setup`, `area:environment`
   - Maintainer action: request first failing command from `make premerge-verify`, classify as docs fix vs script fix, and preserve explicit degraded/blocked messaging.

## What feedback is most valuable

- Reproducible steps tied to **version/commit** and **doctor output**.
- Screenshots that include **degraded / partial** banners, not only “green” paths.
- Suggested wording that **tightens** claims instead of expanding them without evidence.

## Related

- [Adoption guide](adoption-guide.md)
- [Claims vs reality](claims-vs-reality.md)
- [Demo / screenshot runbook](../runbooks/launch-and-demo.md)
