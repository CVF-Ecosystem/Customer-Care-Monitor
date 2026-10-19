# R132 — WAIT_DATA request-only user workflow

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: Luna Max frontend followup; Sol Medium owned synthetic fixture
Reviewer: Root independent source/browser reviewer
Commit mode: WORKER_MUST_NOT_COMMIT
providerExecutionAuthority: NONE
dispatchKind: INITIAL

SPEC: docs/specs/M3_WAITDATA_REQUEST_UI_R132_SPEC_2026-10-19.md. Source/authority seed base: fc4cc961b58c011f305d22045e2ccd2fef2c7865. Root accepts the intended SPEC and owns review/integration/commits. Owner authorizes continued roadmap, Root orchestrator/reviewer and Luna Max/Sol Medium workers. R131 accepted evidence reconciliation chooses this direct M3-05 product gap; it does not grant provider effects.

## Worker ownership and finite route

Luna owns only seven frontend files in the seed: API/types, new InterventionDataFollowup.vue, existing case view, VN/EN catalogs, new intervention-data-followup.test.ts and existing intervention-ui.test.ts. Return docs/reviews/R132_FRONTEND_WORKER_RETURN_2026-10-19.md. Two cached Node checks maximum; record exact commands/exits and remaining caps. Follow F01..F06; no backend code or runtime.

Sol owns only backend/api/handlers/intervention_followup_browser_test.go. Return docs/reviews/R132_FIXTURE_WORKER_RETURN_2026-10-19.md. Follow F07/F08 and existing owned fixture/helper patterns. At most one GOPROXY=off/GOSUMDB=off/GOTOOLCHAIN=local compile-only Go invocation; no DB/container/browser/server start before Root source admission. The fixture code is a direct proof dependency of the new user flow, not generic tooling. Root creates/owns the bounded synthetic campaign after independent source review.

Each worker may append one pre-BUILD role/rehydration acknowledgment to the active R132 handoff. No other shared-file edits, dispatcher seeds, state/tranche/order/catalog edits, commits, delegation, source snapshots, bulk evidence, provider/secret/installed effects, downloads, push/merge/deploy. Workers must return COMPLETE_PENDING_REVIEW, never self-approve. Real provider calls0. Root independent REVIEW is not done by an implementation worker.

Runtime route: source audit -> cached tooling and exact synthetic ownership readiness -> one actual frontend/backend browser flow -> terminal/durable/provider-zero checks -> owned server/process/MySQL final stop and protected-resource comparison -> bounded independent disposition. If browser unsupported, stop for owner transfer as directed; no workaround. Two consolidated repair rounds; third without a new cause escalates cost. Budgets are seed.executionBudget and are shared across all workers/Root, never reset.

F05/F07 verification refinement: first capture uses an isolated one-shot source-read query fault in the opt-in owned fixture, returning the real handler's sanitized503 without a write; after visible error and explicit reload/manual retry, second capture201 and request201. This replaces the optional unchanged409 probe within the existing2 capture cap, proves the specified safe-error UI branch, and adds no provider, production backend or external effect. Root independently audits the fault scope/removal before runtime.

## Checker Source Read-Ahead Block

Read AGENTS, manifest/policy, bootstrap and canonical active state (parkedOperatorCheckpoint is the state field), memory current truth, active handoff, implementation roadmap/index, R132 immutable seed/record/SPEC/order, R131 matrix/review and shared roadmap-control/shell cleanup learnings. Root R132 doctor PASS27/27 with known pin warning. Knowledge ingestion completed; gate unit89/89 unchanged. Rehydrate and declare each role, acknowledge active handoff before source writes. Legacy/profile/catalog gates apply; APPL not-applicable on closed packets is not a checked active contract. Do not edit installed profile modules/lock/Core.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R132 immutable seed, current roadmap and exact owner request | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | order and synchronized continuity/catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R132_WAITDATA_REQUEST_UI_2026-10-19.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | frontend/fixture source and bounded UI/native/browser proof | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | original outcomes and current applicability limitations | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | pre-BUILD rehydration/role acknowledgment | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | frontend/fixture and exact worker returns | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent source/native/browser/terminal review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed source, actual receipts, scope, continuity and catalog | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | committed source and bounded request-only outcome | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | state/memory/handoff/status/roadmap/catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | local committed R132 range and pinned static gates | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |


## Public export

Disposition: LOCAL_ONLY. No new push/merge/deploy or provider authority. Actual UI software receipts never imply live channel, AI quality, hosted or full M3 acceptance.
