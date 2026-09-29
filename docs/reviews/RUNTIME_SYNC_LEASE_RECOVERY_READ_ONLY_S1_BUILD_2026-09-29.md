# CCMAI-RUNTIME-016 BUILD evidence — conditional lease recovery of GET-only sync runs

**Tranche:** `CCMAI-RUNTIME-016` · **Role:** IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD (Claude) · **Date:** 2026-09-29 · **Base commit:** `6b946ed` · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_SYNC_LEASE_RECOVERY_READ_ONLY_S1_2026-09-29.md), [work order](../work_orders/CCMAI_RUNTIME_016.md) · **Status:** `REVIEW_PENDING` for independent Codex review. No push and no FREEZE.

## Rehydration and role transition

Before BUILD, Claude re-read the manifest, policy, active state, active handoff, SPEC and work order.

- The worktree was clean at `6b946ed`, and core `26c686c` matches the manifest.
- `ACTIVE_SESSION_BOOTSTRAP_READ_MODEL.json` is absent. This is recorded as `BOOTSTRAP_MIGRATION_PENDING` and does not block the tranche.
- Claude recorded `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff before editing any source.

## Source audit of eligible adapters (read-only)

Both eligible adapters make outbound requests through a single helper. That helper is the only outbound call, and it issues GET only:

- `backend/channels/facebook.go:33`: `doRequest` builds `http.NewRequestWithContext(ctx, "GET", …)`. `FetchRecentConversations`, `FetchMessages` and `HealthCheck` all go through it.
- `backend/channels/pancake.go:106`: `doRequest` builds `http.NewRequestWithContext(ctx, http.MethodGet, …)`. Every fetch goes through it.

The engine's attachment download is also a GET (`http.NewRequestWithContext(ctx, http.MethodGet, …)`).

`zalo_oa` is not eligible. Its sync can issue a POST that refreshes a single-use token. The audit found no mutating outbound request for either eligible type, so `BUILD_BLOCKED` did not apply. The engine source now records the allowlist (`leaseEligibleChannelTypes`) as an audited contract.

## Changed source

Production changes are in `backend/db/models/channel.go`, `backend/engine/sync.go`, `backend/engine/scheduler.go` and `backend/api/handlers/channels.go`. `db/mysql.go` did not change: the ordinary `AutoMigrate` adds the new column. The adapters, router, frontend, jobs, credentials and CVF core were not touched.

**Schema.** `Channel.SyncLeaseUntil *time.Time` (`gorm:"type:datetime(3)"`, `json:"-"`) is internal and nullable. Existing rows are not backfilled.

**Admission marker.** The single conditional reservation update (R013/R014) now also writes the lease:

`sync_lease_until = CASE WHEN channel_type IN ('facebook','pancake') THEN NOW(3) + INTERVAL <lease> MICROSECOND ELSE NULL END`

This value comes from the row's current type and the database clock, and it is written in the same statement as `syncing` and the run ID. After the update, one silent read reports whether a lease was stored (`SyncReservation.Leased`). This read does not decide admission. If the read fails, no heartbeat starts. The run may then expire and be released, and R015's ownership fence stops its writes. The failure log line does not contain the run ID.

**Heartbeat.**
- Only a leased run starts a heartbeat goroutine. It fires every `syncHeartbeatInterval` (1 min) against a `syncLeaseDuration` (5 min).
- Each beat runs `UpdateColumn("sync_lease_until", NOW(3)+lease)` only under the predicate `id AND tenant_id AND last_sync_status='syncing' AND sync_run_id=<id> AND sync_lease_until IS NOT NULL`. It requires exactly one affected row. `updated_at` is left unchanged, and the statement runs in the silent session.
- If a beat affects zero rows, the class is `sync_ownership_lost`. If it fails with a DB error, the class is `sync write failed`. Either way the run context is cancelled.
- The run checks the heartbeat's failure at every gate: before the conversation fetch, each conversation, each message fetch and each message; after a failed adapter call or attachment download; and before its final write. On failure the run stops.
  - `sync_ownership_lost` is returned as that class, not as a partial result. The deferred transition is skipped because the run no longer owns a row.
  - A write failure lets the existing deferred `syncing → error` transition run under the same run ID.
- The heartbeat is stopped and joined before any terminal write, and again in the deferred handler. A Zalo run or a legacy run never gains a lease, because the heartbeat predicate requires a lease that is not NULL.

**Terminal writes.** `recordSyncStatus` handles the final success, partial and error writes and the deferred error write. The manual panic write in `handlers/channels.go` is the other terminal write. All of them clear `sync_lease_until` in the same statement that clears the run ID. That statement still carries R014's run-ID predicate and exact-row check.

**Release.** `RecoverExpiredSyncLeases(limit)`:
1. It scans at most `limit` rows (default 100) with `channel_type IN allowlist AND status='syncing' AND sync_run_id IS NOT NULL AND sync_lease_until IS NOT NULL AND sync_lease_until < NOW(3)`, ordered by deadline.
2. For each row it runs one `UpdateColumns` that repeats every predicate, together with the observed tenant and run ID.
3. That update sets `error` with a bounded message, clears the run ID and the lease, and stamps `updated_at` so the scheduler throttle applies.

It never touches `last_sync_at`, credentials, conversations, messages or attachment references. It triggers no analysis and launches no worker. A scan failure or a failed row write leaves that row `syncing` and returns the bounded `sync write failed`. A zero-row update means a concurrent heartbeat or terminal write won; the row is skipped silently. All of these statements run in the silent session.

**Scheduler.** `Start()` runs one recovery batch at startup and registers a one-minute `recover-expired-sync-leases` job, independent of each channel's due time. A released channel re-enters only through the ordinary `syncAllChannelsTask` reservation after its normal throttle.

## Tests (disposable MySQL, synthetic adapters and transports; DB-time expiry simulated by moving the stored deadline into the past)

**New file `backend/engine/sync_lease_recovery_test.go`:**
- **Admission marker.** `facebook` and `pancake` get `Leased` and a future lease written with the run ID. `zalo_oa` and `unknown_type` get NULL.
- **Heartbeat.**
  - A forged ID or the wrong tenant is rejected, and the row is unchanged.
  - The owner extends an expired lease by DB time, and `updated_at` is unchanged.
  - A Zalo run cannot gain a lease.
  - A DB error is reported as the bounded `sync write failed` class, with no trigger text.
- **Release.**
  - An expired pancake run and an expired facebook run are released: status `error`, bounded message, run ID and lease cleared, checkpoint and credentials unchanged, no fetch or trigger, and the old reservation is fenced.
  - These rows stay byte-unchanged: a live lease, a Zalo row with a forced expired lease, an old-binary row (run ID with NULL lease), and a legacy row (NULL run ID with an expired lease).
- **Deterministic races** (a seam between scan and release):
  - A heartbeat extends first → the row is not released and the owner keeps its run.
  - The final status is written first → the recorded success survives, and the checkpoint is set.
  - A newer generation reserves and its own lease is also expired → a release under the old observed run ID leaves the new run `syncing`.
  - A release DB error → bounded class, and the row stays `syncing`.
- **Heartbeat loss cancels a full `SyncReservedChannel`.** The adapter is paused in a context-aware GET.
  - An atomic recovery-like release → `sync_ownership_lost` within the timeout, no conversation written, no trigger, and the recovery's state is kept.
  - A forced heartbeat DB error → bounded `sync write failed`, and the run's own deferred transition leaves `error` with the run ID and lease cleared and no checkpoint.
- **Paused A→B generation handoff across full runs.**
  1. A is paused in its GET, with a heartbeat too slow to notice. Its lease expires and recovery releases it.
  2. B reserves (leased, new ID) and writes its credentials.
  3. A resumes. It returns `sync_ownership_lost` and writes no conversation or trigger. B's row is byte-identical: status, lease, checkpoint and credentials.
  4. B then completes `success`, with its checkpoint, one trigger, and its own conversation.
- **Terminal clearing.** Success, partial and error each clear the lease.
- **Scheduler.** The recovery task releases the expired pancake run and leaves an expired-lease Zalo row `syncing`. The next `syncAllChannelsTask` respects the throttle (0 fetches). After the attempt time is backdated, the channel is admitted normally: it ends `success` and the lease is cleared.
- **GORM output.** The run ID and the `sync_run_id` predicate never reach GORM's real output sink (Info, Warn-on-error, Warn-on-slow). The heartbeat and recovery statements run in the silent session.

**Other new and updated test files:**
- **New `backend/db/sync_lease_migration_test.go`.** On the fresh schema the column is a nullable `datetime` and a second `AutoMigrate` succeeds. An old-schema copy is upgraded twice. It holds an R015 `syncing` run with a run ID, a legacy `syncing` row and an idle row, and none of them is backfilled. The emulated model's tag is asserted identical to `models.Channel`'s.
- **New `backend/api/handlers/sync_lease_test.go`.** A manual pancake request returns 202 with `Leased` and a stored lease. A second request returns an exact 409 with no second launch. The panic write clears the status, run ID and lease.
- **Updated fixtures.** The R013 "zero-row" triggers in `sync_start_test.go` and `sync_single_flight_test.go` now also keep `sync_lease_until` unchanged, just as R014 made them keep `sync_run_id`.

## Commands and results

1. **Focused R016 tests.** `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Run 'Lease|Heartbeat|Recovery|ExpiredGeneration|TerminalWritesClear|ManualSyncCarries' -VerboseTests` → exit 0, 29 PASS (`api/handlers`, `db` and `engine` all `ok`).
2. **Full backend suite.** `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1` → exit 0. All **14** packages with tests are `ok`: `api/handlers` 20.8s, `db` 0.9s, `engine` 17.9s. A run earlier in the session, before the new tests were added, also passed all 14 packages.
3. **Build, vet and format.** From `backend/`, `go build ./...` and `go vet ./...` are clean. `gofmt -l` prints nothing for the changed and new Go files.
4. **Catalog, doctor and diff check.**
   - Catalog: `powershell -ExecutionPolicy Bypass -File scripts/manage_cvf_downstream_catalog.ps1 -Check` → PASS.
   - Doctor: `powershell -ExecutionPolicy Bypass -File ../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1 -ProjectPath <project>` → `RESULT: PASS (25/25)` after continuity sync.
   - `git diff --check` is clean apart from LF/CRLF notices.
5. **Cleanup.** The script removed its disposable MySQL container and network after every run, and the `ccma-test` container and network lists are empty. The persistent Compose stack and its database were not accessed. `go.mod` and `go.sum` are unchanged.

## Mutation checks (scripted; each restored and verified byte-equal)

| Mutation | Failing tests |
|---|---|
| Allowlist broadened to include `zalo_oa` | admission marker (zalo), heartbeat (zalo gains a lease), release (zalo released), scheduler |
| Run-ID predicate removed from the release update | race "newer generation … not released under the old id" |
| Expiry predicate removed from the release update | race "heartbeat wins" |
| R015 ownership fence disabled in `withOwnedSyncWrite` | A→B generation handoff (A writes over B), and the release test's fence check |
| Run-ID predicate removed from the heartbeat | heartbeat exact-run test (a forged ID extends) |

## Mixed version, rollback and claim limits

- **Older binaries (R015 or earlier) never set a lease.** Their runs keep a NULL lease, so R016 never releases them. Their terminal writes do not clear a lease, but they only ever finish their own NULL-lease runs.
- **An idle row always carries a NULL lease.** Every exit path of an R016 run clears the lease together with the run ID: terminal write, deferred transition, panic write and release. An older binary's reservation therefore always starts from a NULL lease and cannot inherit an expired lease that would make its run reclaimable. This depends on every R016 exit path clearing the lease, which the terminal-clearing tests cover.
- **Rolling back to R015 leaves a leased `syncing` row blocked.** R015 has no recovery code. Rolling forward to R016 later releases it once expired.
- **Rolling back never makes an older binary release anything.**
- **Expiry revokes ownership; it does not prove the worker died.**
  - A former worker may finish a GET that is already in flight.
  - R015's fence stops it from publishing conversations, messages or counts, or advancing the checkpoint. Its attachment bytes land under attempt-unique keys and may be left orphaned.
  - Its heartbeat, if it is still running, fails on zero rows and cancels its context.
- **Not included in this tranche:**
  - no manual force endpoint;
  - no recovery for Zalo, unknown types, NULL-lease rows or NULL-ID rows;
  - no job-run change and no provider credential change.
- **The heartbeat runs only inside the sync context.** The manual and scheduler contexts time out after 10 minutes. After a timeout the run ends through its own terminal write.
- **Still open:** Zalo and legacy crash recovery, worker-death proof, and a multi-process rollout plan. These remain separate decisions.
- **Evidence limits.** No real channel or provider was contacted. This is synthetic evidence only, not live CVF governance proof.
