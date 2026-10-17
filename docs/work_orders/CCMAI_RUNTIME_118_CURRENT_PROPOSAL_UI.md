# R118 current proposal UI binding and understandable history

Status: COMPLETE_PENDING_REVIEW
Risk: R2
dispatchKind: INITIAL
Worker: /root/r118_ui_worker
Reviewer: Codex /root
Commit steward: Codex /root
providerExecutionAuthority: NONE

Authority: CVF_SESSION/authority/CCMAI-RUNTIME-118.json. Root commits immutable seed before BUILD; baseCommit must contain seed. Earliest open stage inherited: source UI correction, accepted R117 backend contract. M3 product outcome: display and submit against current proposal after data refresh, with understandable history. Source/component acceptance only; actual rendered screen proof pending.

## Requirements and acceptance

A1. Consume authoritative active_proposal_id plus case run_id/active_execution_id/current snapshot/digest. Never select proposals.at(-1). A valid explicit pointer resolves exactly one proposal with matching current run/snapshot/digest and valid revision. Contradictory duplicate top-level/case pointers, unknown targets, duplicate matches or lineage mismatch disable proposal-bound actions with understandable localized notice, never silently fall back.
A2. Legacy null/omitted pointer: fallback only when active_execution_id absent and exactly one proposal matches full current run/snapshot/digest with valid revision; mirror backend intervention_m3.go146-175. Missing lineage or ambiguity fails closed. No backend permission expansion. ASSIGN_DATA/ESCALATE_DATA remain proposal-less and send expected id empty/revision0. Other actions require selected valid proposal. Preserve request one-shot, reader restriction, route invalidation/conflict semantics.
A3. Label selected proposal current rather than initial. Localize OBSERVE_DATA/REQUEST_RERUN/EXECUTE_RERUN in Vietnamese and English. Translate exact EXECUTE_RERUN system reason approved execution; retain human-entered free text verbatim. No raw enum/technical reason in these ordinary flows. No executor button, secret/grant fields or provider activation.
A4. Discriminating component/presentation regressions: explicit pointer differs from last array entry and submitted id/revision follow pointer; unknown/mismatching/ambiguous pointer disables proposal-bound submit; current unique legacy succeeds, ambiguous/execution legacy fails; proposal-less data actions unchanged; new history labels/exact system reason localized and human free text retained. Existing reader/route/conflict tests preserved. Adjust old fixtures to actual backend fields, not weaken selector to satisfy mocks.
A5. Worker max3 focused frontend invocations total (tests/typecheck, can combine via evidence recorder), Root max2, same-scope repair rounds2. Capture exact commands/raw stdout/stderr/exit and source hashes under own probes. No broad full-suite rerun; R110 historical26PASS/3DashboardFAIL/typecheckPASS remains qualified. No mock-governance claim. No runtime start/provider/secret/DB/browser product campaign, dependency install or backend/tooling changes.

Worker rehydrates AGENTS/progressive continuity, acknowledges committed material SHA in active handoff before BUILD. Worker edits allowlisted frontend/test paths and own return/probes only, never seed/continuity/commits. Root owns record transitions/catalog/review/commit. Return source-only COMPLETE_PENDING_REVIEW, exact checks/limitations/remaining current UI proof and closeability recheck. No FREEZE with required rendered proof missing. No push/merge/deploy.

## Public Export Disposition

Disposition: DEFERRED_PRIVATE_ONLY
Local source/component scope only; not published.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R118 seed and documentary order | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R118 order continuity catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R118_CURRENT_PROPOSAL_UI_2026-10-18.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | R118 UI focused software tests and exact source identity | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | R118 UI source/component evidence boundaries | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | R118 source-only BUILD acknowledgment | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | R118 implementation return and source/evidence packet | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | R118 independent implementation and evidence review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | R118 changed documents source/binary pins catalog continuity | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | R118 exact source/evidence commit | BOUNDED_PATH_FAMILIES | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | R118 memory state handoff implementation catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | R118 local range profile checks | BOUNDED_PATH_FAMILIES | commit-steward | CONTINUITY_COMMIT | continuity |

Material-SHA marker: 40a301fa33e0b9cb364b283008d08b449da39d18.
