# CCMAI-RUNTIME-028 — F06 job admission and run-owned cancellation

Status: REVIEW_PENDING. Issued 2026-10-02 by Codex (ORCHESTRATOR -> SPEC_AUTHOR -> WORK_ORDER_AUTHOR). Risk ceiling R2.

Authority: [SPEC](../specs/RUNTIME_JOB_RUN_OWNERSHIP_F06_2026-10-02.md), [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), immutable dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-028.json`, committed before BUILD at `6958e281d190935578809bb9900dd95d2a29509e`. Owner routes Claude implementation and Codex independent review. F05/R025/R026 retain REVIEW_PASS / FREEZE_OPEN; F06 source is unchanged at dispatch.

## Entry and roles

Claude is IMPLEMENTATION_WORKER and COMMIT_STEWARD for one local BUILD commit returned REVIEW_PENDING. Codex is ORCHESTRATOR/WORK_ORDER_AUTHOR and independent REVIEWER, not this tranche's implementation/repair worker.

Before BUILD, rehydrate manifest/policy, compact bootstrap or state fallback, memory prose, active handoff, implementation status, docs index, this SPEC/order, tranche and seed; run workspace doctor and required knowledge ingest. Acknowledge `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` in the active handoff and synchronize BUILD phase/mode/marker/status. Compact bootstrap absence is BOOTSTRAP_MIGRATION_PENDING, nonblocking. Contradictory current facts => BLOCKED_CONTINUITY_DRIFT at INTAKE. Read F03/R025 and F05/R027 accepted terminal/snapshot contracts plus R002/R009/R011 regressions. The seed is read-only, seeded by Codex and present at the base commit; do not edit it.

## Allowed source scope

Exact allowlist is the seed. Its broader test globs permit retaining existing probes, not rewriting unrelated semantics.

- `backend/engine/job_run_ownership.go` and its test: shared coordinator/reservation/target cancellation/owned cleanup. No distributed lease/schema or reclaim.
- `analyzer.go`, `analyzer_modes.go`: all methods use one admission/lifecycle; async reserved execution consumes the same owned run; checked early failures, cancel/publication/notification gates. Preserve signatures and F05 plan selection.
- `analyzer_incremental.go`: ONLY terminal ownership/cancel serialization and consistent Job/JobRun locking needed for this lifecycle. Source-version candidates/provenance/eligibility are read-only. Unused `isOrdinaryIncremental` cleanup remains outside scope.
- `analyzer*_test.go`, `reanalyze_test.go`, `scheduler.go`, `scheduler*_test.go`: shared cron/after-sync admission and bounded busy reporting; keep normal scheduling and sync behavior.
- `backend/api/handlers/jobs.go`, `job*_test.go`: synchronous reservation before202, exact run_id cancellation, bounded failures, atomic busy guard for DeleteJob/ClearJobRuns. Preserve R002 cascades and R009 config/F05 validation. No result-clear redesign.
- `agents.go`, `agent*_test.go`: only analysis-agent busy/error aggregation and focused shared-admission proof; permissions/MCP dispatch remain unchanged.
- `frontend/src/views/Jobs/JobDetail.vue`, `frontend/src/stores/jobs.ts`, `frontend/src/__tests__/job*.spec.ts`: target captured run_id, pending cancellation/polling and truthful error copy, with F05 regressions retained. No i18n-wide, route or unrelated view redesign.
- SPEC/order/roadmap, BUILD evidence under `docs/reviews/`, tranche/status/continuity and catalog/index when required. No worker authority-seed edits or tooling/workflow/.cvf/core changes.

## Implementation and evidence sequence

1. Trace every launcher/public method and the result/snapshot/evaluation/progress/fail/terminal/notification boundary before editing. Record one consolidated implementation plan, lock order and cancellation serialization against F06-01..10. If the contract cannot fit the seed, return BUILD_BLOCKED with the concrete boundary; no speculative scope widening.
2. Implement shared tenant/job admission and opaque run ownership first; integrate all entry paths without duplicate reservation or JobRun creation. Then targeted cancellation and lifecycle cleanup; then the two destructive guards and UI.
3. Execute every SPEC acceptance group A..F on disposable MySQL with barrier-controlled synthetic providers and disabled outbound effects. Cover real route-to-Analyzer and scheduler entrypoints; launcher-only stubs are insufficient. Record all required old-source and mutation detectors, source restoration and final reruns.
4. Run focused affected Go tests and full `scripts/test-backend.ps1` gate with R019 sentinels and no DB-unavailable skips; `go build ./...`, `go vet ./...`, gofmt/diff checks; race tests where supported. Run frontend full tests, forced typecheck (clear cached build metadata as its existing workflow requires), build, docs build, catalog and doctor. No new persistent service or real provider/channel call.
5. Synchronize REVIEW_PENDING in state, memory marker and current prose, handoff header/next move, IMPLEMENTATION_STATUS.currentPhase, order and tranche; keep BUILD/source truth separate from review acceptance. Record worker role acknowledgment and limits in `docs/reviews/RUNTIME_JOB_RUN_OWNERSHIP_F06_BUILD_2026-10-02.md`. Preserve R027 PASS; F06 remains OPEN. `buildCommit` may stay null at worker commit, with exact SHA supplied to reviewer; never invent a self-referential SHA or REVIEW_PASS.
6. Before commit run default downstream preflight, complete explicit source/evidence/continuity path preflight, and 46 gate tests. Existing untracked `knowledge/_index.json` and Python bytecode remain outside commits; disclose full-worktree failures and do not call them PASS. If any in-scope gate/test fails, resolve within scope or return blocked with evidence; do not declare closure.
7. Commit only scoped BUILD/evidence/continuity artifacts locally; return exact SHA and matrix to Codex. No push/FREEZE or worker self-approval. REVIEWER Codex evaluates all F06 requirements, provenance and retained probes; any repairs route back to Claude under unchanged authority when applicable.

## Stop conditions and prohibited effects

No real provider/channel call, persistent DB/customer data, credentials, push, merge, deployment, FREEZE, schema/DSN/timestamp/snapshot-format migration, distributed lease/recovery, permission change, adapter change, workflow/core edit, unrelated dead-code cleanup or tenant timezone activation. Channel/demo deletions retain inherited parent-guard behavior; F07/F02 live/message proof stays separate. Synthetic evidence asserts local admission/cancellation/storage/request behavior only, never CVF controlling AI. No claim current GitHub Actions passed from old PR head.

If a real boundary expansion is necessary, provide a concrete proposed amendment for Codex; do not repeatedly ask owner confirmation for routine same-scope repairs. At a third repair round without an independent new root cause, record REVIEW_COST_ESCALATION_REQUIRED.

## Handoff to reviewer

Return BUILD SHA, changed paths, requirements-to-tests matrix (no missing rows), provider/DB/network fixture boundaries, exact counts/skips and cleanup, old-source behavioral probes, mutations and restoration, test/gate/docs/typecheck results, panic/DB/cancellation limitations and any dissent. Requested cancellation is not observed terminal cancellation. Crash-stale/unowned rows remain fail-closed and need separate recovery authority.

## R028-R1 — consolidated same-scope repair (Codex, 2026-10-02)

Independent [review](../reviews/CCMAI_RUNTIME_028_F06_INDEPENDENT_REVIEW_2026-10-02.md) of BUILD `419a31ce0e34442a7410ad7ffb81d8586115e856` returns CHANGES_REQUIRED / FREEZE_OPEN. Claude transitions to REPAIR_WORKER / COMMIT_STEWARD after rehydration and acknowledgment; independent REVIEWER remains Codex. Seed, risk, scope, effects and commit ownership are unchanged.

1. F06-R1-01: make early error, panic and Abort/setup terminal paths honor the accepted-cancel outcome and serialize with terminal-first rejection through the same owner decision and checked finalizer. Preserve bounded errors, no checkpoint/notification and DB-failure blocking. Two reviewer probes currently store error instead of cancelled after an accepted cancel.
2. F06-R1-02: reject tenant/job mismatch against the reservation before consuming it or making any start/provider/source/publication effect; use its bound identity for Abort/finalization. A reservation for A currently executes B (one synthetic provider call, two published results) before the finalizer detects the mismatch. Add different-job, cross-tenant, occupied-B, valid exactly-once and cleanup tests.
3. F06-R1-03: pending cancellation settles only when its exact run ID is observed in a recognized terminal status. Missing/unknown/empty status remains unconfirmed and polling; observed error/cancelled/success outcomes receive truthful copy. Three new mounted probes currently report ended and stop polling without terminal evidence.

Retain `analyzer_f06_review_test.go` and the three new mounted cases unchanged in intent; do not skip/weaken them. Run all three repair groups plus inherited ownership/F03/F05/config/permission/snapshot tests, full uncached backend with R019 and frontend, forced typecheck/build, gate/catalog/docs/doctor and diff checks. First independent handler run failed with truncated output, isolated replay 26/26 passed; disclose this and investigate any recurrence rather than silently counting retries as unconditional PASS. Extend BUILD evidence with requirements, repaired boundary detectors/mutations, exact counts/skips, source restoration and cleanup. Return one local REVIEW_PENDING repair commit; no self-approval, push, real API, persistent DB, scope widening or FREEZE.
