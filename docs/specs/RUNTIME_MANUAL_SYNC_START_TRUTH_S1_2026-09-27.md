# S1 manual sync start truth

**Tranche:** `CCMAI-RUNTIME-007` · **Phase:** SPEC · **Risk:** R2 · **Entry:** R001–R006 independent REVIEW PASS / FREEZE open; S1 IN_PROGRESS.

## Problem and decision

`SyncChannelNow` loads a tenant-scoped channel and writes `last_sync_status=syncing`, but ignores the write result and always starts a goroutine and returns HTTP 202 `sync_started`. A failed write therefore produces a false accepted-start signal and can leave the visible state stale. The panic recovery write also ignores its result and selects by channel ID alone.

The manual request must establish an observable, tenant-scoped start state before acknowledging dispatch. This is a dispatch acknowledgment, not evidence that the channel sync completed or that an upstream source is complete.

## Contract

1. After the existing tenant-scoped channel lookup, persist `syncing` with a tenant-and-channel predicate. Verify both the DB error and exactly one affected row. Only then launch the background worker and return HTTP 202 with the existing `sync_started` response. On write failure or zero affected rows, return an observable non-2xx response and start no worker. Do not expose DB internals or credentials to the client.
2. Panic recovery must attempt a tenant-scoped `error` status write and log a write failure. Its persisted error text must be bounded and must not include a raw panic value that may carry credentials. The log may identify the channel and the failure class without leaking secrets.
3. Preserve the existing successful sync path, checkpoint rules, `partial` semantics, response shape, tenant permission gate and frontend wording. Do not treat 202 as a completed sync. This tranche does not solve concurrent manual/scheduler/agent runs or stale in-flight workers; record those as S1 residuals.

## Acceptance

- A disposable-MySQL handler test forces the `syncing` update to fail and proves non-2xx response, unchanged visible state and no worker launch. A zero-row update also cannot yield 202. A normal synthetic-channel request yields 202 only after the start state is observable.
- Focused recovery test proves tenant-scoped, bounded failure status and verifies an injected status-write failure is logged/returned to the testable helper without a silent success claim. No real channel credential, adapter call, customer data or provider call is used.
- Focused/full backend tests on disposable MySQL, build, vet, catalog check, doctor and diff check pass. Record exact commands, fixture cleanup and limits in BUILD evidence.

## Deferred

Cross-path single-flight/lease coordination, crash recovery, real-channel sync, upstream history coverage, S1 closure and FREEZE need separate disposition. This local reliability change is not live CVF governance proof.
