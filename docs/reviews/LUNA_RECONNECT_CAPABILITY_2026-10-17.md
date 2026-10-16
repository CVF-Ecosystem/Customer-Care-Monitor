# Luna integrated-browser reconnection capability

Status: CAPABILITY_UNAVAILABLE_IN_LUNA_SESSION  
Date: 2026-10-17  
Role: Luna Max IMPLEMENTATION_WORKER, capability-only intake

Luna's actual integrated `mcp__cua_repl.js` first call was `await cua.getState();`. The documented CUA API loaded, but the inventory was exactly `apps: []`, `browsers: []`. The call returned no error string; there was no browser surface to select. Luna did not create a tab, navigate, click, or capture a screenshot. Browser version and cache state are `NOT_OBSERVED`; product UI behavior is also `NOT_OBSERVED`. Raw inventory fields are preserved in [capability_inventory.json](probes/luna_reconnect_20261017/capability_inventory.json).

Root separately reports a fresh benign local HTTP health-page check passed on Root's browser: the page changed from “Chưa kiểm tra” to “Đã kết nối,” a screenshot was captured, the owned tab was closed, and the health server exited. Root's evidence is at [after_click.png](probes/root_luna_reconnect_20261017/after_click.png). This is Root-owned capability evidence; it does not establish browser access in Luna's session and it is not product-case evidence.

The current R109 interrupted campaign remains stopped. Its earlier actions, spent budgets, missing original finalization, and NOT_RUN cases remain unchanged. This capability check did not start the product, fixture, backend, Vite, or the campaign runner, and it makes no claim about B07, B08, B09, or product acceptance.

## Read-only language audit

The exact UI text is split across four source files that form the minimal proposed edit set:

- [vi.ts](../../frontend/src/i18n/vi.ts) and [en.ts](../../frontend/src/i18n/en.ts): translated keys for `ai_provider`, `nav_mcp`, `mcp_connections`, `mc_sub`, `api_key`, and `iv_local_only`.
- [JobDetail.vue](../../frontend/src/views/Jobs/JobDetail.vue): hard-coded “Chưa cấu hình AI Provider” and the instruction to configure an API key.
- [Dashboard.vue](../../frontend/src/views/Dashboard.vue): visible “API Server”, “Database”, and “Scheduler” service names.

The visible labels are consumed by `DefaultLayout.vue`, `Settings.vue`, `StepAnalysisSchedule.vue`, `MCPConnections.vue`, and `InterventionCases.vue`; those consumers need no edits if the translated values are corrected. No source file was changed in this audit.

Suggested plain-language wording for a newly authorized UI repair:

| Current wording | Vietnamese proposal | English proposal |
| --- | --- | --- |
| AI Provider / Nhà cung cấp AI | Dịch vụ AI | AI service |
| Chưa cấu hình AI Provider | Chưa thiết lập dịch vụ AI | AI service is not set up |
| API key instruction | Thêm khóa truy cập Claude hoặc Gemini trong phần Cài đặt AI trước khi chạy phân tích. | Add a Claude or Gemini access key in AI settings before running analysis. |
| Kết nối MCP / MCP Connections | Kết nối dữ liệu | Data connections |
| MCP page description | Tạo kết nối để trợ lý có thể tra cứu dữ liệu bạn cho phép. | Create a connection so your assistant can look up the data you allow. |
| API Server | Ứng dụng | Application |
| Database | Kho dữ liệu | Data storage |
| Scheduler | Lịch chạy tự động | Automated schedule |
| `iv_local_only` | Chỉ có dữ liệu trong hồ sơ nội bộ; chưa xác nhận được nội dung đầy đủ từ kênh. | Only the internal record is available; the full channel history has not been verified. |
| API Key | Khóa truy cập | Access key |

These are proposals only; Root owns any future source scope and validation.

## Proposed next bounded route

Root may prepare a fresh R110 authority for the remaining B07/B08/B09 checks and the two open wording findings, without replaying R109 or redoing B01-B06/B10. First settle the wording in source, focused tests, and independent review. Then bind a fresh dispatch to the reviewed frontend commit and exact locale/component hashes, the precise runner/config, and either a backend binary whose source binding is independently verified or a newly admitted build if backend source changes. Create a new fixture only under that successor's budget; R109's consumed counters and evidence stay immutable. Bind the exact image/container/network/volume and exclusive ownership, and compare protected inventory before and after, including R102 and user-owned resources.

The successor should capture B07 with the reader role after owner logout; capture B08 only when the actual viewport is observed at 390×844 in both the browser and evidence; and run B09 only with a documented, safe offline control, otherwise retain `NOT_RUN`. Record browser/tool provenance, DOM and original images, actual UI POST attempts, pre-action reservations, and exact cleanup results. Root's fresh local-health browser capability is a dispatch prerequisite only; it does not satisfy these product checks. Root must issue the successor authority and admission before any product runtime action.
