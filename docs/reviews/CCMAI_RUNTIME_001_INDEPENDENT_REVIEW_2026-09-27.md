# Independent review: CCMAI-RUNTIME-001

**Reviewer:** Claude (REVIEWER, not the implementer) · **Ngày:** 2026-09-27 · **Target:** `ade74addfd9890c3418c99ee02aecd6b73ee3b4a` · **Authority:** `CCMAI-RUNTIME-002` Gate A · **Disposition:** `PASS_WITH_REPAIRS` (repair round 1, re-verified).

## Inputs

- Spec `docs/specs/RUNTIME_FOUNDATION_S0_S1_2026-09-27.md`
- Work order `docs/work_orders/CCMAI_RUNTIME_001.md`
- BUILD evidence `docs/reviews/RUNTIME_FOUNDATION_S0_S1_BUILD_2026-09-27.md`
- Source/test at target: `backend/engine/sync.go`, `backend/engine/scheduler.go`, `backend/engine/sync_status_test.go`, corpus, `frontend/src/views/Channels.vue`, `frontend/src/views/Channels/ChannelDetail.vue`; callers `api/handlers/channels.go` (`SyncChannelNow`) and `api/handlers/agents.go` (`handleSyncAgent`).

## Checklist

| Area | Result | Basis |
|---|---|---|
| Checkpoint `success/partial/error` | PASS | `buildSyncStatusUpdates` adds `last_sync_at` only for `success`; decrypt/adapter/fetch-list failures use `error`; any item failure makes `finalStatus()` `partial`. Unit test `TestBuildSyncStatusUpdatesOnlyAdvancesSuccessfulCheckpoint`, `TestSyncProgressFinalStatus`. |
| After-sync suppression | PASS (inspection) | `SyncChannel` returns before `TriggerAfterSyncJobs` for `partial`, `error` and failed `success` status write. No unit test drives the full DB path; recorded as residual coverage gap, not a defect. |
| Aggregate error propagation | PASS | `SyncAllChannels` returns load error and `errors.Join` of channel errors; `handleSyncAgent` maps it to `status: error`. |
| Bounded observability | PASS | 10 details × 300 runes, total 4000 runes, hidden count retained. `TestSyncProgressErrorMessageIsBounded`. |
| Attachment failure | PASS | `downloadAttachments` returns aggregated failure; message is still upserted without `LocalPath`, conversation counted failed, channel `partial`, checkpoint retained so the next run replays. Local-disk fallback on tenant store error is intended behavior, not a failure. |
| Scheduler activity ownership | PASS | Scheduler no longer logs `sync.completed/sync.error`; `SyncChannel`/`updateSyncStatus` own `sync.completed`, `sync.partial`, `sync.error`. |
| Scheduler retry cadence | **DEFECT R1-A → repaired** | See below. |
| UI wording | **DEFECT R1-B → repaired** | Detail/history/chips distinguish partial as warning; list-view snack still claimed success. See below. |
| S0 corpus claim | **DEFECT R1-C → repaired** | See below. |
| Claim boundary / no provider | PASS | No AI/provider code touched; evidence claims data reliability only. |

## Findings and repairs

**R1-A (medium, regression).** Scheduler skipped channels using only `last_sync_at`. Because `partial`/`error` now keep that checkpoint, a failing channel was re-synced on every 5-minute tick regardless of its configured interval (e.g. 1440 minutes), multiplying external adapter calls and `sync.error/partial` activity rows. Before this tranche every status advanced `last_sync_at`, so the throttle held. Repair: `channelSyncDue` in `backend/engine/scheduler.go` throttles on the last attempt — `updated_at` for `partial`/`error` when newer than the successful checkpoint — while the fetch window still starts from the last success. Test `TestChannelSyncDueThrottlesFailedAttempts`.

**R1-B (low, UI claim).** `Channels.vue` `syncNow` showed a green success snack after `POST /sync`, which only returns `202 sync_started`; the result was not known and could be `partial`/`error`. Repair: informational snack stating sync started and pointing to the channel status chip.

**R1-C (low, evidence).** Handoff/BUILD claimed the corpus "validates 12/12" but no test loaded it. Repair: `TestS0VietnameseCorpusIsComplete` checks version, `synthetic_only`, 12 unique required case IDs, allowed `expected`, non-empty reason/risk.

## Observations (not defects in this scope)

- `SyncChannelNow` marks `syncing` and ignores that DB error; if the final status write fails the chip may stay `syncing`. Pre-existing; handler outside the tranche's allowed paths.
- Scheduler and manual sync can overlap for the same channel (no lock). Pre-existing; idempotent upsert limits impact.
- Zalo token-refresh callback ignores marshal/update errors; adapter credential code is forbidden scope.

## Commands and results (Go 1.27.0 local toolchain, module `go 1.26.0`)

- Before repair, target code: `go test ./engine -count=1` → `ok`.
- After repair: `go vet ./engine` → clean; targeted tests (`ChannelSyncDue|S0Vietnamese|SyncProgress|BuildSyncStatus`) → 5 PASS.
- `go build ./...` → PASS; `go test ./... -count=1` → all packages `ok`.
- `frontend: npm run build` → built.
- MySQL-backed engine tests skip on the host without a reachable DB; during Gate B they ran against a disposable `mysql:8.0` container together with these repairs (34 engine tests PASS, see `RUNTIME_SNAPSHOT_EVIDENCE_S1_BUILD_2026-09-27.md`).
- `gofmt -l engine` lists only pre-existing CRLF-checkout files; repaired/new files are formatted.

No provider API, channel credential, real sync or customer data was used. Repair round 1 closed all findings; Gate B may open.
