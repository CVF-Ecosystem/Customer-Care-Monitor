# CCMAI-UX-016 — consume reviewed QC count in Dashboard

**Date:** 2026-09-29 · **State:** BUILD complete / REVIEW_PENDING for independent Claude review · **Risk ceiling:** R2 · **Authority:** [SPEC](../specs/DASHBOARD_QC_VIOLATION_CARD_UX016_2026-09-29.md), [R017 review](../reviews/CCMAI_RUNTIME_017_INDEPENDENT_REVIEW_2026-09-29.md), [UI roadmap](../roadmaps/UI_UX_REDESIGN_ROADMAP_2026-09-27.md), [BUILD evidence](../reviews/DASHBOARD_QC_VIOLATION_CARD_UX016_BUILD_2026-09-29.md).

## Route and scope

Codex: `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD`. Claude: independent `REVIEWER` after one local BUILD/evidence commit. Record each role transition in the active handoff before acting. This resolves the parked UX-02 UI/API intersection after R017's backend API contract passed review.

Implementation paths: `frontend/src/views/Dashboard.vue`, `frontend/src/i18n/vi.ts`, `frontend/src/i18n/en.ts` and a focused test under `frontend/src/__tests__/`. Records: this SPEC/work order, BUILD evidence under `docs/reviews/`, active state/handoff, memory, `IMPLEMENTATION_STATUS.json`, UI and S1 roadmaps. No backend, API client/store, router, database, provider/channel call, persistent stack, deployment, push or FREEZE.

## Acceptance

1. Second card displays `qc_violation_count` only when it is a nonnegative safe integer; valid zero remains zero. Missing/invalid/error/loading shows `—`, never `issues` or a stale count. Update vi/en label and hint to describe counted QC violation rows.
2. Focused mounted-component test uses synthetic API responses where `issues` and `qc_violation_count` differ. Prove valid value, zero, missing/invalid value, request failure and refresh behavior. Mock data here proves UI rendering only, not CVF governance.
3. Run focused frontend test, frontend build/typecheck, catalog `-Check`, workspace doctor and `git diff --check`. Record exact results and no provider/network effect. Check the exact changed set, synchronize continuity, and make one local BUILD/evidence commit. Return `REVIEW_PENDING` to Claude; Codex must not self-PASS or FREEZE.
