# CCMAI-RUNTIME-020 / F01-A — HTTP agent permission admission

Status: SPEC accepted for bounded WORK_ORDER. Date: 2026-09-30. Risk: R2 (authorization and access to tenant data / job execution). Source: [F01 review](../reviews/CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md), [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md). F01 remains open beyond this tranche because MCP has a separate tenant-only tool router.

## Observed contract

`backend/api/router.go:237-243` applies JWT to `/agents`, but `AgentRun` and `AgentQuery` in `backend/api/handlers/agents.go` check only a `user_tenants` membership row. Ordinary channel/message/job routes use `middleware.RequirePermission`. The HTTP agent run path can dispatch sync or analysis; query returns tenant-scoped conversation, message or result rows. Existing R011 admission tests use member permissions `{}` and direct handler calls, so they must be updated for a successful authorized fixture without weakening their config-failure checks. `backend/mcp/handlers.go` independently uses tenant membership for MCP tools, including `cqa_trigger_job`; it needs a later, separately bounded authorization tranche.

## Intended HTTP contract

1. After request validation and tenant membership, the server checks the stored role/permissions for the **requested tenant**, using the same `owner`/`admin` bypass and member permission parsing semantics as tenant routes. JWT alone or a tenant membership row alone does not grant a member any agent data or execution permission. Missing/malformed permission JSON and unrecognized roles fail closed. Never trust a role/permission/tenant claim supplied in action parameters.
2. The exact supported pair determines required permissions. All named permissions are conjunctive:

   | Agent | Action or query resource | Required tenant permissions |
   |---|---|---|
   | `cqa.sync` | run `sync_all`, `sync_channel` | `channels:w` and `messages:w` |
   | `cqa.qc` | run `analyze_quality` | `jobs:w` and `messages:r` |
   | `cqa.classify` | run `classify_conversations` | `jobs:w` and `messages:r` |
   | `cqa.sync` | query `conversations`, `messages` | `messages:r` |
   | `cqa.qc` | query `violations` | `jobs:r` |
   | `cqa.classify` | query `tags` | `jobs:r` |

   These rights reflect current side effects: sync updates channel state and writes messages; analysis reads conversations/messages and creates job results; query reads the corresponding data. Reuse or extract the existing permission decision so HTTP agent and tenant routes do not silently diverge. A member with only one of a two-right pair is denied.
3. Unknown agent names remain 404 after tenant admission. An unsupported run action or query resource is rejected before config load, dispatch or data read; it never falls through to a broader agent-level permission. Keep malformed request 400 and nonmember 403 ahead of configuration loading. Known valid action/resource with insufficient rights returns generic 403 without disclosing data, credentials or permission JSON. Owner/admin remain admitted for supported pairs; their action/resource still must be recognized. `GET /agents`, `/agents/capabilities` and `/:agentName/health` remain JWT-only discovery/health endpoints and must not return tenant data.
4. Preserve existing permitted execution behavior, synchronous response shape, tenant predicates, R011 config admission, sync single-flight and result integrity. This tranche changes HTTP agent admission, not the engine, MCP, ordinary tenant-route policy, provider routing or frontend.

## Acceptance evidence

- Disposable MySQL tests with a real `user_tenants` row for each role/permission case. For all six supported action/resource pairs, show a member missing each required right gets 403, and a member with the complete rights reaches the same permitted path. Test owner/admin, nonmember, malformed/missing permission JSON, unknown agent and unsupported action/resource. Include a cross-tenant attempt where the user has rights in one tenant but none in the requested tenant.
- On denied run, observe zero config loads, zero dispatcher calls, zero job-run inserts, no channel-state write and zero outbound attempts. Preserve a positive detector proving the same observers fire on an authorized run. On denied query, prove no returned tenant rows and, preferably, no query-handler execution; prove authorized tenant-scoped data returns while another tenant's rows do not.
- Run focused new tests and existing `AgentRun` config-admission tests, then the full backend suite on disposable MySQL with `TEST_DB_DSN`; use the R019 gate on a full JSON log if available. Document cleanup and exact test/gate results. Synthetic dispatchers/transport are valid for this application authorization check; no provider call or CVF runtime-governance claim is made.
- Independently inspect the MCP router and record that its tenant-only tools remain F01-B. No F01 complete/FREEZE claim until both surfaces and public CI evidence are settled.

## Decision boundary

The matrix is intentionally explicit; changes to required rights, supported action names, or MCP scope require a SPEC/work-order amendment before BUILD. The `query:scores` and `query:rules` capability labels have no handler implementation today; do not authorize invented data endpoints under this work order. No live channel/provider call, persistent DB, deployment, push, API key or secret change is authorized.
