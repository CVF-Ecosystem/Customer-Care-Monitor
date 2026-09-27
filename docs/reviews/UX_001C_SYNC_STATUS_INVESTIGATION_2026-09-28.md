# Investigation — why demo channels show "Lỗi" (`CCMAI-UX-001c`, finding UX-06)

**Date:** 2026-09-28 · **Investigator:** Claude (read-only investigation under the owner's 2026-09-28 roadmap instruction) · **Risk of the investigation:** R1 (read-only; disposable environment) · **Outcome:** root cause confirmed; **fix not implemented**. It belongs to the runtime sync/scheduler scope under Codex orchestration (R001/R007/R008 surfaces), so it is routed there as a proposal and recorded as `BLOCKED_OUT_OF_UX_SCOPE`.

## Finding (baseline)

UX-06: both demo channels show "Trạng thái đồng bộ: Lỗi" while "Đồng bộ lần cuối" is "—".

## Evidence

1. **Fresh import is clean.** In a disposable environment (`scripts/ui-screenshots.ps1 -Mode app`, isolated DNS), channels right after `POST /demo/import` have an empty `last_sync_status` (UI shows "—").
2. **The scheduler produces the error.** `engine/scheduler.go` `Start()` registers `sync-all-channels` every 5 minutes. `syncAllChannelsTask` selects every `is_active = true` channel, including demo channels. `channelSyncDue` returns true when `last_sync_at` is nil.
3. **The sync fails before any network call.** Demo channels are written with `CredentialsEncrypted: []byte(`{"demo":true}`)` (`api/handlers/demo.go`), which is plaintext, not ciphertext. Disposable-run log at +5 min:
   ```
   [sync] starting sync for channel Coffee Zalo OA (zalo_oa)
   [scheduler] sync channel Coffee Zalo OA failed: sync error: decrypt failed: decrypt: cipher: message authentication failed
   ```
   After this, the API returned `last_sync_status = "error"`, `last_sync_at = null` for both channels. That is exactly the baseline screen.
4. **The persistent development stack shows the same pattern** (read-only `docker logs ccma-app-1`): 33 such failures so far, one per channel every 15 minutes (the default interval), for the older-named demo channels. Nothing was changed there.
5. **No outbound request.** Because decryption fails first, no adapter reaches Zalo/Facebook. The screenshot environment now also blocks external DNS (UX-000 addendum A2).

## Why the UI is misleading, not wrong

The stored status is true: a sync was attempted and failed. The screen is misleading because (a) it never says *when* the failed attempt happened ("Đồng bộ lần cuối" shows only the last **success**), (b) the error reason is not shown, and (c) demo channels can never sync, so the error repeats forever.

## Proposed fix (for a runtime tranche; not done here)

| Option | Scope | Note |
|---|---|---|
| A. Scheduler skips demo channels (for example, tenant `settings.is_demo_data` or a demo marker in channel metadata) | `engine/scheduler.go`, tests | Smallest behavior change; demo channels stay "Chưa đồng bộ". Needs a reliable demo marker. |
| B. Demo import creates channels with `is_active = false` | `demo.go` | Changes what the Channels page shows ("Tạm dừng"); weaker demo. |
| C. Record the attempt time on failure and show "Lần thử cuối: … (lỗi: …)" | backend field + UI | An API change (new field). Improves truth for real channels too. |

Recommendation: A for demo data, and C for real channels as part of the Channels screen tranche once an API decision is made. The UX Channels screen (roadmap screen 4) should also show "Chưa đồng bộ" for an empty status and the last *attempt* rather than only the last success. The UX-000 `SyncStatusChip` already maps empty to "Chưa đồng bộ".

## Boundaries

No source change, no persistent database change, no provider call, no real channel call, no deployment or push. The disposable environment and its image and volumes were removed.
