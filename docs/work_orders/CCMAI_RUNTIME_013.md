# CCMAI-RUNTIME-013 — cross-path channel sync single-flight

**Date:** 2026-09-29 · **State:** BUILD complete, REVIEW_PENDING (Claude, local commit; evidence in docs/reviews/RUNTIME_CHANNEL_SYNC_SINGLE_FLIGHT_S1_BUILD_2026-09-29.md) · **Risk ceiling:** R2 · **Authority:** [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [SPEC](../specs/RUNTIME_CHANNEL_SYNC_SINGLE_FLIGHT_S1_2026-09-29.md), R012 independent REVIEW PASS.

## Route and independence

Codex: `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR`. Claude: `IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` for one local BUILD/evidence commit. Codex performs independent R2 REVIEW. Before source edit, rehydrate current CVF state/handoff and record `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff. R001–R012 remain REVIEW PASS / FREEZE open; S1 is IN_PROGRESS.

## Objective and allowed scope

Ensure at most one admitted run per tenant/channel across manual, scheduler and agent sync entry paths sharing MySQL. Allowed source: `backend/engine/sync.go`, `backend/engine/scheduler.go`, `backend/api/handlers/channels.go`, and `backend/api/handlers/agents.go`. Allowed tests: focused files under `backend/engine/` and `backend/api/handlers/`; small private test seams are allowed if restored by cleanup. Allowed records: this work order, SPEC, new BUILD evidence under `docs/reviews/`, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, and S1 roadmap status. No frontend, router, config, adapter/provider client, models/migration, persistent Compose DB, CVF core or unrelated worktree edit. Return a bounded change request before editing an additional path.

## BUILD requirements

1. Implement one shared DB conditional reservation by `id AND tenant_id AND last_sync_status != syncing` (including NULL). Classify busy, missing/changed tenant and DB failures. Never use check-then-write as the arbiter. Preserve R007/R008 manual config and start ordering: valid config, successful reservation, worker launch, then `202`. A busy manual request returns exact `409 {"error":"sync_already_running"}` without launching.
2. Make public engine `SyncChannel` reserve first; give the manual worker a narrowly exposed already-reserved execution entry so it does not reserve twice. Route scheduler and both agent actions through the reserving entry. A busy scheduler channel is skipped and not counted; a busy agent action returns non-success, and `sync_all` does not hide a busy member behind aggregate success. Keep unrelated accepted response shapes and analysis behavior.
3. Final success/partial/error status writes, including manual panic recovery, must be scoped to tenant and current `syncing` state, with checked error and exact row count. Preserve success-only checkpoint advance and trigger-after-recorded-success. If a run fails before its normal final write, ensure it attempts to transition that tenant/channel from `syncing` to an error state without masking the original failure; a failed transition remains observable and does not claim completion. Do not add a timeout takeover, startup cleanup or process-local-only lock. Record the lack of run-generation fencing as a limit for the separate recovery tranche.
4. Prove the SPEC acceptance with disposable MySQL and synthetic transport/adapter or controlled seams. Include a same-channel race, cross-tenant and independent-channel cases; manual 202/409, scheduler and agent busy paths; accepted-run detector, status-write failure and zero-row cases. Assertions must observe worker/adaptor dispatch, status and checkpoint, not only HTTP codes. No real channel/provider endpoint, saved Alibaba key, customer data or live CVF governance claim.
5. Run focused and full backend tests, `go build ./...`, `go vet ./...`, catalog `-Check`, project doctor and `git diff --check`. Evidence records commands, exact outcomes, mutation or other non-vacuity check, disposable-resource cleanup, claim limits and residual crash-state recovery. A failing gate returns `BUILD_BLOCKED`, not REVIEW_PENDING.

## Exit and effect boundary

Synchronize continuity/status and create one **local** BUILD/evidence commit, without push. Return `REVIEW_PENDING` to Codex. No real channel/provider call, deployment, S1 closure or FREEZE is authorized. The separate crash recovery and backend-dependent UI/API intersections remain open.
