# R137 — execution read APIs and UI source

Status: CLOSED
Risk: R2
Worker: Sol Medium delegated implementation worker
Reviewer: Root independent REVIEWER
Commit mode: WORKER_MUST_NOT_COMMIT
providerExecutionAuthority: NONE
dispatchKind: INITIAL

## Authority, outcome and bounded route

Immutable seed: CVF_SESSION/authority/CCMAI-RUNTIME-137.json at 7d468de3 (resolve full SHA from Git); inherited intended contract: docs/specs/M3_EXECUTION_UI_R136_SPEC_2026-10-20.md. Owner continues as Root ORCHESTRATOR/REVIEWER and permits Sol Medium assistance. This new R2 authority covers source and local software checks only. Source/security work receives independent Root review before acceptance. Provider/credential/grant/execute and runtime effects require separate applicable authority; none is inferred here.

Roadmap M3-05 needs request-linked execution history and one deliberate writer action. One integrated implementation, consolidated independent source/component review and up to two same-scope repair rounds are the finite route. No new design/helper chain. E01..E05 source behavior can be reviewed; E06 real-router/browser/actual execution evidence stays NOT_RUN/OPEN. Full M3, M1 business priority, pilot/release and parked Facebook/Zalo OA remain open.

## Worker ownership and acknowledgment

Worker edits only authorized backend/frontend source/test paths plus docs/reviews/R137_SOURCE_WORKER_RETURN_2026-10-21.md, compact JSON evidence under docs/reviews/probes/r137_execution_read_ui_20261020/, and appends pre-BUILD acknowledgment to the active R137 handoff. Root owns shared state/memory/tranche/order/catalog and all commits. Worker must not edit seeds, commit, use network/credentials, run DB/container/server/browser/provider or execute routes, or delegate. Rehydrate current canonical continuity and declare role before BUILD; inherit Root observed doctor27/27 and public Core identity. Read R136 SPEC/review, actual executor/startup/grant/models/router/API/component lifetime and roadmap/repair learning before source.

## Source acceptance S01..S06

S01: Add writer-only execution-readiness GET with exact existing route permission stack and read-only tenant/actor/request/source/grant/budget checks. READY/nonready/error DTO follows R136; no handle except READY matching actor, no mutation/lock/reservation/secret output. Unknown reads fail 503 and no-store applies to success/errors. Do not call writer helpers or alter executor/startup/issuer/schema. Mirror source/current/latest revision/job/channel/config/grant predicates conservatively; do not confuse source absence with DB outage or hide uncertainty as a known mismatch.

S02: Add reader execution-status GET independent of configured transport. Scoped case/request ownership, safe DTO, no-row observation versus unknown read, original request immutable; terminal PUBLISHED projection must verify request/execution/run/proposal/EXECUTE_RERUN event lineage and historical published version. No current mutable version substitution. Any inconsistent lineage is unavailable/hold, not false success.

S03: Add typed bounded GETs and strict existing two-field one-shot POST to frontend API, no shared Axios refresh replay. Render reader-safe status/history on detail after WAITING_ON_DATA -> OPEN; request list validation must permit historical snapshot identity after publication. Actor/session/tenant/case/request/token/role/permission lifetime invalidation prevents stale readiness/results. No raw enums, grant IDs, config/provider/evidence diagnostics in ordinary VN/EN UI.

S04: Writer deliberately refreshes current detail/readiness and sees a consequence/confirmation before a single action. Persist non-secret origin-scoped IN_FLIGHT marker before POST keyed to tenant/case/request/actor; failed storage means no send. Never auto retry/resend after 401/409/5xx/timeout/abort/reload/session switch, never delete hold after logout or no-row GET. Bounded read observation max12status reads, minimum5seconds apart, maximum1minute, cancellation/unmount/generation protection. Uncertainty is read-only pending operator reconciliation; no grant/body/token persistence.

S05: Meaningful selected pure Go tests may cover safe status/projection predicates and serializers without DB/provider mocks. UI mocks cover structure/RBAC/lifetime only, including reader/no-jobs writer, pre-send storage failure, one-shot POST, uncertainty across remount/session, no-row hold, stale responses and published OPEN mount. Do not claim mock governance proof. Add DB-integration test definitions only if clearly NOT_RUN and no auto invocation. Compile/typecheck existing packages with offline/cache-only tools; preserve all failures/skips honestly. Do not run broad historical integration suites or existing MySQL tests under this zero-DB cap.

S06: Return exact changed paths/source hashes, commands/exits/counters, failures/skips, source/component versus real-router/browser/live limits, protected executor/issuer/schema unchanged, unresolved defects and closeability recheck. COMPLETE_PENDING_REVIEW is a worker handback, never self-review/FREEZE. Root independent review owns disposition and any follow-up effect plan. No actual execution or CVF runtime/governance claim without fresh authorized real-provider request/response.

## Budgets and failure conditions

Seed caps: workerGo6/reviewerGo4 compile or DB-free selected pure tests; workerNode6/reviewerNode4 focused component test invocations; workerBuild2/reviewerBuild2 frontend typecheck/build invocations. Use only cached dependencies with GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local; npm --offline/installed tools, no download. Run from project root using go -C backend and npm --prefix frontend. Static repository gates as required. Record actual invocation count including failures; no cap resets. No DB/browser/fixture/server/provider/credential/grant/execute effects. Stop before an uncovered path/effect boundary, missing dependencies, source contradiction needing executor change, or repair round three without independent new cause; record defect/scope decision rather than silently expand. Same-scope repairs continue without repeated owner confirmation while authority/caps remain.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R137 immutable seed, current roadmap and exact owner request | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | order and synchronized continuity/catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R137_EXECUTION_READ_UI_2026-10-20.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | bounded source, pure/component tests and acceptance contract | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | original outcomes and current applicability limitations | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | pre-BUILD rehydration/role acknowledgment | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | source/test identity and exact worker return | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent source/component software review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed source, software evidence, scope, continuity and catalog | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | committed source/component evidence and remaining runtime acceptance | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | state/memory/handoff/status/roadmap/catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | local committed R137 range and pinned static gates | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |

## Bounded source disposition

REVIEW_PASS_SOURCE_COMPONENT_ONLY at 4cce40f2495f571b612122121d377c10499b4b02; Root36frontend/4pureGo/typecheck-buildPASS. API/handlers compile-only; E01..E06 native/browser/execution acceptance OPEN. One consolidated formal repair round, original failures preserved. All DB/browser/provider/credential/grant/execute runtime effects0. No installed activation/publication; fresh integrated effect-proof order and applicable authority required before further effects.
