# Owner account deferral and next local scope

Date: 2026-10-03 (Asia/Saigon). Status: PLANNING_ONLY. Codex ORCHESTRATOR / SPEC_AUTHOR; R1 documentation assessment under project R2 ceiling.

## Owner direction

Owner: "phần account facebook, zalo oa park lại, test sau, làm phần khác trước".

Park Facebook account and Zalo OA account setup, credential/connectivity work and live account/channel tests until the owner resumes them. This is deferred work, not accepted or closed work. Preserve existing local-contract acceptance, outstanding live compatibility/completeness limitations and prior failures. No credentials or network calls were used for this assessment. Pancake live prerequisites remain separate and unchanged; the owner did not explicitly park Pancake.

## Next objective: MCP job execution

Read-only source assessment confirms `backend/mcp/handlers.go` toolTriggerJob performs an own-tenant job lookup then returns `job_trigger_unavailable`. `backend/mcp/tools.go` publishes that unavailable behavior. R035 accepted this truthful response; it did not implement execution.

The HTTP path in `backend/api/handlers/jobs.go` already validates mode/date/cap and configuration, reserves a tenant-scoped run through `engine.ReserveJobRun`, then starts `Analyzer.RunReserved` and returns its run ID. `backend/engine/job_run_ownership.go` owns admission, cancellation, release and abort. These are reusable contracts; MCP must not invent a second queue or bypass ownership.

Proposed DESIGN for a separate bounded work order:

1. Share transport-independent job admission/start behavior with the HTTP path; preserve HTTP request/response and mode/date/cap semantics. MCP initially exposes only the existing default since_last behavior unless a broader contract is explicitly recorded.
2. Preserve R021 MCP rights, tenant membership and own-tenant lookup. Reject unauthorized/missing jobs before configuration, reservations or workers; keep generic public errors.
3. Return an accepted run ID only after a successful reservation and start. Busy/configuration/DB/start failures must not report success; preserve abort failure blocking and ownership cleanup.
4. Require mounted MCP authorization/admission tests, duplicate MCP/HTTP admission, rejected-request no-effect checks, startup failure/abort cases, request-lifetime independence and HTTP regression coverage. Local fixtures do not establish AI governance, provider quality or live-channel readiness.
5. Worker Claude implements; Codex independently reviews. Product dispatch is R2 and needs a separate committed dispatcher seed, SPEC, work order and tranche record before BUILD. This assessment grants no product edit or runtime/credential/provider/network authority. Actual Analyzer/provider execution and its evidence must be separately bounded; no mock governance claim is allowed.

## Continuity and next move

R043 and prior scoped local closures remain unchanged. Current active FREEZE describes the already closed R043 contract, not this new objective. Next move is local SPEC/work-order preparation for MCP execution under standing orchestrator authority, with Facebook/Zalo OA account work parked. No new BUILD, review acceptance or FREEZE is claimed.

Bootstrap read model absent: BOOTSTRAP_MIGRATION_PENDING (nonblocking); canonical state/memory/handoff/status/index read. Workspace doctor PASS 25/25; knowledge ingested into task-local temporary output. Read-only lookup incidents: guessed backend/handlers/jobs.go did not exist, then rg Windows wildcard paths returned error123; canonical paths discovered with rg --files and corrected reads succeeded. Neither incident changed files.
