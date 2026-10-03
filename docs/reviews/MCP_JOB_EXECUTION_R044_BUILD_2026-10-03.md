# R044 BUILD evidence: shared MCP/HTTP job dispatch

Date: 2026-10-03/04 (Asia/Saigon). Role: Claude IMPLEMENTATION_WORKER / BUILD COMMIT_STEWARD. Risk ceiling R2.
Status: BUILD complete, **REVIEW_PENDING** for independent Codex review. The worker does not set REVIEW_PASS or FREEZE.
Order: [R044](../work_orders/CCMAI_RUNTIME_044.md). SPEC: [JE-01..10](../specs/MCP_JOB_EXECUTION_R044_2026-10-03.md).

## Identity and scope

- Seed `CVF_SESSION/authority/CCMAI-RUNTIME-044.json` first committed at `665f2e5ab780cbe8a1d374ccba676c1e926fbb47` (= baseCommit); present at baseCommit and unchanged since.
- BUILD acknowledgment and phase sync (before any edit): `521aa5a`. Source commit `dff77f3967b146c9d0f6871855960d14e1879a00`. Fixture-marker fix `48918e2f953dbb32e7f5f159b1eceab7dfe3848f` = **buildCommit**.
- Changed source (seed `allowedPaths` only): new `backend/jobdispatch/dispatch.go`, `dispatch_test.go`; `backend/mcp/handlers.go`, `tools.go`, `permission_admission_test.go`, `trigger_contract_test.go`, new `trigger_execution_test.go`; `backend/api/handlers/jobs.go`, new `job_dispatch_shared_test.go`. Engine, schema, dependencies, UI, workflows, gates, parent CVF, other tests: untouched. Existing HTTP seams (`loadJobDispatchConfig`, `startTriggerJob`, `startTestRunJob`, `newTriggerAnalyzer`, `triggerJobParams`) were kept, so protected tests such as `job_config_admission_test.go` ran unchanged and passed.
- Final source sha256 (first 16 hex): dispatch.go `9e7a7ec41ba95855`, mcp/handlers.go `502400023232a569`, mcp/tools.go `a84c43502741769d`, api/handlers/jobs.go `2e850ebad1e335d1`, trigger_execution_test.go `9d097bb0e4ba067e` (comment-only marker change after the verified runs; see incidents).

## Design as built

`jobdispatch.Service.Dispatch(job, Params)`: config load (error or nil => `ErrStartFailed`, nothing reserved) -> `engine.ReserveJobRun(context.Background(), job, timeout)` -> `Start` (default `StartWorker`, bounded goroutine, panic recovered without logging its value) -> on start error `AbortReservation` -> returns the persisted `res.RunID()`. Errors: `ErrBusy`, `ErrMissing`, `ErrStartFailed` (aliases of engine errors). HTTP `TriggerJob` adapts its seams into a per-request `Service` and maps errors to 202/409/404/500; query parsing is unchanged. MCP `toolTriggerJob`: authorization (unchanged) -> tenant+job lookup (`Job not found`) -> any argument other than `tenant_id`/`job_id` => `invalid_run_parameters` -> `jobdispatch.Default.Dispatch(Params{Mode:"since_last"})` => `{"message":"job_triggered","run_id":...}` or tool errors `job_already_running` / `Job not found` / `job_start_failed`. No request context reaches the worker. Success means reserved + worker handed off, not completed analysis (stated in the published tool description).

## JE matrix (all on disposable loopback MySQL, synthetic rows)

| ID | Evidence (test names) |
| --- | --- |
| JE-01 | `TestMCPEachToolRequiresExactlyItsRights`, `TestMCPOwnerAndAdminBypassLettersButNotMembership`, `TestMCPBadPermissionData...`, `TestMCPSpoofedArguments...`, `TestMCPMountedRouteEnforcesTokenAndToolPermissions`, `TestTriggerAdmissionPrecedesLookup` (every denial: zero jobs queries, zero config loads/starts, zero runs; re-admitted control sees 1/1) |
| JE-02 | `TestTriggerLookupKeepsTenantPredicateAndGenericErrors` (SQL predicates + binds, foreign/unknown/empty/missing/non-string, forced error), `TestTriggerExtraArgumentsAreRejectedBeforeDispatch` (12 extra-arg shapes; auth and not-found win), `TestTriggerForcedReadErrorHasNoEffects` |
| JE-03 | `TestTriggerAdmittedDirectAndMounted`, `TestMountedTriggerRunsSharedWorkerWithPersistedIdentity` (exactly `{message, run_id}`; run_id = stored running row; completion results carry that id), `TestTriggerDescriptionAndToolsList` |
| JE-04 | `TestTriggerDispatchFailuresAreGenericToolErrors` (config error, nil config, busy, vanished job => `Job not found`, forced admission error, start failure; ToolResult errors only; unknown tool still -32602), jobdispatch `TestConfigFailureReservesNothing`, `TestAdmissionFailureIsGenericAndStartsNothing`, `TestMissingJobIsReportedAsMissing` |
| JE-05 | jobdispatch `TestConcurrentDispatchHasOneOwner`, `TestStoredRunningRowWithoutOwnerBlocks`; mcp `TestTriggerConcurrentMountedCallsHaveOneOwner`, `TestMountedTriggerSecondCallWhileRunningIsBusy`; handlers `TestMixedTransportsShareOneOwnerWithRealAnalyzer` (both orders), `...ConcurrentAdmissionHasOneWinner` (12 rounds, 4+4), `...SimultaneousDispatchHasOneWinner` (30 rounds, rendezvous so both transports are inside dispatch), `...DifferentJobsAreIndependent` |
| JE-06 | jobdispatch `TestStartFailureAbortsReservationAndReleasesOwnership` (row error, owner released, no results/notification/checkpoint), `TestFailedAbortKeepsStoredRowBlocking`; mcp `TestTriggerFailedAbortKeepsBlockingThroughMCP` (writes failed via Update callback; row never deleted) |
| JE-07 | `TestMountedTriggerWorkerOutlivesRequestContext`, `TestMountedTriggerWorkerFailureModes/{error,panic,cancel}` (terminal `error`/`error`/`cancelled`, owner released, panic value not stored), `TestMountedTriggerOwnershipHeldThroughCompletionTail` (terminal row stored while owner held; second call busy; slot free after), jobdispatch `TestDispatchReturnsPersistedReservationIdentity`, `TestDefaultTimeoutIsBounded`, `TestWorkerPanicClosesRunAndReleasesOwnership` |
| JE-08 | Whole handlers suite incl. unchanged `job_trigger_modes*`, `job_run_ownership`, `job_config_admission` tests (modes, full alias, dates, caps, 202/409/404/500/400, test-run, cancel): 0 failures |
| JE-09 | `TestTriggerRejectionsHaveNoWritesRunsDispatchOrOutboundRequests` (write callbacks, table checksums, positive controls for write, HTTP trap and reservation), `TestTriggerForcedErrorsOnMountedRouteHaveNoEffects` (config/nil/admission/extra-arg, mounted) |
| JE-10 | mounted MCP and HTTP through real disposable MySQL with injected synthetic config/provider; `TestMountedTriggerRunsSharedWorkerWithPersistedIdentity`, `TestMixedTransportsShareOneOwnerWithRealAnalyzer`, `TestProductionDispatchWiringUsesRealLoaderAndAnalyzer` (production seams are `config.Load` / `engine.NewAnalyzer`, `Default` has no overrides) |

Sanitized synthetic exchange (mounted MCP): request `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"cqa_trigger_job","arguments":{"tenant_id":"<synthetic>","job_id":"<synthetic>"}}}`; response result `isError` absent, one text item `{"message":"job_triggered","run_id":"<uuid of the stored running row>"}`. Synthetic provider reply: `{"verdict":"PASS","score":95,...}` labeled `test-double`. This is **application proof only, not CVF governance proof**.

## Commands, exits, counts (docker `golang:1.26-alpine`, disposable `mysql:8.0`, `-count=1 -p 1 -json`, GOPROXY=off, CGO off)

| Run | Result |
| --- | --- |
| jobdispatch (before marker edit) | 10 pass, 0 fail, 0 skip |
| mcp | 40 pass at first full run; 41 pass, 0 fail, 0 skip after the final forced-error test and marker edit |
| api/handlers (before shared tests) | 331 pass, 0 fail, 0 skip; shared/mixed tests 7 pass |
| Final uncached real tree: jobdispatch + mcp + api/handlers + engine | exit 0, 836 pass events (subtests included), 0 fail, **3 skip** (engine Zalo sync tests `TestZaloSyncMessageFailureIsPartialAndRetryCompletes/*` x2 and `TestZaloSyncMessageTokenRefreshPersistsAndRetriesSameOffset`, unrelated to this change), 11m52s |
| `go build ./...`, `go vet ./...` (GOPROXY=off, GOTOOLCHAIN=local) | exit 0 |

NOT RUN: race detector (CGO disabled in the image); any real provider, channel, notification, sync, customer or persistent DB, real config/`.env`, external network, application start; engine notification seam (unexported, engine edits prohibited), so test jobs use `output_schedule none` and no outputs and the tests assert `notification_logs` unchanged and no default-transport HTTP (raw sockets are not observed).

## Controls on scratch copies of `backend/` (real tree never mutated; byte hashes printed before/after, real file hash re-printed)

Old-source detector: baseline production `mcp/handlers.go`, `mcp/tools.go` (blob from `665f2e5`, byte-exact) with the new mcp tests: **22 failing tests, 19 passing**, including `TestMountedTriggerRunsSharedWorkerWithPersistedIdentity`, `TestTriggerAdmittedDirectAndMounted`, `TestTriggerDescriptionAndToolsList`. A first attempt (OLD) was inconclusive: my PowerShell copy mangled newlines and both packages failed to build; retained as a harness defect, not a kill. The handlers package cannot compile against baseline `jobs.go` (it references the new adapter), so only mcp is a semantic detector.

| Mutant | Change | Failing tests (killed) |
| --- | --- | --- |
| M01 | tenant predicate removed from MCP lookup | 3 (`TestTriggerLookupKeepsTenantPredicate...`, `...ExtraArguments...`, `...Rejections...`) |
| M02 | early success after failed start | 7 (jobdispatch start-failure/failed-abort, mcp dispatch-failures/failed-abort, handlers post-reservation setup) |
| M03 | busy admission bypassed (success on ErrBusy) | 19 |
| M03b | wrong-tenant reservation | 53 |
| M04 | request-lifetime context parents worker | 19 |
| M05 | missing abort on start failure | 7 |
| M06 / M06b | HTTP limit / date range not forwarded | 10 / 12 |
| M07 | denied MCP request still dispatches | 21 |
| M08 | extra-argument rejection removed | 3 |
| M09 | nil config accepted | 4 |
| M10 | returned run id differs from persisted id | 22 |
| M11 | run timeout unbounded | 1 (`TestDefaultTimeoutIsBounded`) |
| M12 | MCP mode changed to unanalyzed | 2 |
| M13 | not-found mapped to start failure | 1 |

All 15 patterns matched exactly once and compiled; **no survivors, no build errors, no timeouts** among the mutants. The mutants ran on the test bytes before the comment-only marker edit; the real tree was re-run (mcp 41/0/0) afterwards. A worker campaign is not independent reviewer proof.

## Incidents and observations (retained)

- `GOFLAGS=-mod=mod` vet once rewrote `backend/go.mod`; restored with `git checkout` immediately, never committed (status shows no go.mod change).
- First failed-abort test used a read fault; the engine's finalizer has a best-effort fallback `UPDATE` that bypasses it, so the row became `error`. Tests now fail UPDATEs. The engine fallback log line prints the driver error text (pre-existing, out of scope; this package's own line has none).
- Early test assumptions corrected: an admitted call issues several `jobs` queries (asserted `>=1`); one denial case first ran the admitted user and would now dispatch (fixed to a single call).
- Preflight `secrets` FAILED on two synthetic marker strings in `trigger_execution_test.go` right before the source commit; my chained command committed anyway (`dff77f3`). Fixed in `48918e2` by the sanctioned `cvf-allow-secret-fixture` marker; preflight then 7/7.
- In the simultaneous-dispatch test the MCP request won all 30 rounds and HTTP won all 12 rounds of the other concurrency test, so alternation is not claimed; exclusion (one accepted, others busy) is what is asserted.
- `gofmt -l` still lists four `mcp/*.go` files whose import blocks were already unsorted at baseline; not reformatted to keep the diff scoped.

## Limits and claims

Local application evidence on synthetic disposable fixtures. No CVF AI governance, model quality, live compatibility, hosted readiness or runtime/deployment claim; no push, merge or FREEZE. Facebook/Zalo OA accounts remain parked; R043 and earlier dispositions unchanged. Reviewer should confirm seed author/timing, rerun the mounted suites on an isolated fixture, and decide whether the engine fallback log text and the missing race run need follow-up.
