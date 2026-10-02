# CCMAI-RUNTIME-029 / F07 BUILD evidence — Dashboard service status is an explicit unknown

**Date:** 2026-10-02 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 · **Role:** IMPLEMENTATION_WORKER + COMMIT_STEWARD (Claude); independent REVIEWER: Codex
**Authority:** [SPEC](../specs/RUNTIME_DASHBOARD_SERVICE_STATUS_F07_2026-10-02.md), [work order](../work_orders/CCMAI_RUNTIME_029.md), dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-029.json` (base `0f442f2`, present at the base commit, allowed paths and effects identical to the tranche record, not edited). The role acknowledgment `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` was written into the handoff and the BUILD phase synchronized (preflight 7/7) **before the first source or test edit**.
Frontend only, synthetic mocked API for UI rendering only. Real provider/channel/notification calls: **zero**; no backend, endpoint, scheduler, dependency, workflow or tooling change; no persistent DB; no push/merge/FREEZE. **This change does not measure, and the evidence does not claim, actual API, database or scheduler availability, hosted CI success or runtime CVF governance** (no `governanceReceipt`).

## Source trace (before editing)

`frontend/src/views/Dashboard.vue` declared `services = ref([{name:'API Server', ok:true}, {name:'Database', ok:true}, {name:'Scheduler', ok:true}])`, rendered a `color="success"` `mdi-check-circle` heading and `ok ? normal : error` chips. `loadDashboard` (one `GET /tenants/<id>/dashboard`) updates tenant metrics only; the other requests are the existing `GET demo/status` and the demo import/clear actions. Nothing in the view calls `/health`, and `/health` is a constant response, so no UI state could have been derived from a measurement.

## Change

- `Dashboard.vue`: `services` state is removed; `const serviceNames = ['API Server', 'Database', 'Scheduler']` holds names only. The card keeps its title, shows a neutral `mdi-help-circle-outline` heading icon (no color), one visible explanation line, and for each row a neutral tonal chip with a help icon and the unknown text. No success/error class, check-circle or outage icon remains in the card; state is conveyed by visible text, not color or a hover tooltip. The unrelated `mdi-check-circle` elsewhere on the page (the recent-activity empty state) is untouched.
- `i18n/vi.ts`, `i18n/en.ts`: `service_health_unknown` (`Chưa có dữ liệu kiểm tra` / `No health-check data`) and `service_health_note` (`Chưa có phép kiểm tra sức khỏe cho các dịch vụ này.` / `Health checks are not available for these services.`). Global `normal`/`error`/`no_data` are unchanged; i18n parity tests pass.
- No new request, polling, timer or timestamp; Dashboard tenant/date parameters, Vietnam business-day filters, latest-request guard, QC semantics, metrics/charts/cost/activity and demo flows are untouched.

## Requirements and acceptance matrix

All tests are in `frontend/src/__tests__/dashboard-service-status.spec.ts` (13 tests; real Dashboard, Vuetify, both dictionaries; the card is located by its existing title so the same lookup runs on old source). The shared assertion `expectUnknownAndNeutral` checks names and order, exactly three unknown chips and one explanation, absence of the words Bình thường/Lỗi/Normal/Error, no `text-/bg-success|error|warning` classes, no check-circle/alert/close-circle icon, and the help icon.

| ID / group | Evidence |
|---|---|
| F07-01, A initial/pending | `A: shows the unknown state at first render and while the Dashboard request is pending` — deferred request: unknown before anything settles (no green frame), while pending, and after it resolves. |
| F07-04, B ordinary | three tests: populated with positive counts (also asserts the metric `4242` still updates), populated with zero job/channel counts, empty response — rows unchanged. |
| F07-04, C failure/repeat | network error, timeout, HTTP 500, HTTP 503 database-failure-shaped body, and `success → 503 → success → network error → success` through the existing date-refresh path, checked after every step — never healthy, never down. (These fixtures describe UI responses only, not a tested outage.) |
| F07-05, D hints/boundary | four incidental payloads (`health`, `services`, `scheduler_status`, `status`, nested `database`/`scheduler`, active/idle/zero-job scheduler hints): unknown stays; then fake timers advance **10 minutes**: every `GET` URL matches `/tenants/t1/(dashboard|demo/status)$`, none matches `/health/i`, and there is no POST. |
| F07-02/03, E locale/appearance | vi and en: chip + explanation resolve without raw keys, names/order persist; computed Vuetify classes of the three chips and the heading are neutral and carry the help icon. Real captures below. |
| F07-06, F regression | the retained `dashboard-qc-card`, `dashboard-business-day` and `i18n` specs pass unchanged; full suite below. |
| F07-07 | this record separates implementation from acceptance; F07 stays OPEN until independent review. |

## Old-source detector and mutations

- **Old Dashboard (before the change)**: the new spec was run first, against the unmodified `Dashboard.vue`: `npx vitest run src/__tests__/dashboard-service-status.spec.ts` → **13 failed / 13** (log kept in the session scratchpad as `f07_old_source_failures.log`). Representative assertions: `expected +0 to be 3` (no unknown chips), and `expected 'v-chip v-theme--light text-success v-…' not to match /text-(success|error|warning)|bg-(success|error|warning)/`. After the implementation the same file passes 13/13.
- **Mutations** (applied to `Dashboard.vue`, spec run, source restored byte-for-byte, spec rerun 13/13): one row (Database) shown healthy again with the success color → **killed**; success `mdi-check-circle` heading restored → **killed**; explanation removed → **killed**; rows made to depend on the request outcome (healthy after load) → **killed**.

## Captures (real rendered pages)

Captured with the project's own tooling (`scripts/ui-screenshots.ps1 -Mode app -KeepEnvironment`, then `scripts/ui-screenshots.mjs --locale en` against the same disposable app): a throwaway Compose project (MySQL in tmpfs, throwaway secrets outside the repo, built-in synthetic demo data, no provider call) serving the freshly built frontend. Light theme, desktop 1440×900 and mobile 390×844 (×2 scale); the reports show `pagesWithJsErrors: 0`, `pagesWithHorizontalOverflow: 0`, `externalRequests: []`. The images are crops of the Service Status card from those real captures:

- [vi desktop](f07-captures/dashboard-service-status--vi--desktop.png), [vi mobile](f07-captures/dashboard-service-status--vi--mobile.png)
- [en desktop](f07-captures/dashboard-service-status--en--desktop.png), [en mobile](f07-captures/dashboard-service-status--en--mobile.png)

They show the neutral help icon heading, the explanation, and three `Chưa có dữ liệu kiểm tra` / `No health-check data` chips, readable at both widths. The disposable project, its image, network and volumes, and the temporary env/token files were removed (`docker ps -a`, `docker network ls`, `docker volume ls` and the image list show no `ccma-uishot-20261002195110` resources).

## Request inventory

Source: `Dashboard.vue` issues `GET /tenants/<id>/dashboard`, `GET /tenants/<id>/demo/status`, and, only on user clicks, the existing demo `POST demo/import` and `DELETE demo/reset`; no other call. Tests (group D) assert only the two GETs and no POST over a simulated 10 minutes. The live capture reports `externalRequests: []`. The backend `GET /health` exists and answered 200 in the disposable app, but the Dashboard never requests it.

## Gates

- Frontend: vitest **26 files / 268 tests pass** (255 before + 13 new), 0 skipped; `vue-tsc -b --force` clean; production build OK.
- Docs build PASS, workspace doctor 25/25 PASS, gate unit tests 46/46 PASS, explicit-`--files` preflight over every changed path (including the four captures) **7/7 PASS** including catalog and tranche (not a whole-worktree pass: the untracked `knowledge/_index.json` and Python bytecode directories stay excluded and make the default preflight fail the tranche gate).
- Not run (not applicable to a frontend-only change): backend suite, R019 DB gate, race detector, any provider/channel/DB fixture.

## Limits

The card states that **no health check exists**; it does not show that any service is healthy or down. A future telemetry contract (probes, auth, timeout, freshness, degraded/down/unknown semantics, scheduler liveness) is a separate work order. Synthetic API responses prove UI rendering only. The English locale capture shows pre-existing Vietnamese demo data in the activity list (unrelated, untouched).

## Reviewer evidence-scope clarification (Codex, 2026-10-02)

The worker Gates phrase “any provider/channel/DB fixture” NOT RUN refers to backend integration-test fixtures, not the capture environment. The Captures section explicitly records disposable MySQL in tmpfs and generated throwaway local secrets; that environment did run for screenshots. No persistent DB or real provider/channel call is claimed. Reviewer did not recreate the capture environment or inspect credentials: the committed crops were visually checked, while full-page reports and cleanup inventory remain worker-reported. Source/test acceptance is evaluated separately in the independent review record.
