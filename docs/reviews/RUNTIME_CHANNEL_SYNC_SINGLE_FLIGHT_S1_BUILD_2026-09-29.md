# CCMAI-RUNTIME-013 BUILD evidence — cross-path channel sync single-flight

**Tranche:** `CCMAI-RUNTIME-013` · **Role:** IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD (Claude) · **Date:** 2026-09-29 · **Base commit:** `2a74875` · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_CHANNEL_SYNC_SINGLE_FLIGHT_S1_2026-09-29.md), [work order](../work_orders/CCMAI_RUNTIME_013.md) · **Status:** `REVIEW_PENDING` for independent Codex review; no FREEZE.

## Rehydration and role transition

Before BUILD, Claude re-read the manifest, policy, active state, active handoff, SPEC and work order; core `26c686c` matches the manifest. `ACTIVE_SESSION_BOOTSTRAP_READ_MODEL.json` is absent (`BOOTSTRAP_MIGRATION_PENDING`, non-blocking). The role transition `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` was recorded in the active handoff before any source edit.

## Source trace and change

Changed source: `backend/engine/sync.go`, `backend/engine/scheduler.go`, `backend/api/handlers/channels.go`, `backend/api/handlers/agents.go`.

- **One shared admission.** `engine.ReserveChannelSync(tenantID, channelID)` runs a single conditional `UPDATE channels SET last_sync_status='syncing', last_sync_error='', updated_at=now WHERE id AND tenant_id AND (last_sync_status IS NULL OR last_sync_status <> 'syncing')`. Exactly one affected row admits. Zero rows are classified by a follow-up read that never decides admission: no row for that tenant → `ErrSyncChannelMissing`; row `syncing` → `ErrSyncAlreadyRunning` (`sync_already_running`); an idle row with zero rows, a multi-row change or any DB error → `ErrSyncNotAdmitted`. Errors and logs carry no SQL or credential text.
- **Engine entries.** `SyncChannel` now reserves first, then calls the new exported `SyncReservedChannel` (decrypt, adapter, fetch, upserts, final status, trigger), which never reserves again. `SyncAllChannels` is unchanged and therefore reports a busy member as an aggregate error wrapping `sync_already_running`. Private seams `newSyncAdapter` (default `channels.NewAdapter`) and `triggerAfterSync` (default scheduler trigger) exist only for deterministic tests.
- **Final status writes.** The former `updateSyncStatus` is now `recordSyncStatus(tenantID, channelID, status, msg)`: scoped to `id AND tenant_id AND last_sync_status='syncing'`, checked `.Error` and `RowsAffected == 1`; the checkpoint (`last_sync_at`) remains success-only and the after-sync trigger still runs only after a recorded success. If a run leaves `SyncReservedChannel` without a recorded final status (failed write, zero rows, panic), a deferred tenant-scoped `syncing → error` transition is attempted with a fixed bounded message; a failure of that transition is logged (`could not leave syncing state`) and never masks the original error, and a panic is re-raised after the attempt.
- **Manual handler.** Order is unchanged: tenant-scoped lookup (404) → config admission (500) → reservation → worker launch → `202`. A busy channel returns exact `409 {"error":"sync_already_running"}` and launches nothing; a channel that vanished returns 404 `channel_not_found`; any other reservation failure returns the existing generic 500 `sync_start_failed`. The worker calls `SyncReservedChannel`. `updateChannelSyncStatus` (now used by the panic path) is conditional on `syncing`, so a finished, missing or transferred row is never overwritten.
- **Scheduler.** A busy channel is skipped with a log line, not counted as synced and not touched; other errors stay logged failures.
- **Agents.** `sync_channel` busy returns `error` with the single bounded reason `sync_already_running`; `sync_all` routes through `SyncChannel`, so a busy member makes the aggregate non-success. Other response shapes are unchanged.

Not changed: config, adapters/provider clients, models/migrations, router, frontend, jobs, analysis behavior, external requests. No timeout takeover, startup cleanup or process-local lock was added.

## Tests

New `backend/engine/sync_single_flight_test.go` (disposable MySQL, synthetic tenants/channels/credentials, recording fake adapter and trigger; no real channel) and `backend/api/handlers/sync_single_flight_test.go`. Existing panic tests in `sync_start_test.go` now put the channel in `syncing` first because the panic write is conditional; a helper `markSyncing` was added.

- Reservation: 16 concurrent admissions of one channel → exactly 1 winner and 15 busy; tenant/channel scoping (other tenant and unknown channel → missing, row unchanged; another channel of the same tenant and another tenant's channel admit independently); empty and NULL status admit; forced DB error and a zero-row trigger admit nothing and leak no trigger text.
- Engine entry: busy `SyncChannel` creates no adapter call, no trigger, no status overwrite (`updated_at` unchanged); accepted-run detector (adapter fetched once, `success`, checkpoint advanced, one trigger observed after the success status); `SyncReservedChannel` on a pre-reserved channel runs (a second acquire would report busy); a gated in-flight run makes a second entry busy while an independent channel completes.
- `SyncAllChannels` with a busy member: aggregate error wrapping busy, busy member not fetched, idle member still succeeds, other tenant untouched. Scheduler task: busy channel skipped with a skip log, status untouched, eligible channels run.
- Completion: partial and fetch-error keep `last_sync_at` and fire no trigger; final-write DB error (trigger) reports failure, no checkpoint, no after-sync trigger, and the failed transition is logged; zero-row final write after tenant reassignment reports `0 rows affected` and leaves the transferred row and checkpoint untouched; a row finished by someone else is not overwritten; an adapter panic ends in a bounded `error` status and is re-raised.
- Handlers: first manual request `202` persisted-before-launch; second while busy exact `409`, launches stay 1, status stays `syncing`; a preset-busy row is unchanged (`updated_at`) with no dispatch; wrong tenant stays 404; the manual worker (`runManualSync`) ends in its own `error` state, proving it does not reserve twice; panic recovery does not overwrite a non-syncing row; agent `sync_channel` busy → bounded non-success and untouched row; agent detector (idle channel reaches the engine past admission for `sync_channel` and `sync_all`); agent `sync_all` with a busy member is not success.
- Test seams (`newSyncAdapter`, `triggerAfterSync`, loader/launcher/transport) are restored in `t.Cleanup`; triggers are dropped; no `t.Parallel`. The gated fetch and its wait are time-bounded so a broken admission fails instead of hanging.

## Commands and results

1. Focused: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Run 'Sync|Reserve|Scheduler|Agent|Manual|OAuth|Channel' -VerboseTests` → exit 0, 113 PASS / 0 SKIP (`api/handlers` 15.6s, `engine` 3.9s).
2. Full: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1` → exit 0; all 13 packages with tests `ok` (`api/handlers` 20.0s, `engine` 9.4s).
3. From `backend/`: `go build ./...` and `go vet ./...` clean; `gofmt -l` on the changed Go files prints nothing except the pre-existing CRLF state of `agents.go`, which git normalizes.
4. Catalog `powershell -ExecutionPolicy Bypass -File scripts/manage_cvf_downstream_catalog.ps1 -Check` → PASS; doctor `powershell -ExecutionPolicy Bypass -File ../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1 -ProjectPath <project>` → `RESULT: PASS (25/25)` after synchronization; `git diff --check` clean (only LF/CRLF notices).
5. Cleanup: the script removed its disposable MySQL container and network after each completed run. An aborted mutation run (below) left one test container and its network; both were removed by hand, and `docker ps -a --filter name=ccma-test` and the matching network list are empty. The persistent Compose containers (`ccma-app-1`, `ccma-db-1`, `ccma-nginx-1`) and database were not touched; `go.mod`/`go.sum` unchanged.

## Mutation checks — performed

Each mutation was applied temporarily, the focused tests were run, and the file was restored byte-for-byte (`cmp` against a backup):

- A. Reservation without the `<> 'syncing'` predicate → 9 tests failed (same-channel winner count, engine/scheduler/agent/manual busy cases, concurrent entry).
- B. Manual worker calling `SyncChannel` instead of `SyncReservedChannel` → `TestRunManualSyncDoesNotReserveAgain` failed.
- C. Final status write reduced to `id` only → the zero-row reassignment and finished-row tests failed.
- D. Scheduler busy-skip disabled → `TestSchedulerSkipsBusyChannelAndRunsOthers` failed (busy logged as a failure, not a skip).

The first attempt at A hung a concurrency test (the second entry was admitted and blocked in the gated fetch); that run was stopped by hand, sources were restored, and the wait was then made time-bounded.

## Limits and residuals

- **Crash recovery is not solved.** A process crash after reservation leaves `syncing` until a separate tranche defines recovery. There is no run generation/lease: a future recovery that clears `syncing` while an old worker is still alive is not fenced, so this reservation does not guarantee exclusivity against that case.
- The deferred `syncing → error` transition after a failed final write may itself fail (logged); the row can then stay `syncing`.
- After an engine panic in a manual worker, the handler's own conditional panic write finds the row already `error` and logs a 0-row message; the recorded state is still correct.
- Synthetic adapter and disposable MySQL only: real adapters, real provider latency, multi-process deployments and clock skew are untested. The Zalo token-refresh callback, conversation/message upserts and job semantics are unchanged.
- Not live CVF governance proof: no provider API call was made and none was required, because no governance behavior is claimed.
