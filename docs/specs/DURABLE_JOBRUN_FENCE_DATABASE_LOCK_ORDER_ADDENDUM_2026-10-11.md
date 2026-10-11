# R081 database lock, outcome, incarnation, and cutover addendum

**Tranche:** CCMAI-RUNTIME-081. **Phase:** BUILD. **Risk:** R1 documentary only. **Status:** proposed contract accepted for documentation; implementation readiness and runtime validation remain open.

This addendum binds the accepted R080 intent to the versioned Go/GORM model and writer paths. It describes a target contract, not installed schema, generated SQL, a live database, deployed writers, or runtime proof. The [R081 identity record](../reviews/probes/r081_jobrun_fence_design_gate_2026-10-11.json) binds cited bytes and separately records the R080 and R081 source comparisons.

## Authority and source boundary

R080's database-owned monotonic generation and active grant remain the selected architecture, accepted as an intended design at `402d6a6` / review `56b5328` / scoped closure `99f987c` ([spec](DURABLE_JOBRUN_FENCE_DESIGN_SPEC_2026-10-11.md), [independent review](../reviews/DURABLE_JOBRUN_FENCE_DESIGN_REVIEW_2026-10-11.md), [closure](../reviews/R080_SCOPED_INTENDED_DESIGN_CLOSURE_2026-10-11.md)). R081 adds explicit SQL/lock order, model-edge, Job creation, and cutover details; it does not revise R080's first-pass findings or claim runtime acceptance. [D01][D02][D03][D04][W01]

The source baseline is `62b079f91fffbf2ffcec2421876f3ea74267f912`; the R080 comparison to `efa21565f9a87aa509d7a088ff1ff2d3945baae7` remains preserved. The separately named R081 comparison is `62b079f..17da38e5dc2e005bc45065a44daf887640c35649`; `git diff --name-only ... -- backend` was empty. These Git comparisons identify repository source only. [D04]

The versioned module pins are Go `1.26.0`, GORM `1.31.1`, and MySQL driver `1.6.0`. Startup schema work currently calls `AutoMigrate` and then raw SQL for three unrelated unique indexes; there is no R081 fence migration or readiness gate in this source. Generated DDL, engine/version, isolation, actual constraints, replicas, and installed schema remain **UNKNOWN**. [S01][S02]

## Source facts that shape the integration

| Area | Versioned source fact | Contract consequence |
|---|---|---|
| Models | `Job` has `TenantID` and a `Tenant` association; `JobRun` has tenant/job IDs and a `Job` association keyed only by `JobID`. Neither row has a fence field. Conversation relates to Channel and Messages by single IDs. Result, snapshot, usage, and notification rows carry IDs but do not declare JobRun associations. [S03][S04][S05][S06][S07][S08] | Validate the full tenant/job/run tuple in each transaction. Do not infer composite tenant constraints, cascades, or deployed foreign keys from tags. |
| Admission and publication | Reservation currently locks Job then counts running JobRuns and inserts one. The result transaction currently locks Conversation before JobRun. Finalization locks Job then JobRun. The registry mutex is released before DB waits; the per-owner mutex spans publication's DB transaction. [S09][S10][S11] | Standardize DB rows as Tenant → Job → JobRun → Conversation → child. Preserve the separate local-mutex edges below. |
| Creation and deletion | HTTP `CreateJob` assigns a server UUID. `ImportDemoData` creates two UUID Jobs and two already-terminal JobRuns in one transaction. `ResetDemoData` deletes tenant rows. No other in-repository production Job importer/recreator was found in the bounded source search; tests also seed rows directly. [S12][S13] | Treat demo rows as history, not admissions. Out-of-repository SQL/import/restore and deployed writers remain unenumerated. |
| Conversation writers | A sync write locks its Channel row before upserting a Conversation or Message. Channel delete/purge currently lock Conversations first; demo reset currently locks Conversations before JobRuns. [S13][S14][S15] | Tenant-wide destructive paths must take Job/JobRun before Channel and Conversation, and Channel before Conversation; reject active sync markers before deletion. |
| Destructive job data | `ClearJobResults` reads run IDs and performs unchecked separate deletes. `DeleteJob`/`ClearJobRuns` use a Job guard and then lock runs. CLI prune checks running rows once, then applies later batches. [S12][S16] | Recheck grants and target identity inside each deleting transaction; the initial prune count is advisory only. |

The source's model associations and raw SQL describe the application's schema intent, not the DDL installed by `AutoMigrate` on a particular MySQL version. The intended target adds no reverse `jobs.active_run_id → job_runs.id` foreign key, avoiding a Job↔JobRun FK cycle. Keep the existing JobRun→Job association only as far as the selected migration actually generates it; use explicit tenant/job predicates for correctness. Do not rely on `ON DELETE CASCADE`: delete children explicitly, in one checked transaction. A future migration must inventory and reconcile every actual FK/cascade before enabling the fence. [S02][S03][S04][S06][S07][S08]

## Proposed persisted contract and query order

After a separately authorized migration, the target rows are:

| Row | Intended fields/invariant |
|---|---|
| Tenant | Internal `job_fence_state`: `BLOCKED` by default; only the migration/cutover operator can set `READY`. Normal application writes never promote it. |
| Job | `fence_generation BIGINT UNSIGNED`; nullable `active_run_id CHAR(36)` and `active_run_generation BIGINT UNSIGNED`, both null or both set. Counter never decreases or wraps. |
| JobRun | Immutable `fence_generation BIGINT UNSIGNED` captured at admission. Exact identity is `T=(tenant_id,job_id,run_id,generation)`. |

Reserve generation `0` for migrated legacy rows and demo-import terminal history. New admissions use positive `generation = prior_generation + 1`; exhaustion fails closed. HTTP and demo Job creation mint fresh UUIDs, set counter `0`, and leave the grant pair null. Demo-created terminal JobRuns use generation `0` and never carry an active grant. Fence fields are server-owned and excluded from client updates. If restore/import can reuse a Job UUID or rewind its counter while a pre-restore writer survives, `T` can be resurrected; no in-repository source proves otherwise. [S03][S12][S13][S17]

Required lookup indexes: `jobs(tenant_id,id)` for ordered tenant scans; `job_runs(tenant_id,job_id,status,id)` for running-row checks; and `job_runs(tenant_id,job_id,id)` for ordered run locks. Preserve the existing primary IDs and other indexes. The active pair has no FK to JobRun. Do not add cascade behavior. Because current tags do not express composite tenant/job ownership and source cannot establish installed constraints, exact composite-FK requirements are a cutover blocker until an isolated DDL inspection confirms the target and no incompatible edge. [S02][S03]

Use this partial order for every transaction that takes more than one class of row lock:

```text
Tenant → Job → JobRun → Channel → Conversation → child rows
```

A single-Job transaction may start at Job; a sync transaction may start at Channel. Neither may later acquire an earlier class. Within a class, enumerate IDs and acquire exact rows in ascending ID order; `ORDER BY` alone does not prove physical lock order. Admission's empty running-row check is a plain indexed read after the Job lock, not an empty `FOR UPDATE` range scan. The parent Job lock excludes compatible same-Job admissions/writers; any old writer that skips it blocks cutover. The future fixture must establish MySQL's actual isolation, gap, FK, and implicit-lock behavior. [S09][S11][S14][S15][S16][S17]

Admission transaction:

```text
lock Tenant; require job_fence_state=READY
lock Job WHERE id=? AND tenant_id=?
require active pair is null; read matching running JobRuns in id order; require none
require generation < UINT64_MAX; let g = generation + 1
UPDATE Job with (g, run_id, g) WHERE old generation matches and active pair is null
INSERT JobRun(tenant_id, job_id, run_id, generation=g, status='running')
read back and verify exact T and active pair; COMMIT
dispatch only after COMMIT returns success
```

Any failed statement with a known transaction rollback leaves no admitted run. A commit/readback/rollback error is **unknown**: that caller never dispatches, resumes, clears, or retries the reservation. A later distinct explicit request locks authoritative rows: a committed active tuple or any incompatible running row is busy; only an exact empty state with the tenant READY may allocate the next generation. A failed locked observation admits nothing. The in-memory registry remains a cancellation aid, never the persisted grant. [S09]

Each run-attributed DB write first locks the exact Job, then JobRun, verifies tenant/job/run/generation, the active pair, and allowed phase in that transaction, then locks any Conversation IDs in sorted order before child writes. Hold through commit. This covers usage, summaries, snapshots, evaluations/results, activity, and terminal/checkpoint updates. A provider call is outside DB locks; the grant remains active while it is in flight, but cannot revoke a request already sent. [S10][S11]

Keep the local lock graph separate: `coord.mu` protects only the owner map and is released before DB waits; preserve the short Job→`coord.mu` lookup without adding a `coord.mu`→DB edge. `owner.mu` may span the publication transaction. Cancellation completes its checked DB read before taking `owner.mu`; terminal selection releases it before DB finalization. Never take `owner.mu` while holding DB row locks, and never nest the two mutexes. [S09]

Terminalization locks Job→JobRun, requires exact T and `running`, then atomically writes terminal run state plus Job status/checkpoint. It retains the grant. A checkpoint may have committed with a lost response; no retry or tail follows an unknown result. Known rollback requires exact readback before any local reopen. The grant clears only in a separate exact retirement transaction after all synchronous work is joined and every durable tail marker is settled. [S11]

Before each notification send, future code needs a durable per-run/per-destination attempt identity; after a known send result it records that result and updates any result marker under T. A send, log, marker, or commit ambiguity is `UNKNOWN`: do not resend, do not retire, and keep the grant, even if availability remains blocked. Current `NotificationLog` has no unique attempt key and is written after sending, so this idempotency/control detail remains unresolved. No remote exactly-once or no-effect claim is made. [S07][S11]

## Destructive paths, multirow edges, and cutover

| Path | Intended locked sequence and fail-closed rule |
|---|---|
| Create/update Job; demo import | Tenant→Job. Update rejects any active grant, including terminal tail. Demo import's terminal sample rows are generation `0`; no implicit active owner. |
| DeleteJob, ClearJobRuns, ClearJobResults | Job→all matching JobRuns in ID order; reject any active grant, any running row, or tuple mismatch; then delete result/snapshot/usage/notification children before run/job rows. ClearJobResults becomes one checked transaction. |
| DeleteChannel/PurgeChannelConversations | Tenant→all tenant Jobs→all JobRuns→target Channel→matching Conversations in sorted order→messages/results/snapshots→parents. Reject any tenant active grant/running row and any target Channel with a sync run in progress. This deliberately blocks unrelated tenant jobs to avoid JSON channel-membership inference. File removal follows only confirmed DB commit; unknown commit leaves files untouched and is not retried. |
| ResetDemoData | Tenant→all Jobs→all JobRuns→all Channels→all Conversations, sorted; reject any active job grant, running JobRun, or active channel sync; then checked child-first deletes. |
| CLI prune | Recompute each batch's targets under Tenant→Job→JobRun locks in stable order; require exact stale run/job/tenant set and no active grant. Delete only children. An ambiguous batch is not repeated. |

The parent lock prevents same-Job phantoms only if every compatible admission, writer, and mutation takes it first. Tenant serializes Job creation/configuration against tenant reset; Channel serializes sync conversation/message writes against channel deletion/purge. FK checks and child writes may take implicit parent locks, so the fixture must test the full graph, not infer it from this order. A known statement error with verified rollback differs from an ambiguous commit; in the latter case report `UNKNOWN`, do not claim rollback, and do not replay a destructive batch. [S09][S12][S13][S14][S15][S16]

Intended staged cutover:

1. Add nullable fence fields and `Tenant.job_fence_state=BLOCKED` through an explicit versioned migration; never let startup `AutoMigrate` set READY or assign owners. Admission remains disabled.
2. Stop/adopt every old app, scheduler, worker, manual SQL, and import writer; drain provider work and terminal notification tails. A `running` legacy JobRun or unknown tail blocks its tenant; do not rewrite status, infer owner death, or reclaim by age/map absence.
3. Backfill only after that barrier: historical Job/JobRun generations become `0`, active pairs stay null, and status/checkpoint/results remain unchanged. Validate tenant/job/run joins, pair consistency, unique/index/FK shape, and no running legacy rows. Then finalize non-null/default semantics while READY is still blocked.
4. Deploy only compatible writers. The cutover owner sets the tenant READY bit under the Tenant lock after verifying the external quiescence/adoption receipt and migration checks. Any partial/failing step leaves BLOCKED; application admission fails closed. No automatic migration repair or recovery is authorized.

This staged gate preserves R078's no-rewrite startup acceptance. It cannot establish an old-writer drain or non-reused identity from repository search. A database restore can rewind both UUID/grant rows and the generation counter; a separate restore barrier or non-restorable incarnation source must be selected before any deployment claim. [D03][D04]

## Future acceptance plan and disposition

Candidate **new test files** for a separately authorized R2 tranche: `backend/db/job_run_fence_migration_test.go`, `backend/engine/job_run_fence_database_test.go`, `backend/api/handlers/job_run_fence_destructive_test.go`, and `backend/cli/job_run_fence_prune_test.go`. Existing test helpers use `TEST_DB_DSN` and may skip when absent; they do not create an isolated schema themselves. Later work must require an owner-provided disposable MySQL schema, fail before tests if isolation is not proven, use independent DB connections/coordinators, and capture actual engine/isolation/DDL. No R081 tests or DB calls were made. [T01][T02]

Future bounded campaign proposal: one worker campaign and one independent R2 reviewer campaign; each may invoke at most one `go build ./...` and one targeted `go test -count=1 -timeout=20m` over only the four new test paths (maximum 4 Go processes total, zero retries). Stop on first failure, skip, unknown fixture ownership, or cleanup uncertainty. Preserve exact source/test/`go.mod`/`go.sum` hashes, command, exit/timing, raw `go test -json` digest and PASS/FAIL/SKIP counts, fixture DDL digest, engine/isolation facts, and pre/post unique-schema identity. Drop only the proven task-owned schema after evidence capture; if ownership or cleanup is uncertain, stop and preserve it. This ceiling is a proposal, not authority to run. [W01]

The matrix must include healthy/stale grant with independent coordinators and Jobs; owner-map/publication/cancel races; every usage/summary/result/finalizer/tail write; admission, writer, terminal, retirement, commit/readback/rollback loss; counter overflow, delete/recreate UUID, backup restore, legacy cutover/restart; DeleteJob/ClearJobRuns/ClearJobResults, channel delete/purge, demo reset, and every prune batch; actual FK, multirow, deadlock, gap/phantom locks; and delayed DB/remote responses. Use synthetic provider/notifier seams only for application path counts; no provider proof follows. [D01][W01]

**Disposition:** `DOCUMENTARY_GATE_ACCEPTED_FOR_PROPOSED_CONTRACT_WITH_RUNTIME_VALIDATION_OPEN`; `implementationReady=false`. Named blockers are (1) actual generated/installed FK and lock/isolation shape, (2) old-writer/terminal-tail quiescence and migration-owner evidence, (3) backup-restore/non-reused incarnation control, and (4) durable notification attempt identity/outcome. Each needs source or isolated fixture evidence and owner resolution before implementation readiness. The next move is root's independent review and a separate governed blocker-resolution/R2 work order if accepted. This tranche creates no implementation, migration, recovery, provider, DB, deployment, or governance authority; deployed adoption remains UNKNOWN and Facebook/Zalo OA remains parked. [W01]

## Source and artifact references

- [S01] `backend/go.mod`; [S02] `backend/db/mysql.go`; [S03] `backend/db/models/job.go`.
- [S04] `backend/db/models/conversation.go`; [S05] `backend/db/models/channel.go`; [S06] `backend/db/models/message.go`; [S07] `backend/db/models/setting.go`; [S08] `backend/db/models/analysis_snapshot.go`.
- [S09] `backend/engine/job_run_ownership.go`; [S10] `backend/engine/analyzer.go`; [S11] `backend/engine/analyzer_incremental.go`; [S12] `backend/api/handlers/jobs.go`; [S13] `backend/api/handlers/demo.go`.
- [S14] `backend/api/handlers/channels.go`; [S15] `backend/engine/sync.go`; [S16] `backend/cli/prune_results.go`; [S17] `backend/pkg/helpers.go`; [S18] `backend/db/activity.go`.
- [T01] `backend/engine/snapshot_db_test.go`; [T02] `backend/db/sync_run_migration_test.go`.
- [D01] [R080 intended design spec](DURABLE_JOBRUN_FENCE_DESIGN_SPEC_2026-10-11.md); [D02] [R080 independent review](../reviews/DURABLE_JOBRUN_FENCE_DESIGN_REVIEW_2026-10-11.md); [D03] [R080 scoped closure](../reviews/R080_SCOPED_INTENDED_DESIGN_CLOSURE_2026-10-11.md); [D04] [R080 identity](../reviews/probes/r080_source_identity_2026-10-11.json); [W01] [R081 work order](../work_orders/CCMAI_RUNTIME_081.md).
