# Dispatch recovery requirements — R076

Status: intended behavior only; documentary repair submitted; independent
review pending. Current source truth is summarized separately from these
requirements. R076 authorizes only this document, the paired decision, and the
source identity record.

Paired artifacts: [decision](../decisions/DISPATCH_RECOVERY_R076_2026-10-11.md)
and [source identity record](../reviews/probes/r076_source_identity_2026-10-11.json).

## Scope

Specify a safe future response to an uncertain or stranded JobRun. Do not
implement recovery, change the existing startup sweep, run tests, access a
database, or dispatch application/provider work under this tranche.

Future recovery must preserve tenant isolation, the Job to JobRun lock order,
run/result/receipt/history identity, cancellation ordering, and current
entrypoint semantics. A running row remains unresolved while any possible
writer has not been demonstrably fenced or quiescent. Age, missing local
owner, process-local inactivity, or a human request alone is insufficient.

## Current source truth

1. backend/engine/scheduler.go:97-128 is an existing startup sweep. It queries
   every row with status running and attempts to mark each failed. It has no
   explicit still-running predicate in the update, no Job parent lock, no
   owner/fence check, and no post-write readback. backend/main.go:69-76 starts
   this scheduler in the application entrypoint. This behavior is not a safe
   recovery proof and remains an open hardening/compatibility issue.
2. backend/engine/job_run_ownership.go:61-89 keeps owner registry and token
   in memory per process. Admission locks the tenant-scoped Job before
   checking/inserting the run (lines 107-190); the persisted row model
   backend/db/models/job.go:48-60 has no owner, lease, heartbeat, or fencing
   field. Separate process coordination is therefore UNKNOWN from these
   sources.
3. backend/jobdispatch/dispatch.go:66-148 validates, reserves, and then
   starts a bounded goroutine. Successful dispatch means reserved plus worker
   handoff. Start failure invokes an abort attempt. Dispatcher and Abort
   comments say a failed abort leaves a running row blocking admission
   (backend/jobdispatch/dispatch.go:67-70, 117-122;
   backend/engine/job_run_ownership.go:356-361, 386-390). The called path
   invokes the shared close/finalize path; after checked transaction failures,
   `finalizeOrdinaryRun` can best-effort update the exact still-running row to
   `error` without advancing the checkpoint, then return an error
   (backend/engine/job_run_ownership.go:356-397;
   backend/engine/analyzer_incremental.go:263-284). The fallback does not
   check affected-row count or read back the row, so an abort error alone does
   not establish that the stored row is still `running`; the fallback may have
   persisted `error`, matched no row, or failed. A crash around reservation
   or handoff can also leave the execution stage uncertain.
4. backend/engine/job_run_ownership.go:258-306 and
   backend/api/handlers/jobs.go:571-603 preserve exact local cancellation:
   the current owner is signalled after checked reads; a running row without
   a local owner is classified ErrRunNotOwned. Recovery must not turn this
   into remote cancellation or steal a live owner's run. In ordinary
   execution, `executeReserved` defers `owner.release()` until after all
   function effects, including the notification call, have returned
   (backend/engine/analyzer.go:53-72, 411-420); `release` removes only the
   matching process-local owner (backend/engine/job_run_ownership.go:193-204).
   That local lifecycle does not prove remote or restarted writers are
   quiescent.
5. backend/engine/analyzer_incremental.go:167-284 has checked terminal
   transactions, exact running-row validation, readback, and a best-effort
   exact-row error fallback with no checkpoint update. backend/engine/analyzer.go:343-420
   commits terminal state before notification; ordinary checkpoint advancement
   requires a complete, error-free ordinary run. A terminal-write error can
   therefore have a fallback or an already-committed terminal state; it is
   not synonymous with a stranded running row.
6. backend/engine/analyzer.go:245-329 can call the provider and write usage
   and results before finalization. Notifications follow finalization
   (backend/engine/analyzer.go:411-424;
   backend/notifications/dispatcher.go:38-124). A crash can leave partial
   effects or notification uncertainty. No exactly-once provider or delivery
   promise is supported by this source audit.
7. HTTP TriggerJob accepts the current validated modes, date bounds, and
   decimal cap (backend/api/handlers/jobs.go:402-452, 504-544). MCP accepts
   tenant_id and job_id only and starts since_last with no dates and cap zero;
   extra arguments are rejected (backend/mcp/handlers.go:330-360;
   backend/mcp/tools.go:159-168). The accepted MCP contract is also recorded
   in docs/specs/MCP_JOB_EXECUTION_R044_2026-10-03.md:14, 22-28.
8. Cron and after-sync call Analyzer.RunJob (backend/engine/scheduler.go:279-299,
   318-363); analysis agents call the same Analyzer entry point
   (backend/api/handlers/agents.go:265-310). HTTP test-run uses a separate
   reservation/start route. Channel sync has a separate lifecycle. Future
   recovery must not silently change these seams.
9. The model status comment lists running/success/error
   (backend/db/models/job.go:48-60), while current Analyzer source also emits
   partial and cancelled and the startup sweep writes failed
   (backend/engine/analyzer.go:363-376;
   backend/engine/scheduler.go:115-118). Recovery status taxonomy and
   compatibility are OPEN; R076 chooses no new or replacement status.

Source citations refer to implementation baseline
9fc86fe012d1cc8c1aeb7d9e7a55e79c1c032c2d. The worker compared each cited
file's Git blob at that baseline and documentary HEAD
f92a80a78e7c87424792338fc8cec4634f42aff8, and computed physical SHA256 values
in [the source identity record](../reviews/probes/r076_source_identity_2026-10-11.json).
This identifies the source snapshot only; it does not certify a deployed
build.

## Intended recovery contract

- Eligibility: do not act on age or missing local ownership. Require evidence
  that every process that could write the exact run is fenced or quiescent.
  For a recovery decision, establish this exclusion before relying on the row
  observation and maintain it continuously through inspection, any
  conditional transition, and exact persisted readback/classification. This
  must exclude old, restarted, and newly admitted writers. If the proof is
  stale, lost, or uncertain at any point, stop; do not claim recovery or
  permit a replacement admission until a safe exclusion barrier is
  re-established. A process-local no-owner observation is insufficient.
- Identity: bind every inspection and any future mutation to the exact
  tenant_id, job_id, and run_id. A job-level or global sweep is not sufficient.
- Serialization: use the existing Job then JobRun lock order. A later
  implementation must condition on the exact row still being running and on
  the separately designed durable fence/owner generation. The current source
  has no persisted fence field; whether a schema migration is needed is OPEN.
- Checked outcome: verify update count and read back the exact persisted row
  and related job state. If the read, write, commit, or readback is ambiguous,
  do not retry the transition, claim success, advance the checkpoint, or
  start another worker. Keep the case unresolved for a separate reconciliation
  decision.
- Partial effects: never infer that provider work, usage, result writes, or
  notifications did not happen merely because the row remains running.
  Recovery itself must not call providers, publish results, send notifications,
  or automatically restart execution.
- Terminal/history: preserve the original run_id, results, snapshots,
  evaluations, summaries, usage observations, notification logs, and
  notified_at values. A verified terminal row is a no-op. No destructive
  cleanup or replacement run is part of recovery.
- Checkpoint: recovery never advances last_run_at. Only the existing ordinary
  Analyzer completion path may do so after a complete, error-free run.
- Cancellation: preserve current local exact-run cancellation and terminal
  serialization. A process-local owner cannot prove the status of a remote
  writer. Cross-process cancellation and recovery interaction remain OPEN.
- Idempotency and retries: no recovery-trigger idempotency key or persisted
  owner generation is present in the inspected source. Do not promise
  exactly-once execution/provider calls or notification delivery. A later
  user-triggered run is a distinct action after the blocking state is settled;
  it is not an automatic retry of the prior run.

## Acceptance matrix for a separately authorized implementation

Every row below is a future acceptance requirement. All runtime evidence is
NOT_RUN in R076.

| ID | Scenario | Required result |
|---|---|---|
| AR-01 | Startup with a running row and no proof about other instances | Recovery makes no state change; admission remains blocked. The existing scheduler sweep is explicitly covered by the future work order. |
| AR-02 | Row exceeds any proposed age threshold, or local JobRunActive is false | No reclaim. Tests must include a live writer in a second coordinator/process sharing an isolated database. |
| AR-03 | Exact old writer is fenced and a valid exclusion barrier remains continuously held from exact-row inspection through checked transition and verified readback | Only a separately approved terminal reconciliation may run, using the established fence and conditional identity check; tests also prove old, restarted, and newly admitted writers stay excluded for the whole interval. No execution restart or checkpoint advance. If no durable fence exists, expected result remains blocked. |
| AR-04 | Same job has a different run, or another tenant has a running row | Only the exact authorized row may be considered. Wrong-run and foreign-tenant rows remain byte/state unchanged. |
| AR-05 | Success, error, partial, cancelled, or other terminal state is already stored | No rewrite, duplicate run, history deletion, notification replay, or checkpoint update. Current status taxonomy must first be reconciled. |
| AR-06 | Crash after reservation but before worker handoff; or after partial provider/result effects but before terminal write; old process restarts or a new writer attempts admission | Do not infer that no work occurred. Without fence and outcome proof, keep unresolved; never mark success or auto-retry. The old/restarted/new writer must remain excluded while recovery is considered. |
| AR-07 | Terminal commit/readback outcome is uncertain, or the exclusion proof is lost during an attempted transition | Do not claim recovery or start another worker. Re-establish safe exclusion before exact-row reconciliation; a verified terminal is preserved and remaining ambiguity fails closed without a second transition. Preserve the existing finalizer fallback as a distinct source outcome: it may persist `error`, but its error return alone does not establish stored state. |
| AR-08 | Terminal run exists but notification send or notification-state write is uncertain | Do not rerun analysis or mark delivery successful from inference. Preserve existing records and handle delivery as a separately specified concern. |
| AR-09 | Cancellation races with terminalization or recovery | Exact local cancellation continues to serialize with terminal commit. Recovery cannot steal the locally owned run; remote-owner uncertainty blocks mutation. |
| AR-10 | HTTP, MCP, test-run, cron, after-sync, agent, and sync entrypoints | Existing mode/default/error and ownership contracts remain intact. MCP remains since_last with empty dates/cap zero; HTTP parameter validation and the separate sync lifecycle remain unchanged. |
| AR-11 | Fault/concurrency probe drops, expires, or makes the writer-exclusion proof stale at controlled points before inspection, before transition, and before readback; paired positive probe holds a valid barrier throughout | The positive case may reach only its separately authorized checked result while all competing writers remain excluded. A proof loss before mutation causes zero writes; loss during/after a possible transition cannot be reported as success, retried, or opened to admission, and requires safe re-establishment plus exact reconciliation. The test must distinguish both outcomes rather than treating a generic error as proof of fencing. |

## Named negative controls and evidence handling

The future campaign must include these discriminating controls:

- NC-01 Age-only reclaim: remove the fence requirement; the old active writer
  control must fail the intended assertion.
- NC-02 Local-owner absence: treat an empty local registry as proof of death;
  the remote-owner case must fail.
- NC-03 Scope/conditional-write removal: remove tenant, job, run, or
  still-running predicates, or violate Job then JobRun locking; wrong-run,
  cross-tenant, and terminal-race controls must fail.
- NC-04 Outcome promotion: force uncertain/partial execution to success,
  checkpoint advancement, or automatic worker retry; the fault case must fail.
- NC-05 Notification replay: retry or mark delivery after an uncertain
  terminal/notification gap; the notification boundary control must fail.
- NC-06 Continuous exclusion: remove, expire, or ignore the fence between
  inspection, transition, and readback. The stale/lost-proof case must fail
  the intended assertion, while the positive case retains the same valid
  barrier throughout and rejects old, restarted, and newly admitted writers.

For each applied mutation, record the exact source and test snapshot hashes,
expected and actual change count, patch/digest, command and exit, the named
assertion that failed, byte-for-byte restoration, and a restored-baseline
result. Use an isolated disposable data store and synthetic rows only; cleanup
must verify that all temporary resources are gone. A compile error, timeout,
no-op mutation, or unrelated failure is INCONCLUSIVE, not a behavioral kill.
Any future claim that CVF governs AI behavior still requires a real provider
API request/response receipt. No such claim is in scope here.

## Recommended follow-on and boundary

First candidate: a separate read-only topology/owner-fence diagnostic that
enumerates all relevant instances and verifies which process, database,
scheduler-start, and fencing facts are observable. Prefer sanitized
operator-supplied evidence. If it needs production host/database inspection,
its risk floor is R2 and it requires a new immutable seed. It performs no row
writes, dispatch, provider, channel, or notification calls. UNKNOWN is a valid
result and does not resolve or make safe the current startup sweep.

Priority after that diagnostic: a separate R2+ design/implementation work
order must address cleanupStuckRuns before adding recovery behavior. Candidate
path class is the scheduler, engine ownership/finalization, JobRun model and
any required migration, focused tests, and scoped evidence. Exact paths,
status semantics, architecture, and risk ceiling belong in that later seed.
Proposed roles: root ORCHESTRATOR/WORK_ORDER_AUTHOR, one named
IMPLEMENTATION_WORKER, and an independent REVIEWER. Proposed budget: one
bounded implementation campaign with a restored-source pass; no automatic
replay or third repair without a new cost disposition. Runtime/provider calls
remain zero unless a later contract separately requires external evidence.

Failure boundaries: unknown topology or fence; stale/lost/uncertain exclusion
proof at any point; nonunique or mismatched tenant/job/run identity;
conditional write count mismatch; failed or ambiguous commit/readback;
evidence of partial effects; or an outcome status that cannot be represented
honestly. Any such case leaves the row unresolved, does not advance
last_run_at, and stops the operation; replacement admission remains blocked
until a safe barrier is re-established and the exact state is reconciled.

## DR01–DR10 coverage and authorship

| Order requirement | Location in this packet |
|---|---|
| DR01 current source facts and exact citations | Current source truth; source identity JSON |
| DR02 alternatives and bounded recommendation | Recommended follow-on; paired R076 decision |
| DR03 live/remote/fenced owner, effect, terminal, and crash states | Intended recovery contract; AR-01 to AR-09 and AR-11 |
| DR04 continuous fencing, exact conditional write, uncertain outcomes | Eligibility, Identity, Serialization, Checked outcome, AR-03 and AR-11 |
| DR05 idempotency, retries, cancellation, and entrypoint compatibility | Intended contract; AR-09 and AR-10 |
| DR06 intended, implemented, and unknown separation | Status and source truth headings |
| DR07 positive/fault/concurrency/tenant controls and NOT_RUN evidence | AR-01 to AR-11; NC-01 to NC-06 |
| DR08 prerequisites, candidate paths, R2+ floor, roles/budget/failure | Recommended follow-on and failure boundaries |
| DR09 source hashes and R074 limits; no runtime or HEAD certification | Source identity reference and scope notes |
| DR10 exact artifact set and independent review | Exact three-file repaired packet; first-pass F1/F2/M1 and repair round 1 recorded in the identity record; root reviewer reserved |

Worker ownership is limited to this spec, its [paired decision](../decisions/DISPATCH_RECOVERY_R076_2026-10-11.md), and
[source identity record](../reviews/probes/r076_source_identity_2026-10-11.json). Root owns metadata
and commits and remains the independent reviewer. Root supplied the startup
sweep and finalizer-fallback source pointers; the worker independently
verified them. First-pass CHANGES_REQUIRED disposition, assistance
attribution, and one documentary repair round are recorded in the identity
record. No source, test, application
runtime, database, provider, network, credential, or core operation was
performed.
