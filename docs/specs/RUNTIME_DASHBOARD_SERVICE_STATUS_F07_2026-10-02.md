# F07 — Truthful Dashboard service-status presentation

Status: SPEC_ACCEPTED_FOR_WORK_ORDER. Date: 2026-10-02. Tranche: CCMAI-RUNTIME-029.
Author: Codex, SPEC_AUTHOR. Risk ceiling: R2. Planning only; source remediation and F07 review remain OPEN.

## Intake and inspected source

Owner requests separate F07 planning after R028-R2 independent REVIEW_PASS / FREEZE_OPEN. Inherit the accepted [F01–F08 source finding](../reviews/CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md) and [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md); no prior tranche is frozen by this order.

At planning baseline `fade6ae9d3286edca130afb16f60c311cd3405cd`, `frontend/src/views/Dashboard.vue` declares API Server, Database and Scheduler with `ok: true`. The service card renders success/error chips and a success check-circle heading. `loadDashboard` updates tenant metrics, not service health. `backend/api/router.go` registers `/health` returning constant status/version without a database or scheduler probe. These are inspected source facts, not observed production outages. Dashboard request success, active_jobs and active_channels cannot establish component health.

## Design decision and boundary

Choose the roadmap's explicit unavailable-data option. Retain the three service rows, show neutral unknown status with an explanation that health checks are unavailable. Do not introduce health measurement in this tranche. A future telemetry contract must separately define probes, auth, timeout, freshness, degraded/down/unknown semantics and scheduler liveness before any measured-health presentation.

Operational-status presentation retains an R2 ceiling conservatively and independent reviewer responsibility. BUILD changes only frontend UI/copy and UI tests; no AI governance behavior, provider routing, DB or external-service test. Mock responses are allowed only for these UI rendering checks. No synthetic test may be called runtime CVF governance proof; such a claim requires a separately authorized real-provider receipt.

## Requirements

| ID | Required behavior |
| --- | --- |
| F07-01 | API Server, Database and Scheduler initially and continuously display an explicit unavailable/unknown state. Remove hardcoded `ok: true` and the success/error inference. No initial green frame before requests settle. |
| F07-02 | Vietnamese chip: `Chưa có dữ liệu kiểm tra`; English: `No health-check data`. Explain once in the card: `Chưa có phép kiểm tra sức khỏe cho các dịch vụ này.` / `Health checks are not available for these services.` Use scoped i18n keys; preserve global `normal`, `error` and `no_data` semantics elsewhere. |
| F07-03 | Chips and heading use neutral theme styling and a neutral information/help icon. No success/error color, check-circle or outage icon for these unknown rows. Visible text conveys state without relying on color or a hover-only tooltip. Retain Service Status title, names and readable responsive layout. |
| F07-04 | Dashboard request pending, successful populated or empty response, rejected/network/timeout response and repeated success/failure refresh never promote any service to healthy or declare it down. Unknown remains unknown. HTTP/DB-shaped error fixtures describe only received UI responses, not an actually tested service outage. |
| F07-05 | Ignore incidental fields such as `health`, `services`, `scheduler_status` in Dashboard payloads; there is no accepted health schema. Scheduler active/idle/zero-job counts are metrics and cannot change the unknown status. Do not call `/health`, add health polling, timers, probes or synthetic check timestamps. |
| F07-06 | Preserve Dashboard tenant/date request parameters, Vietnam business-day filters, latest-request guard, QC unavailable/count semantics, metric/chart/cost/activity updates, demo flows and navigation. Do not fix unrelated Dashboard data-retention/error behavior. |
| F07-07 | BUILD evidence separates implementation from acceptance, identifies exact source/commit and limitations, includes reproducible UI checks and retained regressions. F07 stays OPEN until independent REVIEW; no FREEZE or actual operational-health claim. |

## Acceptance and evidence

Mounted tests must exercise the real Dashboard with Vuetify and both i18n dictionaries; isolate the Service Status card rather than banning unrelated green metrics or global words across the page. Include these rows in the worker requirements-to-tests matrix:

| Group | Observable acceptance |
| --- | --- |
| A — initial and pending | Mount with deferred Dashboard request: all three names and explicit unknown chips visible before resolution; heading and chips neutral. No fabricated healthy state at first render. |
| B — ordinary response | Resolve populated and empty metrics, including positive/zero active-job/channel counts: all three services remain unknown; existing metrics update as before. |
| C — failure and repeat | Reject generic network/timeout and HTTP 500/503 responses (including synthetic DB-failure-shaped response); test success then failure and failure then success through existing refresh. All service rows remain unknown without success/down styling. No claim this measured a real API or DB outage. |
| D — untrusted hints and Scheduler | Supply incidental service-health-like payload fields and scheduler active/idle/failure hints. None changes unknown or creates a health call/timer. Assert outbound UI request inventory retains existing Dashboard/demo requests only; no `/health` request. |
| E — locale and appearance | Both Vietnamese and English chip/explanation text resolve without raw keys; service names/order persist. Inspect computed Vuetify classes and mounted heading/icon plus a rendered capture for each locale. Card remains readable at desktop and narrow/mobile width. |
| F — regressions and detector | Retain and run dashboard-qc-card, dashboard-business-day and i18n tests plus full frontend suite. At least one mounted status test must fail against the old Dashboard with hardcoded success. Restore source after the probe, rerun focused tests, and document exact assertions. A scoped mutation restoring one row's success styling/text must be caught; retain the detector. |

Frontend full tests, forced `vue-tsc -b --force`, build, docs build, catalog `-Check`, workspace doctor, diff check and required downstream gate tests/preflight complete the BUILD verification. No backend suite or DB fixture is required for a frontend-only change. Record exact counts, skipped/not-run checks, fixture restoration and screenshot paths in `docs/reviews/RUNTIME_DASHBOARD_SERVICE_STATUS_F07_BUILD_2026-10-02.md`; do not add fixture payloads or screenshots outside authorized artifact classes.

## Disposition

Claude implements under the immutable R029 seed; Codex reviews independently. Return one local REVIEW_PENDING BUILD commit with exact SHA. Reviewer checks requirements, old-source detector, appearance, request boundary, source scope and seed author/timing. Missing acceptance or weakened regressions returns CHANGES_REQUIRED. R028/F06 and R027/F05 remain REVIEW_PASS / FREEZE_OPEN; F02 limits, backend observability, governance proof and deployments stay separate.
