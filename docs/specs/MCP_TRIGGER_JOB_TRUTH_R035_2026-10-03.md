# MCP trigger-job response truth — R035

Date: 2026-10-03 (Asia/Saigon). Author: Codex SPEC_AUTHOR. Status: R1 REVIEW_PASS / FREEZE_OPEN for the local contract after independent re-review. Risk ceiling R2. Source baseline `45d14730b2518638276825d9a0b6cfe2518bbfe4`. This contract covers local application behavior only; no CVF runtime governance or real AI/channel invocation claim.

## Intake and design

*Historical baseline: the next paragraph describes the pre-R035 source at `45d14730b2518638276825d9a0b6cfe2518bbfe4`. Current behavior is in Implementation truth.*

`backend/mcp/handlers.go::toolTriggerJob` performs a tenant-scoped lookup and returns triggered/queued without a queue or Analyzer. `tools.go` promises immediate execution. R021 permission tests deliberately preserve this old placeholder; IMPLEMENTATION_STATUS records its correction as a separate need. Owner reaffirms Codex orchestration/review under standing local authority; Claude implements after manual transfer.

Choose a truthful unavailable response while keeping the tool name/schema, permission policy, tenant lookup and existing non-dispatch behavior. Actual execution/queue wiring is a separate future objective. This tranche has no live-input dependency. R034 offline acceptance, R033 local FREEZE and all live boundaries are inherited unchanged.

## Acceptance contract

| ID | Required behavior and discriminating evidence |
| --- | --- |
| MT-01 | An authenticated, authorized member with both jobs:w and messages:r, or owner/admin, calling an existing own-tenant job receives ToolResult `isError=true`, one text content containing exactly `job_trigger_unavailable`, and no JSON-RPC protocol error. No triggered/queued/accepted/run identifier or job name is returned. Direct and mounted tools/call paths must agree. |
| MT-02 | Admission still precedes job lookup. Missing either right, unknown role, malformed permission data, absent membership or cross-tenant call retains R021's denial contract and zero protected job lookup. Owner/admin bypass letters only. Preserve all other tools' positive/negative contracts. |
| MT-03 | Job lookup retains both job ID and requested tenant predicates. Unknown, empty or other-tenant job IDs under an admitted tenant return the existing generic `Job not found` tool error. A forced job-query error returns that same generic result without SQL/driver/job data. Query observers and a cross-tenant job fixture discriminate accidental predicate removal. |
| MT-04 | Both unavailable and denial/not-found/error paths perform no writes, job-run creation, queue/analyzer/sync/media/notification dispatch or external requests. Observe write callbacks and before/after synthetic table state, including jobs, job_runs, messages, results and notification records. Use a bounded test-only outbound trap if needed; do not introduce production dispatch hooks just for tests. State the observation limits. |
| MT-05 | Published `cqa_trigger_job` description clearly states MCP execution is unavailable and the call returns an error without starting/queuing a job. Preserve its name/input schema and every other tool description/policy. Check actual tools/list output; no promise of immediate execution. |
| MT-06 | Update R021 test expectations only for this authorized response/description change. An admitted trigger now reaches the observed own-tenant lookup and returns unavailable; that is distinct from permission denial. Do not teach all admitted tools to accept arbitrary errors, weaken query/tenant/side-effect assertions or remove unrelated cases. Complete MCP suite passes on disposable MySQL with zero DB skips. |

Keep handler changes confined to toolTriggerJob and necessary comments, tool-description changes confined to cqa_trigger_job. No edits to authorization helpers/policies, OAuth/server routing, HTTP jobs endpoints, engine, schemas, configuration or frontend.

## Verification and failure evidence

Use cached Go tooling only (`GOPROXY=off`, `GOTOOLCHAIN=local`, root cwd, `go -C backend`). Disposable MySQL with synthetic fixtures and loopback transport is permitted; do not connect to the running app/production DB or inspect its configuration/secrets. Missing local tool/image/DB prerequisites yield BUILD_BLOCKED; do not download or silently skip required DB cases.

Run new focused contract tests and all `./mcp` tests with `-count=1 -json`; report top-level/subtest PASS/FAIL/SKIP and no unavailable-DB acceptance. Run build/vet and available race, retaining an honest unavailable-race NOT RUN. Full DB-dependent backend suite is not required for this bounded response change; label it NOT RUN. Required repository preflights/gate tests/catalog/doctor/diff still apply. Preserve prior R021 acceptance as historical source-specific evidence.

Show new response and tools/list detectors fail semantically on the exact pre-R035 baseline. Apply finite mutations for (1) removing isError from unavailable, (2) restoring triggered/queued response, (3) removing the tenant predicate, and (4) restoring the immediate-run description. Each must compile and fail named assertions; restore bytes/hashes and rerun baseline. Preserve setup/build failures, survivors and NOT_APPLIED separately. These tests validate application contracts on synthetic data, not governance decisions or provider execution.

## Implementation truth

REVIEW_PASS locally, FREEZE_OPEN. Claude's original BUILD (evidence: [BUILD record](../reviews/MCP_TRIGGER_JOB_TRUTH_R035_BUILD_2026-10-03.md)) changes only `toolTriggerJob` and the `cqa_trigger_job` description: an admitted call with an existing own-tenant job returns the tool error `job_trigger_unavailable`; miss or query error stays `Job not found`; authorization policy, tenant predicate and the no-dispatch behavior are unchanged. The tests include disposable-MySQL contract tests with zero skips, old-source detector failures and nine applied mutations (all killed after one survivor was fixed). The first BUILD was returned CHANGES_REQUIRED (forced jobs-read-error no-effects coverage, stale prose); the R1 repair adds that detector without changing product source. Exact repair22204abc0a19841cd9c50ce03c0af032896d3b4e is accepted by independent Codex re-review; MCP18/28 PASS0 skips and repaired GORM/raw error-only detectors verified. Worker11 campaign remains attributed. Race NOT RUN (CGO unavailable); full DB-dependent backend suite NOT RUN. No FREEZE, hosted readiness, deployment, actual job execution or global F02 closure follows from this specification.

Historical initial review of exact BUILD10ad83381ce76b86763c1ee06eab02ddf4734cac returned CHANGES_REQUIRED: MT-04 committed no-effects campaign omits forced jobs-read-error observation; independent error-only write mutant survives its suite, while the reviewer probe detects it. Bounded R1 also retires stale current handoff/order planning prose. This does not establish a production write/dispatch defect. [Review](../reviews/CCMAI_RUNTIME_035_INDEPENDENT_REVIEW_2026-10-03.md); original worker evidence above remains source-specific history, not acceptance.

Current acceptance: [R1 independent re-review](../reviews/CCMAI_RUNTIME_035_R1_INDEPENDENT_REREVIEW_2026-10-03.md) settles R1-01..02 for this local contract; previous repair-next instructions are historical. NOT RUN and bounded callback/checksum/default-client limits unchanged. No actual dispatch, live/governance claim or FREEZE.
