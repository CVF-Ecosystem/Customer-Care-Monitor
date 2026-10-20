# R136 — source-based execution UI design

Status: CLOSED
Risk: R1
Worker: Delegated source/design worker
Reviewer: Root independent source/design reviewer
Commit mode: WORKER_MUST_NOT_COMMIT
providerExecutionAuthority: NONE
dispatchKind: INITIAL

Seed/base6ec8c124b1b5fa32ed6e68a0f7366fb2513ddf3f, documentary D01..D06 acceptance below. Owner continues R135 next INTAKE/DESIGN, no implicit production/security/provider effect.

## Worker ownership and route

Write only docs/specs/M3_EXECUTION_UI_R136_SPEC_2026-10-20.md, docs/reviews/R136_DESIGN_WORKER_RETURN_2026-10-20.md, compact source identity docs/reviews/probes/r136_execution_ui_design_20261020/source_identity.json, and append pre-BUILD acknowledgment to active handoff. Read production source only. No source/metadata/seeds/commits/runtime/browser/DB/Go/Node/provider/credential/network calls/downloads/delegation. Existing doctor prerequisite may be inherited from Root observed27/27. Return COMPLETE_PENDING_REVIEW with current-vs-intended facts, exact source pins, unresolved decisions and closeability recheck. Root independently reviews, records shared continuity and owns commits. Repairs2 consolidated max.

## Documentary acceptance D01..D06

D01: Pin actual current source and separate what exists from proposals. Trace R115 intended design, R116/R117 accepted backend bounds and R134/R135 current UI. main.go DOES bind LoadSyntheticInterventionService to WaitDataExecutor when explicit synthetic startup configured; default disabled. Do not falsely say no process binding. Execute POST exists with strict two-field input and safe replay; no grant issuance/readiness/status API currently assumed without source evidence.

D02: Decide minimal execution UI requirement and API changes. Preserve separate server-provisioned grants, no new public issuance/approval workflow, no user entry of raw grant/configuration/provider IDs. Propose concrete read-only eligibility and execution-status projections for existing actor-bound server grant plus existing execute POST. Define routes, methods, role/tenant/job permissions, safe output DTOs/status/error behavior and no mutation on GET. Existing execution admission remains authoritative and rechecks under locks; readiness never reserves/consumes/guarantees authority. Show client-internal opaque handle only where necessary and authorized.

D03: State stale/expired/consumed/actor or source mismatch, concurrent or terminal execution, process disabled, missing budget/config and permission revoked behavior. Unavailable/unknown readiness fails closed. Define current-request/revision/version binding, server clock/deadline, cache headers, no credential/provider/config/evidence internals in ordinary UI. Explicitly preserve synthetic-only restriction and separate operational provisioning prerequisite; no installed activation or production readiness promise.

D04: User flow uses understandable VN/EN labels; separate saved request from actual execution. Explicit deliberate single-click action after refreshed current readiness and summary, no auto execute, retry or resend on 5xx/timeout/abort/reload. Before every new action authority is fresh. Resolve uncertain outcome via GET/read-only status, preserve exact admitted body identity if needed for diagnostics; never invent success/no-call from client errors or grant absence. Define polling bounds/cancellation/generation/tenant/case/request/session/permission lifetime, duplicate-submit lock and result/history linkage. Missing/reader jobs:w cannot obtain execution control/opaque grant.

D05: Concrete finite next R2 source/evidence scope including exact candidate files and useful tests: structure/RBAC/lifetime only UI mocks, real-router state checks explicitly scoped as software, genuine end-to-end execution proof requires fresh owner-authorized real provider request/response/call cap, never synthetic output as governance proof. Historical R117 physical1/1 consumed, no reuse or stale stored grant. No actual provider/DB call or acceptance claim in R136.

D06: Provide an acceptance table with observable criteria, remaining owner/operational decisions, and finite route to usable synthetic execution (one integrated implementation/evidence order if feasible, no helper-tranche chain). Do not expand into new grant issuance UI/provider settings/recovery/retry/bulk data/pilot work. Root must be able to dispatch concrete code scope after design acceptance; any external-effect boundary remains explicit. Full M3 remains PARTIAL until actual required scope evidenced or explicitly dispositioned by authority.

## Checker Source Read-Ahead Block

Rehydrate AGENTS/manifest/policy/continuity/current handoff/status/index, R136 seed/record/order and R135 delta; inspect R115 SPEC, R117 review, actual main/config/startup/capability/router/waitdata handlers/models/engine and followup frontend/API/lifetime. Read roadmap/readiness learnings. Declare worker and acknowledge before documents. Source citations use current pinned hashes/lines; historical source IDs are evidence pointers, never falsely current. Gate/static/design checks are not AI/CVF runtime governance proof.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R136 immutable seed, current roadmap and exact owner request | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | order and synchronized continuity/catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R136_EXECUTION_UI_DESIGN_2026-10-20.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | design documents, pinned source facts and acceptance contract | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | original outcomes and current applicability limitations | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | pre-BUILD rehydration/role acknowledgment | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | design/spec/source identity and exact worker return | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent source/design contract review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed design, source citations, scope, continuity and catalog | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | committed intended contract and next bounded R2 route | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | state/memory/handoff/status/roadmap/catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | local committed R136 range and pinned static gates | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |





## Public export

Disposition: LOCAL_ONLY. R1 documentary design only; future implementation/security/provider effect requires separate R2 authority and actual evidence. No push/merge/deploy.
