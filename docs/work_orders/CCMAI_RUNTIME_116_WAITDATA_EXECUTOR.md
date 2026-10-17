# R116 WAIT_DATA executor implementation and conditional live acceptance

Status: ACTIVE
Risk: R2
dispatchKind: INITIAL
Worker: /root/r116_executor_worker
Reviewer: Codex /root
Commit steward: Codex /root
providerExecutionAuthority: ROOT_ONLY_CONDITIONAL_FRESH_OWNER_NEW2

Implement the entire accepted docs/specs/M3_WAITDATA_EXECUTOR_R115_SPEC_2026-10-17.md, inherited REVIEW_PASS for intended design only. Product outcome: complete current WAIT_DATA revision -> one grant-bound execution/new run/new snapshot -> same-case active proposal for human review, immutable original/history and faithful usage. Full M2/M3/pilot/release remain OPEN. Authority seed is CVF_SESSION/authority/CCMAI-RUNTIME-116.json; exact allowed source/test paths and prohibited effects there are binding. No frontend/dependency/Core/gate edits.

Worker may edit only seed-listed product/test files plus docs/reviews/probes/r116_worker/** and docs/reviews/R116_IMPLEMENTATION_WORKER_RETURN_2026-10-17.md. Root owns scripts/r116_waitdata_fixture.py, r116_root evidence, seed/record/continuity/catalog/order/review/commits and secret access. Begin source-only. Up to6 focused offline Go invocations (GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local); worker must reserve every invocation and preserve command/stdout/stderr/exit/source hashes. Do not run native DB tests (including existing tests) before Root admission: compile -run '^$' and pure focused engine tests only. No downloads. New tests native-only fail closed when explicitly opted in and DSN absent; ordinary unit checks may skip honestly, but skip never native PASS.

Acceptance A1..A8 from SPEC required: strict512byte route/body and auth; replay-before-stale/version/grant state checks after current permissions; transaction-aware local owner+claim/grant/JobRun/snapshot/CAS admission, no standalone ReserveJobRun faux atomicity; nullable unique exec linkage preserving legacy attempts; current source and permission recheck at durable call/debit marker; no locks across network; no refunds/retries/ownerless reclaim; faithful settlement and manual holds on uncertainty; original-owner-only atomic same-case new active proposal+projection+linked immutable history; legacy full-candidate uniqueness before client selection; source and destructive lineage locks in consistent tenant-first sorted parent order. No RunManual guard bypass/history clearing. Ordinary runner stays nil; no public grant issuer or client provider config/secret override. Same body replays after publication without another debit/run/call; changed body conflicts.

Use current code and exact accepted SPEC as contract, not optimistic guesses. Implement discriminating positive/negative/concurrency/fault/legacy/destructive tests using isolated real MySQL only after admission. Pure DB/unit doubles prove software, never AI governance. Root will own real synthetic Alibaba proof after independent source review and committed source/binary/fixture admission, max2 NEW physical calls, stopfirstvalid or any uncertainty, no retries. Root only existing ignored secret access, pinned qwen3.7-flash endpoint, thinkingfalse, request<=6000/completion<=768. Worker has no secret/provider/network authority. Fresh grants bind exact source/config/run/request/revision/actor/deadline; no old budget reuse.

Finite checks: worker6 and Root4 focused Go invocations, worker1/Root1 owned new fixture start+teardown max, source repairs2. Root may use worker's one fixture for a serial worker and independent reviewer campaign without extra start; new fixture image pinned in seed, names ccmai-r116-*, loopback only, protected baseline and original terminal cleanup mandatory. Existing fixtures/app/DB remain untouched. Runtime capability failure stops runtime, no restart/install/alternate tool stack. Stop on admission/source/permission/owner/call/settlement ambiguity, unexpected effects, unprovable protected cleanup. Third repair absent independent root cause -> REVIEW_COST_ESCALATION_REQUIRED.

Before returning, recheck closeability, hashes, source/DB/live distinction and all failed/unrun checks, budget and cleanup receipts. Status COMPLETE_PENDING_REVIEW only if the claimed source scope is coherent; missing native/live acceptance explicitly OPEN. Root independently verifies whole change, migrations/locks/negative discriminators/source and binary bindings, effect counters/rows/history, exact real provider request/response and cleanup before scoped acceptance. FREEZE prohibited while required evidence missing or failing. No push/merge/deploy/installed activation/autosend/account tests. R114 and Dashboard baseline gaps retained.

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
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R116 seed and documentary order | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R116 order continuity catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R116_EXECUTOR_DESIGN_2026-10-17.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | R116 executor focused software tests and exact source identity | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | R116 executor source/DB/live evidence boundaries | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | R116 source-only BUILD acknowledgment | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | R116 implementation return and source/evidence packet | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | R116 independent implementation and evidence review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | R116 changed documents source/binary pins catalog continuity | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | R116 exact source/evidence commit | BOUNDED_PATH_FAMILIES | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | R116 memory state handoff implementation catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | R116 local range profile checks | BOUNDED_PATH_FAMILIES | commit-steward | CONTINUITY_COMMIT | continuity |

Material-SHA marker:b06209f28c36afc954f754531db6c363057d5cf7. Worker source-only BUILD; DB/live require Root committed admission.
