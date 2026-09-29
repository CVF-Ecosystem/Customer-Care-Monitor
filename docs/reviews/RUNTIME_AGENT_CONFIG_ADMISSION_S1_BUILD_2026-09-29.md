# CCMAI-RUNTIME-011 BUILD evidence — agent-run configuration admission

**Tranche:** `CCMAI-RUNTIME-011` · **Role:** IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD (Claude) · **Date:** 2026-09-29 · **Base commit:** `b0e39b1` · **Risk:** R2 · **Authority:** [SPEC](../specs/RUNTIME_AGENT_CONFIG_ADMISSION_S1_2026-09-29.md), [work order](../work_orders/CCMAI_RUNTIME_011.md) · **Status:** `REVIEW_PENDING` for independent Codex review; no FREEZE.

## Rehydration and role transition

Before BUILD, Claude re-read the manifest, policy, active state, active handoff, session memory, SPEC and work order; core `26c686c` matches the manifest. `ACTIVE_SESSION_BOOTSTRAP_READ_MODEL.json` is absent (`BOOTSTRAP_MIGRATION_PENDING`, non-blocking). The role transition `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` was recorded in the active handoff before any source edit.

## Source trace and change

Only `backend/api/handlers/agents.go` changed in source. `AgentRun` now runs in this order:

1. JSON binding → 400 `invalid_request` (unchanged).
2. `verifyTenantAccess` → 403 `tenant_access_denied` (unchanged; applies to unknown names too).
3. Unknown agent name → 404 `agent_not_found`, decided **before** any config load (previously the load ran first).
4. One call to the private seam `loadAgentRunConfig` (default `config.Load`). On error or nil config: log `[agent] run <name> not admitted: configuration invalid` (failure class only) and return 500 `{"error":"agent_run_failed"}`. No cache.
5. The same `*config.Config` pointer goes to the existing `handleSyncAgent` (`cqa.sync`) or `handleAnalysisAgent` (`cqa.qc`, `cqa.classify`) through two private seams, `dispatchSyncAgent` and `dispatchAnalysisAgent`. The 10-minute timeout, HTTP 200 `AgentRunResponse` shape, action routing and engine behavior are unchanged; the response is still synchronous, not a 202.

`AgentQuery`, `AgentHealth`, `ListAgents`, the handlers' bodies, `backend/config`, engine, scheduler and everything else are untouched.

## Tests

New file `backend/api/handlers/agent_run_config_admission_test.go`, on disposable MySQL with a synthetic user, two synthetic tenants (user linked to one) and a synthetic channel. The stub dispatchers reproduce an engine side effect (a channel status write) so "no write" checks are not vacuous; no engine, adapter, credential or provider is invoked.

- `TestAgentRunConfigFailureIsNotAdmitted`: 3 agents × {loader error, nil config} = 6 subtests. Each asserts exact 500 `agent_run_failed`, exactly one load, zero sync/analysis dispatches, channel row unchanged, and no `SECRET`, `ENCRYPTION_KEY`, request-param secret or config error text in body or log; the log names the agent and the failure class.
- `TestAgentRunPassesValidatedConfigToEngine` (accepted-path detector): for each known agent, 200 with the unchanged body shape, one load, one dispatch on the right route, the identical config pointer, the request tenant, and the observer's write visible.
- `TestAgentRunAdmissionOrder`: malformed JSON and missing required field → 400, unauthorized tenant → 403 for unknown and all three known names, unknown name after tenant authorization → 404 with a failing loader stub; each with zero loads and zero dispatches.
- `TestAgentRunUsesRealConfigValidation`: the real `config.Load` with synthetic `t.Setenv` values. Too-short JWT secret → generic 500 with no `JWT` text and no dispatch; a valid environment reaches the dispatcher with the validated values. Package seams are restored in `t.Cleanup`; no test uses `t.Parallel`.

## Commands and results

1. Focused: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/handlers -Run 'TestAgentRun' -VerboseTests` → exit 0; 4 tests / 22 leaf subtests PASS; `ok .../api/handlers 4.798s`.
2. Full: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1` → exit 0; all 13 packages with tests `ok`, including `api/handlers` (16.8s) and `engine` (6.2s).
3. From `backend/`: `go build ./...` and `go vet ./...` clean. `gofmt -l` on the new test file prints nothing (`agents.go` in the working tree is CRLF, as the checked-out file already was; git normalizes it, and the 29+/10- diff shows only the intended lines).
4. `git diff --check` clean (only the existing LF/CRLF notices). Catalog `powershell -ExecutionPolicy Bypass -File scripts/manage_cvf_downstream_catalog.ps1 -Check` → PASS. Doctor `powershell -ExecutionPolicy Bypass -File ../.Controlled-Vibe-Framework-CVF/scripts/check_cvf_workspace_agent_enforcement.ps1 -ProjectPath <project>` → `RESULT: PASS (25/25)` after status, roadmap, handoff, memory and state were synchronized.
5. Cleanup: the test script removed its MySQL container and network after every run. The persistent Compose database was not used. `go.mod`/`go.sum` unchanged.

## Mutation check — performed

Temporarily changing the guard in `agents.go` to `if false && (err != nil || cfg == nil)` made `TestAgentRunConfigFailureIsNotAdmitted` fail in all 6 subtests (`FAIL ... 1.642s`). The source was then restored from a byte-for-byte backup, confirmed by `grep -c "if false"` = 0 and an unchanged diff stat, and the passing focused run above precedes this check. A reviewer can repeat it in an isolated worktree.

## Untested behavior and limits

- Engines were never run: dispatch is stubbed, so real `SyncAllChannels`, `SyncChannel`, analyzer and provider behavior are untested. `handleSyncAgent`/`handleAnalysisAgent` bodies (including their ignored `db.DB...Find` error and per-job error handling) are outside this tranche.
- Config validation itself is `config.Load` unchanged; only the admission decision is tested.
- Scheduler, cross-path sync single-flight, crash recovery, callback final DB-write errors and UI/API intersections remain separate S1 work.
- This is not live CVF governance proof: no provider API call was made and none was required, because no governance behavior is claimed. No saved key, real channel or customer data was used.
