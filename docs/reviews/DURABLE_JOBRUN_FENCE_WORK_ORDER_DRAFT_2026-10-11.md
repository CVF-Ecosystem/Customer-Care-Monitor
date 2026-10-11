# Nonbinding follow-on work-order draft: durable JobRun fence

**Status:** proposal only. No next tranche is active and no code, schema, test, database, runtime, recovery, provider, or deployment authority is granted. This draft follows the selected logical contract in R080 but separates architecture selection from implementation readiness. Source and documentary identities are listed in [I01].

## Immediate proposed move: bounded documentary design gate

The selected direction is a DB-owned monotonic per-Job generation and active grant. R080 now specifies phase-specific outcomes for ambiguous admission, writer, terminal, and retirement transactions, plus the compatible local registry/publication/cancellation mutex protocol. The remaining concrete implementation-readiness blocker is binding those logical rules to the repository's MySQL/GORM transaction and lock behavior: row-lock and commit/readback semantics, implicit foreign-key and multirow edges, every Job create/recreate path, and the Tenant-wide versus per-Job mutation paths. The reviewed source comparison is not evidence of a deployed engine/version or runtime topology. [SP01]

The only immediate next proposal is one R2 documentary design gate before any schema/source BUILD. It would resolve the lock and identity inventory from versioned source and specify the exact isolated test plan, or leave an explicit unresolved item and stop. It may not inspect a live database, host, environment file, container, credential, provider, or deployment. If the source does not establish a required fact, the gate records it as unresolved; it does not infer a value from deployment. [SP01]

**Proposed gate roles and ceiling:** a worker named only by a future seed authors the addendum; an independent R2 reviewer accepts or returns it; root owns orchestration, the work order, review, closure, and any commit. The bounded gate would permit one documentary author pass and one independent review pass, at most the three artifacts below, and zero application-native, Go build/test, DB, host/container, provider, or network invocations. Only local source/document identity, citation, and static plan checks are proposed. This is a ceiling proposal, not current authority; the future immutable seed must repeat or narrow it before any work.

**Candidate gate artifacts, to be narrowed in a future seed:**

- `docs/specs/DURABLE_JOBRUN_FENCE_DATABASE_LOCK_ORDER_ADDENDUM_2026-10-11.md` — exact transaction phases, authoritative row observations, owner/registry mutex edges, target fixture assumptions, Job UUID create/recreate inventory, lock graph, and isolated acceptance plan.
- `docs/reviews/DURABLE_JOBRUN_FENCE_DATABASE_LOCK_ORDER_REVIEW_2026-10-11.md` — independent acceptance or return of that addendum.
- `docs/reviews/probes/r080_jobrun_fence_design_gate_2026-10-11.json` — source identities, per-writer coverage, proposed fixture and tests, and explicit `NOT_RUN` boundary.

The gate is accepted only if it consistently resolves: (1) an ambiguous admission can lead either to an observed active grant or an atomic rollback, with no dispatch before proof and no automatic retry; (2) ambiguous retirement can lead either to an observed active grant or a retired terminal row only after old synchronous DB writers are drained; (3) exact-tuple DB writers and retirement serialize on the same Job lock; (4) the registry mutex is never held across a DB wait, publication may retain the owner mutex through its DB transaction, cancel reads before taking that mutex, and no DB-held path takes the owner mutex; (5) cancellation and terminal winner behavior remains compatible with R076; (6) every listed source writer, destructive path, and Job ID create/recreate route appears in the graph. If any item remains technically unresolved, the gate stays open and source implementation does not begin. External requests still have no exactly-once, cancel, or replay guarantee. [SP01]

The gate's later acceptance plan must discriminate healthy and faulted outcomes for independent same-Job admission, unrelated Jobs, generation increase/non-reuse, each DB writer before and after terminalization, each mutation/deletion race, cancellation versus publication/terminalization, admission/writer/terminal/retirement commit loss and readback, restart, and old binaries. The plan names observable rows, grants, checkpoint, notifications, and no-op/busy results. Every test remains `NOT_RUN` at the documentary gate. [SP01]

## Conditional implementation outline (not executable authority)

Only after the documentary gate is independently accepted may root consider a separately seeded R2+ implementation order. The seed must name the worker, independent reviewer, base commit, exact paths, fixture, finite test/build ceilings, evidence, stop conditions, and restore process. It must include every writer below or narrow the slice without claiming all-writer safety. A separate rollout/adoption decision must identify and upgrade or quiesce old binaries and out-of-repository writers before any deployed reliance. [SP01][H01]

**Candidate source paths for a future seed:**

- `backend/db/models/job.go` and `backend/db/mysql.go` — logical grant fields and migration behavior.
- `backend/engine/job_run_ownership.go` and `backend/jobdispatch/dispatch.go` — admission, exact tuple, registry/publication/cancellation, terminal, and retirement.
- `backend/engine/analyzer.go` and `backend/engine/analyzer_incremental.go` — provider-adjacent DB effects, usage, results, summaries, checkpoints, and finalization.
- `backend/api/handlers/jobs.go` — create/update/delete/clear, including `ClearJobResults`.
- `backend/api/handlers/channels.go` and `backend/api/handlers/demo.go` — channel/purge cascades and tenant reset.
- `backend/notifications/dispatcher.go` and `backend/db/activity.go` — terminal-tail sends, logs, markers, and activity errors.
- `backend/cli/prune_results.go` — batch mutation revalidation after the current one-time running count.

**Exact candidate tests for that future seed:**

- New `backend/engine/job_run_fence_db_test.go` — independent connections, admission, generation, stale writers, terminal tail, commit/readback faults, and lock contention.
- New `backend/api/handlers/job_run_fence_mutations_test.go` — update/delete/clear/channel/purge/reset behavior.
- New `backend/db/job_run_fence_migration_test.go` — schema creation and repeated migration on a disposable fixture.
- New `backend/notifications/job_run_fence_dispatcher_test.go` — terminal notification outcome, log/marker failures, and no replay after unknown send result.
- Existing `backend/engine/job_run_ownership_test.go`, `backend/notifications/dispatcher_test.go`, and `backend/cli/prune_results_test.go` — local cancellation, current notification behavior, and prune mutation coverage.

**Candidate later implementation/evidence documents:**

- `docs/specs/DURABLE_JOBRUN_FENCE_IMPLEMENTATION_SPEC_2026-10-11.md` — accepted implementation contract after the documentary gate.
- `docs/reviews/DURABLE_JOBRUN_FENCE_IMPLEMENTATION_REVIEW_2026-10-11.md` — independent source/test/evidence review.
- `docs/reviews/probes/r080_jobrun_fence_build_evidence_2026-10-11.json` — tested commit, fixture identity, commands/results, injected failures, and explicit non-claims.

These are candidate paths only. The future seed may narrow or amend them before BUILD; a newly found writer requires review and new authority. The implementation proposal covers atomic grant allocation; exact tuple checks in every participating DB transaction; checked notification/activity markers; lock-ordered mutation integration; preserving local cancellation/checkpoint semantics; and a drained terminal-tail retirement. It excludes owner-death recovery, TTL/age reclaim, cross-process cancel, startup sweeps, provider replay, remote exactly-once, mixed-binary guarantees, production DB access, and deployment. [SP01]

## Conditional evidence, stop, and preservation rules

If a future implementation seed is created, it should require an isolated disposable synthetic DB with independent connections/processes and identify its engine/version/isolation, fixture, tested commit, command, and injected failure for every result. Admission ambiguity must distinguish committed-active from rollback/no-grant; the first caller never dispatches without proof, and a later independent request may proceed only through a fresh locked transaction. Writer/terminal ambiguity keeps the active grant and stops later effects. Retirement ambiguity may expose either an active grant or a retired terminal row only after the old DB writer path has drained. Tests must assert persisted state and effect counts, not just error strings. A remote response lost after send is never replayed and never treated as proof of no remote effect. [SP01]

Stop before schema/source work if the independent gate cannot resolve a lock/identity rule, if any writer misses the exact tuple, if a path can mutate around a guarded writer, if cancellation can invert owner/DB lock order, or if a transaction result would be converted into an assumed rollback/commit. Keep the active grant on inconsistency; do not add recovery under this proposal. A newly discovered path is a stop-and-review condition, not permission to widen a seed. [SP01]

For any later DB BUILD, preserve a pre-migration synthetic fixture and logs. Never run a destructive down migration with an active grant. Cleanup stays inside the disposable fixture and follows evidence capture. Failed tests and the first finding stay in the review/evidence record. Review remains independent from implementation. If accepted, the documentary gate is only a prerequisite for root to consider a separate implementation seed; it does not close deployment readiness. [H01]

No fixed model-time or cost estimate is proposed. This draft makes no claim that code is ready, a migration ran, a grant is durable in any environment, all writers are fenced, deployment adopted the design, or CVF governance behavior was proved. [I01][SP01]

### References

[I01]: probes/r080_source_identity_2026-10-11.json
[SP01]: ../specs/DURABLE_JOBRUN_FENCE_DESIGN_SPEC_2026-10-11.md#L1-L174
[H01]: R079_SCOPED_PREREQUISITE_INTAKE_CLOSURE_2026-10-11.md#L1-L11
