# R026 / F04 — Vietnam business-day read/filter contract

Status: SPEC_ACCEPTED_FOR_BOUNDED_BUILD. Date: 2026-10-02. Author: Codex (ORCHESTRATOR -> SPEC_AUTHOR -> WORK_ORDER_AUTHOR). Risk ceiling R2. Implementation remains OPEN until independent REVIEW. Design decisions below are the acceptance contract, not claims about current source.

## 1. Intake, source truth and boundary

At local baseline `9ecc83982da1e29737a7f5b6613eb5252539dd55`, R025/F03 is REVIEW_PASS / FREEZE_OPEN. F04 was accepted as a source finding at report `7481196` and the [independent source review](../reviews/CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md); the relevant code remains:

- `Dashboard.vue` constructs browser-local dates then calls `toISOString`; `dashboard.go` uses 24-hour truncation, UTC date parsing, inclusive BETWEEN, and SQL DATE on stored wall times.
- Dashboard today/month costs have lower bounds only; daily series have no upper calendar bound. The VN day helper already in `frontend/src/utils/format.ts` and `backend/pkg/helpers.go` does not repair these query paths.
- `results.go` uses UTC dates and inclusive last-second upper bounds for list and CSV/XLSX export. Cost Logs uses raw day strings plus 23:59:59. Message export selects conversations by UTC dates; its default UI dates use UTC ISO.
- Production config still has `parseTime=True&loc=Local`; Compose sets app TZ to Asia/Ho_Chi_Minh. Disposable test DSNs use loc=UTC. Existing timestamp storage must not be silently reinterpreted.

R026 covers these **read/report surfaces** and their date controls. Analyzer conditional/date selection, dispatch and checkpoint modes remain F05; cancellation registry remains F06. No claim of a platform-wide date migration or all writer semantics. No schema/DSN/Compose/timestamp migration, adapter, AI behavior, tenant-timezone-settings activation, or general UI timestamp-format redesign.

## 2. Design decisions and date API

Business timezone is fixed **Asia/Ho_Chi_Minh (UTC+07:00)** for the current product contract, independently of browser timezone, UI language, server TZ, and the currently unactivated tenant timezone setting. Show a concise Vietnam-date label beside the covered date controls (existing i18n conventions). Do not imply the Settings selector controls these reports.

Keep public query keys `from` and `to` and date-only `YYYY-MM-DD` strings. `to` is an **inclusive calendar date** in the public API, converted internally to the next business midnight. All covered predicates use `timestamp >= fromStart AND timestamp < toExclusive`; no BETWEEN/23:59:59/end-minus-one-second. A same-day range covers the full day, including stored milliseconds. Parse calendar dates strictly, reject malformed/impossible nonempty dates, reversed ranges or boundaries outside the database's representable range with HTTP 400 and bounded `invalid_date_range`, before query/file output. Empty dates are omitted, not invalid. No silent fallback on invalid input.

| Surface | Defaults / selection identity |
| --- | --- |
| Dashboard | Both dates absent: current VN calendar day `[todayStart, tomorrowStart)`. When a date is explicitly supplied, omitted opposite bound is open. All filtered cards, channel counts, recent QC/classification, and cost_period share that interval. Preserve issues vs qc_violation_count meanings. |
| Results list + export | No dates: unrestricted as today. Optional one-sided bounds allowed. `date_field=conv` filters conversation last_message_at; `eval` filters result/evaluation created_at. Identical shared predicates for totals/counts, fetched list and full-filter CSV/XLSX export. Preserve pagination, result types, confidence and source-integrity semantics. |
| Cost Logs | No dates: unrestricted; optional one-sided bounds. List and total share the parsed interval. Preserve provider filter, pagination and page-only cost display; no new aggregate endpoint. |
| Message export | Both dates remain required. Select conversations by last_message_at, with the same VN interval; preserve channel/tenant filters and export **all messages** of the selected conversations. This order does not change message-coverage or selection identity. TXT/CSV formats stay; no new message-window filter. |

R026 may change an internal filter parser signature to propagate validation errors. All callers/tests must remain compiled and covered; unrelated filter behavior is preserved.

### Timestamp storage compatibility (mandatory first BUILD check)

Read the actual MySQL dialector/driver location, timestamp column types/precision and bound-parameter behavior before writing queries. Available local primary source: `gorm.io/driver/mysql@v1.6.0/mysql.go` DSNConfig/Explain, and `go-sql-driver/mysql@v1.8.1/connection.go` plus `packets.go` convert `time.Time` values with `cfg.Loc` before serialization. The current DSN is not evidence that stored DATETIME wall times are universally UTC.

Use typed instants for range parameters so the existing driver location remains authoritative; do not pre-format UTC strings against Local-storage timestamps or apply +07 twice. For day aggregation, bucket **instants in VN**, not raw SQL DATE(wall time). Choose a bounded aggregate strategy using the same calendar interval boundaries or a verified storage-aware conversion; do not require MySQL named-zone tables, load all historical/message rows into memory, change SQL session time_zone, or change the DSN. Record the chosen strategy and its read/query cost. Required storage matrix: actual loc=UTC and actual loc=Asia/Ho_Chi_Minh connections, matching fixtures written via the same driver. Retain runtime support for the current UTC/VN deployments; never infer historical mixed-zone rows from their numeric values. Unidentified/mixed historical encoding or a required migration is BUILD_BLOCKED, not license to shift stored rows.

## 3. Dashboard calendar aggregates

- `cost_today`: bounded current VN day; `cost_this_month`: bounded VN month `[firstDay, firstDayNextMonth)`, independent of the user's selected interval. No future-month/day leakage. Capture one clock instant per request; tests may use a private deterministic clock seam, restored on cleanup.
- `cost_by_day` and `messages_by_day`: VN date keys `YYYY-MM-DD`, with current field/order/count meanings. Preserve the existing start horizon (today minus 30 calendar days), bounded above at tomorrowStart. This retains up to 31 calendar dates; do not silently introduce a separate series-length change or make these independent series follow the selected filter.
- Message count, distinct customer conversations/chat_count and agent reply_count remain per business day. Do not change sender-role or cost/token semantics. Keep queries tenant scoped and bounded by the series interval.
- Modified date-dependent queries must not report success with unusable/failed date aggregation. Return bounded generic failure without SQL/connection details. Broad remediation of every unrelated swallowed DB error is outside the order; preserve the existing R017 QC-query failure contract.

## 4. Frontend calendar controls

Use one testable Vietnam-calendar helper (reuse existing vnDateKey where appropriate; add `utils/businessDay.ts` for calendar arithmetic). Compute from a captured instant in VN; never combine browser-local getFullYear/getDate with UTC ISO date extraction. Calendar shifts/first-of-period operations must stay in that calendar, including when the browser runs in UTC, Asia/Ho_Chi_Minh or America/New_York.

- Dashboard default and `28days`: exactly 28 calendar dates including today. `7days`: exactly seven including today. Today: same date both bounds. Week begins Monday; month/quarter/year begin on the VN first day, and end on today. Preserve available preset choices.
- Results keeps `all` as its default/empty dates; today/7days/28days/month follow the same contract. List and export submit exactly the selected date strings and retain date_field and other filters.
- Message export default: exactly seven VN calendar dates including today. Cost Logs keeps empty/manual defaults. Date-only inputs are carried as dates, not parsed as midnight instants and re-serialized through the browser timezone.
- Changing the off-by-one preset lengths is intentional and must be recorded in BUILD evidence; do not weaken old tests without identifying their obsolete contract. No global formatter behavior change across unrelated views.

## 5. Executable acceptance evidence

Test the actual handlers on disposable MySQL, not just helper output. Put IDs and monetary/token/count expectations in assertions so UTC/VN mistakes cannot pass on a zero-row fixture.

1. For `from=to=2026-10-02`, fixtures at `2026-10-01T16:59:59.999Z` excluded, `17:00:00Z` included, `2026-10-02T16:59:59.999Z` included, `17:00:00Z` excluded. Test today at VN 00:00/00:30/06:59/07:00 and a clock just before midnight; cover month/year rollover and leap-day validation.
2. Repeat real handler/aggregate selection on both storage locations, restoring DB pools/globals and cleaning fixtures. Verify raw stored times and decoded instants so the matrix is genuine, not two labels on one connection. Other tenants and channels must not contaminate results.
3. Dashboard: filtered counts/recent rows/cost_period, independent today/month costs, both series date buckets with customer/reply/distinct-conversation counts. Seed out-of-day/out-of-month data to detect missing upper bounds. Preserve QC count-vs-issues and its generic failure test.
4. Results: conv and eval date modes with deliberately different dates; compare expected IDs/counts to actual list, CSV and XLSX contents at boundaries. Cost Logs: returned IDs and total, one-sided/no-date behavior. Message export: TXT/CSV expected conversation IDs/content, retained all-message behavior and tenant/channel isolation.
5. Invalid/missing/reversed/equal/leap dates as applicable: exact status and no file bytes on error. Empty/no-date defaults preserve the table above.
6. Mount actual Dashboard/Results/Messages/CostLogs components to capture request/export query parameters and timezone labels. Test browser-zone-independent presets at early VN hours and year/month/quarter/week transitions. Use the **same named boundary cases/query strings** in handler fixtures. Record whether this is linked UI/API/DB contract proof or an actual browser-to-server run; do not describe isolated helper tests as full-flow evidence.
7. Demonstrate old-source failures and meaningful mutations: UTC day parsing; inclusive/last-second upper bound; raw SQL DATE bucket; browser-local/UTC-ISO preset; missing today/month upper bound. Mutations must compile and fail behavior assertions, then be restored. Never count a compile error or unavailable DB skip as detecting the defect.

Run focused tests, full backend with the R019 five-sentinel gate, full frontend tests, forced typecheck/build, docs build, catalog/doctor/preflight/diff checks. Report exact commands/results, changed expectations and optional skips. Synthetic data proves local calendar/query/report semantics only; no provider, upstream completeness or CVF AI-governance proof. R025 ordinary selection/terminal persistence and F05/F06 remain untouched.

## 6. Governed dispatch

Dispatcher seed/base `8f366ebd3ed2df3a1302a0635844d9a5f5d0d1a9` is committed before BUILD. [Work order](../work_orders/CCMAI_RUNTIME_026.md) grants the bounded paths; Claude is IMPLEMENTATION_WORKER/COMMIT_STEWARD, Codex independent REVIEWER. Return one local REVIEW_PENDING commit and [BUILD evidence](../reviews/) with no push/merge/deployment/provider/channel call/persistent DB/parent-CVF work/FREEZE. Alibaba authorization persists separately; this tranche needs zero calls and grants none.
