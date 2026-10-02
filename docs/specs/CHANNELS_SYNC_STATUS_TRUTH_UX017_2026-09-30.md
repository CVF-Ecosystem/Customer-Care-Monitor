# CCMAI-UX-017 — truthful sync status on Channels screens

**Date:** 2026-09-30 · **Phase:** SPEC · **Risk:** R2 · **Entry decision:** R018 and UX-016-F1 passed independent REVIEW; S1 remains IN_PROGRESS. UX-06 presentation can enter at SPEC using the reviewed R001/R007/R013/R018 status behavior and the current Channels API. [UI roadmap](../roadmaps/UI_UX_REDESIGN_ROADMAP_2026-09-27.md), [UX-06 investigation](../reviews/UX_001C_SYNC_STATUS_INVESTIGATION_2026-09-28.md), [R018 review](../reviews/CCMAI_RUNTIME_018_R1_INDEPENDENT_REREVIEW_2026-09-29.md).

## Source-grounded problem

`GET /channels` and `GET /channels/:id` expose `last_sync_status` and `last_sync_at`; `last_sync_at` advances only on a successful terminal sync. They do not expose the last attempt time, a safe error reason or the server-owned demo fixture marker. `Channels.vue` currently shows an empty status as `—` and labels `last_sync_at` “Đồng bộ lần cuối”. `ChannelDetail.vue` has its own status mapping and labels the same timestamp “Đồng bộ lần cuối”. The shared `SyncStatusChip` already handles never/syncing/success/partial/error/unknown and says syncing has started, not finished. The list also highlights “Kết nối lại” on any non-Pancake error although error alone does not prove credential expiry. Detail polling calls a three-minute timeout an error even though the channel may still be syncing.

## Decision and required behavior

1. On both Channels list and detail, display `last_sync_status` using the shared `SyncStatusChip`/`syncKind` semantics. Empty/null means “Chưa đồng bộ”; unknown means unknown, never success. Preserve distinct syncing, success, partial and error states. An inactive channel's activity chip stays a separate fact and must not replace its sync status.
2. Label `last_sync_at` as **“Lần đồng bộ thành công” / “Last successful sync”**. A null timestamp displays **“Chưa có lần đồng bộ thành công” / “No successful sync yet”**. A partial/error/syncing status may coexist with an older successful timestamp; show both independently, without presenting the timestamp as the latest attempt or erasing it. Use the existing timestamp formatting on each screen. Do not derive an attempt time from `updated_at`, `created_at`, a client clock or sync history.
3. The list's manual-sync acceptance says only that sync started. Detail polling may report success or partial only after an observed terminal status; if polling times out or a refresh fails, show an **unconfirmed** warning, not a confirmed sync failure. Do not clear, fabricate or advance a checkpoint in the UI.
4. An error status alone must not visually assert credential expiry. Keep the existing optional reauth action for applicable channel types but make it secondary/neutral and explain that it is for credential problems when confirmed by the operator. Do not infer auth failure from `last_sync_status`, expose raw backend errors, add an automatic reauth call or change route behavior.
5. Use vi/en translation keys for new shared status/time/uncertainty text. Existing channel creation/edit, active/inactive state, test connection, sync request contract, history, permissions and navigation must keep their behavior. The demo fixture marker stays server-owned and absent from the UI/API; historical demo error rows may still show error until a separately authorized reset.

## Acceptance evidence

- Mounted component tests with synthetic channel responses cover both screens: empty, syncing, success, partial, error and unknown statuses; null and previous-success timestamps; a partial/error status with an older success time; active/inactive independence. Assert displayed text, not only helper output.
- Verify accepted manual sync still says started. For detail, fake polling/time or a controlled store seam proves timeout and refresh failure are labeled unconfirmed, and an observed terminal partial/error remains distinct. Keep tests deterministic without real network, sleeps or provider calls.
- Verify a generic error does not render reauth as a warning/required action and that the optional action still follows its existing channel-type and permission boundary. Test both Vietnamese and English wording for the new labels.
- Run focused/full frontend tests, forced typecheck, frontend build, catalog `-Check`, workspace doctor and diff check. Capture any material inability to mount a route or distinguish API states as `BUILD_BLOCKED` rather than inventing data.

## Boundaries

Frontend presentation only. No backend/schema/API/store change, no sync/recovery/adaptor behavior change, no real provider/channel request, no persistent Compose data mutation, no push, deployment or FREEZE. Synthetic tests prove UI rendering, not CVF governance or live sync correctness. Zalo/legacy/mixed-version crash recovery remains a separate S1 tranche.
