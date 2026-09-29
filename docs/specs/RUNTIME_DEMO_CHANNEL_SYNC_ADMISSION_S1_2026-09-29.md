# CCMAI-RUNTIME-018 — demo channel sync admission

**Date:** 2026-09-29 · **Phase:** SPEC · **Risk:** R2 · **Authority:** [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [UX-06 investigation](../reviews/UX_001C_SYNC_STATUS_INVESTIGATION_2026-09-28.md), [UI roadmap boundary](../roadmaps/UI_UX_REDESIGN_ROADMAP_2026-09-27.md), [R013 single-flight review](../reviews/CCMAI_RUNTIME_013_INDEPENDENT_REVIEW_2026-09-29.md).

## Problem and decision

`ImportDemoData` creates active Zalo/Facebook channels with plaintext `{"demo":true}` in the credential column. The scheduler sees them as due, reserves them and records a decrypt failure as their sync status. These fixture channels cannot contact an upstream channel. A tenant-level `is_demo_data` flag is insufficient because a real channel may coexist with demo data.

Add an **internal, server-owned channel marker** (`is_demo_fixture`, default false, never writable through channel metadata or serialized in channel responses). New demo imports set it true. On migration, mark only legacy rows matching the exact historical fixture identity: known demo external ID (`demo-zalo-oa` or `demo-fb-page` with matching type), exact plaintext demo credential bytes, and a tenant whose settings contain `is_demo_data=true`. Migration is idempotent and leaves every other row untouched, including a real channel in a demo tenant and a same-named channel with encrypted credentials.

For those exact legacy fixture rows, a previous fixture-only decrypt failure may be cleared to the never-synced state **only** when `last_sync_at` is NULL, status is `error`, error text begins with `decrypt failed:`, and no run ID/lease is active. Preserve all other status, checkpoint and audit/activity records. A channel with an active run is marked but its run state is not changed by migration.

## Runtime contract

1. Scheduler excludes marked demo fixtures before due-check or `SyncChannel`; a marked fixture produces no reservation, credential decryption, adapter call, status mutation or analysis trigger. Unmarked active channels still follow the existing interval and single-flight logic.
2. `ReserveChannelSync` is the shared atomic admission boundary for manual, scheduler and agent sync. It refuses a marked fixture without writing status or run ID, with a bounded `ErrDemoFixture` class. The conditional UPDATE must carry the marker predicate; the zero-row classifier distinguishes a marked fixture from missing/busy channels. No outbound adapter work follows denial.
3. Manual `POST /channels/:channelId/sync` rejects a marked fixture with HTTP 409 `demo_channel_not_syncable` **before** configuration loading and before a worker dispatch. It still rejects a marker change between the initial read and reservation. Existing 202/409/error behavior for real channels remains unchanged. Agent sync paths may retain their existing generic action error shape, but must not dispatch or mutate a marked fixture.
4. Expired-lease recovery does not act on marked fixtures. Historical run rows, especially active `syncing`, are not silently reclaimed by this tranche.
5. A newly created real channel remains unmarked even in a tenant with demo data. Channel API input or metadata cannot forge or clear the marker. Conversion of a fixture channel to a real channel is by delete/create under the existing API, not by mutating the marker.

## Evidence and limits

Disposable MySQL tests must cover fresh import, old-schema migration twice, exact/non-exact legacy rows, guarded status cleanup, scheduler, shared admission, manual route, tenant coexistence, DB error fail-closed behavior and no outbound on denial. A positive detector must prove an unmarked due channel still syncs. Mutations removing the scheduler and admission predicates must fail focused tests.

No real channel/provider API call, persistent Compose DB access, frontend change, credential decoding shortcut, live CVF governance claim, deployment or push. Older binaries ignore the marker; a mixed-version rollout is unsupported. This tranche does not recover Zalo/legacy crashed runs or alter real-channel error reporting. REVIEW and FREEZE are separate.
