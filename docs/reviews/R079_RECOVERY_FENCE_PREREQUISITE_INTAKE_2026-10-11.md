# R079 recovery-fence prerequisite intake

**Disposition at worker handback:** documentary BUILD packet complete; root review pending. This R1 packet records source facts, conditional inferences, an inherited safety contract, and a nonbinding next DESIGN/SPEC proposal. It does not implement or prove a durable fence, owner-death detection, recovery, deployment adoption, or CVF governance.

## Evidence boundary and current baseline

The inspected source baseline is `62b079f91fffbf2ffcec2421876f3ea74267f912`; comparison HEAD at the BUILD release was `b94a81d7bd5d182d79d5f5275d496cf4b743b14b`. The source identity record binds every cited source and inherited document to per-file Git blob, physical SHA-256, line count, and citation ranges [I01]. The backend tree has no diff from the baseline to the comparison HEAD. These are repository identities, not deployed-binary identities.

This packet distinguishes current source facts [S01][S02][S03][S04][S05][S06][S07][S08][S09][S10][S11][S12][S13][S14][S15][S16][S17][S18][S19][S20][S21], conditional source inferences, inherited R076 intent [D01][D02], inherited accepted R078 evidence [D04][D05], and deployment facts that remain **UNKNOWN**. R077 is used only for its still-relevant topology-unknown boundary; its older source baseline is not current truth [D03]. R078 was accepted on synthetic MySQL application evidence for the local startup no-rewrite contract (both roles built the source; positive/restored commands passed 12 top-level/16 events, and named mutation M01 failed its expected assertion). R079 ran no application/native commands and inherits that limited evidence without replay [D04][D05]. R078 does not establish fleet, mixed-binary, durable-fence, recovery, provider, hosted, or deployment safety.

The R076 intended contract remains: age or absence of a local owner is insufficient; exclude old, restarted, and newly admitted writers continuously through exact identity inspection, conditional transition, and verified readback; uncertainty stops recovery and replacement admission; preserve partial effects and do not retry/replay or advance the checkpoint [D01][D02]. R076 did not choose a schema, status, queue, lease, or recovery implementation. The current JobRun row has no persisted owner/fence generation [S14].

## Current dispatch and ownership path

HTTP exposes trigger, test-run, and cancel routes [S01]. HTTP trigger and MCP trigger use the shared dispatcher; it validates configuration, reserves a persisted run, then hands it to a worker, with a checked abort attempt on start failure and panic recovery around the worker [S02][S03][S04]. Cron, after-sync, and agent analysis call Analyzer entry points [S10][S13]. Main starts the scheduler before the HTTP router [S12].

There is no standalone test-run executor: `TestRunJob` reserves synchronously and starts the same `StartWorker` path with mode `test_run`; `RunReserved` resolves the mode and sends test-run, conditional, unanalyzed, and since-last through `executeReserved`. Public Analyzer methods use the same reserved execution path [S02][S05][S07]. Test-run selection is capped by the handler and selects recent, not-yet-evaluated conversations; it still uses the shared provider/result/finalizer machinery [S02][S05].

Admission first excludes a same tenant/job in the current process's owner map. Its DB transaction locks Job, counts only matching JobRuns whose status is `running`, then creates a running row (documented order Job → JobRun). The owner token and cancel context remain process-local [S06]. This prevents duplicate local owners and serializes persisted admission while a matching row remains running. It does not provide a persisted generation or prove what another process is doing. Cancellation checks the local owner and stored exact run, then signals that local context; a running row without a local owner fails closed as not owned [S06]. Local `publish` and terminal/cancel mutexes serialize same-owner cancellation against result publication and terminalization, but they are not cross-process predicates [S06].

R078 removed the unconditional JobRun startup sweep. Current `Scheduler.Start` still recovers a separate channel-sync lease class, loads cron work, and starts the scheduler; that channel lease is not a JobRun fence [S10][S11][D04][D05]. The source no longer calls a JobRun cleanup sweep, but live binaries, other writers, DB sharing, and rollout state remain unknown [D03][D04][D05].

## Effect and write inventory

| Surface | Current identity, guard, and outcome handling | Fence-design consequence |
|---|---|---|
| Reservation / worker handoff / abort | Local owner key is tenant+job; DB admission locks Job and checks only `status='running'` before insert. Configuration/start failure and panic attempt the common exact-run finalizer; failed finalization may leave a running row blocking admission [S03][S04][S05][S06]. | A local map or running-row test is not an owner-death proof. Future admission/recovery must use a durable owner epoch and exact tenant/job/run identity or retain the blocking state. |
| Provider calls, including test-run | Single and batch modes check local cancellation before starting a call and pass the owner context to the provider. A call already issued may have completed remotely even when its return is failed or uncertain [S07][S09]. | A database fence cannot undo an in-flight provider effect or prove non-execution from a client error. No replay or exactly-once claim follows. |
| Usage records | After a response, Analyzer inserts `AIUsageLog` for tenant/job/run. The receipt records `WRITE_OUTCOME_UNKNOWN` while the write is in flight, then `WRITE_SUCCEEDED` or `WRITE_FAILED` from the returned DB result; publication proceeds independently, so failed usage persistence does not block result writes [S07][S09][S15]. | A future durable fence must cover the write predicate. Preserve an explicit ambiguous/failed usage outcome; do not describe all writes as unchecked. |
| Snapshot, evaluation, violation, classification, and result rows | `saveResults` runs in a transaction and checks each insert error. It locks Conversation first, then JobRun, to verify parent existence before writing. The JobRun lookup is by id+tenant only: it does not check job id, running status, or generation. Local `owner.publish` serializes only this process's owner/cancel path [S06][S07][S20][S21]. | This Conversation → JobRun lock order differs from admission/finalizer's Job → JobRun order. The existence lock is deletion protection, not durable execution ownership. Future design must inventory and reconcile lock-order dependencies rather than call every writer transaction Job → JobRun. |
| Summary/progress | Initial, single-error, single-progress, and batch-progress summaries update the run model and inspect `.Error`, recording a preparation gap/log on failure. They do not condition on a durable owner generation or verify affected rows/readback [S07]. | Include summary writes in the stale-writer boundary; an error check alone is not a fence. |
| Terminal status/checkpoint | Finalizer locks Job then exact tenant/job/run with status still `running`, conditionally updates terminal status and Job status/checkpoint, and verifies persisted values. Ordinary clean completion alone may advance `last_run_at`; explicit modes and partial/error/cancelled outcomes do not [S08][S14]. On failure, a best-effort exact-running-row error update checks `.Error` but not affected rows/readback; the returned error alone does not prove what persisted [S08][D02]. | This checked finalizer protects its terminal transaction, not earlier writes or every other effect. Future spec must retain outcome uncertainty and checkpoint rules. |
| Activity records | Started/completed/notification-error paths call `LogActivity`; its `DB.Create` result is not returned or checked [S07][S19]. | If activity is included in a protected run audit, its identity and failure outcome need explicit treatment; do not infer durable audit completion from the call. |
| Notifications after terminal | Successful finalization is stored before completed activity and notification delivery. The owner slot remains held locally until execution returns, after notification handling [S07]. A different process has a different local map and persisted admission only checks running rows; after terminal status it may admit while the original process is still delivering. This is a source-derived concurrency possibility, not a reproduced incident. | Local slot exclusion ends at release; the terminal persisted row no longer blocks another process. Do not claim global whole-effect exclusion from the running-row guard. |
| Delivery, log, and `notified_at` | Dispatcher calls external notifier, then `DB.Create`s NotificationLog and updates result `notified_at`; the latter DB outcomes are ignored. Send errors are represented as failed log status if the log write succeeds. Telegram uses an HTTP request context; email ignores the passed context and calls SMTP [S16][S17][S18]. | Separate DB-writer fencing from already-started external delivery. An uncertain send or marker write cannot safely imply no delivery or justify replay; no exactly-once or retroactive cancellation guarantee. |
| Early failure, panic, start abort, cancellation | Invalid input, selection/setup failures, Analyzer panic, worker-start failure, and explicit abort use the owner-bound terminal path without checkpoint. Cancellation is process-local and accepted before the terminal serialization point; terminal errors reopen local cancellation [S03][S04][S05][S06][S07][S08]. | Test all exit paths under stale/current ownership and uncertain terminal outcomes; preserve the existing local cancellation contract while designing any cross-process rule. |

### Release and cross-process distinction

The local owner slot is released after the worker's last local effect, including notifications [S06][S07]. The persisted row becomes terminal earlier, in the finalizer. Thus: (1) same-process admission remains locally excluded until notification work returns; (2) another process is not covered by that local map; and (3) persisted admission can stop blocking once the row is terminal, even while post-terminal notification work is still running. This rules out interpreting the current running-row admission check as a durable whole-effect fence. It does not establish that a second process currently exists or that such overlap has occurred.

The result-writer transaction also has a separate lock purpose/order: Conversation → JobRun parent-existence/deletion protection, whereas admission/finalizer use Job → JobRun. Future design must decide a compatible lock graph across all writers and deletion paths, and test deadlock/error outcomes; it must not label the existence lock as an ownership check [S07][S08].

## Dependency matrix for any future safe recovery decision

| Dependency / fault | Required design rule | Healthy contrast / observable result |
|---|---|---|
| Concurrent admission for the exact job | A current durable owner generation serializes admission across processes; only the current generation may write or finalize. | One valid owner admits once; a second coordinator sharing the isolated DB receives busy and causes no provider call or second running row. |
| Delayed or stale old writer | Keep exclusion continuously held from exact row inspection through transition and readback; stale generation must fail every protected DB write, not just finalization. | Current generation writes are accepted; stale generation's progress/result/terminal attempts are rejected with no state change. |
| Restart / local owner absent | Do not equate process restart or local-map absence with death. In the inspected R078+ source contract, compatible admission leaves a matching `running` row blocking; that behavior is not guaranteed for unknown older or mixed binaries. | With compatible writers, a different free job remains admissible while the unresolved row stays unchanged. Deployed behavior remains UNKNOWN until compatibility/quiescence is established. |
| Old binary / mixed rollout | A field or new lock is ineffective against a binary that ignores it. Require a compatibility or complete old-writer quiescence/rollout barrier before relying on new fencing. | Only after the barrier is proven may the compatible-writer contract be relied on. Before it, R079 makes no fleet claim and grants no recovery/replacement authorization; deployment parity is UNKNOWN. |
| Lost/stale proof or ambiguous DB read/write/commit/readback | Stop. Do not retry a possibly committed transition, claim recovery, advance checkpoint, or admit replacement until exclusion is re-established and exact state is reconciled [D02]. | A healthy checked transition has one exact identity, verified terminal readback, and no competing writer throughout. |
| Wrong tenant/job/run or already-terminal state | Scope every read, conditional write, and verification to the exact tenant/job/run and generation; terminal is a no-op [D02]. | A valid exact running row is the only candidate; other tenants, runs, and terminal rows remain unchanged. |
| Pre-terminal and post-terminal effects | Cover usage, snapshots/evaluations/results, progress, terminal/checkpoint, and post-terminal logs/notification state. Explicitly distinguish local DB writes from external calls already in flight. | Stale DB writes are rejected; an external outcome already in flight remains marked uncertain and is never inferred absent or replayed. |
| Cancellation, finalizer, and lock interaction | Preserve local cancel-vs-terminal serialization and checkpoint semantics; specify interaction with cross-process fencing and the distinct Job→JobRun and Conversation→JobRun lock paths. | Accepted local cancellation closes cancelled with no ordinary checkpoint/notification; terminal winner rejects later local cancellation; no lock-order deadlock. |

These are future discriminating requirements, not executed tests. A test that only sees a generic error is insufficient: the positive held-barrier case and the faulted/lost-proof case must have different observable outcomes. Synthetic isolated-DB/multi-process evidence, any real provider evidence, and sanitized operational rollout evidence are separate evidence classes. R079 performed none of them. Any future claim that CVF governs AI/agent behavior still needs a real provider request/response under the applicable policy; mock output cannot prove governance.

## Options and selected next slice

| Direction class | Feasibility and limitation | Rollout / availability consequence |
|---|---|---|
| Database-owned monotonic owner generation | The DB already serializes admission and terminalization on Job/JobRun rows, so a persisted generation is a plausible coordination primitive. It is useful only if every protected DB writer checks the same current generation and identity; a column alone does nothing for legacy writers or external delivery. Schema and transaction design remain unselected. | Needs a schema/migration plan and an old-binary compatibility barrier. Until every possible writer honors or is excluded, the design must not rely on the row remaining blocked or permit recovery; the compatible-writer path may keep it blocked, but fleet behavior is UNKNOWN. |
| External distributed lock/lease with fencing token | A shared coordinator could allocate monotonic tokens across processes. A lease alone does not fence DB writes; the token still has to be enforced at every DB effect, and external sends remain non-transactional. | Adds coordinator availability and rollout dependencies; old binaries can ignore the token. Recovery remains disabled until the DB-enforced token path and compatibility barrier are proven. |
| Keep unresolved runs blocking; no automated recovery | This is the repository/new-compatible-writer fallback: inspected R078+ admission keeps a matching `running` row busy without claiming owner death. It does not cover pre-R078 startup behavior or unknown writers, and it may leave a job unavailable pending a separately governed resolution. | Deployed reliance still requires evidence that every possible writer is compatible or quiesced. Until then the fleet effect is UNKNOWN; no age/local-owner bypass. |

**Selected earliest safe next DESIGN/SPEC slice:** a provider-free, source-only design decision packet that compares the DB-generation and external-token classes against the exact effect inventory, specifies a compatible lock graph (including the two current lock orders), and sets an old-binary rollout/quiescence prerequisite. It must preserve “keep blocking” as the fallback and state precisely what a DB fence cannot say about an in-flight provider call or notification. No live topology is needed to author this intended contract; deployed UNKNOWN remains an operational gate, not a reason to defer the design. The follow-on must not select a schema/status/recovery operation until that design decision is reviewed and separately authorized.

The paired nonbinding draft gives candidate paths, bounded role/scope, acceptance, and failure conditions [W02]. It grants no new seed, BUILD, schema/migration, runtime, recovery, or deployment authority. Root remains independent reviewer and commit owner for this packet. Worker first pass is pending; worker repairs: 0; root supplied targeted source-review pointers on terminal/release semantics, actual shared test-run path, usage/error handling, and lock order, which the worker independently verified and cited. This is not a completed review, freeze, or implementation-quality ranking.

[I01]: probes/r079_source_identity_2026-10-11.json
S01: `backend/api/router.go#L174-L184`
S02: `backend/api/handlers/jobs.go#L368-L603`
S03: `backend/mcp/handlers.go#L330-L360`
S04: `backend/jobdispatch/dispatch.go#L71-L147`
S05: `backend/engine/analyzer_modes.go#L35-L273`
S06: `backend/engine/job_run_ownership.go#L61-L397`
S07: `backend/engine/analyzer.go#L39-L952`
S08: `backend/engine/analyzer_incremental.go#L167-L284`
S09: `backend/engine/source_execution_receipt.go#L64-L202`
S10: `backend/engine/scheduler.go#L58-L334`
S11: `backend/engine/sync.go#L1166-L1221`
S12: `backend/main.go#L69-L85`
S13: `backend/api/handlers/agents.go#L265-L310`
S14: `backend/db/models/job.go#L48-L60`
S15: `backend/db/models/setting.go#L34-L45`
S16: `backend/notifications/dispatcher.go#L38-L124`
S17: `backend/notifications/telegram.go#L13-L79`
S18: `backend/notifications/email.go#L10-L39`
S19: `backend/db/activity.go#L10-L26`
S20: `backend/engine/snapshot.go#L64-L92`
S21: `backend/db/models/analysis_snapshot.go#L5-L20`
[D01]: ../decisions/DISPATCH_RECOVERY_R076_2026-10-11.md#L7-L30
[D02]: ../specs/DISPATCH_RECOVERY_R076_2026-10-11.md#L102-L160
[D03]: R077_REPOSITORY_TOPOLOGY_INTAKE_2026-10-11.md#L3-L5
[D04]: R078_INDEPENDENT_STARTUP_HARDENING_REVIEW_2026-10-11.md#L3-L27
[D05]: R078_SCOPED_LOCAL_STARTUP_CLOSURE_2026-10-11.md#L3-L30
[W02]: R079_DURABLE_FENCE_DESIGN_WORK_ORDER_DRAFT_2026-10-11.md

