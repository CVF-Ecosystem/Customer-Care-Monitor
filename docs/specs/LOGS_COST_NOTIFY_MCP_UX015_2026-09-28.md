# SPEC — Activity logs, cost logs, notification history and MCP connections (`CCMAI-UX-015`)

**Date:** 2026-09-28 · **Author:** Claude (`ORCHESTRATOR` by owner direction → `SPEC_AUTHOR`; design approved under the owner's design delegation) · **Risk:** R2 · **Work order:** [CCMAI_UX_015](../work_orders/CCMAI_UX_015.md) · **Roadmap:** screen 8. **Status:** BUILD done, `REVIEW_PENDING` ([evidence](../reviews/LOGS_COST_NOTIFY_MCP_UX015_BUILD_2026-09-28.md)).

**Approved design:** canvas https://claude.ai/artifact/8E2fGLG5qEwcmZK6bGVJyS ("Cài đặt & phụ trợ", private to the owner), **version id `1790608303-0cb6`**. UX-015 artboards: "Nhật ký hệ thống — 1280", "Nhật ký chi phí — 1280 — tổng của trang", "Lịch sử thông báo — 390 dark — mở nội dung, lỗi tải", "Kết nối MCP — 1280 — secret vừa tạo". The UX-014 boards in the same canvas are unchanged from `1790607141-5a73`.

## 0. INTAKE (owner-directed)

Presentation of `views/ActivityLogs.vue`, `CostLogs.vue`, `NotificationLogs.vue`, `MCPConnections.vue`. Nothing here changes endpoints, payloads, permissions, logging or the MCP OAuth flow. The tranche does not depend on UX-012..014: the files are disjoint, shared UX-000 components are imported only, and the i18n additions are separate `lg_*`/`mc_*` blocks.

## 1. Fixed contract (source truth at `e24da27`)

| Fact | Consequence |
|---|---|
| `GET /tenants/:id/activity-logs?page&per_page&action` filters with `action LIKE '<value>%'` and returns `{data,total,page,per_page}`. Actions actually written: `user.login`, `channel.delete`, `channel.purge_conversations`, `job.delete`, `job.clear_results`, `job.clear_runs`, `job.run.started`, `job.run.completed`, `notification.error`, `sync.completed`, `sync.error`, `sync.partial`, `settings.storage`. | Vietnamese labels for these actions; an unknown action shows its raw value. The filter adds `settings` (the old list missed `settings.storage`). Detail and error strings are written by the server in English and shown verbatim (held §5). |
| `GET /tenants/:id/cost-logs?page&per_page&provider&from&to` returns `{data,total,page,per_page,exchange_rate}`; no summed cost. Providers written: `claude`, `gemini`, `openai`, `xai`. | The filter lists all four (the old one listed only claude/gemini). The footer sums **only the current page** and says so; a filtered grand total is `BLOCKED_API_CONTRACT` (§5). |
| `GET /tenants/:id/notification-logs?page&per_page` returns `{data,total,…}`; `status` is `sent` or `failed`; `channel_type` is `telegram` or `email`. | The old view swallowed errors and showed "Chưa có thông báo nào": a failure now shows an alert with "Thử lại". |
| `GET/POST/DELETE /mcp/clients` are scoped to the **signed-in user** (not the company). Create returns `client_secret` once; the server keeps only its hash. Scopes allowed: `read`, `write`. | The subtitle says the connections are the user's own. The secret is shown once, with a warning, and the close button reads "Tôi đã lưu secret, đóng". Load failure is no longer shown as the empty state; revoke uses `ConfirmDialog` and reports failure. |
| Changing a filter while on page N kept page N (the old views). | A filter change returns to page 1 (a presentation fix; request parameters unchanged). |

## 2. Intended behavior

1. **Common:** page title plus a one-line subtitle; a desktop table and mobile cards (≤ `md`); skeleton on first load; load error → `role="alert"` block with "Thử lại"; separate empty states for "nothing yet" and "nothing matches the filter"; dates `dd/mm/yyyy hh:mm`; pagination unchanged (20 per page); all copy through i18n; token colors so dark mode works.
2. **Activity logs:** subtitle "{n} sự kiện · mới nhất ở trên"; labelled "Loại sự kiện" select with "Tất cả"; columns Thời gian / Sự kiện (tonal chip: error actions red, delete/clear orange, others neutral) / Người thực hiện ("Hệ thống" for `system` or empty) / Chi tiết (+ error in red).
3. **Cost logs:** subtitle "{n} lần gọi AI theo bộ lọc · tỉ giá {rate} ₫ = 1 US$"; filters Nhà cung cấp (Claude, Gemini, ChatGPT, Grok) / Từ ngày / Đến ngày; columns Thời gian, Nhà cung cấp, Model, Token vào, Token ra, Chi phí (US$, 4 decimals), Chi phí (₫) with tabular numbers; footer "Tổng của trang này ({n} lần gọi)" and, when `total` exceeds the rows shown, the note that it is not the full total.
4. **Notification history:** subtitle "{n} thông báo đã ghi nhận"; each item shows status chip (Đã gửi / Gửi lỗi), channel (Telegram / Email), time, recipient, error line when failed, and a "Xem nội dung"/"Ẩn nội dung" toggle with `aria-expanded` revealing subject and body in a token-bordered block.
5. **MCP connections:** subtitle "Kết nối của riêng bạn để Claude Web/Desktop truy vấn dữ liệu qua MCP"; list shows name, client ID (monospace), redirect URIs, scopes (Đọc / Ghi), created time and "Thu hồi"; create dialog on `AppDialog` with name, redirect URIs, scopes; after creation the same dialog shows client ID, secret, warning, "Sao chép secret" and "Tôi đã lưu secret, đóng". Copy failure reports an error; errors use an error-coloured snackbar.

## 3. Current vs intended

Cost footer "Tổng" over one page → "Tổng của trang này" with note. Provider filter missing openai/xai → all four. Activity action filter missing `settings.storage` → added; raw action codes → Vietnamese labels. Notification and MCP load failures shown as empty → alert + retry. MCP revoke via native `confirm`, failure silent → `ConfirmDialog`, error snack. Error snackbars coloured success → error colour. Hard-coded `bg-grey`/`white` blocks unreadable in dark mode → tokens. US-style `toLocaleString()` dates → `dd/mm/yyyy hh:mm`. Filter change kept page N → page 1.

## 4. Invariants

Endpoints, query parameters, payloads and page size unchanged; action filter values are prefixes the server already supports; MCP create payload unchanged (`name`, `redirect_uris`, `scopes`); no secret is written anywhere except the one-time display; no permission change.

## 5. Held

1. Filtered grand total for cost logs — `BLOCKED_API_CONTRACT` (`/cost-logs` returns no sum).
2. Vietnamese detail text for activity logs — server writes English detail strings; translating needs a structured log contract.
3. Notification-log permission (route has no `RequirePermission`; the menu uses `jobs`) — permission contract, unchanged.
4. MCP connections per company rather than per user — data contract, unchanged.

## 6. Acceptance

vue-tsc, build and full vitest. Focused tests with a mocked API (UI only, no governance claim):
- Activity: action labels are Vietnamese, unknown action raw, `system` shown as "Hệ thống"; changing the filter requests page 1 with the prefix;
- Cost: footer reads "Tổng của trang này" and the not-full-total note appears when `total` > rows; provider filter offers openai and xai;
- Notification: a failed request shows the error alert, not the empty text; retry loads; the toggle reveals the body;
- MCP: load failure shows the alert; revoke asks through `ConfirmDialog` before `DELETE`; the created secret is shown with the warning.

Disposable captures (synthetic data, persistent `ccma` untouched) of the four screens at desktop/mobile × light/dark with 0 JS errors/overflow/external requests; the MCP secret state is captured with a synthetic client in the disposable environment only. Diff/catalog/doctor.
