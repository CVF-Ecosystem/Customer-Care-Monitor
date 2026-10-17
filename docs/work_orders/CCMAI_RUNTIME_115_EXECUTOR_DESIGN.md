# R115 actual WAIT_DATA executor/integration design

Status: COMPLETE_PENDING_REVIEW
Risk: R1
dispatchKind: INITIAL
Worker: /root/r115_executor_spec
Reviewer: Codex /root
Commit steward: Codex /root
providerExecutionAuthority: NONE

This is a documentary BUILD, not implementation of an executor. Source inspect current backend only. Produce exactly docs/specs/M3_WAITDATA_EXECUTOR_R115_SPEC_2026-10-17.md, docs/reviews/planning/R115_EXECUTOR_IMPLEMENTATION_WORK_ORDER_DRAFT_2026-10-17.md, docs/reviews/probes/r115_worker/source_identity.json and docs/reviews/R115_DESIGN_WORKER_RETURN_2026-10-17.md. Root owns continuity/authority/catalog/review/commit. Source source-of-truth68a3d30b remains unchanged; pin current HEAD separately from implementation identity.

Requirements: trace real R102 revision/request/read/replay/current permission/locks, RunManual ownership guard and source binding, JobRun reservation/recovery/checkpoint, provider attempt/budget/usage/publication, supported actual source writers and their lock orders. Specify one coherent minimal manual executor path, same-case lineage and immutable original history. Do not bypass guards by clearing owner/history or calling RunManual blindly. Define actor/permission revocation, tenant/source/job/snapshot/revision/request identity, idempotency/concurrent execution, execution-authority binding separate from intent, changed-source publication conflict and what survives failure. Never hold a DB transaction across a provider call. No fabricated exactly-once external-call guarantee; uncertain provider/DB crash window must fail safe without automatic replay.

Resolve concrete data/state/API/transaction boundaries and source-writer integration scope; partition non-goals honestly rather than claiming generic sync completeness. Every factual source claim needs path/function/verified line range and physicalSHA/Gitblob identity in JSON. Separate intended design, source-inspected current facts, inherited exact-revision tests and runtimeNOT_RUN. Give discriminating positive/negative/concurrency/fault/recovery cases with observable rows/counters/history/usage outcomes; real provider calls under separately granted authority are required for actual AI/governance proof. Pure unit/DB evidence must not masquerade as that proof.

Future implementation order must remain one unfenced Status: DRAFT line, NOT_DISPATCHABLE, no installed authority/budget/secret implied. Exact candidate paths, dependency edges, bounded coherent same-scope repair and finite failure conditions; budget/grant numbers are proposals only. List absent fresh provider authority, source/binary/fixture/capability/protected/terminal receipt readiness explicitly. UI missing offline/raster evidence and Dashboard failures remain open without becoming generic source-design blockers. Future ordinary UI wording must use understandable Vietnamese.

No changes to product/tests/dependencies/Core/gates/old records. Zero product-test, Go/native/Node/browser/Docker/DB/runtime inspection/provider/network/secret effects. Knowledge ingest only through canonical Core to .cvf/runtime/r115_knowledge_worker.json outside exclusive roots. Worker cannot commit/edit seeds/self-review. Up to2 same-scope document repairs; third without new independent root cause stops. Return exact hashes/status/roles, closeability recheck and all unproved assumptions. Root full independent source/citation/design review before commit; no productFREEZE or fullM3 claim.

## Public Export Disposition

Disposition: DEFERRED_PRIVATE_ONLY
Local intended design only, no publication or runtime authority.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R115 seed and documentary order | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R115 order continuity catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R115_EXECUTOR_DESIGN_2026-10-17.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | R115 source identity and factual acceptance matrix | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | R115 intended versus current facts evidence limits | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | R115 source-only worker acknowledgment | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | R115 design return and documentary packet | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | R115 independent source and intended design review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | R115 changed documents source pins catalog continuity | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | R115 exact documentary commit | BOUNDED_PATH_FAMILIES | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | R115 memory state handoff implementation catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | R115 local range profile checks | BOUNDED_PATH_FAMILIES | commit-steward | CONTINUITY_COMMIT | continuity |

Material-SHA marker:3fb8be01ea24ba1793f6b6da2a4505f1975e1405. Product implementation/runtime NOT_DISPATCHED.
