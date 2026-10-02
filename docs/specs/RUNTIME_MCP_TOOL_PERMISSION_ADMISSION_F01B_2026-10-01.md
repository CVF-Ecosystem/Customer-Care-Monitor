# CCMAI-RUNTIME-021 / F01-B — MCP tool permission admission

Status: SPEC accepted for bounded WORK_ORDER. Date: 2026-10-01. Risk: R2 (tenant data authorization). Authority: [F01 source review](../reviews/CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md), [F01-A SPEC](RUNTIME_AGENT_HTTP_PERMISSION_ADMISSION_F01_2026-09-30.md), [R020 independent review](../reviews/CCMAI_RUNTIME_020_F01A_INDEPENDENT_REVIEW_2026-09-30.md), [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md).

## Observed source and decision

`backend/mcp/server.go` authenticates `/mcp` with an OAuth bearer token and places the token's user ID in context. `backend/mcp/handlers.go:25-91` then checks only a `user_tenants` membership count for eleven tenant-scoped tools; `cqa_list_tenants` skips that check. The count query ignores its error. `cqa_get_tenant` serializes `models.Tenant` including `settings`; `cqa_list_tenants` exposes channel, job and conversation counts. The other tools can read messages, jobs, results and notification bodies, or reach the `cqa_trigger_job` handler. MCP has no tenant permission gate. R020 corrected only HTTP `/agents` run/query and is independent of this router.

The current `toolTriggerJob` only looks up a tenant job and returns a `triggered`/queued message; it does **not** dispatch a job. F01-B protects this tool but must not claim job execution or silently add dispatch. `tools/list` and `initialize` publish protocol/tool metadata, not tenant data.

## Intended contract

1. `tools/call` uses the authenticated OAuth user ID and a stored `user_tenants` row for the **requested** tenant. Never trust `tenant_id`, role or permission claims in tool arguments as proof of rights. Missing membership, DB lookup failure, unrecognized role, missing/malformed permissions and missing required right fail closed before a protected tool handler reads data or performs a side effect. Owner/admin bypass only the individual permission letters, not supported-tool validation, membership or tenant isolation. Reuse `middleware.PermissionDenial` so member parsing matches HTTP routes.
2. Bind each supported tool name to these exact rights. A pair of rights is conjunctive; neither right alone is enough:

   | Tool | Required rights | Returned-data boundary |
   |---|---|---|
   | `cqa_list_tenants` | Membership only | List only the authenticated user's tenant `id`, `name`, `slug`; no channel/job/conversation counts or raw settings. A DB error is an error, not an empty success. |
   | `cqa_get_tenant` | Membership only | Only `id`, `name`, `slug`; no raw `settings`, associations or counts. |
   | `cqa_list_channels` | `channels:r` | Requested-tenant channels only. |
   | `cqa_list_conversations`, `cqa_get_messages`, `cqa_search_messages` | `messages:r` | Requested-tenant conversations/messages only. |
   | `cqa_list_jobs`, `cqa_get_job_results`, `cqa_search_violations` | `jobs:r` | Requested-tenant jobs/results only. |
   | `cqa_get_stats` | `messages:r` **and** `jobs:r` | Requested-tenant aggregate counts only. |
   | `cqa_get_notification_logs` | `settings:r` | Requested-tenant notification records only. |
   | `cqa_trigger_job` | `jobs:w` **and** `messages:r` | Preserve the current lookup/response behavior; do not add dispatch. |

3. Unknown tool names cannot fall back to membership-only admission; preserve the JSON-RPC unknown-tool error. Missing tenant ID, malformed arguments and missing/invalid bearer token retain bounded protocol errors. Denied calls return the existing MCP `ToolResult` error form with generic text; neither response nor log includes stored permission JSON, token, tenant settings or SQL diagnostics. `tools/list` remains metadata-only and may list tools regardless of current per-tenant rights; authorization is mandatory at `tools/call`.
4. Preserve valid tool behavior and tenant predicates except for the explicit discovery projection change. The removed count fields and raw `settings` are a deliberate response-contract change: update tool descriptions and focused tests, and record it in BUILD evidence. OAuth issuance, client scopes, token lifetime, ordinary HTTP routes, R020 matrix, analyzer execution and frontend are outside this slice.

## Acceptance evidence

- Use disposable MySQL with real membership rows for owner, admin, member, unknown role, malformed/missing permissions, nonmember and cross-tenant cases. For every protected tool, prove an allowed member gets the existing tenant-scoped result and a member missing its required right gets a generic MCP error. For each two-right tool, independently remove each right. Exercise `cqa_list_tenants` with mixed memberships and an injected DB failure, and ensure no counts/settings appear; `cqa_get_tenant` must not expose a recognizable `settings` fixture.
- Exercise the mounted `/mcp` route with bearer authentication and JSON-RPC `tools/call` for at least one read tool and `cqa_trigger_job`, including invalid/expired bearer, member denied, owner/admin allowed and cross-tenant attempts. Observe no protected handler query or job lookup after a denied call; the positive fixture must prove the observer can detect a permitted lookup. Verify no job run/dispatch occurs from `cqa_trigger_job` and do not call a provider or channel.
- Run focused MCP tests, full backend suite and `go build ./...` on disposable MySQL; replay the full `go test -json` output through the R019 five-sentinel gate. Run downstream preflight, catalog, doctor and `git diff --check`. Preserve exact commands, exit codes, DB cleanup and evidence limits. These are application RBAC checks; no CVF AI-governance claim is made.

## Boundary

F01 can be reviewed as fixed across HTTP and MCP after this tranche passes independent REVIEW, but FREEZE and deployment remain separate. Any need to alter OAuth semantics, persistent schema, provider/channel behavior, job execution or permissions beyond this matrix requires a new SPEC/order decision before BUILD.
