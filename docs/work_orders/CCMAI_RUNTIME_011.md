# CCMAI-RUNTIME-011 — agent-run configuration admission

**Date:** 2026-09-29 · **State:** WORK_ORDER / BUILD authorized after Claude rehydration and role acknowledgment · **Risk ceiling:** R2 · **Authority:** [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [SPEC](../specs/RUNTIME_AGENT_CONFIG_ADMISSION_S1_2026-09-29.md), and R010 independent REVIEW PASS.

## Route and independence

Codex: `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR`. Claude: `IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` for one local BUILD/evidence commit. Codex performs independent R2 REVIEW. Rehydrate the current CVF state/handoff and record `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff before BUILD. R010 and CREDIT-003 remain REVIEW PASS / FREEZE open.

## Objective and allowed scope

Fail closed when `AgentRun` cannot load a valid configuration. Preserve request/tenant admission and the existing known-agent success semantics. Allowed source: `backend/api/handlers/agents.go` only. Allowed tests: focused files under `backend/api/handlers/`. Small private loader and dispatcher seams inside `agents.go` are allowed for deterministic tests; restore package globals and avoid `t.Parallel` with them. Allowed records: this work order, SPEC, new BUILD evidence under `docs/reviews/`, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json` and the S1 roadmap status line.

Do not edit `backend/config`, engine, adapters, provider clients, channels, jobs, scheduler, models/migrations, router/middleware, frontend, CVF core or unrelated local work. If fulfilling the contract needs another path or an external effect, stop and return a bounded change request before editing.

## BUILD requirements

1. Trace `AgentRun` from binding and tenant authorization through config load to the `cqa.sync`, `cqa.qc`, `cqa.classify` and unknown-name branches. Keep malformed 400 and unauthorized 403 ahead of loading. Keep unknown-name 404 after tenant authorization and before loading. For known names, reject load error or nil with generic 500 `agent_run_failed`; log only a failure class, never the config error or request params.
2. Load once, then pass the same `*config.Config` to the existing sync/analysis handler. Preserve HTTP 200 result bodies, action routing, timeout, tenant-scoped channel lookup and analyzer/sync engine behavior. No global config cache, asynchronous conversion or downstream execution change.
3. Use synthetic tenants/config and disposable MySQL. For each known agent name, test loader error and nil; assert exact response, one load, no dispatcher call, no channel/job-run write or outbound request, and no secret/log leak. Test accepted paths with a stub dispatcher that observes pointer identity, route selection and unchanged result body without running an engine. Test malformed input, unauthorized tenant and unknown name order. Exercise actual `config.Load` with synthetic environment. The dispatch observer must make the rejection checks nonvacuous; if a direct mutation is disallowed, record that honestly.
4. Run focused and full backend tests, `go build ./...`, `go vet ./...`, catalog `-Check`, workspace doctor and `git diff --check`. Record exact commands/results, disposal of test containers, untested behavior and provider boundary. A failed gate returns `BUILD_BLOCKED`, not REVIEW_PENDING.

## Exit and effects

After passing gates, synchronize continuity/status and make one **local** BUILD/evidence commit with no push. Return `REVIEW_PENDING` to Codex. No saved Alibaba key, real provider/channel call, customer data, persistent Compose DB change, deployment, S1 closure or FREEZE is authorized. Synthetic tests prove admission only; they are not live CVF governance proof.
