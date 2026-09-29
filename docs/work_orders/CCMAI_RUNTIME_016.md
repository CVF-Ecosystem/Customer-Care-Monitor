# CCMAI-RUNTIME-016 — conditional recovery of GET-only channel sync runs

**Date:** 2026-09-29 · **State:** CHANGES_REQUIRED; R016-R1 repair authorized · **Risk ceiling:** R2 · **Authority:** [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [R016 SPEC](../specs/RUNTIME_SYNC_LEASE_RECOVERY_READ_ONLY_S1_2026-09-29.md), [R015 independent re-review](../reviews/CCMAI_RUNTIME_015_R1_R2_INDEPENDENT_REREVIEW_2026-09-29.md), [R016 independent review](../reviews/CCMAI_RUNTIME_016_INDEPENDENT_REVIEW_2026-09-29.md).

## Route and bounded objective

Codex: `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR`. Claude: `IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD`, then one local `REVIEW_PENDING` commit for independent Codex review. Before editing source, Claude rehydrates the canonical CVF state and records `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff. R015 is `REVIEW_PASS / FREEZE_OPEN`; S1 remains IN_PROGRESS.

Implement the R016 SPEC's lease and conditional release **only** for `facebook` and `pancake` channel sync. A lease expiry revokes local run ownership; it is not proof that the worker died. R015's fencing makes an old worker's local writes fail after release, while these adapters' outbound sync operations are currently GET-only. Zalo and rows without an R016 lease remain blocked.

## Allowed paths and limits

Source: `backend/db/models/channel.go`, `backend/engine/sync.go`, `backend/engine/scheduler.go`, and `backend/api/handlers/channels.go` only for clearing a lease in the existing manual panic write; `backend/db/mysql.go` only if the ordinary idempotent `AutoMigrate` cannot add the nullable deadline. Narrow fixture/signature test updates under `backend/engine/`, `backend/api/handlers/` and `backend/db/` are allowed. Inspect `backend/channels/facebook.go` and `backend/channels/pancake.go` read-only to verify GET-only behavior; modifying an adapter requires a bounded change request first. Records: this SPEC/work order, BUILD evidence under `docs/reviews/`, active state/handoff, memory, `IMPLEMENTATION_STATUS.json`, S1 roadmap, and catalog only if required.

No frontend/router/API contract change, new recovery endpoint, Zalo or legacy-row release, job-run cleanup change, provider credential change, deployment, real channel/provider request, persistent Compose DB access, CVF core edit, push or FREEZE. Keep unrelated worktree files out of the commit.

## BUILD acceptance and stop conditions

1. Implement atomic admission marker and DB-time lease, exact run-ID heartbeat with fail-closed cancellation, and conditional expired-row release as specified. Preserve existing 202/409/manual panic behavior, scheduler admission, checkpoint rules and bounded logging. Test and document mixed-version/rollback behavior.
2. Prove supported and denied channel types, NULL-ID/NULL-lease rows, failed heartbeat, DB/read/write errors, zero rows, terminal clearing and normal scheduler re-admission. Prove races against heartbeat and final status, plus a paused A-to-B generation handoff across full `SyncReservedChannel`; protect B's rows, status, checkpoint and credentials. Use disposable MySQL and synthetic transports/stores only. Include meaningful mutations of allowlist, run-ID/expiry predicates and the old-worker write fence.
3. Run focused and full backend suites, `go build ./...`, `go vet ./...`, catalog `-Check`, project doctor and `git diff --check`. Evidence must state exact commands/results, temporary-resource cleanup, source audit of outbound methods, claim limits, and that Zalo/legacy recovery is still open. Return `BUILD_BLOCKED` if an eligible adapter is found to have a mutating outbound sync operation, or if an atomic release/heartbeat race cannot be proven within these paths. Do not silently broaden scope.
4. Synchronize continuity/status, check the exact changed set, and create one local BUILD/evidence commit. Return `REVIEW_PENDING` to Codex; Claude must not self-PASS or FREEZE.

## Repair addendum R016-R1 — 2026-09-29

Codex's independent review of BUILD `a7c0a78` returned `CHANGES_REQUIRED`: a successful attachment transfer can hide a concurrently failed heartbeat because the post-download `stopErr` check is conditional on `attachmentErr != nil`. A reviewer-only deterministic probe forced a heartbeat DB error while a synthetic store completed on cancellation; the full run returned a bounded error but still persisted a message. See the linked review for the exact reproduction and output.

Claude may edit `backend/engine/sync.go`, focused tests under `backend/engine/`, the existing R016 BUILD evidence, and permitted continuity files. Rehydrate and record `REVIEWER (Codex) → REPAIR_WORKER (Claude)` in the active handoff before source edit. Check heartbeat failure after every attachment download regardless of transfer result; abort before message publication and clean only newly created attempt keys. Add a permanent deterministic regression proving no message/count/checkpoint/trigger after a heartbeat failure during a successful attachment transfer. Keep normal attachment partial semantics when the heartbeat is healthy. Run focused/full backend, build, vet, catalog, doctor and diff check; make one local repair commit and return `REVIEW_PENDING` for Codex re-review. No push, live provider/channel call, expanded recovery eligibility, rollout or FREEZE.
