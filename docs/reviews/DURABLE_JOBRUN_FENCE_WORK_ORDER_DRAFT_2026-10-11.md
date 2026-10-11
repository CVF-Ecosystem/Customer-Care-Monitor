# Nonbinding follow-on work-order draft: durable JobRun fence

**Status:** proposal only. No next tranche is active and no source, schema, migration, test, database, runtime, recovery, provider, or deployment authority is granted. This draft is conditional on the selected R080 intended contract and its implementation-readiness limits [SP01]. Source and documentary identities are listed in [I01].

## Objective and proposed sequence

Prepare a separately reviewed R2+ database-fence implementation that persists the per-Job monotonic generation and active grant, binds every run-related DB writer to the exact tenant/Job UUID/run/generation tuple, preserves a terminal notification tail, and adopts every interacting mutation/deletion path. It must keep unresolved grants blocking and make no recovery, provider replay, remote exactly-once, mixed-binary, or present-fleet claim. [SP01]

The lock graph is specified at the intended level in R080, but no migration or DB concurrency evidence exists. The first gate in any later tranche must therefore be a short DESIGN addendum that binds the target MySQL/GORM schema, FK/implicit lock behavior, multirow acquisition order, Tenant-wide coordination, and all Job ID creation/recreation paths to an isolated database acceptance plan. An independent reviewer must accept that addendum before a schema or source implementation begins. If the graph, ID non-reuse rule, or transaction outcome model cannot be specified without relying on unknown deployment facts, stop and return to DESIGN; do not represent implementation as ready. [SP01]

Only after that gate passes may root create a distinct BUILD seed and work order. That later BUILD would implement and test the reviewed contract on an isolated synthetic database. A separate rollout/adoption decision and evidence would be required before any deployed reliance. Root chooses tranche ID, owner, base commit, paths, budgets, and authority in a new immutable seed; this draft does not preselect or create that authority. [SP01][H01]

## Candidate paths for the later documentary gate

These are proposals for root to narrow in a new seed. They authorize no edits in this R080 packet:

- `docs/specs/DURABLE_JOBRUN_FENCE_DATABASE_LOCK_ORDER_ADDENDUM_2026-10-11.md` — target DB assumptions, Job UUID creation/recreation inventory, Tenant/Job/JobRun/Conversation order, implicit FK lock review, failure/commit ambiguity, and exact isolated-test plan.
- `docs/reviews/DURABLE_JOBRUN_FENCE_DATABASE_LOCK_ORDER_REVIEW_2026-10-11.md` — independent acceptance or return of that addendum.
- `docs/reviews/probes/r080_jobrun_fence_design_gate_2026-10-11.json` — exact source/database fixture identities and NOT_RUN boundary for the gate.

The gate may only read versioned source and create a disposable test plan. It must not inspect a production host/database, environment file, credential, container, provider, or deployment. If the repository does not contain enough schema/driver facts, state the missing fact and leave the gate open; do not collect it from a live system under this draft.

## Candidate paths for a separately authorized implementation

The following paths are a bounded initial candidate set; root may narrow or amend it only in the new seed before BUILD. A newly discovered writer is a stop-and-review condition, not permission to widen the set silently.

**Backend candidate paths:**

- `backend/db/models/job.go` and `backend/db/mysql.go` — grant/generation model and migration path.
- `backend/engine/job_run_ownership.go` and `backend/jobdispatch/dispatch.go` — admission, exact identity, cancellation boundary, terminal and retirement helpers.
- `backend/engine/analyzer.go` and `backend/engine/analyzer_incremental.go` — provider/usage/result/progress/finalizer writes and post-terminal release.
- `backend/api/handlers/jobs.go` — create/update/delete/clear paths, including ClearJobResults.
- `backend/api/handlers/channels.go` and `backend/api/handlers/demo.go` — conversation/channel cascades and tenant reset.
- `backend/notifications/dispatcher.go` and `backend/db/activity.go` — terminal tail writes, error/readback handling, and run-attributed activity.
- `backend/cli/prune_results.go` — prune plan revalidation and batch mutation.

**Test candidate paths:**

- New: `backend/engine/job_run_fence_db_test.go` for independent connections, admission, exact generations, stale writers, terminal tail, failure injection, and lock contention.
- New: `backend/api/handlers/job_run_fence_mutations_test.go` for update/delete/clear/channel/purge/reset behavior.
- New: `backend/db/job_run_fence_migration_test.go` for schema creation and repeated migration on a disposable database.
- Existing: `backend/engine/job_run_ownership_test.go`, `backend/cli/prune_results_test.go`, and the notification tests discovered during the gated inventory, with any required new notification test path named in the seed before edits.

**Document/evidence candidate paths:**

- `docs/specs/DURABLE_JOBRUN_FENCE_IMPLEMENTATION_SPEC_2026-10-11.md` — accepted design addendum and reviewed implementation contract.
- `docs/reviews/DURABLE_JOBRUN_FENCE_IMPLEMENTATION_REVIEW_2026-10-11.md` — independent source/test/evidence review.
- `docs/reviews/probes/r080_jobrun_fence_build_evidence_2026-10-11.json` — hashes, test invocations/results, migration fixture identity, fault outcomes, and explicit non-claims.

The later immutable seed and tranche record remain root-owned. A R2 reviewer must be independent of the worker. Root is the proposed ORCHESTRATOR, WORK_ORDER_AUTHOR, independent REVIEWER, CLOSER, and COMMIT_STEWARD; the implementation worker and any repair worker are named only by the new authority. Reviewer acceptance does not establish deployment adoption. [H01]

## Proposed scope and exclusions

If separately seeded, the implementation would:

1. Apply the reviewed migration on a disposable MySQL fixture; keep the Job UUID as a non-reused incarnation and the per-Job generation monotonic. Reject counter overflow and any old Job UUID reuse.
2. Make admission atomically allocate one grant and exact JobRun tuple, then bind every provider-adjacent DB effect, usage record, summary, result/snapshot, terminal/checkpoint, activity, notification log/marker, and grant retirement to that tuple.
3. Implement the reviewed Tenant → Job → JobRun → Conversation → child-row partial order, including stable multirow sorting and the bounded tenant-wide exclusion behavior for channel/purge/reset operations. No transaction may acquire an earlier lock class after a later class.
4. Make DeleteJob, ClearJobRuns, ClearJobResults, UpdateJob, DeleteChannel, PurgeChannelConversations, ResetDemoData, and prune reject or serialize against any active grant, including terminal-tail grants. Use checked transaction/readback outcomes; uncertain outcomes keep exclusion.
5. Keep provider/notifier requests synchronous with the fenced worker lifecycle where source allows, record known outcomes, and never resend an ambiguous external request. Do not claim the DB fence cancels or exactly-once protects external services.
6. Preserve local cancellation-vs-terminal semantics and checkpoint rules. Do not add cross-process cancel, automatic owner-death recovery, lease expiry, TTL, age-based reclaim, startup sweep, or grant-clearing fallback.

Excluded: production or user database access; deployed schema migration; host/container inspection; provider/network calls; OA account work; automatic recovery or row rewrite; external lock service; remote exactly-once/replay guarantees; mixed-binary rollout; deploy/release; public CVF governance claim. A governance claim, if separately in scope later, still requires the real provider evidence specified by project policy; synthetic provider output is not that evidence. [SP01]

## Proposed evidence and acceptance

All evidence must identify the tested commit, migration/schema fixture, DB engine/version/isolation used, test command, result, and failure injection. Use an isolated disposable database with synthetic data and independent connections/processes. The candidate tests are not claimed executed here.

- Gate review accepts the exact lock graph, source writer inventory, fresh Job UUID rule, and migration behavior before BUILD. Unknown production DB facts stay UNKNOWN.
- A same-Job second admission is rejected while one grant is active; an unrelated Job remains admissible. Generation strictly increases; a recreated Job has a new UUID; wrong tenant/job/run/generation is a no-op.
- Every mapped writer rejects stale identity in both running and terminal states, except the narrow allowed terminal tail. Results and metadata do not change on rejection.
- Admission, writer, finalizer, tail marker, and retirement faults cover statement error, rollback, lost/ambiguous commit, missing row, and readback mismatch. Any uncertain state retains the grant and blocks another admission.
- Cancellation, terminal winner, test-run, partial/error result, and clean ordinary checkpoint cases preserve their documented outputs.
- DeleteJob, ClearJobRuns, ClearJobResults, UpdateJob, DeleteChannel, PurgeChannelConversations, ResetDemoData, and prune are raced against active and terminal-tail grants. Require no partial delete, no deadlock, no orphan, and the intended busy/no-op outcome. Probe actual multirow/FK locks in the selected fixture rather than inferring them from source order.
- Lost provider/notifier response after dispatch produces no automatic replay and no claim that the remote service did nothing. Test doubles may exercise only application-path behavior; they do not constitute CVF governance proof.
- Mixed or old binaries are not covered by repository tests. A later operational adoption packet must identify every writer/binary and show upgrade or quiescence before any deployed reliance. No fleet-wide guarantee may be inferred from a clean test run.

## Stop conditions and preservation

Stop before schema/source work if the independent DESIGN gate cannot resolve the DB lock graph, implicit FK interactions, Job UUID non-reuse, generation overflow, active-grant retirement, or ambiguous-commit behavior. Stop and return to DESIGN if any path bypasses the tuple, a lock inversion/deadlock appears, migration changes existing data unexpectedly, an ambiguous effect would be replayed, a failed readback clears exclusion, or a newly discovered writer lies outside the seed. Do not widen paths or environment effects during BUILD.

Use only disposable synthetic DB state; no production data or credentials. Preserve a pre-migration fixture and test logs. Never run a destructive down migration against a database containing an active grant. Any cleanup is confined to the disposable fixture after evidence capture. Failed tests and first findings remain in the review/evidence record. No worker self-approval; root performs one consolidated independent review before any repair and records root assistance and worker repairs separately. Repair-round escalation follows the project work-order policy. [H01]

## Budget proposal and failure disposition

This proposal has no fixed model-time, token, or cost estimate. The future seed should set a bounded test/build budget after the target fixture is selected. It should prohibit production DB/host/container/network/provider use and runtime deployment; permit only the isolated DB fixture and specified local commands. If an evidence path requires a prohibited effect, stop and ask root to prepare a new authority rather than infer permission from this draft.

Acceptance of this draft would mean only that root may consider a new design gate and later authority. It would not mean the database contract passed, code is ready, a migration ran, a grant is durable in any environment, or the project has fleet/runtime governance proof. [SP01][I01]

### References

[I01]: probes/r080_source_identity_2026-10-11.json
[SP01]: ../specs/DURABLE_JOBRUN_FENCE_DESIGN_SPEC_2026-10-11.md#L1-L158
[H01]: R079_SCOPED_PREREQUISITE_INTAKE_CLOSURE_2026-10-11.md#L1-L11
