# CCMAI-RUNTIME-028 / F06 — independent source and evidence review

Date: 2026-10-02. Reviewer: Codex, independent from implementation worker Claude.
Exact BUILD: `419a31ce0e34442a7410ad7ffb81d8586115e856`.
Disposition: **CHANGES_REQUIRED / FREEZE_OPEN**. F06 remains OPEN.
Authority: [SPEC](../specs/RUNTIME_JOB_RUN_OWNERSHIP_F06_2026-10-02.md), [work order](../work_orders/CCMAI_RUNTIME_028.md), dispatcher seed committed by Codex at `6958e281d190935578809bb9900dd95d2a29509e`; [worker evidence](RUNTIME_JOB_RUN_OWNERSHIP_F06_BUILD_2026-10-02.md).

## Intake, scope and role

Rehydrated manifest/policy/state/memory/handoff/status/index at exact BUILD; REVIEW_PENDING facts agree, no continuity drift. Core `26c686c` matches public origin/main; workspace doctor 25/25; knowledge ingest completed. Compact bootstrap absent: BOOTSTRAP_MIGRATION_PENDING, nonblocking. Role COMMIT_STEWARD (Claude) -> REVIEWER (Codex) recorded in active handoff before evaluation. Risk ceiling R2, live governance evidence required YES; this review asserts local source/storage/request contracts only, with zero real provider/channel calls.

All 26 BUILD paths fit the seed and session/evidence allowances. Seed and issued SPEC are unchanged from dispatch `94875b4`; seed predates BUILD and is the authority observed/authored by Codex in this session. `analyzer_incremental.go` diff is restricted to Job -> JobRun locks, running-only terminal lookup/fallback. MCP placeholder and unused helper remain untouched. Reviewer adds regression probes and review/continuity artifacts; no product repair.

## Blocking findings and bounded repair

| ID | Priority / requirement | Evidence and required repair |
| --- | --- | --- |
| F06-R1-01 | P1, F06-05/07/08, group D | Normal completion uses owner.beginTerminal, but failOwnedRun, panic recovery and reservation Abort bypass that outcome serialization. Two retained probes accept CancelJobRun first, then cause malformed channel JSON or provider panic: both store run and job status error rather than cancelled. Make every terminal path use the same run-owned cancellation/terminal decision and checked persistence; cover early provider selection/input/candidate failures, panic and abort/setup failure, both cancel-first and terminal-first. Preserve no-checkpoint/no-notification and fail-closed DB faults. Keep bounded public/log errors on the changed lifecycle paths. |
| F06-R1-02 | P1, tenant/job/run ownership contract, F06-01/02/08 | RunReserved/executeReserved accept an independent job argument without checking it against the opaque reservation's tenant/job/run tuple. Probe reserves job A, invokes RunReserved with same-tenant job B: provider=1, published=2 (snapshot/evaluation results for B's source under A's run), followed only afterward by terminal target missing. Admission ownership of A does not authorize executing B. Validate the binding before consumption, activity, provider resolution/call, source selection or publication; terminal/Abort must use the reservation's actual bound identity and never mutate an unrelated job. Add same-tenant different-job and cross-tenant cases, including B already occupied; preserve legitimate exactly-once execution and cleanup. |
| F06-R1-03 | P2, F06-10, group F | JobDetail polling uses “not found with running status” as terminal. Three mounted probes return an empty list, status unknown and empty status for the pending target: each displays “Lượt chạy đã kết thúc trước khi hủy kịp có hiệu lực.” and stops polling. Require observation of that exact ID in a recognized terminal state (success/partial/error/cancelled), keep missing/unknown states unconfirmed and polling, and report its observed outcome truthfully. A newer run or list absence must not settle the target. Preserve delayed-click/run-ID targeting, request-failure and F05 UI regressions. |

Retained reviewer tests: `backend/engine/analyzer_f06_review_test.go` (three tests), plus three parameterized mounted cases appended to `frontend/src/__tests__/job-run-cancel.spec.ts`. They intentionally fail at reviewed BUILD; do not remove/skip/weaken them. These are defects in the issued contract, not a scope expansion or a request for distributed guarantees. Claude next REPAIR_WORKER / COMMIT_STEWARD under unchanged seed, one consolidated local repair commit returned REVIEW_PENDING.

## Independent verification

| Check | Observed result |
| --- | --- |
| Focused existing engine | PASS, package 136.367 s. Command below covers ownership/lifecycle/cron plus ordinary/explicit regressions. Not a full-backend rerun. |
| Retained engine reviewer probes | 3/3 FAIL, reproducible isolated replay 1.284 s: early failure stores error/error; panic stores error/error; mismatched reservation makes 1 provider call and publishes 2 result rows before terminal error. |
| Existing handler focused group | First parallel run exit 1 (48.584 s); output was truncated and the failing assertion was not retained. Isolated replay: 26 top-level PASS, zero skips, package 37.218 s. Initial failure is not erased or explained by the replay; worker must include an uncached focused/full rerun and investigate any recurrence. No additional code defect is inferred solely from that truncated run. |
| Frontend original BUILD suite | 25 files, 246/246 PASS, 34.64 s (before reviewer probes). |
| Mounted cancel suite with corrected reviewer probes | 7 existing PASS, 3 new FAIL, 10 total, 10.83 s; no unhandled errors. The first reviewer parameterization incorrectly spread array rows and was corrected before this reported result; its harness errors are not product evidence. |
| Forced vue-tsc and frontend build | PASS; forced project build followed by normal frontend build, production bundling 1.62 s. |
| Gate unit tests | 46/46 PASS, 20.561 s. |
| Race detector readiness | In network-isolated golang:1.26-alpine, gcc/clang absent; CGO_ENABLED=0 go test -race reports “-race requires cgo”. NOT RUN / NOT PASS. Deterministic/multigoroutine tests are separate evidence. |
| Resource cleanup | Each wrapper reports disposable DB/network removal; docker ps and ccma-test network listing confirm no reviewer disposable resources remain. Persistent Compose services untouched. |

Reproducible commands from project root:

~~~powershell
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'Test(Admission|Cancel|Terminal|Publication|StaleOwner|EarlyFailures|ProviderPanic|Timeout|Unresolved|NoLock|Cron|SchedulerOwners|F06Probe|Ordinary|Explicit)' -VerboseTests
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run TestF06Review -VerboseTests
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'Test(Route|AnalysisAgent|JobConfig|TriggerJob|TestRunJob|F06Probe)' -VerboseTests
npm --prefix frontend test -- --run
npm --prefix frontend test -- --run src/__tests__/job-run-cancel.spec.ts
npm --prefix frontend exec -- vue-tsc -b frontend/tsconfig.json --force
npm --prefix frontend run build
python -m unittest discover -s scripts/tests -p 'test_cvf_downstream_gate*.py'
~~~

Disposable probe and handler replay logs were retained under the reviewer's local Temp directory; the committed probes and reported assertion text are portable. No raw customer data, credentials or real-provider transcript is attached.

## Worker notes and evidence limits

Race limitation is accurately disclosed; it is an unresolved verification limit, not approval to label race PASS. Single-process cancellation, database running-row admission, orphan blocking/unowned response, non-durable cancellation and already-issued provider work match the scoped design; they do not excuse the three blocking defects.

Existing test changes were inspected. Config admission retains invalid config/tenant no-launch/no-run/no-owner assertions, validated config identity and cap assertions while adapting to reservations and adding run_id. The stored-cancel hook now requests cancellation through the real coordinator, consistent with the handler's new contract; anchor failure remains non-success/no-call with a bounded error. These edits are justified; no weakened inherited assertion was identified. They do not cover the new failing boundaries.

Worker full backend 883 PASS / 2 optional skips, R019 sentinels, build/vet, old-source probes and mutation results are worker evidence, not independently replayed here. Review does not claim current hosted CI, runtime CVF governance, full provider interruption, distributed fencing, recovery, deployment or FREEZE. F05/R025/R026 acceptance remains unchanged.

Final review-dispatch checks are recorded in the active handoff: complete BUILD + review path preflight/catalog, docs build and diff checks. Default worktree gate must disclose the existing unrelated knowledge index and two Python bytecode files; they stay outside commits. Failing application probes remain explicit CHANGES_REQUIRED criteria.
