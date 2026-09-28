# SPEC — Messages screen redesign (`CCMAI-UX-012`)

**Date:** 2026-09-28 · **Author:** Claude (`ORCHESTRATOR` by owner direction → `SPEC_AUTHOR`; design approved by Claude under the owner's design delegation) · **Risk:** R2 · **Work order:** [CCMAI_UX_012](../work_orders/CCMAI_UX_012.md) · **Finding:** UX-16 (plus UX-12/13 formatting and terms) in the [baseline](../reviews/UI_UX_REVIEW_BASELINE_2026-09-27.md) · **Roadmap:** screen 5. **Status:** `REVIEW_PASS` after R1, FREEZE open ([review](../reviews/CCMAI_UX_012_R1_INDEPENDENT_REREVIEW_2026-09-28.md)).

**Approved design:** canvas https://claude.ai/artifact/Q7KkTJMS95U9ShLJrfP9F4 (private to the owner), **version id `1790601057-3d67`**. Artboards: "Tin nhắn — 1280 — danh sách + hội thoại", "— 1280 — tab đánh giá chất lượng", "— 390 — danh sách", "— 390 dark — hội thoại, tab phân loại", "— 390 — trạng thái: tải, rỗng, không khớp, lỗi, xuất tin nhắn". The orange sticky lists the held items of §5.

## 0. INTAKE (owner-directed, 2026-09-28)

Frontend presentation of `/:tenantId/messages` (`frontend/src/views/Messages.vue`) on the existing conversation APIs. Out of scope: Channels screen, sync, backend/API/store/schema/provider, permissions, data contracts. A later tranche (UX-013..015) does not depend on this one: they touch disjoint views, and this tranche changes no shared component.

## 1. Fixed data contract (source truth at `0cb0483`, `backend/api/handlers/conversations.go`, `engine/analyzer.go`)

| Fact | Consequence |
|---|---|
| `GET /conversations` pages conversations (`per_page` 9 here), newest `last_message_at` first. `total` counts **conversations**. Filters: `channel_type`, `channel_id`, `search` (customer name LIKE), `evaluation`. | Heading shows "n hội thoại", never "tin nhắn" (UX-16). Range "Hiển thị a–b trên n hội thoại". |
| `evaluation`: `evaluated` / `not_evaluated` = has any `conversation_evaluation` of **any job type**. `PASS` / `FAIL` = severity of the **latest** evaluation of any job type. | Labels say "phân tích AI" and "lần gần nhất". |
| Classification writes `conversation_evaluation` with severity **`PASS`** (tags matched) or `SKIP`; QC writes the QC verdict. | A latest `PASS` means "QC passed **or** classified" and is labelled **"Đạt / đã phân loại"**, never plain "Đạt". `FAIL` → "Không đạt". `SKIP` → "Bỏ qua". Other/absent → "Chưa phân tích" only when absent; an unknown value shows "Đã phân tích". |
| `GET /conversations/evaluated` = map conversation → latest severity (any type). | List chip uses the same mapping. |
| `GET /conversations/:id/evaluations` = groups per run with `job_name`, `job_type`, `evaluated_at`, raw `results`; **no `source_integrity_status`, no job id**. | QC/classification tabs show a note that this screen does not check source status and link to Kết quả. No per-group job link. No source chip is invented. |
| `GET /conversations/:id/messages` = messages ascending + conversation meta. | Transcript unchanged. |
| `GET /conversations/export` (permission `messages` w): conversations whose **last message** falls in `[from, to]`, all their messages, optional `channel_type`; TXT or CSV. With no conversations it answers **200 with JSON `{error}`**. | Dialog text states the selection rule and that list filters do not apply. The client must detect the JSON error body and show it instead of downloading it (a defect in the current view). |
| `GET /conversations/:id/page` for deep links (`?conv=`, `?tab=`). | Kept. |

## 2. Intended behavior

1. **Header:** "Tin nhắn", subtitle "{n} hội thoại · mới có tin nhắn gần đây ở trên". "Xuất tin nhắn…" button (text label, shown only with `messages` edit permission as now).
2. **Filters:** search by customer name (debounce 300 ms, as now), Loại kênh, Kênh (options narrowed by type, as now), "Phân tích AI" select with: Đã phân tích · Chưa phân tích · Lần gần nhất: Đạt / đã phân loại · Lần gần nhất: Không đạt. Same query parameters. "Xóa lọc" when any filter is set. On mobile the three selects sit in a "Bộ lọc" bottom sheet with an active-count; search stays visible.
3. **List:** one row per conversation (a real button, ≥ 44 px): channel-type badge with text (Zalo/FB/Pancake), customer name ("Khách hàng" when empty), relative time (`formatRelative`, never negative), analysis chip per §1, "{n} tin nhắn". Selected row highlighted. Range caption and pagination (9 per page).
4. **Layout:** desktop two panes (list 420 px, conversation fills the rest); mobile shows the list, and a selected conversation replaces it with a back button. The list pane is always visible on desktop.
5. **Conversation pane:** customer name, channel name · "{n} tin nhắn"; actions "Sao chép liên kết" (clipboard; success toast; failure toast) and "Tải .txt" (unchanged content). Tabs "Tin nhắn", "Đánh giá chất lượng · {runs}", "Phân loại · {runs}".
   - Messages: bubbles in theme tokens (customer left surface, agent right primary tint), sender and time `T2 28/09 09:00`, attachments/images/lightbox as now.
   - QC tab: the source note, then one card per run: verdict chip (glossary), score, job name, "Đánh giá lúc …", review with "Nhận xét do AI tạo", issues with severity and quoted evidence (always expanded; there are few).
   - Classification tab: the source note, then one card per run: tag chips (theme primary tint, not random colors), summary with the AI label; `SKIP` shows "Bỏ qua · không khớp nhãn nào". Never "Vấn đề", never pass/fail.
   - Loading and empty messages per tab as now.
6. **States:** list loading skeleton; no conversations at all (no filter) → "Chưa có hội thoại" + "Đi tới Kênh chat"; filtered empty → "Không có hội thoại nào khớp bộ lọc." + "Xóa lọc"; list load error → `role="alert"` + "Thử lại"; conversation load error → alert + "Thử lại" inside the pane.
7. **Export dialog:** `AppDialog` with close button; date inputs, format (Văn bản cho AI đọc / CSV bảng tính), loại kênh; the selection-rule text; "Tải về" disabled without both dates; server `{error}` shown as an error toast; network failure → generic error toast.
8. All copy through i18n (`msgs_*` keys, vi/en); dark mode via tokens.

## 3. Current vs intended

| Area | Current | Intended |
|---|---|---|
| Heading count | "Tin nhắn (210)" — 210 conversations (UX-16) | "210 hội thoại" |
| List verdict chip | PASS → "Đạt", everything else → "Không đạt" (SKIP and classification shown wrongly) | §1 mapping |
| Evaluation filter | "Đạt"/"Không đạt" unlabelled scope | "Lần gần nhất: …", PASS named honestly |
| Export with no data | Downloads the JSON error as a file | Error toast |
| Errors | Unhandled | Inline alerts with retry |
| Colors | Hard-coded Vuetify colors, random tag colors | Tokens |
| Strings | Hard-coded Vietnamese | i18n |

## 4. Invariants

Same list/filter/export/deep-link requests and parameters; no source status invented; no "Đạt" claim for a PASS that may be classification; no confidence shown (unchanged); counts named as hội thoại.

## 5. Held — `BLOCKED_API_CONTRACT`

1. QC-only "Đạt" chip/filter: the evaluated map and `evaluation=PASS` do not carry the job type.
2. Source-integrity status for conversation evaluations.
3. Total message count for the tenant/filter.
4. Job link per evaluation group (no job id in the group).

## 6. Acceptance

`vue-tsc -b --force`, build, full vitest; focused tests with a mocked API (UI only): heading counts hội thoại; list chip mapping for PASS/FAIL/SKIP/absent; unchanged list request parameters for each filter; export no-data JSON → toast and no download; export success → download; list load error + retry; classification tab never shows "Vấn đề"; source note present on both evaluation tabs. Disposable captures desktop/mobile × light/dark with 0 JS errors/overflow/external requests, plus a conversation open state. Diff/catalog/doctor.
