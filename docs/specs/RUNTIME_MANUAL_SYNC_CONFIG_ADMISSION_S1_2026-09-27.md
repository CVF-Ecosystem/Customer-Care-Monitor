# S1 manual sync configuration admission

**Tranche:** `CCMAI-RUNTIME-008` · **Phase:** SPEC · **Risk:** R2 · **Entry:** R001–R007 independent REVIEW PASS / FREEZE open; S1 IN_PROGRESS.

## Problem and decision

After R007 records `syncing`, `runManualSync` still calls `cfg, _ := config.Load()` inside the worker. If validation fails, `cfg` is nil and the worker may panic or leave an ambiguous status after HTTP 202. The selected channel and start state are already known at that point. Treat validated configuration as a prerequisite for acknowledging manual dispatch.

## Contract

1. Keep the existing tenant-scoped channel lookup and 404 behavior. After it succeeds but before writing `syncing`, load and validate configuration. If validation fails, return a generic non-2xx response, leave the channel's prior status/checkpoint/error unchanged and launch no worker. Do not expose configuration values or validation detail to the client or log secrets.
2. Pass the validated configuration to the worker launched for that request. The worker must not load it again or receive nil after a successful admission. Preserve R007's tenant-scoped checked start write, 202 body, bounded panic handling and normal `SyncEngine.SyncChannel` behavior.
3. This is manual-dispatch admission only. Scheduler and agent paths, cross-path single-flight, stale status after process crash, real channel sync and provider/AI behavior are outside this tranche.

## Acceptance

- Focused handler tests prove: a valid tenant/channel with a forced configuration-load failure gets non-2xx with no start-state write or worker launch; a valid configuration is the same configuration passed to the launched worker and `syncing` is persisted before the unchanged 202; wrong-tenant lookup still gets 404 without attempting configuration load. Existing R007 write-error/zero-row and panic tests pass after the seam signature changes.
- A test exercises the actual `config.Load` validation behavior using synthetic temporary environment values, without a real channel credential or adapter call. Test seams must be restored after each test; no parallel use of package-global seams.
- Focused/full backend tests with disposable MySQL, build, vet, catalog check, workspace doctor and diff check pass. BUILD evidence records exact commands, cleanup and remaining limits. No live governance claim follows from mocked or synthetic checks.
