# Independent review — CCMAI-UX-012 Messages

**Date:** 2026-09-28 · **Reviewer:** Codex (`REVIEWER`) · **Build commit:** `0ac0dfa` · **Disposition:** `CHANGES_REQUIRED` (R1), not `REVIEW_PASS` or `FREEZE`.

## Verified

- The UX-012 commit changes only its authorized Messages view, additive `msgs_*` translations, focused test, and scoped documentation/evidence. UX-013's current uncommitted files were not changed by this review.
- `npx vitest run src/__tests__/messages-screen.spec.ts`: 7/7 pass in the current shared worktree. The desktop light capture was inspected; counts and PASS/SKIP chip wording match the intended contract. The BUILD's broader test and screenshot claims are recorded in its evidence, not independently repeated here.
- The backend `ListEvaluatedConversations` endpoint returns a map of conversation IDs to their latest severity when available.

## Finding R1-1 — Failed status request produces a false “not analyzed” claim

`frontend/src/views/Messages.vue` catches failure of `GET /conversations/evaluated` in `loadEvaluationMap()` without recording that the map is unavailable. The list renders a chip for every conversation; a missing entry maps to `msgs_chip_none` (“Chưa phân tích”). Consequently, when the status request fails, even evaluated conversations are displayed as “Chưa phân tích”. The pre-build view hid the chip if the map was unavailable. This is a newly introduced false factual label and violates SPEC §1's condition that “Chưa phân tích” is used only when the evaluation is absent.

**Repair acceptance:** Distinguish a successfully loaded empty map from a failed or pending status request. On failure/pending, do not claim a conversation is unevaluated; hiding the chip or displaying an explicit unknown status is acceptable. On a successful map, retain PASS/FAIL/SKIP/other/absent mapping. Add a focused test that rejects the false label when the evaluated-map request fails, and confirms “Chưa phân tích” for an absent entry after a successful response. Preserve API parameters and the UX-012 path boundary. This repair can follow UX-013's commit because the ongoing UX-013 work touches the shared i18n files.

**Next governed move:** Claude completes and commits UX-013 independently, then performs UX-012 R1 under the same work order. Return `REVIEW_PENDING` for Codex re-review. No backend/API, shared component, push, provider call, or FREEZE is authorized by this finding.
