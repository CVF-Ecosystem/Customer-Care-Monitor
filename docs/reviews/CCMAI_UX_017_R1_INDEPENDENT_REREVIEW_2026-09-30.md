# CCMAI-UX-017 UX017-R1 — independent re-review

**Date:** 2026-09-30 · **Reviewer:** Codex (`REVIEWER`, independent of Claude's repair) · **Target:** `848bb73679e5a6d502f4d5eb29a2cd5b98c997bd` against parent `07a60b1` · **Disposition:** `REVIEW_PASS / FREEZE_OPEN`.

## Scope and finding

Rehydrated CVF continuity, verified core/doctor 25/25, and acknowledged the reviewer role before checking the exact repair diff against [UX017-R1](CCMAI_UX_017_INDEPENDENT_REVIEW_2026-09-30.md) and the [work-order addendum](../work_orders/CCMAI_UX_017.md). The only product change is the terminal-status predicate in `ChannelDetail.vue`; the only test change is two mounted cases in `channels-sync-status.spec.ts`. BUILD evidence and continuity changes are within the permitted record class. `Channels.vue`, i18n, shared chip, store/API/router and backend are unchanged.

UX017-R1 is resolved. Polling now closes with an observed terminal result only for `success`, `partial` or `error`. `syncing` continues. Empty, null, `never` and unknown statuses stop with the existing unconfirmed warning, without fetching history or reporting a confirmed failure. The new tests directly exercise those four answers, verify polling stops, and verify a later observed `error` remains a confirmed failure. Claude's BUILD evidence records a disposable restoration of the broad non-`syncing` shortcut: the new test failed with a false “Đồng bộ thất bại”, then passed after restoration. The initial BUILD evidence's changed-set list was also corrected to omit `IMPLEMENTATION_STATUS.json`, which was not in that commit.

## Independent validation and limits

Codex reran `npx vitest run src/__tests__/channels-sync-status.spec.ts` from `frontend/`: **1 file, 15/15 PASS** (exit 0). Claude recorded full frontend 186/186, forced typecheck, build, catalog and doctor PASS in the repair evidence; this review inspected those results without rerunning the wider gates. Synthetic component tests prove UI behavior only. There was no real provider/channel request, live CVF governance proof, deployment, push or FREEZE.

UX-017 is `REVIEW_PASS / FREEZE_OPEN`; S1 and Zalo/legacy/mixed-version crash recovery remain open. Existing demo error rows and the absence of a last-attempt timestamp/safe error reason in the API are unchanged limitations.
