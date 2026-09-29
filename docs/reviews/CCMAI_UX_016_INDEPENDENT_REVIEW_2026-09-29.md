# CCMAI-UX-016 — Independent R2 REVIEW

**Date:** 2026-09-29 · **Reviewer:** Claude (independent; did not author the BUILD) · **Target:** `0af8881` (exact range `9d21fd0..0af8881`) · **Authority:** [SPEC](../specs/DASHBOARD_QC_VIOLATION_CARD_UX016_2026-09-29.md), [work order](../work_orders/CCMAI_UX_016.md), [BUILD evidence](DASHBOARD_QC_VIOLATION_CARD_UX016_BUILD_2026-09-29.md), [R017 review](CCMAI_RUNTIME_017_INDEPENDENT_REVIEW_2026-09-29.md).

**Disposition: `REVIEW_PASS` / `FREEZE_OPEN`**, with one non-blocking test-coverage finding (F1).

## Intake

Rehydrated state, handoff, memory, implementation status, SPEC, work order and BUILD evidence at `0af8881`. The worktree was clean and state and handoff agreed. CVF core `26c686c`. Role transition `COMMIT_STEWARD (Codex) → REVIEWER (Claude)` was recorded in the active handoff before source review.

## Scope check

The changed set matches the work order. Source changes are limited to `Dashboard.vue`, additive `dash_qc_violations`/`_hint` keys in `vi.ts`/`en.ts`, and the new `dashboard-qc-card.spec.ts`. The remaining changes are records: SPEC, work order, evidence, continuity and the two roadmaps. There is no backend, API client/store, router or schema change. The request path and `from`/`to` parameters are unchanged.

## Source findings against the SPEC

| SPEC requirement | Observed | Result |
|---|---|---|
| Card reads only `qc_violation_count` | `stats[1]` is set only from `data.qc_violation_count`. No `data.issues` read remains in `Dashboard.vue` (grep). | PASS |
| Nonnegative safe integer, otherwise `—` | `Number.isSafeInteger(q) && q >= 0 ? q : '—'`. Strings, fractions, negatives, `null`, missing, NaN/Infinity and values above 2^53 all fall to `—`. Valid `0` stays `0`. | PASS |
| Request failure → `—` | `catch` resets to `—` for the current request id. The value was already cleared before the request, so this reset is defensive. | PASS |
| Clear on date refresh | `stats[1] = '—'` runs before every request. All date inputs and quick ranges go through `loadDashboard`. | PASS |
| Old interval never shown as current | A request-sequence guard returns early for any non-latest response, so all cards keep the latest data. | PASS (verified by the reviewer, see F1) |
| Tenant change | `App.vue` keys `router-view` by `tenantId`. The Dashboard remounts and starts at `—`, and the old tenant's pending response lands in an unmounted instance. | PASS |
| Labels describe rows, not results or conversations | vi: “Vi phạm QC” / “Số dòng vi phạm QC … một hội thoại có thể có nhiều vi phạm”. en is equivalent. The text makes no AI-correctness or completeness claim. | PASS |
| Other cards, activity, charts unchanged | Assignments are unchanged apart from the stale-response early return, which is correct for all fields. | PASS |

## Independent reruns (from `frontend/`)

- `npx vitest --run src/__tests__/dashboard-qc-card.spec.ts` → 3/3 PASS.
- `npx vitest --run` → 17 files, 170/170 PASS.
- `npx vue-tsc -b --force` → OK. `npm run build` → 569 modules, built.

## Mutation checks

Each mutant was applied to `Dashboard.vue` alone and restored with `git checkout`.

| Mutant | Committed tests |
|---|---|
| M1 invalid value falls back to `data.issues` | caught |
| M2 card reads `data.issues` | caught (3/3 fail) |
| M3 no clear before request | caught |
| M4 remove stale-response guard | **survives** |
| M5 accept negative | caught |
| M6 coerce strings to numbers | caught |
| M7 remove the `catch` reset | survives (equivalent: the value is already `—` from the pre-request clear) |
| M8 missing → `0` | caught |

**F1 (non-blocking, test-coverage).** The BUILD evidence says the request-sequence ID prevents an older response from overwriting a newer date selection, but no committed test exercises out-of-order responses (M4 survives). The SPEC's literal test requirement, clearing during refresh, is covered.

To check the behavior itself, the reviewer ran a temporary test outside the BUILD. It issues two date requests, resolves the newer one first (`7`) and then the stale one (`99`). As built, the card stays `7`. With M4 applied, it shows `99`. The test was deleted afterwards and nothing was committed from it.

The source behavior is therefore correct, but it is not protected by the committed tests. A later tranche may add that test; this review does not block on it.

## Claim limits

These are component tests with synthetic API responses: UI rendering evidence only, not live provider or CVF governance proof. There was no network or provider call. Nothing here shows that a deployed binary serves the field. There was no rendered-browser capture in the BUILD or in this review. The reviewer did not change BUILD source. There was no push and no FREEZE.
