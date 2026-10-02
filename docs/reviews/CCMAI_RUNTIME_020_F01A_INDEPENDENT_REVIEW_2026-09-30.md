# CCMAI-RUNTIME-020 / F01-A — independent HTTP agent permission review

Date: 2026-09-30. Reviewer: Codex, independent of Claude IMPLEMENTATION_WORKER. BUILD commit: `73cedc2f266a4d42d8fe875ef8660e4b9293f46a`, parent `91b8c964286a488a9f33ae2fc042b345eab1df79`. Disposition: **REVIEW_PASS / FREEZE_OPEN / PARKED** for the HTTP agent authorization slice. F01 remains OPEN for MCP and any other independently discovered tenant-only route.

## Continuity and changed set

At INTAKE, active state, handoff and owner handoff said `REVIEW_PENDING`, while the memory front pointer still said `WORK_ORDER` and `IMPLEMENTATION_STATUS.currentPhase` was `WORK_ORDER`. Codex reported `BLOCKED_CONTINUITY_DRIFT`, aligned only those two stale pointers, and reread current state/handoff/memory/status before source review. This is evidence for the planned downstream continuity gate; it is not an acceptance result. Workspace doctor passed 25/25 but does not compare these values.

The exact BUILD diff changes only `agents.go`, the shared tenant permission helper, focused tests, BUILD evidence and continuity. `git diff --exit-code 91b8c96 73cedc2 -- backend/mcp frontend docker-compose.yml .github` returned 0. No provider, channel, persistent DB, secret, push or deployment is claimed.

## Source and test result

- `AgentRun` and `AgentQuery` load the membership row for the requested tenant, then require the exact supported action/resource pair and all SPEC rights before config load/dispatch or query read. The matrix matches the SPEC: sync run needs `channels:w` + `messages:w`; analysis run needs `jobs:w` + `messages:r`; sync query needs `messages:r`; QC/classification query needs `jobs:r`.
- Owner/admin retain supported-pair access; unknown roles fail closed on the HTTP agent surface. Member permission parsing reuses `middleware.PermissionDenial`, extracted from ordinary `RequirePermission` without changing its error codes. Unsupported run action now returns 400 `unsupported_action` rather than the previous 200 `status:error`; this behavior change follows the SPEC's fail-closed pair requirement. Unsupported query resource remains 400. Nonmember 403 and unknown-agent 404 ordering are preserved.
- New tests independently enumerate all eight concrete matrix rows, remove each required right in turn, exercise malformed/empty permission data, tenant crossing, owner/admin, unsupported pairs and mounted routes with JWT. Denied runs assert zero config loads, dispatches, job runs, channel writes and outbound attempts; authorized runs prove those observers fire. Queries assert no denied row marker and own-tenant results on admission. Existing R011 config-admission fixture now uses valid rights/actions while retaining its failure/order assertions. The BUILD record documents mutation probes that made these tests fail before restoring source.
- Codex independently ran `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./api/... -Run 'TestAgent|TestPermissionDenial|TestRequirePermission'`: exit 0; `api/handlers` and `api/middleware` passed on disposable MySQL; container/network removed. Claude's recorded full backend suite passed with 474 test passes, 0 fails, 2 optional skips, R019 gate PASS and all five DB sentinels PASS. Codex did not repeat that full suite in this review.
- Catalog `-Check` and `git diff --check` passed independently. The one untracked `knowledge/_index.json` is an ingest output outside the BUILD commit.

## Boundaries and next move

This accepts application authorization for HTTP `/agents` run/query only. `backend/mcp/handlers.go` still checks tenant membership for tools including `cqa_trigger_job`; it remains F01-B. No CVF runtime governance or provider claim is made by synthetic dispatcher/transport tests. R020 is parked at `REVIEW_PASS / FREEZE_OPEN` at owner request while Codex addresses downstream machine-gate inheritance. F08 still needs a GitHub Actions run on the reviewed workflow before public CI success can be claimed.
