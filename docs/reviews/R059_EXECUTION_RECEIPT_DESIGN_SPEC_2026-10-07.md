# R059 candidate — Analyzer execution observation DESIGN/SPEC

Date: 2026-10-07 (Asia/Saigon). Author: Codex `/root` ORCHESTRATOR / SPEC_AUTHOR. Status: SPEC_DRAFT_READY_FOR_BOUNDED_WORK_ORDER. Planning document only: no R059 seed, work order, active tranche, implementation or runtime proof is created by this document. Existing R058 local FREEZE remains intact. Document risk R1; future execution-path implementation is R2 and requires independent implementation/review responsibilities. Owner instructs root continuation without subagents; no agent dispatched.

Baseline: published project head `083e0bae9d440ff94a3ce77c5298f1a5808e9aa7`, accepted application source `05c59e9978e5cf108b0d9118f64c7454805ec64e`, independent review `5ff087587f04170579b39a645ce4bf7eb1933fe1`, [R058 closure](CCMAI_RUNTIME_058_LOCAL_PREPARATION_RECEIPT_CLOSURE_2026-10-06.md). Roadmap S2/S3: `docs/roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md`, [preparation contract](../specs/SOURCE_PREPARATION_RECEIPT_R055_2026-10-06.md). Source inspection only; no provider/channel/config/credential/customer/DB/network/runtime action in this planning pass. No claim of full S2, live parity, hosted CI or CVF controlling AI behavior.

## INTAKE and verified implementation boundary

The accepted `source_preparation` receipt describes preparation only. PREPARED_FOR_INFERENCE does not establish provider construction, invocation, response validity or saved results. The next useful dependency is observation of the existing execution boundary, before adding new admission policy.

| Source inspected at baseline | Current behavior and consequence |
|---|---|
| `backend/engine/analyzer.go`, `executeReserved` single path | Calls `AnalyzeChat` after the existing cancellation check. Usage-row insertion precedes result publication; a returned response can coexist with rejected/failed publication. Existing scalar analyzed count increments only after save success. |
| Same file, `runBatchMode` | One `AnalyzeChatBatch` call covers several prepared conversations. A provider error counts errors per item; malformed batch rejects parsing; valid batch may publish a prefix before cancellation. One usage row describes the entire batch call. |
| Same file, finalizer and panic paths | Summary is written initially/progress/final and on selected early failures. Final transaction failure returns in-memory observation without proving stored final state. Receipt must preserve this distinction. |
| `backend/ai/provider.go`, `AIProvider` / `AIResponse` | Two logical methods return content, integer token counts, model/provider and error. The interface does not expose HTTP attempt count, usage-presence flags, billing settlement or durable provider request ID. |
| `backend/ai/retry.go`, `withRetry` | One logical call can execute the adapter function up to four times. Analyzer invocation count is not HTTP request count or billed-attempt count. |
| `backend/ai/claude.go`, `gemini.go`, `openai_compatible.go` | Batch methods wrap input and delegate to single method. Gemini leaves zero counters when usage metadata is absent; compatible response uses zero-valued integer fields without presence flags. A zero adapter count does not prove zero billed usage. |
| `backend/ai/provider.go`, `CalculateCost`; `backend/ai/pricing/pricing.go`, `Lookup` | Unknown model price returns `(0,false)`. Rates may be synced or static; returned cost is a local estimate, not an invoice or price-version receipt. No external pricing lookup was performed. |
| `backend/db/models/setting.go`, `AIUsageLog`; existing usage writes | Numeric cost has no known/unknown field. Analyzer currently discards `Create` error. A response does not prove a usage row was inserted. |
| `backend/db/models/job.go`, `JobRun.Summary` | Existing JSON field can carry additive observation. Schema, usage model and dashboard changes belong to later scopes. |

These statements are local source findings, not reproduced production incidents. Existing raw evidence, failed campaigns, seeds and budgets are preserved; no test or source repair is implied.

## DESIGN decisions

Choose a new `source_execution` member of the existing Summary, separate from `source_preparation`. Version `ccmai.source-execution.v1`; scope `analyzer_provider_interface_only`. Do not rename preparation fields or reinterpret their counts. Observe the application method boundary; never label its counts `api_requests`, `http_attempts`, `billable_calls` or `real_provider_calls`.

Preserve query selection, preparation, provider choice, prompts, retry/delay, cancellation, owner locking, result validation/save, scalar counters, checkpoint, notification and terminal behavior. Observation must not grant permission to infer, retry a failed call, rescue invalid output, skip ambiguous source or mark a canceled result published.

Record logical calls once immediately before the existing method invocation, after the existing cancellation gate. One batch is one logical call with N members, not N calls. Keep ordered input binding as bounded sanitized UUIDs; never retain prompt, transcript, response, rules, names, URLs, credentials, arbitrary provider/model strings or raw errors. Include only tenant/job/run binding, existing mode, sequence, SINGLE/BATCH, item counts and fixed enums. Source metadata remains in the preparation receipt; no extra queries to join snapshots or recover omitted entries.

No model/provider identifiers or monetary/token values in this first slice. Usage-write outcome may be observed from the existing `Create(...).Error` without changing whether execution continues. Adapter usage provenance remains NOT_CERTIFIED. Cost truth and rules/permission provenance require separate contracts; no fabricated rule version, human approval or numeric zero-cost claim.

## SPEC: observable facts

Envelope contains sanitized tenant/job/run IDs, mode, scope/version, exact aggregate counters, retained call entries, omission counts, entry completeness, execution completeness and a fixed stop reason. Execution completeness means observed logical-call processing reached its normal local boundary, not that all tenant data was processed, all outputs were saved or a provider invoice settled.

Each call has a monotonically increasing run-local sequence, method SINGLE/BATCH, `item_count`, retained ordered member UUIDs, omitted-member count and independent member completeness. It records the following dimensions without conflating them:

| Dimension | Allowed meaning |
|---|---|
| Invocation outcome | IN_FLIGHT, RESPONSE_RETURNED, ERROR_RETURNED, INTERRUPTED. Begin before invocation; returned error wins if both response and error are present. Panic leaves INTERRUPTED, never fabricated ERROR_RETURNED. |
| Usage-write outcome | NOT_ATTEMPTED, WRITE_SUCCEEDED, WRITE_FAILED. Observe the existing write result only. Success means that operation returned success, not verified later persistence or full billing coverage. Provider errors and panic before the write remain NOT_ATTEMPTED. |
| Parsing outcome | NOT_ATTEMPTED, ACCEPTED, REJECTED. Batch parser rejection is explicit; single parsing is combined with existing `saveResults`, so use NOT_SEPARATELY_OBSERVABLE for that path rather than inventing a parser result. |
| Publication counts | `items_saved`, `items_save_failed`, `items_not_published`, `items_pending`. Saved increments only after the existing owner publication succeeds and save returns nil; a model's pass/fail meaning is separate. |
| Stop classification | NONE, CANCELLED, OWNERSHIP_NOT_PUBLISHED, PANIC, PROVIDER_SETUP_FAILED, PREPARATION_STOPPED, EXECUTION_ERROR. Use existing observed branches; do not infer ownership loss from an opaque false result. |

For single calls, NOT_SEPARATELY_OBSERVABLE is an explicit additional parsing enum. Save failure is not claimed to mean parsing failure: `saveResults` can fail in validation or storage. For batch REJECTED, all members become not-published; no per-item successful parsing is fabricated. On provider ERROR_RETURNED, all members become not-published. On cancellation during a valid batch, retain actual successful/failed prefix, count the remaining suffix as not-published when that stop is observed. On panic while publication is in progress, unresolved members remain pending; never convert uncertain storage outcomes to saved or failed.

For each logical call: `item_count = items_saved + items_save_failed + items_not_published + items_pending`. Usage rows and items_saved need not agree. Envelope aggregates include every call/member irrespective of retention caps. Invocation counters reconcile exactly: begun = response_returned + error_returned + interrupted + in_flight. A normal terminal receipt has no in-flight call; panic may finalize an active call as interrupted with unresolved publication pending. Provider construction failure/empty source/unchanged ordinary source produce zero logical calls, with honest stop/preparation context.

## Bounds, privacy and persistence

Retain first 200 logical-call entries and at most 200 member bindings across the entire execution envelope, in original order. Cap encoded execution envelope to 128 KiB, including aggregate metadata. Never drop or rewrite `source_preparation` to fit the new receipt. Combined observation overhead is bounded by the existing 256 KiB preparation budget plus 128 KiB execution budget, excluding unchanged legacy scalar fields. Aggregate counts use all observations; report omitted calls and members separately. Stored lists are not exhaustive when flags indicate omission. Tests must include adversarial large batches and prove a compact aggregate envelope still fits.

Strict allowlist serialization, bounded integers with explicit invalid metadata flag, valid UUID normalization and frozen deep copies. Never hash raw prompts/rules as a substitute for excluding them. Fixed error classes contain no wrapped error, SQL, HTTP URL or provider response. No claim that existing application logs are completely redacted by this new receipt.

Add execution observation through a shared Summary builder at every current writer: initial, single error/progress, batch progress, final, early provider failure and panic. Preserve legacy `closeOwnedRun`/Abort and invalid/already-consumed reservation behavior. Frozen snapshots must reflect the current execution prefix, while the preparation receipt remains unchanged. Keep mutable collector run-owned; no new concurrency or shared global map. Return Summary observation after final write failure exactly as existing behavior allows; do not put a claimed terminal-write success inside a payload before its transaction succeeds. Durable proof requires reload after a successful terminal write.

Existing usage-write failure continues with its original result/publication behavior; recording the already available error is observational. No new Summary reads, database query, schema migration, usage-log column, provider wrapper, pricing sync, permissions or UI change.

## Acceptance matrix for the future work order

| ID | Required detector and success condition |
|---|---|
| EX01 | Provider construction is distinct from invocation: empty/unchanged/preparation-error/setup-error paths begin zero calls; existing preparation receipt/scalars remain equal. |
| EX02 | Single success binds one call/one member and independently observes response, usage write and save; stored terminal row matches receipt after reload. |
| EX03 | Response plus error is ERROR_RETURNED with no fabricated usage/save, error branch/counters unchanged; ordinary and explicit modes retain their behavior. |
| EX04 | Mixed successful/failed calls reconcile all envelope/item counts. Save failure after response is not a provider failure or certified parse failure. |
| EX05 | Batch N members = one call. Malformed batch yields response-returned plus parser-rejected, zero saves, N not-published; no duplicated token/cost allocation. |
| EX06 | Existing cancellation barrier before call yields no new invocation. Cancellation while provider is running records its return and rejected publication; mid-batch cancellation preserves the actual published prefix. |
| EX07 | Provider and publication panic retain begun/interrupted and honest pending storage outcomes; terminal/ownership release/legacy Abort behavior unchanged. |
| EX08 | Usage insertion fault is WRITE_FAILED, not missing response or failed save. Existing flow/scalars continue; a canceled response can have a usage row without saved output. |
| EX09 | Initial/progress/early/final write faults retain honest returned-versus-stored limits; mutation erasing only stored terminal receipt must fail a named reload assertion. |
| EX10 | Above both entry/member limits and serializer byte limit, exact aggregate counts/order/omissions reconcile; privacy sentinels absent in receipt; preparation bytes unchanged. |
| EX11 | Legacy scalar keys/values, admission, provider selection/retry, snapshot reference queries, checkpoint and notifications have behavioral regression controls; all old tests retained. |
| EX12 | Applied mutation conflating prepared candidates with invoked calls must fail a named no-call detector; restore exact exported source bytes and show healthy controls pass. No synthetic governance claim. |

Future local tests are synthetic application tests and disposable DB checks, not real-provider governance evidence. A later real-provider/governance claim requires separately bounded authorized provider use and request/response receipt; do not use these tests as substitute proof.

## Proposed WORK_ORDER boundary; not dispatched

Candidate existing paths: `backend/engine/analyzer.go`, `backend/engine/source_preparation_receipt.go` only for the shared builder composition, and `backend/engine/job_run_ownership.go` only if required to preserve receipt-aware early terminal handling. Candidate new files: `backend/engine/source_execution_receipt.go`, `source_execution_receipt_test.go`, `source_execution_receipt_db_test.go`. Exact paths must be validated by the work-order author before an immutable seed; this list is not implementation authority. Protect every old test, original authority seed, committed evidence packet and source outside the approved list.

Before BUILD: one consolidated contract/path/role review; immutable seed committed before activation; root no-subagent instruction retained; implementation and independent R2 reviewer named without provider hardcoding; documented before-edit acknowledgment and build commit ownership. Root may implement only after taking that role under the new work order, and may not accept its own R2 source. Independent review remains undispatched; planning must not silently name the historical child as active worker/reviewer. No request for owner permission is needed to finish this planning document; source BUILD and acceptance cannot be inferred from the old closure seed.

Proposed budget to settle at dispatch: one worker campaign with at most four Go commands, one independent reviewer campaign with at most four Go commands; aggregate max eight. Group pure/new DB/regression/mutation-restoration tests; cache compilation on isolated task resources, retain raw outputs on timeout, exact archive manifests and cleanup proof. No automatic retry, source repair or budget reset on infrastructure failure; disposition first. Python repo checks are separate from application Go budget. Existing R055/R057 consumed budget remains historical, not reused.

## Remaining S2/S3 dependencies

After independently accepting execution observation, design rule/permission/version/WAIT_DATA receipts against actual authority and implementation. Keep prompt skip conditions separate from deterministic gate decisions. Then design adapter usage-presence, model/rate identity, known/unknown/pending estimates and usage-row/dashboard compatibility; no zero-as-free claim. Provider-attempt, billing settlement, crash reconciliation, PII enforcement, budget reservation, human review, immutable audit and actual live/governance evidence remain later work. R059 candidate alone cannot close S2 or start an implicit full S3 BUILD.

## Planning disposition

Root source audit and design consistency check completed; document-ready, implementation NOT_STARTED, evidence NOT_RUN, R2 acceptance NOT_ESTABLISHED. R058 remains FROZEN, original failures/seeds/runtime evidence unchanged. Next allowed move is bounded WORK_ORDER preparation with explicit independent reviewer routing before any source/test BUILD. Current continuation does not authorize subagents, provider/channel/credentials/config/customer/persistent DB/core edits, merge or deployment. Publication request from 2026-10-06 covered the prior committed history; this new planning commit is local unless separately published within valid authority.

Publication setup findings retained: first continuity check rejected abbreviated R058 in nextAllowedMove; root changed the current tuple to exact CCMAI-RUNTIME-058. First docs build rejected a link to a roadmap intentionally excluded from the site; root rendered its canonical path as code. No gate/config waiver or source repair. Final publication checks are recorded in `docs/reviews/probes/r059_planning_publication_checks.json`.
