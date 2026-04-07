# Founder-led demo script (technical buyer)

Use this for a **20–30 minute** walkthrough. Adjust to what is actually running; never narrate past the evidence on screen.

## 0) Framing (2 min)

- MEL is **local-first** incident intelligence and **trusted control** semantics — not a mesh routing stack and not proof of RF coverage unless persisted evidence supports it.
- Today’s goal: show **truth hierarchy** (what is live vs stale vs simulated) and **workflow surfaces** (incidents, diagnostics, exports).

## 1) Environment (2 min)

- Show clone path, `go version`, and that the binary is built with `make build` (or prebuilt artifact, if that is what you use).
- State bind address: default localhost vs any remote exposure — call out auth if non-loopback.

## 2) Cold start (5 min)

```bash
./bin/mel init --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

- Walk through **doctor** output: expected warnings without transports; no fake “all green.”
- Open the embedded UI; point to **transport / status** language (supported vs unsupported paths).

## 3) Fixture-backed proof (5 min)

```bash
make demo-seed
./bin/mel serve --config demo_sandbox/mel.demo.json
```

- Explicitly label data as **seeded/fixture**, not live fleet proof.
- Show **incident queue**, **timeline or diagnostics** — whatever best demonstrates persisted evidence in your build.

## 4) Truth contracts (5 min)

- Open [docs/product/HONESTY_AND_BOUNDARIES.md](product/HONESTY_AND_BOUNDARIES.md) or [docs/operator-truth-principles.md](operator-truth-principles.md) in a tab; connect UI labels to **live / stale / historical / degraded / unknown**.
- Mention **control lifecycle** at a high level: submission ≠ execution; cite [docs/ops/CONTROL_PLANE_TRUST.md](ops/CONTROL_PLANE_TRUST.md) if asked.

## 5) Handoff artifacts (5 min)

- Show or describe **proofpack / export / support bundle** flows that exist in your version (see [runbooks/proofpack-export.md](runbooks/proofpack-export.md), [getting-started/SUPPORT_BUNDLE_GUIDE.md](getting-started/SUPPORT_BUNDLE_GUIDE.md)).
- Emphasize **redaction defaults** and that exports are for review, not magic RCA.

## 6) Q&A boundaries

- **RF / routing / “AI certainty”:** defer to evidence; assistive inference is non-canonical.
- **Enterprise / compliance:** no certification claims unless you have independent evidence; point to [docs/product/EDITION_PACKAGING.md](product/EDITION_PACKAGING.md) and [docs/release/RELEASE_CRITERIA.md](release/RELEASE_CRITERIA.md).

## 7) Close

- Next step for a serious buyer: time-boxed **pilot** with explicit success criteria (see [internal/private/PILOT_READINESS_CHECKLIST.md](internal/private/PILOT_READINESS_CHECKLIST.md)).
