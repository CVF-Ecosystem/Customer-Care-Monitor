# SPEC — Results screen redesign (`CCMAI-UX-011`)

**Date:** 2026-09-28 · **Author:** Claude (`SPEC_AUTHOR`; design approved by Claude under the owner's design delegation) · **Risk:** R2 · **Intake:** [UX-011 INTAKE](RESULTS_SCREEN_UX011_INTAKE_2026-09-28.md) · **Work order:** [CCMAI_UX_011](../work_orders/CCMAI_UX_011.md) · **Findings:** UX-07, UX-08, UX-09 (plus UX-17 dialog close) in the [baseline](../reviews/UI_UX_REVIEW_BASELINE_2026-09-27.md). **Status:** `REVIEW_PASS` ([review](../reviews/CCMAI_UX_011_INDEPENDENT_REVIEW_2026-09-28.md)); FREEZE open. **Build detail:** §2.6 "Tác vụ" and "Kênh" render as one two-line column "Tác vụ · Kênh" (≥ lg) because two columns were clipped at 1440 px with the sidebar; both values stay visible.

**Approved design:** canvas https://claude.ai/artifact/RS54UhuYiHahhC9i1qbKFg (private to the owner), **version id `1790597255-fb44`**. Artboards: "Kết quả — desktop 1280, QC, bảng, có nguồn đã đổi", "Kết quả — mobile 390, thẻ", "Kết quả — 390 dark, hộp thoại toàn màn hình", "Kết quả — 1280 hộp thoại chi tiết QC, nguồn đã đổi", "Kết quả — 390 trạng thái: đang tải, trống, chưa chạy, không khớp, lỗi, xuất quá lớn", "Kết quả — 1280 phân loại: nhãn tách khỏi vấn đề". The orange sticky on the canvas lists the held proposals of §5; they are not part of this BUILD. A later canvas change needs a new approval round and a SPEC update.

**Route:** `/:tenantId/results`, `frontend/src/views/Results.vue`. Existing endpoints only: `GET /results/facets`, `GET /results`, `GET /results/export`, and the messages endpoint already used by `useChatTranscript` for the transcript.

## 1. Fixed data contract (source truth at `831a0f4`, `backend/api/handlers/results.go`)

These are inputs, not design choices. The BUILD must not change them.

| Fact | Where | Consequence for the UI |
|---|---|---|
| One row = the **latest** `conversation_evaluation` per (conversation, job) | `baseQuery` "latest" join | Screen subtitle says so; a rerun does not add rows. |
| `total` = rows matching **all** filters including the selected verdict | `ListResults` | Range caption "Hiển thị a–b trên {total} kết quả". |
| `counts.*` = rows matching all filters **except** the verdict filter, over **all pages** | `verdictCounts` (`baseQuery(false)`) | Chip counts are captioned "theo bộ lọc hiện tại, gồm mọi trang". |
| `page_size` 25 (server max 100), page-based | `ListResults` | Unchanged; pagination below the list. |
| `source_integrity_status` computed **only for the rows of the returned page** | `fetchRows → attachSourceIntegrity` | Any source-status count is page-scoped and says "trên trang này (n kết quả)". No global source count. |
| No confidence and no `evidence_refs` in the row (`detail` is `json:"-"`) | `resultRow` | Results shows no confidence and no quote highlighting (see §5). |
| Classification rows put tag evidence in `issues` | `fetchRows` (`classification_tag` with evidence) | Shown as "Căn cứ gắn nhãn", never as QC "Vấn đề". |
| Export = every row matching the current filters, including verdict; rejected with `export_too_large` above `EXPORT_MAX_ROWS` (default 20 000) | `ExportResults` | Button caption: "Xuất mọi kết quả khớp bộ lọc, không chỉ trang đang xem". Existing too-large message kept. |
| Facets: job list per type, channel list, distinct tags; types with jobs decide tabs | `ResultsFacets` | Unchanged. |

## 2. Intended behavior (the BUILD target)

1. **Header.** Title "Kết quả"; subtitle "Mỗi dòng là kết quả mới nhất của một hội thoại trong một tác vụ AI." Export buttons "Xuất CSV" / "Xuất Excel" (desktop, with the export caption under them). On mobile a single "Xuất" button opens a menu with CSV and Excel and the same caption. Hidden when the company has no jobs (current rule).
2. **Tabs** "Chất lượng CSKH" / "Phân loại" only when both types have jobs (current rule). Switching resets job, tag, verdict and score filters (current rule).
3. **Filters (desktop).** Search by customer name, Tác vụ, Kênh, Điểm (QC) or Nhãn (classification), time (preset + date field), sort, "Xóa lọc", view toggle "Bảng"/"Thẻ" (remembered in `localStorage` key `cqa_results_view`, as now). An active filter button shows its count. **Mobile:** search plus "Bộ lọc" button with count, opening the existing bottom sheet. All filter parameters, debounce and page reset stay as now.
4. **Verdict chips** in a `FilterBar`: QC "Tất cả · n", "Không đạt · n", "Đạt · n", "Bỏ qua · n"; classification "Tất cả", "Đã phân loại", "Bỏ qua". Selecting the active chip returns to "Tất cả" (current toggle rule). Chips are real buttons with `aria-pressed`. Caption: "Số đếm theo bộ lọc hiện tại, gồm mọi trang".
5. **Source panel** (`SourceStatusPanel`) directly above the list whenever at least one row is shown, with a new scope line "trên trang này ({n} kết quả)". It keeps the always-visible local-only note (this replaces the UX-002 `v-alert`; `data-testid="results-source-note"` stays on the note). Alert styling when any row on the page is `changed_since_analysis`.
6. **Table (desktop, "Bảng").** Columns: **Nguồn** (visible header; `SourceStatusChip` small with its text label, every status), Khách hàng, then QC: Kết quả (`VerdictChip`), Điểm ("—" for SKIP/null), Vấn đề (rule: evidence summary, 2-line clamp; SKIP shows "Bỏ qua: {lý do}" muted) — or classification: Nhãn (tag chips; "—" when none). Then Ngày hội thoại, Tác vụ, Kênh (the last two ≥ lg). Rows with `changed_since_analysis` get a left warning edge and tint; `verification_unavailable` rows get a left edge in its color. The whole row opens the dialog; rows are keyboard reachable (Enter/Space).
7. **Cards ("Thẻ" on desktop, always on mobile).** One `ResultCard` per row: name, time, verdict chip, source chip, score (QC non-SKIP only), summary (QC: review, or "n vấn đề · rule names" when failing; classification: "Nhãn: a, b"), meta "tác vụ · kênh". A card opens the dialog. The in-place expansion is replaced by the dialog, which carries the transcript (same data, same endpoint).
8. **Range and pagination.** "Hiển thị {from}–{to} trên {total} kết quả" plus `v-pagination` (7 visible desktop, 3 mobile), shown when there are rows.
9. **Detail dialog** (`AppDialog`, full screen on mobile, close button, Esc). Title: customer name. Subtitle: "tác vụ · kênh". Meta: verdict chip (QC) or "Đã phân loại"/"Bỏ qua" (classification), score (QC non-SKIP), source chip. Body, two columns on desktop (transcript left) and stacked on mobile (details first, transcript after):
   - **Source block:** title "Trạng thái nguồn dữ liệu"; for `changed_since_analysis` the line "Hội thoại hiện tại khác bản đã đánh giá, nên nhận xét dưới đây có thể không còn đúng."; for `verification_unavailable` "Không đối chiếu được hội thoại hiện tại với bản đã đánh giá. Nên xem lại trước khi dựa vào nhận xét này."; for the other two only the chip; always the local-only note.
   - **Dates:** Ngày hội thoại, Ngày đánh giá.
   - **Nhận xét** with `AiGeneratedLabel` when a review exists.
   - **QC:** "Vấn đề ({n})" list — rule name and quoted evidence text; PASS with none: "Không có vấn đề." **Classification:** "Nhãn ({n})" tag chips, then "Căn cứ gắn nhãn" (rule name + evidence). Classification never shows "Vấn đề", pass/fail or a score.
   - **Transcript:** existing `useChatTranscript` rendering (attachments, image lightbox, permission-denied and empty messages), with "Xem tại Tin nhắn".
   - Footer: "Mở tác vụ" (link to the job), "Đóng".
10. **States.** Facet loading: skeleton. No jobs: existing empty card with "Tác vụ AI". Jobs but no results: existing "chưa chạy" card. No match: existing message plus a "Xóa lọc" button. **Load error:** inline `role="alert"` block "Không tải được kết quả. Thử lại sau." with "Thử lại" (re-runs the same request), instead of only a snackbar; for a facets failure the same block retries facets then results. Export errors keep the snackbar (generic or too-large with the server limit).
11. **Dark mode** at parity through theme tokens only; no hard-coded colors in the view.

## 3. Current implementation vs intended (source truth at `831a0f4`)

| Area | Current | Intended |
|---|---|---|
| Source note | UX-002 `v-alert` above list | `SourceStatusPanel` with page-scoped counts and the same note |
| Source column | Icon only, tooltip for the label (UX-09) | Visible chip text for every status |
| Changed rows | Red icon only | Row edge + tint and bold chip |
| Chip counts | No scope caption | Scope caption |
| Page range | None | "Hiển thị a–b trên n" |
| Export scope | Unlabelled | Caption states full-filter export |
| Cards | In-place expansion, Vuetify chips | `ResultCard` + dialog |
| Dialog | No transcript; classification evidence under "Vấn đề"; no AI label | Transcript; "Căn cứ gắn nhãn"; AI label; per-status explanation |
| Load error | Snackbar only; list may look empty | Inline alert with retry |
| Colors | `text-grey`, `bg-blue-lighten-5` etc. | Theme tokens |

## 4. Semantic invariants (review checklist)

- All four source statuses keep their glossary labels and styles from `SourceStatusChip`; none reads as verified or safe; unknown values render as "Không xác minh được".
- The local-only note is visible above every populated list (table and cards, both tabs) and in the dialog; absent in loading/empty/no-match/error states.
- No count on the screen claims a wider scope than its source: verdict chips = all pages under current filters; source counts = this page; range = `total`.
- Filters, sorting, pagination and exports stay server-side with identical query parameters.
- No confidence is shown on this screen (the API has none); `ConfidenceText` is not rendered with a missing value.
- QC "Vấn đề" and classification "Nhãn"/"Căn cứ gắn nhãn" never share a label.
- The "Cần xem lại" filter is **not** offered (it needs server-side source status; §5).

## 5. Held proposals — `BLOCKED_API_CONTRACT` (not in this BUILD)

Each needs a separate Codex-routed API tranche; the UI keeps the current contract meanwhile.

1. **"Cần xem lại" filter / filter by source status.** Source status is computed per returned page; a correct filter and its count need server-side evaluation over the full filter scope.
2. **Global source-status counts** (e.g. "3 nguồn đã đổi trong 110 kết quả"). Same reason; the panel stays page-scoped.
3. **Confidence on Results.** `/results` returns no `confidence`/`confidence_basis`; Job Detail shows it.
4. **Quote highlighting in the transcript.** `/results` does not expose `evidence_refs`.
5. **Page-only or run-scoped export.** The export endpoint exports the full filter; any other scope needs a new parameter or a separately specified client export.

Dashboard `qc_violation_count`, demo-channel scheduler/sync and every backend/API/store/schema/provider change remain outside UX-011 as recorded in the INTAKE.

## 6. Accessibility

Touch targets ≥ 44 px on mobile (cards, filter button, export, dialog close, pagination via Vuetify density). Chips and view toggle expose `aria-pressed`; the chip row is a labelled toolbar. Table rows have `tabindex="0"` and open on Enter/Space. Status is never conveyed by color alone (icon + text). Text contrast via UX-000 token pairs (AA).

## 7. Acceptance

- `npx vue-tsc -b --force`, `npm run build`, `npm test` (full suite).
- Focused tests with a mocked `api` (UI structure only, no governance claim): note above list in table and card view and absent in empty states (UX-002 tests kept green); page-scoped source panel counts and scope text; visible text source chips in the table; range caption from `page`/`total`; count caption; export caption; unchanged request parameters for list and export; classification dialog shows "Căn cứ gắn nhãn" and not "Vấn đề"; QC dialog per-status explanation for changed/unavailable; no confidence element; load error alert + retry re-requests; no-match "Xóa lọc" clears filters.
- i18n parity (`vi`/`en`).
- Disposable `scripts/ui-screenshots.ps1 -Mode app` captures of `/results` desktop/mobile × light/dark, plus the detail dialog (changed source) and the classification tab where the tool allows; 0 JS errors, 0 horizontal overflow, 0 external requests.
- `git diff --check`, catalog `-Check`, workspace doctor.

## 8. Out of scope

Backend, API, stores, schema, provider, other screens (Dashboard, Channels, Messages, Jobs, Job Detail), human override/disputes, and every item in §5.
