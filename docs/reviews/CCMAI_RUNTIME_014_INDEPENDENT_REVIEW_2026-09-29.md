# CCMAI-RUNTIME-014 independent review

**Date:** 2026-09-29 · **Reviewer:** Codex, independent of Claude's BUILD · **Build commit:** `aa399c6` (parent `48245db`) · **Disposition:** `CHANGES_REQUIRED` · **Risk:** R2.

## Scope and continuity

The active state, memory, work order and BUILD evidence recorded `REVIEW_PENDING`; the active handoff header and `IMPLEMENTATION_STATUS.currentPhase` still said WORK_ORDER. At INTAKE, Codex reported `BLOCKED_CONTINUITY_DRIFT`, aligned only those stale summary fields with the committed BUILD record, then re-read current continuity. Role transition: `COMMIT_STEWARD (Claude) → REVIEWER (Codex)`.

The exact `48245db..aa399c6` source diff stays within the allowed model, engine and handler files. BUILD evidence names the actual parent and records full gates and four restored mutations. The nullable internal column, typed reservation, run-ID predicate on terminal/deferred/manual panic writes, and two-generation tests match the main final-write contract. Migration tests cover the live model's fresh schema and an old-schema table with the same new-column tag. Mixed-version and side-effect limits are disclosed.

## Blocking finding R014-R1 — run ID reaches SQL logs

The [SPEC](../specs/RUNTIME_SYNC_RUN_OWNERSHIP_S1_2026-09-29.md) requires logs and responses not to reveal the run ID or SQL detail. `ReserveChannelSync` puts the ID in the `UPDATE` values; `recordSyncStatus` and the manual panic write put it in the `WHERE` predicate. All three use the ordinary `db.DB` logger. `backend/db/mysql.go` configures GORM `logger.Default.LogMode(logger.Info)` outside production and `logger.Warn` in production. The installed GORM v1.31.1 default logger has `ParameterizedQueries=false`; its `Trace` prints the interpolated SQL at Info, and also for errors or slow queries at Warn. Therefore a run ID can be printed to stdout during normal development and on error/slow queries in production. The BUILD test `TestSyncFailureTextsNeverRevealTheRunID` captures the application's `log.Writer()`, while GORM's default writer is a separate `log.New(os.Stdout, ...)`; the test cannot observe this leak. The returned `recordSyncStatus` error also wraps the raw DB error, which can reach agent result errors or worker logs. These are contract failures even though final-write fencing works.

## Independent checks

- Doctor: `powershell -ExecutionPolicy Bypass -File ../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .` → PASS 25/25.
- Focused: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Run 'TestChannelSyncRunID|TestReserveChannelSync|TestSyncReservedChannel|TestRecordSyncStatus|TestSyncTerminalWrites|TestSyncFailureTexts|TestStaleWorker|TestSyncChannelNow|TestRunManualSync|TestStaleManualWorker|TestHandleManualSyncPanic|TestAgentSync|TestSchedulerSkipsBusy'` → exit 0; handler, db and engine packages passed on disposable MySQL; script removed its container/network. `git diff 48245db aa399c6 --check` → clean.
- A supplementary verbose log-capture shell command was rejected by automatic command review before execution. The finding rests on the local application logger configuration and installed GORM logger source, not a captured runtime transcript. No real channel/provider call or governance claim was made.

## Disposition

`CHANGES_REQUIRED`, within R014. The [work order repair addendum](../work_orders/CCMAI_RUNTIME_014.md) authorizes a narrow fix to SQL/result logging and a regression test that observes the actual GORM output sink. Claude returns one local repair commit as `REVIEW_PENDING` for independent re-review. No self-approval, push or FREEZE. S1 remains IN_PROGRESS; automatic recovery and in-flight side-effect fencing remain separate.
