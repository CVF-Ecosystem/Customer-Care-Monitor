# Independent REVIEW — CCMAI-UX-010 Job Detail

**Date:** 2026-09-28 · **Reviewer:** Codex (`REVIEWER`, independent of Claude's BUILD) · **BUILD:** `4c8af9e`, continuity repair `44536eb` · **Risk:** R2 · **Disposition:** `CHANGES_REQUIRED` (`UX010-R1`); no FREEZE.

## Scope and checks

I reviewed the SPEC and its recorded corrections, work order, BUILD evidence, Job Detail source and tests, relevant analyzer semantics, and rendered desktop/mobile light/dark captures. I independently reran frontend Vitest (**111/111 passed**) and `npx vue-tsc -b --force` (pass). `git diff 005c02c..4c8af9e --check` passed. The BUILD report lists 36 captures with zero JavaScript errors, horizontal overflow or external requests; the inspected QC, classification and dialog captures match the stated layout. Workspace doctor passed 25/25 at review intake.

The latest-run default is accepted for scope clarity: the selector names the selected run, metrics and list share that scope, and all-run export labels state their real reach. The two SPEC corrections are sound: live run counters exist, and the 110-versus-100 demo difference is explained by ten SKIP results. This review makes no live-provider or CVF governance proof claim.

## Findings

### UX010-R1-F1 — Evaluated card opens the wrong set

`frontend/src/views/Jobs/JobDetail.vue:98` shows `qc.evaluated`, which excludes SKIP, but links to `filterLink('all')`. The `all` filter at `:563–578` includes SKIP. In the captured demo state the card says **100** evaluated while its link opens **110** conversations. This violates the work-order requirement that a metric drill-down apply the matching filter within the selected scope.

**Required:** make the evaluated-card destination show exactly the non-SKIP groups in the current scope, with a visible selected filter and count; keep the all/skip filters available. Add a focused interaction assertion with PASS, FAIL and SKIP fixtures so the card value and destination count agree.

### UX010-R1-F2 — Classification history uses QC pass semantics

`JobDetail.vue:674–678` always renders `jd_run_summary_qc` (`{passed} đạt / {analyzed} đã đánh giá · {issues} vấn đề`) for completed runs. `backend/engine/analyzer.go:478,586` initializes `passed=false` and sets it only for `qc_analysis`; classification never produces a pass count. A real classification run can therefore show “0 đạt” despite successfully classifying conversations. Its tag count is also called “vấn đề.”

**Required:** branch the run-history summary by job type. Preserve QC wording; give classification a truthful analyzed/classified and tag/skip summary using only counters with verified semantics. If `issues_found` is not a verified tag count, omit that count. Add QC and classification history assertions, including a classification run with nonzero analyzed results and zero `conversations_passed`.

## Repair route and limits

This is one consolidated review and repair round 1. The findings fit the existing R2 frontend scope. Claude may transition to `REPAIR_WORKER`, edit only the allowed frontend/view/i18n/tests and UX-010 evidence/continuity paths in the [work-order addendum](../work_orders/CCMAI_UX_010.md), and return a local repair commit as `REVIEW_PENDING`. Codex independently re-reviews; no self-approval, FREEZE, provider call, running-stack change, deployment or push. UX-000/002/001a and R001–R009 remain REVIEW_PASS / FREEZE open; S1 remains IN_PROGRESS.
