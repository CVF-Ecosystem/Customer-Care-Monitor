# SPEC — Job Detail screen redesign (`CCMAI-UX-010`)

**Date:** 2026-09-28 · **Author:** Claude (`SPEC_AUTHOR`; design approved by Claude under the owner's design delegation) · **Risk:** R2 · **Status:** SPEC ready; **BUILD not authorized**. The roadmap forbids implementing any screen before Phase 0 (`CCMAI-UX-000`) passes independent review, and this tranche also depends on `CCMAI-UX-001a`. A work order will be written after both are REVIEW PASS.

**Approved design:** canvas https://claude.ai/artifact/KkF8P2m1iGRpTRa3mkf4ii (private to the owner), **version id `1790540352-11c7`**. Artboards: "Job Detail — desktop", "Job Detail — mobile 390px", and the 2026-09-28 additions "1280 — hộp thoại chi tiết, nguồn đã đổi", "1280 — lịch sử chạy", "1280 — phân loại", "390 — đang tải, chưa chạy, lỗi, toàn kết quả cũ", "390 — dark, nguồn đã đổi". A later canvas change needs a new approval round and a SPEC update.
**Findings addressed:** UX-04, UX-08, UX-09, UX-10, UX-17, UX-18 (plus UX-13/14 terms and formats via UX-000). **Route:** `/:tenantId/jobs/:jobId`, `frontend/src/views/Jobs/JobDetail.vue`.

## 1. Screen structure

1. **Header.** Back link "Tác vụ AI", title, one meta line (type · schedule · channel count · model · last run time and status). Actions: "Chạy ngay" (primary), "Chạy thử (3 hội thoại)", and an `ActionMenu` "⋯" holding "Sửa" and the destructive "Xóa kết quả" (UX-18). "Xóa kết quả" always opens `ConfirmDialog`.
2. **Metrics.** Four `MetricCard`s under the caption "Số liệu của lần chạy mới nhất (<date time>)". QC: hội thoại đã đánh giá, tỉ lệ đạt, vấn đề phát hiện, điểm trung bình. Classification: đã phân loại, bỏ qua, nhãn nhiều nhất, khiếu nại. Each card links to the list filtered accordingly. Missing values show "—".
3. **Trend** (QC only). Chart grouped by Vietnam day (`qualityTrendByDay`, UX-001a), colors from theme tokens (`pass`/`fail`), not hard-coded hex.
4. **Tabs.** "Kết quả đánh giá" and "Lịch sử chạy".
5. **Results tab.**
   - `SourceStatusPanel` above the list (local-only note always visible, counts per status, alert styling when anything changed).
   - `FilterBar`, with "Cần xem lại" first and default when count > 0 (`needsReview`). Then Tất cả / Không đạt / Đạt / Bỏ qua (QC), or tag filters (classification).
   - Desktop table: Khách hàng, Ngày chat, Kết quả (`VerdictChip`), Nhận xét, Điểm, Vấn đề / Nhãn, **Nguồn dữ liệu** (`SourceStatusChip`, every status, glossary labels).
   - Mobile: one `ResultCard` per conversation. Changed-source cards get the left warning border.
   - Export CSV/Excel stays.
6. **Detail dialog** (`AppDialog`, full screen on mobile; UX-17). Header shows customer · time · channel, verdict, score, source chip and close button. Left column: transcript; right column: source-status panel, AI review with "Nhận xét do AI tạo", `ConfidenceText`, and issues. Selecting an issue scrolls to and outlines the quoted message using the `evidence_refs` already stored in each violation/tag result's `detail` JSON (`engine/analyzer.go`); demo and older results may have none. If an issue has no resolvable reference, it says "Không tìm thấy trích dẫn trong hội thoại hiện tại" rather than highlighting a guess. Footer actions: "Mở trong Tin nhắn", "Hội thoại tiếp theo cần xem lại".
7. **Run history tab.** Table (mobile: cards) of runs: start, status (`running` = "Đang chạy", `success` = "Thành công", `error`/`failed` = "Lỗi", `cancelled` = "Đã hủy", `partial` = "Một phần", anything else "Không rõ"), duration, conversations, summary, action (cancel only for running). No live progress counts: none exist in the API.
8. **States.** Loading skeleton with metrics "—"; "never run" empty state with "Chạy thử"/"Chạy ngay"; load error `role="alert"` with "Thử lại"; all-legacy panel explaining that a rerun produces comparable results. Dark mode at parity.

## 2. Semantic invariants (review checklist)

- All four source statuses stay distinct with glossary labels. The canvas desktop board's shorthand "Kết quả cũ" is **replaced** by "Chưa xác minh (kết quả cũ)" in the build. None reads as reassurance.
- Confidence: "Không có" unless model-reported and uncalibrated, then with the qualifier (R005).
- A 202 from run/test-run means "đã bắt đầu", never "hoàn tất" (R009).
- Delete is only reachable through "⋯" and a confirm dialog.

## 3. UX-04 resolution (design decision; review attention)

Baseline: "Tất cả: 110" (list: grouped conversations across **all** runs) disagrees with "Hội thoại đã phân tích: 100" (latest run summary). Decision: the Results tab defaults to the **latest run** so list counts and metrics describe the same run. A control "Lần chạy: Mới nhất ▾" switches to a specific earlier run or "Mọi lần chạy", and the caption always names the scope. This is frontend filtering by `job_run_id` on data the API already returns. No API change. If Codex judges the default change unacceptable, the fallback is to keep all-runs as default and caption both numbers with their scope.

## 4. Data and scope

Existing endpoints only: job, runs, all job results, cancel, trigger/test-run, export, delete. No backend, API, schema or store-contract change. Allowed build paths (for the future work order): `JobDetail.vue`, new sub-components under `frontend/src/views/Jobs/job-detail/`, UX-000 shared components (additive props only), i18n additive keys, tests and screenshots.

## 5. Acceptance (for the future BUILD)

`vue-tsc -b --force`, build, vitest for scope filtering, needs-review default, evidence resolution (found / not found) and run-status mapping; screenshots desktop/mobile × light/dark for data, loading, empty, error, changed-source, all-legacy and classification states, with 0 JS errors; contrast and 44 px checks; side-by-side with baseline images 06–10 and 23.

## 6. Out of scope

Human override/disputes/assignment (backend-dependent, decision §2.6), Results page redesign (`CCMAI-UX-011`), backend count changes (UX-02 recount BLOCKED_API_CONTRACT).
