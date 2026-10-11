# Durable JobRun fence blocker-resolution specification

Status: PENDING_INDEPENDENT_REVIEW. Tranche CCMAI-RUNTIME-082, documentary R1.
Author: /root/r082_worker, SPEC_AUTHOR and documentary IMPLEMENTATION_WORKER.
Reviewer and closer: /root, independent. Identity and finite census: docs/reviews/probes/r082_fence_blocker_resolution_identity_2026-10-11.json.

## Scope and lineage

This selects intended controls to resolve R081 findings F1–F4. R081 remains CHANGES_REQUIRED/FREEZE_OPEN; its original packet, review, and returned publication are immutable. R080’s DB-owned generation and active-grant architecture remains inherited as intended design, not implementation proof. [D01–D04]

The R080 and R081 comparisons remain separately recorded in docs/reviews/probes/r082_fence_blocker_resolution_identity_2026-10-11.json. The separate R082 source comparison is from returned-review publication b1d7f112fb6e775d0774cbe4e01cafa79cf5eed5 to BUILD release 22d2fc0d082db17f3e407b7b674144c64728c386. These identify repository source only. Current application schema, MySQL version/isolation, generated and installed foreign keys, deployed binaries, writer membership, restore controls, and adoption are UNKNOWN. Runtime evidence is NOT_RUN. [D02–D04]

All controls below are proposed for a later separately authorized R2 implementation. No recovery, retry, status rewrite, provider call, database activity, or runtime acceptance is authorized here.

## F1 — selected incarnation and restore control

Selected mechanism: an operator-enforced restore barrier, backed by the persisted Tenant job_fence_state gate and a separate cutover database principal. A database generation alone cannot survive a restore that rewinds it. The application principal may read the gate but cannot set READY; only the cutover principal may change it. A migration creates missing gate rows as BLOCKED. The actual privilege boundary is a future DDL and deployment check, not established by source. [S02–S05, D05]

For every full or partial restore, keep application routes and database access closed until these ordered receipts are complete:

1. Inventory every app, scheduler, worker, manual/import, and management writer that can touch the database. Stop each and retain its instance/version and stop acknowledgment.
2. Revoke the old application database principal, terminate its sessions, and retain a principal/grant and zero-session receipt. Keep the database network route closed. Do not disclose credentials to the new deployment.
3. Restore the selected backup while application access remains closed. Record backup identity, restore operation, target, and affected tenant scope.
4. Using the cutover principal, set affected tenants to BLOCKED and verify exact tenant IDs and row counts. If partial-restore scope is unknown, block every tenant. Apply and verify the versioned schema/DDL identity before application access.
5. Inventory restored grants, running JobRuns, and notification tails. Do not assign an owner, clear a grant, rewrite a status, or infer recovery. Any active grant, running row, unresolved tail, missing receipt, or failed check keeps the tenant BLOCKED and the database inaccessible.
6. Only after the old writer inventory is stopped and revoked, schema checks pass, and each tenant has no active grant/running row/unknown tail may the cutover principal set that tenant READY. Issue a fresh deployment-specific database principal to the versioned compatible deployment, then restore its route. Retain deployment digest and credential-version reference, never the secret.

A failure at any step leaves routes closed and gates BLOCKED. A restarted old process still has no valid principal; it cannot inherit the new deployment identity. If that cannot be demonstrated, no tenant is READY. The cutover receipt must bind restore ID/scope, writer inventory, stop acknowledgments, revoked principal/session count, blocked tenant set/count, schema/DDL digest, active-grant/run/tail inventory, new binary/version, and new principal version. These deployment receipts and the principal boundary are unavailable in this source-only tranche; owner/deployment evidence is a hard readiness blocker. A delayed provider or notifier effect already sent before cutover cannot be revoked.

For new Jobs, require server-generated UUIDv4 through the existing helper, reject caller IDs, and never fall back to a supplied/recycled ID on collision. New Jobs start at generation 0 with the active pair null; imported demo history and migrated legacy rows use generation 0 and no active grant. The existing UUID source is not proof of global non-reuse. Each admitted run increments the unsigned BIGINT generation under the Job lock; overflow fails closed, successful commit consumes it, known rollback does not, and ambiguous commit/readback does not dispatch or retry. Restore safety comes from the barrier, not assumed counter durability. Preserve every restored run/status unchanged.

## F1 — durable notification-attempt contract

One immutable plan is captured for each terminal T = (tenant_id, job_id, run_id, generation). It records eligibility, ordered unique destination membership, nonsecret target identity, config/version reference, and hashes of the exact subject/body bytes passed to each notifier. Never store provider credentials in the plan. A canonical destination_key binds provider type and normalized nonsecret target; duplicate target configurations reject the plan as known no-send rather than creating two deliveries. Changed Job config cannot change an existing plan. [S05, S08, S12, S17]

The intended attempt table has a unique key (tenant_id, job_id, run_id, generation, destination_key, attempt_no), with attempt_no fixed to 1. Persist the plan and all RESERVED attempt rows in a checked transaction before any send. The immutable attempt is the per-destination marker; NotificationLog is written in the same checked settlement transaction. Update JobResult.notified_at only as an aggregate after every planned destination is settled; it is not the destination-level authority.

| Observation | Durable state and action |
|---|---|
| Cancelled run, output disabled/none, empty results, or no destinations | Record plan NOT_REQUIRED under T; no send. |
| Duplicate target or notifier creation/configuration failure before Send | Record REJECTED_NO_SEND or FAILED_NO_SEND under T; no send. |
| Reservation has known rollback or its commit/readback is ambiguous | Do not call Send. Ambiguous reservation remains UNKNOWN and blocks retirement. |
| Reservation committed and this original worker invokes Send | No DB lock is held across the remote call. No other process may adopt RESERVED or send it. |
| Send returns success and attempt/log/aggregate marker transaction is confirmed | Record ACCEPTED (client accepted request, not recipient delivery); settle only after exact primary readback if commit response was lost. |
| Send errors after invocation, or attempt/log/marker commit/readback/rollback is ambiguous | Record or observe UNKNOWN; stop the caller, never replay, never retire. Only a structured adapter result proving no request could have taken effect may settle FAILED_NO_SEND. The current generic Send error does not prove that. |

A RESERVED row left by a crash is not a recovery queue. It stays unresolved; no restart, operator, or later caller may resend it under this contract. No exactly-once or recipient-delivery guarantee is made. All attempts, NotificationLog rows, per-destination markers, and the aggregate notified marker require exact T checks in their write transaction. The current dispatcher sends before an unchecked log insert and shared notified_at update; these are future integration changes, not current guarantees. [S17]

## F2 — global lock, write, and operation contract

Choose Tenant-prefix locking for every compatible tenant-scoped database mutation. The total row-class order is Tenant → Job → JobRun → Channel → Conversation → child; within a class, lock primary keys in ascending order. Channel→Tenant and Job→Tenant are implicit model-parent relationships; Conversation→Channel and Message→Conversation are additional parent edges. Explicit tenant/job/run predicates establish correctness; association tags do not prove composite foreign keys. Do not add a reverse Job.active_run_id→JobRun foreign key. Actual FK and engine acquisition order remain fixture-gated. [S03–S05, S08–S15, D02]

Every application transaction starts by locking exactly one Tenant row by id and requires job_fence_state=READY. Then lock the required parents with tenant-qualified predicates. Admission locks Tenant then Job, requires active pair null and no incompatible running row, checks generation < max, updates the pair/counter with old-generation and null-pair predicates (RowsAffected=1), inserts JobRun with the same T and running status, then commits. Any exact-row mismatch, unexpected count, or failed check aborts. Gate changes are cutover-only.

Every run-attributed usage, summary, snapshot, evaluation/result, activity, terminal/checkpoint, notification, and retirement write locks Tenant→Job→JobRun, verifies exact T, exact active pair, tenant/job/run IDs, and allowed phase, then locks Conversations in sorted order before child rows. Each update includes exact tenant/job/run/generation predicates. Hold database locks through commit; a single-row mutation requires RowsAffected=1, while a multirow operation compares affected rows with the exact locked set. A missing/stale T is a no-op with no child write. Provider/network calls happen outside transactions; a call already sent cannot be fenced retroactively.

For multirow work, discover candidate IDs without locks, begin a transaction, lock Tenant, re-read the exact in-tenant candidate set, and compare it with the discovery set. A mismatch aborts without mutation; the caller returns conflict rather than retrying an ambiguous transaction. Every compatible inserter/deleter first locks the same Tenant, so that parent serializes new tenant rows; do not rely on gap/phantom locks. This is a cooperative-writer design. Hidden FK locks, triggers, actual indexes, isolation, and noncompatible writers are unknown and block readiness until the isolated fixture gate passes.

| Operation group | Intended ordered parents and checks |
|---|---|
| Job create/update/delete; clear results/runs | Tenant→Job→JobRun(s). Creation uses fresh UUID, generation 0, null grant. Updates and deletes reject any active pair, running row, mismatch, or unresolved tail. Lock run IDs sorted; delete result/snapshot/usage/notification/activity children before JobRun/Job. ClearJobResults is one checked transaction. |
| Demo import | Tenant first; verify empty scope; create Jobs and terminal JobRuns at generation 0, then Channels, Conversations, Messages, results/usage/activity, and update the demo setting last. Check every insert/update count. |
| Demo reset | Tenant→all Jobs→all JobRuns→all Channels→all Conversations in sorted order. Reject active grants, running runs, sync reservations, or unresolved notification tails. Delete children then parents; check every statement and final Tenant-setting update. |
| Channel sync writes/status | Reserve under Tenant→Channel; release DB locks during remote fetch. Each write/terminal transaction reacquires Tenant→Channel→Conversation→child and checks exact sync_run_id, tenant, channel, and phase. Multirow sets are rediscovered/revalidated under Tenant. |
| Channel delete/purge and files | Tenant→Channel→Conversations→Messages; reject active sync and unsettled child work. Commit a durable deletion intent before filesystem removal. File removal occurs outside DB locks. A failed or uncertain filesystem result leaves the tombstone and channel blocked; do not report deletion complete or erase the intent. |
| CLI prune batches | Discover candidate result IDs, group by tenant, then each batch locks Tenant→Job→JobRun(s)→children sorted and revalidates exact tenant/time/run predicates. Delete only the locked candidate set and check affected count; never use an unqualified broad delete. |

Local locks remain a separate graph: coordinator map mutex is released before DB waits; the per-owner mutex may span publication. Cancellation completes its checked DB read before taking the owner mutex; terminal selection releases the owner mutex before its DB finalizer. Never take owner mutex while holding DB rows, never add a coordinator-mutex→DB edge, and never nest the two local mutexes. These are intended source changes, not a proof that current code or MySQL meets them. [S13–S15, S17–S21]

## F2 — phase outcomes and retirement

| Operation | Known rollback | Commit/readback/rollback ambiguity |
|---|---|---|
| Admission | No T is dispatched and the rolled-back generation is not consumed. | Original caller stops: no dispatch, resume, cleanup, or retry. A later distinct explicit request uses a primary locked read; exact active pair or running row is busy, only exact empty/READY state may admit. |
| Run writer | No committed child mutation. | Caller stops; no tail follows an unknown write. A stale T later fails the exact grant predicate. |
| Terminal/checkpoint | No terminal transition; reopen only after exact primary readback proves the transaction rolled back and the same T is still running. | It may have committed. Do not reopen cancellation, publish, or continue the tail. Grant remains until a distinct observation proves state. |
| Notification | Known pre-send failure is FAILED_NO_SEND; no send. | No replay. RESERVED/UNKNOWN or ambiguous log/marker state blocks retirement. |
| Destructive operation | No partial success is reported. | Do not claim no deletion or repeat a broad delete; reconcile exact locked state under separate authority. |
| Retirement | Grant remains when rollback is known. | Stop and make no global held-grant claim. A later distinct admission may proceed only if primary locked readback finds the active pair null and all other admission conditions hold. |

Retirement is a separate transaction after all synchronous workers and provider/notifier calls have returned, terminal T is exact, all durable activity and notification tails are settled, and no marker is UNKNOWN. Lock Tenant→Job→JobRun; update only where tenant/job/run/generation and active pair equal T and run is terminal; clear both active-pair fields atomically, leave generation unchanged, require RowsAffected=1, then commit. A stale writer cannot match a later T. Lost retirement response does not imply the grant remains held; a delayed remote effect cannot be canceled, so an unjoined/uncertain worker prevents retirement. No recovery, automatic retry, or age-based reclamation is selected. [D02]

## F3 — bounded creation census

The finite source census and identity ranges are in docs/reviews/probes/r082_fence_blocker_resolution_identity_2026-10-11.json. It covers HTTP Job create and demo import, plus 26 tracked test files with raw INSERT or INSERT-SELECT fixtures, including snapshot_db_test.go:137, analyzer_modes_test.go:153, scheduler_startup_hardening_test.go, shared handler/MCP/jobdispatch fixtures, and prune. HTTP/demo use pkg.NewUUID. Test inserts use fixture-supplied helper UUIDs or deterministic synthetic IDs; the INSERT-SELECT fixtures supply a new test ID while copying other Job fields. Current fixtures predate fence columns and do not initialize generations; future idle/terminal rows must explicitly use generation 0/null grant and any running fixture must seed a matching positive generation/grant. JobResult inserts are excluded. [C01–C26]

The bounded search covered tracked backend Go sources for INSERT INTO jobs, INSERT-SELECT from jobs, and Job model create forms. It is not proof against dynamic SQL, external/manual SQL, imports, restore tooling, or untracked/deployed writers. Those memberships remain UNKNOWN and require a deployment-owner inventory before READY.

## F4 — future fixture and test contract; unavailable fact

No tracked source in the inspected repository supplies a digest-pinned disposable MySQL image/server and task-owned schema create/drop fixture. Do not invent a version or digest. The missing fact owner is the future R2 environment/fixture owner. Before tests, that owner must provide a versioned fixture manifest naming exact MySQL server version and image digest, isolated schema creation/drop principal, unique task-owned schema, network/credential scope, effective isolation/DDL capture, and cleanup receipt. Existing TEST_DB_DSN skip/shared helpers are not isolated proof. go.mod pins GORM and the MySQL driver, not deployed server, isolation, generated DDL, or foreign keys. [S01–S02, S23–S24, D02]

Future new test files, only under separate R2 authority:
- backend/db/job_run_fence_migration_test.go: TestJobRunFenceMigrationDDL, TestJobRunFenceRestoreBarrier.
- backend/engine/job_run_fence_database_test.go: TestJobRunFenceAdmission, TestJobRunFenceWriterAndNotification, TestJobRunFenceRetirementOutcomes.
- backend/api/handlers/job_run_fence_destructive_test.go: TestJobRunFenceJobDestructive, TestJobRunFenceDemoImportReset, TestJobRunFenceChannelSyncDelete.
- backend/cli/job_run_fence_prune_test.go: TestJobRunFencePruneBatches.

From the project root, proposed commands are go -C backend build ./... and go -C backend test -json -count=1 -timeout=20m ./db ./engine ./api/handlers ./cli -run '^(TestJobRunFenceMigrationDDL|TestJobRunFenceRestoreBarrier|TestJobRunFenceAdmission|TestJobRunFenceWriterAndNotification|TestJobRunFenceRetirementOutcomes|TestJobRunFenceJobDestructive|TestJobRunFenceDemoImportReset|TestJobRunFenceChannelSyncDelete|TestJobRunFencePruneBatches)$'. Each worker and independent R2 reviewer gets at most one build and one targeted test invocation: four Go invocations total, zero retry. Require every named test to emit PASS in its intended package; exact four packages; no FAIL, SKIP, timeout, empty selection, or uncertain cleanup. Preserve raw JSON, command/exit, schema and DDL, server/isolation, source/build identities, schema ownership, and cleanup receipts. Matrix covers restore/counter/overflow, stale/multi-coordinator and independent Job, every writer/tail, owner/cancel/publication, terminal/retirement outcome loss, duplicates/config changes/send and marker ambiguity, destructive/demo/sync/prune, legacy rows, implicit FK, multirow/gap/phantom/deadlock behavior, and delayed remote limits. NOT_RUN and NOT_EXECUTABLE until separate R2 source, fixture, credential, database, network, and environment authority exists.

## Disposition

| Finding | Selected intended decision | Missing evidence / owner / interim rule |
|---|---|---|
| F1 incarnation | Privileged restore barrier; DB-owned per-Job generation and active grant; preserve restored rows. | Deployment owner must supply every restore/principal/process receipt. Until then, routes closed and Tenant gates BLOCKED. |
| F1 notification | Immutable plan and one no-replay attempt per destination/T; UNKNOWN blocks retirement. | R2 implementer must add schema/adapter outcomes and primary readback. Until then this contract is not implemented and no send is treated as fenced. |
| F2 locks/outcomes | Tenant-prefix graph, exact T predicates, joined settled-tail retirement, explicit ambiguity matrix. | R2 fixture owner must establish generated/installed FK, engine lock, isolation, and rows-affected behavior. Until then, no deadlock-free or integration-ready claim. |
| F3 census | Finite tracked-source creation census bound in [I01]. | Deployment owner must inventory manual/import/restore writers. This regex census is not universal; no adoption claim. |
| F4 tests | Exact future package selectors and stop conditions defined. | R2 fixture owner must provide version/digest and isolated schema authority. Until then, NOT_RUN/NOT_EXECUTABLE. |

Worker handback is PENDING_INDEPENDENT_REVIEW, not acceptance. No implementation, migration, runtime, deployment, adoption, or CVF governance claim is made. R081 and its seven prior quality arrays remain unchanged; root independently decides review and any freeze. Facebook/Zalo OA accounts remain parked.

[I01]: ../reviews/probes/r082_fence_blocker_resolution_identity_2026-10-11.json
