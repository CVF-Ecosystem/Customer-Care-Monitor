# CCMAI-RUNTIME-029 — F07 Dashboard service-status truth

Status: REVIEW_PENDING. Issued 2026-10-02 by Codex (ORCHESTRATOR -> SPEC_AUTHOR -> WORK_ORDER_AUTHOR). Risk ceiling R2.

Authority: [SPEC](../specs/RUNTIME_DASHBOARD_SERVICE_STATUS_F07_2026-10-02.md), [accepted F07 finding](../reviews/CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md), [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), dispatcher-owned `CVF_SESSION/authority/CCMAI-RUNTIME-029.json`. Exact seed commit is the tranche record's `baseCommit`; seed must already exist there before BUILD. Owner requested this separate F07 plan; no F07 product change occurs during planning.

## Roles and entry

Claude: IMPLEMENTATION_WORKER / COMMIT_STEWARD for one local BUILD returned REVIEW_PENDING. Codex: ORCHESTRATOR / WORK_ORDER_AUTHOR, then independent REVIEWER; no Codex product repair or worker self-approval. Planning synchronization/local commits belong to Codex SESSION_SYNC_STEWARD / COMMIT_STEWARD.

Before any BUILD edit, rehydrate manifest/policy, compact bootstrap or state fallback, memory current prose, active handoff, implementation status, docs index, SPEC/order/tranche/seed; run workspace doctor and knowledge ingest. Write the transition `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` in the active handoff before changing source; synchronize BUILD in state/marker/handoff/status/order/tranche. Missing compact bootstrap is BOOTSTRAP_MIGRATION_PENDING, nonblocking. Contradictory current facts stop at INTAKE with BLOCKED_CONTINUITY_DRIFT. Seed is immutable; confirm it exists at baseCommit and never edit it. R028-R2 limits and REVIEW_PASS / FREEZE_OPEN remain historical and unchanged.

## Allowed scope

Exact allowlist and effects are in the seed. Only:

- `frontend/src/views/Dashboard.vue`: service-state rows, neutral card heading/chips and visible explanation; preserve all other data/request/demo/navigation behavior.
- `frontend/src/i18n/vi.ts`, `en.ts`: scoped service-health unknown/explanation keys; no unrelated wording or global `normal`/`error` semantics changes.
- `frontend/src/__tests__/dashboard*.spec.ts`, `i18n.spec.ts`: mounted status acceptance and retained Dashboard/i18n regressions; broad Dashboard test glob permits retaining probes, not weakening them.
- This SPEC/order, roadmap, BUILD review evidence and captures under `docs/reviews/`, tranche/status/continuity and index/catalog as required. No backend, dependencies, shared API/router, unrelated frontend view, tooling/workflow/.cvf or parent-core edits.

## Bounded execution

1. Trace the service-card data/rendering, existing load/refresh requests and locale keys. Record a consolidated implementation/acceptance plan covering F07-01..07 and groups A..F. Contract is explicit unknown, not measured health. If scope cannot contain the change, return BUILD_BLOCKED with a concrete proposed amendment.
2. Implement neutral unavailable presentation and two scoped locale keys. Do not infer health from Dashboard success/failure, active jobs, payload hints or `/health`. No new calls/polling/timers/timestamps.
3. Run mounted deferred/success/empty/failure/repeat/payload-hint/locale tests, inspect theme classes/icons and render desktop/mobile captures. Fixtures remain UI-only and secret-free. Capture real rendered component output, not a hand-written substitute. Keep existing QC/date/i18n assertions.
4. Show a retained mounted assertion fails against old Dashboard; restore and rerun. Kill a narrow mutation restoring one fabricated success row. Document exact commands, observed failure, restoration and final pass. Never leave mutated source or silently discard failed checks.
5. Run full frontend tests, forced typecheck, build, docs build, catalog, doctor, diff and mandatory downstream gate tests. No backend/DB/provider/channel fixture is needed. Record test counts/skips and all not-run checks honestly. Synthetic responses assert UI rendering only; real provider evidence is required for any runtime CVF governance claim and is outside this order.
6. Record evidence at `docs/reviews/RUNTIME_DASHBOARD_SERVICE_STATUS_F07_BUILD_2026-10-02.md`; link capture paths and map all requirements/groups to evidence. Synchronize REVIEW_PENDING across current prose, state, marker, handoff, status, work order and tranche. Preserve predecessor REVIEW_PASS / FREEZE_OPEN. `buildCommit` may remain null in the worker commit, with exact SHA supplied to reviewer; never invent a self-referential hash.
7. Before the local commit run default preflight, complete explicit scoped-path preflight and gate tests. Exclude pre-existing `knowledge/_index.json` and Python bytecode from commits; disclose default full-worktree failure without calling it PASS. Return exact local SHA, changed paths and evidence to Codex for independent REVIEW. No push or FREEZE.

## Failure conditions and effects

Any fabricated green/healthy/down state, initial success flash, missing locale text, heading implying success, new health request/polling, unrelated regression, unproven detector, missing evidence or in-scope failing check returns CHANGES_REQUIRED or BUILD_BLOCKED. Same-scope repairs continue under existing authority; third same-root repair requires REVIEW_COST_ESCALATION_REQUIRED. Do not widen seed or acceptance silently.

Prohibited: real provider/channel call, persistent DB/customer data mutation, credential read/use/commit, push, merge, deployment, FREEZE, backend/health endpoint/scheduler/schema/permission/workflow/tooling change, parent-CVF edit, dependency change, health polling or unrelated UI redesign. The service card explicitly lacks health measurements; this tranche cannot claim actual API/DB/Scheduler availability, hosted CI success or runtime CVF governance.

## Reviewer return

Return exact BUILD SHA, changed set, F07-01..07 and A..F matrix, full/focused counts, old-source and mutation outputs/restoration, desktop/mobile captures in both locales, request inventory, gate/docs/catalog/doctor results and limits. Codex independently checks source and evidence, confirms dispatcher seed author/timing and records disposition. F07 remains OPEN during DISPATCH_READY/BUILD/REVIEW_PENDING; FREEZE remains OPEN throughout this order.
