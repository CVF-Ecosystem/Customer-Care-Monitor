# Work order CCMAI-RUNTIME-005 — S1 confidence truth

**State:** `REVIEW_PENDING` after BUILD (owner-approved path addition for demo.go and one compile-only results_test.go fixture; see the active handoff) · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER` -> `COMMIT_STEWARD`) · **Independent reviewer:** Codex (`REVIEWER`, next) · **Authority:** owner “next”, [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), and [SPEC](../specs/RUNTIME_RESULT_CONFIDENCE_TRUTH_S1_2026-09-27.md).

## Entry and role route

R001–R004 have independent REVIEW PASS but remain FREEZE open. This new S1 tranche inherits their accepted evidence and enters BUILD after this SPEC/WORK_ORDER. Before BUILD, Claude must rehydrate the current manifest, policy, state, memory, active handoff, implementation status and docs index; declare the CVF context; and append `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` acknowledgment to the active handoff. Claude returns one local BUILD/evidence commit for independent Codex R2 REVIEW. No self-approval or FREEZE.

## Objective and scope

Remove fabricated confidence certainty from new QC/evaluation results and from every job-result consumer, while preserving historical rows and treating classification numbers only as uncalibrated model reports.

Allowed implementation paths: `backend/db/models/job.go`, `backend/db/mysql.go` only if schema migration needs it, `backend/engine/analyzer.go`, focused tests in `backend/engine/` and `backend/db/`, `backend/api/handlers/jobs.go` and focused handler tests for job-result serialization, `backend/notifications/dispatcher.go` and its focused tests, `frontend/src/stores/jobs.ts`, and `frontend/src/views/Jobs/JobDetail.vue` only if it currently presents confidence. A small helper/test in these same directories is allowed. No provider, prompts, adapter, sync, snapshot, aggregate Results, scheduler, permission, customer data, deployment or CVF core change. If actual API wiring needs a different handler file, document the exact dependency and return a bounded change request before editing it.

Allowed accompanying paths: new `docs/reviews/RUNTIME_RESULT_CONFIDENCE_TRUTH_S1_BUILD_2026-09-27.md`, active handoff/state, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, roadmap/SPEC/work-order status lines. Keep unrelated `.gitignore` and `docs/references/` out of the commit.

## Required BUILD and evidence

1. Trace all `JobResult` writers, readers, JSON responses, notification formatting and current schema before editing. Implement the SPEC's nullable `confidence` and nullable persisted `confidence_basis`; explain how existing rows with NULL basis remain unavailable without rewriting their numeric values, plus migration/rollback behavior. Do not turn a model estimate into calibrated probability.
2. Implement the SPEC across new persisted rows, the actual job-result API and notifications. Never substitute `0` or `1.0` for unknown. Keep result verdict, score, evidence refs, tenant isolation and snapshot linkage unchanged. Any schema edit must be idempotent and checked on fresh and existing-table disposable MySQL.
3. Add focused tests for new QC/evaluation/tag records, legacy QC/tag rows, both job-result API paths, notification text and invalid model confidence. Include a regression that fails on the current fabricated `1.0` behavior. Use synthetic direct fixtures; no mock-provider output as governance proof.
4. Run focused and full backend tests on disposable MySQL, `go build ./...`, `go vet ./...`, relevant frontend build/type check if touched, catalog `-Check`, workspace doctor and `git diff --check`. Record exact commands/results, temporary DB cleanup, migration observations, consumer compatibility and remaining limits in BUILD evidence. Do not claim live CVF governance or confidence calibration.

## Exit and boundary

On success, transition `IMPLEMENTATION_WORKER -> SESSION_SYNC_STEWARD -> COMMIT_STEWARD`, synchronize continuity/status, make one local commit without push and return `REVIEW_PENDING` to Codex. If preserving historical/API semantics needs paths or effects outside this work order, or a required check fails, return `BUILD_BLOCKED` with the exact boundary/failure.

**External-effect ceiling:** local source/docs/tests and disposable MySQL `CCMA` only. No real provider API, credential use, channel sync, customer data, persistent Compose DB change, deploy, push, S2/S3/S5 implementation or FREEZE.
