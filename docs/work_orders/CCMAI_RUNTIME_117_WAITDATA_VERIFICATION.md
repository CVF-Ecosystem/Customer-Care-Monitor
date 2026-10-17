# R117 WAIT_DATA finite readiness repair and product verification

Status: ACTIVE
Risk: R2
dispatchKind: INITIAL
Worker: /root/r117_execution_worker
Reviewer: Codex /root
Commit steward: Codex /root
providerExecutionAuthority: ROOT_ONLY_NEW1_CONDITIONAL

Authority: CVF_SESSION/authority/CCMAI-RUNTIME-117.json, first committed seed980f9563 before BUILD. Inherit accepted R115 SPEC A1..A8, R116 source5d38dd51 and exact source-only review; no DB/native/live PASS inherited. Goal: same-case new WAIT_DATA proposal for human review with faithful real usage and original lineage preserved. R116 start1of1 and old budgets unchanged; old unused physical2 unavailable, new R117 max1 physical call under current owner permission.

Worker source-only task: create scripts/r117_waitdata_fixture.py as narrow fork of R116; old file immutable. New names/path/labels/admissions/effect markers R117; new dedicated NAT bridge Internal=false, explicit gateway_mode_ipv4=nat and loopback host_binding_ipv4=127.0.0.1 plus -p127.0.0.1::3306. No OS/container egress confinement claim. Inspect actual full ports JSON before extracting: empty/null/malformed/non-loopback/wrong multiple mapping HOLD, no index panic/retry. Separate exact ownership/resource/exclusivity validation from ready-port validation; cleanup must stop only exact owned container even if port absent, before/after protected inventory recorded and owned resources preserved. Retain all earlier strict source/committed admission/secret guard/physical marker/child ownership/raw receipts checks. Partial creation uncertainty remains HOLD. Runtime helper must not invoke R116 script/campaign.

Inherited Go tests remain named R116 and require CCMAI_R116 variables/logical DB r116_fixture as compatibility only, explicitly on NEW R117 owned loopback port. Change inherited live harness exact owner grant guard to CCMAI-RUNTIME-117-OWNER-NEW1-20261017; do not falsify adapter provider or weaken one-effect marker. Supervisor uses R117 actual grantId/fresh physical cap1/Root custodian/new dirs but passes exact inherited technical env. New supervisor source/binary pinning must include own script path and all backend Go/go.mod/sum, binaries under .cvf/runtime/r117_execution/bin. Root readback may adapt admitted pure read-only R116 readback into new r117_root path; no old fixture read/write.

Add meaningful offline Python regression tests for full published-port validation and cleanup absent-port owned container, rejection of wrong identity/mount/network/exclusive resources. Native doubles prove supervisor control logic only. Up to3worker Python invocations with raw command/stdout/stderr/exit/hash evidence; worker Go2 reserved for bounded same-scope actual product repairs, no routine formatter or redundant compile. Do not start Docker, DB, binaries, secrets/provider or commit. Root independently reviews entire change, runs meaningful offline check and compiles exact source, commits source/binary admission before one setup. Actual fixture/port/DSN/binaries pinned in committed runtime admission before serial handlers+engine native, noSKIP. Only both native PASS permit committed live admission and one physical Alibaba synthetic request max6000bytes/768completion/thinkingfalse. Root focused Go/binary checks6 total, fixture1start/1teardown, physical1, repairs2. No retries after any uncertainty, no automatic second campaign/tool-repair successor. Third repair absent independent new cause -> REVIEW_COST_ESCALATION_REQUIRED.

Root owns seed, admissions, continuity/record/catalog/review/commit and secret access. Worker may edit only new supervisor/Python test, backend allowlisted paths if contract repair is required, own docs/reviews/probes/r117_worker/** and R117_IMPLEMENTATION_WORKER_RETURN_2026-10-17.md. Root readback/admission/raw evidence separate. No frontend/dependency/gate/Core edits, installs/downloads/daemon restarts/alternate runtime; old fixtures/app/DB remain untouched. FullM2/M3/pilot/release, R114 UI qualifications/R110Dashboard3failures/account park remain OPEN. No mock as governance proof, public/full milestone claim, ordinary installed activation, push/merge/deploy. FREEZE forbidden while required evidence missing/failing.

Finite route: supervisor source repair+independent review -> one owned native/live campaign if admission succeeds -> independent source/effects/DB/wire/usage/history/human-action/cleanup review. Preserve every failure/unrun check and old authority. Before returning, recheck closeability and exact source hashes; COMPLETE_PENDING_REVIEW is source scope only until Root independently observes required runtime.

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
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R117 seed and documentary order | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R117 order continuity catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R117_WAITDATA_VERIFICATION_2026-10-17.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | R117 executor focused software tests and exact source identity | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | R117 executor source/DB/live evidence boundaries | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | R117 source-only BUILD acknowledgment | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | R117 implementation return and source/evidence packet | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | R117 independent implementation and evidence review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | R117 changed documents source/binary pins catalog continuity | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | R117 exact source/evidence commit | BOUNDED_PATH_FAMILIES | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | R117 memory state handoff implementation catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | R117 local range profile checks | BOUNDED_PATH_FAMILIES | commit-steward | CONTINUITY_COMMIT | continuity |

Material-SHA marker:980f9563. Worker source-only BUILD; DB/live require Root committed admission.
