# Dispatch recovery design decision — R076

Status: worker first pass; independent review pending. Date: 2026-10-11.
Authority: CCMAI-RUNTIME-076, R1 source-only DESIGN/SPEC. This record does not
authorize application changes or recovery operations.

## Decision

1. A running JobRun is not eligible for recovery based on its age or because
   the current process has no local owner. Until all possible old writers are
   demonstrably fenced or quiescent, leave the run unresolved and keep normal
   admission fail-closed.
2. The repository already has a startup status sweep. Treat it as an existing
   safety and compatibility concern, not as proof that recovery is safe. R076
   does not change or endorse that sweep.
3. If a later, separately authorized recovery change is proposed, it must bind
   to one tenant, job, and run identity; retain Job then JobRun lock order; use
   checked conditional writes and readback; and stop on uncertain outcomes.
   Recovery must not turn partial or uncertain execution into success, retry
   provider work, replay notifications, or advance the ordinary checkpoint.
4. Preserve the current HTTP, MCP, test-run, cron, after-sync, agent, and
   cancellation contracts until a separately reviewed specification changes
   them. A returned run_id means admission and worker handoff, not completion.
5. No queue, lease format, persisted fencing field, interrupted status, or
   automatic retry policy is selected here. Those decisions depend on
   deployment topology and old-writer exclusion evidence that are currently
   unknown.

## Current source facts

- backend/main.go:69-76 starts the scheduler. Scheduler.Start calls
  cleanupStuckRuns at startup (backend/engine/scheduler.go:97-101). On a
  successful query, cleanupStuckRuns selects every row whose status is
  running, then attempts to set each loaded model to failed with a finish time
  and restart message (backend/engine/scheduler.go:104-128). The update call
  has no explicit still-running, tenant, or job predicate; this helper does
  not take the Job parent lock, consult an owner/fence, or read back the
  result. Query/update failures are logged and skipped.
- Admission also has process-local ownership keyed by tenant/job and a
  database transaction that locks the Job parent before checking for running
  JobRuns and inserting a new run (backend/engine/job_run_ownership.go:61-89,
  107-190). Its documented lock order is Job then JobRun. The persisted
  JobRun fields are id, job id, tenant id, start/finish times, status, summary,
  error, and created time; the owner token remains in the process-local
  structure, not the row (backend/db/models/job.go:48-60;
  backend/engine/job_run_ownership.go:76-98). This does not establish
  cross-process fencing.
- The shared dispatcher validates configuration, reserves the persisted row,
  then starts a goroutine. A synchronous start failure attempts an abort; a
  failed abort leaves the row running and blocking admission
  (backend/jobdispatch/dispatch.go:66-148). Reservation precedes worker
  handoff, so a crash in that interval cannot be classified from the current
  JobRun row alone.
- Cancellation signals only the locally registered owner after a checked
  exact-row read. When no local owner exists but a stored running row does,
  CancelJobRun returns ErrRunNotOwned and writes nothing
  (backend/engine/job_run_ownership.go:258-306; backend/api/handlers/jobs.go:571-603).
- Ordinary finalization makes up to three checked transactions, locks Job
  then the exact running JobRun, writes terminal state plus job/checkpoint
  state, and verifies both. After failures it makes a best-effort update of
  the exact still-running row to error without touching the checkpoint
  (backend/engine/analyzer_incremental.go:167-284). Therefore the source does
  not support a blanket claim that every failed terminal write leaves a row
  running.
- Provider calls and per-conversation result/usage writes can precede the
  terminal transaction; notification dispatch follows terminal persistence.
  Notification send/log/readback failures are not all surfaced at their shown
  call sites (backend/engine/analyzer.go:245-329, 343-420;
  backend/notifications/dispatcher.go:38-124). A partial external effect or a
  terminal/notification gap is not evidence that replay is safe.

The source statements above are bound to implementation commit
9fc86fe012d1cc8c1aeb7d9e7a55e79c1c032c2d and the R076 identity snapshot
records identical blobs at documentary HEAD f92a80a78e7c87424792338fc8cec4634f42aff8.
The file-by-file SHA256 and line map are in
docs/reviews/probes/r076_source_identity_2026-10-11.json.

## Source fact, inference, proposal, and unknown

- Source fact: the application entrypoint starts a scheduler whose startup
  path attempts to mark all selected running rows failed. The path does not
  establish that an old writer has stopped.
- Conditional inference: if multiple app instances share the same database,
  one instance's startup sweep could race a still-active writer in another
  instance; after the row changes state, a new admission may be possible.
  This is a code-level risk under that topology, not a reproduced incident or
  a claim about any deployment.
- Proposed requirement: no new recovery write may occur until old-writer
  exclusion is demonstrated for every process that can write the run, and the
  exact row identity and terminal outcome are checked.
- Unknown: deployed instance count and startup behavior; database sharing,
  dialect and isolation; whether any durable lease/fence exists outside this
  source; whether a writer is alive remotely; whether a provider, result, or
  notification side effect occurred; and how ambiguous commit outcomes are
  handled in a deployed environment.

R074 remains FROZEN for its accepted source-audit dossier. Its statement that
deployed crash recovery and multi-instance behavior are unverified remains
true. R076 adds the repository startup-sweep path that the prior audit did not
cover; the new source fact does not establish deployed recovery or invalidate
R074's historical scope. See R074 source audit lines 95-107, its independent
review lines 7-22, and scoped closure lines 5-13.

## Options considered

| Option | Decision | Reason |
|---|---|---|
| Reclaim by age or missing local owner | Reject | Neither proves remote owner death, quiescence, or absence of partial effects. |
| Manual clear of a running row | Reject as a bypass | Human action alone does not fence a writer or settle terminal uncertainty. |
| Keep a row blocking when ownership is unknown | Adopt as the proposed safe default | Preserves fail-closed admission while topology is unknown; may require a later support workflow. |
| Add a durable queue, lease, or fencing token | Defer | No deployed topology or lease contract is known; this is a separate architecture and schema decision. |

## Recommended next slice and priority

The next candidate is a separately seeded, read-only topology and owner-fence
diagnostic. Prefer an operator-supplied sanitized deployment inventory. If
live host or production database metadata is required, classify it at least
R2 and authorize it in a separate work order. It may enumerate all instances
that can share the database, scheduler startup paths, database lock/isolation
behavior, and the evidence available to fence an old writer. It must make no
JobRun or Job write, start no run, and call no provider, channel, or notifier.
If any writer-exclusion fact remains unknown, it returns UNKNOWN and leaves
the row unresolved.

This diagnostic does not remedy cleanupStuckRuns. The highest-priority
follow-on is a separately reviewed R2 source/design decision for that existing
startup sweep and its compatibility behavior, before adding a second recovery
path. Any later runtime or persisted-state implementation is at least R2 and
needs its own seed, exact allowed paths, independent reviewer, bounded budget,
and failure rules. Candidate paths may include the scheduler, engine
ownership/finalization, JobRun model/migration, and focused tests; no such
paths or implementation are authorized by R076.

## Status and review boundary

Current implementation includes the startup sweep described above. A safe
cross-process recovery protocol, old-writer fencing, recovery outcome status,
and diagnostic workflow are NOT IMPLEMENTED or verified here. Application
tests, runtime, provider/channel/database operations, and live governance
proof are NOT RUN. This packet makes no production, delivery, exactly-once, or
CVF AI-governance claim. Facebook and Zalo OA account work remains parked.

Root supplied source pointers for the startup sweep and finalizer fallback
nuance; the worker independently read and hashed those sources. This is
first-pass assistance attribution, not an independent review finding.
