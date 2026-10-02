# CCMAI-RUNTIME-025 / F03 BUILD evidence — analyzer incremental coverage

**Date:** 2026-10-01 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 · **Role:** IMPLEMENTATION_WORKER + COMMIT_STEWARD (Claude, owner-confirmed); independent REVIEWER: Codex
**Authority:** [SPEC](../specs/RUNTIME_ANALYZER_INCREMENTAL_COVERAGE_F03_2026-10-01.md), [work order](../work_orders/CCMAI_RUNTIME_025.md), dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-025.json` (commit `517406f`, unedited; allowed paths and prohibited effects identical to the tranche record). Base `d8ec254`.
Synthetic provider and disposable MySQL only. Real provider/channel calls: **zero**; no API key, persistent DB, schema/snapshot-format change, push or FREEZE. The pre-existing untracked `knowledge/_index.json` is not part of the commit.

## Source trace before the change (planning HEAD `1e74d18`, unchanged at base)

- Ordinary `RunJob`/`RunJobWithProvider(limit 0)` → `runJobInternalExt(full=false, max=0, no dates, no since-override, exclude=false)`: `since = last_run_at` (or now − 24h); candidates `last_message_at > since` **and** `NOT EXISTS (job result of this job with created_at >= last_message_at)`; snapshot loaded with `sent_at > since` (windowed, `omitted_earlier_messages` recorded); on a clean run `last_run_at = finishedAt` (**completion**). Single mode loaded lazily per conversation; batch prepared all snapshots first; both saved the prepared snapshot.
- Terminal writes: run status update retried 3× with errors only logged; `db.DB.Model(&job).Updates(...)` unchecked and not tenant-scoped; notification sent regardless of whether those writes succeeded.
- Scheduler: cron `runScheduledJob` reloads the job and calls `NewAnalyzer(cfg).RunJob`; `TriggerAfterSyncJobs` filters by tenant/active/after_sync and input channel, then runs `RunJob` in a goroutine. Both reach the ordinary path.
- Why moving only the checkpoint to the start is not enough: a conversation whose `last_message_at` is before any checkpoint (backdated/late ingestion), a late message with an old `sent_at` that does not raise `last_message_at`, and an in-place edit are all excluded by `last_message_at > since`, by the result-time `NOT EXISTS`, or by the `sent_at > since` window, independent of where the checkpoint sits.

## Implementation

`backend/engine/analyzer_incremental.go` (new):
- `isOrdinaryIncremental(full, max, from, to, sinceOverride, exclude)` names the only covered mode; everything else keeps its previous path (F05 untouched).
- `ordinaryIncrementalCandidates`: all conversations of the job's tenant and input channels, `ORDER BY last_message_at ASC, id ASC` (NULL first in MySQL, not omitted), no time filter.
- `sourceVersionChanged`: latest committed `conversation_evaluation` for the same tenant, job (via `job_runs.job_id` and `job_runs.tenant_id`) and conversation, `created_at DESC, id DESC`. No evaluation or a legacy evaluation without a snapshot → changed. A linked snapshot is loaded by id **and tenant**; missing, or failing `VerifySnapshotProvenance(snapshot, tenant, conversation, evaluation.job_run_id)` → `ErrSourceVersionUnverifiable` (an error, never a skip). Otherwise compare the stored digest with the current full snapshot's digest. DB errors are returned.
- `prepareOrdinaryIncremental`: per candidate, `loadConversationSnapshot(conv, zero)` (existing builder/format/PII handling, full local history), skip empty source, run the decision, keep only changed conversations with **the exact prepared snapshot**; snapshot/verification errors are counted as errors; checks cancellation between candidates.
- `finalizeOrdinaryRun`: one transaction that locks the run (`id`, `tenant_id`, `job_id`) and the job (`id`, `tenant_id`) rows `FOR UPDATE`, writes the run's terminal status/finish/summary/error and the job's `last_run_status`/`updated_at` and — only when given — `last_run_at`; a missing row is not retried, other failures retry up to 3 attempts. On final failure it makes a best-effort run `error` mark (no checkpoint) and returns an error.
- Private test seams: `analyzerNow`, `ordinaryFinalizeRetryDelay`, `sendJobNotifications` (production values unchanged).

`backend/engine/analyzer.go`:
- `runJobInternalExt` captures `scanStart` before candidate selection; in the ordinary mode it uses the candidates/preparation above, sets `conversations_found` to the conversations needing work (changed + errors), analyzes the prepared list in single mode and passes it to `runBatchMode` (new `preparedIn/usePrepared` parameters; other modes prepare inside as before). On a clean run the checkpoint is `scanStart.Truncate(time.Second)`; partial/cancelled/error runs pass no checkpoint. A finalization failure returns `(run with status error, err)`, skips the completion activity and **sends no notification**. Non-ordinary modes keep the old selection, windowed snapshot and finalization byte-for-byte in behavior (the single-mode loop body was extracted into a closure shared by both modes).
- `Analyzer.providerOverride` (private) lets scheduler tests route a synthetic provider through the real Analyzer.

`backend/engine/scheduler.go`: private seams `newScheduledAnalyzer` (= `NewAnalyzer`) used by both entry points and `afterSyncJobFinished` (no-op) deferred in the after-sync goroutine. Lookup, active/deleted checks, routing and lifecycle are unchanged.

## Tests (`analyzer_incremental_test.go`, `scheduler_incremental_test.go`; disposable MySQL, single **and** batch unless noted)

- Mode boundary table for `isOrdinaryIncremental`.
- **Insert during run 1:** two conversations inserted from the provider call (after selection/preparation) with last-message times before run 1 finished, one of them later given a message after finish → run 2 evaluates exactly both; run 3 makes no provider call and adds no evaluation/snapshot/result. Checkpoint equals the scan start truncated to the second (analyzer clock pinned with a fractional second).
- **Empty first run then late old ingestion:** zero-work success records the scan start; a conversation dated 72h back (before the checkpoint and the initial 24h window) is evaluated next, then skipped.
- **Late message / edit without newer `last_message_at`:** an inserted message dated before the checkpoint and later an in-place content edit are each analyzed once; the transcript contains the change and the saved snapshot's message IDs equal the submitted transcript's (provenance-valid, expected count); replays make no call.
- **Change after preparation:** a message inserted during the provider call is absent from both the submitted transcript and the saved snapshot; the next run re-analyzes with it; the third run skips.
- **Receipts are job/tenant bound:** another job's evaluation (even with a future timestamp), this job's legacy evaluation without a snapshot and an orphan snapshot without an evaluation never hide a conversation; other-channel and other-tenant conversations are never evaluated; PASS evaluations without findings are valid receipts (next run: 0 calls).
- **Unverifiable receipt** (tampered manifest, deleted snapshot, snapshot of another conversation linked): non-success run, `conversations_errors = 1`, checkpoint unchanged, no provider call.
- **Classification** uses the same decision (analyze → skip → late message → analyze).
- **Failures keep the checkpoint:** provider error, result-save failure (invalid evidence), cancellation.
- **Terminal write failure:** a MySQL trigger failing the job update, and the job moving to another tenant during the run (tenant-scoped write target missing; a delete is blocked by the `job_runs` foreign key) → run returns an error, stored run not `success`/`running`, checkpoint unchanged (trigger case), **no notification**; control: success and exactly one notification.
- **Other modes unchanged:** positive limit excludes analyzed conversations and keeps the checkpoint; unanalyzed-only still skips an analyzed conversation with a new message (F05 residual); full rerun repeats analysis; an out-of-range date window selects nothing.
- **Scheduler paths (real lookup/routing/writes, synthetic provider):** cron `runScheduledJob` ×3 (analyze, late old conversation, skip); after-sync routing to another channel/tenant starts no run (checked synchronously), two after-sync invocations awaited through the completion hook (late conversation analyzed, then a successful run with no call).
- Existing regressions kept unchanged and passing: `TestKhongDanhGiaLaiCuocChatCu`, `TestDanhGiaLaiKhiCoTinNhanMoi`, `TestChayBiCatGhiPartialVaGiuMocQuet`, `TestChayTheoLichDocLaiJobTuDB`, R002/R004 snapshot/provenance tests (including sentinel `TestSingleAndBatchShareSnapshotContract`), the whole engine package.

## Non-vacuity

- **Pre-BUILD source** (`git show HEAD:` for `analyzer.go` and `scheduler.go`, with only the test seams patched in — `providerOverride`, `newScheduledAnalyzer`, `afterSyncJobFinished` — `analyzer_incremental.go` removed and a shim defining the other seams and `isOrdinaryIncremental=false`; all restored and byte-compared): **10 of 12** new tests failed. `TestOrdinaryRunReceiptsAreJobBound` and `TestNonOrdinaryModesKeepTheirContracts` pass on old code by design (old selection is already job-scoped; the second asserts unchanged behavior).
- **Mutations of the new code** (disposable MySQL, `-run 'Ordinary|NonOrdinary|ScheduledEntryPoints'`): finish-time checkpoint (2 tests fail), event-time candidate filter (8), event-time snapshot window (8), any evaluation suppresses / result-time style (receipts test), provenance bypassed (unverifiable test), always-changed duplicate replay (8), receipt not job-bound (receipts test), terminal failure ignored (terminal-write test) — **8/8 detected**. Two of my mutation edits first failed to compile (an unused variable/import after the edit); I corrected the mutations, not the tests, and re-ran them. All temporary edits were restored and byte-compared (`restored-identical`).

## Gates (repo root)

| Check | Result |
|---|---|
| Whole engine package before adding the new tests: `scripts/test-backend.ps1 -Packages ./engine` | `ok` (existing behavior preserved) |
| Focused: `scripts/test-backend.ps1 -Packages ./engine -Run 'Ordinary\|NonOrdinary\|ScheduledEntryPoints\|KhongDanhGia\|DanhGiaLai\|ChayBiCat\|ChayTheoLich\|SingleAndBatch' -VerboseTests` | all PASS |
| Full: `go test ./... -json -count=1 -p 1` on disposable MySQL | exit 0; 586 pass / 0 fail / 2 optional skips (572 pass but 14 DB-unavailable skips in the first run, see below) |
| R019 gate on the full JSON log | `GATE PASSED`, 0 invalid records (306813 events), five sentinels PASS, 0 DB-unavailable skips; optional skips `pricing.TestFetchThatTuNguonNgoai`, `storage.TestNoiCatS3` |
| `go build ./...`, `go vet ./engine` | OK |
| `git diff --check` | clean (CRLF warnings only) |
| Cleanup | disposable MySQL containers/networks removed after every run; persistent Compose DB never used |

**First full run failed the R019 gate (kept for the record):** 572 pass / 0 fail but 14 tests skipped as DB-unavailable (`Error 1040 Too many connections`): my new fixture reconnected without closing the previous pool, exhausting the disposable server's connection limit for later tests (the same trap noted in R020). The gate caught it as designed. I added `db.Close()` before `connectTestDB` in `setupIncFixture` and re-ran the whole suite (the numbers in the table). A later re-run was interrupted by the session ending and left a disposable MySQL container and network (`ccma-test-*-72a71e05`) behind; I removed them and confirmed `docker ps -a` / `docker network ls` show none.

The wrapper's two non-JSON lines were stripped before replaying the Go JSON through the R019 gate, as in R021–R024; the committed script is unchanged. Preflight runs with `PYTHONDONTWRITEBYTECODE=1` and an explicit `--files` list of every committed path; with the untracked `knowledge/_index.json` present a default whole-worktree preflight is **not** a PASS, and none is claimed.

## Behavior changes and limits for the reviewer

1. Ordinary runs (manual ordinary, cron, after-sync) now examine **every** conversation of the tenant's input channels and build a **full local snapshot** for each on every run; the first run after deployment analyzes historical never-analyzed conversations and re-analyzes conversations whose last receipt was windowed, legacy (no snapshot) or of a different source. This raises local DB/read work every run and may raise the first run's provider work. No candidate cap was added.
2. `conversations_found` in the run summary now means "conversations that needed work" (changed + errored) for ordinary runs, not "conversations in the time window".
3. The checkpoint (`last_run_at`) is the scan start, informational for ordinary runs (selection no longer depends on it); `finished_at`/`updated_at` record completion. The checkpoint and terminal run status are written together, tenant-scoped; a write failure now returns an error and suppresses the notification.
4. **Not covered / residuals:** candidate enumeration is not a global transactional snapshot (rows inserted after discovery are handled next run); only fields in the existing canonical snapshot count as source changes (raw-data-only changes do not); rule changes still need an explicit full rerun; F05 mode semantics (unanalyzed/since-last/date/limit) and F06 cancellation/ownership races are unchanged and OPEN; synthetic provider and disposable MySQL prove local scheduling/bookkeeping only — no provider correctness, upstream completeness or CVF AI-governance claim.

## Changed set

`backend/engine/analyzer.go`, new `backend/engine/analyzer_incremental.go`, `backend/engine/scheduler.go`, new `backend/engine/analyzer_incremental_test.go`, new `backend/engine/scheduler_incremental_test.go`, this evidence, work-order status line, roadmap F03 row, handoff, active state, session memory (marker, current-tranche prose and BUILD paragraph), tranche record, implementation status.

## Disposition

`REVIEW_PENDING` for Codex independent REVIEW. Claude does not self-approve; no push, merge, deployment or FREEZE.
