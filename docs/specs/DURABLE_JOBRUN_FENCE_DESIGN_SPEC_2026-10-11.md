# Intended durable JobRun fence design specification

**Tranche:** CCMAI-RUNTIME-080. **Phase:** BUILD of the authorized source-only DESIGN/SPEC packet. **Risk ceiling:** R1, documentary. **Status:** worker first packet; independent review pending. **Owner:** Luna max, SPEC_AUTHOR and documentary IMPLEMENTATION_WORKER. Root owns independent review, all commits, and closure.

This specification selects a DB-owned per-job generation and active grant as the intended durable writer fence. It does not describe installed schema or code, authorize recovery, or establish deployed adoption. The exact source and document identities, citation ranges, and comparison lineage are in [I01]. Every source statement below is bounded by that identity record.

## Baseline and evidence boundary

The source baseline carried from accepted R079 is `62b079f91fffbf2ffcec2421876f3ea74267f912`; the current R080 comparison is BUILD release `efa21565f9a87aa509d7a088ff1ff2d3945baae7`. The repository artifact lineage also retains project closure `25b3f409` and the prior R079 original comparison `b94a81d7` separately from its repair comparison `643ea0db`. The identity record verifies the backend tree is unchanged from the source baseline to the current comparison and records each comparison separately. These are Git source identities, not deployed binary or database identities. [I01][D05][D06]

R079's accepted inventory is the current source-only intake for this design. Its first review retained historical CHANGES_REQUIRED findings and recorded the later documentary repair and acceptance; that history remains visible and is not converted into worker first-pass approval here. R078's accepted synthetic MySQL evidence concerned only the local startup no-rewrite contract. R080 inherits that narrow evidence without replay. Durable fencing, owner-death recovery, fleet compatibility, and deployment adoption remain unproved; deployed facts are UNKNOWN. [D03][D04][D05][D06][D07]

## Current source facts that constrain the design

HTTP trigger and test-run, MCP dispatch, cron, after-sync, agent analysis, and direct Analyzer entry points converge on the reservation and shared execution path. Test-run is a mode on that path, not a separate ownership mechanism. Current admission uses a process-local owner map, locks the Job row, counts matching JobRun rows whose status is running, and inserts a running row. The local owner and cancellation token are not persisted. The scheduler source has no JobRun startup sweep; that source fact says nothing about binaries or other writers not represented by this comparison. [S01][S04][S05][S06][S07][S08][S11][S12][S13][S14]

Current Job and JobRun models have no persisted incarnation/generation grant. Result persistence checks inserts in a transaction, but its current parent locks are Conversation then JobRun; it does not prove the active owner. Admission, destructive guards, and finalization use Job then JobRun. The normal finalizer stores run status and Job status/checkpoint with readback; its best-effort error fallback is weaker. Current local cancellation is serialized against publication and terminalization, and current release follows synchronous notification work in the worker. These local and transaction facts do not fence a second process. [S08][S09][S10][S15]

The current effect paths include provider calls, usage rows, snapshots/evaluations/results, initial and progress summaries, terminal/checkpoint writes, activity, notification sends/logs/markers, and worker release. The notification dispatcher sends after terminalization, writes NotificationLog and notified_at without checking those DB errors, then returns. The current result-clearing, channel deletion/purge, demo reset, and CLI prune paths have distinct locking or timing behavior; they are mapped below rather than inferred safe from the owner guard. [S03][S09][S10][S16][S17][S18][S19][S20][S21]

## Design decision and alternatives

**Selected intended direction — DB-owned monotonic generation plus an active grant on the Job row.** The protected data and admission state already live in the project database, and the source uses Job/JobRun row transactions. A generation stored and checked in that same database can be bound to every protected database transaction. An external token allocator would still need every database writer to check its token in the transaction that commits the data; a lease by itself cannot reject stale DB writes. The DB-owned option adds no independent coordinator dependency and gives a concrete same-database predicate. Its safety depends on complete writer adoption and old-writer compatibility, neither of which is current evidence. [S06][S08][S09][S15][S22]

| Option | What it can enforce | Availability and adoption consequence |
|---|---|---|
| DB-owned monotonic generation with active grant — selected | Admission and DB writers can atomically compare the current Job grant with the immutable JobRun tuple while holding the Job row lock through commit. | Adds a short Job-row serialization point and schema work. Any writer that omits the predicate bypasses it. Old binaries must be stopped or proven compatible before deployed reliance. |
| External fencing-token allocator | Can allocate a token only if the token is then stored and checked by every database writer in its commit transaction. | Adds coordinator availability, partial-failure, and rollout dependencies without removing the DB enforcement work. It cannot fence a provider or notification call already sent. Not selected. |
| Keep unresolved runs blocking | Existing compatible source keeps a matching running row busy; the intended grant design also remains blocking when ownership or outcome is uncertain. | Can leave a Job unavailable indefinitely. This is a fail-closed source-compatible fallback, not a guarantee about unknown, older, mixed, or external writers. No age, TTL, local-map absence, or process restart reclaims a grant. [D01][D02][D04][D05] |

The selected direction is an intended database contract, not a claim that a generation field alone is sufficient. A writer that does not lock and verify the exact active tuple is outside the fence until adopted or quiesced. A generation cannot undo a remote effect or establish that an external request failed without effect. [S09][S17][D02]

## Intended persisted identity and state

The logical ownership tuple is **T = (tenant_id, job_id, run_id, generation)**. The Job UUID is the incarnation boundary: every create or recreate path must mint a fresh server-side UUID and must not accept or recycle a caller-supplied old Job ID. If any future import or administrative path can reuse a Job ID, this tuple is insufficient and a durable non-reused incarnation value must be added before implementation. The current HTTP create path generates a UUID server-side; it is not proof that all possible writers do so. [S02][S15]

The intended data shape, not present in source, is:

| Record | Intended fields and invariant |
|---|---|
| Job | `fence_generation` is an unsigned monotonic counter; `active_run_id` and `active_run_generation` are either both null or both set. The counter is never decremented or reset while this Job UUID exists. Counter exhaustion fails admission closed; it never wraps. |
| JobRun | Persist `fence_generation` captured at admission. Existing tenant, job, and run IDs plus that generation must exactly match the Job's active grant before a protected write. A run's captured generation is immutable. |
| Active grant | The pair `(active_run_id, active_run_generation)` identifies the one admitted run for that Job. The grant remains present across terminal status and the post-terminal tail. It is cleared only by an exact, checked retirement transaction after all synchronous worker effects and durable markers have settled. |

Job ID is not a tenant-global counter. It is the non-reused incarnation UUID. A newly created Job receives a fresh identity; a stale writer holding the deleted Job ID cannot match a different Job. A different rule is required if an out-of-band path can restore or reuse an ID. This is an implementation-readiness condition, not an assumption about deployment. [S02][S15]

The intended states are derived from the exact grant and run row, not from age or process presence:

| Persisted observation | Intended decision |
|---|---|
| No active grant and no running JobRun | Eligible for admission after exact locked read and checked transaction. |
| Active grant exactly matches one running JobRun | That tuple alone may perform ordinary run writes and terminalize. A second admission is busy. |
| Active grant matches a terminal JobRun | Terminal tail only: completion/activity records, notification attempts/logs/markers, then exact retirement. All analysis, usage, result, summary, checkpoint, and provider-start effects are closed. Another admission remains busy. |
| Active grant missing or mismatched; duplicate/missing run; run is running without a grant; or any read is uncertain | Reject/no-op and preserve exclusion. Do not repair the row, clear the grant, infer owner death, or retry a possibly committed transition. Reconciliation/recovery is a separate authority. |
| Terminal JobRun with no active grant | May be available only after every writer is on the compatible fence or old writers are quiesced. This state is not evidence of deployed compatibility. |

## Admission, write guard, terminal tail, and retirement

**Admission.** Create the run ID before the transaction. Lock the tenant row first, then the exact tenant-scoped Job row, then matching JobRuns in stable ID order. This short Tenant lock serializes admission with tenant-wide channel/reset operations and Job create/config updates; it intentionally serializes admissions within one tenant for the brief reservation transaction. The transaction checks that the active-grant pair is empty, checks for any incompatible running row, verifies counter capacity, increments the counter, creates exactly one JobRun with the captured generation, sets the active-grant pair, and verifies the exact persisted tuple before commit. All errors abort the transaction. The worker starts only after the commit and exact readback are known successful. A lost/ambiguous commit or readback means no worker start and no automatic clear/retry; the grant remains blocking until a separately authorized reconciliation establishes exact state. The local owner map remains a cancellation aid only and must not be held across a database lock. [S06][S08]

**Protected DB writes.** Every transaction that mutates data for a run begins at the Job lock, then locks the exact JobRun, then (when conversation-bound) locks Conversation rows in ascending stable ID order, then writes child rows. It verifies tenant, Job UUID, run ID, generation, active-grant pair, and allowed run state in that same transaction. It holds these locks until commit. Missing/wrong/terminal identity is a checked no-op/rejection with no writes. A pre-transaction read is advisory and cannot authorize the later write. Statement, commit, or readback uncertainty stops later effects and leaves the active grant in place. No blanket claim is made until each source writer adopts this predicate. [S06][S08][S09][S10]

**Terminal transition.** The finalizer locks Job then JobRun and requires the exact current grant and a running row. It writes the JobRun terminal state and Job status/checkpoint together and verifies both. The grant is deliberately retained. Ordinary clean completion alone may advance the ordinary checkpoint; test-run and other explicit modes, errors, partial work, cancellation, and failed/uncertain finalization do not gain checkpoint advancement. If terminal commit/readback is uncertain, do not run the notification tail or release. A best-effort status update that cannot verify the exact tuple never clears the grant. [S10][D02]

**Post-terminal tail and retirement.** After verified terminalization, only exact-run activity/notification work is allowed. Before each tail DB write, the writer verifies the same tuple and terminal state. The future notification path must check and durably record every configured destination's attempt result, check NotificationLog writes and notified_at updates, and distinguish known send failure from an unknown send outcome. A known, durably logged failure is not replayed. If send outcome, log commit, marker commit, or readback is ambiguous, leave the grant active; do not resend or infer no delivery. Retirement is a final transaction that locks Job then JobRun, verifies exact tuple, terminal state, and completed durable tail markers, and clears only the active-grant pair. Generation remains unchanged. Retirement occurs only after all synchronous calls and spawned work have returned or been joined. A terminal status by itself never proves the old effect path has stopped. [S07][S09][S16][S17]

The DB fence applies only to DB writers that participate in it. Provider and notifier calls get an exact-tuple check immediately before dispatch, but that check cannot be held atomically with the remote service. Once a request is sent, an error or lost response does not prove whether the remote effect occurred. The worker must not retry a possibly sent request as part of fencing, and the design makes no remote exactly-once, retroactive cancellation, or replay-safety claim. The persisted grant stays active until the synchronous effect path returns and its database outcome is known. [S09][S16][S17]

## Writer and effect integration map

| Effect/path | Current source behavior | Intended fence rule |
|---|---|---|
| HTTP, MCP, cron, after-sync, agent, direct Analyzer, including test-run | These paths converge on ReserveJobRun/RunReserved/executeReserved; test-run uses the shared executor. [S01][S04][S05][S06][S07][S08][S11][S12][S13][S14] | All admissions allocate the same DB grant. No route/mode may construct an unfenced JobRun or bypass reservation. |
| Provider invocation | Analyzer checks local cancellation before calls; single and batch calls execute outside a DB transaction. A DB generation cannot revoke an in-flight call. [S09] | Exact-tuple pre-dispatch check; keep grant active while synchronous call is in flight. A returned error is not evidence of no remote effect. No automatic replay. |
| Usage writes | AIUsageLog writes occur after provider responses and are distinct DB writes. [S09][S16] | Guard every usage insert by T in its transaction. On write/commit uncertainty, stop and retain grant; do not proceed as if recorded or absent. |
| Snapshot, evaluation, violation, classification, result | saveResults currently has a transaction and insert checks, but locks Conversation before JobRun and lacks the grant. [S09][S18] | Reorder to Job → JobRun → sorted Conversation → child rows; exact T predicate held through commit. If Conversation is absent or deletion won, write nothing. |
| Initial, error, and progress summaries | Updates inspect DB errors but have no durable generation predicate. [S09] | Guard each update by T; current-running state only. A failure or uncertain outcome stops subsequent analysis effects and retains the grant. |
| Normal/test-run terminal and checkpoint | Finalizer checks Job/JobRun and verifies terminal state; its failure fallback is weaker. [S10][S15] | Require T in the same finalizer transaction. Preserve current checkpoint rules; keep active grant after terminal and do not use the fallback to release it. |
| Start/completion/notification-error activity | Activity writes ignore Create errors and are not run-token guarded. [S07][S24] | Route run-attributed activity writes through guarded transactions or fail retirement if the required durable record is uncertain. They are not ownership proof. |
| Notification send, NotificationLog, notified_at | Send follows terminalization; log and marker errors are ignored, and dispatcher returns nil. [S07][S17] | Narrow terminal-tail allowlist under T. Check send outcome as far as the client reports it, check durable log/marker writes, never replay ambiguous sends, and retain grant on uncertainty. |
| Start abort, panic, early failure, cancellation | These close through the common finalizer; failed close can leave a running row. Local cancel races with publish/terminal under a mutex. [S06][S07][S08] | Use the exact grant for every close. Accepted cancel before terminal closes cancelled, suppresses later provider/notification work, and never advances the ordinary checkpoint. Terminal winner refuses later cancel. Unknown close keeps grant. No cross-process cancel/reclaim is added. |
| Worker release | The local owner is released after synchronous worker effects, including notification work, but that local release is process-local. [S06][S07] | Clear durable active grant only through verified retirement after the complete tail; local-map absence never releases it. |

## Mutation, deletion, and lock-order contract

The intended lock partial order is **Tenant → Job → JobRun → Conversation → child rows**. Admission, Job create/config update, channel/purge, reset, and multi-Job prune acquire Tenant first. A single-Job writer or mutation may begin at Job; a run writer then takes JobRun and, when conversation-bound, sorted Conversation rows. Operations may begin at a suffix only if they never acquire an earlier class later in the transaction. All multirow locks use stable ascending IDs within each class. The in-process owner mutex is never held while a transaction waits for a DB lock. This is the target logical graph; actual MySQL/InnoDB locks, foreign-key checks, isolation, and query plans still require the future isolated-database tests below. [S06][S09][S19][S20][S21][S22]

| Existing path | Current interaction | Required future integration |
|---|---|---|
| UpdateJob / CreateJob | CreateJob mints a server UUID; UpdateJob applies an allowlist directly and does not take the run guard. [S02] | Internal fence fields are never client-writable. Create/recreate must mint a fresh Job UUID. Both operations lock Tenant first; UpdateJob then locks Job and rejects every update while any grant is active, including terminal tail. |
| GuardJobMutation / DeleteJob / ClearJobRuns | GuardJobMutation locks Job and checks local ownership plus running rows; DeleteJob and ClearJobRuns then lock runs and delete/reset data in transactions. [S03][S08] | Guard must reject any active grant, including terminal-tail grants, not only running status/local owner. Lock Job then JobRuns in stable order and check every child statement. No deletion/reset on busy or uncertainty. |
| ClearJobResults | Current path plucks run IDs and performs multiple unchecked, nontransactional deletes without GuardJobMutation. [S03] | Convert to one checked transaction; lock Job then sorted JobRuns, reject any active grant, delete results/snapshots/usage/notification rows, and return no success on any uncertain statement/commit. |
| DeleteChannel / PurgeChannelConversations | Current code locks Conversation rows then deletes result/snapshot/message children and channel state; file removal follows commit. It does not acquire Job/JobRun. [S19] | For this tenant-wide destructive operation, lock Tenant then all tenant Jobs and JobRuns in stable order, reject if any active grant exists, then lock affected Conversations and delete children. Do not acquire Job or JobRun after Conversation. This broad gate can block unrelated jobs, an explicit availability cost. File deletion remains post-commit; no JobRun token can roll back filesystem effects. |
| ResetDemoData | Current path locks Conversations then JobRuns, opposite the selected writer prefix, and deletes Jobs, runs, results, messages, and other tenant rows. [S20] | Reorder to Tenant → all Jobs → all JobRuns → all Conversations in stable order; block the reset if any active grant exists. Tenant-scoped creation/admission/configuration paths must participate in the Tenant prefix so reset cannot race a new Job/grant. Any statement/commit uncertainty aborts the whole transaction. |
| CLI prune | Current CLI counts running runs once before planning/apply; later batch transactions delete results/snapshots without rechecking grants. [S21][S23] | Treat the initial count as advisory only. Re-read each batch under locks in Tenant/Job/JobRun/Conversation order, sort targets, and reject the batch on any active grant or changed plan. Include this operational writer in adoption before any fleet claim. |
| All other database writers and old binaries | The inspected repository sources are not a census of deployed processes, external SQL, or older binaries. [D05][D06] | Inventory and adopt every writer before relying on the fence. A writer that cannot be upgraded must be stopped/quiesced and evidenced before rollout reliance. No present all-writer or fleet guarantee. |

This lock contract specifies the logical order and the intended changes to the named paths. It does not claim that the existing DB schema, GORM-generated statements, implicit foreign-key locks, or unknown deployed MySQL isolation already satisfy it. If a future DB test finds an inverse/hidden lock edge or any delete can commit around a guarded writer, stop implementation, return to DESIGN, and amend the graph before release. Do not weaken the active-grant check to make a test pass. [S19][S20][S21][S22]

## Fault and healthy acceptance matrix for a later tranche

All rows below are **NOT_RUN**; they are discriminating future requirements, not evidence from this R1 packet. Use isolated MySQL instances and independent DB connections/processes. A successful test must assert rows/grants/checkpoints/effect markers, not only an error string. No real provider call is needed for DB-fence behavior; if a later public claim asserts CVF governance behavior, the separate policy requires a real provider request/response receipt. [D03][D04]

| Scenario | Healthy contrast and required observation | Faulted/stale observation |
|---|---|---|
| Same-job concurrent admission | One exact grant and one JobRun; independent second coordinator receives busy and performs no provider dispatch. Another Job remains admissible. | DB/readback uncertainty leaves the grant blocking and starts no second worker. |
| Generation and identity | Each admitted run has strictly greater generation; terminal retirement leaves the counter unchanged; a recreated Job has a fresh UUID. | Wrong tenant/job/run/generation, wrapped/exhausted counter, or reused Job UUID is rejected with no mutation. |
| Every pre-terminal DB writer | Current T writes usage, summary, snapshots/evaluations/results, terminal/checkpoint. | Stale T at each writer surface writes nothing, including after a newer generation exists. |
| Terminal boundary | Verified terminal row retains grant; only exact notification/activity tail is allowed; successful, known-failed sends are durably distinguished. | Post-terminal analysis/progress/provider effects are rejected. Ambiguous send/log/marker leaves grant active, produces no resend, and prevents new admission. |
| Commit/readback loss | Exact persisted state permits the next documented step only after verified readback. | Lost/uncertain admission, writer, terminal, retirement, or readback leaves exclusion in place; no guessed rollback, checkpoint, retry, or reclaim. |
| Restart and old binary | With only compatible writers, restart/local-map absence does not clear grant; unrelated Job can proceed. | Old/mixed binary or unknown process prevents any fleet claim until a separate quiescence/adoption barrier is evidenced. No age/TTL recovery. |
| Cancellation and checkpoint | Pre-terminal accepted local cancel closes cancelled without ordinary checkpoint/notifications; terminal winner rejects later cancel; clean ordinary completion alone advances the checkpoint. | Stale cancel, wrong run, failed terminal, or unknown outcome does not advance checkpoint or clear grant. |
| Mutation/deletion and lock graph | Exercise UpdateJob, DeleteJob, ClearJobRuns, ClearJobResults, DeleteChannel, PurgeChannelConversations, ResetDemoData, and prune concurrently with active and terminal-tail grants. Assert intended busy/no-op, no partial deletes, no deadlock, no orphan rows. | Inject transaction/statement failure and opposite scheduling; rollback leaves no partial destructive effect or released grant. Verify actual FK and multirow lock behavior. |
| External effect boundary | A test provider/notifier can record dispatch count for application-path tests that make no governance claim. | Simulate lost response after dispatch; no automatic resend or assertion that the remote service did nothing. Real-provider evidence is a separate class only when separately authorized/required. |

## Readiness limits, roles, and next governed move

The contract selects an architecture and specifies the named repository integrations. It is not implementation-ready for deployment. The concrete future blockers are: (1) no migration/source change exists; (2) every listed writer, including ClearJobResults, channel/demo deletion, and prune, must adopt the contract; (3) actual MySQL dialect/version/isolation and implicit FK/multirow lock behavior must pass the isolated concurrency matrix; (4) every Job creation/recreation path must preserve a fresh UUID incarnation; and (5) old binaries and out-of-repository writers must be upgraded or quiesced before any deployed reliance. Unknown topology alone did not prevent this design decision; it prevents a fleet claim until evidence exists. [S02][S19][S20][S21][S22][D05][D06]

The paired later work-order draft is proposal only and names a separately authorized R2+ implementation slice, exact candidate paths, independent reviewer, evidence classes, stop conditions, and barriers. No source/schema/test/runtime/recovery/provider/deployment path is authorized by this spec. Root remains independent reviewer, CLOSER, and COMMIT_STEWARD; worker first-pass review is pending. Root's prehandback source pointers are identified in [I01] as navigation assistance and were independently checked against source; they are not findings acceptance or worker approval. First-pass dissent, any later root assistance, repairs, and measured versus unmeasured costs must remain separately recorded by the independent review. No model-time or cost estimate is asserted. [D06][W02][I01]

There is no CVF governance claim, deployed claim, runtime proof, or test result in this document. The worker performed no runtime, provider/network, DB/host/container, application, Go, repository-test or gate execution. Root separately ran 46 repository governance-gate unit tests (PASS, 38.818s); these check repository metadata and do not prove runtime governance. [I01]

### References

[I01]: ../reviews/probes/r080_source_identity_2026-10-11.json
[S01]: ../../backend/api/router.go#L174-L184
[S02]: ../../backend/api/handlers/jobs.go#L62-L189
[S03]: ../../backend/api/handlers/jobs.go#L191-L366
[S04]: ../../backend/api/handlers/jobs.go#L368-L603
[S05]: ../../backend/mcp/handlers.go#L330-L360
[S06]: ../../backend/jobdispatch/dispatch.go#L71-L147
[S07]: ../../backend/engine/analyzer_modes.go#L73-L273
[S08]: ../../backend/engine/job_run_ownership.go#L114-L397
[S09]: ../../backend/engine/analyzer.go#L53-L952
[S10]: ../../backend/engine/analyzer_incremental.go#L167-L284
[S11]: ../../backend/engine/scheduler.go#L58-L334
[S12]: ../../backend/engine/sync.go#L243-L243
[S13]: ../../backend/main.go#L69-L85
[S14]: ../../backend/api/handlers/agents.go#L265-L310
[S15]: ../../backend/db/models/job.go#L6-L132
[S16]: ../../backend/db/models/setting.go#L19-L45
[S17]: ../../backend/notifications/dispatcher.go#L38-L125
[S18]: ../../backend/engine/snapshot.go#L64-L92
[S19]: ../../backend/api/handlers/channels.go#L248-L390
[S20]: ../../backend/api/handlers/demo.go#L408-L497
[S21]: ../../backend/cli/prune_results.go#L64-L153
[S22]: ../../backend/db/mysql.go#L13-L59
[S23]: ../../backend/cli/prune_results.go#L216-L263
[S24]: ../../backend/db/activity.go#L10-L26
[D01]: ../decisions/DISPATCH_RECOVERY_R076_2026-10-11.md#L7-L30
[D02]: ../specs/DISPATCH_RECOVERY_R076_2026-10-11.md#L102-L160
[D03]: ../reviews/R078_INDEPENDENT_STARTUP_HARDENING_REVIEW_2026-10-11.md#L3-L27
[D04]: ../reviews/R078_SCOPED_LOCAL_STARTUP_CLOSURE_2026-10-11.md#L3-L30
[D05]: ../reviews/R079_SCOPED_PREREQUISITE_INTAKE_CLOSURE_2026-10-11.md#L1-L11
[D06]: ../reviews/R079_INDEPENDENT_PREREQUISITE_REVIEW_2026-10-11.md#L1-L31
[D07]: ../reviews/R079_RECOVERY_FENCE_PREREQUISITE_INTAKE_2026-10-11.md#L1-L99
[W02]: ../reviews/DURABLE_JOBRUN_FENCE_WORK_ORDER_DRAFT_2026-10-11.md#L1-L94
