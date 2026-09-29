# CCMAI-RUNTIME-015 BUILD evidence — persisted sync side-effect ownership

**Date:** 2026-09-29 · **Builder:** Codex · **Base:** `60ed988` · **Status:** `REVIEW_PENDING` for independent Claude review · **Risk:** R2. No push, deployment, recovery/reclaim or FREEZE.

## Route and changed scope

Claude reported that its command/write tools repeatedly returned no verdict before any BUILD, handoff acknowledgment or commit. Codex independently rehydrated manifest, policy, active state/handoff, implementation status and docs index; verified clean `60ed988`, pinned core `26c686c` matching `origin/main`, and doctor 25/25. The active handoff records the route change `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Codex)` before source edit. Claude is the independent R2 reviewer when its tools recover. The SPEC/work order were amended to reuse same-identity attachment objects when present, preventing an orphan on every unchanged replay.

Production source changed only in `backend/engine/sync.go` and `backend/channels/zalo_oa.go`. Focused tests were added/updated under `backend/engine/` and `backend/channels/`. No model, migration, router, frontend, scheduler, agent handler, OAuth handler, storage implementation or CVF core change.

## Implemented contract

- `withOwnedSyncWrite` starts a silent GORM transaction, locks the tenant-owned channel row with `SELECT ... FOR UPDATE`, compares `syncing` and the reservation's run ID, then performs the local write before releasing the lock. Missing/changed ownership returns `sync_ownership_lost`; DB failures return `sync write failed`. The run ID and SQL values are not printed by GORM for the guarded transaction. The sync loop stops on lost ownership rather than reporting a partial conversation and continuing.
- Conversation create/update, message create/replay merge, message-count update and Zalo refreshed-credential persistence run inside that ownership transaction. Existing replay merge rules remain. The credential write checks DB error and exactly one row; the Zalo callback now returns an error, so a failed persistence stops refresh retry instead of silently continuing.
- The attachment path remains tenant-first for the existing file route. A stored object with the same `(type, URL, name)` is reused when it exists, including legacy `<tenant>/<conversation>/<filename>` paths. A genuinely new download uses `<tenant>/<conversation>/<random attempt>/<filename>`, so an old in-flight download cannot overwrite a newer run's bytes. Only an owned message write publishes the new key. On failed publication, cleanup checks the DB for a reference before deleting only a new attempt key. If the reference check fails, it keeps the object: an ambiguous DB commit must not lead to deleting a referenced file.
- Attachment success/failure logs use bounded operation classes and do not print the external URL or store error text; the focused synthetic signed-URL test checks the application log sink.

## Tests and gates

- Focused legacy R003/R013/R014 tests passed on disposable MySQL before the new tests. Focused R015 ownership, lock, attachment, bounded-error and synthetic Zalo callback tests passed. The first full backend run found one stale R013 assertion: tenant reassignment now returns `sync_ownership_lost` before the terminal zero-row write. The assertion was corrected to the new fail-fast contract; subsequent focused and full runs passed.
- Two direct detector mutations were run together and restored: removing `FOR UPDATE` failed `TestOwnedWriteSerializesWithOwnershipChange` because takeover completed before the write committed; replacing the attempt-unique key with the old shared filename failed `TestInFlightOldAttachmentCannotReplaceNewRunObject` because keys collided. Restored source is the tested source.
- Full `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1` passed all 14 tested backend packages on disposable MySQL (`api/handlers` 21.571s, `engine` 14.482s in the final run after the signed-URL log repair); the script removed its MySQL container and network. A focused `TestLegacyAttachmentPathIsReusedWithoutDownload|TestInFlightOldAttachmentCannotReplaceNewRunObject` run also passed and removed its disposable resources. `go build ./...`, `go vet ./...`, catalog `-Check`, JSON parse, and `git diff --check` passed; project doctor passed 25/25 after continuity synchronization.

## Claim boundary and residual risk

These are **local persisted-effect fences**, not crash recovery or live provider proof. A provider request already in flight may finish after the run loses ownership; a Zalo refresh can rotate a single-use token before the guarded local persistence reports failure. Reclaim is still unauthorized. Mixed old/new binaries are unsafe for takeover because old workers do not carry these guards. New attempt objects can remain orphaned after a crash, a failed cleanup, or replacement by a changed attachment identity; unchanged identities reuse existing objects. Cleanup deliberately prefers an orphan over deleting a possibly referenced object. No real channel/provider call, CVF governance claim, persistent Compose DB mutation or automatic recovery was made.
