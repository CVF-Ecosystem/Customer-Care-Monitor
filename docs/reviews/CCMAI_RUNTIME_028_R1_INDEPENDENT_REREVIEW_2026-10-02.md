# CCMAI-RUNTIME-028 R1 — independent re-review

Date: 2026-10-02. Reviewer: Codex, independent from repair worker Claude.
Exact repair: `b5703518f1ac2f10bb7bd25cf7edb5e89798ecb0`, on original BUILD `419a31ce0e34442a7410ad7ffb81d8586115e856`.
Disposition: **CHANGES_REQUIRED / FREEZE_OPEN**, consolidated repair R028-R2.
Authority: unchanged dispatcher seed `6958e281d190935578809bb9900dd95d2a29509e`, [SPEC](../specs/RUNTIME_JOB_RUN_OWNERSHIP_F06_2026-10-02.md), [order](../work_orders/CCMAI_RUNTIME_028.md), [first review](CCMAI_RUNTIME_028_F06_INDEPENDENT_REVIEW_2026-10-02.md), appended [worker evidence](RUNTIME_JOB_RUN_OWNERSHIP_F06_BUILD_2026-10-02.md).

## Intake and provenance

Manifest/policy/state/current memory/handoff/status/index and order/record agree on REVIEW_PENDING; no continuity drift. Core `26c686c` matches public origin/main, doctor 25/25; knowledge ingest completed. Compact bootstrap absent: BOOTSTRAP_MIGRATION_PENDING, nonblocking. Role COMMIT_STEWARD (Claude) -> REVIEWER (Codex) acknowledged before re-review. After an interruption, current continuity was reread and doctor repeated before completing disposition.

All 14 repair paths are within the existing seed/session/evidence scope. Seed/SPEC and the three original engine reviewer probes are unchanged; original mounted cases retain their assertions. No reviewer product fix. Only a new failing application-log probe and review/continuity artifacts are added here. No real provider/channel call, persistent DB, push/deployment/core work or FREEZE.

## R1 findings assessed

- **F06-R1-01 cancellation outcome repaired:** early failure, panic and Abort now use closeOwnedRun with the owner decision and checked finalizer. Accepted cancel stores cancelled; terminal-first rejects later cancel. Normal completion still preserves F03/F05 checkpoint separation. Post-terminal notification panic keeps the stored/returned terminal outcome rather than re-closing it. Original two cancellation probes and new terminal-path/fault groups pass. The bounded application-log portion remains unmet, detailed below.
- **F06-R1-02 settled:** RunReserved/executeReserved validate tenant/job binding before consuming the reservation or producing effects; Abort uses the actual bound identity. Original wrong-job probe passes, plus other-job/occupied-job/cross-tenant/exactly-once/bound-Abort tests.
- **F06-R1-03 settled:** pending cancellation settles only on that exact ID in success/partial/error/cancelled, with observed-outcome copy. Missing/unknown/empty/newer-only states remain pending. Original three mounted probes and added terminal/status tests pass.

## Remaining bounded repair

| ID | Priority / contract | Evidence and required change |
| --- | --- | --- |
| F06-R2-01 | P2, F06-07; bounded-log requirement already in R028-R1 | failOwnedRun still prints `cause` with `%v` at analyzer.go:860. Retained new probe injects harmless synthetic driver detail into the real conversation query and captures the standard application log: returned/stored error is bounded, but that detail reaches the application sink. TestReviewF06EarlyFailureBoundsDriverDetailInAppLogs FAIL. Replace this newly introduced raw-cause logging with a fixed bounded class/code (do not print arbitrary driver/parser/config/provider error values). Keep checked terminal behavior and useful correlation. Cover relevant early failure classes and retain the probe; no global logging rewrite or candidate/provenance redesign. The test isolates the application sink and does not certify all inherited GORM/SQL sinks. |
| F06-R2-02 | P2, required passing regression evidence, group F | Focused handler run again FAILS, now with the assertion retained: TestTriggerJobRealRouteConditionalDateCapRepeatsEvaluation, job_trigger_modes_route_test.go:178, conditional returns409 job_already_running. Its waitRun helper returns as soon as a stored row is non-running, then immediately launches the next request. F06 intentionally holds ownership past terminal persistence through activity/notification/cleanup. Log records job.run.completed activity after the failed launch, consistent with the worker still owning that tail. This identifies a test synchronization defect; do not relax correct production admission. Wait for the same run's observed terminal state AND actual ownership release/worker exit before asserting the next launch202, and join workers before fixture cleanup. Keep all existing F05 date/cap/repeated-evaluation/checkpoint assertions. Add deterministic delayed-tail coverage to show a launch remains409 while ownership is held and succeeds only after release, then rerun focused/full uncached gates with full logs. |

The second failure is a new fully captured recurrence on the current repair; it supplies a concrete cause for this run. It cannot prove which assertion caused the original truncated run, so that earlier record remains historical/unexplained. Four worker repeats without recurrence are not evidence that the original failure was harmless.

Claude next REPAIR_WORKER / COMMIT_STEWARD for one consolidated same-scope R028-R2 commit returned REVIEW_PENDING. Reviewer retains `analyzer_f06_logging_review_test.go`; inherited probes are not removed/skipped/weakened. No seed, objective, risk/effect class or commit-owner change. This is repair round two; any proposed third same-root repair requires the AGENTS REVIEW_COST_ESCALATION_REQUIRED audit.

## Independent checks

| Check | Result |
| --- | --- |
| Engine ownership/R1/F03/F05 focused command | 53 top-level PASS, zero FAIL/SKIP, package 220.000 s on disposable MySQL; includes all three original engine probes. |
| New bounded application-log probe | 1 FAIL, zero skips, package 2.125 s; returned/stored outcomes stay bounded, standard app log contains injected synthetic detail. |
| Handler focused command | 25 top-level PASS, 1 FAIL, zero skips, package 73.647 s; failure and complete log retained. |
| Frontend | 25 files / 255 tests PASS, 14.51 s; includes original three UI probes and new R1 status cases. |
| Forced typecheck and production build | PASS; forced vue-tsc project build, then normal build, bundling 1.51 s. |
| Gate tests | 46/46 PASS, 22.276 s. |
| Doctor | 25/25 PASS before review and after resume. |
| Cleanup | All three wrappers report disposable DB/network removal; no ccma-test containers/networks remain. No command touched persistent Compose data. |

Commands from project root:

~~~powershell
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'Test(Admission|Cancel|Terminal|Publication|StaleOwner|EarlyFailures|ProviderPanic|Timeout|Unresolved|NoLock|Cron|SchedulerOwners|F06|Ordinary|Explicit|EveryTerminal|Reservation|AbortUses|FailedTerminalClose|PanicAfterCommit)' -VerboseTests
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run TestReviewF06EarlyFailureBoundsDriverDetailInAppLogs -VerboseTests
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'Test(Route|AnalysisAgent|JobConfig|TriggerJob|TestRunJob|F06Probe)' -VerboseTests
npm --prefix frontend test -- --run
npm --prefix frontend exec -- vue-tsc -b frontend/tsconfig.json --force
npm --prefix frontend run build
python -m unittest discover -s scripts/tests -p 'test_cvf_downstream_gate*.py'
~~~

Complete local logs are under the reviewer's Temp directory; portable commands, assertion text and the new probe are committed. Worker 905 PASS / 2 optional skips, R019, build/mutations and repeat-run results remain worker evidence, not independently rerun full-suite proof. Final scoped preflight/catalog/docs/diff verification is recorded in handoff; default worktree failure on the pre-existing unrelated index/bytecode files stays explicit.

## Procedural nonconformance and remaining limits

Claude disclosed that the repair-role acknowledgment was written with final sync, after the first repair edit. This does **not** satisfy AGENTS' before-edit acknowledgment requirement; disclosure does not retroactively establish compliance. Existing R028-R1 dispatch/unchanged authority predated the repair, and no authority expansion/core edit was found. Preserve the late record without backdating. Before the next repair, rehydrate and write a fresh role acknowledgment before any edit; independent reviewer checks that record on return. No procedural-compliance or FREEZE claim is issued here.

Race detector remains unavailable with CGO off/no C compiler: NOT RUN, not PASS. Deterministic/contention tests are separate evidence. Ownership/cancellation remains local plus DB admission; crash-stale running rows block/unowned, requests are not crash-durable, already-issued provider work may finish. These accepted boundaries are not the blocking defects. Synthetic evidence proves local contracts only, not CVF governing AI. F05/R025/R026 acceptance remains unchanged; F06 stays OPEN.
