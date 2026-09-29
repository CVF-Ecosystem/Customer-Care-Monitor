# CCMAI-RUNTIME-018 — block demo fixture sync attempts

**Date:** 2026-09-29 · **State:** WORK_ORDER / Claude BUILD authorized, REVIEW_PENDING on return · **Risk ceiling:** R2 · **Authority:** [SPEC](../specs/RUNTIME_DEMO_CHANNEL_SYNC_ADMISSION_S1_2026-09-29.md), [UX-06 investigation](../reviews/UX_001C_SYNC_STATUS_INVESTIGATION_2026-09-28.md), [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md).

## Route and allowed scope

Codex: `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR`. Claude: `IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD`, then a single local BUILD/evidence commit `REVIEW_PENDING` for Codex's independent R2 review. Claude rehydrates current CVF state and records `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff **before** source edit.

Allowed source: `backend/db/models/channel.go` (internal marker), `backend/db/mysql.go` (idempotent exact legacy backfill), `backend/api/handlers/demo.go` (new fixture marker), `backend/engine/scheduler.go`, `backend/engine/sync.go` (shared admission and recovery predicates), and `backend/api/handlers/channels.go` only for the bounded manual rejection. Focused tests under `backend/db/`, `backend/engine/` and `backend/api/handlers/` are allowed. Records: this SPEC/work order, BUILD evidence under `docs/reviews/`, active state/handoff, memory, `IMPLEMENTATION_STATUS.json`, S1/UI roadmaps and catalog only if required.

No frontend, other API route, adapter, OAuth, credential format, real provider/channel call, persistent Compose DB, CVF core edit, push, deployment or FREEZE. Keep unrelated worktree files out of the commit. If the exact legacy identity or atomic admission cannot be proven in these paths, stop with `BUILD_BLOCKED`; do not switch to tenant-wide skipping, names alone or a client-writable metadata flag.

## Acceptance and gates

1. Implement the SPEC's server-owned marker, fresh import and exact idempotent legacy backfill. Clear only the narrowly defined old fixture decrypt error; do not touch unrelated rows, successful checkpoints, active run IDs/leases or activity logs. Verify fresh and upgrade migration twice on disposable MySQL.
2. Scheduler skips marked fixtures before due-check; shared `ReserveChannelSync` rejects marked fixtures atomically for all entry paths; manual route returns the specified 409 before config load/worker. Recovery excludes marked fixtures. Test marked/unmarked and mixed tenant cases, real-channel success detector, zero rows, DB errors, status/run-ID immutability and no adapter/outbound call. Mutate/remove both critical predicates and prove focused tests fail.
3. Run focused/full backend tests, `go build ./...`, `go vet ./...`, catalog `-Check`, workspace doctor and `git diff --check`. Evidence must list exact commands/results, temporary MySQL cleanup, migration effects, denial response, mixed-version/rollback limits and that no real provider/channel was contacted. Do not claim CVF governance proof from synthetic tests.
4. Synchronize continuity/status and check the exact changed set. Make one local BUILD/evidence commit; return `REVIEW_PENDING` to Codex. Claude must not self-PASS or FREEZE.

## R018-R1 repair order — 2026-09-29

Codex's independent R2 review of `f4bc93c` is `CHANGES_REQUIRED / REVIEW_OPEN`: [review evidence](../reviews/CCMAI_RUNTIME_018_INDEPENDENT_REVIEW_2026-09-29.md). The legacy backfill's text comparisons inherit case-insensitive MySQL collation, so a near-match type/external ID can be marked and a near-match status/error prefix can be cleared. This violates the SPEC's exact identity and narrow cleanup contract.

Claude may act as `REPAIR_WORKER` under the existing R2 authority after rehydration and recording `REVIEWER (Codex) → REPAIR_WORKER (Claude)` in the active handoff. Allowed implementation paths for this repair are only `backend/db/mysql.go` and focused `backend/db/` tests; review evidence, this work order and continuity records may be updated. Make the four text checks case-sensitive, add wrong-case type, external ID, status and error-prefix rows to the disposable-MySQL backfill matrix, and show that removal of the case-sensitive condition makes a focused test fail. Preserve the canonical fixture cases, idempotence, all unrelated rows and the fresh import/runtime admission behavior. Run focused/full backend tests, build, vet, catalog, doctor and diff-check. Record exact commands and MySQL cleanup in an R1 evidence addendum, then one local repair/evidence commit `REVIEW_PENDING` for Codex re-review. No provider/channel call, persistent Compose DB, global collation change, push, deployment, self-PASS or FREEZE.
