# SPEC — Results source-status note and column header (`CCMAI-UX-002`)

**Date:** 2026-09-28 · **Author:** Claude (`SPEC_AUTHOR`) · **Risk:** R2 (claim boundary of the reviewed R004 source-status display) · **Work order:** [`CCMAI_UX_002`](../work_orders/CCMAI_UX_002.md) · **Finding:** UX-07 in the [UI/UX baseline](../reviews/UI_UX_REVIEW_BASELINE_2026-09-27.md) · **Roadmap:** Phase 1.

## Problem (source truth at `5a850ff`)

`frontend/src/views/Results.vue` shows a source-integrity status on every result, but the local-only caveat `results_source_note` is visible only inside the detail dialog (and as a hover tooltip on card chips). The table's source column header is empty; its name exists only as `aria-label`/`title`. This is the same class of defect Codex required fixing on Job Detail in R006-R1, where the note now sits above the list in both views.

## Required behavior

1. S1. Whenever the Results list renders at least one row (table or card view, desktop or mobile, QC or classification tab), the note `results_source_note` is rendered as visible text directly above the list, before the first row. It is not a tooltip and needs no interaction.
2. S2. It is not shown for the loading skeleton, the "no run yet" state, the "no match" state, or the company-without-jobs state — there is no status to qualify there.
3. S3. The table view's first column header shows the visible text `results_col_source` ("Nguồn"/"Source"). The per-row icon, its tooltip and its color are unchanged.
4. S4. The dialog keeps its existing note. Labels, status values, ordering, filters, pagination, export and API calls are unchanged.

Presentation follows the Job Detail precedent (`v-alert type="info" variant="tonal" density="compact"`), so both screens read alike until the Results screen redesign (`CCMAI-UX-011`) replaces both with the shared `SourceStatusPanel` from UX-000.

## Acceptance

- A component test mounts `Results.vue` with a mocked `api` module (UI-structure only; no governance claim) and asserts: note present above the first row in table view and in card view; header text "Nguồn" in table view; note absent when the result list is empty.
- `vue-tsc -b --force`, `npm run build`, `npm test` pass.
- Screenshots of `/results` desktop/mobile × light/dark from `scripts/ui-screenshots.ps1 -Mode app`, 0 JS errors.

## Out of scope

Any other Results change, the shared component adoption, backend/API, other screens.
