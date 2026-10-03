# MCP trigger-job response truth — R035

Date: 2026-10-03 (Asia/Saigon). Author: Codex SPEC_AUTHOR. Status: ACCEPTED_FOR_BOUNDED_BUILD, implementation pending. Risk ceiling R2. Source baseline `45d14730b2518638276825d9a0b6cfe2518bbfe4`. This contract covers local application behavior only; no CVF runtime governance or real AI/channel invocation claim.

## Intake and design

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

NOT BUILT. Existing source still reports triggered/queued. New acceptance begins only after exact Claude BUILD and independent Codex REVIEW. No FREEZE, hosted readiness, deployment or global F02 closure follows from this specification.
