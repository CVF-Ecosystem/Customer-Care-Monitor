# CCMAI-RUNTIME-021 — F01-B MCP tool permission admission

Status: REVIEW_PASS / FREEZE_OPEN after Claude BUILD `b46d587ac350f2dfef76a6dac7e352604be0a037` and [Codex independent review](../reviews/CCMAI_RUNTIME_021_F01B_INDEPENDENT_REVIEW_2026-10-01.md). Issued 2026-10-01 by Codex (ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR). Planning base and dispatcher seed: `8fa6ccb54d66664ba1c91671725c05b7ad35fa44`. Risk ceiling R2. Authority: [SPEC](../specs/RUNTIME_MCP_TOOL_PERMISSION_ADMISSION_F01B_2026-10-01.md), [F01-A review](../reviews/CCMAI_RUNTIME_020_F01A_INDEPENDENT_REVIEW_2026-09-30.md), [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), `CVF_SESSION/authority/CCMAI-RUNTIME-021.json`. The R020 F01-A dependency is REVIEW_PASS / FREEZE_OPEN and its parked gate/PR-evidence condition is satisfied at PR head `3e0b37e`; this work order starts F01-B without FREEZE of R020.

## Assignment and phase gate

Claude is IMPLEMENTATION_WORKER and COMMIT_STEWARD for one local BUILD commit. Codex is independent REVIEWER. Rehydrate manifest/policy, active state/handoff, memory, implementation status, docs index, this SPEC/order and dispatcher seed; run workspace doctor. Acknowledge `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` and the R021 tranche in the active handoff **before** source editing. Return exact local SHA, changed paths, test and negative evidence, and `REVIEW_PENDING`; do not self-approve, push or FREEZE. The seed is dispatcher-owned and must not be edited by the worker.

## Allowed scope

- `backend/mcp/handlers.go`: exact per-tool authorization before handler dispatch, membership lookup with checked DB error, safe tenant discovery projection.
- `backend/mcp/tools.go`: descriptions for the discovery projection change only.
- New focused tests `backend/mcp/*_test.go`; test helpers may use disposable MySQL and synthetic records/tokens. Reuse `backend/api/middleware.PermissionDenial` read-only. `backend/mcp/server.go` and `backend/mcp/oauth.go` are read-only references; mounted-route tests may import their exported route setup without editing them.
- One BUILD evidence record under `docs/reviews/`; this order/SPEC only for necessary clarification; roadmap, implementation status, active state/handoff, session memory and catalog/index when required by a registered artifact change. The authority seed and `.cvf/` are read-only.
- No ordinary HTTP handler, engine, database model/migration, frontend, Compose, CI workflow, CVF core, credentials or API-key file changes. If the SPEC cannot be met inside this scope, return `BUILD_BLOCKED` with the exact dependency instead of broadening it.

## Build and proof sequence

1. Enumerate all twelve tool names from `getAllTools()` and match each to the SPEC matrix. Confirm `cqa_trigger_job` currently does not dispatch; record this existing behavior. Check the current response models for tenant settings, counts and notification content. Do one consolidated dependency and predictable-defect review before changing source.
2. Implement a fail-closed membership/role/permission decision keyed by the authenticated user, requested tenant and exact tool name. Use the shared HTTP permission parser for each required right. Unknown role/tool, DB error, absent/malformed permission JSON and spoofed argument rights cannot admit a call. Denial happens before the protected handler query or job lookup. Keep OAuth/JSON-RPC response conventions and tenant-scoped predicates.
3. Project `cqa_list_tenants` and `cqa_get_tenant` to `id`/`name`/`slug` only and update descriptions. Do not expose counts or `settings` through those two tools. Preserve other authorized outputs and the current non-dispatch behavior of `cqa_trigger_job`.
4. Add meaningful negative and positive disposable-MySQL tests for each matrix row, each half of a conjunctive rule, owner/admin, malformed/missing member permissions, unknown role/tool, no membership, DB error, cross-tenant access and token admission on mounted `/mcp`. Use a recognizable forbidden-data marker and a positive lookup observer. If a test cannot prove side-effect isolation directly, state the limit rather than claiming it. Do not write a test that merely duplicates the matrix implementation.
5. Run focused MCP tests, full backend suite on disposable MySQL (`TEST_DB_DSN`, `-count=1 -p 1`, JSON log), R019 DB sentinel gate, `go build ./...`, downstream preflight and catalog `-Check`, workspace doctor, and `git diff --check`. Document exact commands/results/cleanup and any optional skips. Keep the existing `knowledge/_index.json` outside the commit.
6. Commit only authorized paths locally once and return `REVIEW_PENDING`. For a small issue within the same scope, Codex may repair it during independent review under the existing CVF reviewer-local repair rule; do not create an automatic extra tranche. A change in objective, matrix, risk, external effect or allowed paths needs a new decision before BUILD.

## Review and claim boundary

Codex checks the exact diff, the denied-before-handler ordering, real DB membership evidence, OAuth mounted path, two-right cases, tenant projection and R019 gate. Synthetic permission data proves application RBAC only. A successful response from `cqa_trigger_job` still does not prove queued execution. F01 remains OPEN until independent R021 review; PR #1 stays draft and its SHA-specific CI evidence is historical if any later code is pushed. No provider/channel call, persistent DB, push, merge, deployment, parent-CVF work or FREEZE under this order.
