# CCMAI-RUNTIME-103 — official browser tooling recovery

Status: ACTIVE
Risk: R2
Worker: /root/r103_tooling_worker (Luna Max)
Reviewer: Codex /root
Commit mode: WORKER_MUST_NOT_COMMIT
providerExecutionAuthority: NONE
dispatchKind: INITIAL

## Authority, phase entry and exact scope

Immutable seed CVF_SESSION/authority/CCMAI-RUNTIME-103.json committed at7ef0593ab757f8a1e75cb0674e70627f9a9fcd7b before BUILD; intended T01..T05 contract docs/reviews/R103_BROWSER_TOOLING_SPEC_2026-10-15.md. Owner says xử lý đi after official-source recovery explanation. INTAKE actual empty driver157 cache/Node24.19/newer Chromium cache and upstream code -> DESIGN exact npm driver + existing Node, no version faking -> SPEC T01..05 -> WORK_ORDER. No product-source or project dependency change. Existing R101 no-download packet remains historical; this NEW authority explicitly permits only necessary official tooling downloads/cache writes at the exact named paths. Worker stages an inert reviewed repair script/plan first; Root admits network/cache runtime after independent source review.

CWD always project root; external mutation only seed-authorized local driver157/browser cache. Verify official npm dist.integrity SHA512, exact package version and archive path/type bounds before extraction into ignored staging. Preserve current empty dir/preexisting browsercache; no overwrites/deletes or global config. Reuse verified existing node.exe. Browser installer: pinned driver, Chromium only as needed, PLAYWRIGHT_SKIP_BROWSER_GC=1, record declared hosts/redirects, no OS dependency install. Single driver/required browser install campaign, finite timeouts800MB/30min; no automatic retries or old-CDN fallback. Root owns any supported capture/UI/fixture/server operation; worker no Docker/browser-page/UI/provider or commit. CLI version/install checks are allowed only after Root script admission.

T01 official provenance/integrity/safe archive; T02 exact installed1.57 CLI and node hash; T03 compatible required browser artifacts without cache deletion; T04 separately attributed local blank-page capture/control, tool integration distinct from standalone package health; T05 complete command/files/request/failure/protection evidence. Root reviews all findings before acceptance. Tooling PASS alone never closes R101/UI/product. UI branch only if supported browser capture PASS and Root admits pinned R101 binary/exact frontend/existing fixture runner, one backend and Vite launch; old source drift disclosed, no sourceguard weakening. R101 baseline counters retained; no newfixture or R102 resource change.

Worker return includes exact script/hash, raw commands/stdout/stderr/exits, archive SRI/source URLs, files added/preexisting inventory, Node/driver/browser versions, actual effects/failures and pending checks. Stop on ownership/integrity/version/scope/security or command failure and diagnose before repair. No blind retries, provider/mock/secret/real customer/channel, source/dependency/Core/gate/workflow/global settings, user-app restart/kill, publication/deploy. Third repeated same-root repair stops cost escalation under AGENTS. No hosted CI or AI governance claim.

## Checker Source Read-Ahead Block

AGENTS/manifest/policy/workspace rules; current compact bootstrap/state/memory/handoff/implementation/index; immutable R103 seed and T01..T05 contract; prior R101 blocked reports/qualification and pinned fixture/binary/frontend records; upstream exact driver layout and official npm metadata. Fresh declaration/doctor and role acknowledgment before BUILD. Worker never edits seed or commits.

## Public Export Disposition

Disposition: DEFERRED_PRIVATE_ONLY
Local tooling recovery and synthetic evidence only; no public release or installed product activation.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R103 immutable seed SPEC and order | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | order continuity catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_BROWSER_TOOLING_R103_2026-10-15.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | T01 through T05 pinned driver/cache and capability evidence | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | actual failures counters limits and repair return | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | acknowledgment source first Root runtime admission | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | source and actual worker return | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent driver/cache and browser capability review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed material static gates catalog | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | exact committed material and bounded disposition | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | memory state handoff implementation roadmap catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | exact local committed range with profile pins | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |
