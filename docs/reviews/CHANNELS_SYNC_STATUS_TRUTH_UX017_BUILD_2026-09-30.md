# CCMAI-UX-017 BUILD evidence — truthful sync status on Channels screens

**Date:** 2026-09-30 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 · **Role:** IMPLEMENTATION_WORKER (Claude); independent REVIEWER: Codex
**Authority:** [SPEC](../specs/CHANNELS_SYNC_STATUS_TRUTH_UX017_2026-09-30.md), [work order](../work_orders/CCMAI_UX_017.md).
Synthetic API responses prove UI rendering only. No provider/channel request, backend/API/store/router change, or CVF governance claim.

## Changes

| File | Change |
|---|---|
| `frontend/src/views/Channels.vue` | Status uses shared `SyncStatusChip` (six states; empty = "Chưa đồng bộ", never `—`). Timestamp labelled "Lần đồng bộ thành công" with null shown as "Chưa có lần đồng bộ thành công"; an older success time stays beside partial/error/syncing. Manual-sync toast uses i18n "started" wording. The reauth button is now a neutral text button labelled "Kết nối lại (tùy chọn)" with a hint title; its visibility condition (error and not pancake) and route call are unchanged. Local `syncColor`/`syncLabel` removed. |
| `frontend/src/views/Channels/ChannelDetail.vue` | Same chip and timestamp label/no-success text. `doSync` split: a rejected start request stays an error; a poll timeout or refresh failure is a **warning "unconfirmed"**; only an observed terminal status yields success/partial/error. A sync-history failure after an observed status is ignored (secondary). Local status mapping removed. |
| `frontend/src/i18n/vi.ts`, `en.ts` | New keys `ch_last_success_sync`, `ch_no_success_yet`, `ch_sync_started`, `ch_sync_unconfirmed`, `ch_reauth_optional`, `ch_reauth_hint`. The legacy `last_sync` key is left in place (now unused). |
| `frontend/src/__tests__/channels-sync-status.spec.ts` | 13 mounted tests (below). |

Read-only as required: `SyncStatusChip.vue`, `utils/review.ts`, channel store/API, router. No API data was missing, so no `BUILD_BLOCKED`.

## Tests and test data

Mounted real `Channels.vue` / `ChannelDetail.vue` with synthetic channels, a mocked `api`, real Pinia store, Vuetify, vue-i18n and router; fake timers for polling; no sleeps or network.

- List: six states in order (`''`, syncing, success, partial, error, unrecognised) with `data-sync` and text; null vs older success time for partial/error/syncing; inactive+error independence; English labels; accepted 202 shows "started" and not success; reauth only on facebook/zalo error rows (not pancake, not success), neutral and optional in vi and en.
- Detail: six-state chip and time matrix; English labels and inactive independence; poll timeout (60 × 3 s) is a warning "unconfirmed", not failure/success, chip still `syncing`; refresh failure is unconfirmed; observed success/partial/error each keep their own message and alert class; rejected start request is an error with no polling; no reauth control on detail.
- Negative detectors: empty status is not `—`; unknown is not "Đã đồng bộ"; the old "Đồng bộ lần cuối"/"Last Synced" labels are absent; a partial card must show its older success time labelled as success; timeout must not say "Đồng bộ thất bại".

## Non-vacuity

Stashing only the two view files (old presentation, new tests) → **10 failed / 3 passed**; stash popped, worktree restored, all 13 PASS.

## Gates

| Command | Result |
|---|---|
| `npx vitest run src/__tests__/channels-sync-status.spec.ts` | 13/13 PASS |
| `npx vitest run` (full frontend) | 18 files, 184 tests PASS |
| `npx vue-tsc -b --force` | exit 0 |
| `npm run build` | built successfully |
| `scripts/manage_cvf_downstream_catalog.ps1 -Check` | PASS |
| `check_cvf_workspace_agent_enforcement.ps1 -ProjectPath <root>` | PASS 25/25 |
| `git diff --check` | clean (CRLF warnings only) |

## Claim limits and residuals

- `last_sync_at` remains "last successful"; no attempt time, error reason or demo marker exists in the API and none is invented. Historical demo error rows still show error until a separately authorized reset.
- The list does not poll; it shows started wording and the status at the next fetch.
- The list reauth button is still not gated by `canEdit` (unchanged existing behavior; the backend enforces permissions). Detail error text may still show the backend `last_sync_error` as before.
- No browser/screenshot check of layout was performed; jsdom-class tests do not prove visual quality. Real-channel sync completeness, Zalo/legacy/mixed-version recovery and S1 closure remain open.

## Final changed set

Four source/test files above, this evidence file, `CVF_SESSION/handoffs/AGENT_HANDOFF_V1_2026-09-26.md`, `CVF_SESSION/ACTIVE_SESSION_STATE.json`, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`.

## Disposition

`REVIEW_PENDING` for Codex. Claude does not self-approve; no push or FREEZE.
