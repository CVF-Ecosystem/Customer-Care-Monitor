# R077 nonbinding startup-hardening work-order draft

**Status:** proposal only. This is not an immutable seed, dispatch, implementation authority, or approval to inspect a deployment. Any implementation requires a separate R2+ work order, seed, BUILD release, and independent review.

## Proposed objective

Stop a new application process from automatically rewriting persisted JobRun status during scheduler startup when the old writer's state is unknown. Keep unresolved `running` rows unchanged so existing database admission continues to block a second run. This is a fail-closed default, not a recovery mechanism.

The source supports the smallest slice without a deployed owner-death proof: remove the `cleanupStuckRuns()` call from `Scheduler.Start` and remove the private helper if a fresh reference search confirms it has no other caller (`backend/engine/scheduler.go:97-128`). Do not replace it with age-based cleanup, local-owner absence checks, a force switch, or another status rewrite. A stale row may remain blocked; that consequence is explicit and requires a separate future recovery design. This does not establish behavior for older binaries that may still run the existing sweep.

## Candidate paths and roles

| Path | Proposed use |
|---|---|
| `backend/engine/scheduler.go` | Remove the JobRun startup sweep call and now-unused private helper; retain startup sync-lease recovery, cron loading, and scheduler start/stop behavior. |
| `backend/engine/scheduler_startup_hardening_test.go` | New focused integration tests that invoke `Scheduler.Start` and `Scheduler.Stop` against isolated synthetic data, with a JobRun-write side-effect trap. |

Existing regression files are to be run, not modified under this proposal: `backend/engine/scheduler_ownership_test.go` and `backend/engine/sync_lease_recovery_test.go`. `backend/main.go`, JobRun schema/model, sync code, handlers, providers, notification paths, and deployment files are outside the proposed change unless a new reviewed work order explicitly expands them.

Proposed roles for a future seed: root ORCHESTRATOR/WORK_ORDER_AUTHOR/CLOSER/COMMIT_STEWARD; one separately named IMPLEMENTATION_WORKER; independent REVIEWER. R2 is the minimum risk because the current startup path writes persisted execution state and the proposed change changes that behavior. Reviewer must not be the implementation or repair worker.

## Intended behavior and compatibility

1. In the hardened binary, the startup-recovery path makes no JobRun update attempt. A `running` row with no owner in the new process remains `running`; age, an empty local owner map, or unknown deployment topology never authorizes a rewrite. A stale row may consequently remain blocked until separately resolved.
2. A distinct process-like coordinator that sees the same persisted `running` row must still fail same-job reservation through database admission (`ErrJobBusy`); a same-process local busy guard alone is not an acceptable test. An unrelated free job remains normally admissible. No new run/provider/result/notification/retry is caused by the startup hardening itself.
3. A deterministic fixture may assert full JobRun row equality during the bounded `Scheduler.Start` interval only while jobs and owners are controlled so no legitimate finalizer can run. This fixture-level equality does not prohibit ordinary owner/cron finalization. A live owner may terminalize its own row after the Start observation window; attribute that write to the owner finalizer, not startup recovery.
4. Existing terminal rows receive no startup-recovery update attempt. The former read-then-primary-key sweep could overwrite a row terminalized after its read; the no-rewrite scope removes that startup mutation path. This is a source-based race rationale, not a reproduced incident.
5. The ordinary JobRun finalizer, cancellation, HTTP/MCP/test-run, cron, after-sync, and analysis-agent contracts remain unchanged. A non-owner or mismatched cancellation target cannot cancel another process's local owner; an exact owner cancellation/finalization may settle its row normally. Do not infer from a finalizer/Abort error alone that a stored row is still `running`.
6. The distinct channel sync-lease recovery remains unchanged. `Scheduler.Start` invokes it before `loadCronJobs`; a future test must account for its permitted synthetic Channel updates separately from JobRun writes. It must not assert that startup performs no writes of any kind.
7. Fault outcomes are scoped: unknown remote ownership leaves the row untouched and same-job admission blocked; a read/transaction fault in reservation returns `ErrJobAdmission` and creates no run. Do not add a global "no scheduler start on query error" requirement: `loadCronJobs` currently ignores its query error and `Scheduler.Start` continues to `Start` the scheduler. A cron-load query fault is not a successful reservation.
8. No schema/status migration, JobRun lease/fence, cross-process cancellation, operator override, manual clear, age threshold, checkpoint advance, recovery write, provider, channel, notification, credential, network, deployment, or public-readiness change is included.
9. The source-level guarantee applies only to binaries containing the reviewed change. If old and new binaries share a database, an old binary may still execute its sweep. No fleet-wide or deployed claim is allowed without separate sanitized deployment evidence and authority.

## Required call-path acceptance

The primary regression must invoke `NewScheduler` (or an equivalent real scheduler instance), call `Scheduler.Start`, then call `Scheduler.Stop`. Calling `cleanupStuckRuns` alone, or testing only the reservation helper, cannot prove the startup path is hardened.

| ID | Scenario | Required result |
|---|---|---|
| SH-01 | Real `Scheduler.Start` in process-like B while coordinator A holds a live worker/reservation for a persisted `running` row; B uses a fresh coordinator with no local owner for that key | The startup-recovery write trap records zero JobRun update attempts; the row remains `running` while A is held; B's subsequent same-job reservation returns `ErrJobBusy`. This proves persisted admission, not a shared local guard, blocks the second start. |
| SH-02 | Controlled running and terminal rows during real Start, plus a reversible legacy-sweep regression | In the controlled Start window, no startup-attributed JobRun update is attempted and the fixture rows remain equal. Reintroducing the old sweep in a disposable source copy makes the named zero-attempt assertion fail, even when the trap suppresses the write. Equality is not required after a separately controlled legitimate owner finalizer runs. |
| SH-03 | Normal free-job admission and exact cancellation contrasts across coordinators | Coordinator B can reserve an unrelated free job; wrong/non-owner or mismatched cancellation cannot cancel A's live run; A's exact owner cancellation may settle its row through the ordinary finalizer. These expected ordinary writes are attributed outside the startup-recovery observation window. |
| SH-04 | Startup with a synthetic expired eligible channel lease and an ineligible control | Existing R016 channel recovery behavior remains separately bounded and correct; the JobRun trap still records zero startup-recovery writes. No real channel or provider is used. |
| SH-05 | Unknown ownership and read/admission fault controls | A persisted `running` row absent from B's local owner map is treated as unknown/live-remote and remains blocked. A reservation read/transaction fault returns `ErrJobAdmission` with no new run. An injected `loadCronJobs` read error preserves current continuation to scheduler start; it does not require a global no-start policy or count as an admitted run. |
| SH-06 | Cron and after-sync entrypoints and finalization compatibility | Existing focused ownership tests pass; ordinary busy-run, cancellation, and finalization contracts retain their current order and scope. A concurrent legitimate finalizer may write its own row and is not a startup-recovery mutation. |
| SH-07 | Startup job loading and shutdown | Controlled jobs cannot invoke providers or notifications; scheduler shutdown completes within a process/test bound and removes all temporary callbacks, rows, triggers, channels, and fixtures. |

The test fixture must use an isolated disposable synthetic database, synthetic tenant/job/run/channel IDs, separate owner coordinators, and explicit setup/cleanup. Install a side-effect trap that records attempted JobRun update operation and target ID even when it blocks the write; checking only final row state is insufficient. Hold the live owner behind a barrier through the Start assertion window, tag/attribute that interval, and release it only for the separate ordinary-finalizer/cancellation contrast. Control loaded jobs and channel rows so the real start path is exercised without external effects. Bound the test process and scheduler shutdown; remove callbacks, triggers, rows, channels, DB resources, and fixtures in guaranteed cleanup. Existing sync-lease recovery behavior is a separate control, not evidence of JobRun fencing.

All SH scenarios are future acceptance requirements and remain **NOT RUN** in this documentary intake.

Mutation/restoration control: in a disposable checkout or source overlay, snapshot the candidate source bytes and SHA256, then restore the exact pre-change startup-sweep helper and its `Scheduler.Start` call from the recorded source baseline. Assert the expected patch matches, and run the real-Start detector. It must fail at the named attempted-write assertion (a compile error, timeout, or unrelated failure is INCONCLUSIVE; an incomplete/zero-match patch is NOT_APPLIED). Restore in guaranteed cleanup, require byte-for-byte/SHA256 equality to the pre-mutation snapshot and an empty diff, then rerun the restored baseline. Attribute each command and receipt to worker or reviewer; a later reviewer run does not become worker evidence.

## Proposed validation budget

Nonbinding proposal for the future seed:

- Worker: one implementation campaign, at most three focused Go test invocations (restored baseline, applied old-sweep detector, post-restoration baseline) and one backend compile/build.
- Independent reviewer: one review campaign, at most three focused Go test invocations using the same bounded sequence and one backend compile/build against the exact reviewed source snapshot.
- Aggregate ceiling: six focused Go test invocations and two builds. This qualifies the earlier two-test-per-role proposal so the applied detector and restored baseline both fit. No full backend suite, browser, Compose application startup, live provider/channel call, production/persistent database, host inspection, or deployment test.
- A worker repair stays within the same scope and budget. At repair round three without an independent new root cause, record `REVIEW_COST_ESCALATION_REQUIRED` and stop.

Focused tests should include the new real-startup acceptance plus existing `TestCronAndAfterSyncSkipAnOccupiedJob`, `TestCronVersusAfterSyncAdmitExactlyOne`, `TestSchedulerOwnersAreCancellableByRun`, `TestSchedulerRecoversExpiredLeaseAndReadmitsAfterThrottle`, and `TestRecoveryReleasesOnlyExpiredAllowlistedR016Runs`. Any DB prerequisite, timeout, or harness failure is NOT RUN/INCONCLUSIVE for the intended assertion, not a pass. Exact commands, environment, and fixture ownership belong in the future immutable seed.

## Failure and authority boundaries

Fail the future tranche if any JobRun update attempt remains attributable to startup recovery, an age/local-owner/manual bypass is introduced, a running row stops blocking admission, the existing channel lease lifecycle changes without a new contract, the actual `Scheduler.Start` path is not covered, the side-effect trap cannot detect attempted writes, cleanup is unbounded, or evidence is presented as deployed proof. Legitimate owner/cron finalization outside the startup-recovery observation window remains permitted. A need for owner-death detection, durable fencing, recovery status, schema work, or mixed-version deployment support requires a separate design and scope decision. No action in R077 implements or dispatches this draft.

Paired evidence: [repository topology intake](R077_REPOSITORY_TOPOLOGY_INTAKE_2026-10-11.md) and [source identity](probes/r077_topology_identity_2026-10-11.json). Root added these navigation links after first review (M1); substantive proposal remains worker-authored.
