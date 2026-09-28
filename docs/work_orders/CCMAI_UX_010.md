# Work order CCMAI-UX-010 — Job Detail screen redesign

**State:** `CHANGES_REQUIRED` after [independent BUILD review](../reviews/CCMAI_UX_010_INDEPENDENT_REVIEW_2026-09-28.md); repair `UX010-R1` authorized · **Risk:** R2 · **Assignee:** Claude (`REPAIR_WORKER`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** [SPEC](../specs/JOB_DETAIL_SCREEN_UX010_2026-09-28.md) with canvas version `1790540352-11c7`; orchestrator decisions in the [overnight review](../reviews/UI_OVERNIGHT_BUILDS_INDEPENDENT_REVIEW_2026-09-28.md) §"Orchestrator decisions"; dependency gate satisfied by [UX000-R1 re-review PASS](../reviews/UI_FOUNDATION_UX000_R1_INDEPENDENT_REREVIEW_2026-09-28.md) and UX-001a REVIEW_PASS.

## Entry and role route

`ORCHESTRATOR (Codex) → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` (Claude), acknowledged in the active handoff before BUILD. One local BUILD/evidence commit, `REVIEW_PENDING`; no self-approval, FREEZE or push.

## SPEC corrections found while writing this work order (source truth at `005c02c`)

Reading the full `JobDetail.vue` and `engine/analyzer.go` showed two SPEC premises were wrong. The corrections below govern this BUILD and are recorded in the SPEC:

1. **Live progress exists.** While a run is `running`, `job_runs.summary` carries `conversations_found`/`conversations_analyzed`/`conversations_errors`, and the current view already shows a progress bar from them. SPEC §1.7 said there were no live counts; that was wrong. The redesigned screen keeps the progress indicator, labelled as the running run's progress. The canvas "Lịch sử chạy" board shows "—" for a running row; the build shows the real `analyzed/found` from the summary instead.
2. **UX-04 cause.** The metric cards and the list are computed from the same grouped results (each conversation's latest-run results). The baseline gap "Tất cả: 110" vs "Hội thoại đã phân tích: 100" comes from SKIP: 110 = 80 đạt + 20 không đạt + 10 bỏ qua, and the analyzed/pass-rate metrics exclude SKIP. It is **not** two different data sources, as the SPEC and baseline assumed. Fix: the evaluated-count card says it excludes skipped conversations, and its hint shows the total and skipped count. Every count names its scope.
3. **Latest-run default kept.** The orchestrator's accepted decision still gives one coherent scope, so it is implemented as decided. Default scope = the most recent run that has results. The selector offers each run and "Mọi lần chạy" (today's behavior: the latest results per conversation across runs). Metrics, list, source panel and filters all follow the selected scope; the caption names it. This is frontend filtering by `job_run_id` over the existing all-results response. The rationale is now scope clarity, not reconciling two sources. Codex should confirm the default still suits this narrower reason.

## Allowed scope

- `frontend/src/views/Jobs/JobDetail.vue` (rewrite of template and script within the SPEC);
- new `frontend/src/views/Jobs/job-detail/**` (pure helpers and sub-components);
- UX-000 shared components under `frontend/src/components/ui/**`: additive, backward-compatible props only;
- `frontend/src/i18n/vi.ts`, `en.ts`: additive keys only;
- new tests under `frontend/src/__tests__/`;
- evidence `docs/reviews/JOB_DETAIL_SCREEN_UX010_BUILD_2026-09-28.md` + `docs/reviews/assets/ux-010-2026-09-28/**`; this work order; SPEC status/correction note; roadmap status; continuity files.

Not allowed: backend, API, models, stores (`frontend/src/stores/**` unchanged), other views, running database, CVF core, provider calls.

## BUILD requirements

SPEC §1–§5 with the corrections above, plus orchestrator decisions:
- Export buttons are labelled as exporting all runs ("Xuất mọi lần chạy"), because the export endpoint has no run filter.
- Metric-card drill-down applies the matching filter within the current scope (real links via route query).
- Evidence highlighting uses only `detail.evidence_refs` (`message_id` + exact `quote`) resolved against the loaded messages; an unresolvable reference says so and highlights nothing. The old substring heuristic is removed.
- Destructive actions (xóa kết quả, xóa lịch sử chạy) live in the `⋯` menu and always confirm.
- Existing behaviors kept: run dialog modes, test run, cancel, polling, AI-not-configured dialog, attachments/lightbox, classification tags.

## Evidence

`npx vue-tsc -b --force`, `npm run build`, `npm test` (new tests for scope default/filtering, grouping, needs-review, evidence resolution, run-status mapping, SKIP-aware metrics); screenshots desktop/mobile × light/dark via `scripts/ui-screenshots.ps1 -Mode app` including the dialog and run-history states where the tool allows, 0 JS errors; `git diff --check`, catalog `-Check`, doctor. A failing gate that cannot be fixed within scope → `BUILD_BLOCKED`.

## External-effect ceiling

Local source/tests/docs; the disposable screenshot environment (isolated DNS, removed afterwards). No provider call, real channel sync, customer data, persistent `ccma` change, deployment, push or FREEZE.

## REVIEW repair addendum UX010-R1 (2026-09-28)

Independent review found two truth defects; see [UX-010 review](../reviews/CCMAI_UX_010_INDEPENDENT_REVIEW_2026-09-28.md). Same objective, risk and external-effect ceiling apply. Allowed implementation paths: `frontend/src/views/Jobs/JobDetail.vue`, `frontend/src/views/Jobs/job-detail/logic.ts` if a pure helper is useful, additive `jd_*` keys in `frontend/src/i18n/vi.ts` and `en.ts`, and focused tests in `frontend/src/__tests__/`. Also allowed: UX-010 BUILD evidence addendum, this work order, roadmap/continuity/status/catalog sync. Do not alter backend, API, stores, other views or unrelated UI behavior.

1. The QC evaluated card must open a visible filter containing exactly non-SKIP conversations within the selected run scope. Keep existing all, pass, fail and skip filters. Test the actual card interaction and destination count with PASS/FAIL/SKIP data.
2. Run-history summaries must distinguish QC from classification. Classification must never show “đạt” or QC “vấn đề” counts; use verified classification counters and a truthful translation in both languages. Test both job types, including classification with `conversations_passed: 0`.

Record the `REVIEWER → REPAIR_WORKER` role transition and tranche acknowledgment in the active handoff before editing implementation. Rerun focused and full frontend tests, forced vue-tsc, build, i18n parity, diff check, catalog check and doctor; capture affected QC metric and classification history states in the disposable UI environment if available. Return one local repair/evidence/continuity commit as `REVIEW_PENDING` for independent Codex re-review. Do not self-PASS, FREEZE or push.
