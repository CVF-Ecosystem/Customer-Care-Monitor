# CCMAI-UX-016-F1 — independent review

**Date:** 2026-09-30 · **Reviewer:** Codex (`REVIEWER`, independent of Claude's BUILD) · **Target:** `2dc34566f6077720e2c988bd50f444d41d95319d` against parent `7db7d07` · **Disposition:** `REVIEW_PASS / FREEZE_OPEN`.

## Authority and scope

Reviewed [F1 work order](../work_orders/CCMAI_UX_016_F1.md), [UX-016 SPEC](../specs/DASHBOARD_QC_VIOLATION_CARD_UX016_2026-09-29.md), [original independent finding F1](CCMAI_UX_016_INDEPENDENT_REVIEW_2026-09-29.md), [BUILD evidence](CCMAI_UX_016_F1_BUILD_2026-09-30.md) and the exact commit diff. The changed paths are the one allowed frontend test, the named BUILD evidence and four continuity/status files. `Dashboard.vue`, API/store/router/i18n and backend source are unchanged. The worktree was clean before the reviewer acknowledgment.

## Finding disposition

F1 is resolved. The committed mounted-component test starts from a settled QC count of 4, captures two deferred `/dashboard` calls from distinct `from` dates, checks the loading mark, resolves the newer response with QC 7 and `issues` 13, then resolves the older response with QC 99 and `issues` 55. It asserts that the card stays at 7 after both resolutions. This exercises response order and field selection without a sleep or network request.

Claude's BUILD evidence records a disposable mutation that bypassed `if (requestId !== dashboardRequestId) return`: the focused test failed with actual 99 versus expected 7; the source was restored and the focused suite passed. Codex inspected that evidence, verified the guard remains in unchanged `Dashboard.vue`, and independently reran `npx vitest run src/__tests__/dashboard-qc-card.spec.ts` from `frontend/`: **1 file, 4/4 tests PASS** (exit 0). Claude's full frontend suite (171/171), typecheck, build, catalog and doctor results are recorded in BUILD evidence; this review does not represent an independent rerun of those broader gates.

## Boundary

This is synthetic UI behavior evidence only. It does not prove CVF runtime governance or real provider/channel behavior. The earlier UX-016 `REVIEW_PASS` remains valid, F1 test coverage now passes review, and FREEZE, deployment and push remain open. Zalo/legacy/mixed-version crash recovery and S1 completion are separate.
