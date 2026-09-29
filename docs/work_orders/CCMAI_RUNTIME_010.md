# CCMAI-RUNTIME-010 — channel credential/OAuth configuration admission

**Date:** 2026-09-29 · **State:** WORK_ORDER / BUILD authorized after Claude rehydration and role acknowledgment · **Risk ceiling:** R2 · **Authority:** [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [SPEC](../specs/RUNTIME_CHANNEL_CONFIG_ADMISSION_S1_2026-09-29.md), and R009 independent REVIEW PASS.

## Route and independence

Codex is `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR`. Claude is the `IMPLEMENTATION_WORKER`, then `SESSION_SYNC_STEWARD → COMMIT_STEWARD` for one local BUILD/evidence commit. Codex performs independent R2 REVIEW. Before BUILD, Claude must rehydrate the canonical CVF state/handoff, declare context, and acknowledge this role transition in the active handoff. `CCMAI-CREDIT-003` remains a separate `REVIEW_PENDING` history-cleanup tranche; because Codex implemented it, Claude may independently review it before or alongside R010, without treating that review as R010 acceptance.

## Objective and allowed scope

Handle `config.Load()` error/nil at the five credential/OAuth call sites in `backend/api/handlers/channels.go`: `CreateChannel`, `TestChannelConnection`, `ZaloOAuthCallback`, `ReauthChannel`, `FacebookOAuthCallback`. A small private channel-security config-loader seam is allowed in that file for deterministic tests. Allowed tests: focused files under `backend/api/handlers/`. Allowed records: this work order, SPEC, new BUILD evidence under `docs/reviews/`, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, and the S1 roadmap status line. Do not edit `backend/config`, agents, jobs, engine, adapters, scheduler, models/migrations, frontend, provider clients, CVF core or unrelated local work.

## BUILD requirements

1. Trace all five handler paths before changing them. Preserve malformed-request and wrong-tenant behavior. Load config after the existing admission checks but before any secret operation, side effect or outbound exchange. Error/nil must fail closed with a route-appropriate generic response or callback redirect; do not log secret-bearing config errors.
2. Keep valid-path response/status and OAuth state/credential behavior unchanged. Avoid multiple loads or a new global mutable config cache. If a test seam is package-global, restore it after every test and do not run those tests in parallel.
3. Test each failure/nil path with direct assertions: no panic, no DB write, no token exchange/outbound request and no misleading success. Show an accepted-path detector that proves the tests would observe a side effect. Test wrong-tenant and malformed callback order. Use disposable MySQL, synthetic credentials and stubbed outbound functions; do not use saved keys or real OAuth endpoints.
4. Run focused and full backend tests, build, vet, catalog `-Check`, workspace doctor and diff check. Record exact commands/results, cleanup, mutation/nonvacuity evidence and limitations in BUILD evidence. If a check fails, return `BUILD_BLOCKED`; do not call the work complete.

## Exit and effect boundary

Synchronize continuity/status and create one **local** BUILD/evidence commit, with no push. Return `REVIEW_PENDING` to Codex. No real provider call, channel sync, persistent DB mutation, production credential, deployment, S1 closure or FREEZE is authorized. If completing the contract requires another file or external effect, stop at the boundary and report the requested scope change before editing.
