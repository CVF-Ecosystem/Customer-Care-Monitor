# S1 agent-run configuration admission

**Tranche:** `CCMAI-RUNTIME-011` · **Phase:** SPEC · **Risk:** R2 · **Entry:** R001–R010 independent REVIEW PASS / FREEZE open; S1 IN_PROGRESS.

## Source finding and decision

`AgentRun` in `backend/api/handlers/agents.go` binds the request, verifies the requested tenant, then discards the error from `config.Load()` and passes a possibly nil configuration to the sync or analysis engine. An invalid runtime configuration can therefore panic or begin an action that cannot run safely. Admit a known agent run only after a non-nil validated configuration. The agent API uses synchronous HTTP 200 for its existing result shape; this tranche changes only failure admission.

## Contract

1. Preserve JSON binding/400 and tenant authorization/403 before configuration loading. Preserve tenant authorization even for an unknown agent name. After tenant authorization, an unknown agent name retains 404 `agent_not_found` without loading configuration or invoking an engine.
2. For `cqa.sync`, `cqa.qc` and `cqa.classify`, load configuration once before constructing a sync engine or analyzer, reading active jobs/channels for dispatch, writing execution state or making an outbound request. On load error or nil config, return generic HTTP 500 `{"error":"agent_run_failed"}`. No secret, config validation text, credential, user action params or token may appear in response/log.
3. On valid config, pass that exact pointer to the existing sync or analysis route. Keep the current HTTP 200 `AgentRunResponse` structure, action semantics, tenant-scoped channel lookup, timeout and engine behavior. Do not turn a synchronous result into a 202 dispatch claim.
4. `AgentQuery`, `AgentHealth`, `ListAgents`, R010 channel routes, job routes, scheduler, sync single-flight/crash recovery and callback DB-write errors are outside R011. No real channel, AI provider or CVF governance claim is authorized by this tranche.

## Acceptance

- Focused tests cover loader error and nil config for all three known agent names: generic 500, exactly one load, no engine invocation, channel/job-run write or outbound call, and no configuration detail leak. Use an accepted-path detector with the same dispatch observer to prove known agents receive the validated pointer and retain the existing response shape. Validate malformed request, unauthorized tenant and unknown name precedence.
- Exercise real `config.Load` with synthetic environment values. Use a private, restored test seam if needed for the loader and dispatcher; do not run seam tests in parallel. No saved key, real provider, real channel or persistent Compose database.
- Run focused and full backend tests on disposable MySQL, `go build ./...`, `go vet ./...`, catalog check, workspace doctor and `git diff --check`. Evidence records exact commands, cleanup and the absence of live provider/governance proof.
