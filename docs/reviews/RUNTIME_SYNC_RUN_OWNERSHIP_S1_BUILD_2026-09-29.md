# CCMAI-RUNTIME-014 BUILD evidence — sync run identity and final-write ownership

**Tranche:** `CCMAI-RUNTIME-014` · **Role:** IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD (Claude) · **Date:** 2026-09-29 · **Base commit:** `48245db` · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_SYNC_RUN_OWNERSHIP_S1_2026-09-29.md), [work order](../work_orders/CCMAI_RUNTIME_014.md) · **Status:** `REVIEW_PENDING` for independent Codex review; no FREEZE.

## Rehydration and role transition

Before BUILD, Claude re-read the manifest, policy, active state, active handoff, SPEC and work order; core `26c686c` matches the manifest. `ACTIVE_SESSION_BOOTSTRAP_READ_MODEL.json` is absent (`BOOTSTRAP_MIGRATION_PENDING`, non-blocking). The role transition `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` was recorded in the active handoff before any source edit.

## Source trace and change

Changed source: `backend/db/models/channel.go`, `backend/engine/sync.go`, `backend/api/handlers/channels.go`. `backend/db/mysql.go`, scheduler and agents needed no edit (the scheduler and agents reach the engine only through `SyncChannel`, whose signature is unchanged).

- **Schema.** `models.Channel` gains `SyncRunID *string` (`gorm:"type:varchar(36)"`, `json:"-"`), a nullable internal column added by the ordinary startup `AutoMigrate`. No backfill, no change to existing rows or to a legacy `syncing` status. It is not in `ChannelResponse` and not serialized from the model.
- **Reservation.** `ReserveChannelSync` now returns `(SyncReservation{TenantID, ChannelID, RunID}, error)`. The same single conditional update (`id AND tenant_id AND status not syncing`) also sets `sync_run_id` to a fresh `uuid.New()` (crypto-random) value; the reservation is returned only when exactly one row changed. R013 error classes (`ErrSyncAlreadyRunning`, `ErrSyncChannelMissing`, `ErrSyncNotAdmitted`) and log text are unchanged and carry no ID. No second arbiter was added.
- **Carrying the reservation.** `SyncChannel` passes the reservation to `SyncReservedChannel(ctx, channel, reservation)`, which rejects an empty reservation or one whose tenant/channel differ from the channel with `ErrSyncReservationMismatch` before any adapter work. The manual handler passes the reservation to `startManualSync`, `runManualSync` and `handleManualSyncPanic`. Nothing is inferred from the channel snapshot or a global.
- **Terminal writes.** `recordSyncStatus` (final success/partial/error and the deferred error transition) and the manual `updateChannelSyncStatus` (panic path) now require `id AND tenant_id AND last_sync_status='syncing' AND sync_run_id = <run id>`, check the error and exactly one affected row, and clear `sync_run_id` to NULL in the same update. `last_sync_at` remains success-only and the after-sync trigger still runs only after a recorded success. An empty reservation cannot write. A stale worker therefore gets a `0 rows affected` error and changes nothing; failure text does not contain the run ID (tested).

Not changed: scheduler/agent behavior, response shapes (manual 202/409/404/500), config admission order, adapters, providers, models other than `Channel`, router, frontend, jobs. No startup sweep, timeout reclaim, lease, heartbeat, endpoint or recovery was added.

## Tests

All on disposable MySQL with synthetic tenants/channels/credentials, the R013 recording fake adapter and trigger recorder, and fixture-only "recovery" (a direct SQL release). No real channel or provider.

- `backend/db/sync_run_migration_test.go`: the fresh schema has `channels.sync_run_id` as nullable `varchar(36)`, and a second `AutoMigrate` succeeds. The "old schema" is a throwaway table (`channels_014_old`) created without the column, holding a legacy `syncing` row and an idle row, then upgraded twice through GORM `AutoMigrate` by a model that declares the column with the identical tag (asserted equal to `models.Channel`'s). The column appears once, idempotently; both legacy rows keep their status and a NULL ID. The live `channels` table is never dropped or altered, because other packages' tests share the database in parallel.
- `backend/engine/sync_run_ownership_test.go`: distinct, stored, internal run IDs per generation, cleared by the terminal write; 16 concurrent reservations give one winner whose ID is the stored one and no reservation for losers; a legacy NULL-ID `syncing` row stays busy and byte-unchanged while an idle row is admitted; failed and other-tenant reservations return no reservation and store no ID; mismatched/empty reservations are rejected before any fetch or trigger without touching the owner's row; `recordSyncStatus` with a forged ID, wrong tenant, missing channel, other channel, or against a NULL-ID row changes nothing; success/partial/error each clear only their own ID (another channel's run is undisturbed) with checkpoint and trigger only after success; failure text/log never contain the run ID.
- Deterministic two-generation race (`TestStaleWorkerCannotFinishOverNewerRun`, four modes: success, partial, error, panic): reserve A, block A in its adapter, release the row as a recovery would, reserve B, then let A finish/fail/panic. A returns an error and B's status, error, `updated_at`, checkpoint and ID are unchanged, no after-sync trigger fires; B then completes normally and clears its ID.
- `backend/api/handlers/sync_run_ownership_test.go`: the launcher receives a reservation whose ID equals the stored one and names the request's tenant/channel; a busy request returns exact 409, launches nothing and leaves the owner's ID; the ID is absent from the channel model JSON and the channel list response; `runManualSync` finishes only with its own ID (a dropped/forged ID leaves the owner's `syncing` and ID intact); a stale manual worker and its panic handler cannot clear a newer run, while the owner's panic handler still works.
- Existing R013 tests were adapted: the launcher stub and panic tests use real reservations (`reserve` helper); the "zero-row" triggers now also keep `sync_run_id` unchanged (otherwise the reservation legitimately changes a row).

## Commands and results

1. Focused: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Run 'Sync|Reserve|Scheduler|Agent|Manual|OAuth|Channel|RunID|Stale|Record' -VerboseTests` → exit 0, 138 PASS / 0 SKIP.
2. Full: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1` → exit 0; all 14 packages with tests `ok` (`api/handlers` 20.0s, `db` 0.4s, `engine` 11.5s).
3. From `backend/`: `go build ./...` and `go vet ./...` clean; `gofmt -l` on the changed Go files prints nothing except the pre-existing CRLF state of `channels.go`, which git normalizes.
4. Catalog `powershell -ExecutionPolicy Bypass -File scripts/manage_cvf_downstream_catalog.ps1 -Check` → PASS; doctor `powershell -ExecutionPolicy Bypass -File ../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1 -ProjectPath <project>` → `RESULT: PASS (25/25)` after synchronization; `git diff --check` clean (only LF/CRLF notices).
5. Cleanup: the script removed its disposable MySQL container and network after each run; `docker ps -a --filter name=ccma-test` and the matching network list are empty. The persistent Compose containers and database were not touched; `go.mod`/`go.sum` unchanged.

## Mutation checks — performed

Each mutation was applied temporarily by script, the focused tests were run, and the file was restored (verified equal to the original):

- A. `recordSyncStatus` without the `sync_run_id` predicate (status-only) → 9 tests failed (stale worker in all four modes, non-owner writes, manual worker with a forged ID, stale manual worker).
- B. Manual worker passing a different run ID to `SyncReservedChannel` → the manual-worker tests failed.
- C. Manual panic write without the run ID predicate → `TestStaleManualWorkerAndPanicHandlerCannotClearNewerRun` failed.
- D. Reservation not storing the ID → 13 tests failed.

## Schema compatibility, rollback and mixed-version limits

- Adding the column is additive and nullable: the previous binary ignores it. Rolling the binary back leaves the column and any run IDs of in-flight runs in place; an old binary's terminal writes do not clear or check the ID, so a row it finishes keeps a stale non-NULL `sync_run_id` while idle. That is harmless to R013 admission (which ignores the ID and overwrites it on the next reservation) but not clean. Dropping the column is a manual step and is not needed.
- Mixed versions during a rolling deploy: an old-binary worker can finish a new-binary run by status only, and a new-binary worker cannot finish a run reserved by an old binary (NULL ID → zero rows, run left `syncing`, reported as ownership lost). Old-binary runs reserved before the upgrade are legacy NULL-ID rows: they stay `syncing` and blocked, as with any crash-stuck row.
- This tranche does **not** fence conversation/message upserts, attachment downloads or the Zalo token-refresh persistence of a stale worker; only final status and checkpoint writes are fenced. A stale worker also still consumes its adapter call. Automatic recovery, lease/heartbeat and timeout takeover are not implemented, so a stuck `syncing` still needs a future tranche that defines evidence of worker death or side-effect fencing.
- Clock and multi-process behavior beyond one MySQL primary, and real adapters, are untested.
- Not live CVF governance proof: no provider API call was made and none was required, because no governance behavior is claimed.

## Repair R014-R1 — run ID and SQL values in GORM output (Claude, REPAIR_WORKER, 2026-09-29)

Role transition `REVIEWER (Codex) → REPAIR_WORKER (Claude)` was recorded in the active handoff before any edit. The [independent review](CCMAI_RUNTIME_014_INDEPENDENT_REVIEW_2026-09-29.md) was right: GORM's logger has its own writer and interpolates query values, so the reservation (run ID in the `UPDATE` values) and the run-ID-fenced terminal and manual panic writes (run ID in `WHERE`) printed the ID and SQL at Info (development) and on errors/slow queries at Warn (production). The BUILD's log test captured only the application's `log.Writer()` and could not see this.

**Fix (query-scoped; `backend/db/mysql.go` untouched, other DB logging unchanged).**
- New `engine.RunWriteDB()` returns `db.DB.Session(&gorm.Session{Logger: db.DB.Logger.LogMode(logger.Silent)})`. It is used only for `ReserveChannelSync`, `recordSyncStatus` (final and deferred error writes) and the manual `updateChannelSyncStatus` (panic write). The session reads `db.DB.Logger` at call time.
- Raw driver errors no longer cross the boundary: `recordSyncStatus` returns `update sync status <status>: write failed` and `updateChannelSyncStatus` returns `update sync status: write failed`; zero-row results keep `N rows affected`. Agent results and worker logs therefore carry only that class. Exact-row fencing, the run-ID predicate, R013 response shapes and reservation classes are unchanged.

**Tests observing the real sink.** `db.DB.Logger` is replaced, only around the calls under observation, by `logger.New(log.New(&buf, "", 0), …)` (GORM's own logger type, writer = buffer), at the three levels where the default logger prints SQL: Info, Warn on error, and Warn on slow query (threshold 1ns).
- Positive detector (`TestGormSinkObserverSeesOrdinaryStatements`, `TestHandlerGormSinkObserverSeesOrdinaryStatements`): an ordinary `db.DB` statement carrying a marker value shows up in the sink at every level (a deliberately failing statement at "Warn on error"), so the absence checks are meaningful.
- `TestRunIDWritesNeverReachTheGormSink` (engine): accepted run (reservation + success write), a run whose final and deferred writes fail by trigger, a forged-ID write and a failed reservation. Neither the issued run IDs, the forged ID, the column name `sync_run_id`, the trigger message `forced` nor `Error 1644` appear in the sink, the application log or any returned error.
- `TestManualRunIDWritesNeverReachTheGormSink` (handlers): manual reservation through the handler, an owner panic write, then a forced-failure panic write and a worker run whose writes fail; same absences plus the exact returned class `update sync status: write failed`.
- `TestRunIDWriteFailuresAreBoundedClasses` and `TestAgentResultNeverCarriesRawWriteErrors`: the engine error and the agent-visible result contain only the bounded class (`write failed`), never the trigger text.

**Mutation checks (restored, verified equal):** `RunWriteDB` returning plain `db.DB` → both sink tests failed at all three levels; `recordSyncStatus` wrapping the raw DB error again → the sink, bounded-class and agent-result tests failed.

**Gates re-run.** Focused `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Run 'Sink|BoundedClasses|NeverCarries' -VerboseTests` → exit 0, all PASS. Full `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1` → exit 0, all 14 packages `ok`. `go build ./...` and `go vet ./...` clean; catalog `-Check` PASS; doctor 25/25 and `git diff --check` clean after synchronization. The script removed its disposable MySQL container and network after every run; `docker ps -a`/network lists for `ccma-test` are empty; the persistent Compose stack was untouched.

**Claim limits.** Only statements issued through `RunWriteDB` are silenced. Other statements keep default GORM logging; none of them takes the run ID as a value or predicate (the classification `SELECT` and the activity-log read do not), but a future statement that does must also use it. Silencing also hides GORM's own error/slow trace for these three writes; failures remain visible as the bounded application log lines. The sink tests replace the logger on the shared `db.DB` for the duration of a call and run sequentially. Synthetic adapter and disposable MySQL only; final-write fencing only, no recovery, lease or side-effect fencing; not live CVF governance proof.
