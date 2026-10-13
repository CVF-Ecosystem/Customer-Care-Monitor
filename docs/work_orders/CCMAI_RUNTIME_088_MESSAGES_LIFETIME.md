# CCMAI-RUNTIME-088 — Messages selection lifetime

Memory class: FULL_RECORD
docType: work_order
Status: REVIEWER_ACCEPTED
Risk: R2
Worker: ccmai-r088-worker
Reviewer: Codex /root
Owner: Local orchestrator / independent reviewer
providerExecutionAuthority: FORBIDDEN

## Dispatch Prompt Envelope

- Role: IMPLEMENTATION_WORKER, actor ccmai-r088-worker, shared-workspace INTERNAL_AGENT. Operator prefers manual Sonnet; record exposed model/effort honestly, unavailable telemetry null with reason. Local owns independent review, commits and publication.
- Canonical packet: docs/work_orders/CCMAI_RUNTIME_088_MESSAGES_LIFETIME.md. Commit mode: WORKER_MUST_NOT_COMMIT. No worker push, merge, deploy or self-review.
- executionBaseHead: WORKER_MUST_CAPTURE_AT_REPAIR_START; HEAD remains 086330e1860df291ea58a56726914bb0b4d53087 with uncommitted prior candidate. Preserve it. Immutable authority/baseCommit: 45ac7a91a60f9f64239492e0db838c809ca51fca. Verify seed equals its first committed content; capture actual current HEAD before edits, which is later than this seed commit.
- Current-time notes: F06 only; Node/npm available, locked frontend/node_modules already installed; no second dependency setup. One locked public dependency setup is admitted; Docker/API keys/provider/backend/DB/Go/SMTP/customer data are not. Current status is REVIEW/CHANGES_REQUIRED after round1; one same-scope repair is admitted, worker must acknowledge BUILD/ACTIVE before edits.
- Do-not-misread notes: independent UI implementation from a behavior contract, no private CVF source copy. GOV002 repository controls are accepted; product/native/live/FREEZE gaps are still open. An id comparison alone does not protect A→B→A.
- First actions: rehydrate AGENTS/manifest/policy/bootstrap/memory/active handoff/implementation/index and the two shared learning owners; run independent Core doctor, verify source hashes and seed, read SPEC and named source/test scopes. Before source edits or dependency setup, acknowledge actual actor/HEAD in the active handoff, switch state/memory/handoff/bootstrap/implementation/record/order consistently to BUILD/ACTIVE and run preflight plus the explicit-packet pre-dispatch gate. Stop dependent work if either fails.
- Return: preserve round0 evidence and author a separate round1 return; COMPLETE_PENDING_REVIEW only with all mandatory evidence and a CLOSEABLE recheck; otherwise one consolidated BLOCKED_WITH_REASON with exact prerequisite/owner. Leave all source/evidence uncommitted for Local. On return switch every surface to REVIEW/REVIEW_PENDING, display COMPLETE_PENDING_REVIEW and clear activeWorkOrders.

## Purpose, Authority And Finite Route

Implement [R088 SPEC](../reviews/planning/R088_MESSAGES_LIFETIME_SPEC_2026-10-12.md), UI01-UI08, fixing F06 from the [seven-finding inheritance roadmap](../reviews/planning/CCMA_RUNTIME_CVF_INHERITANCE_ROADMAP_2026-10-12.md). User outcome: Messages shows the currently selected tenant/conversation; late requests do not replace its messages, evaluation, loading/errors or resources.

Owner says continue after GOV002 acceptance while retaining manual worker / Local review. Authority: project AGENTS/manifest/policy, immutable CVF_SESSION/authority/CCMAI-RUNTIME-088.json at 45ac7a91a60f9f64239492e0db838c809ca51fca. Source baseline db4eb5cf20de43320ac435e7ec4d9389cb41f303, hashes/read scope/prerequisites in docs/reviews/probes/r088_dispatch_source_audit_2026-10-12.json. Public Core e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba read-only; no private code/evidence export.

INTAKE/DESIGN/SPEC inherit the bounded review and completed specification; this package enters WORK_ORDER. Package B is selected first because its UI proof is independent of failed/unknown native DB prerequisites. It supports S1/S5 correctness, not full job admission/human workflow/release acceptance. Auth/outbound findings stay OPEN, including P1 concerns; no automatic successor. Stop after one implementation package and independent Local review.

## Required Source-To-Acceptance Reads

| Source / bounded scope | Acceptance derived |
| --- | --- |
| SPEC UI01-UI08, complete | User-visible identity, pending-data clearing, ABA, stale success/error/finally, reset/tenant/unmount/remount, list/map/initialization and image cleanup contract |
| Messages.vue selection/list/map functions; template header/count/download/loading consumers; mobile back, watchers, onMounted deep-link awaits, onUnmounted/debounce/image code | Guard every relevant publish/continuation boundary, including the period before the new response; healthy tabs/deep links/export/date behavior preserved |
| stores/conversations.ts, complete; Messages.vue calls are its only current consumers | Shared-store message/conversation and list/total writes need their own generation protection; do not hide a race with component-only checks |
| messages-screen.spec.ts / messages-business-day.spec.ts existing mounted harness and assertions; package.json/lock/vite test settings | Real component/store with mocked API; retain existing regression assertions; no dependency/config changes |
| feedback_orchestrator_roadmap_control.md controls and feedback_cvf_repair_workflow.md evidence contract | Finite outcome, phase acknowledgment, semantic detector credit, genuine skips/failures and consolidated repair |
| Public pinned closeability classifier, continuity activeWorkOrders and role/claim/routing readers; legacy validate_record/check_authority/check_order_status | Exact status/phase/history/authority and return consistency; N/A is not active-contract proof |

No full corpus read is required or claimed. Capture additional decision-relevant reads in the return, rather than asserting all files were read.

## Allowed Scope And Role Ownership

Worker source/test write scope is exactly the seed's frontend paths: Messages.vue, conversations.ts, optional conversationRequestScope*.ts helpers, new messages-lifetime*.spec.ts / conversations-lifetime*.spec.ts and bounded conversation-lifetime fixtures, plus the two existing Messages test files for additive/documented harness corrections. Do not weaken/delete existing assertions or edit unrelated UI/API/store code. No broad component/style refactor.

Worker owns bounded acknowledgment/BUILD/return synchronization in current continuity, active R088 tranche/order and new R088 evidence under docs/reviews/probes/r088_messages_lifetime and docs/reviews/R088_MESSAGES_LIFETIME_WORKER_RETURN_2026-10-12.md. Existing historical records, seeds, GOV002 worker evidence and old acceptance are protected. Old GOV002 handoff status-only supersession and catalog/roadmap planning updates belong to Local before dispatch; worker does not rewrite them. Worker may append a source-backed new shared learning record only within its current defect scope; do not promote Core changes.

Local owns independent review, evidence-determined metadata fixes, catalog/roadmap acceptance status, buildCommit/closure and project branch publication. Worker may repair same-scope findings autonomously; source/evidence ownership remains independent at review. A new source/effect/authority boundary requires Local routing, not worker scope widening.

## Work-Order Fulfillment Manifest And Evidence

| Deliverable | Mandatory positive/negative evidence |
| --- | --- |
| UI01-UI03 detail selection | Settled A→pending B has no stale header/count/data/download; reversed A/B messages and evaluation; old A/new A differ; late success/error/finally cannot clear a current spinner or change error/data; current failure and healthy recovery control |
| UI04 shared mutation boundary | Direct real-store overlapping detail and list calls, plus mounted real-store component. An old lifetime cannot win or invalidate newer work; tests may not stub the functions under judgment |
| UI05 reset/lifetime | Mobile back/deselect, tenant t1→t2 same-id, unmount/remount using one Pinia. Late responses/errors leave old data cleared and new lifetime intact |
| UI06-UI07 list/map/startup | Reversed page/filter/search list responses and stale list catch/finally; tenant-bound evaluation map; old initialization/deep-link continuation after unmount/tenant switch cannot auto-select; healthy deep-link and tab controls |
| UI08 resources | Synthetic pending image fetch after reset/unmount cannot retain leaked blob URL or stale cache; active resource survives old-owner cleanup; debounce cannot publish after teardown |
| Source sensitivity | Before production edit, new tests fail on isolated unchanged source for named real behaviors. Record the failing assertion and healthy control. Missing exports/modules, build/setup failure or timeout earns no detector credit |
| Final candidate | All focused lifetime tests plus the two existing Messages suites PASS, no mandatory skip; forced vue-tsc and build PASS once. Evidence binds source/test LF hashes, actual execution HEAD, exact commands/results/test identities, optional mutation/restoration and unchanged protected paths |
| Continuity / return | BUILD acknowledgment before edits; seed unchanged; return consistent REVIEW_PENDING/COMPLETE_PENDING_REVIEW and no executable orders, independent identities, honest model/effort/token/bill, CLOSEABLE recheck |

## Effects, Budgets And Stop Conditions

One public locked dependency setup: npm --prefix frontend ci --ignore-scripts, maximum 10 minutes, only registry.npmjs.org packages already in package-lock.json. Keep package/lock/config byte-identical; do not read credentials/.npmrc/.env/local tokens or use private registries. If required binaries/tools remain unavailable, report one consolidated blocker instead of changing the toolchain or starting another setup campaign.

One focused UI regression campaign for the final candidate, then forced type-check/build. Suggested commands from project root:

```powershell
npm --prefix frontend ci --ignore-scripts
npm --prefix frontend test -- <actual-new-lifetime-test-paths> src/__tests__/messages-screen.spec.ts src/__tests__/messages-business-day.spec.ts
Push-Location frontend
try { npx --no-install vue-tsc -b --force } finally { Pop-Location }
npm --prefix frontend run build
python -B scripts/cvf_downstream_gate.py preflight
python -B ../.Controlled-Vibe-Framework-CVF/scripts/lib/downstream_governance/cvf_downstream_gate_runner.py run --phase worker-return --project-root . --trusted --packet docs/work_orders/CCMAI_RUNTIME_088_MESSAGES_LIFETIME.md --packet docs/reviews/R088_MESSAGES_LIFETIME_WORKER_RETURN_2026-10-12.md --no-receipt
```

For npm exec type-check, ensure its cwd is frontend (use Push-Location frontend with try/finally Pop-Location or a subprocess cwd); do not let a root-cwd no-op earn PASS. An npm dependency cache is ignored machine setup, not repository evidence. Capture sanitized output and process deadlines; exit0 with missing/mandatory SKIP is not acceptance.

Only synthetic API/fetch/data boundaries. No actual browser/API/customer endpoint, backend/dev server, DB/Docker/Go/SMTP/provider or release campaign. No real-provider governance claim: these tests check UI structure/state behavior only. docs55 remains a separate known product closure debt; do not rerun docs build to rediscover it. No worker commit/push, gate edits, dependency changes or private Core export.

Repeat only when a named failure or actual source/test change requires it. Preserve every failed/partial invocation. Consolidate one repair set, no automatic successor; at repair round three without independently new root cause stop REVIEW_COST_ESCALATION_REQUIRED. Unexpected source drift, missing tools after the admitted setup, gate/authority contradiction or required outside-scope edit returns one blocker packet.

## Review Dispatch Convergence And Invocation Budget Control

Review-Dispatch Convergence Control: REQUIRED

dispatchKind: REWORK
dispatchSurface: INTERNAL_AGENT
parentAssignmentId: CCMAI-RUNTIME-088
reviewRoundCount: 1
priorFindingSetDigest: 3129726740a0c7a9d0569207a4646e90812ee02c8d7d24fb7588213231e2dbe1
dependencyAuditDisposition: COMPLETE_BEFORE_FIRST_REPAIR
reworkFindingDisposition: CONSOLIDATED_ALL_DEPENDENT_FINDINGS
newIndependentCriticalEvidence: NONE
regressionGuardDisposition: REQUIRED_AND_PLANNED_FOR_EACH_TARGETED_DEFECT
cumulativeExternalInvocationCount: 0
externalInvocationCeiling: 0
usageAvailability: NOT_APPLICABLE_INTERNAL_AGENT
quotaAdmissionDisposition: NOT_APPLICABLE_INTERNAL_AGENT
nextDispatchDisposition: ONE_CONSOLIDATED_REWORK
rootCauseClusterId: CCMAI-MESSAGES-REQUEST-LIFETIME
reworkGeneration: 1
consolidatedDefectClassSweep: COMPLETE_BEFORE_REWORK_DISPATCH
successorTrancheOpened: NO
implementationAutonomyDisposition: CONTRACT_AUTHORITY_EVIDENCE_OUTCOME_ONLY
preExecutionReviewAdmission: NOT_REQUIRED_BEFORE_EXECUTION
preExecutionReviewTrigger: NONE
nextRoutineReviewBoundary: WORKER_RETURN_OR_TERMINAL_OUTCOME
reviewerWorkBoundary: EVALUATE_RETURNED_EVIDENCE_NOT_RECREATE_IMPLEMENTATION

Task classification: PURE_LOCAL_IMPLEMENTATION; USES_EXISTING_OWNER; NETWORK_READ limited dependency setup; PUBLIC synthetic data; GIT_REVERSIBLE; BOUNDED_CLUSTER; MULTI_ROLE_NO_COMMIT; KNOWN_PATTERN. Downstream installed phase controls are actual bindings; this does not claim an unavailable full parent router/bundle was invoked.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R088 seed/spec/order and source audit | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | current order/continuity/catalog; installed profile protected | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_MESSAGES_LIFETIME_R088_2026-10-12.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | seed frontend source/test/helper families; synthetic fixtures | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | new R088 defect/evidence disclosure; no downstream Core ADIF registry | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | BUILD phase/roles/current continuity/order under same seed | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | scoped source/test/return and current synchronized metadata | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent returned evidence and evidence-determined metadata | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed candidate and gate receipts | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | scoped F06 decision and actual material SHA | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | current state/memory/handoff/implementation/catalog and roadmap status | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | exact clean material/continuity ranges with actual pin arguments | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |

Graph rows declare ownership; they do not mean twelve private-Core gates are installed here. Actual bindings: independent Core doctor, legacy seven-gate preflight, installed explicit-packet phase runner, focused UI/type/build checks and catalog. Core ADIF N/A because no Core mutation/registry; disclosure is in R088 evidence. A failing installed gate is never waived.

Preparation choreography: new immutable seed is committed before activation, followed by one dispatch material commit and one handoff-only marker sync if needed. These are preparation, not worker BUILD or closure. Later reviewed implementation uses one material plus at most one continuity commit. Worker never commits.

## Worker Return Self-Proof

The new return must include rootCauseClusterId and reworkGeneration from this packet; consolidatedDefectClassSweep COMPLETE_ALL_KNOWN_DEPENDENCIES; productionBindingEvidence naming actual source/test hashes and commands; adversarialRegressionDisposition PASS_TARGETED_DEFECT_CLASS only from meaningful old-source/negative assertions; successorTrancheOpened NO; implementationAutonomyDisposition CONTRACT_AUTHORITY_EVIDENCE_OUTCOME_ONLY; actual internal/external-agent/provider counts and numeric usage or an explicit unavailable reason; terminalReadinessVerdict READY_FOR_REVIEW or a reason-bearing blocked verdict. Record actual model/session effort with scale limits, token/bill null with reasons if unavailable. None of these fields proves provider identity or substitutes for independent review.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
outsideAuthorityBlockers: NONE
nextRepairRoute: ONE_CONSOLIDATED_SAME_SCOPE_R088_REPAIR
workerRedispatchAllowed: YES

At dispatch this states the admitted path is technically closeable; it is not evidence that UI tests/dependencies already pass. Worker repeats the recheck at return; external blockers must be named instead of fabricating NONE.

## Checker Source Read-Ahead Block

Dispatcher read the pinned classifier, graph constants/columns/dependencies, continuity active-status list, legacy status/seed base/first-commit rules and catalog. Named source and test reads drive UI01-UI08; no complete corpus/private-source inheritance claim. Pre-dispatch must actually CHECK this DISPATCH_READY packet, not print no-candidate N/A. Reuse existing evidence; reviewer does not reimplement the fix or replay a broad campaign without a named contradiction/information gain/cost reason.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY. Preparation before actual publication; no predicted export. Local may publish only the authorized project branch after final committed gates; record real remote/SHA/artifact evidence. No private Core export, main push, merge, hosted CI or deploy claim.

## Consolidated Round1 Repair Admission (2026-10-13)

Finding R088-R1-F01 and exact digest above are defined in docs/reviews/R088_INDEPENDENT_LOCAL_REVIEW_ROUND1_2026-10-13.md and docs/reviews/probes/r088_local_review_round1/consolidated-findings.json. Preserve original return/proof and Local evidence. Repair pending automatic deep-link versus explicit selection/deselect intent across awaited page success/rejection/other-page reload boundaries, using existing Messages.vue/lifetime tests only. The two Local behavioral failures are mandatory regressions; retain healthy current/other-page deep links and all original regressions. No new dependency install or optional mutation campaign required.

reviewerLocalRepairBoundary: WORK_ORDER_FORBIDS_REVIEWER_REPAIR
reviewerLocalRepairBasis: Owner retains manual worker and independent Local review; R088-R1-F01 requires production source changes and fresh mounted regression evidence. Local authored proof/findings and continuity only, leaving implementation to the same worker.

One same-scope repair, no successor/seed expansion. Use a new round1 evidence subfolder and return artifact so round0 remains immutable. BUILD acknowledgment before edits; transition CHANGES_REQUIRED -> BUILD/ACTIVE consistently, run preflight and explicit active-packet pre-dispatch gate actually CHECKED, then repair. Return REVIEW_PENDING/COMPLETE_PENDING_REVIEW with activeWorkOrders empty. Local decides final acceptance/commit/publication. Third-round escalation rule retained. Global channels and audit vulnerabilities remain separate disclosed debt.

## Reviewer-Local Repair Assessment

unchangedObjective: YES
unchangedDesign: YES
unchangedPaths: YES
unchangedAuthority: YES
unchangedEffects: YES
unchangedCommitOwner: YES
evidenceDetermined: NO

The defect evidence is determined; evidence for a correct repaired implementation is not yet available. New mounted regressions and repaired candidate type/build proof are required, and owner retains manual worker implementation with independent Local review. This is not evidence-determined metadata repair.

## Final independent acceptance

Local final review docs/reviews/R088_FINAL_INDEPENDENT_LOCAL_REVIEW_2026-10-13.md accepts F06 bounded synthetic UI at actual material a8865d15296ce1a82ae368363d8f7de29392cdd0. No further worker repair; operator changed future role route to autonomous Local + Luna max for new separately bounded A/C packages. Historical dispatch/rework instructions above are not an active order. Product/native/live/accounts/deploy remain open or parked.
