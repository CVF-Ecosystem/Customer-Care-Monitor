# R140: independent selection-read UI repair

Status: REVIEWER_ACCEPTED
workOrderStatusGrammar: cvf.workOrderStatusGrammar@1.0.0
Risk: R2
Worker: Assigned separate implementation worker
Reviewer: Root independent source reviewer; independent admission review before release
Authority: `CVF_SESSION/authority/CCMAI-RUNTIME-140.json`
Parent: CCMAI-RUNTIME-139 / same M3 registry, held integration

## Outcome and Root decision

When an operator switches saved requests or refreshes the same request, only the newest valid read may change status, readiness, error, loading or hold display. Responses from older selections must have no visible or durable side effect. The backend/POST execution contract remains unchanged. This fixes A03 from the Nov-5 audit; it does not solve R139 runtime lifecycle or M3-05/M3-08.

Root has completed the causal assessment and selected the design below. The worker implements it; worker does not research alternative browser stacks, create another roadmap, reopen P01, change acceptance, manage grants, appoint reviewers or commit. Parent assessment: `docs/reviews/R139_CAUSAL_CHAIN_ASSESSMENT_2026-11-05.md`. M3 history/costs/UNKNOWNs remain in the same registry; R140 is independent work, not a renamed integration repair.

## Source identity and first action

Seed is committed before the preparation base. Read current compact continuity and this packet, then verify `source_preimages.json` in `docs/reviews/probes/r140_selection_read_20261105/`. Root release receipt `docs/reviews/probes/r140_selection_read_20261105/root_release.json` must say RELEASED_SOURCE_ONLY and pin this work order, seed and source-preimage manifest. Verify its hashes and that its preparation base is an ancestor of checkout HEAD; missing/mismatched release means stop. Record actual checkout HEAD, role, exact allowed files and all counters at zero in the worker return before edits. If relevant source bytes differ, stop and report the exact difference; do not overwrite parallel work or repin yourself. Unrelated changes must be preserved and disclosed. Root owns state/authority/catalog and commits.

Required startup command from project root: `$env:PYTHONDONTWRITEBYTECODE='1'`. Reuse latest Root doctor/profile receipts if identities unchanged. Public doctor1.2 may emit the documented LOCK_INVALID; use the exact verified operator-local1.3 runner for installed-profile checks. No Core upgrade or profile edits.

## Exact worker scope

- `frontend/src/components/InterventionExecutionStatus.vue`
- `frontend/src/__tests__/intervention-execution-status.test.ts`
- `docs/reviews/R140_SELECTION_READ_WORKER_RETURN_2026-11-05.md`
- `docs/reviews/probes/r140_selection_read_20261105/worker_*` for secret-free commands, counters, test results and source hashes

Existing generated ignored compiler outputs are incidental only, not deliverables. No other source/test file, new helper module, dependency/lockfile, i18n, API, backend, runtime harness, Core/gate/CI, authority, registry, roadmap or continuity edits. If necessary correctness crosses these boundaries, return a specific blocker to Root before editing. Worker must not commit, push, start app/browser/DB/container/provider, access secrets, install or download dependencies.

## Fixed implementation contract

1. Keep the existing execution/lifetime `generation`, parent controller, POST and post-send observation ownership. Introduce a separate monotonically increasing selection/read-attempt identity for UI reads. Do not solve selection by blindly incrementing shared execution generation or aborting the execution controller.
2. Every initiated selected read (initial selection, explicit selection including A->B->A, refresh, and any readSelected reuse by observation) captures immutable requestId, read-attempt identity and current lifetime. A newer read invalidates the older read even if requestId is equal. Distinguish passive poll ownership from manual-read cancellation; do not cancel an already sent operation or expand polling bounds.
3. Abort superseded read work where safely supported, detach owned listeners, and guard commits with the identity even if a mocked/real promise ignores abort. The predicate includes mounted/current lifetime, request identity, current read identity and applicable signal. No request is sent as a consequence of stale completion.
4. Check ownership before every read-origin status/readiness/error/loading mutation, timer installation/callback, durable OBSERVED-hold update and follow-up readiness request. Recheck on resolve, reject and finally. Old reads cannot clear a newer spinner/status/error, renew a stale expiry, or update another request's hold. New selection clears only its intended current presentation and loads its own existing hold.
5. Lifetime/permission/actor/token/tenant/case/external-signal changes and unmount invalidate reads and clean their timers/listeners. Preserve existing read DTO validation, localized labels, current grant expiry calculation and unavailable/hold behavior.
6. Preserve the one-deliberate-POST path, fresh detail/readiness revalidation, pre-send IN_FLIGHT marker, no resend, uncertain hold and existing bounded observation (12 reads, >=5s spacing, <=60s). Keep selection blocked while sending. No new transport, retries, endpoint or observer lifecycle. If shared read ownership would change these semantics, stop with a concrete counterexample rather than widening scope.

## Acceptance matrix and required evidence

Tests use deterministic deferred promises and controlled fake timers; obsolete promises must resolve/reject despite aborted signals. Assertions concern presentation/lifetime only, not AI governance. Prefer user-observable rendered status, errors, buttons and holds; use request identity/call counts to diagnose ownership. Tests must fail on original source for the reported race, without exposing private component methods just to mirror implementation.

| ID | Arrange / trigger | Required result |
|---|---|---|
| S01 | A status pending; select B; resolve B then reject A | B status/readiness/error unchanged; stale A emits no new readiness or execute call |
| S02 | A-old pending; select B then A-new; resolve A-new then A-old | A-new state persists although IDs match; old hold/expiry not installed |
| S03 | Old readiness pending across A->B and A->B->A | Old success and rejection cannot change current readiness/button/error or expiry timer |
| S04 | Same-request initial/selection read overlaps newer refresh | Newest read wins; stale finally cannot clear new loading; user-accessible overlapping path tested without bypassing UI disabling |
| S05 | Abort ignored by an obsolete status/readiness promise | Identity guard still prevents stale mutation and follow-up network work |
| S06 | Change actor/permission/token/tenant/case or unmount while read pending | Existing lifetime protections remain; no late mutation, owned timers/listeners disposed |
| S07 | Current valid read success and current read failure | Normal status/readiness rendering and localized unavailable behavior work; no blanket suppression of current errors |
| S08 | Current request has durable uncertain hold; obsolete read reports observed execution | Hold ownership stays correct; no marker deletion or accidental clearing/resend |
| S09 | Existing deliberate send/poll tests, double click and storage failure cases | Existing component suite passes; pre-send storage failure produces zero POST, deliberate send <=1, uncertainty preserved, original polling bounds unchanged |
| S10 | Source/typecheck review | Exact two source/test files, TypeScript passes, no dependency/API/backend/execution-contract change |

All new selection-only regressions assert execute-call count0. Existing mocked send/poll tests validate frontend behavior only; no live provider/governance acceptance is claimed. Full E04/T3 and E06 remain unproved.

## Commands, bounds and stop

Use only installed dependencies. From `frontend`: `node node_modules/vitest/vitest.mjs run src/__tests__/intervention-execution-status.test.ts`; typecheck: `node node_modules/vue-tsc/bin/vue-tsc.js -b --pretty false`. Capture command, source SHA, start/end, stdout/stderr, exit code, total/pass/fail/skip and invocation counters in worker evidence. Do not fetch tools through npx.

Worker budget45 active minutes, up to3 focused test processes (one red test on unchanged component, one green, one only if changed-source diagnosis warrants), one typecheck. After Root review, at most one consolidated same-scope repair with additional15 minutes, one focused test and one typecheck; no self-release of that reserve. Root review ceiling15 minutes, consume evidence without routine duplicate runs. Mandatory static gates are separate from product test invocations but failures/time remain disclosed. No browser/provider/DB/runtime or network effect; old caps never reset.

Stop at budget exhaustion, missing installed dependency, protected source needed, inability to reproduce either source-derived race, conflict with no-resend/polling semantics, or conflicting source/authority. Return what was observed and exact remaining requirement; do not open a helper tranche, add retries or claim completion from fewer cases. Source repair within initial scope needs no repeated operator permission.

## Worker return and Root acceptance

Return `COMPLETE_PENDING_REVIEW` only when S01-S10 are evidenced, all required commands pass, no skipped critical case, exact scope respected and source hashes match. Otherwise return `BLOCKED_WITH_REASON` or `CHANGES_REQUIRED`, including original failed output. Explicitly report each matrix ID, red/green distinction, actual costs/counters, unknowns and any case not run; never sum overlapping test runs into unique cases.

Root checks source diff, the independent behavior oracle, stale success/error/finally/timer paths, preserved send/hold semantics and final-byte evidence; separate reviewer may advise. Worker never self-approves. Root then owns material commit, committed-range checks and continuity. Product remains OPEN/BLOCKED even after bounded source acceptance; route runtime proof is a separate R139 prerequisite.

## Gate-To-Role Closeability Contract


closeabilityContractVersion: cvf.gate-role-closeability@1.0.0
closeabilityDisposition: CLOSEABLE
returnTimeRecheck: REQUIRED_BEFORE_REPAIR
implementationTopologyPolicy: BOUNDED_PATH_FAMILIES
foreseeableFileSplitDisposition: COVERED_BY_BOUNDED_PATH_FAMILY

| gateId | mustPassBy | repairOwner | repairPhase | mutationSurface | topology | commitOwner | commitPhase | dependsOn |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| authorization_review | PRE_DISPATCH | orchestrator | PRE_DISPATCH | R140 immutable seed, current roadmap and exact owner request | EXACT_PATHS | commit-steward | DISPATCH_COMMIT | NONE |
| pre_dispatch_gate | PRE_DISPATCH | orchestrator | PRE_DISPATCH | order and synchronized continuity/catalog | BOUNDED_PATH_FAMILIES | commit-steward | DISPATCH_COMMIT | authorization_review |
| dispatch_continuity | IMPLEMENTATION | session-sync-steward | IMPLEMENTATION | CVF_SESSION/handoffs/AGENT_HANDOFF_R140_SELECTION_READ_REPAIR_2026-11-05.md material-SHA marker | EXACT_PATHS | session-sync-steward | DISPATCH_CONTINUITY_COMMIT | pre_dispatch_gate |
| focused_checker_tests | WORKER_RETURN | implementation-worker | WORKER_RETURN | bounded source, pure/component tests and acceptance contract | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| adif_integrity | WORKER_RETURN | implementation-worker | WORKER_RETURN | original outcomes and current applicability limitations | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests |
| pre_implementation_autorun | IMPLEMENTATION | implementation-worker | IMPLEMENTATION | pre-BUILD rehydration/role acknowledgment | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | dispatch_continuity |
| worker_return_fast | WORKER_RETURN | implementation-worker | WORKER_RETURN | source/test identity and exact worker return | EXACT_PATHS | commit-steward | MATERIAL_COMMIT | focused_checker_tests,adif_integrity,pre_implementation_autorun |
| reviewer_fast | REVIEW | reviewer | REVIEW | independent source/component software review | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | worker_return_fast |
| pre_commit | PRE_MATERIAL_COMMIT | reviewer | PRE_MATERIAL_COMMIT | reviewed source, software evidence, scope, continuity and catalog | BOUNDED_PATH_FAMILIES | commit-steward | MATERIAL_COMMIT | reviewer_fast |
| terminal_completion_review | POST_MATERIAL | reviewer | POST_MATERIAL | committed source/component evidence and remaining runtime acceptance | EXACT_PATHS | commit-steward | CONTINUITY_COMMIT | pre_commit |
| continuity | CONTINUITY_COMMIT | session-sync-steward | CONTINUITY_COMMIT | state/memory/handoff/status/roadmap/catalog | BOUNDED_PATH_FAMILIES | session-sync-steward | CONTINUITY_COMMIT | terminal_completion_review |
| committed_range_closure | POST_MATERIAL_CLOSURE | reviewer | POST_MATERIAL_CLOSURE | local committed R140 range and pinned static gates | EXACT_PATHS | session-sync-steward | CONTINUITY_COMMIT | continuity |


## Complete milestone coverage

```cvf-milestone-coverage-json
{"schemaVersion":"cvf.milestoneCoverage@1.0.0","authority":{"path":"docs/reviews/planning/CCMA_PRODUCT_DELIVERY_ROADMAP_2026-10-13.md","heading":"## 7. M3 — Người duyệt, giao việc và kết quả xử lý","sha256":"712817cd25936efd4fb13e0dd456ebe4ddceb5fccfd1cacb83042838724065a4"},"milestone":{"id":"M3","consumer":"Người duyệt và người xử lý ca trên một installation Pancake","outcome":"Hoàn tất review, giao việc, kết quả và lịch sử qua browser/backend theo tiêu chí M3 đã duyệt.","checkpoint":{"trigger":"Independent selection-read source return or scope/cap contradiction","owner":"Root ORCHESTRATOR / independent source reviewer","decision":"Accept bounded source correctness or one finite repair; never close R139 integration or grant runtime."}},"phase":"DISPATCH","action":{"kind":"IMPLEMENT","selected":["M3-06"],"owner":"Assigned separate implementation worker","output":"Source correction and deferred-promise regressions for independent selection-read identity in existing Vue component","budgetMinutes":45,"stop":"Stop on protected surface or missing dependency; no product/provider/runtime effects or automatic budget reset. One Root-reviewed consolidated repair only within seed bounds.","next":"Return exact source/evidence to Root for independent review; R139 M3-05/M3-08 remains blocked."},"rows":[{"id":"M3-01","state":"IMPLEMENTED_UNPROVED","gap":"Quy tắc ưu tiên và applicability current-source còn mở.","owner":"Product Owner / Root","destination":"M1-01 quyết định trigger/quy tắc chờ và ngoại lệ; Owner xử lý quy tắc queue còn mở, Root đối chiếu source applicability trước G3.","evidence":[{"path":"docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md","sha256":"cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]},{"id":"M3-02","state":"IMPLEMENTED_UNPROVED","gap":"Quyền, reason và binding current-source chưa có acceptance đầy đủ.","owner":"Root / Product Owner","destination":"Root kiểm binding và trace; Owner chấp nhận process tại G3.","evidence":[{"path":"docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md","sha256":"ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"},{"path":"docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md","sha256":"cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]},{"id":"M3-03","state":"IMPLEMENTED_UNPROVED","gap":"Applicability lineage và negative hiện hành còn cần đối chiếu.","owner":"Root / Product Owner","destination":"Root kiểm source; G1 stale/competing negative nếu source thay đổi hoặc lộ gap; Owner chấp nhận effect.","evidence":[{"path":"docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md","sha256":"ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"},{"path":"docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md","sha256":"cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}]},{"id":"M3-04","state":"IMPLEMENTED_UNPROVED","gap":"Role, SLA và demo current-source chưa hoàn tất.","owner":"Product Owner / Root","destination":"M1-02/04 quyết định role/SLA; G3 demo assignment, outcome và audit.","evidence":[{"path":"docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md","sha256":"ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"}]},{"id":"M3-05","state":"BLOCKED","gap":"E01–E06 chưa có runtime proof; physical-relaunch và integrated same-context/original-clock đang HOLD.","owner":"Root / Product Owner","destination":"Cùng R139 G0–G3 sau causal-chain audit, thiết kế tích hợp và admission phù hợp; Owner chấp nhận user consequence.","evidence":[{"path":"docs/reviews/R137_INDEPENDENT_SOURCE_REVIEW_2026-10-21.md","sha256":"63cc0bc309a235c7d36384b7b78afa48acafea8267622d93d191780633071783"},{"path":"docs/reviews/R139_GEMINI_NATIVE_BROWSER_CHECK_ROOT_REVIEW_2026-10-28.md","sha256":"0f434528ae218886eca1c8c68efb68d3cab164cbad57b2a2b51af79f14135d47"}],"independence":{"reviewer":"Root and independent Astra admission reviewer","reason":"R140 only changes local read-selection presentation lifetime. It does not use or change product execution, runtime context/clock, provider, grant, backend, notification or held integration criteria. Any such dependency stops worker.","evidence":[{"path":"docs/reviews/R139_CAUSAL_CHAIN_ASSESSMENT_2026-11-05.md","sha256":"c3f3f68330a752f32b1312583d5da2cbe3ef1e8629e4d53dfb3c0ceb800f764a"}]}},{"id":"M3-06","state":"IMPLEMENTED_UNPROVED","gap":"Proven selection-read stale success/error race; separate read identity and deferred interleaving tests required. Full browser lifetime remains unproved.","owner":"Root / Product Owner","destination":"R140 source/test correction then Root review; actual browser proof remains R139 G2 after integration admission.","evidence":[{"path":"docs/reviews/R137_INDEPENDENT_SOURCE_REVIEW_2026-10-21.md","sha256":"63cc0bc309a235c7d36384b7b78afa48acafea8267622d93d191780633071783"}]},{"id":"M3-07","state":"UNKNOWN","gap":"Phạm vi R101 hiện không gửi notification; applicability với source và scope hiện hành còn cần kiểm tra.","owner":"Root","destination":"G3 đối chiếu applicability của exclusion đã có; nếu scope đổi, áp dụng change control trước khi xét contract notification.","evidence":[{"path":"docs/reviews/R131_M3_ACCEPTANCE_MATRIX_2026-10-19.md","sha256":"7dabfc6a2a8bbf42a4447063e961a428849b0a020ae4c02589a7fb323510f9ac"}],"independence":{"reviewer":"Root and independent Astra admission reviewer","reason":"R140 only changes local read-selection presentation lifetime. It does not use or change product execution, runtime context/clock, provider, grant, backend, notification or held integration criteria. Any such dependency stops worker.","evidence":[{"path":"docs/reviews/R139_CAUSAL_CHAIN_ASSESSMENT_2026-11-05.md","sha256":"c3f3f68330a752f32b1312583d5da2cbe3ef1e8629e4d53dfb3c0ceb800f764a"}]}},{"id":"M3-08","state":"BLOCKED","gap":"Current-source browser/backend demo và disposition viewport/offline còn mở; physical-relaunch HARD_HOLD, integrated lifetime DESIGNED_NOT_EXECUTABLE; successor admission pending; relaunch prevention unresolved.","owner":"Root / Product Owner","destination":"Cùng R139 causal-chain audit trước thiết kế G2 proof và G3; Owner chấp nhận qualification viewport/raster/offline.","evidence":[{"path":"docs/reviews/R139_GEMINI_NATIVE_BROWSER_CHECK_ROOT_REVIEW_2026-10-28.md","sha256":"0f434528ae218886eca1c8c68efb68d3cab164cbad57b2a2b51af79f14135d47"},{"path":"docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md","sha256":"cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"}],"independence":{"reviewer":"Root and independent Astra admission reviewer","reason":"R140 only changes local read-selection presentation lifetime. It does not use or change product execution, runtime context/clock, provider, grant, backend, notification or held integration criteria. Any such dependency stops worker.","evidence":[{"path":"docs/reviews/R139_CAUSAL_CHAIN_ASSESSMENT_2026-11-05.md","sha256":"c3f3f68330a752f32b1312583d5da2cbe3ef1e8629e4d53dfb3c0ceb800f764a"}]}}],"supportingWork":[],"conclusions":{"tranche":"PENDING","trancheSummary":"R140 worker source/test deliverable not started; preparation only","trancheEvidence":[],"product":"PENDING","productSummary":"M3 not accepted; R139 integration BLOCKED, all eight criteria retained","acceptance":[],"owner":"Root technical reviewer; operator product/business decisions"}}
```

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
