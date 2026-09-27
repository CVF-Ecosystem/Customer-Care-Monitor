# REPAIR evidence: CCMAI-RUNTIME-002 Gate B, owner-authorized escalation repair

**Work order:** `CCMAI-RUNTIME-002` Gate B, "Owner-authorized narrow repair after
review-cost escalation" · **Authority:**
`docs/reviews/CCMAI_RUNTIME_002_GATE_B_REREVIEW_ROUND3_2026-09-27.md`
(`CHANGES_REQUIRED / REVIEW_COST_ESCALATION_REQUIRED`, target repair commit
`7a284b8`), narrowed by owner disposition packaged into
`docs/work_orders/CCMAI_RUNTIME_002.md` lines 125-157 · **Ngày:** 2026-09-27 ·
**Repairer:** Claude (`REPAIR_WORKER`, scope limited to `ResetDemoData`
atomicity only — no other deletion path, schema, provider/runtime, S2/S3/S5 or
FREEZE authority) · **Kết quả:** repair applied, `REVIEW_PENDING` for Codex
re-review.

## Root cause (R3-E1)

`ResetDemoData` (`backend/api/handlers/demo.go`) used a hand-rolled
`tx := db.DB.Begin()` / `tx.Commit()` pair. Every delete/update statement in
between (`Message`, `JobResult`, `AnalysisSnapshot`, `AIUsageLog`,
`NotificationLog`, `ActivityLog`, `JobRun`, `Job`, `Conversation`,
`AppSetting`, `Channel`, and the final tenant-settings update clearing the
demo flag) was called without checking `.Error`. In MySQL/InnoDB, one failed
statement inside a transaction does not automatically abort it — later
statements still run and `Commit()` can still succeed. Codex's re-review
reproduced this with a `BEFORE DELETE` trigger forcing a `job_results` delete
to fail: the handler still deleted `job_runs`/`conversations`, cleared the
demo flag, and returned HTTP 200, leaving an orphaned `job_result` behind
already-deleted parents.

## What changed

**`backend/api/handlers/demo.go` — `ResetDemoData`:** the two parent-locking
reads and the eleven delete statements plus the tenant-settings update all now
run inside a single `db.DB.Transaction(func(tx *gorm.DB) error { ... })`
closure — the same pattern already used by `DeleteChannel` and
`PurgeChannelConversations`. Every statement's `.Error` is checked and wrapped
with `fmt.Errorf(...)` context; any non-nil error returned from the closure
triggers `gorm`'s automatic rollback, so no statement's effect can survive a
later statement's failure. On success the closure returns `nil` and the
transaction commits as before. On failure the handler logs the wrapped error
and returns `500 {"error":"reset_demo_data_failed"}` instead of `200`.

No other deletion path, schema, model, or migration was touched. `ClearJobResults`,
`DeleteChannel`, `PurgeChannelConversations`, `DeleteJob`, and `ClearJobRuns`
were not in scope for this narrow repair and were not modified.

## New regression tests

New file `backend/api/handlers/demo_test.go` (package `handlers`, reusing the
existing `connectChannelsTestDB` helper from `channels_test.go`):

- **`TestResetDemoDataHappyPathClearsAllTenantData`** — seeds one tenant
  (`is_demo_data:true`), channel, conversation, message, job, job run, and job
  result; calls `ResetDemoData` directly through the handler; asserts `200`,
  every seeded row gone, and the tenant's `settings` decodes to `{}`. This is
  the first unit test for the handler's ordinary success path — proves the
  rewrite to `db.DB.Transaction` did not change success-path behavior.
- **`TestResetDemoDataFailureRollsBackEverything`** — the permanent regression
  test for R3-E1, using the same `BEFORE DELETE` trigger technique Codex used
  to reproduce the defect: a forced `job_results` delete failure via a MySQL
  trigger matching one seeded row's ID. Asserts the response is non-2xx, and
  that the channel, conversation, job, job run, and job result all still exist
  by ID afterward, and that the tenant's `settings` still decodes
  `is_demo_data: true`. (The tenant-settings equality check compares the
  decoded JSON value rather than the raw string because MySQL's `json` column
  type re-serializes stored text, e.g. inserting a space after `:`, which is
  unrelated to the atomicity behavior under test.)

Both tests skip with a message when `TEST_DB_DSN` is unset, matching every
other DB-backed test in this package.

## Verification

- `go build ./...`, `go vet ./...` — clean.
- Disposable `mysql:8.0` container on an isolated Docker network (no host
  data, removed after the run; persistent Compose `ccma` stack untouched
  throughout except for the controlled rebuild/restart below);
  `log_bin_trust_function_creators` set once via root so the trigger-based
  failure-injection test can create its trigger, matching the same setting
  used for the round-2/round-3 trigger tests:
  - `go test ./api/handlers -run TestResetDemoData -v` — both new tests PASS.
  - `go test ./engine/... ./api/handlers/... ./cli/... -run 'TestSaveResultsFailsWhenParentDeletedFirst|TestWriterHoldsParentLockDeleteWaitsThenCleansEvidence|TestDeleteChannelFailureRollsBackWholeCascade|TestDeleteChannelRemovesResultsAndSnapshotsTogether|TestPurgeChannelConversationsRemovesEvidenceKeepsChannel|TestDeleteJobRemovesRunsAndEvidence|TestApplyPrunePlanCleansOrphanSnapshotsKeepsReferenced|TestSnapshotDigestChangesWithAttachmentIdentity' -v` — every named round-1/round-2/round-3 regression and both race-ordering tests PASS unchanged.
  - `go test ./... -count=1` — all 13 packages `ok`.
  - `AutoMigrate` run twice back-to-back against the same disposable `CCMA`
    schema — both clean, no error, no duplicate-index or schema drift (no
    schema/model change was made this repair, so this reconfirms the existing
    baseline).
- `gofmt -l`: `demo.go`'s only diff (once CRLF is normalized to LF for
  comparison) is pre-existing struct-literal alignment inside
  `buildDemoTemplates`'/`ImportDemoData`'s seed data (lines 109-280),
  unrelated to and disjoint from this repair's changed lines (import block and
  `ResetDemoData`, lines 3-14 and 424-493 per `git diff`); confirmed via
  `git diff` hunk ranges. `demo_test.go` is gofmt-clean.
- `git diff --check` on the changed set — clean, no trailing whitespace.
- One transient side effect during verification: running `go run` with
  `GOFLAGS=-mod=mod` inside the test container (for a throwaway AutoMigrate
  driver program) caused `go.mod` to reclassify `github.com/minio/minio-go/v7`
  from indirect to direct. This was unrelated to the repair, outside the
  authorized scope, and was reverted with `git checkout -- backend/go.mod`
  before finishing; the full test suite was re-run afterward without
  `-mod=mod` to confirm `go.mod`/`go.sum` stay untouched (`git status` clean
  on both files).
- Persistent Compose `ccma`: `docker compose -p ccma build app` succeeded with
  this repair's source; `docker compose -p ccma up -d app` recreated only the
  `app` container (the `db` container and its volume were never touched,
  reset, or recreated). Post-restart logs show `Database connected
  successfully`, `Database migration completed`, `0 cron jobs`, static
  pricing (sync still opt-in/off) — no error. `channels`/`tenants`/`job_results`
  row counts on the persistent `CCMA` schema remained `0` before and after,
  confirming the disposable test container and the persistent volume never
  mixed data.
- Downstream catalog check (`scripts/manage_cvf_downstream_catalog.ps1
  -Check`) — PASS.
- CVF workspace doctor
  (`../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1
  -ProjectPath .`) — PASS 25/25.
- No frontend or docs files were touched this repair, so the frontend build
  was not re-run (same reasoning as round 3's evidence).

## Claim boundary

No Claude/Gemini/OpenAI/xAI or other provider API was called and no API key
was used. No channel sync, real customer data, deploy, or push. This repair
closes only the R3-E1 finding named in the owner's narrow authorization — it
does not reopen, re-certify, or re-scope any other part of Gate B, and it does
not authorize S2/S3/S5, FREEZE, or any AI-runtime-governance claim. The new
tests prove MySQL/InnoDB transaction-rollback behavior under this schema and
this code; they are not a claim about any other database engine or about
production load. Local commit only; Codex re-reviews the changed set and this
evidence next.
