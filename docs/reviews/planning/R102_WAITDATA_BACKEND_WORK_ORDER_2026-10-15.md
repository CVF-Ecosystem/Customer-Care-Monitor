# R102 candidate work order — WAIT_DATA backend revision and rerun intent

Status: DRAFT
Readiness: PREPARED_SOURCE_CONTRACT; DISPATCH_AUTHORITY_NOT_INSTALLED
Candidate tranche: CCMAI-RUNTIME-102
Risk: R2
Author / dispatcher: Codex /root
Worker: Sol Medium (proposed implementation worker; not dispatched)
Reviewer: Codex /root (independent of future implementation)
Commit steward / closer: Codex /root
Commit mode: WORKER_MUST_NOT_COMMIT
providerExecutionAuthority: NONE
dispatchKind: INITIAL

## Objective and contract

Implement the source-revision and durable-request capability in [R102 SPEC](R102_WAITDATA_BACKEND_SPEC_2026-10-15.md), W01..W08. User can record locally supplemented source and request reanalysis without overwriting the original analysis/human workflow. Stored requests explicitly await execution authority; this packet includes no actual AI rerun.

Owner requested to work on a roadmap portion independent from blocked UI, then said continue with the proposed backend WAIT_DATA SPEC/work-order preparation. This DRAFT is the concrete output of that preparation. It is not an installed dispatcher seed, source BUILD grant, runtime grant or permission to spend R101 unused slots. Current canonical tranche remains R101 REVIEW until an explicit recorded tranche transition.

## First-read and source baseline

Follow project AGENTS rehydration and fresh doctor. Read manifest/policy/workspace rules, bootstrap/current state, memory current section, active handoff, IMPLEMENTATION_STATUS, docs index, M3-05 roadmap, R101 intended SPEC/seed, independent Claude-return review and this SPEC. Read targeted source listed below and `docs/reviews/learnings/feedback_orchestrator_roadmap_control.md`. Do not read full history without a missing/contradictory fact.

Planning source baseline:200df3e4a7bc2a326425922a5230508a4fd9b0fe. Prior R101 source17/binary hashes and native evidence are inherited only at their exact revision, not acceptance of changed source. Dirty session/review/catalog files predate this packet and must be inventoried. R101 source was independently unchanged at Claude handback; full UI suite has3 inherited Dashboard failures, browserB01..B10 NOT_RUN. Retain these without claiming this packet fixes or supersedes them.

## Proposed allowed implementation paths for the future seed

These paths are a proposed bounded scope, not an active widening of R101:

- `backend/db/models/intervention_waitdata.go` — new records.
- `backend/db/models/intervention.go` — nullable event references only; preserve old row semantics.
- `backend/db/mysql.go` — additive AutoMigrate entries only.
- `backend/engine/intervention_waitdata.go`, `backend/engine/intervention_waitdata_test.go` — transaction-bound observation and intent, local canonical helpers, focused tests.
- `backend/api/handlers/intervention_waitdata.go`, `backend/api/handlers/intervention_waitdata_test.go` — authenticated bounded handlers and real-router MySQL tests.
- `backend/api/router.go` — new tenant routes only.
- Candidate SPEC/order copies at finalized paths declared by dispatcher before BUILD; review/evidence/session/status/catalog as prescribed by the gate.

Existing snapshot/sync/permission/model/router source may be read. Do not edit shared snapshot, sync writer, current RunManual, provider adapters, existing R101 action implementation/tests or frontend merely to make this slice work. If a genuine shared helper change is required, return one consolidated exact-path design finding before implementation; dispatcher must settle that boundary before seeding, not let worker widen it.

No frontend/UI/Dashboard changes, channel import/sync changes, installed executor, provider factory/transport or budget/JobRun coordinator changes. No message-writing or transcript-upload API.

## Proposed effects and budgets — require dispatcher activation

Current preparation: zero source writes, Go/native launches, Node checks/builds, Docker/DB/server/browser/provider effects. R101 counters remain workerGo1/reviewerGo2/aggregate3of16, fixtures2of2, Node3/build1/provider0/mock0; old budgets never reset.

For a future activation, propose at most8 new aggregate Go/build/test-binary launches (worker4, independent reviewer3, dispatcher reserve1), source audit before any launch, and at most1 new cached-image isolated labelled synthetic MySQL fixture bound to loopback. These are new R102 proposals, not permission to create a third R101 fixture or migrate its preserved volume. Zero Node/browser/server/provider/mock-provider/secret effects. Runtime identity, safe protection inventory, source hashes, cached image availability, cleanup owner and launch counters must be recorded before launch. If local prerequisites are unavailable, stop; no install/pull or infrastructure-repair campaign.

Do not run source on original app/DB or read real configuration/credentials. No external request/download, real customer/channel/account, notification, runner activation, dependency changes, Core/gate/workflow edits, commit by worker, push/merge/deploy/release. Test fixtures directly seed synthetic source and contain no AI runner. Never fake provider calls to establish governance behavior.

## Phases, ownership and closeability

INTAKE: owner accepts backend-first roadmap ordering, R101 acceptance remains OPEN. DESIGN: append-only data observation + nonexecuting intent, no overwrite/ownership bypass. SPEC: accepted intended W01..08, source acceptance NOT_RUN. Current WORK_ORDER preparation is complete as a candidate, activation pending.

Before BUILD, Root must resolve any consolidated source-design issues, finalize exact paths/budgets/roles/effects in a new dispatcher-owned immutable seed, commit it, bind the tranche record baseCommit to that seed commit, set grammar fields and work order status, install the full gate-to-role closeability table, run explicit packet pre-dispatch gates, and synchronize/acknowledge the tranche transition. Do not treat a DRAFT profile applicability skip as successful dispatch validation. No bootstrap exception or editable seed shortcut.

Proposed gate ownership for activation: dispatcher owns authorization/pre-dispatch/seed; implementation worker owns source and finite W01..08 evidence/worker return; Root independently owns source/runtime admission/reviewer-fast/pre-commit/result evaluation; session steward owns continuity/catalog; Root alone commits and decides bounded disposition. Worker must not review itself, edit seed or commit. Before first repair, independent reviewer consolidates foreseeable findings and dependency edges. Up to2 same-scope consolidated repair rounds; third without independent new root cause requires REVIEW_COST_ESCALATION_REQUIRED.

BUILD source change → Root consolidated source admission → one bounded actual MySQL campaign → independent REVIEW → conditional local acceptance. A failing mandatory check or missing evidence keeps acceptance open; inherited UI failures still block full-product closure even when an isolated backend scope is evaluated separately. No universal full-suite PASS claim.

## Required worker return

- Fresh declaration/phase-role acknowledgment, accepted source/seed/baseCommit and exact changed paths.
- API behavior and schema contract, transaction lock order, permission/source/replay failure handling.
- W01..08 mapping to test names, commands/cwd, raw stdout/stderr, actual exit and no-SKIP verification. No failure deletion or aggregate-only claims.
- Source/test/binary hash manifest, schema before/after, row-count invariants and replay/concurrency/barrier results.
- Current budget counters, owned fixture IDs/image/network/volume/listeners and safe protected inventory before/after; exact cleanup receipts.
- No source-provider/API-governance assertion from human-software tests. Explicit request state AWAITING_EXECUTION_AUTHORITY/NOT_RUN, actual rerun/browser/fullM3 OPEN.
- Return COMPLETE_PENDING_REVIEW only after required scoped evidence is present; otherwise BLOCKED_WITH_REASON with repro and no automatic retry.

## Stop and next move

Stop for authority/source drift, scope or external-effect expansion, secrets/customer data, wrong fixture ownership, exhausted budget, unexpected SKIP, provider construction, or source guard bypass. Retain failed evidence; only own labelled resources may be cleaned. Do not silently repair Dashboard or browser tooling.

Next governed move: Root reviews this candidate against the read-only adviser findings, resolves known issues in one pass, then prepares immutable dispatch authority. The preparation itself does not close R101, M3-05, M2 or the roadmap. M1 business decisions may proceed separately; M4/M5 implementation remains dependent on their exits.
