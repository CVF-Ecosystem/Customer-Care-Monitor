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

1. On startup of the hardened binary, every existing JobRun row remains byte/state-identical. In particular, a `running` row gets no status, finish-time, error-message, summary, result, receipt, or checkpoint rewrite, regardless of age or local owner-map state.
2. The same `running` row continues to block another reservation through existing tenant/job database admission. No new run row, provider call, result publication, notification, or automatic retry is started by the hardening.
3. Existing terminal JobRuns remain unchanged. The startup path performs no JobRun update attempt, including when a concurrent terminal transition could race an older read-then-write sweep.
4. The ordinary JobRun finalizer, cancellation, HTTP/MCP/test-run, cron, after-sync, and analysis-agent contracts remain unchanged. Do not infer from a finalizer/Abort error alone that a stored row is still `running`.
5. The distinct channel sync-lease recovery remains unchanged. `Scheduler.Start` currently invokes it before `loadCronJobs` and the JobRun sweep. A future test must account for its allowed synthetic Channel updates separately from JobRun writes; it must not assert that startup performs no writes of any kind.
6. No schema/status migration, JobRun lease/fence, cross-process cancellation, operator override, manual clear, age threshold, checkpoint advance, recovery write, provider, channel, notification, credential, network, deployment, or public-readiness change is included.
7. The source-level guarantee applies only to binaries containing the reviewed change. If old and new binaries share a database, an old binary may still execute its sweep. No fleet-wide or deployed claim is allowed without separate sanitized deployment evidence and authority.

## Required call-path acceptance

The primary regression must invoke `NewScheduler` (or an equivalent real scheduler instance), call `Scheduler.Start`, then call `Scheduler.Stop`. Calling `cleanupStuckRuns` alone, or testing only the reservation helper, cannot prove the startup path is hardened.

| ID | Scenario | Required result |
|---|---|---|
| SH-01 | Real `Scheduler.Start` with a synthetic `running` JobRun and a second reservation attempt | The row's full persisted state is unchanged; a JobRun-write trap records zero attempted updates; the reservation remains blocked. |
| SH-02 | Real start with terminal and unrelated tenant/job rows, plus a controlled old-sweep regression | No JobRun update is attempted and no row is rewritten. Reintroducing the old startup sweep makes the test fail on the attempted write, even if a test trigger suppresses the mutation. |
| SH-03 | Startup with a synthetic expired eligible channel lease and an ineligible control | Existing R016 channel recovery behavior remains separately bounded and correct; the JobRun trap still records zero writes. No real channel or provider is used. |
| SH-04 | Cron and after-sync entrypoints, cancellation, and finalization compatibility | Existing focused ownership tests pass; a busy run remains blocked and cancellation/finalization retain their current order and scope. |
| SH-05 | Startup job loading and shutdown | Controlled jobs cannot invoke providers or notifications; scheduler shutdown completes within the test bound and removes all temporary callbacks, rows, triggers, and fixtures. |

The test fixture must use an isolated disposable synthetic database, synthetic tenant/job/run/channel IDs, and explicit setup/cleanup. Use a database/GORM side-effect trap that counts attempted JobRun updates; checking only final row state is insufficient if a trigger can suppress a write. Control loaded jobs and channel rows so the real start path is exercised without external effects. Bound the test process and scheduler shutdown; confirm temporary DB resources and traps are removed. Existing sync-lease recovery behavior is a separate control, not evidence of JobRun fencing.

## Proposed validation budget

Nonbinding proposal for the future seed:

- Worker: one implementation campaign, at most two focused Go test invocations and one backend compile/build.
- Independent reviewer: one review campaign, at most two focused Go test invocations and one backend compile/build against the exact reviewed source snapshot.
- Aggregate ceiling: four focused Go test invocations and two builds. No full backend suite, browser, Compose application startup, live provider/channel call, production/persistent database, host inspection, or deployment test.
- A worker repair stays within the same scope and budget. At repair round three without an independent new root cause, record `REVIEW_COST_ESCALATION_REQUIRED` and stop.

Focused tests should include the new real-startup acceptance plus existing `TestCronAndAfterSyncSkipAnOccupiedJob`, `TestCronVersusAfterSyncAdmitExactlyOne`, `TestSchedulerOwnersAreCancellableByRun`, `TestSchedulerRecoversExpiredLeaseAndReadmitsAfterThrottle`, and `TestRecoveryReleasesOnlyExpiredAllowlistedR016Runs`. Any DB prerequisite, timeout, or harness failure is NOT RUN/INCONCLUSIVE for the intended assertion, not a pass. Exact commands, environment, and fixture ownership belong in the future immutable seed.

## Failure and authority boundaries

Fail the future tranche if any JobRun startup update remains, an age/local-owner/manual bypass is introduced, a running row stops blocking admission, the existing channel lease lifecycle changes without a new contract, the actual `Scheduler.Start` path is not covered, the side-effect trap cannot detect attempted writes, cleanup is unbounded, or evidence is presented as deployed proof. A need for owner-death detection, durable fencing, recovery status, schema work, or mixed-version deployment support requires a separate design and scope decision. No action in R077 implements or dispatches this draft.

Paired evidence: [repository topology intake](R077_REPOSITORY_TOPOLOGY_INTAKE_2026-10-11.md) and [source identity](probes/r077_topology_identity_2026-10-11.json). Root added these navigation links after first review (M1); substantive proposal remains worker-authored.
