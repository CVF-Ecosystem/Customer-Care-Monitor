# CCMAI-RUNTIME-025 / F03 — independent review

Date: 2026-10-02 (Asia/Saigon). Reviewer: Codex; implementation worker: Claude. Risk R2. Result: REVIEW_PASS with bounded reviewer repairs / FREEZE_OPEN.

## Exact source and authority

- Dispatcher seed/base: `517406f1bfafbfe48b2cef31b558a76b8e67d8c0`; planning `d8ec254e6b83afeaed51014755979749055485d8`.
- Reviewed Claude BUILD: `b53f56448962d3f7c6aa070e0882560160a90c2e`, 13 changed files within the order. Immutable seed unchanged and committed before BUILD.
- Intake correction: `7aae8ac3b90576253ad1bea50cb4dcd185e2518a`. Implementation-status prose still claimed F03-F07 source untouched; narrowed to F04-F07 before source review. Machine pointers and memory current prose already said REVIEW_PENDING. Missing compact bootstrap recorded as nonblocking BOOTSTRAP_MIGRATION_PENDING; doctor 25/25 PASS against public core `26c686cc99b8be965d2760f27fe875b03376c643`.
- Entry worktree: only pre-existing untracked `knowledge/_index.json`, excluded. Reviewer changes are explicitly enumerated below and committed with this review; original BUILD is not represented as passing without repair.
- Authority: [SPEC](../specs/RUNTIME_ANALYZER_INCREMENTAL_COVERAGE_F03_2026-10-01.md), [work order](../work_orders/CCMAI_RUNTIME_025.md), dispatcher seed and tranche record. Small same-scope reviewer repairs explicitly authorized. No provider/channel calls, persistent DB, push, deployment, parent-CVF implementation/test or FREEZE.

## Contract review

| Area | Independent conclusion and evidence |
| --- | --- |
| Mode boundary | `isOrdinaryIncremental` covers only unlimited ordinary mode. Full/date/positive limit/explicit since/unanalyzed keep the old selection/finalization branch; F05 remains separate. |
| Source selection | Candidates are all conversations in the job's tenant/input channels, ordered deterministically. Latest tenant/job/conversation evaluation is selected via job-run scope; no receipt/legacy is eligible, linked receipt requires snapshot provenance and digest equality. Event time and result time no longer suppress changed ordinary source. |
| Shared snapshots | Preparation loads full local messages and reuses the exact snapshot in single/batch analysis and saving. No snapshot schema/format change. Unchanged source is skipped; malformed linked receipts/query failures remain errors. |
| Scheduling/checkpoint | Cron and after-sync enter the same real analyzer with synthetic-provider seams. Error-free complete scans use second-truncated scan-start checkpoint. Terminal job/run writes are tenant scoped, row locked and transactional. Two gaps in completion/persistence were found and repaired below. |
| Workload/count | Accepted behavior: every ordinary scan loads full snapshots; initial historical/legacy work may increase with no candidate budget. `conversations_found` counts changed/error conversations needing work rather than window population. This is specified, not a performance guarantee. |
| Limits | Enumeration is not one global transaction snapshot; inserts after enumeration wait for the next scan. Only canonical snapshot fields drive source changes. Rule changes still require manual full rerun. No real upstream coverage, provider quality or CVF AI governance claim. F02 and F05/F06 remain OPEN independently. |

## Independent failures and bounded reviewer repairs

The following retained probes ran against BUILD product source on disposable MySQL and failed (negative run exit 1, engine 4.332 s):

1. `TestOrdinaryReviewCancellationAtCompletion`: all four single/batch × empty/nonempty cases reported success and advanced checkpoint after cancellation. Empty cases were canceled before scanning; nonempty cases used a provider double that canceled at the final response and returned a valid result. BUILD never rechecked context at the terminal boundary.
2. `TestOrdinaryReviewSuppressedCheckpointWrite`: a tenant/job-targeted BEFORE UPDATE trigger retained the old checkpoint/status/time without returning a SQL error. BUILD accepted success with nil error. Checking only SQL error and RowsAffected > 1 did not prove intended terminal values were persisted.

Reviewer-local repair assessment: same objective, acceptance contract, authorized engine paths, R2 and external-effect class; no provider/credential use, schema/interface redesign or F06 registry work. Codex acts as REVIEWER performing bounded repairs, then SESSION_SYNC_STEWARD/COMMIT_STEWARD/ORCHESTRATOR; Claude's BUILD attribution and independence remain explicit. No extra REWORK tranche.

- `backend/engine/analyzer.go`: ordinary completion checks `ctx.Err()` before deciding success/checkpoint, covering empty scans and the final provider response.
- `backend/engine/analyzer_incremental.go`: while both rows remain locked, read back all intended terminal values, including JSON summary and optional checkpoint. A suppressed write fails and rolls back; an already-identical zero-change write remains valid. Completion timestamps are normalized to datetime(3) milliseconds for exact comparison; checkpoint remains seconds.
- `backend/engine/analyzer_review_test.go`: retains both negative probes and a same-second replay positive control, including stored completion >= stored start. This guards legitimate MySQL no-op writes and prevents completion rounding backward to the checkpoint's seconds.

## Executed evidence and attribution

Independent commands:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'OrdinaryReview' -VerboseTests
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'Ordinary|NonOrdinary|ScheduledEntryPoints|KhongDanhGia|DanhGiaLai|ChayBiCat|ChayTheoLich|SingleAndBatch|Snapshot' -VerboseTests
```

First post-repair focused run: 46 top-level PASS, engine 38.498 s, exit 0. After adding stored timestamp assertions, one attempt failed at compilation because the reviewer used an incorrect models import; fixed to `backend/db/models`, without changing assertions. Final corrected-source run: 46 top-level PASS, engine 38.441 s, exit 0; cancellation/suppressed-write negatives and same-second timestamp positive all pass. Each wrapper creates and removes its disposable MySQL/network; existing app/database services are not targets. Providers are synthetic and notifications are suppressed/observed by the fixtures.

Local logs (not repository artifacts): `%TEMP%/ccmai-r025-review-negative.log`, `ccmai-r025-review-positive.log`, `ccmai-r025-review-final.log` (compile failure), `ccmai-r025-review-final-corrected.log`.

Host `go build ./...` passed after the initial repairs. Docs build initially passed in 24.65 s; final docs build passed in 24.14 s; explicit preflight passed 7/7 including catalog and diff check passed. Gate unit tests independently passed 46/46 (36.041 s). Claude's [BUILD evidence](RUNTIME_ANALYZER_INCREMENTAL_COVERAGE_F03_BUILD_2026-10-01.md) reports full backend 586 pass / 0 fail / 2 optional skips, R019 sentinels, 10/12 old-source failures and 8/8 mutations. Those are worker evidence inspected here, not an independent rerun of the entire backend/R019/mutation suite.

## Disposition

REVIEW_PASS for source-level F03 local scheduling/bookkeeping after BUILD plus the explicitly attributed reviewer repairs; original BUILD alone failed the independent probes. Final corrected-source tests pass; repository preflight passed 7/7 before commit. Next: bounded F04 DESIGN/SPEC/WORK_ORDER. FREEZE, deployment and broader live claims remain open. Draft PR #1 at `3e0b37ea729c75fe1a319dda61ee384b71706e7c` provides historical F08/GOV hosted evidence, not CI evidence for these later local changes.

Repository verification: first review preflight failed 6/7 because the reviewer appended a descriptive history token instead of the literal current status; corrected to `REVIEW_PASS`. Final explicit 12-file preflight (including every review change) passed all seven gates. The unrelated untracked `knowledge/_index.json` remains excluded; this is not a whole-worktree PASS. Catalog families already cover the new review, so generated index/catalog require no change.
