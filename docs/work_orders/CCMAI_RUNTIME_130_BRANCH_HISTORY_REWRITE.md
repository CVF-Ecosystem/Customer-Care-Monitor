# Feature branch history cleanup — R130

Status: ACTIVE
Risk: R2
Worker: Codex /root Git history maintenance
Reviewer: Owner independent review of the rewritten graph required before force-push

## Objective

Remove `docs/reviews/probes/review_completion_20261018/local_verification_originals.zip` from every commit reachable from `codex/ccma-m2-intervention`, while preserving the complete feature branch file tree as published by R129 and retaining the local backup.

## Design and bounds

The ZIP entered at `90bc02ed14c5e3997ec97fe400a64cf06cae6f70`, parent `9ab54062976de181ffbb910be44393021d1bc3db`. Rewrite only that commit and descendants. Keep the parent and all earlier history unchanged. Preserve every file at the current tip except the ZIP, which remains absent. Publish only to `codex/ccma-m2-intervention` using force-with-lease after independent owner review of the new graph and tree comparison. Do not modify or merge into `main`; do not delete the local backup.

## Acceptance and stop conditions

- Rewritten commit ancestry starts at the original parent and retains the R129 cleanup docs.
- The ZIP path is absent from every commit reachable from the new feature tip.
- Old and new tip trees are byte-identical except for the already absent ZIP path.
- Local backup hash matches the pre-rewrite Git blob.
- `origin/main` remains at its original SHA.
- Owner reviews and accepts the rewritten graph before force-push.

Any unrelated tree change, changed parent before the target commit, unexpected main movement, backup mismatch, gate failure, or remote branch movement stops publication.

## Gate-To-Role Closeability Contract

closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R130 seed, immutable 9ab54062 parent, ZIP introduction commit and bounded owner request | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | work order, R130 seed and synchronized continuity | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R130_BRANCH_HISTORY_REWRITE_2026-10-19.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | parent-boundary and target-commit validation; no product tests | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | exact original/new tree comparison and local backup hash | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | authorized range 90bc02ed..branch tip only | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | rewritten graph scan, preserved tree and continuity | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent owner review of exact graph and tree comparison | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed candidate and CVF gate receipts | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | all reachable branch commits free of ZIP and actual rewritten SHA | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | state, memory, handoff, tranche and catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | force-with-lease feature ref readback and unchanged main | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |
