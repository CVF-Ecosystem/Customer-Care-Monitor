# SPEC — AI job list, create and edit (`CCMAI-UX-013`)

**Date:** 2026-09-28 · **Author:** Claude (`ORCHESTRATOR` by owner direction → `SPEC_AUTHOR`; design approved under the owner's design delegation) · **Risk:** R2 · **Work order:** [CCMAI_UX_013](../work_orders/CCMAI_UX_013.md) · **Findings:** UX-12, UX-13, UX-14 in the [baseline](../reviews/UI_UX_REVIEW_BASELINE_2026-09-27.md) · **Roadmap:** screen 6. **Status:** `REVIEW_PASS`, FREEZE open ([review](../reviews/CCMAI_UX_013_INDEPENDENT_REVIEW_2026-09-28.md)).

**Approved design:** canvas https://claude.ai/artifact/P6t4P6pboawbA9PFVTT276 (private to the owner), **version id `1790603381-76c3`**. Artboards: "Tác vụ — 1280 — danh sách", "— 390 dark — danh sách", "— 1280 — tạo, bước quy tắc", "— 390 — tạo, bước đầu ra (email chưa gửi thử được)", "— 390 — trạng thái: tải, rỗng, lỗi, xác nhận xóa, sửa không tải được".

## 0. INTAKE (owner-directed)

Presentation of `/:tenantId/jobs`, `/jobs/create`, `/jobs/:jobId/edit`: `views/Jobs/JobList.vue`, `JobCreate.vue`, `JobEdit.vue`, `components/JobWizard/*` and `components/CronPicker.vue`, which only these two views use. Job Detail (UX-010, reviewed) is not touched. No dependency on UX-012.

## 1. Fixed contract (source truth at `0ac0dfa`)

| Fact | Consequence |
|---|---|
| `GET /jobs` (store `fetchJobs`) returns jobs with `job_type`, `schedule_type` (`cron`/`after_sync`/`manual`), `schedule_cron`, `input_channel_ids` (JSON text), `is_active`, `last_run_at`, `last_run_status`. | The list shows type, schedule in words, channel count, active state and last run, derived only from these fields. |
| `DELETE /jobs/:id` (permission `jobs` d) deletes, in one transaction, the job's runs, results, usage logs and notification logs, then the job. | The confirm dialog says so. |
| Create posts the wizard form (store `createJob`, `outputs` and classification `rules_config` parsed); Edit PUTs the listed fields (store `updateJob`). | Payloads unchanged. |
| `POST /test-output` accepts `type` telegram/email but **only sends Telegram**; email returns 400 `unsupported output type`. The wizard requires every output to pass a test before step 4 continues. | An email output can never pass. The UI says so beside the email output (`BLOCKED_API_CONTRACT`); the gate is not relaxed. |
| JobEdit maps a fetched job into the form; if the fetch fails the old view still rendered the form with defaults, and Save would overwrite the job with them. | On load failure the form is not rendered; an alert with "Thử lại" is shown instead. |
| Run statuses: `running`, `success`, `partial`, `error`/`failed`, `cancelled`, anything else unknown (reviewed mapping `runStatusKind`, UX-010). | Same labels as Job Detail (`jd_run_*`); no run → "Chưa chạy". |

## 2. Intended behavior

1. **List header:** "Tác vụ AI" (UX-13), "{n} tác vụ", "Tạo tác vụ" (same `canEdit('jobs')` condition as today).
2. **List (desktop table / mobile cards):** name (link to detail) and description; type chip "Chất lượng CSKH" / "Phân loại"; schedule in words ("Mỗi ngày lúc 07:00", "Mỗi tuần vào T2, T4 lúc 07:00", "Mỗi tháng ngày 1 lúc 07:00", "Sau mỗi lần đồng bộ", "Chạy thủ công"; an unparseable cron shows "Theo lịch: <cron>"); "{n} kênh"; "Đang bật"/"Đã tắt"; last run chip + `formatDateTime` (UX-12, UX-14); "⋯" menu with "Sửa" and "Xóa tác vụ" (danger) → `ConfirmDialog`. Edit/Delete remain available as today (no permission gating change).
3. **List states:** skeleton while loading; empty "Chưa có tác vụ AI" + "Tạo tác vụ"; load error alert + retry; delete error toast; delete success removes the row (store behavior).
4. **Create:** back link "Tác vụ AI"; title "Tạo tác vụ AI"; desktop step bar with 6 labelled steps (current bold, done filled); mobile "Bước n/6 · <tên>" + progress bar. Footer: Quay lại / Tiếp theo (disabled with the step's reason as `role="status"` text) / "Tạo tác vụ" on the last step. Create failure → alert with the server `error` text when present; success → list (unchanged).
5. **Edit:** back link to the job; title "Sửa tác vụ AI"; five collapsible sections as today; Save/Cancel; success toast then navigate (unchanged); save failure alert; load failure as §1.
6. **Wizard steps:** copy through i18n with "tác vụ" terms. Type step: two option cards (radio semantics). Channel step: channel list with type label and "Đang bật/Đã tắt"; empty → link to Kênh chat. Rules step: QC Markdown + skip conditions + templates; classification rows titled "Nhãn {n}" with "Tên nhãn", "Mô tả", "Mức độ" (existing data). Output step: "Loại", Telegram fields, email fields, template, "Gửi thử" with result; for email the notice from §1 and no test button. Schedule step unchanged in behavior, cron preview as a plain note. Confirm step: summary list (tên, loại, kênh, quy tắc, đầu ra, lịch phân tích, lịch gửi).
7. Theme tokens; no hard-coded colors.

## 3. Current vs intended

List dates `toLocaleString()` (US format) → `formatDateTime`; English status chip → Vietnamese run labels; native English `confirm('Delete this job?')` → `ConfirmDialog` with cascade text; icon buttons without labels → labelled "⋯" menu; "Công việc" → "Tác vụ AI"; no loading/error state → states; create error only in console → alert; edit load failure renders defaults → blocked form; email output silently fails its test → explicit notice.

## 4. Invariants

Store calls and payloads unchanged; the output-test gate unchanged; delete still confirmed; no permission-visibility change; no new status wording beyond the reviewed run labels.

## 5. Held

`BLOCKED_API_CONTRACT`: email output test (`/test-output` sends only Telegram). Not in scope: showing Edit/Delete by permission (permission behavior), toggling `is_active` (no endpoint in UI today).

## 6. Acceptance

vue-tsc, build, full vitest; focused tests with mocked API/store (UI only): schedule describer (daily/weekly/monthly/after_sync/manual/unparseable); list renders type/schedule/channel count/run label/`formatDateTime`; delete goes through the confirm dialog and calls the store only on confirm; list load error + retry; JobEdit load failure renders no form and no Save; email output shows the notice and no test button; create step footer shows the disabled reason. Disposable captures desktop/mobile × light/dark of list, create steps, edit, confirm dialog; 0 JS errors/overflow/external requests. Diff/catalog/doctor.
