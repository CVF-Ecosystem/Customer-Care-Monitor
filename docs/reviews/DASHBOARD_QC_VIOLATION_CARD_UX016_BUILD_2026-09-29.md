# CCMAI-UX-016 BUILD — Dashboard QC violation card

**Date:** 2026-09-29 · **Builder:** Codex (`IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD`) · **Disposition:** `REVIEW_PENDING` for independent Claude R2 review · **Authority:** [SPEC](../specs/DASHBOARD_QC_VIOLATION_CARD_UX016_2026-09-29.md), [work order](../work_orders/CCMAI_UX_016.md), [R017 review](CCMAI_RUNTIME_017_INDEPENDENT_REVIEW_2026-09-29.md).

## Change

- The second Dashboard card reads the reviewed `qc_violation_count` field. Its vi/en label and tooltip describe QC violation rows in the selected period, including multiple findings per conversation. The legacy `issues` field is not used for that card or modified in the API.
- The card starts as `—`, clears to `—` before each Dashboard request, and stays `—` on request failure or if the field is missing, not a nonnegative safe integer, or otherwise invalid. A valid zero remains `0`. A request sequence ID prevents an older request from overwriting a newer date selection.
- Other Dashboard values, date parameters, recent activity and charts retain their existing sources. No backend, API client/store, router, schema or provider change.

## Checks

- `npm test -- --run src/__tests__/dashboard-qc-card.spec.ts` from `frontend/` → 1 file, 3 tests PASS. The mounted Dashboard received synthetic responses with `issues=91`, `qc_violation_count=4`, proving it shows 4 rather than 91; valid zero, missing/string/negative/fraction values, request failure and clearing during date refresh were checked.
- `npm run build` from `frontend/` → exit 0, `vue-tsc -b` and Vite build PASS (569 modules transformed). No live API key or provider call.
- Catalog `-Check`, workspace doctor, JSON parsing and `git diff --check` after continuity synchronization: recorded in the handoff.

## Limits

The test uses synthetic API responses for UI contract rendering only; it is not real-provider or CVF governance proof. It does not prove a deployed binary serves the new API field. R017's backend COUNT was separately reviewed; deployment and FREEZE remain open. Claude must independently review this R2 BUILD before PASS. No push or release.
