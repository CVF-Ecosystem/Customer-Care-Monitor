# CCMAI-UX-016-F1 — commit the Dashboard stale-response regression

**Date:** 2026-09-30 · **Phase:** WORK_ORDER · **Risk ceiling:** R2 (inherited UX-016 contract) · **Entry decision:** UX-016 has independent `REVIEW_PASS / FREEZE_OPEN`; reviewer finding F1 is non-blocking and supplies the test contract. This tranche inherits the accepted UX-016 SPEC and enters at WORK_ORDER. It does not reopen the accepted product behavior.

**Authority:** [UX-016 SPEC](../specs/DASHBOARD_QC_VIOLATION_CARD_UX016_2026-09-29.md), [UX-016 independent review, F1](../reviews/CCMAI_UX_016_INDEPENDENT_REVIEW_2026-09-29.md), [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md).

## Objective and roles

Commit a deterministic mounted-component test for the existing `Dashboard.vue` request-sequence guard. The test must prove that a response from an older date selection cannot replace the newer selection's QC count after the newer response has rendered. Claude is `IMPLEMENTATION_WORKER`, then `SESSION_SYNC_STEWARD` and `COMMIT_STEWARD`. Codex remains the independent `REVIEWER` for this tranche. Claude must rehydrate the canonical CVF state and acknowledge `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff before BUILD.

## Allowed changes

- Implementation: `frontend/src/__tests__/dashboard-qc-card.spec.ts` only. Small local helper changes in that test file are allowed.
- Evidence: one new `docs/reviews/CCMAI_UX_016_F1_BUILD_2026-09-30.md` file.
- Continuity: active handoff, `CVF_SESSION/ACTIVE_SESSION_STATE.json`, `CVF_SESSION_MEMORY.md`, and `IMPLEMENTATION_STATUS.json` only as needed to record BUILD truth and route review.
- `Dashboard.vue` is read-only for this work order. If the committed test reveals a real source defect, stop at `BUILD_BLOCKED` and report the minimal source repair needed; do not widen the path silently.

No backend, API/store/router/i18n change, provider/channel call, persistent Compose database, external deployment, push or FREEZE. Do not include unrelated worktree files in the commit. Synthetic API responses prove only UI behavior, not CVF governance.

## Acceptance contract

1. Mount the real Dashboard component with the existing synthetic API seam. Start from a settled baseline, then issue two distinct date selections whose `/dashboard` promises remain unresolved. Verify the request parameters distinguish the selections.
2. Resolve the newer request first with `qc_violation_count = 7` and a deliberately different `issues` value. Assert the visible QC card shows `7`. Resolve the older request afterward with `qc_violation_count = 99`; after Vue/promise flushing, assert the visible QC card still shows `7`. The test must be deterministic and must not use sleeps or network calls.
3. Demonstrate non-vacuity: temporarily remove or bypass the `requestId !== dashboardRequestId` stale-response guard in a disposable edit, run the focused test and record that it fails by showing `99`; restore the source exactly and rerun to PASS. Do not commit the mutation. A same-value or unobserved second request does not satisfy F1.
4. Run the focused Vitest file, full frontend Vitest suite, `npx vue-tsc -b --force`, `npm run build`, catalog `-Check`, workspace doctor and `git diff --check`. Record exact commands/results, the mutation result and the final changed set in the BUILD evidence. A test failure or unproven mutation returns `BUILD_BLOCKED`.
5. Update continuity to `REVIEW_PENDING` for Codex, verify the staged paths, and make one local BUILD/evidence commit. Return the commit SHA and evidence path. Claude must not self-approve F1 or mark UX-016/S1 frozen.

## Review and remaining scope

Codex will inspect the exact diff, rerun the focused test and verify the mutation evidence before issuing an independent disposition. UX-016's earlier REVIEW_PASS stands; F1 completion is a separate review. Zalo/legacy/mixed-version crash recovery and the rest of S1 remain open.
