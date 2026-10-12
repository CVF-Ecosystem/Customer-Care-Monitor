# CCMAI-GOV-002-C1 portable CVF inheritance

Memory class: FULL_RECORD
docType: work_order
Status: COMPLETE_PENDING_REVIEW
Owner: Local orchestrator / independent reviewer
Risk: R2
providerExecutionAuthority: FORBIDDEN

## Dispatch Prompt Envelope

- Role: REPAIR_WORKER (round1), actor identity `ccmai-gov002-worker`; shared-workspace INTERNAL_AGENT. Operator prefers Sonnet; record actual model/effort if exposed, otherwise null with reason. Local owns independent review and final disposition.
- Canonical packet: `docs/work_orders/CCMAI_GOV_002.md`.
- Commit mode: WORKER_MUST_NOT_COMMIT. Root is commit steward; no worker push, merge or deploy.
- executionBaseHead: read actual HEAD and verify the committed authority seed before BUILD; dispatch authority base is f7bf57d0f4fe0e1920efdcc3609b6a8ea84cb68d.
- Current-time notes: owner explicitly requests inheritance/publication/another-computer continuation and retains manual worker + Local reviewer. No provider/API keys, Docker, SQL, Go, customer data or application execution is admitted.
- Do-not-misread notes: migrate one project, preserve existing gates and evidence, and do not claim runtime AI governance or whole-roadmap completion. The original GOV-002 seed was never activated; C1 is the initial executable packet after source audit discovered a real compatibility boundary.
- Required first actions: rehydrate AGENTS/manifest/policy, compact current state and active handoff; run doctor; read the mapped sources below; acknowledge phase/role/next move in the active handoff before BUILD. Stop on unrelated changes or mismatched seed.
- Return contract: COMPLETE_PENDING_REVIEW only with settled mandatory checks and closeability; otherwise BLOCKED_WITH_REASON with one consolidated blocker report. Return exact changed files, evidence paths, actual invocations, Git status and claim limits. No commit.

## Purpose And Finite Outcome

Make the repository carry the useful, public-safe CVF repository controls, version pins and agent continuation instructions that survive a clone to another computer. Deliver one coherent installation/migration package; Local then reviews and publishes the accepted candidate. This is an operator-requested portability task, not another runtime roadmap milestone. Finish when the pinned profile and existing gates pass together and a separate-path clone can initialize and resume from truthful current project state.

Authority: explicit owner request 2026-10-12; project AGENTS.md, manifest/policy, immutable `CVF_SESSION/authority/CCMAI-GOV-002-C1.json`, and public Core `e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba`. The public Core is read-only. Private parent history, provider memory and local secrets are not export material.

## Required First Reads And Source-To-Acceptance Mapping

All rows mandatory at their stated scope; no full historical-state/corpus read is requested. Sources under `<Core>` resolve to the manifest sibling public Core; all other paths are project-relative. NEW marks required output, not an existing source.

| ID | Exact source and bounded scope | Derived condition and observable acceptance | Evidence / failure condition |
| --- | --- | --- | --- |
| PI01 | `<Core>/docs/reference/downstream_gate_profile/CVF_DOWNSTREAM_GATE_PROFILE_MIGRATION.md`, full; `<Core>/scripts/lib/downstream_governance/cvf_dg_install.py`, `install`/`verify_install`; profile JSON `installedFiles`, `projectPaths`, `continuityContract`, `trustAnchors` | Explicit install from public e9ddcb; byte-pinned profile1.1.0, runner and bundle. Preserve project learning home and existing CI. No mandatory-control override or hand-editing inherited code. | Lock hashes + trusted doctor + install idempotence. Module/lock drift must fail before imported untrusted code. |
| PI02 | `<Core>/scripts/lib/downstream_governance/cvf_dg_continuity.py`, full; current state top-level keys/current facts; implementation status top-level keys/current facts; memory front marker/current owner directive; active handoff Current State | Explicit schema1.0/contract1.0 adoption; synchronize mode/phase/role/handoff/tranche/next move. Preserve every legacy field/value under permitted x- extensions; do not silently discard history. Add required implementation activeWorkOrders. | Before/after structural reconciliation and hashes; one surface drift must fail. Do not promote source-only acceptance to runtime proof. |
| PI03 | `scripts/cvf_downstream_gate.py`, STATUS_PHASE/TRANSITIONS, validate_record, check_order_status, check_authority; `<Core>/governance/compat/check_gate_to_role_closeability.py`, status grammar and check_work_order | Keep legacy record statuses. New profile-bound records opt in explicitly to strict work-order status mapping below; old records still use their existing comparison. No status hidden inside a fence, duplicate declaration, or renamed failure to evade a check. | Existing46 tests untouched/pass plus focused new positive/negative fixtures. Unknown, contradictory or unpinned opt-in fails. |
| PI04 | `<Core>/scripts/lib/downstream_governance/cvf_downstream_gate_runner.py`, run_phase/parser; `cvf_dg_review.py` and `cvf_dg_routing.py`, declarations checked; project `.github/workflows/governance.yml`, full | Existing all-PR gates stay in place. Add generated `.github/workflows/cvf-downstream-gates.yml` with three workflow digest pins and explicit PR/push range, read-only permissions. Preserve worker/reviewer independence and reviewer-local repair routing. | Local pre-dispatch/worker-return/reviewer-fast/pre-commit/pr-ci receipts; resolved range with mandatory controls individually visible, no blanket PASS for N/A. |
| PI05 | `scripts/initialize_cvf_clone.ps1`, full; manifest portable relative paths; `<Core>/scripts/check_cvf_workspace_agent_enforcement.ps1`, freshness/pin/profile checks | Align project pin/header with public e9ddcb; same initializer must work in a disposable fresh clone under a different absolute directory. Binding stays ignored, no original-machine path in tracked instructions. | Clone/init/doctor/preflight logs with actual Git source commit/tree and exit codes. Refuse dirty/unrelated Core, no reset or downgrade. |
| PI06 | `docs/reviews/learnings/README.md`, full; `feedback_orchestrator_roadmap_control.md`, full; `feedback_cvf_repair_workflow.md`, before-BUILD/repair requirements; current roadmap S0-S7 checkpoint/P0-P4 only | Preserve shared learning records; materialize the missing standard learning template. Human continuation guide and AGENTS route each phase to the actual commands and canonical current facts. Keep accounts parked, docs55 and R086 native/fence gaps OPEN. | Readable quickstart + current handoff/bootstrap model, exact acceptance matrix. No claim that writing instructions proves agent compliance or product progress. |
| PI07 | Authority `historicalHandoffMigration.paths`; only each file's Status declaration and Git blob identity; catalog manager and artifact registry current pointer | Change only listed obsolete `Status: ACTIVE` declarations to `Status: SUPERSEDED`; retain paths and all other bytes. New handoff alone is ACTIVE. Regenerate views from registry. | Aggregate before/after blob/digest ledger and byte comparison after normalization of this one declared metadata replacement. No broad handoff rewrite or file moves. |

Source fidelity: profile/contract identifiers and actual CLI commands verified from public source; current legacy status mismatch verified before dispatch; no conflicting meaning is delegated to worker. Existing installer/lock/CI profile paths are new in this project, not new Core implementations. Public migration contract exists; the private parent profile standard is not present in this public checkout and is not required as a missing dependency. This is a bounded source audit, not a complete repository/corpus scan.

## Design And Compatibility Contract

Do not edit pinned Core modules. Add a narrow prospective opt-in to the project-owned gate:

- Legacy tranche `status`, phase model, history transition checks, review-evidence requirements, immutable authority checks and negative tests retain their meanings.
- A new record opts in using `workOrderStatusGrammar: cvf.workOrderStatusGrammar@1.0.0` and `workOrderStatus`. Both must be present together; malformed/unknown/partial opt-in fails.
- Validate opt-in against an installed profile lock identifying `cvf.downstreamGateProfile@1.1.0`; no undeclared/unpinned grammar choice. Identity remains independently checked by the profile/doctor; do not treat typed strings as source authentication.
- Allowed canonical mapping: WORK_ORDER -> DRAFT; DISPATCH_READY -> DISPATCH_READY; BUILD -> ACTIVE; BUILD_BLOCKED -> BLOCKED_WITH_REASON; REVIEW_PENDING -> COMPLETE_PENDING_REVIEW; CHANGES_REQUIRED -> BLOCKED_WITH_REASON; REVIEW_PASS -> REVIEWER_ACCEPTED; PARKED -> PARKED; FROZEN -> CLOSED.
- The work order carries exactly one unfenced Status line equal to the mapped value. Legacy records without opt-in must still compare against their original status. Parent phase, independence and immutable authority checks must not be relaxed.
- During WORK_ORDER/BUILD, implementation.activeWorkOrders lists the active admitted order; during REVIEW/FREEZE it is empty, with review/pending states still visible in tranche/handoff. A waiting review is never falsely marked executable.
- Implementation order resolves transient dependencies: acknowledge/enter BUILD under existing legacy grammar first; implement/test compatibility; install tooling and then adopt opt-in + continuity together before any profile gate/commit/return. Do not publish a partially migrated state.

Keep original legacy values under x- extensions, preserving JSON type and value, rather than rebuilding their prose. Required standard fields stay at the root. Record relocated field names in evidence so future agents can find project-specific history without a default full read. Shared historical orders/seed/tranche/evidence are unchanged. Historical handoff status-only migration is expressly authorized and does not change their original outcomes.

## Allowed Scope And Ownership

Exact authority is the committed C1 seed; it permits the public profile files, lock, generated additional workflow, bounded legacy gate compatibility plus its NEW focused test file, manifest/header pin, initializer, README/continuation guide, shared learning template/home, current continuity/status/catalog/return evidence, and the enumerated historical handoff status metadata. No backend/frontend/product code, existing test edits, existing workflow weakening, policy changes, Core changes or unrelated cleanup.

Root pre-dispatch/session-sync owns authoring this packet, decision, current handoff, tranche and catalog. Worker owns one bounded implementation package and its allowed-scope synchronization/return, acknowledges each role/phase change, and does not alter dispatcher seeds. Local owns independent REVIEW, any evidence-determined metadata repair, material/continuity commits and project branch publication. R2 self-review is forbidden.

## Work-Order Fulfillment Manifest And Acceptance

| ID | Required output / observable condition | Negative assertion / proof |
| --- | --- | --- |
| PI01 | `scripts/cvf_gates/` installed files and `.cvf/gate-profile.lock.json` from public pinned Core; installer preserves learning records | Tampered copied module and inconsistent workflow/lock pins fail; install rerun changes no tracked bytes |
| PI02 | State adopts `cvf.downstreamContinuityContract@1.0.0`; memory Current Truth; implementation required fields; NEW compact bootstrap model | No missing/contradictory surface, no lost extension values, unknown standard field rejected |
| PI03 | Additive gate compatibility + NEW `scripts/tests/test_cvf_downstream_gate_profile_compatibility.py` | Legacy46 unchanged; all mapped transitions tested; partial/unknown/mismatched/unpinned/fenced/duplicate declarations fail; R2 self-review, widened seed and forbidden secrets remain rejected |
| PI04 | NEW generated pinned workflow; existing governance/backend/frontend/docs workflows untouched | No disabled mandatory control/path-filter escape; unresolved PR base and wrong pins fail. No hosted result invented |
| PI05 | Updated pin, initializer and NEW `docs/reference/CVF_PROJECT_PORTABLE_CONTINUATION.md`; README links guide | Actual distinct-path clean clone/init/doctor passes, ignored bindings/receipts excluded, original-machine absolute path unnecessary; dirty/unrelated Core not overwritten |
| PI06 | Existing learning home preserved, missing LEARNING_RECORD_TEMPLATE installed; AGENTS commands actually resolve | No provider-local memory authority; no copied private parent corpus/history/secrets; old roadmap gaps plainly retained |
| PI07 | Enumerated obsolete handoff status-only replacements; new active handoff unique; regenerated catalog | Historical body/path bytes unchanged after allowed status substitution; registry active pointer matches state; Git diff --check |
| PI08 | NEW `docs/reviews/GOV002_PORTABLE_INHERITANCE_WORKER_RETURN_2026-10-12.md` and bounded probe/receipt evidence | COMPLETE_PENDING_REVIEW only if mandatory local checks settled; no FREEZE, CI success, runtime behavior or deployed claim from static tests |

Bound verification: reuse installer/profile source evidence; use isolated disposable copies for tamper, drift and role/phase negatives, never mutate protected live production source. Run all existing gate self-tests once because the legacy checker changes. Run affected new compatibility tests and public profile phases; repeat only for a named finding/change. No application tests, Docker or provider calls. Fresh-clone test may fetch only the declared public project/Core repositories; use a temporary local Git candidate commit in an isolated copy if worker's no-commit working tree needs a faithful cloneable snapshot, recording tree/manifest binding without claiming a published source SHA. No changed source may be omitted from that test snapshot.

Expected commands (PowerShell from project root; Python -B / PYTHONDONTWRITEBYTECODE=1):

```powershell
python -B ../.Controlled-Vibe-Framework-CVF/scripts/lib/downstream_governance/cvf_downstream_gate_runner.py install --project-root . --core-root ../.Controlled-Vibe-Framework-CVF --core-commit e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba
python -B scripts/cvf_downstream_gate.py preflight
python -B -m unittest discover -s scripts/tests -p 'test_cvf_downstream_gate*.py'
python -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase pre-dispatch --project-root .
python -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase worker-return --project-root .
python -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase reviewer-fast --project-root .
python -B scripts/cvf_gates/cvf_downstream_gate_runner.py run --phase pre-commit --project-root .
# pr-ci also passes explicit --base/--head and all three --expect-*-sha256 pins read from the workflow/lock.
powershell -ExecutionPolicy Bypass -File ../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1 -ProjectPath .
powershell -ExecutionPolicy Bypass -File scripts/manage_cvf_downstream_catalog.ps1 -Write
```

A hook template existing in scripts/cvf_gates is INSTALLED only; do not claim local hook invocation unless a hook was deliberately activated and actually ran. Local hook installation is optional, never overwrites an existing hook. Required explicit pre-commit command and CI remain the portable route. Ignore `.cvf/runtime/`, Python bytecode and machine-local bindings without hiding unauthorized tracked changes.

## Review Dispatch Convergence And Invocation Budget Control

Review-Dispatch Convergence Control: REQUIRED

dispatchKind: REWORK
dispatchSurface: INTERNAL_AGENT
parentAssignmentId: CCMAI-GOV-002-C1
reviewRoundCount: 1
priorFindingSetDigest: 4c062e7c74efefa2d2e9e6cd5811cbfb48b152ee7cd9c99905e6e8826f4c5bdb
dependencyAuditDisposition: COMPLETE_BEFORE_FIRST_REPAIR
reworkFindingDisposition: CONSOLIDATED_ALL_DEPENDENT_FINDINGS
newIndependentCriticalEvidence: NONE
regressionGuardDisposition: REQUIRED_AND_PLANNED_FOR_EACH_TARGETED_DEFECT
cumulativeExternalInvocationCount: 0
externalInvocationCeiling: 0
usageAvailability: NOT_APPLICABLE_INTERNAL_AGENT
quotaAdmissionDisposition: NOT_APPLICABLE_INTERNAL_AGENT
nextDispatchDisposition: ONE_CONSOLIDATED_REWORK
rootCauseClusterId: CCMAI-GOV-002-PORTABLE-ADOPTION
reworkGeneration: 1
consolidatedDefectClassSweep: COMPLETE_BEFORE_REWORK_DISPATCH
successorTrancheOpened: NO
implementationAutonomyDisposition: CONTRACT_AUTHORITY_EVIDENCE_OUTCOME_ONLY
preExecutionReviewAdmission: NOT_REQUIRED_BEFORE_EXECUTION
preExecutionReviewTrigger: NONE
nextRoutineReviewBoundary: WORKER_RETURN
reviewerWorkBoundary: EVALUATE_RETURNED_EVIDENCE_NOT_RECREATE_IMPLEMENTATION
reviewerLocalRepairBoundary: NEW_EVIDENCE_REQUIRED
reviewerLocalRepairBasis: F2 requires candidate-bound fresh-clone, range and negative proof absent from the returned evidence; source F1 remains independent worker ownership. Reviewer corrects session/catalog metadata only.
unchangedObjective: YES
unchangedDesign: YES
unchangedPaths: YES
unchangedAuthority: YES
unchangedEffects: YES
unchangedCommitOwner: YES
evidenceDetermined: NO

One initial worker package; same-authority bounded repairs stay in this packet. One consolidated Local review, no per-row implementation recreation. Third repair round without an independent new root cause stops. Record actual worker model/effort/tokens/cost when available; null with a specific unavailable reason otherwise. Never invent provider telemetry or require a provider call for these static repository checks.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | C1 seed and prospective packet; old seed immutable | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | current packet, state and catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_PORTABLE_CVF_GOV002_2026-10-12.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | scripts/cvf_downstream_gate.py; new compatibility tests; profile copies fixed by reinstall | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | bounded adoption defect/evidence record, no Core ADIF registry exists here | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | current phase/role/authority and installed profile; admitted atomic migration precedes first profile run | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | allowed installation, continuity and return evidence | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent review; evidence-determined metadata corrections only | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | exact reviewed candidate and gate receipts | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | scoped adoption review and real material-SHA evidence; product blockers retained | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | current memory, state, implementation, handoff and catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | clean split-range evidence and continuity pointer | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |

Graph labels are declaration obligations, not proof that twelve private-Core gates run here. Actual local gate bindings are the legacy seven, public profile phase controls, pinned closeability checker and catalog; Core ADIF is NOT_APPLICABLE_WITH_REASON (absent downstream registry, no Core mutation). Record actual command/results and N/A scope separately. No invention of a private-core preflight dependency.

## Worker Autonomy / No-Question Rule And Stop Conditions

Choose ordinary implementation details within this contract; repair allowed-scope defects without asking owner again. Return a consolidated BLOCKED_WITH_REASON only for genuine outside-authority blockers, source/pin drift, dirty/unrelated checkout, unresolved grammar semantics, required missing evidence, authority widening or exceeded repair boundary. Do not redispatch yourself, edit seeds, lower a failed gate, erase prior failures, or turn a known product docs55 blocker into permission for unrelated repairs.

The55 inherited docs deadlinks remain a separate P0 repair and block historical product FREEZE; do not run a duplicate failing docs build merely to reconfirm the already retained receipt. The new adoption package must introduce no broken local links. Existing Docs workflow may report that inherited failure on this branch; never say all CI green. No main push, merge or Pages/product deployment in this work order.

## Checker Source Read-Ahead Block

Local dispatcher read public profile schema/constants, installer, continuity contract, status classifier, graph parser, required gate IDs and exact graph columns before authoring. Legacy check_order_status/STATUS_PHASE and authority comparison were read before the compatibility decision. Worker reads only the mapped decision-relevant functions/files, then uses valid positive/negative artifacts. Structural checks do not prove reading or comprehension.

## Return-Time Closeability Recheck

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
outsideAuthorityBlockers: NONE
nextRepairRoute: CONSOLIDATED_WORKER_REPAIR_WITHIN_EXISTING_AUTHORITY
workerRedispatchAllowed: YES

Existing authority can reach acceptance after the authorized round1 repair; current implementation is CHANGES_REQUIRED, so display Status is BLOCKED_WITH_REASON. Worker is explicitly admitted to rehydrate, acknowledge actual repair BUILD, synchronize record/work-order mapping and start this bounded repair; no new seed or successor. Original procedural failures remain disclosed.

## Claim Boundary And Public Export Disposition

Prepared dispatch is not installed/adopted control coverage. Static/hermetic profile proof does not assert CVF governance over AI behavior; live proof would require separately admitted real provider evidence. Hosted CI success requires actual run URL/SHA/conclusion and is not established here. Product R086 source-only acceptance, docs55, native/fence integration, accounts and deployment gaps remain unchanged.

Public Export Disposition: DEFERRED_PRIVATE_ONLY for the private CVF parent; no private parent export is authorized. This project packet may be published to the owner's already specified project remote/branch. The Local receipt records the real branch commit after publication; it must not label planned delivery as EXPORTED.

## Consolidated Repair Admission - Round1

Local independent review: `docs/reviews/GOV002_PORTABLE_INHERITANCE_INDEPENDENT_REVIEW_2026-10-12.md`. Binding findings: `docs/reviews/probes/gov002_review_round0/consolidated-findings.json`, SHA256 4c062e7c74efefa2d2e9e6cd5811cbfb48b152ee7cd9c99905e6e8826f4c5bdb. Original worker evidence under `gov002_review_round0/original-*` is immutable for this repair. Existing C1 seed unchanged; no new source paths/effects or runtime authority. No commit.

- R1-F1: Repair the three captured parser false accepts and add actual fenced/tilde, malformed/contradictory/duplicate/backtick regressions; healthy mappings and old controls retained. New tests must discriminate the returned adapter from repaired behavior in isolated copies.
- R1-F2: Complete PI01/02/04/05 actual idempotence, tamper/drift/wrong-pin/range and faithful separate-root clone/init/doctor/preflight proof. Bind logs to candidate tree/manifest; local pr-ci uses an explicit legitimate base/head and three pins. For active work-order applicability, add explicit --packet docs/work_orders/CCMAI_GOV_002.md while it has ACTIVE status, so a no-candidate N/A is not misreported as contract coverage. Worker-return routing must distinguish N/A from checks. No live/provider/app effects.
- R1-F3: Make guide directly usable with actual repo/branch, Python/PyYAML prerequisites, correct relative links and progressive reads; put installed phase command routing in AGENTS. Preserve the known docs55/runtime gaps.
- R1-F4: Acknowledge repair BUILD before changes; preserve initial missed acknowledgment, correct actual execution identity/HEAD/time/unavailable telemetry, opt in on current tranche and return synchronized REVIEW_PENDING/COMPLETE_PENDING_REVIEW with closeability and candidate manifest. Do not backdate or erase original receipts.

Local owns current session/catalog correction, not source/test repair. On return provide a new round1 receipt and update canonical worker return while preserving original snapshots. Same-scope repairs remain under C1; third-round stop rule unchanged. Fresh-clone source test may use a local candidate snapshot commit in a disposable repo, never a commit in this worktree. Actual worker model is recorded honestly; do not re-label Gemini evidence as Sonnet.
