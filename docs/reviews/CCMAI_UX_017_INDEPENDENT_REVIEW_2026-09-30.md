# CCMAI-UX-017 — independent R2 review

**Date:** 2026-09-30 · **Reviewer:** Codex (`REVIEWER`, independent of Claude's BUILD) · **Target:** `0e371eb69bb90b0e3e5b59ab1b92b6cc8b2ba9fa` against parent `5032f4a` · **Disposition:** `CHANGES_REQUIRED / REVIEW_OPEN`.

## Scope and checks

Rehydrated canonical CVF continuity, verified core and doctor 25/25, and acknowledged the reviewer role before inspecting the exact commit. Source changes are confined to both allowed Channels views, vi/en strings and one focused mounted UI test file; evidence and continuity are in the permitted record class. Shared `SyncStatusChip`, store/API/router and backend source are unchanged. The BUILD evidence's final-changed-set paragraph lists `IMPLEMENTATION_STATUS.json`, but that file is not in the commit; this is a record correction, not a source defect.

Codex independently reran `npx vitest run src/__tests__/channels-sync-status.spec.ts` from `frontend/`: **1 file, 13/13 PASS** (exit 0). Claude's evidence records full frontend 184/184, forced typecheck, build, catalog and doctor PASS; those wider gates were inspected, not independently rerun here. UI mocks establish presentation behavior only, with no real provider/channel or CVF governance claim.

## Blocking finding UX017-R1 — unknown poll result reported as confirmed failure

In `ChannelDetail.vue` `doSync`, a completed `fetchChannel` sets `observed = true` for **every value other than `syncing`**. The code then treats `success` and `partial` specially and reports all remaining values as an error. Thus an empty/null status, `never`, or an unexpected string during polling after an accepted sync produces the confirmed “Đồng bộ thất bại” alert. None of these values is an observed terminal `error`. This contradicts UX-017 SPEC §3 and the work order's “observed success/partial/error” contract. The current 13 tests cover those values as static chips but omit them as responses **during polling**, so they do not detect this branch.

Required repair: only `success`, `partial` and `error` may close polling as observed terminal outcomes. If a fetch returns any other non-`syncing` value, show the existing unconfirmed warning and stop that polling attempt; do not report a failure, clear the checkpoint, or infer a retry outcome. Add a deterministic mounted test for at least empty and unknown poll responses, while retaining the existing terminal success/partial/error, timeout, refresh-failure and rejected-start checks. A negative detector must fail if the non-`syncing` shortcut is restored.

## Disposition and boundaries

The status chip and success-only timestamp semantics otherwise match the SPEC, and the optional reauth action is visually neutral. No production source repair is made by this reviewer. Same-scope R2 repair authority is appended to the work order; Claude returns one local `REVIEW_PENDING` repair/evidence commit for Codex re-review. FREEZE, push, deployment, real channel/provider calls, Zalo/legacy crash recovery and S1 closure remain open.
