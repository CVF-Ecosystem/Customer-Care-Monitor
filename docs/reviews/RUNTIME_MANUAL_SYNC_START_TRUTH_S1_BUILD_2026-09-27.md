# BUILD evidence: CCMAI-RUNTIME-007 — manual sync start truth

**Implementer:** Claude (`IMPLEMENTATION_WORKER`) · **Date:** 2026-09-27 · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_MANUAL_SYNC_START_TRUTH_S1_2026-09-27.md), [work order](../work_orders/CCMAI_RUNTIME_007.md) · **Status:** `REVIEW_PENDING` for independent Codex REVIEW. No FREEZE.

## Entry

Claude acknowledged `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` in the active handoff at `630975e`, after rehydrating manifest, policy, state, handoff, memory, implementation status, docs index, SPEC and work order. The changed source is `backend/api/handlers/channels.go` plus the new focused test `backend/api/handlers/sync_start_test.go`. No path addition was needed.

## Trace (before editing)

`SyncChannelNow` (`POST /channels/:channelId/sync`, `RequirePermission("channels","w")`):

1. Tenant-scoped lookup `id = ? AND tenant_id = ?`, returning 404 `channel_not_found` on a miss. **Kept.**
2. `db.DB.Model(&channel).Updates(syncing, "", now)` selected by the loaded model's primary key only, with **both the error and the affected-row count ignored**.
3. The worker goroutine was **always** started and 202 `{"message":"sync_started"}` was **always** returned.
4. Panic recovery logged `%v` of the raw panic value, then wrote `last_sync_error = "panic: <raw value>"` selected by **channel ID only**, again ignoring the write result.

## Implementation (`channels.go`)

- **`updateChannelSyncStatus(tenantID, channelID, status, message)`** writes `last_sync_status`, `last_sync_error` and `updated_at` with the predicate `id = ? AND tenant_id = ?`. It returns an error on a DB error or if `RowsAffected != 1`. MySQL counts changed rows, and `updated_at` always changes, so zero means the row was not written.
- **`SyncChannelNow`** keeps the lookup. It then records `syncing` through the helper. On any error it logs the channel ID and error server-side, returns `500 {"error":"sync_start_failed"}` with no DB text, and starts no worker. Only after a successful write does it call `startManualSync` and return the unchanged 202 `sync_started`.
- **`startManualSync`** is a package-level function variable, the small private test seam the work order allows, defaulting to `go runManualSync(...)`. Tests replace it to observe dispatch without running an adapter.
- **`runManualSync`** is the previous goroutine body, moved unchanged: it loads config, creates `NewSyncEngine`, calls `SyncChannel` with a 10-minute timeout and logs errors. Normal `SyncEngine` behavior, checkpoints and `partial` semantics are untouched.
- **`handleManualSyncPanic`** logs only the channel ID and the panic's Go type (`%T`), never the raw value. It records a tenant-scoped `error` status with the fixed bounded message `manualSyncPanicMessage` ("Đồng bộ thủ công dừng do lỗi nội bộ; xem nhật ký máy chủ."). If that write fails or matches no row, it logs "not recorded" and returns the error instead of silently succeeding.

## Tests (`sync_start_test.go`, disposable MySQL)

The worker launcher is stubbed in every test, so no adapter, credential or provider is ever called. Failure injection uses per-channel `BEFORE UPDATE` triggers, the same technique as the existing `channels_test.go` / `demo_test.go`, and each test drops its own trigger.

| Test | Proves |
|---|---|
| `TestSyncChannelNowRecordsStartBeforeAcknowledging` | 202 with the exact `{"message":"sync_started"}` body. Exactly one launch. At launch time the DB already shows `syncing` and no response byte has been written. The persisted status is `syncing` with an empty error. |
| `TestSyncChannelNowWriteFailureStartsNoWorker` | A trigger-forced write error gives 500 `{"error":"sync_start_failed"}` with no trigger/DB text, zero launches, and the visible status still `success`. |
| `TestSyncChannelNowZeroRowUpdateStartsNoWorker` | A trigger that keeps every column unchanged (zero affected rows) gives non-2xx and zero launches, with the status unchanged. |
| `TestSyncChannelNowOtherTenantCannotStart` | Another tenant gets 404 and zero launches, with the status unchanged. |
| `TestHandleManualSyncPanicRecordsBoundedTenantScopedStatus` | Correct tenant: status `error` with exactly the bounded message. The raw panic value `access_token=SECRET-DO-NOT-LEAK` is absent from both the DB and the log; the log names the channel and panic type. |
| `TestHandleManualSyncPanicWrongTenantIsNotSilent` | Another tenant's recovery returns an error, leaves the channel unchanged, and logs "not recorded" without the secret. |
| `TestHandleManualSyncPanicWriteFailureIsReturnedAndLogged` | A trigger-forced write failure is returned and logged, with no secret and the status unchanged. |

**Non-vacuity (mutation).** The handler was changed to ignore the start-write result (`false && err != nil`), which is the pre-change behavior. `TestSyncChannelNowWriteFailureStartsNoWorker` and `TestSyncChannelNowZeroRowUpdateStartsNoWorker` then failed with `status 202, want non-2xx; body {"message":"sync_started"}`. The source was restored from a backup, with no marker left.

## Verification

Host Windows Application Control still blocks newly built Go test binaries, as recorded in R005/R006, and it was not bypassed. Tests ran in `golang:1.26-alpine` on disposable `mysql:8.0` `ccma-r007-db` / network `ccma-r007-net` (no host data), with the module cache read-only, `GOPROXY=off` and `GOFLAGS=-mod=readonly`.

**Environment note.** The first focused run failed only at trigger creation (`Error 1419 ... log_bin_trust_function_creators`). The pre-existing `TestDeleteChannelFailureRollsBackWholeCascade` failed the same way. The `SET GLOBAL` had run while MySQL was still restarting after init. The setting was re-applied and verified (`@@log_bin_trust_function_creators = 1`), and the rerun passed. This is the same environment requirement Codex recorded in the R004 review, not a code defect.

```text
host: go build ./... ; go vet ./api/...                                      clean
container: go test ./api/handlers -run 'TestSyncChannelNow|TestHandleManualSyncPanic|TestDeleteChannel|TestPurgeChannel' -count=1 -v
                                                                            10/10 PASS (7 new + 3 existing channel tests)
mutation (ignore start-write result)                                        2 regressions FAIL as expected; restored
container: go build ./... && go vet ./... && go test ./... -count=1 -p 1     all 13 packages ok
gofmt -l (LF-normalized)                                                    sync_start_test.go clean; channels.go is flagged only for 3 pre-existing
                                                                            hunks (~167, ~525, ~604, also flagged at HEAD), which do not overlap this change (688-758)
git diff --check                                                            clean
backend/go.mod, backend/go.sum                                              unchanged
scripts/manage_cvf_downstream_catalog.ps1 -Check                            PASS
check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .                    PASS 25/25
```

Cleanup: the disposable container and network were removed. The persistent Compose `ccma` stack was not started, reset or touched. No frontend file changed.

## Limits and residuals

- **Zero affected rows:** because `RowsAffected` counts changed rows, two start writes for the same channel within the same millisecond could report zero and yield 500 for the second request. That rejection is conservative: no worker starts and no false 202 is sent.
- **Status after a failed start:** a failed start write leaves the previous visible status; nothing new is persisted. The client sees the 500.
- **Not solved here** (S1 residuals per the SPEC): concurrent manual/scheduler/agent runs and single-flight coordination; stale `syncing` after a crash or process restart; the ignored `config.Load()` error inside the worker, which is pre-existing and unchanged.
- **Untested panic path:** `runManualSync`'s actual `recover()` → `handleManualSyncPanic` path is not triggered by a test, because that would require running the real sync engine. The helper itself is fully tested, and the call site is covered by source inspection.
- 202 still means only "start recorded and worker dispatched", not a completed sync or complete upstream data. No real channel sync, credential, customer data, provider API, deployment or push was used, and this is not live CVF governance proof.
