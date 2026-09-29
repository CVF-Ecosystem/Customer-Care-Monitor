# CCMAI-RUNTIME-016 R016-R1 independent re-review

**Date:** 2026-09-29 · **Reviewer:** Codex, independent of Claude repair · **Repair commit:** `7edd8c4` (parent `c5ea48a`) · **Disposition:** `REVIEW_PASS / FREEZE_OPEN` · **Risk:** R2.

## Review and independent check

At INTAKE the handoff header still said `CHANGES_REQUIRED`, while active state, memory, status and the repair result said `REVIEW_PENDING`. I aligned that header, re-read continuity and recorded `COMMIT_STEWARD (Claude) → REVIEWER (Codex)` in the active handoff before review. Core `26c686c` matches the manifest; the workspace doctor passed 25/25.

I inspected the exact `c5ea48a..7edd8c4` diff against R016-R1 and the original [finding](CCMAI_RUNTIME_016_INDEPENDENT_REVIEW_2026-09-29.md). The only production edit is in `backend/engine/sync.go`: after each attachment download, `stopErr()` is checked even when the transfer returns success. On heartbeat failure it cleans the new attempt keys and returns the bounded failure before `upsertMessage`. A second `gate()` precedes the conversation count write. The lease/recovery allowlist, predicates and outbound adapter behavior were not changed. The new full-run test forces a heartbeat DB error while a synthetic store completes its `Put` at cancellation; it asserts no message, count, checkpoint or analysis trigger, a bounded failure, owned terminal error and cleanup. A healthy successful transfer is a positive detector; a healthy failed transfer retains partial behavior.

Independent command: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'Test(HeartbeatFailureDuringSuccessfulAttachmentTransferPublishesNothing|SuccessfulAttachmentTransferWithHealthyHeartbeatIsPublished|FailedAttachmentTransferWithHealthyHeartbeatStaysPartial|HeartbeatLossCancelsTheRun|ExpiredGenerationCannotTouchTheNewerRun)'` → exit 0 (`engine` 1.635s). The test script removed its disposable MySQL container and network. `git diff --check`, catalog `-Check` and the workspace doctor passed after review records were synchronized.

## Disposition and limits

R016-R1 is resolved: the previously reproduced successful-transfer cancellation boundary now has a permanent passing regression. Claude's mutation restoring the old condition failed that regression and was restored, as recorded in [BUILD evidence](RUNTIME_SYNC_LEASE_RECOVERY_READ_ONLY_S1_BUILD_2026-09-29.md). The added pre-count `gate()` is sensible defense in depth, but its removal did not fail a deterministic test; it is not separately proven by mutation. This observation does not reopen the reproduced R016-R1 defect.

R016 remains `REVIEW_PASS / FREEZE_OPEN`; S1 remains `IN_PROGRESS`. This synthetic MySQL review does not prove real channel behavior or CVF governance. Lease expiry revokes ownership but does not establish worker death. Zalo, legacy and mixed-version recovery remain separate decisions. No live provider/channel call, rollout, push or FREEZE occurred. Next governed move: Codex as ORCHESTRATOR may scope the next S1 tranche after the owner requests it.
