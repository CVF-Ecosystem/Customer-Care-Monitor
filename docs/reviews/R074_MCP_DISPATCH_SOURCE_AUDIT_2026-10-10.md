# R074 MCP dispatch source audit

Status: READY_FOR_ROOT_REGISTRATION; independent review pending

Audit date: 2026-10-10

Scope: read-only source and documentary evidence review under CCMAI-RUNTIME-074

## Finding

The current MCP trigger path uses the shared jobdispatch.Service and acknowledges a successful reservation plus a successful in-process worker handoff with a run_id. That acknowledgment does not establish analysis completion, result publication, notification delivery, or crash recovery. The shared dispatcher reserves a persisted running JobRun and starts a goroutine; the worker then enters the Analyzer's reservation-aware execution path. These are source observations, not live behavior or delivery guarantees (backend/mcp/handlers.go:335-360; backend/jobdispatch/dispatch.go:66-148; backend/engine/analyzer_modes.go:148-183).

The source audit is bound to implementation baseline 9fc86fe012d1cc8c1aeb7d9e7a55e79c1c032c2d and documentary HEAD 67d8aa2eb32e80337881b342abb447f38c620a9d separately. The R074 acknowledgment/metadata commits after the baseline did not change backend/ in the inspected comparison. All 18 Go files relied on here have identical Git blobs at the two commits and matching physical SHA256 values recorded in [the source identity record](probes/r074_dispatch_audit_source_identity_2026-10-10.json). That record covers cited source files; it is not a worker re-hash of every item in the protected 439-file inventory.

## Current MCP call path

backend/api/router.go:246-250 mounts the MCP routes. backend/mcp/server.go:53-80 applies bearer-token authentication and attaches the authenticated user context. backend/mcp/server.go:96-128 handles MCP requests and routes tools/call to the tool-call handler.

backend/mcp/handlers.go:32-99 parses the call, obtains the user identity, checks the tool policy and authorization, then selects the tool implementation. The policy and authorization lookup are in backend/mcp/handlers.go:111-158; they check the tenant and user membership/rights in source. The trigger arguments and tool schema accept only tenant_id and job_id (backend/mcp/handlers.go:330-360; backend/mcp/tools.go:159-168). toolTriggerJob queries the job under the supplied tenant before dispatch. Extra arguments are rejected. It passes mode since_last to jobdispatch.Default.Dispatch.

The configured MCP tool description says this is a background run and that run_id is returned after reservation and worker start, not after analysis (backend/mcp/tools.go:159-168). Source confirms the response carries the run ID when the dispatch start function returns without error (backend/mcp/handlers.go:335-360; backend/jobdispatch/dispatch.go:66-115).

## Admission, handoff, and outcomes

Dispatch validates configuration before reservation, reserves through engine.ReserveJobRun using a 30-minute timeout and a background context, then calls its configured start function or StartWorker. The service returns the persisted reservation ID after that handoff returns successfully. A reservation failure stops before worker start; a start failure triggers an abort attempt before the MCP handler maps it to its generic start-failure result (backend/jobdispatch/dispatch.go:23-45,66-123; backend/mcp/handlers.go:335-360).

The observable source-level result classes are:

| Condition | MCP source mapping | Ordering / limit |
|---|---|---|
| Invalid dispatch configuration | Generic job_start_failed | Configuration validation precedes reservation. |
| Job already running | job_already_running | Returned from reservation before worker handoff. |
| Job missing | Job not found | Returned from reservation before worker handoff. |
| Other reservation or worker-start error | Generic job_start_failed | Start errors cause an abort attempt; abort persistence can itself fail. |
| Reservation and handoff accepted | job_triggered with run_id | Does not wait for Analyzer completion, saved results, notifications, or restart recovery. |

These mappings are source behavior only (backend/jobdispatch/dispatch.go:66-123; backend/jobdispatch/dispatch.go:125-148; backend/mcp/handlers.go:335-360). The handoff starts an in-process goroutine. A panic in that worker is recovered and followed by an abort attempt; that is not evidence of process-restart recovery or a durable queue (backend/jobdispatch/dispatch.go:125-148).

## Reservation, context, and terminal ownership

The coordinator first claims a process-local slot keyed by tenant and job, then performs the database reservation transaction, which locks the tenant/job scope, checks for an existing running row, and inserts a running JobRun (backend/engine/job_run_ownership.go:107-190). Thus source has both in-process ownership and persisted admission state. A persisted running row by itself does not establish a queue, lease recovery, distributed deduplication, or exactly-once execution.

The MCP dispatch path uses context.Background with its configured timeout instead of propagating the incoming MCP request context. RunReserved validates and consumes the reservation before entering Analyzer execution (backend/jobdispatch/dispatch.go:66-115; backend/engine/analyzer_modes.go:148-183). The owner serializes publication/cancellation and terminal transitions, and releases its process-local ownership at the end of the Analyzer execution (backend/engine/job_run_ownership.go:193-315; backend/engine/analyzer.go:39-90).

On the ordinary analysis path, terminal persistence updates the run and job/checkpoint state before the Analyzer attempts notifications. Notification is conditional on analyzed results, configured outputs, and cancellation state. The worker remains in the Analyzer path through that notification attempt before deferred ownership release (backend/engine/analyzer.go:343-425; backend/engine/analyzer_incremental.go:167-283). The notification dispatcher reads results, sends through configured notifier code, and marks results notified; some database read/write errors are not checked at the shown call sites (backend/notifications/dispatcher.go:39-125). This is a source-level reliability question, not a reproduced incident or proof of external delivery.

## Other entrypoints and shared seams

| Entry point | Dispatch/execution seam | Relationship to MCP |
|---|---|---|
| MCP cqa_trigger_job | jobdispatch.Default.Dispatch with since_last | Uses the shared dispatcher, tenant lookup and engine reservation. |
| HTTP manual /trigger | TriggerJob adapts to the shared dispatch service; HTTP parses mode/date/cap options | Shares the dispatcher and reservation seam. HTTP accepts options MCP does not expose. |
| HTTP test-run | Reserves a run and calls StartWorker directly | Shares engine reservation/worker code but bypasses jobdispatch.Service. |
| Scheduler cron and after-sync | Calls Analyzer.RunJob directly | Shares Analyzer reservation/ownership; does not use jobdispatch.Service. |
| Agent analysis | Calls Analyzer.RunJob for tenant jobs | Shares Analyzer reservation/ownership; does not use jobdispatch.Service. |
| Agent/manual/scheduled channel sync | Uses SyncEngine and SyncReservation | Separate channel-sync lifecycle, not jobdispatch or JobRun admission. |

Sources: HTTP routes and MCP mounting (backend/api/router.go:129-143,174-185,236-250); HTTP trigger and parameter parsing (backend/api/handlers/jobs.go:402-452,504-544); test-run and worker start (backend/api/handlers/jobs.go:368-400,462-497,555-562); agent routes (backend/api/handlers/agents.go:124-176,235-310); scheduled/after-sync calls (backend/engine/scheduler.go:279-363); channel sync (backend/engine/sync.go:248-324,337-366); ordinary Analyzer reservation (backend/engine/analyzer.go:39-53).

## Analyzer, provider, receipts, and notifications

The Analyzer prepares and records source observations before lazily creating a provider, and creates it only when prepared conversations require analysis. The provider method calls in source are AnalyzeChat and AnalyzeChatBatch. The provider-construction source reads tenant AI settings and decrypts a configured key, but this audit did not read runtime configuration or credentials and did not call a provider (backend/engine/analyzer.go:101-200,230-330,427-476,744-891).

The preparation, execution, usage, and rule receipts are bounded local observations carried in JobRun.Summary. Preparation caps and omission/unvisited semantics are explicit in source; the execution receipt is scoped to the Analyzer provider interface; usage observations explicitly say billing is not observed and price revision is not captured; rule observation does not confer permission or control execution (backend/engine/source_preparation_receipt.go:12-90,180-275; backend/engine/source_execution_receipt.go:9-75,137-256; backend/engine/usage_observation_receipt.go:5-6,72-91; backend/engine/rule_observation_receipt.go:13-63).

This audit invoked no provider, channel, notifier, or database operation. It establishes no runtime API call, billable usage, invoice, successful durable write, notification delivery, or CVF governance behavior.

## Historical evidence and its boundary

The records below are documentary inheritance only. R074 did not rerun their tests and they do not certify the current baseline or live behavior.

| Record | Accepted scope and exact-source evidence | Limits relevant here |
|---|---|---|
| [R035 R1 independent re-review](CCMAI_RUNTIME_035_R1_INDEPENDENT_REREVIEW_2026-10-03.md) | REVIEW_PASS for the local MCP contract; repair 22204abc0a19841cd9c50ce03c0af032896d3b4e, original product build 10ad83381ce76b86763c1ee06eab02ddf4734cac. Exact local archived tests reported 18 top-level / 28 total passing outcomes, plus targeted forced-write mutation checks. | This is historical and predates the current dispatch path. It does not establish the current jobdispatch integration, real queue execution, or live provider/channel behavior. Full backend and race were not run. |
| [R044 initial review](CCMAI_RUNTIME_044_INDEPENDENT_REVIEW_2026-10-04.md) and [R044 R1 re-review](CCMAI_RUNTIME_044_R1_INDEPENDENT_REREVIEW_2026-10-04.md) | The initial build was CHANGES_REQUIRED; the separate R1 re-review accepted repair source b4ec91ea03a9f247e209389c3792c86494eac8b3 with local grouped tests (393 completed passes in the reported campaign; targeted engine 54 passed). The accepted repair was test-only notification coverage. | Keep the initial finding and later repair distinct. These tests were local; no real provider, channel, configuration, credential, network, or persistent production data was used. |
| [R045 local MCP execution closure](CCMAI_RUNTIME_045_LOCAL_MCP_EXECUTION_CLOSURE_2026-10-04.md) | Separate local closure inheriting the accepted R044 source/evidence under R045 seed 73e3de2f33031248f1a14fe11dcfb6e85bac1876; no new product/runtime implementation. | A metadata/local execution-contract closure, not live or current-head proof. |
| [R053 successor review](CCMAI_RUNTIME_053_SUCCESSOR_INDEPENDENT_REVIEW_2026-10-06.md) | Accepted bounded lazy-provider evidence for production source prefix 727d322 and test snapshot prefix 145bd411; inherited 42 top-level / 98 passing outcomes. | Evidence was inherited, not rerun by R074. Full engine/backend, provider/channel, runtime governance, and hosted CI were not run. |
| [R057 independent review](CCMAI_RUNTIME_057_INDEPENDENT_REVIEW_2026-10-06.md) and [R058 closure](CCMAI_RUNTIME_058_LOCAL_PREPARATION_RECEIPT_CLOSURE_2026-10-06.md) | R057 accepted source 05c59e9978e5cf108b0d9118f64c7454805ec64e and bounded preparation receipt tests (114 completed top-level executions / 272 passing outcomes, including restored tests and a killed semantic mutation). R058 closed only that local observation contract. | Receipt persistence was tested locally; no real provider/channel or broader runtime governance proof. |
| [R061 independent review](R061_INDEPENDENT_EXECUTION_RECEIPT_REVIEW_2026-10-07.md) and [R062 closure](R062_LOCAL_EXECUTION_RECEIPT_CLOSURE_2026-10-08.md) | R061 accepted source d10e164249b5e7a87206d1d3bc5aec43a1a8ac2f; 38 top-level / 70 passing outcomes, two named semantic mutations killed, then 16 restored / 48 passing outcomes. R062 closed the bounded local execution-observation contract. | These were logical Analyzer interface observations, not HTTP attempts, billed calls, provider billing provenance, or live execution. Full backend/race were not run. |
| [R064 local rule-observation closure](R064_LOCAL_RULE_OBSERVATION_CLOSURE_2026-10-08.md) | Accepted source 55e836a1b980b3e5d4075ceb33466786c860f963 with 53 top-level / 152 passing outcomes, semantic mutation checks, and restored controls. | Original summary/container snapshots were unavailable; the record documents reconstruction limits. No provider behavior was proven. |
| [R065 usage-observation review and closure](R065_INDEPENDENT_USAGE_OBSERVATION_REVIEW_AND_CLOSURE_2026-10-08.md) and [R068 final usage-presence review](R068_FINAL_INDEPENDENT_USAGE_PRESENCE_REVIEW_2026-10-10.md) | R065 accepted source 72e4c367707d262e45ed9f36164d3e471a4d04b4 as local estimate/observation only. R068 accepted source bb0bdeb5403070eb0aa2224b96ad8fd48febd843 for adapter usage-presence parsing, with local AI tests and semantic mutation checks. | Neither establishes billing/invoice truth or a real provider API call. R068 full-backend/race/provider use was not run; its separate compile review is not provider evidence. |

## Evidence classes and unknowns

**Confirmed source facts:** the statements above tied to code paths and line ranges, plus source identity hashes in the JSON record.

**Inherited tested evidence:** only the exact-source local test and review results named in the historical table. They were not executed by R074 and are not generalized to current HEAD.

**Inferences:** the in-process goroutine handoff is not a durable queue; persisted admission state does not itself show restart recovery; shared reservation code does not establish distributed or exactly-once execution. These are bounded conclusions from the inspected paths, not claims about every repository component or deployed topology.

**Unknown / unverified:** whether deployed processes recover running rows after a crash; whether multiple instances coordinate safely; whether a client disconnect affects a started run; actual database durability under faults; whether external notification sends succeed or duplicate; whether provider calls and result writes succeed in any deployment; and any real provider billing or CVF governance behavior. R074 exercised none of these runtime conditions.

## Bounded next-scope options

1. **Decide asynchronous dispatch durability and recovery semantics.** Before implementation, specify deployment topology, restart behavior, duplicate/idempotency policy, and who owns stranded running rows. A new work order must name permitted runtime/data effects and exact acceptance evidence. R074 grants no queue, recovery, or multi-instance change authority.
2. **Decide MCP/HTTP trigger-option parity.** Product/spec owners should choose which modes, dates, and caps MCP should accept, and whether the since_last default is intentional. A separate spec/work order should define compatibility and testable request/response behavior before any source change.
3. **Specify notification reliability and effect handling.** Decide retry, duplicate suppression, and surfaced failure semantics for post-terminal notification attempts. Local controllable-notifier tests can establish source behavior; contacting real destinations requires separately authorized external-effect evidence. Facebook and Zalo OA account work remains parked.

All three are proposals only. No implementation, test, live dispatch, account action, or runtime effect occurred in R074.

## R074 artifact and review boundary

This audit consists only of this narrative, [structured audit facts](probes/r074_dispatch_audit_facts_2026-10-10.json), and [source identity hashes](probes/r074_dispatch_audit_source_identity_2026-10-10.json). Root owns index/registry and metadata; independent review remains pending. No source, test, existing evidence, seed, catalog, continuity, configuration, or core artifact was edited. No application tests were run by this worker.
