# CCMAI-RUNTIME-021 / F01-B BUILD evidence — MCP tool permission admission

**Date:** 2026-10-01 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 · **Role:** IMPLEMENTATION_WORKER + COMMIT_STEWARD (Claude); independent REVIEWER: Codex
**Authority:** [SPEC](../specs/RUNTIME_MCP_TOOL_PERMISSION_ADMISSION_F01B_2026-10-01.md), [work order](../work_orders/CCMAI_RUNTIME_021.md), dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-021.json` (commit `8fa6ccb`, unedited). Base `4850bb2`.
Disposable MySQL and synthetic rows/tokens only; no provider/channel call, persistent DB, secret, push or FREEZE. The pre-existing untracked `knowledge/_index.json` is not part of the commit.

## Source audit before the change

- Twelve tools in `getAllTools()`; `handleToolsCall` checked only a `user_tenants` count (`mcpVerifyTenantAccess`, error ignored) for eleven tenant-scoped tools. `cqa_list_tenants` skipped it and appended per-tenant channel/job/conversation counts; `cqa_get_tenant` serialized `models.Tenant` including `settings`.
- `cqa_trigger_job` only looks up the tenant's job and returns `status: triggered` / "queued" text; it creates no `job_run` and calls no analyzer. This **existing behavior is unchanged** and is asserted by the tests (0 `job_runs` after permitted calls).
- Unknown tool names previously got the unknown-tool RPC error only *after* the membership gate (a non-member got "access denied" instead).

## Implementation (`backend/mcp/handlers.go`, `backend/mcp/tools.go`)

- `toolPolicies`: explicit table of all twelve tools → tenant-scoped flag and conjunctive rights, exactly the SPEC matrix (`middleware.PermissionDenial` parses each right, so semantics match the HTTP routes: owner/admin skip letters; empty/malformed/missing permissions deny).
- `handleToolsCall` order: parse params → authenticated user (unchanged) → **unknown tool = JSON-RPC `-32602 Unknown tool` with no DB access** → `tenant_id is required` → `authorizeToolCall` → handler. A protected handler is never reached before the decision.
- `authorizeToolCall`: one membership lookup keyed by the authenticated user and the **requested** tenant with its error checked. Lookup error → generic `authorization unavailable`; no row → existing `access denied: you don't have access to this tenant`; role other than exactly `owner`/`admin`/`member` (Go, case-sensitive) → generic `permission denied`; missing right → `permission denied`. Arguments carry no rights. Logs name user/tenant/tool/reason class only (never permission JSON, token, settings or the SQL error).
- `cqa_list_tenants`: reads the caller's memberships (error checked → `Unable to list tenants`, never an empty success), keeps only recognized roles (compared in Go; MySQL collation would fold `OWNER`), then selects `id,name,slug` only. Counts removed; empty result is `[]`.
- `cqa_get_tenant`: `id,name,slug` only (no `settings`). Descriptions of both tools in `tools.go` updated.
- Removed `mcpVerifyTenantAccess`. Other handlers' queries and tenant predicates are untouched.

## Deliberate contract changes (for the reviewer)

1. `cqa_list_tenants` / `cqa_get_tenant` responses lose `channels_count`, `jobs_count`, `conversations_count`, `settings` and associations (SPEC §4).
2. A member whose permissions previously passed on membership alone now needs the matrix rights; `cqa_get_stats` and `cqa_trigger_job` need two rights.
3. A membership with an unrecognized role is also omitted from `cqa_list_tenants` (my reading of "unknown role cannot admit a call"; the SPEC says membership only for that tool — flag if Codex wants it list-visible).
4. An unknown tool now errors before the tenant checks (previously a non-member/missing `tenant_id` saw a different message first).

## Tests (`backend/mcp/permission_admission_test.go`, disposable MySQL)

The matrix is re-declared in the test (independent of `toolPolicies`) and a test asserts it covers exactly the twelve published tools. A GORM query-callback **observer** records which tables each call queries and can inject a failure on a table.

- Every protected tool: exact rights → success, own-tenant marker present, other tenant's data absent, and the observer **sees the protected table lookup** (positive control). Each required right removed while every other letter of every resource is granted → generic `permission denied`, only `user_tenants` queried, no marker in the response, 0 `job_runs`. For `cqa_get_stats` and `cqa_trigger_job` each half alone is denied.
- Owner/admin with `""`, `{}` and non-JSON permissions admitted for all ten tools; the same owner row is denied (`access denied`) in a tenant where the user has no membership.
- Fail-closed data: `""`, `{}`, non-JSON, array, non-string values, `null`, array-valued rights, roles `guest`, empty, `OWNER`, `Admin`, `viewer` (also for `get_tenant`).
- Spoofed `role`/`permissions`/`user_id` arguments ignored; full rights in A/C do not reach tenant B or a missing tenant; nonmember denied; a zero-rights membership in B stays denied despite full rights in A.
- `get_tenant`: membership alone admits; keys are exactly `id,name,slug`; the settings fixture (`R021-FORBIDDEN-SETTINGS-…`) never appears. `list_tenants`: mixed memberships (A, C yes; B no), exactly three keys, no counts/settings/tenant B; outsider gets `[]`; unrecognized-role row omitted; injected failure on `user_tenants` and on `tenants` → error result, no diagnostics.
- Injected `user_tenants` failure for every protected tool → `authorization unavailable`, no protected lookup; response and log contain no SQL diagnostic marker; log carries `membership_lookup_failed`. Denial log has no stored permission JSON.
- Unknown tools (`cqa_nope`, empty, upper-case, trailing space, suffix) → `-32602` with zero DB queries; missing `tenant_id` → bounded error with zero DB queries; malformed params → `Invalid params`; no authenticated user → `authentication required`.
- Mounted `/mcp` (`SetupMCPRoutes`, real bearer-token rows): missing/invalid/expired token → 401 touching only the token table; member without `channels:r` / `jobs:w` → generic denial with no `jobs` query for the trigger; exact-rights member, owner and admin (broken permissions) → trigger succeeds, the `jobs` lookup observed once, **0 job runs**; permitted read returns only tenant A; owner of A calling tenant B denied without touching protected tables; unknown tool → RPC error; `tools/list` (12 tools, no tenant query, updated descriptions) and `initialize` still work.

**Limits:** side-effect isolation is shown by table-level query observation, `job_runs` counts and the absence of dispatch code — not by tracing every possible code path. Other handlers still ignore their own DB errors (pre-existing, out of scope). Synthetic permission data proves application RBAC only; a successful `cqa_trigger_job` still does not prove queued execution.

## Non-vacuity mutations (temporary edits to `handlers.go`, restored byte-for-byte; `-run ./mcp` on disposable MySQL)

| Mutation | Result |
|---|---|
| right loop iterates `rights[:0]` (no right checked) | 5 tests failed |
| only the first right checked (`rights[:1]`) | `EachToolRequiresExactlyItsRights` failed (plus a panic-driven `GetTenant` failure) |
| membership lookup error admits | `MembershipLookupErrorFailsClosedWithoutLeaks` failed |
| unrecognized-role check disabled | `BadPermissionDataAndRolesFailClosed` failed |
| `cqa_trigger_job` needs `jobs:r` instead of `jobs:w` | `EachToolRequiresExactlyItsRights` + mounted-route test failed |

The `restored-identical` check (`cmp` against the pre-mutation copy) passed before the final run. I did not run a mutation for the get_tenant settings projection or the list_tenants projection; those are covered by direct assertions only.

## Gates (exact commands in repo root)

| Check | Result |
|---|---|
| Focused: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./mcp -VerboseTests` | 11 test functions (+10 subtests) PASS |
| Full: `go test ./... -json -count=1 -p 1` on disposable MySQL (`TEST_DB_DSN`, `golang:1.26-alpine`, GOPROXY=off) | exit 0; 495 pass / 0 fail / 2 optional skips (was 474 at R020) |
| R019 gate: `python scripts/ci_db_test_gate.py <go-test-json>` | `GATE PASSED`, 0 invalid records, five sentinels PASS, 0 DB-unavailable skips; optional skips `pricing.TestFetchThatTuNguonNgoai`, `storage.TestNoiCatS3` |
| `go build ./...`, `go vet ./mcp` | OK |
| `git diff --check` | clean (CRLF warnings only) |
| Cleanup | disposable MySQL containers/network removed after each run; none left (`docker ps -a`, `docker network ls`) |

Note on the gate input: my wrapper copy of `test-backend.ps1` prints two non-JSON lines ("Waiting for…", "Removed…"); the first replay counted them as 2 invalid records and failed. I stripped exactly those two non-`{` lines (verified they are the wrapper's own text) and replayed the unmodified Go output, which passes. The committed script is unchanged.

Downstream preflight, catalog `-Check` and workspace doctor results are recorded in the BUILD commit message / handoff after the commit (they run on the committed range).

## Claim limits

Application RBAC for the MCP surface only, with synthetic rows. No CVF AI-governance, provider, channel or deployment claim. F01 stays OPEN until independent R021 REVIEW; PR #1 remains draft at `3e0b37e` and its hosted-CI evidence is historical for any later code.

## Changed set

`backend/mcp/handlers.go`, `backend/mcp/tools.go`, new `backend/mcp/permission_admission_test.go`, this evidence, work-order status line, roadmap F01 row, handoff, active state, session memory, tranche record, implementation status.

## Disposition

`REVIEW_PENDING` for Codex independent REVIEW. Claude does not self-approve; no push, merge, deployment or FREEZE.
