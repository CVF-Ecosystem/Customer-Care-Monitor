# CCMAI-RUNTIME-095 — real synthetic M2 application evidence

Status: ACTIVE
Risk: R2
Worker: /root/m2_worker
Reviewer: Codex /root
Commit mode: WORKER_MUST_NOT_COMMIT
providerExecutionAuthority: OWNER_APPROVED_MAX2_SOURCE_SIGNATURE_REQUIRED
dispatchKind: INITIAL

## Product outcome and authority

Accepted SPEC: docs/specs/M2_INTERVENTION_LIVE_R095_SPEC_2026-10-14.md. Immutable dispatcher seed at5f8170dcab6bc8fec0a8e06dee59c0de71d0ed62, CVF_SESSION/authority/CCMAI-RUNTIME-095.json; exact budgets/paths/effects authoritative. Owner approved 2 new calls on2026-10-14. R093 exhausted unchanged; R094 source frozen unchanged. Sol builds narrow harness only; Root reviews and signs exact source/binary before first credential read/real effect. Worker must not commit/edit seed/self-review. Actual rejected output remains rejected, no tuning/retry.

## Scope and finite execution

Only seed allowed paths plus canonical session/status/review/catalog. Worker5/root3 Go or direct compiled-test invocations total8; compile offline GOPROXY=off/GOSUMDB=off. One new labelled cached mysql:8.0 --pull never fixture shared, no recreate/DB reset after live. Exactly worker1/reviewer1 requests global2 cap,758 requested completion tokens,owner768 ceiling,6000 requestbytes,1MiB response,60seconds/call,300seconds campaign; model/region/endpoint hash seed-pinned. Two same-scope repairs within unchanged counters, third stops. No existing app/DB, production, real channel/customer, outbound, dependency download, core/gate/policy, push/deploy. Credentials never argv/env/disk/tracked/logs; memory/stdin only. Offline tests/source signing precede secret read; return exact hashes,commands,exits,failed checks,usage,receipt/ledger and cleanup. No mock AI governance claims.

## Phase and source admission

INTAKE approval; DESIGN shared durable fixture/recorder/markers; SPEC accepted above; WORK_ORDER committed seed before BUILD; BUILD offline harness only pending Root source signature; REVIEW independent Root, formal bounded disposition before FREEZE. Root signs source+binary+frozen dependency hashes only after consolidated source/sanitation/budget review. Live no-call preconditions observed via real application route; physical call request/response receipt is mandatory. API200 does not settle output validity.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
outsideAuthorityBlockers: FULL_M2_M3_PILOT_OUT_OF_SCOPE
nextRepairRoute: SAME_SEED_WITHIN_COUNTERS_NO_LIVE_RETRY
workerRedispatchAllowed: YES_WITHIN_EXISTING_SCOPE_AND_TWO_REPAIR_ROUNDS

## Checker Source Read-Ahead Block

Read canonical continuity,seed/SPEC/order,legacy/profile applicability/role/claim rules and roadmap/repair learnings. Doctor before material; legacy local/range,profile pre-dispatch/catalog and89unit gates before commit. Checks prove repository only. Root controls source signature and minor metadata corrections.

## Public Export Disposition

Disposition: DEFERRED_PRIVATE_ONLY
No push/merge/release/deploy.

## Gate-To-Role Closeability Contract


closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R095 committed seed and order | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | order continuity catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_M2_INTERVENTION_R095_2026-10-14.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | source safety checks and local software receipts | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | return defect and evidence disclosure | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | acknowledgment and admitted metadata | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | application source and actual isolated local return | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent documentary/source review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed docs and gate receipts | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | actual M2 bounded live material SHA and bounded disposition | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | state memory handoff implementation catalog roadmap | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | exact committed range with pins | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |


