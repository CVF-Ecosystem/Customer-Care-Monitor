# R132 — WAIT_DATA source follow-up and request-only UI

Status: ACCEPTED_INTENDED_CONTRACT
Risk: R2
Date: 2026-10-19
SPEC author: Root ORCHESTRATOR / SPEC_AUTHOR
Implementation workers: Luna Max frontend followup; Sol Medium owned synthetic fixture
Independent reviewer: Root independent source/browser reviewer

## Product outcome and design

R131 identifies the concrete M3-05 gap: backend capture/read revision and create/read rerun-request APIs exist, while the case screen offers only data ownership/escalation. Add a small WAIT_DATA follow-up panel in the existing case view. A permitted person records newly synchronized local conversation content with a reason, sees what was recorded, and saves a reassessment request for the latest complete local revision. A reader sees stored revision/request history without write controls. Explain in ordinary VN/EN that saving a request does not start AI. No new execute/grant control, backend production logic or schema, source editing/import, model/key UI or provider call.

INTAKE: owner roadmap continuation, R131 matrix/review and current R128 source. DESIGN: use four existing revision/request read/create endpoints, separate from execution; preserve history and exact source/version binding. SPEC: F01..F08 below. WORK_ORDER: separate immutable R132 seed committed before BUILD. The intended contract is accepted by Root; product implementation is unaccepted until independent REVIEW. Source base is fc4cc961b58c011f305d22045e2ccd2fef2c7865.

## F01 — Honest display and bounded reads

Render revision coverage/local observation, reason/date and saved request status/history for WAITING_ON_DATA cases. Map all wire enums/errors to ordinary VN/EN; no raw IDs, hashes, request keys, grants, schema/internal enum names or driver text in normal UI. Render text safely through Vue, no v-html. Keep existing proposal, participant, assessment and human-history presentation. A recorded local revision never proves complete upstream channel history. Preserve truthful disabled/synthetic capability messaging.

Backend lists are ascending and paginated. Use bounded page sizes and show any remaining history. Never assume the last item on a partial first page is the latest revision. Disable requesting until the latest relevant complete revision can be established from a complete bounded history or the successful capture response; fail closed on truncation/unknown/mismatched snapshot lineage. Support a manual More control as needed, with no unbounded automatic loop. Request history also remains honestly paginated.

## F02 — Record newly available local source

Use POST data-revisions with expected_version, expected_snapshot_id, expected_snapshot_digest and a nonempty reason <=500 characters. Bind to the displayed case's original snapshot and current version, not a proposed local success projection. Read the response and refresh current detail/history after success. An unchanged source 409 is an ordinary visible failure, not a saved revision. Incomplete observed content remains incomplete and cannot enable request creation. This UI does not edit or synchronize the source itself.

## F03 — Request only

POST rerun-requests binds current case version, original snapshot ID/digest, latest complete data_revision_id/digest, bounded reason and an intent-scoped request_key. Use a stable unique intent key; no automatic retry/replay. Render the persisted response/history and reload on success. A request stays pending execution authority and is not a completed reassessment, approved AI execution or resolved case. Do not call any execute, grant, job-run or provider endpoint, including indirectly on mount/watch/success. Backend remains the final permission/version/source authority.

## F04 — Permissions and lifetime

Capture needs intervention_cases read+write; request creation additionally needs jobs write. Honor known-role/current auth helpers, with owner/admin semantics and member-specific rights; reader/no-jobs/unknown/revoked states must not gain controls or write calls. Read-only history uses existing case read rights. Bind every read/write to tenant, case, token, role and permissions. Abort/ignore stale responses on route/session/permission changes and unmount; clear old forms/selections/history. Share a write-busy interlock with existing case actions so the two forms cannot concurrently submit different writes. No automatic conflict replay.

## F05 — Safe errors and uncertain writes

Show friendly messages for invalid input, session/permission changes, missing case, source unchanged, pending request, conflict, unavailable service and unknown completion. No arbitrary server error text. Conflict and uncertain write require an explicit reload/review before another intent; do not show saved/success based only on a button click or locally increment versions. Preserve safe user text as appropriate, but clear stale revision selection. Disable double-click and finish busy state only for the active lifetime.

## F06 — Frontend checks

Luna may run two cached local Node checks: focused new/existing intervention UI structure/RBAC/lifetime tests and typecheck/Vite build. Mocked detail/status inputs are only UI structure/presentation fixtures; they must not be presented as server permission, source validation, approval/execution or AI-governance proof. Test visible saved/unknown/error states, reader/no-jobs controls, pagination/truncation, stale response clearing, busy/no duplicate dispatch, no execute route and no raw enum/ID leakage. Root independently audits exact transport bindings and source and may use the remaining two Node invocations for review/repair. No dependency install/download.

## F07 — One opt-in native/browser fixture

Sol adds only backend/api/handlers/intervention_followup_browser_test.go, reusing existing test helpers. Default opt-in absent => SKIP. Before db.Connect/AutoMigrate or app start, validate explicit opt-in, fresh absolute non-symlink evidence directory and matching DSN host/port/database against a Root-created identity nonce in that exact synthetic DB. Restrict DB name/prefix and loopback host; do not touch installed containers/DB. Root provisions at most one cached-image MySQL fixture and preserves it after one final stop.

Seed one WAIT_DATA case and new synthetic local customer text before the effect baseline. No real/mocked provider is installed; executor/runner remain nil. Native read/write permission probes use the real router and prove reader capture403 and no-jobs request403 with unchanged fingerprints. They do not approve or run AI. Serve the actual frontend build and real API on a loopback ephemeral port. Journal method/path/status only, never tokens or real credentials. Ready/terminal/stop use fresh external evidence files and nonce ownership. A valid stop closes the owned server, joins its Serve result, writes a final terminal even on failure, and lets the held test process report its actual exit. Bound timeout; no generic kill.

Expected browser campaign: owner login; capture201 (optional unchanged capture409); request201; reload detail/history; logout; reader login/reload sees saved history with zero write controls; logout. At most4 auth,2 capture,1 request browser POSTs; zero execute/provider posts. After terminal verify one revision/request, correct version/history, WAITING_ON_DATA still waiting, original snapshot/proposal/source identity unchanged across actions, all attempts/budgets/grants/claims/runs/provider accounting unchanged. Native permission denials are explicitly separate from browser actions. Root observes actual server/test exit and stops only the identity-proven synthetic MySQL; preserve original app resources.

## F08 — Evidence and completion

All bulky raw logs, screenshots, binaries and snapshots go outside Git under ../.ccmai_local_evidence/r132_waitdata_request_20261019; Git receives only source, text work/review records and compact hash/exit/counter summaries. Record exact current source hashes and actual commands/exits; no full-suite, exact raster or hosted proof is inferred. Root browser/native verification uses actual zero-provider product software APIs only; no assertion of CVF governance from synthetic or mocked AI output. Real provider/governance acceptance requires a new applicable real-call contract and is outside R132.

Budgets are the immutable seed limits: MySQL1 start/1 final stop; Go build/test4 total; Node4 total; browser1 campaign; provider0; at most2 consolidated same-scope repair rounds. Source audit before product runtime. If cached tooling unavailable or runtime browser cannot operate, stop/report for owner transfer; no alternative automation/download or unbounded replay. Do not reuse old consumed budgets.

Exit: Root independently accepts this bounded request-only user capability from source and actual permitted evidence, with precise qualifications. M3 remains partial for independent queue/reject/browser-failure or continuous provider-flow criteria not covered. No push/merge/deploy authority.
