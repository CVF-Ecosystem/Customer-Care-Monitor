# CCMAI-RUNTIME-030 — F02-D Pancake message-window coverage

Status: DISPATCH_READY. Issued 2026-10-02 by Codex (ORCHESTRATOR -> SPEC_AUTHOR -> WORK_ORDER_AUTHOR). Risk ceiling R2.

Authority: [SPEC](../specs/RUNTIME_PANCAKE_MESSAGE_COVERAGE_F02D_2026-10-02.md), [F02 assessment](../reviews/F02_REMAINING_EVIDENCE_AND_FREEZE_ASSESSMENT_2026-10-02.md), dispatcher-owned `CVF_SESSION/authority/CCMAI-RUNTIME-030.json`. Seed committed at `cbc7cba3af7d1b5ce77911fbb7e1fa62ff70c39f`, which is the tranche's baseCommit before BUILD. Worker must never edit seed. This planning order changes no product source; F02 remains OPEN.

## Roles and entry

Claude: IMPLEMENTATION_WORKER / COMMIT_STEWARD for one bounded local BUILD returned REVIEW_PENDING. Codex: planning author/session steward/local planning commit steward, then independent REVIEWER. No worker self-review or reviewer product repair. R029 and earlier REVIEW_PASS / FREEZE_OPEN dispositions remain unchanged.

Before any BUILD edit, rehydrate manifest/policy/bootstrap fallback, state/memory/active handoff/status/index, SPEC/order/tranche/seed and applicable shared learnings; run workspace doctor and knowledge ingest. Record `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` in active handoff before changing files. Synchronize BUILD in state/front marker/header/current prose/status/order/tranche, then run preflight before first source edit. Missing compact bootstrap is BOOTSTRAP_MIGRATION_PENDING, nonblocking. Contradictions require BLOCKED_CONTINUITY_DRIFT at INTAKE; resolve predictable fields/dependencies in one consolidated pass. Prevalidate all continuity replacements before writing; use rerunnable synchronization.

## Allowed scope and effects

Exact seed allows:

- `backend/channels/pancake.go`: message-only traversal/validation/error/order helpers. Preserve accepted conversation enumeration, mapping/redaction, endpoint/request-helper retry/throttle/size behavior.
- `backend/channels/pancake*_test.go`: retain accepted probes and add F02D-01..06/09 discriminating tests.
- `backend/engine/sync_pancake*_test.go`: actual adapter/engine/disposable-MySQL F02D-07..08 integration. No engine source edit.
- This SPEC/order and runtime roadmap; required session/tranche/status/review evidence/catalog/index records under portable gate conventions.

Permitted tests: loopback HTTP servers or closed synthetic RoundTripper, disposable isolated MySQL/test tenant, synthetic attachment responses if necessary, local test/build/vet/docs/catalog/doctor/gate commands and local bounded commits. Block unexpected network hosts and directly observe attempted external effects. Disposable fixtures must not read/use existing customer/channel/provider credentials. Synthetic test credentials are not live secrets. No authorized real channel/provider call, persistent database, push/merge/deploy/FREEZE, parent-core/schema/UI/permissions/workflow/tooling/dependency change or timestamp migration. Public endpoint documentation retrieval already performed; it is not channel proof.

## Bounded execution

1. Record consolidated implementation plan for all nine requirements and endpoint assumptions. Preserve first-seen mapping, validate old/duplicate rows, physical offsets, deterministic chronological output, fixed page budget and explicit terminal validation. Return BUILD_BLOCKED if existing engine semantics cannot satisfy integration acceptance within test-only engine scope.
2. Implement message traversal within allowed source. A missing array, duplicate-only page, old timestamp or budget exhaustion must not yield successful coverage. Preserve existing helper and conversation contracts; expose separate message coverage error. Errors/logs must not reveal test sentinel tokens, provider bodies or token-bearing URLs.
3. Run adapter matrix for shape/row/order/window/overlap/nonprogress/200-page/context/error/security boundaries. Record actual request positions and output ID sets. Use deterministic cancellation hooks rather than sleeps to claim ordering. Preserve mapping/media/redaction assertions and explain any changed legacy expectations.
4. Run actual adapter + engine on disposable MySQL: complete multiple-page exact storage, since equality, chronological normalization/overlap, failed-conversation diagnostic discard with successful peer storage, no checkpoint advance/no after-sync dispatch on partial, and successful retry/idempotent replay. Read DB values/IDs and directly count dispatch/network effects. Do not call skipped DB tests PASS.
5. Retained tests must fail against original adapter. Apply four mutations from F02D-09 separately with exactly verified replacement counts; restore original bytes in finally and verify identity. NOT_APPLIED/no-op is untested; BUILD-ERR is INCONCLUSIVE; survivor requires a discriminating test and rerun. Keep unexplained failures as unexplained even if later runs pass.
6. Run full channels, focused R013–R016/R023 engine regressions, full backend/build/vet, relevant race check where supported, docs build/catalog/doctor/diff and mandatory downstream gates/tests. Record exact commands, counts/skips, output and limits; race unavailable means NOT RUN. Teardown and deletion are separate steps using verified fixed absolute targets and native PowerShell literal paths. Verify cleanup independently.
7. Write `docs/reviews/RUNTIME_PANCAKE_MESSAGE_COVERAGE_F02D_BUILD_2026-10-02.md`: F02D-01..09 matrix, request inventory, exact returned/stored ID evidence, terminal/nonprogress/budget failures, checkpoint/dispatch evidence, original-source/mutations/restoration, regression counts/skips, fixture isolation/cleanup and not-run limits. No raw real secrets or customer data. Return REVIEW_PENDING and exact local BUILD SHA to Codex; buildCommit may be null in that commit and supplied afterward, never self-referential.
8. Synchronize current state/marker/header/prose/status/order/tranche/roadmap to REVIEW_PENDING. Run default preflight, complete explicit changed-path preflight, `preflight --base origin/main --head HEAD`, mandatory gate unit tests and diff before local commit. Exclude pre-existing `knowledge/_index.json` and two Python bytecode files; disclose their full-worktree/PR failures rather than calling those gates PASS. Commit only authorized files; no push or FREEZE.

## Failure conditions and reviewer return

False nil-success on incomplete/untrusted traversal, filtered invalid row, missed eligible ID, changed mapping/redaction, checkpoint/analysis after partial, missing deterministic detector, in-scope regression, undocumented skips or widened effects require CHANGES_REQUIRED/BUILD_BLOCKED. Same-scope repairs remain authorized; third same-root repair requires REVIEW_COST_ESCALATION_REQUIRED. A real scope/effect boundary requires a new dispatcher decision, never seed edits.

Return BUILD SHA, changed set, acceptance matrix, tests/DB observations, sanitized request inventory, detector output/byte restoration and gate/docs/catalog/doctor/cleanup evidence. Independent Codex REVIEW verifies source, exact integrated revision, seed author/timing, retained tests and evidence limits. Review may accept only local Pancake message contract; live offset snapshot stability, global F02, channel credentials, provider/governance readiness and FREEZE remain OPEN.
