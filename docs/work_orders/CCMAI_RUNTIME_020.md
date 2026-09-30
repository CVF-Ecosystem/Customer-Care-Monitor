# CCMAI-RUNTIME-020 — F01-A HTTP agent permission admission

Status: WORK_ORDER / Claude IMPLEMENTATION_WORKER next. Issued 2026-09-30 by Codex (ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR). Planning base: `8cb55f880a220ba028f9574b74960f582048d606`. Risk ceiling R2. Authority: [SPEC](../specs/RUNTIME_AGENT_HTTP_PERMISSION_ADMISSION_F01_2026-09-30.md), [F01 source review](../reviews/CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md), [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md).

## Assignment and gate

Claude is IMPLEMENTATION_WORKER and COMMIT_STEWARD for one local BUILD commit. Rehydrate manifest/policy, current state, memory, handoff, implementation status, docs index, this SPEC/order; run workspace doctor. Acknowledge `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff **before** source editing. Return commit SHA, changed paths, test/evidence record and `REVIEW_PENDING`; Codex independently reviews. No self-approval or FREEZE.

## Allowed scope

- `backend/api/handlers/agents.go`: exact HTTP action/resource admission and requested-tenant role/permission lookup; preserve tenant isolation and existing R011 config gate.
- `backend/api/middleware/tenant.go`: a small shared permission-decision extraction only if needed to keep semantics identical to tenant routes. `backend/api/router.go` only if needed to wire an equivalent gate; do not alter unrelated routes.
- `backend/api/handlers/agent_run_config_admission_test.go` may update successful fixture permissions and per-agent action strings so existing R011 positives continue to test valid calls. Add focused HTTP agent authorization tests under `backend/api/handlers/`; middleware tests under `backend/api/middleware/` if the shared check changes. Use disposable MySQL and synthetic dispatch/transport; never a persistent Compose DB.
- One BUILD evidence record under `docs/reviews/`, this order/SPEC only for necessary clarifications, roadmap/status/continuity and catalog/index if artifact registration changes.
- Read-only references: `backend/mcp/`, ordinary tenant routes, engine, channels, DB models, frontend, `.cvf/` and CVF core. If MCP changes prove necessary to close this slice, stop and report the dependency; F01-B is a separate work order. Do not alter secrets, provider code, DB migrations, production data or Compose.

## Build sequence and failure conditions

1. Inspect all current agent names/actions/resources, existing tenant permission helper and R011 test fixture. Record an exact implementation plan and the SPEC matrix in BUILD evidence. Unsupported pairs must fail before side effects; do not infer permissions solely from the agent name or a request parameter.
2. Implement fail-closed authorization using the requested tenant's stored membership/role/permissions. Preserve nonmember 403 and unknown-agent 404 ordering; reject supported-pair violations before config load, dispatch or query. Keep owner/admin admission for supported pairs.
3. Add negative and positive tests for every matrix row, one-right-only cases, malformed/missing permissions, cross-tenant membership, unknown/unsupported pairs, and owner/admin. For denied runs, use the existing R011 fixture's write/job-run/outbound observers plus zero-config-load assertion; prove its positive detector still fires. For queries, seed distinguishable rows in two tenants and assert denied/allowed response boundaries. Where practical, exercise mounted routes and JWT as well as handler behavior.
4. Run focused tests, R011 regressions, full Go suite on disposable MySQL (`-count=1 -p 1`), `go build ./...`, and the R019 JSON gate on the full suite log. Check `git diff --check`, catalog `-Check`, workspace doctor and cleanup. Record commands, exit codes, counts and claim limits. If a pre-existing test uses the old `{}` fixture as an allowed member, update only its authorization fixture/valid action; do not weaken its original config/order assertions.
5. Commit only allowed files locally once and return `REVIEW_PENDING`. If a backend test fails from a separate issue or disposable MySQL cannot run, report `BUILD_BLOCKED` with evidence instead of changing unrelated code or calling an external service.

## Review boundary

R020 can pass REVIEW for the HTTP agent surface while F01 remains OPEN for MCP tool authorization and any other independently discovered tenant-only route. A passing synthetic permission test is application authorization evidence, not proof that CVF governs AI runtime. F08 public GitHub Actions verification also remains pending. No push, deployment, provider/channel call, API key use or FREEZE under this order.
