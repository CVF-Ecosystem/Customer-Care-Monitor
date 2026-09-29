# CCMAI-RUNTIME-016 independent review

**Date:** 2026-09-29 · **Reviewer:** Codex, independent of Claude BUILD · **BUILD commit:** `a7c0a78` (parent `6b946ed`) · **Disposition:** `CHANGES_REQUIRED` · **Risk:** R2. FREEZE remains open.

## Scope and checks

At INTAKE, the active handoff header still said `WORK_ORDER` although the state, memory, status and BUILD evidence said `REVIEW_PENDING`. I aligned only that stale header, rehydrated and took `COMMIT_STEWARD (Claude) → REVIEWER (Codex)` before source review. The worktree was clean at `a7c0a78`; core `26c686c` matches the manifest and doctor passed 25/25.

I inspected the exact `6b946ed..a7c0a78` source diff against the R016 SPEC/work order and the BUILD evidence. The changed production paths are within scope. The `facebook` and `pancake` adapters' outbound sync helpers use GET (`backend/channels/facebook.go:33`, `backend/channels/pancake.go:106`); Zalo's refresh POST remains outside the recovery allowlist. Admission writes the lease in the same conditional update as run ID; recovery repeats tenant, type, status, observed run ID and DB-time expiry predicates. Terminal and manual panic writes clear the lease. Existing tests cover supported/denied types, migration, heartbeat/release races, scheduler readmission and a paused A→B takeover.

Independent focused command: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Run 'Lease|Heartbeat|Recovery|ExpiredGeneration|TerminalWritesClear|ManualSyncCarries'` → exit 0 (`api/handlers`, `db`, `engine` passed). Disposable MySQL and network were removed; the persistent Compose stack was untouched.

## Blocking finding R016-R1 — a failed heartbeat can still publish an attachment message

In `backend/engine/sync.go:454`, the post-download check returns the heartbeat error only when **both** `attachmentErr != nil` and `hbErr != nil`. If an attachment store completes successfully just as the heartbeat fails, `attachmentErr` is nil, so the run proceeds to `s.upsertMessage` at line 464. R015's ownership guard can still succeed when the heartbeat failed due to a DB write error but the row remains owned. This violates SPEC clause 2: a failed heartbeat cancels the run and stops further work.

I added a temporary reviewer-only test and removed it after running. The synthetic store's `Put` waits for the run context to be canceled, then returns success, modeling completion at the cancellation boundary. A MySQL trigger forces the heartbeat update to fail. The full `SyncReservedChannel` returns bounded `ErrSyncWriteFailed`, but the test fails because **one message was persisted after heartbeat failure**. The probe output included `lease heartbeat ... failed: sync write failed`, `downloaded attachment`, `synced 1/1 conversations, 1 messages`, then `message persisted after heartbeat failure: 1`; test runner exit 1. Its disposable MySQL and network were removed. The reviewer probe is not part of this review commit.

**Required repair:** Check `stopErr()` after `downloadAttachments` regardless of the attachment transfer result, before `upsertMessage`; clean only new attempt keys when aborting. Add a permanent deterministic regression with a storage operation that completes successfully when cancellation arrives, forced heartbeat failure, and assertions for no new message/count/checkpoint or analysis trigger, bounded error and owned terminal status. Preserve the existing per-attachment partial behavior when no heartbeat failed. Recheck the neighboring gates so a successful attachment transfer cannot hide cancellation.

## Boundaries and next move

The main lease/release predicates and GET-only allowlist are supported by source and the existing focused checks. The blocking finding prevents `REVIEW_PASS`; the green BUILD suite lacks this cancellation boundary. Lease expiry remains a release condition, not proof of worker death. Zalo and legacy rows remain excluded. This is synthetic runtime evidence, not live-provider or CVF governance proof.

Claude may repair R016-R1 within `backend/engine/sync.go`, focused engine tests and the existing BUILD evidence/continuity paths under the same R2 work order. Return one local `REVIEW_PENDING` repair commit for independent Codex re-review. No push, real provider/channel call, rollout, Zalo/legacy recovery or FREEZE.
