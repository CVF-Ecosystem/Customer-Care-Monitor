# Nonbinding next DESIGN/SPEC work-order draft: durable JobRun fence

**Status:** proposal only; no next tranche has been activated. This draft is an R1 documentary follow-on to the R079 source intake [W01]. Root must review it, choose the next tranche/seed, and authorize any later boundary separately.

## Objective

Produce a source-backed design decision for whether and how a future durable JobRun owner fence could exclude stale, restarted, and newly admitted writers continuously across all run-related DB effects, without relying on row age or a process-local owner map. Compare a DB-owned monotonic generation with an external fencing-token service; retain unresolved-row blocking as the safe fallback. Define how terminal status, cancellation, ordinary checkpoint behavior, notification delivery, ambiguous outcomes, and old-binary compatibility constrain any later recovery operation [D01][D02][W01].

The design may proceed while deployment topology remains UNKNOWN. It must label that unknown as an operational rollout prerequisite and must not invent a deployed fact or treat R077's older source snapshot as current truth [D03].

## Candidate deliverables and exact paths

This draft proposes only documentary candidates:

- `docs/specs/DURABLE_JOBRUN_FENCE_DESIGN_SPEC_2026-10-11.md` — source-backed options, selected design or explicit unresolved decision, data/transaction/lock-order contract, compatibility and rollout prerequisites, external-effect limits, and acceptance matrix.
- `docs/reviews/DURABLE_JOBRUN_FENCE_DESIGN_REVIEW_2026-10-11.md` — independent source/evidence review and disposition for that spec.
- `docs/reviews/DURABLE_JOBRUN_FENCE_WORK_ORDER_DRAFT_2026-10-11.md` — a separately reviewed proposal for any later implementation tranche, if the design establishes a testable path.

Root allocates the actual tranche ID, immutable authority seed, allowed paths, and tranche record only after reviewing this proposal. These candidate names do not authorize those files or any source change.

## Boundaries and roles

- **Phase:** next DESIGN/SPEC only; R1 repository/documentary scope.
- **Roles:** root ORCHESTRATOR and WORK_ORDER_AUTHOR owns routing and any seed; a named SPEC_AUTHOR prepares the next design; a named reviewer returns findings before freeze; root owns commit/closure. Role transitions and source boundaries are recorded before each phase. If a later implementation is R2+, its reviewer must be independent of its implementation worker.
- **Allowed evidence:** versioned repository source/docs and sanitized owner-supplied topology/build identity statements if voluntarily available. An unknown or absent deployment fact stays UNKNOWN.
- **Budget:** application/native invocations 0; Go/build/test 0; provider/API/network 0; DB, host, container, browser, and credential/`.env` reads 0; automatic retries 0. Pure text, Git identity, hashing, and citation validation only.
- **No authority:** no implementation, tests, schema or migration, status taxonomy, lease/queue/recovery operation, row write, owner bypass, provider call, notification replay, deployment observation, rollout, push, merge, or release. No CVF-governance claim or mocked governance proof.

## Required design questions and dependencies

The spec must inventory the actual trigger paths (HTTP trigger/test-run, MCP, cron, after-sync, agent, and direct Analyzer modes through shared `executeReserved`) and all relevant effect surfaces: provider calls; `AIUsageLog`; result snapshot/evaluation/violation/classification writes; progress/summary; abort/early failure/panic; terminal status/checkpoint and its best-effort error fallback; activity; notification send, `NotificationLog`, `notified_at`; and local release/cancellation. It must distinguish current checked-error handling from missing durable ownership predicates [S02][S03][S04][S05][S06][S07][S08][S09][S10][S11][S12][S13][S14][S15][S16][S17][S18][S19][W01].

It must decide how the lock graph accommodates admission/finalization's Job → JobRun order and `saveResults`' Conversation → JobRun parent-existence order. The existing latter lock does not prove active ownership. It must state which DB effects will carry the fence predicate, how ambiguous commits/readbacks remain blocked, and how an old binary that ignores a new token is excluded before the token is trusted [S06][S07][S08][W01].

It must separately define external-call limits. A persisted token can reject a stale DB write only where the write checks it; it cannot retroactively cancel an already-issued provider/SMTP/API call, establish that a failed client request had no remote effect, or justify replay. No exactly-once claim is allowed [S07][S16][S17][S18][D02].

The spec must compare at least the two proposed fence classes against the keep-blocking fallback, state availability consequences, name assumptions, and either select an implementable design contract or identify a concrete decision that genuinely prevents selection. Unknown deployment topology alone is not a blocker to intended design; it blocks operational reliance until compatibility/quiescence evidence exists [D03][D04][D05][W01].

## Acceptance for the documentary DESIGN/SPEC tranche

1. Every material source statement has a valid line citation and identity record; current source is distinguished from R076 intent, R078 accepted synthetic startup evidence, and deployed UNKNOWN.
2. The option comparison explains where durable exclusion is enforced for admission, all DB writes, finalization, and any pre/post-terminal effect; a generation field or external lease is not treated as sufficient by itself.
3. The design resolves or explicitly leaves open the lock-order integration and defines the old-binary compatibility/quiescence barrier.
4. Positive and faulted scenarios distinguish a valid continuously held barrier from stale/lost proof; wrong tenant/job/run and already-terminal rows are no-op; restart/local-owner absence and a live second writer are covered.
5. Cancellation and clean ordinary checkpoint semantics remain explicit. Terminal-state uncertainty, partial results, usage-write failure, external send uncertainty, notification-state failure, and out-of-order stale writes do not become “no effects,” success, or automatic replay.
6. It names separate future evidence classes: synthetic isolated DB/multi-process checks for the source contract; real provider evidence only if a later authorized claim requires external provider behavior; sanitized operational rollout evidence for binary/topology compatibility. None is run or claimed by this design tranche.
7. The reviewed result authorizes only a future, separate work-order proposal. It does not authorize code, migration, recovery, deployment, or runtime proof.

## Failure conditions and next gate

Return for revision if any source citation is invalid, an effect surface or entry point is omitted, a local lock/existence check is mislabeled as a global fence, a stale binary can bypass the proposed condition without a rollout barrier, lock-order risk is ignored, ambiguous outcomes are retried or called successful, external effects are promised exactly once, deployed facts are inferred, or the doc expands into an implementation plan without a separate seed.

If accepted, root may prepare a fresh bounded work order and immutable seed for a later phase. Any schema, production DB/host inspection, provider/network, recovery operation, or runtime budget is a boundary change requiring separate risk/authority review. Until then, current `running` rows stay fail-closed under existing admission; owner-death/recovery/deployment remain OPEN/UNKNOWN [D01][D02][D03][D04][D05][W01].

[W01]: R079_RECOVERY_FENCE_PREREQUISITE_INTAKE_2026-10-11.md
[D01]: ../decisions/DISPATCH_RECOVERY_R076_2026-10-11.md#L7-L30
[D02]: ../specs/DISPATCH_RECOVERY_R076_2026-10-11.md#L102-L160
[D03]: R077_REPOSITORY_TOPOLOGY_INTAKE_2026-10-11.md#L3-L5
[D04]: R078_INDEPENDENT_STARTUP_HARDENING_REVIEW_2026-10-11.md#L3-L27
[D05]: R078_SCOPED_LOCAL_STARTUP_CLOSURE_2026-10-11.md#L3-L30

