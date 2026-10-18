# R094 M2 intervention worker return

Status: COMPLETE_PENDING_REVIEW
Tranche: CCMAI-RUNTIME-094
Role: `/root/m2_worker` IMPLEMENTATION_WORKER; Codex `/root` independent REVIEWER and COMMIT_STEWARD
Source base: `2baca924a836dc11006b1680d72a0b44770a2fa9`
Commit: NONE — worker did not commit

Historical initial source/receipt only. Root's [consolidated review](R094_INITIAL_INDEPENDENT_M2_REVIEW_2026-10-14.md) returned F1–F4; the revised source is identified in [repair1 return](R094_R1_M2_INTERVENTION_REPAIR_RETURN_2026-10-14.md). The original PASS/failure logs and hashes below are preserved and do not validate the revised source.

## Actual local capability

An explicitly configured `InterventionService` can run one fixture-provisioned manual `intervention` job through the authenticated tenant HTTP route. It records an AnalysisSnapshot, a stable tenant/job/conversation/use-case case, a revisioned proposal when the output validates, and readable queue/detail responses. Empty or insufficient source records WAIT_DATA without a transport call. The tenant/job MySQL ledger consumes its configured cap before the one HTTP attempt; an ambiguous transport outcome retains the consumed slot. Job-run ownership is shared with the existing coordinator, and accepted cancellation prevents publication. The ordinary application leaves `handlers.InterventionRunner` nil, so this route is unavailable until a separately reviewed server-owned installation; this worker return is an injected application-route fixture, not an installed live product.

Source identities: [source SHA-256 manifest](probes/r094_worker/source_hashes.json). Implemented source families are `backend/db/models/intervention.go`, `backend/engine/intervention.go`, `backend/api/handlers/intervention.go`, route/migration/legacy guards, and a scoped `AnalyzeChatOnce` path. Existing ordinary adapter defaults remain callable. No frontend, M3 human mutation, general setup endpoint, real channel or customer data, notification, provider credential, or live provider effect was added.

## Evidence and counters

Worker Go invocations **6/6**: compile-only checks 1–2 PASS; native attempt 3 FAIL before application assertion because fixture `raw_data` was empty JSON; native attempt 4 FAIL after WAIT_DATA because test config's rate cap was zero; attempts 5 and 6 PASS after fixture corrections. Raw outputs are retained at [first failure](probes/r094_worker/native_first_fail.log), [second failure](probes/r094_worker/native_second_fail.log), [first pass](probes/r094_worker/native_initial_pass.log), and [final pass](probes/r094_worker/native_final_pass.log). Final command from `backend/` with `GOPROXY=off`, `GOSUMDB=off`, and only the owned fixture DSN: `go test ./api/handlers ./ai ./engine -run '^(TestR094|TestOpenAICompatibleGoiVaDocPhanHoi$)' -count=1 -v`. Exit 0. The native route test executed through `api.SetupRouter`, JWT and tenant middleware, actual disposable MySQL, and loopback `httptest` HTTP. Its synthetic response is software test data, **not CVF governance proof**. The engine package had no test matching that selector; its result is compile-only.

| Contract | Worker evidence | Remaining gap |
| --- | --- | --- |
| M2A01 manual positive persistence | Native route asserted proposal/detail/queue, snapshot-bound output and budget consumed. | Independent source/schema review pending; default installation disabled. |
| M2A02 deny/wait zero call | Empty WAIT_DATA and member permission denial both left loopback call count zero for those branches. | Foreign/real-channel, partial sync, missing transaction truth, PII/policy variants not separately exercised. |
| M2A03 durable cap | Native route asserted consumed cap after service recreation and no second call; ambiguous transport retained consumed slot and UNKNOWN attempt. | Simultaneous contention, alternate caller identity, DB settlement failure and crash restart NOT_RUN. |
| M2A04 strict evidence | Unknown canonical message ID produced OUTPUT_INVALID and zero proposal. | Fabricated quote, Unicode offsets, missing kind anchor, wrong digest/schema and malformed JSON NOT_RUN as R094 native cases. Existing snapshot validator has earlier independent source evidence, but this return does not transfer that into complete M2 acceptance. |
| M2A05 stale/cancel | Native route edited a source message during loopback call; no proposal and STALE_SOURCE. Existing coordinator accepted cancellation during request; no proposal and CANCELLED. | Appended-message race and failed publication transaction NOT_RUN. Local DB comparison does not prove upstream source completeness. |
| M2A06 bypass/defaults | Trigger/test-run and mutation routes rejected intervention fixture job; `AnalyzeChatOnce` redirect refused without retry, and ordinary `AnalyzeChat` smoke remained PASS. | Scheduler/after-sync/agent/MCP source paths only covered by common Analyzer guards in source, not each native route. |
| M2A07 tenant queue | Native owner and member read, malformed member permission denial, limit/cursor response exercised. | Foreign-tenant source/detail 404 and multi-page count/order consistency NOT_RUN. |
| M2A08 failure truth | Native invalid ID and ambiguous transport persisted safe failure codes, no proposal, and unknown attempt. | Injected DB write/settlement failure and response size/truncation variants NOT_RUN. |
| M2A09 fixture isolation | One worker-owned MySQL container on 127.0.0.1:50999, cached mysql:8.0, no pull; removed container and its identified anonymous volume. Existing ccma-nginx/app/db IDs unchanged. | Root independent fixture/evidence check pending. |

Fixture ownership: container `52f6dad9aa269b530950300db0c5fb96f02d90c42a7c92556df181d41d154680`, name `ccmai-r094-worker-20261014`, labels `ccmai.owner=m2-worker` and `ccmai.tranche=R094`. Docker volume event identified anonymous volume `2de090f10ba906d784a6ba113ba077d5d1a028d0e071a5ba41356dd5a0e28c6c` created at 2026-10-14T00:38:47Z; it was unreferenced after the container was stopped/removed and then removed explicitly. Postcheck found neither target; existing container IDs `0770fa03aee1` (nginx), `07822f249ff0` (app), `ee1fc7f68263` (DB) remained running. No separate named network or worker volume remains.

## Review request and claim boundary

Please review the source and the untested matrix rows together as one consolidated R2 finding set. Do not mark M2A03/M2A04/M2A07/M2A08 or full M2 accepted solely from the worker PASS. In particular, confirm lock order and source-currentness limits, case/proposal revision semantics, cap immutability, settled usage knownness, and safe finalization on DB errors. The worker Go budget is exhausted; any same-scope correction needs Root's formal review route and remaining reviewer budget, with no silent additional worker Go invocation. R093 provider cap 2/2 remains exhausted. No real provider call/request-response exists in R094, so no runtime AI governance claim is made. M2 live, M3 browser/human workflow, business/pilot, hosted CI and deploy remain OPEN.
