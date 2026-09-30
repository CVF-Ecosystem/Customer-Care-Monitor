# CCMAI-RUNTIME-020 / F01-A BUILD evidence — HTTP agent permission admission

**Date:** 2026-09-30 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 · **Role:** IMPLEMENTATION_WORKER + COMMIT_STEWARD (Claude); independent REVIEWER: Codex
**Authority:** [SPEC](../specs/RUNTIME_AGENT_HTTP_PERMISSION_ADMISSION_F01_2026-09-30.md), [work order](../work_orders/CCMAI_RUNTIME_020.md), [F01 review](CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md).
Planning HEAD `91b8c96` (base `8cb55f8`). Synthetic dispatchers/transport and disposable MySQL only; no provider/channel call, persistent DB, secret, push or FREEZE. The pre-existing untracked `knowledge/_index.json` is not part of the commit.

## Source audit and plan

`AgentRun`/`AgentQuery` ([agents.go](../../backend/api/handlers/agents.go)) checked only a `user_tenants` row; tenant routes use `middleware.RequirePermission`. MCP (`backend/mcp/handlers.go`) independently checks membership only (`:101`, `:120`; `cqa_trigger_job` at `:82`) — **unchanged and still F01-B**.

Plan: (1) extract the tenant permission decision into `middleware.PermissionDenial(role, permsJSON, resource, action)` and make `RequirePermission` call it (same codes/log); (2) in the agent handler load the membership row for the **requested** tenant once (`tenantMembership`), keep the 400 → 403 `tenant_access_denied` → 404 unknown-agent order, then apply the explicit matrix; (3) update the R011 fixture's member permissions and per-agent actions without weakening its assertions.

## Implementation

- `backend/api/middleware/tenant.go`: `PermissionDenial` returns `""`, `no_permissions`, `invalid_permissions` or `permission_denied` exactly as before; `RequirePermission` logs and aborts as before.
- `backend/api/handlers/agents.go`: `agentRunPerms` / `agentQueryPerms` tables and `authorizeAgentOperation`. Only roles `owner`, `admin`, `member` are recognized (anything else, including empty/uppercase, is denied); rights are conjunctive; the 403 body is the generic `{"error":"permission_denied"}`; the log carries user, tenant, agent, op and a reason class only (never permission JSON or params). An unsupported run pair returns 400 `unsupported_action` before config load/dispatch; an unsupported query resource keeps its existing 400 `unknown resource: …` before any read. Unknown agent stays 404 after tenant admission (queries now also check the name before reading, same 404). Discovery/health handlers are unchanged (JWT-only).
- SPEC matrix as implemented (and written out independently in the tests): `cqa.sync` run `sync_all`/`sync_channel` → `channels:w`+`messages:w`; `cqa.qc` `analyze_quality` and `cqa.classify` `classify_conversations` → `jobs:w`+`messages:r`; queries `conversations`/`messages` → `messages:r`, `violations`/`tags` → `jobs:r`. `query:scores` / `query:rules` remain unimplemented and unauthorized.

## Tests (disposable MySQL, `-count=1 -p 1`)

- `agent_run_config_admission_test.go` (fixture only): member permissions now `{"channels":"rw","messages":"rw","jobs":"rw"}` and requests use a valid action per agent (`bodyFor`); all original assertions (config failure, order, real config validation, detector observers) are unchanged and pass.
- New `agent_permission_admission_test.go` (5 tests, 8 matrix rows): for every row, dropping each required right and read-only-where-write-required → 403 generic body, **0 config loads, 0 dispatches, 0 job_runs, channel unchanged, 0 outbound**, and no seeded row markers in denied query bodies; complete rights → 200 through the same path with the positive detector observers firing (channel `syncing`, 1 job run, 1 outbound; correct agent/action dispatched) and, for queries, only own-tenant rows (two tenants seeded with distinguishable markers) and a 403 for the other tenant. Owner/admin with empty/malformed/`{}` permissions are admitted for all supported pairs, and rejected for unsupported pairs (400) without config load. Bad permission data fails closed: empty, `{}`, malformed, non-string values, array, `null`, unrecognized role `guest`, empty role, `OWNER`. Cross-tenant: full rights in another tenant do not help in the requested tenant (and do work in their own tenant); nonmember stays 403 `tenant_access_denied` ahead of unknown agent; unknown agent 404 even for a member with no rights; permission is checked before config admission; denial log contains no permission JSON, params or resource names. Params claiming `role: owner` / permissions are ignored.
- Mounted routes with real JWT parsing: no token 401; member with a valid JWT but `{}` → 403 for run and query with nothing dispatched; `/agents`, `/agents/capabilities`, `/:agent/health` remain 200 without tenant data; a member with the exact rights runs and queries successfully.
- New `middleware/tenant_permission_test.go`: 9 semantic cases for `PermissionDenial` plus `RequirePermission` codes through the extraction.

## Non-vacuity mutations (disposable, restored; `go test ./api/handlers -run TestAgent`)

| Mutation in `agents.go` | Result |
|---|---|
| permission loop iterates `need[:0]` (no rights checked) | 15 failed |
| unrecognized-role check disabled | 1 failed (`…BadPermissionDataFailsClosed`) |
| query gate result ignored (`&& false`) | 7 failed (all query rows) |
| only the first required right checked (`need[:1]`) | 5 failed (the one-right-missing cases) |
| run gate result ignored (`&& false`) | 11 failed |

The source was restored byte-for-byte after each; the final diff contains only the intended change and a full-suite run followed.

## Gates

| Check | Result |
|---|---|
| Focused `./api/...` (disposable MySQL) | 199 pass / 0 fail / 0 skip |
| Full backend `go test ./... -json -count=1 -p 1` (disposable MySQL) | exit 0; 474 pass / 0 fail / 2 optional skips (was 454; +20 new) |
| R019 gate on the full JSON log | `GATE PASSED`, 0 invalid records, all five sentinels PASS, 0 DB-unavailable skips |
| `go build ./...`, `go vet ./api/...` | OK |
| catalog `-Check`, workspace doctor, `git diff --check` | PASS, 25/25, clean (CRLF warnings only) |
| Cleanup | disposable MySQL containers/networks removed (`ccma-r020-*` none left) |

A first focused run exposed a test-only fixture problem, not a source defect: repeated `db.Connect` calls (which never close their predecessor) exhausted the server's connection limit and made later packages' tests skip; the new tests now close the previous pool (`newPermFixture`) and share one fixture per test, after which the full run shows 0 DB-unavailable skips.

## Claim limits

Synthetic application-authorization evidence for the HTTP agent surface only. **F01 stays OPEN**: MCP tools (including `cqa_trigger_job`) still authorize by membership alone (F01-B); any other tenant-only route is unaudited here. No CVF-governance or provider claim; the F08 public GitHub Actions run remains unverified. Behavior change to note for reviewers: unsupported run actions now return 400 `unsupported_action` instead of dispatching and returning a 200 `status:error` body; members who previously passed with `{}` permissions now need the matrix rights.

## Changed set

`backend/api/handlers/agents.go`, `backend/api/middleware/tenant.go`, `backend/api/handlers/agent_run_config_admission_test.go`, new `backend/api/handlers/agent_permission_admission_test.go`, new `backend/api/middleware/tenant_permission_test.go`, this evidence, handoff, active state, session memory.

## Disposition

`REVIEW_PENDING` for Codex independent REVIEW. Claude does not self-approve; no push, deployment or FREEZE.
