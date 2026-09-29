# CCMAI-UX-016 — Dashboard QC violation card

**Date:** 2026-09-29 · **Phase:** SPEC · **Risk:** R2 · **Authority:** [UI roadmap](../roadmaps/UI_UX_REDESIGN_ROADMAP_2026-09-27.md), [UX-001a](DISPLAY_DATA_FIXES_UX001A_2026-09-28.md), [R017 independent review](../reviews/CCMAI_RUNTIME_017_INDEPENDENT_REVIEW_2026-09-29.md).

## Decision

Finish UX-02 in the Dashboard frontend using R017's reviewed additive `qc_violation_count` API field. The second stat card displays the number of **QC violation rows** for the selected Dashboard date interval. Name it “Vi phạm QC” / “QC violations” and explain in the hint that it counts finding rows in that interval, including multiple findings on one conversation. Do not display the legacy `issues` total under this label.

When `qc_violation_count` is absent, malformed, negative or the Dashboard request fails, show an unavailable mark (`—`), not zero or the `issues` value. A valid numeric zero remains `0`. Clear the previous QC number while a new date/tenant request is loading, so an old interval is never shown as the current interval's QC count.

## Required behavior

1. The second card reads only `qc_violation_count` from a successful `GET /dashboard` response. Accept a nonnegative safe integer; reject strings, fractions, negative values and missing data as unavailable. Preserve the other Dashboard cards, recent activity, date filter and API query parameters.
2. Both Vietnamese and English label/hint describe QC violation **rows**, not all results or unique conversations. Do not repurpose the existing `issues` API contract or claim the count verifies AI correctness or channel completeness.
3. Component-level UI tests with a synthetic API response prove a mixed response (`issues` differs from `qc_violation_count`) shows the QC value, valid zero shows `0`, missing/invalid/error shows `—`, and a date refresh does not retain an old QC count. This is UI structure/contract rendering evidence only.

## Boundaries

Allowed implementation: Dashboard view and its vi/en text, plus focused frontend tests. No backend/API/store/router/schema/provider change, demo reset, release or push. Codex is IMPLEMENTATION_WORKER for this previously parked intersection; Claude must independently review the R2 BUILD before any PASS. FREEZE remains separate. S1 remains IN_PROGRESS.
