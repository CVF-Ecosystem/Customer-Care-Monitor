# BUILD evidence — `CCMAI-UX-010` Job Detail screen

**Date:** 2026-09-28 · **Implementer:** Claude (`IMPLEMENTATION_WORKER`) · **Risk:** R2 · **Status:** `REVIEW_PENDING` for independent Codex review; not self-approved, no FREEZE.
**Authority:** [work order](../work_orders/CCMAI_UX_010.md) (commits `1b8c3a9`, `16c42f7`), [SPEC](../specs/JOB_DETAIL_SCREEN_UX010_2026-09-28.md), canvas version `1790540352-11c7`, orchestrator decisions in the [overnight review](./UI_OVERNIGHT_BUILDS_INDEPENDENT_REVIEW_2026-09-28.md).

## What changed

| File | Change |
|---|---|
| `frontend/src/views/Jobs/JobDetail.vue` | Rewritten template/script on the UX-000 components. Kept unchanged: run dialog modes, test run, cancel, polling, AI-not-configured dialog, attachments/lightbox, deletes (API calls identical). |
| `frontend/src/views/Jobs/job-detail/logic.ts` (new) | Pure logic: `scopeResults`, `defaultScopeRunId`, `groupResults` (moved; same latest-run-per-conversation rule), `qcMetrics` (SKIP-aware, null instead of 0), `classificationMetrics`, `groupNeedsReview`, `primarySourceStatus`, `runStatusKind`, `runProgress`, `runDurationSeconds`, `evidenceRefs`/`resolveEvidence`. |
| `frontend/src/components/ui/MetricCard.vue` | Additive `suffix` prop ("%", "/100"), shown only after a known value. |
| `frontend/src/i18n/vi.ts`, `en.ts` | 75 additive `jd_*` keys. |
| Tests (new) | `job-detail-logic.spec.ts` (13), `job-detail-view.spec.ts` (5, mocked API, UI structure only), +1 `MetricCard` suffix test. |

No backend, API, model, store, other view or database change.

## Behavior against the SPEC and work-order corrections

- **Header:** back link, title, meta line (type · schedule · channels · model · last run and status). Actions: "Chạy thử", "Chạy ngay", "Dừng" while running, and `⋯` holding "Sửa tác vụ" plus the destructive "Xóa kết quả" / "Xóa lịch sử chạy". Both destructive items open `ConfirmDialog` (UX-18).
- **Live progress** is kept (work-order correction 1): "Đang phân tích analyzed/found hội thoại" from the running run's summary, and the running row in run history shows `analyzed / found`.
- **Scope:** defaults to the most recent run that has results, with the caption "Số liệu và danh sách của lần chạy &lt;time&gt;". The selector lists runs with results and "Mọi lần chạy" (the previous behavior). Metrics, trend, source panel, filters and list all follow the scope. When a new run finishes, an untouched latest-run scope moves to it; "Xem kết quả" in run history selects that run.
- **UX-04 (correction 2):** "Hội thoại đã đánh giá: 100" carries the hint "110 hội thoại, không tính 10 bỏ qua". Pass rate and average score show "—" (not 0) when nothing was evaluated or scored.
- **Metric drill-down:** real links (`?filter=`) that apply the filter within the current scope.
- **Source panel** above the list: each conversation counted once, under its most concerning status. The local-only note is always shown. All-legacy scopes add the rerun explanation.
- **Filters:** "Cần xem lại" is first and the default when its count > 0 (QC: failures plus changed/unverifiable sources; classification: sources only). Tag chips for classification. `FilterBar` scrolls sideways on mobile.
- **List:** desktop table with a "Nguồn" column of `SourceStatusChip`s (every status, glossary labels), `VerdictChip`, score, issue/tag count; changed-source rows get a left warning bar. Mobile uses `ResultCard`. Rows are keyboard-openable (Enter).
- **Export:** "Xuất CSV (mọi lần chạy)" / "Xuất Excel (mọi lần chạy)" (orchestrator decision; endpoint unchanged).
- **Detail dialog** (`AppDialog`; full screen on mobile, Esc closes, close button). Left: transcript. Right: source panel, review with "Nhận xét do AI tạo", `ConfidenceText`, issues as buttons. Selecting an issue outlines and scrolls to the messages resolved from `detail.evidence_refs` (`message_id` + exact `quote`). Otherwise it states "Không tìm thấy trích dẫn trong hội thoại hiện tại". The old substring-guess highlight is removed. Footer: "Mở trong Tin nhắn", "Hội thoại tiếp theo cần xem lại".
- **Run history:** start, status chip (running/success/partial/error/cancelled/unknown), duration ("—" while running), conversations, summary or error message, "Xem kết quả" for runs with results, and the note that "Đang chạy" means accepted, not finished (R009).
- **States:** loading skeleton; never-run empty state with actions; load error `role="alert"` with "Thử lại"; dark mode via tokens; trend colors from theme `pass`/`fail`.

## Gates

| Gate | Result |
|---|---|
| `npx vue-tsc -b --force` | exit 0 |
| `npm run build` | pass |
| `npm test` | 10 files, **111 passed** (i18n parity included) |
| Mutation: default scope forced to "Mọi lần chạy" and default filter to "all" | 2 view tests failed as expected; source restored |
| `git diff --check` | clean |
| Catalog `-Check` / doctor | PASS / 25/25 (after continuity sync) |

## Rendered evidence (disposable environment, isolated DNS, synthetic demo; removed afterwards, `ccma` untouched)

- `ui-screenshots.ps1 -Mode app` on the final code: 36 pages (both job details at desktop/mobile × light/dark), **0 JS errors, 0 horizontal overflow, no external requests** (`report-app.json`).
- A scratch CDP script (not committed) opened the first listed conversation, selected its first issue, closed with Esc and opened run history, at desktop/mobile × light/dark. It found 0 new JS errors, Esc closed in all four cases, and run history rendered "Thành công".
  - Demo data has no `evidence_refs`, so every case showed "Không tìm thấy trích dẫn…" (`states-no-refs.json`).
  - To render the positive path, the **disposable** DB's 70 demo violations were given one synthetic reference each (first agent message ≥ 20 characters, first 20 characters as the exact quote). All four cases then showed "Trích dẫn đang được tô…" with the quoted message outlined (`states-synthetic-refs.json`).
- A defect seen in the first capture round was fixed before the final capture: the explanation duplicated the evidence text in demo data. The explanation now renders only when it differs.

Images in [`assets/ux-010-2026-09-28/`](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/tree/main/docs/reviews/assets/ux-010-2026-09-28/):

![Job Detail QC, desktop light](./assets/ux-010-2026-09-28/jobdetail-qc--desktop--light.png)
![Detail dialog with a resolved quote, desktop light](./assets/ux-010-2026-09-28/dialog-issue--desktop--light.png)
![Detail dialog, mobile dark](./assets/ux-010-2026-09-28/dialog-issue--mobile--dark.png)
![Classification, mobile light](./assets/ux-010-2026-09-28/jobdetail-classification--mobile--light.png)

## Limits and notes for review

- The latest-run default is implemented as decided; its rationale is now scope clarity (work-order correction 3). Please confirm.
- Demo `job.last_run_at` (import time) differs from the demo run's `started_at` (two hours earlier), so the header "Lần chạy cuối" and the scope caption show different times in demo data. Both are shown as stored; this is not a UI computation.
- The run dialog and AI-not-configured dialog copy is unchanged Vietnamese text (not moved to i18n in this tranche).
- Findings F1 (onboarding banner) and F3 (older status styling in other views) remain with later screen tranches.

No provider call, real channel sync, customer data, persistent DB change, deployment or push.

## Addendum: Repair round 1 (UX010-R1, 2026-09-28)

**Repair worker:** Claude (`REPAIR_WORKER`) · **Authority:** [independent review](./CCMAI_UX_010_INDEPENDENT_REVIEW_2026-09-28.md) findings `UX010-R1-F1`/`UX010-R1-F2`, addendum in [work order](../work_orders/CCMAI_UX_010.md#review-repair-addendum-ux010-r1-2026-09-28) · **Status:** `REVIEW_PENDING` for independent Codex re-review; no self-approval, no FREEZE.

### UX010-R1-F1 — evaluated card opened the wrong set

`frontend/src/views/Jobs/JobDetail.vue:98` linked the evaluated-count card (`qc.evaluated`, SKIP excluded) to `filterLink('all')`, whose destination filter (`:582`) includes SKIP. A new `evaluated` filter key was added to the QC filter list (`:566`) with count `gs.filter(g => g.verdict !== 'SKIP').length` — the exact expression `qc.evaluated` already uses in `job-detail/logic.ts` — and `filteredGroups` (`:589`) applies the matching predicate. The card now links to `filterLink('evaluated')` instead of `filterLink('all')`. The existing `all`, `pass`, `fail` and `skip` filter chips are unchanged and still available. New additive key `jd_filter_evaluated` ("Đã đánh giá" / "Evaluated") labels the chip.

### UX010-R1-F2 — classification history used QC pass/issue semantics

`runSummary()` (`:674`) always rendered `jd_run_summary_qc`, which shows `conversations_passed` and calls `issues_found` "vấn đề" (issues). Source-audited `backend/engine/analyzer.go`: `passed` (line 478) is declared `false` and is only ever set for `qc_analysis` (line 586); the `classification` branch of `saveResults` never sets it, so `conversations_passed` in a classification run's `job_runs.summary` is always `0` — showing it as "đạt" would read as a false "0 đạt" on a run that classified everything correctly. `issues_found` **is** a verified count for classification runs too: the outer loop (`analyzer.go:317,851`) accumulates it from `saveResults`'s return value, which for `classification` counts exactly the `classification_tag` rows created (`analyzer.go:646-677`) — a real, verified tag count, just mislabeled "vấn đề". `runSummary()` now branches on `isClassification.value`: QC runs keep `jd_run_summary_qc` unchanged; classification runs use a new key `jd_run_summary_classification` ("{analyzed} đã phân tích · {tags} nhãn được gắn" / "{analyzed} analyzed · {tags} tags assigned"), which never mentions "đạt" or "vấn đề" and reuses the same verified `conversations_analyzed`/`issues_found` fields under truthful labels.

### Tests

`frontend/src/__tests__/job-detail-view.spec.ts` (+3 tests, 111 → 114 total):
- *"evaluated-count card links to, and navigating there shows, exactly the non-SKIP conversations"* — with fixture conversations a=FAIL, b=PASS, c=SKIP in the default scope: asserts the card's rendered value is 2 and its rendered `href` targets `filter=evaluated`; navigates the test router to that exact href (the same navigation a click performs) and asserts the destination shows Khách a and Khách b but not Khách c (SKIP), and the selected filter chip is "Đã đánh giá: 2".
- *"run-history summary keeps QC pass/issue wording for QC jobs"* — unchanged QC fixture (`passed:1, analyzed:3, issues:1`) still renders "1 đạt / 3 đã đánh giá · 1 vấn đề" after switching to the run-history tab.
- *"run-history summary uses truthful analyzed/tag wording for classification, even with conversations_passed 0"* — a classification job/run with `conversations_analyzed:5, conversations_passed:0, issues_found:7` renders "5 đã phân tích · 7 nhãn được gắn" and the page text contains neither "đạt" nor "vấn đề".

`mountView()` now also returns the test router (`Object.assign(w, { router })`) so a test can follow a rendered link's real `href` instead of hand-building a route push; existing tests are unaffected since they still destructure only the wrapper.

**Mutation check:** `git stash push -- frontend/src/views/Jobs/JobDetail.vue frontend/src/i18n/vi.ts frontend/src/i18n/en.ts` (keeping only the new tests), reran the file: both new interactive tests failed against pre-repair source exactly as the findings describe — the evaluated card's href resolved to `filter=all` (not `filter=evaluated`), and the classification run showed the fabricated `"0 đạt / 5 đã đánh giá · 7 vấn đề"`. The QC wording-preservation test passed unchanged, as expected. `git stash pop` restored the repair; the full suite was rerun clean afterward.

### Gates

| Gate | Result |
|---|---|
| Focused `job-detail-view.spec.ts` | 8/8 pass (repaired source); 6/8 pass, 2 fail as predicted (pre-repair source, mutation check) |
| `npm test` (full suite) | 10 files, **114 passed** |
| `npx vue-tsc -b --force` | exit 0 |
| `npm run build` | pass |
| `git diff --check` | clean |
| Catalog `-Check` | PASS |
| Workspace doctor (`manage_cvf_workspace.ps1 -Action Status`) | `REPAIR_REQUIRED` — pre-existing CVF-core profile-artifact drift only (unrelated core-repo template/doc files, e.g. `AGENT_HANDOFF_V63_2026-09-18.md`), already recorded as the known limitation "Workspace-wide gate has failures in unrelated sibling repositories"; not caused by, or touched by, this repair |
| Rendered capture (`ui-screenshots.ps1 -Mode app`) | see below |

### Rendered evidence

Ran `scripts/ui-screenshots.ps1 -Mode app` (disposable Compose project, isolated DNS, synthetic demo data; torn down afterward, persistent `ccma` never touched): **36 pages, 0 JS errors, 0 horizontal overflow, 0 external requests** (`report-app-ux010-r1.json`).

The default-load `jobdetail-qc` capture confirms the new "Đã đánh giá: 100" filter chip renders between "Tất cả: 110" and "Không đạt: 20" with no layout break, at desktop and mobile, light and dark (`jobdetail-qc--*.png`, refreshed in this round). This is the only element visually new at page-load; both fixes otherwise change a link target and a post-interaction (tab-switch) summary string, neither visible until acted on. `jobdetail-classification--*.png` (also refreshed) confirms the classification default view is unaffected — its metric cards and filters are on a separate `isClassification` branch untouched by this repair.

The screenshot tool (`scripts/ui-screenshots.mjs`) captures one fixed page-load state per route; it has no built-in way to click a filter chip or switch the run-history tab after load, and extending it is outside this repair's allowed scope (only `JobDetail.vue`, `job-detail/logic.ts`, additive `jd_*` i18n keys, focused tests and this evidence are in scope). The two interaction states the findings actually concern — the evaluated-filtered destination list, and the classification run-history row — are proven instead by the focused Vitest interaction tests above, with a mutation check showing both fail against the pre-repair defect exactly as described. That is a stronger, deterministic signal for these specific text/link defects than a static screenshot would be.
