# MCP job execution — R044

Date: 2026-10-03 (Asia/Saigon). Codex SPEC_AUTHOR. Risk ceiling R2.
Status: BUILT locally at exact BUILD 48918e2f953dbb32e7f5f159b1eceab7dfe3848f (REVIEW_PENDING, independent review not done). Work order: [R044](../work_orders/CCMAI_RUNTIME_044.md).

## Intake and design

Baseline `6cebcc0`: MCP performs an authorized own-tenant lookup and returns `job_trigger_unavailable` (accepted R035 contract). HTTP TriggerJob validates query parameters, validates configuration, reserves an owned run and launches RunReserved before returning 202/run_id. R044 implements the separate execution objective; R035 acceptance remains historical, source-specific evidence.

Introduce a transport-independent `backend/jobdispatch` package used by HTTP TriggerJob and MCP toolTriggerJob. It imports existing config/models/engine contracts; engine must not import it. Share configuration validation, reservation, worker start and start-failure abort. Keep authorization and tenant-scoped lookup in each transport, with HTTP query parsing and response mapping in its adapter. No handler-to-handler calls, fabricated Gin contexts, second queue, new coordinator, or direct JobRun insert. Existing engine source remains unchanged.

Preserve HTTP test-run and cancellation, R027 modes/dates/caps, and R028 ownership, timeout and fail-closed terminal behavior. Test-only dependency injection must intercept configuration loading and Analyzer creation before real environment/provider access. Prefer explicit service dependencies; preserve effective coverage of existing handler seams during extraction. Production wiring must use the existing Analyzer, not a test stub or unavailable fallback. A necessary protected-path change returns a scope finding before edits.

MCP keeps its tool name and required tenant_id/job_id schema. It initially supports only since_last, empty dates and cap zero. Reject extra arguments (including mode/full/from/to/limit) with a generic tool error rather than silently executing with different semantics. This does not broaden HTTP's parameter contract.

## Acceptance contract

| ID | Required behavior and discriminating evidence |
| --- | --- |
| JE-01 | Mounted authenticated MCP calls retain R021 requested-tenant stored membership and exact jobs:w plus messages:r rights. Owner/admin retain their existing bypass. Missing either right, malformed permissions, absent membership and foreign tenant fail before protected job lookup/config/reservation/start. All other tools' positive and negative contracts remain intact. |
| JE-02 | Own-tenant lookup uses both predicates. Unknown/empty/foreign job and forced lookup error return the existing `Job not found` tool error, no run ID and zero config/start/write effects. Extra-argument rejection occurs after authorization but before config/reservation/start, returning `invalid_run_parameters`. No raw SQL, config, provider or job data in public errors. |
| JE-03 | Successful tools/call returns a non-error ToolResult whose text is JSON with exactly message=`job_triggered` and run_id equal to the actual persisted reservation ID. Reservation and worker handoff must succeed first. This is acceptance of background execution, not completed analysis or a durable queue promise. Published tools/list description states this and the default-only semantics. |
| JE-04 | Config error or nil config returns `job_start_failed` before reservation; busy returns `job_already_running`; admission missing job returns `Job not found`; other admission/start failures return `job_start_failed`. These are ToolResult errors, never success/run_id or JSON-RPC protocol errors. Preserve authentication/unknown-tool protocol behavior. |
| JE-05 | MCP/MCP and concurrent MCP/HTTP requests share existing ownership: exactly one reservation/start for a tenant/job, other request busy; different jobs/tenants remain independent. Stored running row without local owner blocks. Mount both transport paths and coordinate finite barriers rather than timing sleeps. |
| JE-06 | Synchronous start failure aborts its bound reservation; successful abort closes error and releases local ownership without checkpoint/results/notifications. Failed abort leaves the stored running row blocking later admission even after local release. Observe both database state and ownership/retry outcomes, including forced finalizer failure; never delete the row to simulate cleanup. |
| JE-07 | Worker lifetime is independent of the MCP/HTTP request context and keeps the existing bounded timeout. A real shared worker invokes RunReserved exactly once with since_last/0/empty dates for MCP. Test request cancellation, normal completion, provider error, panic and cancellation using existing engine contracts and synthetic fixtures. Ownership remains held through finalization/notification interception and releases after the last effect. |
| JE-08 | HTTP trigger still returns 202 message/run_id, 409 busy, 404 missing, 500 generic config/admission/start errors, and existing 400 parameter/date errors. All three modes, full alias/conflicts, Vietnam date ranges and decimal caps remain unchanged and reach the worker unchanged. HTTP test-run limit/behavior and cancel routes are preserved. |
| JE-09 | Rejection paths show no protected writes/reservations/workers/outbound effects, with forced read/config/admission errors covered separately. Use mounted transports, write callbacks, table snapshots and startup counters with fixture setup outside observation. State limits: default-client traps alone do not prove universal socket absence. |
| JE-10 | Production shared path is exercised through mounted MCP and HTTP using real disposable MySQL and test-injected synthetic provider/config/notification interception. At least one MCP accepted run completes with matching persisted run/result identities. Real provider/channel/customer/credential/network execution is NOT RUN. This is local application evidence only; no CVF AI governance or live readiness claim. |

## Verification and review boundaries

Complete uncached MCP and API-handler suites, focused shared-dispatch tests, engine ownership/regression tests affected by this extraction, cached backend build/vet. Use root cwd and `go -C backend`, GOPROXY=off/GOTOOLCHAIN=local. Required DB cases must execute on disposable loopback MySQL with synthetic data; missing tools/images/dependencies return BUILD_BLOCKED, not downloads or skipped acceptance. Run race when available; identify individual skips and label unavailable checks NOT RUN.

Show a named old-source detector for the mounted MCP success contract against the unavailable baseline. Apply finite, compiling mutations for tenant predicate removal, early success before failed start, separate/bypassed reservation, request-context parenting, missing abort, altered HTTP parameter forwarding, and denied-request dispatch. Use bounded barriers/worker teardown and byte restoration; build errors, timeouts, survivors and inconclusive attempts are retained separately from semantic kills. A worker campaign is not independent reviewer proof.

Codex independently reviews exact Claude BUILD and source scope against the committed seed, repeats required mounted/regression evidence on isolated disposable fixtures, and checks dependency wiring, rejection ordering, cleanup and claim boundaries. No reviewer source repair or worker self-approval. REVIEW_PASS requires settled findings and existing evidence; FREEZE needs later separate disposition authority.

## Implementation truth

BUILT locally, review pending. The SPEC itself was prepared without execution; the worker BUILD ran only synthetic disposable-MySQL tests with an injected synthetic provider (evidence docs/reviews/MCP_JOB_EXECUTION_R044_BUILD_2026-10-03.md). R043 and prior accepted closures remain unchanged. Facebook/Zalo OA account setup/credentials/live tests are parked; Pancake live prerequisites and global F02/governance/hosted readiness remain open.
