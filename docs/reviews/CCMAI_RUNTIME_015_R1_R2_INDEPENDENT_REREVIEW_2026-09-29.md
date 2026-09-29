# CCMAI-RUNTIME-015 R1/R2 independent re-review

**Date:** 2026-09-29 · **Reviewer:** Claude, independent of Codex's repair · **Repair commit:** `904454d` (parent `43aeeb3`) · **Disposition:** `REVIEW_PASS`; FREEZE open · **Risk:** R2.

## Scope and continuity

I rehydrated the active state, active handoff, the work order's repair addendum and the repair evidence. The worktree was clean at `904454d`, and core `26c686c` matches the manifest. I recorded the role transition `COMMIT_STEWARD (Codex) → REVIEWER (Claude)` in the active handoff before reviewing.

The repair's production change is limited to `backend/engine/sync.go`. It adds a new focused test file, `backend/engine/sync_side_effect_repair_test.go`, and touches continuity and evidence files. All of these are within the R015 paths.

## Findings resolved

- **R015-R1 (storage probe error).** The branch `else if err != nil { return newKeys, ErrSyncWriteFailed }` after `store.Exists` was removed. An unconfirmed old object now falls through to a fresh attempt-unique download. That download uses `luuFileDinhKem`, whose primary `Put` falls back to local storage. `ErrSyncWriteFailed` again means only a failed owned DB write. The ownership check before each attachment is unchanged.
  - `TestSyncReservedChannelContinuesAfterOldAttachmentProbeError` runs a full `SyncReservedChannel`. Its synthetic store fails both `Exists` and primary `Put`. The test shows the run succeeds, the new `local_path` is published and exists on local disk, and a later message is persisted.
  - I confirmed this detector independently. In a throwaway worktree of `904454d`, I restored the removed branch, and the test failed with `sync write failed`. I then removed the worktree.
- **R015-R2 (mid-loop ownership loss).** `TestSyncReservedChannelStopsAfterFirstConversationLosesOwnership` uses a new per-engine `afterConversation` seam.
  - After A's first conversation, the fixture releases the row and reserves B. B writes a conversation, a message and a count.
  - A then returns `sync_ownership_lost`, not a partial status. It does not write its second conversation or message.
  - B's conversation count, `syncing` status and run ID are unchanged.

  Both new seams (`storeForTenant`, `afterConversation`) are nil in production, so the production path is unchanged. They are scoped to the test's engine instance, and the adapter seam is restored by the existing fixture cleanup.

## Independent checks

- **Focused tests.** Command: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Run 'Stale|Owned|Ownership|Attachment|Zalo|Upsert|Reserve|SyncChannel|SyncRun|SyncReserved|SyncAll|Scheduler|RecordSync|Sink|BoundedClasses|RunManual|HandleManual|AgentSync|SyncTerminal|KhoChinh' -VerboseTests`. Result: exit 0, 107 PASS and 0 SKIP, including both repair tests.
- **Full backend suite.** Command: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1`. Result: exit 0. **14** packages are `ok`, including `db`, `api/handlers` (20.8s) and `engine` (14.4s). The repair evidence says "13 tested packages"; that count is off by one because the `db` package has had a test since R014. The underlying result is a pass.
- **Mutation.** Restoring the R1 abort branch fails the R1 test, as described above. The earlier `FOR UPDATE` mutation from the first review still applies, because the guard is unchanged.
- **Catalog and doctor.** Catalog `-Check` → PASS. Doctor → `RESULT: PASS (25/25)`.
- **Cleanup.** Every run removed its disposable MySQL container and network. The `ccma-test` container and network lists are empty, and the persistent Compose stack was not touched.
- **Boundaries.** No real channel, provider, OAuth or S3 endpoint was contacted, and no governance claim is made.

## Remaining observations (non-blocking)

- **O1–O3 from the first review remain open.**
  - O1: a same-millisecond, unchanged message count can update zero rows.
  - O2: `JSON_SEARCH` treats `%` and `_` as wildcards, so the reference check can over-match. The only consequence is an orphaned object, which is the safe direction.
  - O3: Zalo refresh tokens are single-use, so a failed persist after rotation leaves an unusable token.
- **O4.** A failed `Exists` probe is now discarded without a log line. The later transfer logs its own outcome, but the probe failure itself is not visible. Consider a bounded log class.
- **O5.** The repair evidence gives 13 packages instead of 14. This is a documentation accuracy note only.

## Disposition

`REVIEW_PASS` for CCMAI-RUNTIME-015 (BUILD `de0616c` plus repair `904454d`).

- The contract is met: local persisted side effects are fenced by the locked ownership guard.
- FREEZE remains open and is not claimed.
- This is not crash recovery and not live-provider proof.
- A provider call that is already in flight can still complete after ownership is lost, and a rotated Zalo token cannot be undone.
- Reclaim remains unauthorized and is the subject of R016.
- S1 remains IN_PROGRESS.
- No push, source edit by the reviewer, or FREEZE.
