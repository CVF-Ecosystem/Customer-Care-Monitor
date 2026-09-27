# REPAIR evidence: CCMAI-RUNTIME-002 Gate B, round 3

**Work order:** `CCMAI-RUNTIME-002` Gate B, "Repair round 3 — R2-RR3 writer/deletion dependency edge" · **Authority:** `docs/reviews/CCMAI_RUNTIME_002_GATE_B_REREVIEW_ROUND2_2026-09-27.md` (`CHANGES_REQUIRED_ROUND_3`, target commit `a17b50a`) · **Ngày:** 2026-09-27 · **Repairer:** Claude (`REPAIR_WORKER`, same objective/R2-risk/external-effect/commit boundary as Gate B — new independent root cause, not a further round on R2-B1/B2 or R2-RR1/RR2) · **Kết quả:** repairs applied, REVIEW_PENDING for Codex re-review.

Round 2 closed R2-RR1 (channel-cascade transaction atomicity) and R2-RR2 (attachment-fingerprint collision). Codex's re-review of that round found a new, independent root cause at the writer/deletion dependency edge: `saveResults` never locked or confirmed its parent `Conversation`/`JobRun` before committing evidence, and neither table has a foreign key to them, so an analysis in flight could commit `analysis_snapshots`/`job_results` referencing a conversation or job run that a concurrent delete had already removed.

## Root cause (R2-RR3)

`saveResults` (`backend/engine/analyzer.go`) opened its transaction and went straight to `tx.Create(&snapshotRow)` — no read, lock, or existence check on the `Conversation` or `JobRun` the evidence is about to reference. Round 2's `DeleteChannel` fix locks conversations before deleting their children, but that only protects against **other conversation deletes** contending for the same lock; it does nothing to stop a **write** into `analysis_snapshots`/`job_results` from an unrelated transaction that never asked for that lock. A schema probe during review confirmed inserting a snapshot for a nonexistent tenant/run/conversation succeeds (no FK), so the missing check was the entire gap.

## Required implementation outcome — what changed

### 1. Writer now locks and confirms both parents before writing evidence

**`backend/engine/analyzer.go` — `saveResults`:** right after `tx := db.DB.Begin()`, before creating anything, it now does two `tx.Clauses(clause.Locking{Strength: "UPDATE"})` reads — one on `Conversation` by `(id, tenant_id)`, one on `JobRun` by `(id, tenant_id)`. Either row missing returns an error immediately (transaction rolls back via the existing `defer tx.Rollback()`); the caller already counts a `saveResults` error as a per-conversation error, so this fails closed with no orphan.

### 2. Every parent-deletion path audited and given the same lock-first protocol

The locking check only closes the race if every path that deletes a `Conversation` or `JobRun` takes a **conflicting** lock on that same row **before** touching its children — otherwise the writer's check would pass, the delete would proceed unlocked, and the writer's insert would land after the delete's cleanup already ran.

| Path | Deletes | Change |
|---|---|---|
| `DeleteChannel` (`channels.go`) | Conversation | Already locks conversations first as of round 2 (R2-RR1) — no change needed. Audited: its `FOR UPDATE` read is on the same `Conversation` primary-key rows `saveResults` now locks, so it correctly serializes against the writer. |
| `PurgeChannelConversations` (`channels.go`) | Conversation | Was not transactional and read conversation IDs with a plain (non-locking) `Pluck` before deleting anything. Rewritten to the same pattern as `DeleteChannel`: one transaction, `FOR UPDATE` `Find` on conversations first, then children (`job_results`, `analysis_snapshots`, `messages`), then the conversations, then the channel's sync-state reset; attachment-file cleanup moved to after commit (same reasoning as R2-RR1). |
| `DeleteJob` (`jobs.go`) | JobRun | Was not transactional; `runIDs` read via plain `Pluck`. Rewritten: one transaction, `FOR UPDATE` `Find` on `job_runs` first, then `job_results`/`analysis_snapshots`, then `job_runs`, `ai_usage_logs`, `notification_logs`, the `Job` row itself. |
| `ClearJobRuns` (`jobs.go`) | JobRun | Same shape and same fix as `DeleteJob`, keeping the `Job` row and resetting `last_run_at`/`last_run_status` inside the same transaction. |
| `ClearJobResults` (`jobs.go`) | **Neither** | Deletes `job_results`/`analysis_snapshots`/`ai_usage_logs`/`notification_logs` for a job but **keeps every `job_run` row**. Since it never deletes a `Conversation` or `JobRun`, it cannot produce the "evidence outlives its parent" defect this round closes — a result written here always still has a live parent. No change; documented per the audit requirement rather than silently skipped. |
| Demo reset (`ResetDemoData`, `demo.go`) | Conversation, JobRun | Already ran in one transaction, but issued its deletes without locking the parents first. Added two `FOR UPDATE` `Find` calls (conversations, then job runs, scoped to the tenant) immediately after `tx.Begin()`, before any delete — same lock-first ordering as every other path, so `saveResults` correctly waits it out or fails against it too. |

### 3. Legacy compatibility preserved

No schema or migration change was made this round — `job_results.analysis_snapshot_id` stays nullable and the existing `AfterFind`-derived `evidence_status` (`snapshot_bound` / `legacy_unverified`) is untouched, so pre-existing legacy rows keep reading exactly as before (`TestLegacyResultsAreMarkedUnverified`, unchanged, still passes). The fix is enforced purely by locking discipline plus the pre-existing `messages → conversations` foreign key; it does not add a new constraint and therefore has no backfill or invalid-row case to handle.

### 4. DB-backed concurrency tests, both orderings

New tests in `backend/engine/snapshot_db_test.go`:

- **`TestSaveResultsFailsWhenParentDeletedFirst`** ("delete wins"): the conversation is deleted and committed *before* `saveResults` is called. `saveResults` must fail and leave zero snapshot/result rows. PASS.
- **`TestWriterHoldsParentLockDeleteWaitsThenCleansEvidence`** ("writer wins"): a goroutine reproduces `saveResults`' own locking prefix (`FOR UPDATE` on the conversation and job run), signals once it holds the lock, then blocks on a channel before inserting and committing. A second goroutine runs the same lock-first delete cascade `DeleteChannel` uses (`deleteConversationLockingChildrenFirst`, a small test helper mirroring that handler's exact lock/delete-children/delete-parent order). The test asserts the delete has **not** completed within 300ms of the writer holding its lock (proving MySQL actually blocked it), then releases the writer to commit, then asserts the delete completes and removes every row the writer just wrote (`job_results`, `analysis_snapshots`, the conversation itself) — zero survivors. PASS.

Both tests run against a real disposable MySQL container (InnoDB row locking, not a mock), so they exercise the actual blocking/wait semantics the fix depends on, not a simulated one.

### 5. Regression coverage

- All 5 pre-existing Gate B DB tests, both round-1 and round-2 tests: unchanged, still PASS.
- `TestSaveResultsRollsBackSnapshotOnFailure` needed one fixture change: it calls `saveResults` directly with a synthetic `runID` that was never a real `job_runs` row. Since `saveResults` now requires that row to exist, the fixture inserts one before the calls it makes. Behavior under test (oversized-field insert failure rolls back the whole transaction) is unchanged and still PASS.
- Two new happy-path regression tests for the two most substantially rewritten handlers: `TestPurgeChannelConversationsRemovesEvidenceKeepsChannel` (`backend/api/handlers/channels_test.go`) and `TestDeleteJobRemovesRunsAndEvidence` (new file `backend/api/handlers/jobs_test.go`) — both confirm the transactional rewrite still deletes exactly what it should and nothing it shouldn't under normal (non-racing) operation.

## Verification

- `go build ./...`, `go vet ./...` — clean.
- Disposable `mysql:8.0` container (`ccma-runtime002-repair3-test`, host port 33063, removed after the run; persistent Compose `ccma` stack untouched throughout; `log_bin_trust_function_creators` set once via root for the round-2 trigger-based test that still lives in the suite):
  - `go test ./engine -v` — all 34 tests PASS, including the two new R2-RR3 concurrency tests.
  - `go test ./api/handlers ./cli -v` — all tests PASS, including the two new round-3 happy-path tests and every round-1/round-2 regression test (`TestDeleteChannel*`, `TestApplyPrunePlan*`, `TestDonBanDanhGiaTrungGiuLuotMoiNhat`, `TestDonKhongDungToiTinNhanVaCuocChat`).
  - `go test ./... -count=1` — all 13 packages `ok`.
  - `AutoMigrate` run twice back-to-back against the same disposable `CCMA` schema — both PASS, no error, no duplicate-index or schema drift (no migration/model change was made this round, so this reconfirms the existing baseline rather than testing anything new).
- `gofmt -l` on every file touched this round, CRLF-normalized before comparison: `analyzer.go`, `snapshot_db_test.go`, `jobs.go`, `jobs_test.go`, `channels_test.go` clean; `channels.go` and `demo.go` each have only pre-existing, unrelated diffs (confirmed via `git diff -U0` that none of this round's added lines touch those spots — `channels.go`'s three spots are the same ones noted in rounds 1 and 2; `demo.go`'s are struct-literal alignment inside `buildDemoTemplates`' seed data, nowhere near `ResetDemoData`).
- Workspace doctor `check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .` — PASS 25/25.
- No frontend or docs files were touched this round.

## Claim boundary

No Claude/Gemini/OpenAI/xAI or other provider API was called and no API key was used. No channel sync, customer data, deploy or push. This round closes R2-RR3 only — it does not reopen, re-certify or re-scope any other part of Gate B, and it does not authorize S2/S3/S5, FREEZE, or any AI-runtime-governance claim. The concurrency tests prove MySQL InnoDB locking behavior under this schema and this code; they are not a claim about production load characteristics or about any other database engine. Local commit only; Codex re-reviews the changed set and this evidence next.
