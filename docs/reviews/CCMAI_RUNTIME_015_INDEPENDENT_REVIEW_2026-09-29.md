# CCMAI-RUNTIME-015 independent review

**Date:** 2026-09-29 · **Reviewer:** Claude, independent of Codex's BUILD · **Build commit:** `de0616c` (parent `60ed988`) · **Disposition:** `CHANGES_REQUIRED` · **Risk:** R2.

## Scope and continuity

I rehydrated the manifest, policy, active state and active handoff. The worktree was clean at `de0616c` and the pinned core is `26c686c`, which matches the manifest. The role transition `COMMIT_STEWARD (Codex) → REVIEWER (Claude)` was recorded in the active handoff before review. Codex is the builder here because my tools were unavailable at BUILD time; that route change is recorded in the handoff, so this review is independent. The production diff touches only `backend/engine/sync.go` and `backend/channels/zalo_oa.go`, and the test changes stay under `backend/engine/` and `backend/channels/`. Both are within the work order's allowed paths.

## What holds

- **The lock covers every run-owned DB write.** `withOwnedSyncWrite` opens a silent transaction and runs `SELECT … FOR UPDATE` on the `(id, tenant_id)` channel row. It then compares `syncing` and `sync_run_id`, and performs the write before commit. The following writes all go through it:
  - conversation create and update;
  - message create and replay merge (`updateExistingMessage` now takes `tx`);
  - the tenant-scoped message-count update;
  - the refreshed Zalo credential write, which also repeats the run-ID predicate and requires exactly one row.

  Terminal writes stay single-statement and run-ID fenced, as in R014. The only statements outside the lock are reads: the prior-attachment lookup and the cleanup reference count. No run-owned write escapes the guard.
- **Row-lock serialization.** `TestOwnedWriteSerializesWithOwnershipChange` holds an owned transaction open, and an ownership `UPDATE` on the same row blocks until it commits. I confirmed the detector independently. In a throwaway git worktree of `de0616c`, I removed `clause.Locking{Strength: "UPDATE"}` and the test failed with "ownership changed before the owned write committed". I then removed the worktree; the reviewed tree was never modified.
- **Two-generation races.** `TestStaleRunCannotPersistConversationMessageCountOrCredentials` shows that a released-and-re-reserved channel rejects the old reservation's conversation, message, count and credential writes. B's message, count, credentials and run ID stay unchanged. The R014 `TestStaleWorkerCannotFinishOverNewerRun` still passes in all four modes. Forged ID, wrong tenant and empty ID are all rejected.
- **Attachment keys.** A new download gets `<tenant>/<conversation>/<uuid>/<name>`, so the tenant-first `/api/v1/files` contract holds. A same-identity stored object is reused, and so is a legacy `<tenant>/<conversation>/<name>` object. `TestInFlightOldAttachmentCannotReplaceNewRunObject` shows that a late old download gets its own key and cannot publish after takeover. B's bytes and published path stay correct.
- **Cleanup when the commit result is uncertain.** `cleanupAttemptKeys` deletes a new attempt key only after a successful read proves no message references it. On a read failure it keeps the object. It never deletes a referenced object (tested) or a previously stored key.
- **Zalo.** The callback now returns an error, and `refreshToken` returns before replacing the in-memory tokens. Both `FetchRecentConversations` and `FetchMessages` return the `%w` chain unchanged, so `errors.Is` sees `ErrSyncOwnershipLost` or `ErrSyncWriteFailed` and the engine stops the run. The synthetic transport test shows retry stops after exactly two requests. Error texts are bounded (`sync write failed`, `sync_ownership_lost`) and carry no token. The guarded transaction uses the silent session, and the DB-error test checks the GORM sink for the run ID and trigger text.
- **Loop fail-fast.** Every run-owned error class returns immediately rather than becoming a partial result. The deferred error transition is skipped for ownership loss (no row to own) and kept for `ErrSyncWriteFailed`.

## Independent checks

- Focused: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Run 'Stale|Owned|Ownership|Attachment|Zalo|Upsert|Reserve|SyncChannel|SyncRun|SyncReserved|SyncAll|Scheduler|RecordSync|Sink|BoundedClasses|RunManual|HandleManual|AgentSync|SyncTerminal|KhoChinh' -VerboseTests` → exit 0.
  - 105 PASS, 0 SKIP.
  - `api/handlers`, `channels`, `db` and `engine` all `ok`.
  - The script removed its disposable MySQL container and network (`ccma-test` container and network lists are empty).
- Mutation, independent: removing `FOR UPDATE` fails the lock test, as described above. I did not repeat the builder's attempt-key mutation; the builder's evidence for it is consistent with the test's key-inequality assertion.
- Catalog `-Check` → PASS. Doctor `check_cvf_workspace_agent_enforcement.ps1` → `RESULT: PASS (25/25)`.
- I did not rerun the full backend suite. The builder records 14 packages `ok`; my focused run covers every package the diff touches.
- No real channel/provider call, persistent Compose DB use or governance claim.

## Blocking finding R015-R1 — storage `Exists` error aborts the whole run and bypasses the local fallback

In `downloadAttachments`, a reused identity calls `store.Exists(ctx, oldKey)`, and any error returns `newKeys, ErrSyncWriteFailed`. The engine treats `ErrSyncWriteFailed` as run-fatal, so it stops the channel, abandons every remaining conversation and message, and records the deferred error.

For an S3 tenant, `storage.fallbackStore.Exists` returns the primary's error without consulting the local store (`backend/storage/fallback.go`). So an S3 outage aborts the whole channel sync at the first replayed message with a stored attachment. That outage is exactly the case the local fallback exists for. Before R015, this outage would still download to local disk and continue.

This breaks work order requirement 2, "Preserve local/S3 fallback behavior". It also labels a storage availability error with the DB-failure class. No test covers an `Exists` error.

**Required:** an `Exists` error must not end the run. Either treat it as "reuse not confirmed" and take the normal fresh-download path (whose `Put` falls back to local), or record a per-attachment failure. Keep a bounded, non-DB class. Add a synthetic store test in which `Exists` errors: the run continues and later messages are still persisted. Keep every ownership guard unchanged.

## Required finding R015-R2 — no end-to-end mid-loop ownership-loss test

The SPEC asks to "pause A before each category of local write", resume after a takeover, and show that A stops rather than continuing with other conversations. The new stale tests call the write helpers directly with an old reservation. The R014 generation test drives `SyncReservedChannel`, but it does not assert that A created no conversation or message. The adjusted R013 test only covers loss before the first write.

**Required:** add one `SyncReservedChannel` test with at least two synthetic conversations. The takeover happens after the first conversation's writes. Assert:
- no further conversation, message or count rows from A;
- the error class is `sync_ownership_lost`, not a partial status;
- B's row, status and run ID are unchanged.

This is test coverage only; the source behavior looks right.

## Non-blocking observations

- **O1: message-count update can hit zero rows.** `updateOwnedMessageCount` requires `RowsAffected == 1`, and MySQL reports changed rows. This relies on GORM adding `updated_at`. An unchanged count written in the same millisecond as the preceding conversation upsert would report 0 rows. That becomes `ErrSyncWriteFailed` and aborts the run. It is unlikely with real network fetches between the two writes, and I could not observe the statement because the session is silent. Consider treating 0 affected rows as success when the conversation row is known to exist inside the same locked transaction.
- **O2: `JSON_SEARCH` wildcards.** In the cleanup reference check, `JSON_SEARCH` treats `%` and `_` in filenames as wildcards. That can cause false-positive references, which only leave an orphan (the safe direction).
- **O3: Zalo rotation limit.** After a failed token persist, the adapter keeps the old tokens in memory. The run stops, so this is harmless. The single-use rotation risk is disclosed in the BUILD evidence.

## Disposition

`CHANGES_REQUIRED` within R015's allowed paths. The WORK_ORDER_AUTHOR (Codex) should record a repair addendum for R015-R1 and R015-R2. The repair returns one local commit as `REVIEW_PENDING` for re-review. No self-approval, push, FREEZE, reclaim or live provider/channel call. S1 remains IN_PROGRESS.
