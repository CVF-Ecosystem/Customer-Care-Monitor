# R035 BUILD evidence — truthful unavailable MCP trigger response

Date: 2026-10-03 (Asia/Saigon). Tranche `CCMAI-RUNTIME-035`. Risk ceiling R2. Role: Claude IMPLEMENTATION_WORKER / BUILD COMMIT_STEWARD (owner-transferred); acknowledgment recorded in the active handoff `CVF_SESSION/handoffs/AGENT_HANDOFF_MCP_TRIGGER_TRUTH_2026-10-03.md` **before the first source edit**, followed by BUILD synchronization and a passing 7/7 preflight. Independent REVIEWER: Codex. Status: BUILD complete, REVIEW_PENDING, FREEZE OPEN. Authority: [SPEC](../specs/MCP_TRIGGER_JOB_TRUTH_R035_2026-10-03.md), [work order](../work_orders/CCMAI_RUNTIME_035.md), dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-035.json` first committed at `4b714a35bc6f49abe508d3ec4f32504e500054ba` (not edited by the worker).

**Claim boundary.** Application contract tests on synthetic data in a disposable local MySQL. No provider, channel, credential, external network, persistent DB, queue, analyzer or notification was used; the tool still dispatches nothing. These tests are not CVF runtime-governance proof, not hosted readiness and not a FREEZE. R034 offline REVIEW_PASS / FREEZE_OPEN and R033/local-message FREEZE are inherited unchanged; the Pancake live packet stays PREPARED_NOT_DISPATCHED.

## 1. Source identity and changed set

- Baseline: dispatch commit `09b6326` (seed `4b714a3`). Exact BUILD commit SHA: recorded in `CVF_SESSION/tranches/CCMAI-RUNTIME-035.json` `buildCommit` by the follow-up documentation commit (a commit cannot contain its own SHA).
- Source files (SHA-256 of the committed bytes):

| File | Change | SHA-256 |
| --- | --- | --- |
| `backend/mcp/handlers.go` | `toolTriggerJob` only: keeps the own-tenant lookup, returns the fixed error; adds constant `triggerUnavailableText` | `95eeb36fac225846af0613d882a0626f8e413b9f60faad5e063a32ce9486e9fb` |
| `backend/mcp/tools.go` | `cqa_trigger_job` description only | `8f09c4064042736a636c0e032f94dbf9d5039a81ac2c0313982c17c9a8f9fe6b` |
| `backend/mcp/permission_admission_test.go` | trigger expectations: matrix `own` check, `expectedToolError`, `admittedOutcomeProblem`, mounted-route step | `3f39bbf54e2964a064e1c340c6e01870efb40d1b52468a19d495376fe9e7f376` |
| `backend/mcp/trigger_contract_test.go` (new) | MT-01..06 contract tests | `e6e33d9c19df9ded5a91b1db08f737d916f30b6b24406af12c55678c8c76c3dc` |

- Not touched: authorization helpers and `toolPolicies`, OAuth, server routing, other handlers, HTTP jobs endpoints, engine, schemas, dependencies, frontend, workflows, gate tooling, the authority seed. Pre-R035 `handlers.go` SHA-256 was `7ef1f27ff5dd58ff…`.

## 2. Behavior as built

`cqa_trigger_job` authorization is unchanged (conjunctive `jobs:w` + `messages:r`, owner/admin bypass letters only, tenant membership required). After admission the handler still runs `WHERE id = ? AND tenant_id = ?`; a miss or query error returns the generic `Job not found`. A found job now returns `ToolResult{isError:true, content:[{type:"text", text:"job_trigger_unavailable"}]}` with no JSON-RPC error, no job name/ID/tenant and none of the retired "triggered/queued" wording. Description: *"Currently unavailable over MCP: this tool does not start or queue a job. For an existing job in an authorized tenant it returns the error job_trigger_unavailable."* Name and input schema are unchanged.

Synthetic request/response (mounted `POST /mcp`, bearer token for a member with exactly the two rights):

```
-> {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"cqa_trigger_job","arguments":{"tenant_id":"<A>","job_id":"job-<A>"}}}
<- {"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"job_trigger_unavailable"}],"isError":true}}
```

Other-tenant job under an admitted tenant, unknown, empty, missing or non-string job ID, and a forced jobs-query error all return `{"content":[{"type":"text","text":"Job not found"}],"isError":true}`. Denials are unchanged: `permission denied` / `access denied: you don't have access to this tenant`, with zero `jobs` queries.

## 3. MT acceptance matrix

| ID | Committed test(s) | Observed |
| --- | --- | --- |
| MT-01 | `TestTriggerUnavailableDirectAndMounted` | member (exact rights), member (all rights), owner with empty permissions, admin with `not json` permissions: direct and mounted agree; exactly one content item, `isError=true`, text exactly `job_trigger_unavailable`; no `error` key; none of `triggered, queued, accepted, started, run_id, job_run, has been, status`; no job name, job ID, tenant IDs or SQL marker; the own-tenant `jobs` query observed once per call |
| MT-02 | `TestTriggerAdmissionPrecedesLookup`; unchanged R021 tests | missing either right, only either right, malformed permissions, unknown role, upper-case role, absent membership and cross-tenant owner: denial text unchanged, `jobs` queried 0 times, denial never equals the unavailable text; re-admission restores the lookup and the unavailable error |
| MT-03 | `TestTriggerLookupKeepsTenantPredicateAndGenericErrors` | SQL observer (after `gorm:query`) proves one `jobs` query containing `id = ?` and `tenant_id = ?` with bind values job ID and requested tenant; the other tenant's job exists in the DB but returns `Job not found`; unknown/empty/missing/non-string IDs and a forced query error return the same generic text, with no SQL/driver/job data |
| MT-04 | `TestTriggerNoWritesRunsOrOutboundRequests` | across unavailable (member, owner), not-found, rights-denied, tenant-denied and a mounted call: 0 GORM Create/Update/Delete callbacks, 0 default-transport requests, `CHECKSUM TABLE` unchanged for jobs, job_runs, job_results, messages, conversations, channels, notification_logs and tenants (user_tenants excluded: the test changes membership between calls), 0 job runs. Positive controls (a real GORM update and a default-transport request) are seen by the probes |
| MT-05 | `TestTriggerDescriptionAndToolsList` | real mounted `tools/list`: 12 tools; the trigger description says unavailable / does not start or queue / names the error and contains none of `immediately, manually trigger, run now, will run, queued for`; the other 11 descriptions equal an independently written golden set; trigger schema (type object, `tenant_id`, `job_id`, required both, `job_id` description) and policy unchanged |
| MT-06 | edits to `permission_admission_test.go`; `TestAdmittedOutcomeHelperRejectsArbitraryErrors` | matrix expectation and mounted step now require the exact fixed error; only the trigger may count an error as admitted and only with the exact text (`admittedOutcomeProblem`); `permission denied`, `Job not found`, `record not found`, trailing-space text, empty text, non-error text and a protocol error are rejected; a read tool cannot return the trigger error; all other R021 cases untouched |

## 4. Commands and results

All Go commands used the project root, `GOPROXY=off GOTOOLCHAIN=local` and `go -C backend`. Database: an official `mysql:8.0` container started only for this tranche (`--log-bin-trust-function-creators=1 --max_connections=1000`), published on `127.0.0.1` only, synthetic fixtures, removed afterwards; the running application stack (`ccma-*` containers) and its database were not used. Tests ran on the host Go toolchain with `TEST_DB_DSN` pointing to that container.

| Check | Result |
| --- | --- |
| Workspace doctor | 25/25 PASS before BUILD |
| `go test -count=1 -json ./mcp` before R035 edits | exit 0: 11 top-level / 10 subtests PASS, 0 FAIL, 0 SKIP |
| `go test -count=1 -json ./mcp` after | exit 0: 17 top-level / 10 subtests PASS, 0 FAIL, **0 SKIP** (6 new top-level tests) |
| New focused tests (`TestTrigger*`, `TestAdmittedOutcome*`) | 6 top-level PASS |
| `go -C backend build ./...`, `go -C backend vet ./...` | exit 0, exit 0 |
| `go test -race ./mcp` | **NOT RUN**: `go: -race requires cgo` (CGO disabled, no C compiler) |
| Full DB-dependent backend suite | **NOT RUN** (not required for this bounded change) |
| Live/provider/channel/network/GitHub Actions | **NOT RUN** (no authority; not applicable) |
| Docs build, catalog, diff check, default / PR-range / explicit preflights, gate unit tests | recorded in the BUILD-commit handoff entry (see section 8) |

## 5. Old-source detectors

The pre-R035 backend was extracted with `git archive` of the dispatch commit into an isolated directory (pre-change `handlers.go` hash verified) and received only the new/updated test files. Result, exit 1, semantic failures only (no build error, panic or timeout):

- `TestTriggerUnavailableDirectAndMounted`: `want tool error "job_trigger_unavailable", got isErr=false … "Job … has been queued for…`
- `TestTriggerAdmissionPrecedesLookup` (re-admission step), `TestTriggerLookupKeepsTenantPredicateAndGenericErrors` (own job step): same cause
- `TestTriggerDescriptionAndToolsList`: `description must state the tool is unavailable … "Manually trigger an analysis job to run immediately."`
- Updated R021 tests: `TestMCPEachToolRequiresExactlyItsRights/cqa_trigger_job`, `TestMCPOwnerAndAdminBypassLettersButNotMembership` (`owner//cqa_trigger_job: expected the fixed tool error …`), `TestMCPMountedRouteEnforcesTokenAndToolPermissions` (`member trigger … status triggered … want the fixed unavailable error`)
- `TestTriggerNoWritesRunsOrOutboundRequests` **passes** on the old source: it is a safety invariant (the old placeholder also wrote nothing), not a response detector; its sensitivity is shown by mutation MX6 below.

## 6. Mutation evidence (applied, restored, rerun)

Runner (outside the repository): exact one-match byte edits to `handlers.go`, `tools.go` or `permission_admission_test.go`, `go test -count=1 ./mcp` against the disposable DB, bytes restored in `finally`, hash equality verified for all three files, baseline before and after. A kill requires a failing behavioral assertion; build errors, panics and timeouts are INCONCLUSIVE.

First campaign (9 mutants): 8 KILLED, **MX9 SURVIVED** (mutating the shared helper to accept any error): production behavior was protected by exact-text tests, but the helper itself had no detector. I extracted `admittedOutcomeProblem` and added `TestAdmittedOutcomeHelperRejectsArbitraryErrors`, then re-ran the whole campaign on the final tests (the first-attempt results are retained outside the repository). Final campaign, baseline before and after PASS, restored hashes equal the section 1 values, **9 KILLED, 0 SURVIVED, 0 INCONCLUSIVE, 0 NOT_APPLIED**:

| ID | Mutation | Failing tests / first assertion |
| --- | --- | --- |
| MX1 | unavailable result without `IsError` (required mutation 1) | R021 matrix, owner/admin, mounted and `TestTriggerUnavailableDirectAndMounted`: `expected the fixed tool error … got isErr=false … text="job_trigger_unavailable"` |
| MX2 | restore `triggered`/`queued` JSON (required 2) | same tests: `got isErr=false … "Job JOB-R021-FORBIDDEN-… has been queued for execution"` |
| MX3 | remove the tenant predicate (required 3) | `TestTriggerLookupKeepsTenantPredicateAndGenericErrors`: `lookup must carry both id and tenant predicates: SELECT * FROM jobs WHERE id = ? …` |
| MX4 | restore the immediate-run description (required 4) | `TestTriggerDescriptionAndToolsList`: `description must state the tool is unavailable … "Manually trigger an analysis job to run immediately."` |
| MX5 | lookup error returns the driver text | `TestTriggerLookupKeeps…`: `other tenant's job id: want generic Job not found, got … "record not found"` |
| MX6 | unavailable path performs a GORM update | `TestTriggerNoWritesRunsOrOutboundRequests`: `3 GORM write callbacks on jobs during trigger calls` |
| MX7 | policy drops the `messages:r` right | R021 rights test: `missing messages:r: got isErr=true … text="job_trigger_unavailable", want … "permission denied"`; `TestTriggerAdmissionPrecedesLookup`; `TestTriggerDescriptionAndToolsList` (policy check) |
| MX8 | unavailable text appends the job name | exact-text tests: `text="job_trigger_unavailable: JOB-R021-FORBIDDEN-…"` |
| MX9 | shared helper accepts any error | `TestAdmittedOutcomeHelperRejectsArbitraryErrors` |

Mutant logs and hashes are held outside the repository and summarized here.

## 7. Limitations

- The write probe sees GORM Create/Update/Delete callbacks and table checksums; raw `Exec` writes and other processes are covered only by the checksums of the listed tables. The outbound trap replaces `http.DefaultTransport` and `http.DefaultClient`'s transport; it does not see raw sockets or other clients. `user_tenants` is excluded from checksums because the test legitimately changes it.
- The bearer-token bookkeeping table `o_auth_tokens` is tolerated in the write-callback probe (none was observed).
- Race detector NOT RUN (CGO unavailable). Full DB-dependent backend suite NOT RUN. Host `go test` was used against the disposable MySQL rather than the Docker golang image.
- The tool now never executes a job; real queue/analyzer wiring remains a separate objective. Nothing here is live, provider, governance or hosted-readiness evidence.
- Reviewer suggestions: replay MX1, MX3, MX6 and MX9 independently on the exact BUILD SHA, confirm 0 skips on your own disposable DB, and check that the policy table and OAuth/server files are byte-identical to the baseline.

## 8. BUILD identity and hand-back

Exact BUILD commit: `10ad83381ce76b86763c1ee06eab02ddf4734cac` (parent `09b6326`; seed `4b714a35bc6f49abe508d3ec4f32504e500054ba` unchanged). Its four source blobs hash to the section 1 values. The follow-up documentation commit records the SHA in the tranche record, moves the tranche to REVIEW_PENDING and changes no source. Pre-BUILD-commit checks: docs build PASS (first and only run, no dead links), catalog `-Write`/`-Check` PASS, `git diff --check` clean, default, `--base origin/main --head HEAD` and explicit changed-set preflight 7/7 PASS, gate unit tests OK; the same set is re-run for the follow-up commit and noted in the active handoff. Independent Codex REVIEW is next; no self-approval, push, merge, deployment or FREEZE.

## 9. R035-R1 repair evidence (Claude REPAIR_WORKER, 2026-10-03)

Scope: independent review at commit 82a9b7d returned CHANGES_REQUIRED for R035-R1-01 and R035-R1-02 ([review](CCMAI_RUNTIME_035_INDEPENDENT_REVIEW_2026-10-03.md)). Acknowledgment and BUILD synchronization were recorded in the active handoff and passed preflight 7/7 **before the first R1 edit**. Sections 1-8 above stay as written (the original BUILD `10ad83381ce76b86763c1ee06eab02ddf4734cac`, its evidence and its single survivor MX9); where they differ, this section is current.

### R035-R1-01 — forced jobs-read-error no-effects detector

Added `TestTriggerForcedReadErrorHasNoEffects` to `backend/mcp/trigger_contract_test.go` (the only source file changed; product source and `permission_admission_test.go` are byte-identical to `10ad83381ce76b86763c1ee06eab02ddf4734cac`, verified with `git diff`). Fixture membership is prepared before observation, then one call runs with the jobs query forced to fail. It asserts: generic `Job not found` and no protocol error; exactly one attempted `jobs` lookup; 0 GORM Create/Update/Delete callbacks; 0 default-transport requests; `CHECKSUM TABLE` equality for all nine listed tables **including `user_tenants`**; 0 job runs. New test file SHA-256 `d6b3dd7fafb81a0c0192cb9b0eb1f8994fb421a144a8e7df23925c892f5ab27b`.

Mutation proof (runner and rules as in section 6; exact one-match edits; bytes restored; baseline before and after PASS):

| ID | Mutation | Before the new test | With the new test |
| --- | --- | --- | --- |
| MY1 | error-only GORM `Update` on the forced-error branch (`err.Error() != "record not found"`) | **SURVIVED** the complete earlier suite (confirmed on the pre-R1 test file, matching the reviewer finding) | KILLED: `FORCED_ERROR_EFFECT: gorm writes=map[jobs:1] outbound=0` |
| MY2 | error-only raw `Exec` that changes a job name (invisible to the GORM callbacks) | **SURVIVED** the pre-R1 suite | KILLED: `FORCED_ERROR_STATE_CHANGE: table jobs changed (194339397 -> 2534546627)` |

The nine mutations from section 6 were re-run unchanged on the final tests: all KILLED (MX5 is now also caught by the new test). Final campaign: 11 mutants, **11 KILLED, 0 SURVIVED, 0 INCONCLUSIVE, 0 NOT_APPLIED**; restored hashes equal the committed values. The reviewer probe `docs/reviews/probes/r035_error_effect_probe_test.go` was used as replay evidence only; its behavior is incorporated, not copied into production code.

### R035-R1-02 — stale prose retired as historical

The active handoff "Open boundaries" paragraph, the order "Dispatch boundary" section, the SPEC "Intake and design" baseline paragraph and the memory dispatch sentence now carry explicit *historical (dispatch-time)* labels with the current facts beside them; SPEC Implementation truth, status, catalog and continuity describe BUILD `10ad83381ce76b86763c1ee06eab02ddf4734cac` as CHANGES_REQUIRED and the R1 repair as REVIEW_PENDING. Original evidence, NOT RUN entries and the MX9 survivor are preserved.

### Commands and results (project root, `GOPROXY=off GOTOOLCHAIN=local`, disposable `mysql:8.0` on loopback, synthetic data, removed afterwards)

| Check | Result |
| --- | --- |
| `go test -count=1 -json ./mcp` | exit 0: 18 top-level / 10 subtests PASS, 0 FAIL, **0 SKIP** (the reviewer counted 17/27 before this one added test) |
| `go build ./...`, `go vet ./...` | exit 0, exit 0 |
| `go test -race` | **NOT RUN**: `go: -race requires cgo` |
| Full DB-dependent backend suite; frontend; live/provider/channel/network/GitHub | **NOT RUN** (unchanged) |

Limitations are unchanged from section 7: the probes see GORM write callbacks, listed table checksums and the default transport/client only; checksums are not universal write history. Nothing here is live, governance or hosted-readiness evidence; no FREEZE is claimed.

## 10. R1 repair identity and hand-back

Exact R1 repair commit: `22204abc0a19841cd9c50ce03c0af032896d3b4e` (Git parent `82a9b7d16e622b4ee3162870f1e30ba9b099324e`; original BUILD `10ad83381ce76b86763c1ee06eab02ddf4734cac`; seed unchanged). The follow-up documentation commit sets the tranche `buildCommit` to this SHA and changes no source. Docs build, catalog, diff check, preflights and gate unit tests are re-run for that commit. Independent Codex re-review is next at this historical hand-back; no self-approval, push, merge, deployment or FREEZE.

Reviewer metadata correction (2026-10-03): the original section10 incorrectly called357d04b the repair parent; `git show -s --format=%P 22204ab` confirms82a9b7d. [Independent R1 re-review](CCMAI_RUNTIME_035_R1_INDEPENDENT_REREVIEW_2026-10-03.md) records the current disposition; worker test/mutation evidence and all NOT RUN/observation limits above remain unchanged.
