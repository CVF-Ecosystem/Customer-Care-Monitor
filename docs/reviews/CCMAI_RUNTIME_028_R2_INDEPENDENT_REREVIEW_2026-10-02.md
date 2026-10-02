# CCMAI-RUNTIME-028 R2 — independent re-review

Date: 2026-10-02. Reviewer: Codex, independent from implementation/repair worker Claude.
Exact repair: `8035cb05a2d66778810fb0acb89d208238743cfc`, on R1 repair `b5703518f1ac2f10bb7bd25cf7edb5e89798ecb0` and original BUILD `419a31ce0e34442a7410ad7ffb81d8586115e856`.
Disposition: **REVIEW_PASS / FREEZE_OPEN**.
Authority: unchanged dispatcher seed `6958e281d190935578809bb9900dd95d2a29509e`, [SPEC](../specs/RUNTIME_JOB_RUN_OWNERSHIP_F06_2026-10-02.md), [order](../work_orders/CCMAI_RUNTIME_028.md), [R1 re-review](CCMAI_RUNTIME_028_R1_INDEPENDENT_REREVIEW_2026-10-02.md), appended [worker evidence](RUNTIME_JOB_RUN_OWNERSHIP_F06_BUILD_2026-10-02.md).

## Intake, scope and source assessment

Canonical manifest/policy/state/current memory/handoff/status/index and order/tranche agree on REVIEW_PENDING. Doctor 25/25 verifies core `26c686cc99b8be965d2760f27fe875b03376c643` against public origin/main. Required knowledge ingest completed; compact bootstrap absent (BOOTSTRAP_MIGRATION_PENDING, nonblocking). Reviewer-role acknowledgment was written before substantive review. All 14 R2 paths fit existing source/session/evidence authority. Seed/SPEC and original/R1 probes are unchanged. Reviewer changes only review/disposition/continuity records, with no product repair.

F06-R2-01: failOwnedRun no longer accepts the underlying error value. Its callers use fixed classes provider_unavailable, input_channels_invalid or candidate_selection_failed; provider selection stores a fixed user message. Application log retains job/run/class correlation. The terminal-write failure message contains only the checked finalizer's fixed wrapped sentinel. Checked cancellation/terminal behavior is unchanged. The retained independent log probe and added five-class test must pass. This assessment concerns the application sink on this early-error path; inherited GORM/SQL sinks are outside this certification.

F06-R2-02: the F05 route helper now observes a terminal row AND inactive ownership before asserting the next accepted launch; cleanup waits for owner release instead of a fixed 200 ms sleep. All date/cap/repeated-evaluation/checkpoint assertions remain. The new engine test holds the notification tail behind a channel; the handler test holds completion-activity insertion behind a MySQL named-lock barrier. Both inspect the already-terminal row, reject another launch while local ownership remains and accept it after release. Production admission/coordinator/finalizer code is unchanged in R2.

R1 accepted cancellation, reservation binding and exact-run UI fixes remain intact. Retained original reviewer probes are included in independent reruns. No remaining source finding identified in this bounded re-review. Both R2 findings settled; no third repair round is required.

## Independent checks

Complete backend logs are retained in the reviewer's local Temp directory, with portable commands below.

| Check | Result |
| --- | --- |
| Engine ownership/R1/R2/F03/F05 and retained log probe | 56 top-level PASS, zero FAIL/SKIP, 208.151 s on disposable MySQL. Includes all original/R1 probes, log classes and terminal-tail barrier. |
| Handler ownership and F05 routes, including terminal-tail barrier | 27 top-level / 56 subtests PASS, zero FAIL/SKIP, 62.753 s on disposable MySQL. |
| Frontend | 25 files / 255 tests PASS, 43.40 s, including original mounted reviewer probes. |
| Forced typecheck and production build | PASS; bundling 5.39 s. |
| Gate unit tests | 46/46 PASS, 42.577 s. |
| Doctor | 25/25 PASS. |
| Preliminary scoped preflight | Complete 38-path seed/planning/BUILD/review/repair/disposition set: 7/7 PASS, including catalog. |
| Default whole-worktree preflight | FAIL (6/7): tranche rejects only pre-existing knowledge/_index.json and two Python bytecode files outside authority. Those files remain excluded; no whole-worktree PASS claim. |

Initial docs build PASS (30.88 s); final disposition build/preflight/diff verification is recorded in the handoff. Both test wrappers report disposable MySQL/network removal; final docker inventory confirms no ccma-test resources remain. Logs: `ccmai-r028-r2-engine-1d70d837f1c74963bc1bc86836276998.log` and `ccmai-r028-r2-handlers-22f3034fd31b4a90b842b5a0d16cd024.log` in the reviewer local Temp directory.

~~~powershell
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'Test(Admission|Cancel|Terminal|Publication|StaleOwner|EarlyFailures|ProviderPanic|Timeout|Unresolved|NoLock|Cron|SchedulerOwners|F06|Ordinary|Explicit|EveryTerminal|Reservation|AbortUses|FailedTerminalClose|PanicAfterCommit|ReviewF06|EarlyFailureClasses)' -VerboseTests
powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'Test(Route|AnalysisAgent|JobConfig|TriggerJob|TestRunJob|F06Probe)' -VerboseTests
npm --prefix frontend test -- --run
npm --prefix frontend exec -- vue-tsc -b frontend/tsconfig.json --force
npm --prefix frontend run build
python -m unittest discover -s scripts/tests -p 'test_cvf_downstream_gate*.py'
npm --prefix docs run docs:build
~~~

Worker full backend 914 PASS / 2 optional skips, R019 sentinels, build/vet, five killed R2 mutations and three handler repeats remain attributed worker evidence. Independent focused runs do not become a claim of an independently repeated full suite or mutation campaign. The dispatcher seed is unchanged from its first committed content and exists at baseCommit before BUILD; its Codex authorship/timing is supported by the recorded dispatch/session, not inferred from the shared Git identity.

## Historical limits and disposition boundary

R1 repair-role acknowledgment was recorded after edits: historical procedural nonconformance, not backdated or retroactively compliant. R2 handoff and worker evidence explicitly record acknowledgment before R2 edits; a final single commit alone cannot independently prove intra-worktree timing. No overall procedural-compliance claim is issued. The disclosed partial continuity synchronization failed preflight as expected; final canonical surfaces now agree. Preserve that failure/recovery history rather than claiming uninterrupted consistency.

The original first handler failure had truncated output and remains unexplained. The later captured R2-02 failure has the terminal-tail waiting cause and repair; passing repeats do not explain the old failure. Race detector NOT RUN (CGO disabled/no C compiler); deterministic orderings and contention tests do not replace it.

Ownership is in one process plus DB admission during the running-row interval. Unowned crash-stale running rows block admission and cannot be cancelled here; cancellation is not crash-durable; already-issued provider calls may finish. MCP trigger placeholder, dead helper cleanup, F02 live/message limits, F07 and tenant timezone activation remain separate. Synthetic evidence proves local admission/cancellation/storage/request contracts, not CVF governing AI. No real provider/channel/notification call, persistent DB, push/merge/deployment, parent-CVF edit or FREEZE. F05/R025/R026 retain REVIEW_PASS / FREEZE_OPEN. A third same-root repair, if required, needs REVIEW_COST_ESCALATION_REQUIRED before repair begins.

## Next governed move

Reviewer transitions to ORCHESTRATOR / SESSION_SYNC_STEWARD / COMMIT_STEWARD for bounded disposition synchronization and local review commit. Separate F07 INTAKE/DESIGN/SPEC/WORK_ORDER may follow; no F07 BUILD is authorized by this review. F06 functional contract is accepted within its stated limits, while FREEZE remains OPEN.
