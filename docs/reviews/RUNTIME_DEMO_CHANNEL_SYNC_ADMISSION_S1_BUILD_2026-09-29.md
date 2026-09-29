# CCMAI-RUNTIME-018 — BUILD evidence: demo fixture sync admission

**Date:** 2026-09-29 · **Status:** `REVIEW_PENDING` (Claude, IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD) · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_DEMO_CHANNEL_SYNC_ADMISSION_S1_2026-09-29.md), [work order](../work_orders/CCMAI_RUNTIME_018.md). Base `7582064`; role transition `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` recorded in the active handoff before any source edit.

## Change

| File | Change |
|---|---|
| `db/models/channel.go` | `IsDemoFixture bool` (`not null;default:false`, `json:"-"`): internal, never serialized |
| `api/handlers/demo.go` | the two fresh demo channels are created with `IsDemoFixture: true` |
| `db/mysql.go` | `backfillDemoFixtureChannels()` at the end of `AutoMigrate`, in one transaction. Marks only rows with (`zalo_oa`,`demo-zalo-oa`) or (`facebook`,`demo-fb-page`), credential bytes exactly `{"demo":true}`, in a tenant with `settings.is_demo_data = true`. For marked rows only, it resets `last_sync_status`/`last_sync_error` to empty when `last_sync_at IS NULL`, status is `error`, error starts `decrypt failed:`, and `sync_run_id`/`sync_lease_until` are NULL. `updated_at`, checkpoints, run IDs, leases and activity logs are never written |
| `engine/sync.go` | `ErrDemoFixture` (`demo_channel_not_syncable`). `ReserveChannelSync`'s conditional UPDATE carries `is_demo_fixture = FALSE`; the zero-row classifier reads the marker and returns `ErrDemoFixture` (after missing, before busy); a classification read failure stays `ErrSyncNotAdmitted`. `RecoverExpiredSyncLeases` (scan and update) excludes marked rows. `SyncAllChannels` (agent `sync_all`) skips marked rows so a mixed tenant does not report false errors |
| `engine/scheduler.go` | `syncAllChannelsTask` query adds `is_demo_fixture = FALSE`, so fixtures are excluded before the due-check |
| `api/handlers/channels.go` | `SyncChannelNow` returns 409 `{"error":"demo_channel_not_syncable"}` right after the tenant-scoped read, before config load and dispatch; the same 409 maps `ErrDemoFixture` when the marker appears between read and reservation |

Agent `sync_channel` needs no edit: `ErrDemoFixture.Error()` is the bounded string `demo_channel_not_syncable`, returned by the existing generic error shape with no dispatch and no write. No frontend, other route, adapter, OAuth or credential-format change.

## Tests (disposable MySQL, synthetic rows, fake adapter)

- `engine/sync_demo_fixture_test.go` (6): shared admission refuses a marked fixture with zero row writes, a wrong tenant sees `missing`, and an unmarked real channel of the same tenant is admitted (positive detector); `SyncChannel` on a fixture makes no adapter call or trigger and writes nothing; the scheduler over a mixed tenant never mentions or touches two fixtures, syncs both real channels, and constructs adapters only for the 3 real channels; agent `sync_all` returns nil on a mixed tenant; lease recovery releases the unmarked expired lease and leaves the marked look-alike untouched; forced UPDATE failure and forced classification-read failure both return `ErrSyncNotAdmitted` with the rows unchanged.
- `db/demo_fixture_migration_test.go` (2): fresh column is NOT NULL and the throwaway old-schema table upgrades twice with existing rows unmarked; 12 legacy cases (both fixtures, never-attempted, active run, checkpoint present, different error, stale run ID, real channel in a demo tenant, encrypted same-name, non-demo tenant, wrong type, near-miss bytes) checked after two `AutoMigrate` passes: marker exact, only the guarded decrypt error cleared, `last_sync_at`/run/lease/`updated_at` unchanged, second pass identical.
- `api/handlers/demo_fixture_sync_test.go` (3): fresh import marks both channels, the marker is absent from `GetChannel` output and cannot be changed by `UpdateChannel` body or metadata; manual sync of a fixture gives 409 with the exact body, zero config loads, zero launches, an unchanged row, while the real channel in the same tenant gets 202 with one launch; a marker set while the config loads gives 409, no worker, no write.

## Commands and results

- Focused runs of each file via `scripts/test-backend.ps1 -Packages … -Run …`: all PASS.
- `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./...` → 14 packages `ok` (handlers 54.7 s, engine 41.1 s, db 3.3 s); the script removed its MySQL and network and `docker ps -a` shows no `ccma-test*`. The persistent Compose database was not touched.
- `go build ./...` OK; `go vet ./...` OK; `git diff --check` OK. New test files are gofmt-clean; `channels.go`, `demo.go` and other listed files have pre-existing gofmt drift not touched here.
- Catalog `-Check` and workspace doctor: see the final line.

## Mutation checks (one production line each, restored byte-for-byte, verified by `cmp`)

| Mutant | Caught by |
|---|---|
| M1 remove the marker predicate from the reservation UPDATE | 3 engine tests |
| M2 remove it from the scheduler query | scheduler test (log and adapter count) |
| M3 zero-row classifier ignores the marker | 2 engine tests |
| M4 remove it from lease recovery | recovery test |
| M5 remove it from `SyncAllChannels` | agent test |
| M6 remove the manual pre-check | manual-route test (config load count) |
| M7 remove the reservation-denial mapping | race test |
| M8 backfill without the exact credential bytes | backfill test |
| M9 backfill without the demo-tenant condition | backfill test |
| M10 status clear ignores the checkpoint guard | backfill test |
| M11 status clear ignores the run-ID guard | backfill test |

## Failure modes, rollback and claim limits

- Migration effect: marked legacy fixtures lose only the old decrypt error and show as never synced; if the tenant already ran the demo importer twice with different tenants, each is handled by its own tenant setting. A fixture whose tenant no longer has `is_demo_data=true` is not marked (fail-safe: it stays as before).
- `is_demo_fixture` is absent from responses and cannot be written through channel input or metadata. Converting a fixture to a real channel remains delete/create.
- Older binaries ignore the marker and would resume attempting fixtures; a mixed-version rollout is unsupported. Rolling back the binary leaves the extra column in place.
- Not changed: real-channel error reporting, Zalo/legacy crashed-run recovery, historical `syncing` rows on fixtures (left as found), and the frontend status chip.
- Synthetic MySQL plus a fake adapter only; no real provider or channel was contacted, and this is not live CVF governance proof. Not verified against the persistent Compose database or a deployed binary.

## Changed set

`backend/db/models/channel.go`, `backend/db/mysql.go`, `backend/api/handlers/demo.go`, `backend/api/handlers/channels.go`, `backend/engine/sync.go`, `backend/engine/scheduler.go`, three new test files, this file, active state/handoff, memory, `IMPLEMENTATION_STATUS.json` and the S1 and UI roadmaps. Not self-approved; FREEZE open; no push.

**Final gates (after continuity sync):** catalog `-Check` PASS; workspace doctor 25/25 PASS; `git diff --check` clean.
