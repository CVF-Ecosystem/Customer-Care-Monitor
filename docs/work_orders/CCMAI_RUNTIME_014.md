# CCMAI-RUNTIME-014 — sync run identity and final-write ownership

**Date:** 2026-09-29 · **State:** WORK_ORDER / BUILD authorized · **Risk ceiling:** R2 · **Authority:** [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [SPEC](../specs/RUNTIME_SYNC_RUN_OWNERSHIP_S1_2026-09-29.md), R013 independent REVIEW PASS.

## Route and independence

Codex: `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR`. Claude: `IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` for one local BUILD/evidence commit. Codex performs independent R2 REVIEW. Before source edit, rehydrate current CVF state/handoff and record `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff. R001–R013 remain REVIEW PASS / FREEZE open; S1 is IN_PROGRESS.

## Objective and allowed scope

Give every **new** channel sync reservation a run ID and require that ID on terminal status/checkpoint writes. This prepares safe recovery design but does not implement recovery. Allowed source: `backend/db/models/channel.go`, `backend/engine/sync.go`, `backend/api/handlers/channels.go`; `backend/db/mysql.go` only if GORM `AutoMigrate` cannot add the nullable column safely without an explicit migration. Allowed tests: focused files under `backend/engine/`, `backend/api/handlers/` and `backend/db/`, including updates to R013 test fixtures/signatures. Allowed records: SPEC, this work order, new BUILD evidence under `docs/reviews/`, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json` and S1 roadmap. Do not edit scheduler/agent behavior unless compilation or the existing contract demonstrably requires a bounded change request first. No frontend, router, adapters/providers, jobs, persistent Compose DB, CVF core or unrelated worktree edits.

## BUILD requirements

1. Add one nullable internal `sync_run_id` field. Let the ordinary startup `AutoMigrate` add it; do not backfill existing rows or change a legacy `syncing` status. Make the reservation return a typed, nonempty run ID only when the same atomic tenant/channel/busy-checked update succeeds on exactly one row. Preserve R013 error classes and response boundaries. Do not add a second check-then-write arbiter.
2. Carry the returned reservation through both engine entries and the manual handler's launcher, worker and panic path. Ensure an already-reserved worker rejects a mismatched or empty reservation before adapter work. On accepted terminal write, predicate on tenant, channel, `syncing` and run ID, then clear the ID atomically with status/error/checkpoint. Preserve R013 success-only `last_sync_at`, trigger ordering, busy skip and agent result semantics. A stale worker's deferred cleanup and panic handler must not clear a newer run.
3. Use disposable MySQL and synthetic adapters. Test old/new schema migration twice, legacy NULL-ID `syncing` row, concurrent admissions, tenant isolation, failed/zero-row writes, exact manual 202/409 order, agent/scheduler regressions, and a deterministic two-generation race. Include a test that fails if the final predicate loses the run ID or the manual worker drops it. Do not send real channel/provider calls or touch the running Compose database.
4. Run focused and full backend tests, `go build ./...`, `go vet ./...`, catalog `-Check`, project doctor and `git diff --check`. Evidence records exact commands/results, disposable resource cleanup, schema compatibility/rollback limits, and the fact that conversation/message/token-refresh side effects are **not** fenced by this tranche. A failed gate returns `BUILD_BLOCKED`, not REVIEW_PENDING.

## Exit and effect boundary

Synchronize continuity/status and make one **local** BUILD/evidence commit, without push. Return `REVIEW_PENDING` to Codex. No automatic crash recovery, timeout takeover, lease, live channel/provider call, deployment, S1 closure or FREEZE is authorized. R015 or a later SPEC must decide when an orphan can be released and how any old worker's in-flight side effects are contained.
