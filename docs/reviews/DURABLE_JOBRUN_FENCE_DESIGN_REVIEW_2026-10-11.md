# R080 independent intended-design review

Reviewer: Codex /root, independent of Luna max /root/r080_worker. Risk R1 documentary. First packet: `315b48f0cc17475c984ead9667fb7df9cce76e0c`. First disposition: **CHANGES_REQUIRED**. Review scope DF01..10; no application/runtime or CVF governance proof.

## Consolidated first review

F1 — DF03/05/06/07, substantive: blanket commit-ambiguity exclusion/rollback is not established. Admission can roll back while its caller receives uncertainty; absence of a grant is not a persisted global hold. Retirement can clear the grant and commit while its response is lost; another coordinator sees no grant and can admit. An uncertain destructive COMMIT may already have taken effect, so caller-side abort does not prove rollback. Repair the selected contract, state table, writer/deletion rules, acceptance matrix and draft together. Specify justified phase-specific persisted invariants and what every coordinator observes, or explicitly retain a concrete unresolved design blocker requiring a documentary gate before implementation. Do not substitute a caller stopping for persisted exclusion. Preserve no automatic replay/reclaim and no inference that an external request did nothing.

F2 — DF04/05/06, substantive: the proposed prohibition on holding the owner mutex while waiting for DB locks conflicts with current `JobRunOwner.publish`, which holds `o.mu` throughout its DB transaction to serialize accepted cancellation. Define the coordinator registry mutex versus per-owner publication mutex, compatible acquisition order and cancellation/terminal winner protocol, or mark a concrete unresolved replacement protocol blocker. A diagram alone must not erase current semantics. Ensure the Tenant prefix rule for admission/create/config/reset is consistent throughout. Source: backend/engine/job_run_ownership.go lines 196–262 and analyzer.go publication transaction.

F3 — DF08, substantive completeness: the nonbinding draft delegates its whole test/build budget to a future seed and leaves notification-test paths unspecified. Supply finite conditional candidate invocation ceilings and exact candidate test paths, or narrow the immediate next proposal to a bounded documentary blocker-resolution gate while retaining a clearly non-executable later implementation outline. Distinguish the selected architecture from readiness to grant BUILD. No new effects are authorized by these proposals.

Root mechanical corrections M1–M3: activity helper citation added; shared executor citation extended through RunReserved; provider table changed admission-time to pre-dispatch to agree with its existing paragraph; repository-test nonexecution qualified as worker-only because root ran 46 governance gate unit tests (38.818s). Root corrects these directly; no worker substantive prose drafted. CLI ApplyPlan reference already existed and was already cited in the final first packet; no CLI finding.

## Evidence and quality boundary

First static audit: [receipt](probes/r080_first_root_static_audit_2026-10-11.json): 29 input identities, bounded citations/links, protected source371/history118/acceptedinputs75, original five quality arrays and seed identity unchanged. Static correctness did not certify semantic validity. Root local/cached PR preflight PASS7/7; 46 repository gate units PASS, metadata evidence only. Worker native/application/provider/DB/network invocations0; all future acceptance tests NOT_RUN.

First-pass acceptance: NO. Semantic repair rounds: 0 at this review; one consolidated repair authorized under unchanged seed after committed BUILD release and fresh ACK. Worker preparation checker Windows path-normalization failure retained in worker identity record, corrected before its successful static pass. Root checker initially assumed a different lineage schema and was adapted before the recorded first audit; not a worker defect. Root rg literalglob error and stale publication projection correction remain root-attributed. Root navigation/contract assistance is recorded; no matched model comparison or measured inference/token/cost basis exists.

## Current disposition

CHANGES_REQUIRED; implementation/recovery OPEN, deployed UNKNOWN; accounts parked. Original first packet is preserved in Git. Independent re-review follows a separately committed repair packet. A later documentary resolution gate may remain necessary even if this scoped architecture/spec is accepted.

Root precommit metadata error retained: first review publication preflight FAIL5/7 because root used unsupported catalog family review and omitted REVIEW_PENDING in tranche history. Corrected to continuity and explicit BUILD → REVIEW_PENDING → CHANGES_REQUIRED before commit. Not a worker defect or runtime check.
