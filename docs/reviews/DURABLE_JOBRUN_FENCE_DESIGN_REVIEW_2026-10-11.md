# R080 independent intended-design review

Reviewer: Codex /root, independent of Luna max /root/r080_worker. Risk R1 documentary. First packet: `315b48f0cc17475c984ead9667fb7df9cce76e0c`. First disposition: **CHANGES_REQUIRED**. Review scope DF01..10; no application/runtime or CVF governance proof.

## Consolidated first review

F1 — DF03/05/06/07, substantive: blanket commit-ambiguity exclusion/rollback is not established. Admission can roll back while its caller receives uncertainty; absence of a grant is not a persisted global hold. Retirement can clear the grant and commit while its response is lost; another coordinator sees no grant and can admit. An uncertain destructive COMMIT may already have taken effect, so caller-side abort does not prove rollback. Repair the selected contract, state table, writer/deletion rules, acceptance matrix and draft together. Specify justified phase-specific persisted invariants and what every coordinator observes, or explicitly retain a concrete unresolved design blocker requiring a documentary gate before implementation. Do not substitute a caller stopping for persisted exclusion. Preserve no automatic replay/reclaim and no inference that an external request did nothing.

F2 — DF04/05/06, substantive: the proposed prohibition on holding the owner mutex while waiting for DB locks conflicts with current `JobRunOwner.publish`, which holds `o.mu` throughout its DB transaction to serialize accepted cancellation. Define the coordinator registry mutex versus per-owner publication mutex, compatible acquisition order and cancellation/terminal winner protocol, or mark a concrete unresolved replacement protocol blocker. A diagram alone must not erase current semantics. Ensure the Tenant prefix rule for admission/create/config/reset is consistent throughout. Source: backend/engine/job_run_ownership.go lines 196–262 and analyzer.go publication transaction.

F3 — DF08, substantive completeness: the nonbinding draft delegates its whole test/build budget to a future seed and leaves notification-test paths unspecified. Supply finite conditional candidate invocation ceilings and exact candidate test paths, or narrow the immediate next proposal to a bounded documentary blocker-resolution gate while retaining a clearly non-executable later implementation outline. Distinguish the selected architecture from readiness to grant BUILD. No new effects are authorized by these proposals.

Root mechanical corrections M1–M3: activity helper citation added; S07 range broadened (root later verified original range already covered RunReserved); provider table changed admission-time to pre-dispatch to agree with its existing paragraph; repository-test nonexecution qualified as worker-only because root ran 46 governance gate unit tests (38.818s). Root corrects these directly; no worker substantive prose drafted. CLI ApplyPlan reference already existed and was already cited in the final first packet; no CLI finding.

## Evidence and quality boundary

First static audit: [receipt](probes/r080_first_root_static_audit_2026-10-11.json): 29 input identities, bounded citations/links, protected source371/history118/acceptedinputs75, original five quality arrays and seed identity unchanged. Static correctness did not certify semantic validity. Root local/cached PR preflight PASS7/7; 46 repository gate units PASS, metadata evidence only. Worker native/application/provider/DB/network invocations0; all future acceptance tests NOT_RUN.

First-pass acceptance: NO. Semantic repair rounds: 0 at this review; one consolidated repair authorized under unchanged seed after committed BUILD release and fresh ACK. Worker preparation checker Windows path-normalization failure retained in worker identity record, corrected before its successful static pass. Root checker initially assumed a different lineage schema and was adapted before the recorded first audit; not a worker defect. Root rg literalglob error and stale publication projection correction remain root-attributed. Root navigation/contract assistance is recorded; no matched model comparison or measured inference/token/cost basis exists.

## Current disposition

CHANGES_REQUIRED; implementation/recovery OPEN, deployed UNKNOWN; accounts parked. Original first packet is preserved in Git. Independent re-review follows a separately committed repair packet. A later documentary resolution gate may remain necessary even if this scoped architecture/spec is accepted.

Root precommit metadata error retained: first review publication preflight FAIL5/7 because root used unsupported catalog family review and omitted REVIEW_PENDING in tranche history. Corrected to continuity and explicit BUILD → REVIEW_PENDING → CHANGES_REQUIRED before commit. Not a worker defect or runtime check.

Root source-line correction: RunReserved is analyzer_modes.go148–183, already within the worker first S07 range73–183. Root incorrectly called its extension to273 a missing executor citation. The extension is valid but unnecessary for that claim; this is root-attributed overhead, not a worker defect. Worker notified before repair handback.

Repair1 prehandback assistance: root inspected partial edits and reminded worker that F1 also covered remaining reset/deletion rollback phrases, admission readback-versus-dispatch consistency, and an unknown terminal result whose committed checkpoint may already have advanced. This stayed within the first consolidated F1 finding and same repair round; it is parent assistance, not first-pass worker autonomy or a second formal packet review.

## Independent repair1 result

Final disposition: **REVIEW_PASS — intended architecture/spec and nonbinding proposal only** at repaired packet `402d6a6f64c67dc6af62a7e2a73064cc628e06b8`. Original first disposition remains CHANGES_REQUIRED. One worker prose repair; root mechanically refreshed identity after acknowledged transfer, plus corrected root46 gate-test timing. The worker's repair validation wrapper failed at JS parse before execution and made no changes; no worker full repair identity validation is claimed.

F1 settled for this scope: admission ambiguity never dispatches that reservation; another explicit request uses the authoritative locked row and distinguishes committed-active from rollback/no-grant. Writer/terminal transactions retain the grant; an atomic clean terminal commit may already have advanced checkpoint despite lost response. Retirement occurs after the complete local effect/writer path is drained; ambiguous retirement can expose active or retired state and never authorizes old-caller resumption. Deletion/reset/prune distinguish known rollback from unknown possibly committed state and prohibit automatic repetition. External effects remain unrevoked and unreplayed; legacy cutover is a prerequisite.

F2 settled for the logical contract: registry mutex is released before DB waits, owner publication mutex may span its transaction, DB-held paths never acquire that owner mutex, cancel completes its DB read first, and terminal uncertainty does not reopen publication/cancel without known rollback and exact readback. This specifies intended ordering; actual FK/query/driver integration is OPEN.

F3 settled by narrowing the immediate proposal to three documentary gate artifacts, one author pass/one independent review pass and zero application/test/DB/provider/network effects. Exact conditional notification and other test paths are named. Later implementation is expressly non-executable; fixture and finite test/build ceilings require a new seed after accepted integration design.

| Requirements | Independent result |
|---|---|
| DF01 | PASS exact preserved baselines, shared modes/entrypoints, current/intended/inherited/UNKNOWN separated. |
| DF02 | PASS DB generation selected against external allocator and compatible keep-blocking; dependency/availability limits explicit. |
| DF03 | PASS logical tuple/non-reuse, atomic guards and phase-specific uncertainty; no claim fields are installed. |
| DF04 | PASS full DB/external effects and terminal tail/retirement; no column-only or remote exactly-once claim. |
| DF05 | PASS intended graph and named destructive integrations with concrete MySQL/GORM/FK/multirow blockers and bounded next gate; actual integration NOT accepted. |
| DF06 | PASS cancel/terminal/checkpoint distinctions, restart/old-writer/cutover barriers and disabled recovery. |
| DF07 | PASS discriminating future contrasts and persisted outcomes; every runtime test NOT_RUN. |
| DF08 | PASS bounded documentary immediate proposal, exact conditional implementation paths, independent R2 reviewer and new-authority requirement. |
| DF09 | PASS root final30 identities/36 reference definitions, six unchanged backend comparisons, source371/history118/accepted75, seed immutable and original five quality arrays unchanged. |
| DF10 | PASS preserved dissent, one prose repair, mechanical/identity transfer and root/worker errors separated; separate documentary quality sample, no model ranking/cost claim. |

[Final static receipt](probes/r080_independent_final_audit_2026-10-11.json) and [assessment](R080_LUNA_MAX_DURABLE_FENCE_DESIGN_ASSESSMENT_2026-10-11.md) bind this result. No same-scope review finding remains. MySQL/GORM/FK/query/migration/incarnation integration, runtime acceptance, fleet adoption, installed fencing and owner-death recovery remain OPEN/UNKNOWN. Local repository gates are metadata checks, not hosted CI or CVF runtime governance proof. Scoped local FREEZE remains a separate CLOSER decision.
