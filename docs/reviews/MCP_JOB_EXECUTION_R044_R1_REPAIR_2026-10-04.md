# R044 repair round 1 evidence: notification-tail regression and reproducible worker receipts

Date: 2026-10-04 (Asia/Saigon). Role: Claude REPAIR_WORKER / repair BUILD COMMIT_STEWARD. Risk ceiling R2.
Status: repair complete, **REVIEW_PENDING** for independent Codex re-review. The worker does not set REVIEW_PASS or FREEZE.
Order: [R044](../work_orders/CCMAI_RUNTIME_044.md). Findings: [independent review](CCMAI_RUNTIME_044_INDEPENDENT_REVIEW_2026-10-04.md) R044-R1-01..02. Original evidence: [BUILD](MCP_JOB_EXECUTION_R044_BUILD_2026-10-03.md), unchanged.
Machine receipts: `docs/reviews/probes/r044_r1_worker_receipts.json` (counts, names, full hashes, diffs; no raw logs). Synthetic disposable local application evidence only: **not CVF AI governance proof**.

## Identity and scope

- Seed `CVF_SESSION/authority/CCMAI-RUNTIME-044.json`, first committed at `665f2e5ab780cbe8a1d374ccba676c1e926fbb47` (= baseCommit), unchanged.
- Repair acknowledgment and BUILD phase sync before any edit: `ac10f30`. Repair source commit (**buildCommit**): `b4ec91ea03a9f247e209389c3792c86494eac8b3`. Original BUILD `48918e2f953dbb32e7f5f159b1eceab7dfe3848f`.
- Changed source: only `backend/mcp/trigger_execution_test.go` and `backend/api/handlers/job_dispatch_shared_test.go` (both already in the seed allowedPaths), +375/-3 lines. **No product source, engine, notifications, dependency or go.mod change.** Production blob hashes equal the independent review's (dispatch.go `9e7a7ec41ba9...`, mcp/handlers.go `502400023232...`, mcp/tools.go `a84c43502741...`, api/handlers/jobs.go `2e850ebad1e3...`).
- Full committed-blob sha256 values are in the receipts (`repairGroupedRuns.*.sourceBlobSha256`). The containers ran the CRLF working copies of the same content (`core.autocrlf=true`).

## R044-R1-01 - maintained notification-enabled regression

Both packages replace `http.DefaultTransport`, which the Telegram notifier's nil-Transport client uses, with an in-process responder for one synthetic destination. It never delegates, counts every other request as `foreign` and fails it. No engine export or source change.

| Test | Package | What it pins |
| --- | --- | --- |
| `TestMountedTriggerOwnershipHeldThroughNotificationTail` | mcp | Mounted MCP, shared worker and real Analyzer reach exactly one intercepted send. While the send is parked the accepted run row is terminal (`success`, finished) with the correct job and tenant and the local owner is still held. A second mounted call is `job_already_running` with no new reservation, worker start or provider call. No notification log and no notified mark exist before the send returns. After release: exactly one `sent` log bound to the accepted run id, job, tenant and recipient; every result of the run marked notified; one matching send and zero foreign requests; joined cleanup; the slot is reusable (next call accepted). |
| `TestHTTPTriggerIsBusyWhileMCPHoldsTheNotificationTail` | handlers | The same MCP-held tail observed from HTTP: the HTTP trigger returns 409 `job_already_running`, a second MCP call is busy, run and provider counts are unchanged; after the send the log and marks are bound to the MCP run and HTTP is admitted (202) on the freed slot. |
| `TestNotificationDetectorReportsNoSendWithoutConfiguredOutputs` | mcp | **Finite named no-output control** for the positive-send detector: the identical harness with no output configured; the real run completes and the detector reports no send, no foreign request and no log. |
| `TestNotificationEnabledCancellationAndRejectionSendNothing` | mcp | Preserved zero-notification controls with a notification-enabled job: an extra-argument rejection reserves, starts and sends nothing; an owner cancellation ends `cancelled` with no send and no log. |

The reviewer probe under `docs/reviews/probes` is untouched replay evidence; equivalent coverage now lives in the maintained suite. The test-file header comment that said the notification seam cannot be intercepted was updated.

Runs of the four tests on the repair tree (`go test ./mcp ./api/handlers -run <4 tests> -count=1 -p 1 -json`, disposable synthetic MySQL, one container per run): first run 4 pass; three further independent containers (`BASE` in the receipts) 4 pass, 0 fail, 0 skip each.

### Detector controls (scratch copies of `backend/`, one exact change each; real tree never mutated)

Selection for N01-N07: the four tests above (N05e: the engine ownership selection). Each mutated run is followed by byte restoration of the scratch file, a full-hash equality check against the baseline and a rerun of the same selection on the restored copy. Full hashes and unified diffs are in the receipts.

| ID | Change | File | Baseline -> mutated sha256 (12) | Mutated run (events pass/fail/skip) | Named failing tests | Outcome | Restored sha256 = baseline | Restored run |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| N01 | configured notification never sent | `backend/engine/analyzer.go` | `7b6f715822b8` -> `5d9699c8deaf` | 2/2/0 | h:TestHTTPTriggerIsBusyWhileMCPHoldsTheNotificationTail; m:TestMountedTriggerOwnershipHeldThroughNotificationTail | KILLED | yes | 4/0/0 |
| N02 | ownership released before the notification tail | `backend/engine/analyzer.go` | `7b6f715822b8` -> `a7770101d98b` | 2/2/0 | h:TestHTTPTriggerIsBusyWhileMCPHoldsTheNotificationTail; m:TestMountedTriggerOwnershipHeldThroughNotificationTail | KILLED | yes | 4/0/0 |
| N03 | notification log not bound to the run | `backend/notifications/dispatcher.go` | `a8fd21d5c1a8` -> `14074eee50d0` | 2/2/0 | h:TestHTTPTriggerIsBusyWhileMCPHoldsTheNotificationTail; m:TestMountedTriggerOwnershipHeldThroughNotificationTail | KILLED | yes | 4/0/0 |
| N04 | results never marked notified | `backend/notifications/dispatcher.go` | `a8fd21d5c1a8` -> `f4384dc4eb47` | 0/0/0 | none | BUILD_ERROR | yes | 4/0/0 |
| N04b | results never marked notified (compiling variant of N04, which failed to build: 'declared and not used: now') | `backend/notifications/dispatcher.go` | `a8fd21d5c1a8` -> `bbf0d2f34aa4` | 2/2/0 | h:TestHTTPTriggerIsBusyWhileMCPHoldsTheNotificationTail; m:TestMountedTriggerOwnershipHeldThroughNotificationTail | KILLED | yes | 4/0/0 |
| N05 | cancelled run is allowed to notify | `backend/engine/analyzer.go` | `7b6f715822b8` -> `c34c21be66cd` | 4/0/0 | none | SURVIVED | yes | 4/0/0 |
| N05e-attempt1 | cancelled run is allowed to notify (engine ownership selection as the detector) [HARNESS DEFECT: run_focus.ps1 did not apply the engine selection; this attempt ran the four notification tests, i.e. it duplicates N05] | `backend/engine/analyzer.go` | `7b6f715822b8` -> `c34c21be66cd` | 4/0/0 | none | SURVIVED | yes | 4/0/0 |
| N05e | cancelled run is allowed to notify (engine ownership selection as the detector) | `backend/engine/analyzer.go` | `7b6f715822b8` -> `c34c21be66cd` | 53/1/0 | e:TestCancelAfterPartialProgressDoesNotNotify | KILLED | yes | 54/0/0 |
| N06 | extra-argument rejection removed | `backend/mcp/handlers.go` | `502400023232` -> `6bd8db381328` | 3/1/0 | m:TestNotificationEnabledCancellationAndRejectionSendNothing | KILLED | yes | 4/0/0 |
| N07 | busy admission bypassed (success on ErrBusy) | `backend/jobdispatch/dispatch.go` | `9e7a7ec41ba9` -> `9310f4c0f2da` | 2/2/0 | h:TestHTTPTriggerIsBusyWhileMCPHoldsTheNotificationTail; m:TestMountedTriggerOwnershipHeldThroughNotificationTail | KILLED | yes | 4/0/0 |

Reading the table: N01, N02, N03, N04b and N07 are killed by the two positive tests on named assertions (never sent, early release, unbound log, missing notified marks, bypassed busy); N06 is killed by the rejection control. **N04 did not compile** (`declared and not used: now`), so it is a BUILD_ERROR, neither a kill nor a survivor; N04b is its compiling variant. **N05 survived** the MCP/HTTP selection: the cancelled single-conversation run analyzes no result, so allowing a cancelled run to notify is not observable there. The same mutant is killed by the existing engine ownership selection (`engine:TestCancelAfterPartialProgressDoesNotNotify`, run N05e). The first N05e attempt was a harness defect (the engine selection was not applied, so it duplicated N05) and is retained. The MCP/HTTP cancellation check therefore stays a preserved control, not a mutation-discriminated one.

## Grouped reruns on the repair tree

Uncached, `-count=1 -p 1 -json`, disposable synthetic MySQL, cached modules read-only, `GOFLAGS=-mod=readonly GOPROXY=off GOTOOLCHAIN=local CGO_ENABLED=0`, golang:1.26-alpine, mysql:8.0.

| Run | Exit | Events pass/fail/skip | Top-level / subtests (pass) | Skips |
| --- | --- | --- | --- | --- |
| `go test ./jobdispatch ./mcp ./api/handlers` | 0 | 393 / 0 / 0 | 190 / 203 | none (prior 389 = 186 + 203; +4 new top-level tests) |
| `go test ./engine -run '^Test(Admission\|Cancel\|Terminal\|Publication\|StaleOwner\|EarlyFailures\|ProviderPanic\|Timeout\|UnresolvedRunning\|NoLockInversion\|Cron\|Scheduler\|Reserved\|Reservation)'` | 0 | 54 / 0 / 0 | 28 / 26 | none (targeted selection, not a full-engine certification) |
| `go build ./...` and `go vet ./...` (local cached modules, `-mod=readonly`, offline) | 0 and 0 | - | - | go.mod unchanged |

Package seconds, log digests and exact commands are in the receipts. A named Docker volume `ccma-r044-gocache` holds only the Go build cache (added for speed after a roughly five minute cold baseline); it is task-owned and removed at the end of the round.

## R044-R1-02 - reproducible worker mutation and grouped-run receipts

The original BUILD worker's scratch artifacts survived and were **recovered read-only**. No historical record is fabricated and no later replay is relabeled as original. Sources: the worker's `mutate.py`, `campaign.ps1`, `run_isolated.ps1`, `campaign_summary.txt` and the per-mutant `go test -json` logs (their sha256 values are in the receipts). For each of the 15 original mutants the receipts hold the exact replacement (old and new text), the full baseline and mutated sha256, the campaign's logged hash prefixes, the log sha256, and top-level, subtest and skip counts with every failing test name.

Verification performed now, for all 15: (a) the surviving mutated copy equals the BUILD blob plus exactly the recorded replacement, byte for byte; (b) the baseline hash equals the campaign's logged prefix; (c) every campaign copy differs from BUILD only in the mutated file and the two `cvf-allow-secret-fixture` comments added later in `48918e2`.

Original command (per `campaign.ps1`): `go test ./jobdispatch ./mcp ./api/handlers -count=1 -p 1 -json` in golang:1.26-alpine with disposable mysql:8.0 and the same environment as above. Counts are test events (subtests included) recomputed from the surviving logs; they match the worker's table.

| ID | File | Baseline -> mutated sha256 (12) | Events pass/fail/skip | Top-level fail / subtest fail | Result |
| --- | --- | --- | --- | --- | --- |
| M01 | `backend/mcp/handlers.go` | `502400023232` -> `4c7111974f93` | 386/3/0 | 3 / 0 | KILLED |
| M02 | `backend/jobdispatch/dispatch.go` | `9e7a7ec41ba9` -> `7841cbe8191d` | 382/7/0 | 5 / 2 | KILLED |
| M03 | `backend/jobdispatch/dispatch.go` | `9e7a7ec41ba9` -> `9310f4c0f2da` | 370/19/0 | 16 / 3 | KILLED |
| M03b | `backend/jobdispatch/dispatch.go` | `9e7a7ec41ba9` -> `c21963053c58` | 184/53/0 | 39 / 14 | KILLED |
| M04 | `backend/jobdispatch/dispatch.go` | `9e7a7ec41ba9` -> `5eea063699a3` | 370/19/0 | 13 / 6 | KILLED |
| M05 | `backend/jobdispatch/dispatch.go` | `9e7a7ec41ba9` -> `4d2887d42daf` | 382/7/0 | 5 / 2 | KILLED |
| M06 | `backend/api/handlers/jobs.go` | `2e850ebad1e3` -> `e5f02b6c6d79` | 379/10/0 | 4 / 6 | KILLED |
| M06b | `backend/api/handlers/jobs.go` | `2e850ebad1e3` -> `14fc8ddf2da4` | 377/12/0 | 4 / 8 | KILLED |
| M07 | `backend/mcp/handlers.go` | `502400023232` -> `0064d170b873` | 368/21/0 | 11 / 10 | KILLED |
| M08 | `backend/mcp/handlers.go` | `502400023232` -> `6bd8db381328` | 386/3/0 | 3 / 0 | KILLED |
| M09 | `backend/jobdispatch/dispatch.go` | `9e7a7ec41ba9` -> `8402d6ff75ff` | 385/4/0 | 4 / 0 | KILLED |
| M10 | `backend/jobdispatch/dispatch.go` | `9e7a7ec41ba9` -> `53d0b21eb02c` | 367/22/0 | 16 / 6 | KILLED |
| M11 | `backend/jobdispatch/dispatch.go` | `9e7a7ec41ba9` -> `761341c6508b` | 388/1/0 | 1 / 0 | KILLED |
| M12 | `backend/mcp/handlers.go` | `502400023232` -> `24aa9b30c63b` | 387/2/0 | 2 / 0 | KILLED |
| M13 | `backend/mcp/handlers.go` | `502400023232` -> `41b5fd37a31d` | 388/1/0 | 1 / 0 | KILLED |

**Restoration model (honest limit):** each original mutant ran on a disposable scratch copy, so the working tree was never mutated and the campaign had no per-mutant restored-baseline run. Its restoration evidence is the campaign line "real file untouched `<prefix>`" plus the baseline hashes above, with the pre-campaign real-tree run and the post-marker MCP run as surrounding baselines. The replay above does have per-mutant restored-baseline runs, but it is **new evidence for new mutants (N-series)**, not a replay of M01-M13.

Other recovered receipts (in the receipts file):

- **Old-source control, first attempt** (harness defect, not a kill): 10 pass events; the mcp and handlers packages failed to build and ran nothing (a newline-mangling PowerShell copy).
- **Old-source control, corrected** (baseline `665f2e5` mcp/handlers.go, mcp/tools.go and api/handlers/jobs.go with the R044 tests; mcp only, since handlers cannot compile against the baseline jobs.go): 19 pass / 22 fail events = 10 pass / 18 fail top-level plus 9 pass / 4 fail subtests. The BUILD evidence wording "22 failing tests, 19 passing" counted events including subtests.
- **Final grouped real-tree run** (command inferred from the package events: jobdispatch, mcp, handlers, engine; predates the two marker comments): 836 pass / 0 fail / 3 skip events = 397 pass / 1 skip top-level plus 439 pass / 2 skip subtests. Named skips, all engine Zalo sync and unrelated: `TestZaloSyncMessageTokenRefreshPersistsAndRetriesSameOffset`, `TestZaloSyncMessageFailureIsPartialAndRetryCompletes/HTTP_500_(not_a_connection_error)`, `TestZaloSyncMessageFailureIsPartialAndRetryCompletes/actual_connection_error`.
- **Final MCP run after the marker edit**: 41 pass / 0 fail / 0 skip events (28 top-level).
- The 15-mutant campaign is now tied to recovered logs but remains worker evidence, not independent reviewer proof.

## Retained incidents and process notes

- Original BUILD incidents unchanged and still in force: secrets preflight FAIL then source commit `dff77f3`, fixture-marker repair `48918e2`, OLD harness defect, go.mod touched and restored, failed-abort read-fault attempt, winner distribution, unsorted-import `gofmt -l` baseline.
- This round: the first BUILD continuity sync failed the tranche gate (history must end with the current status); it was fixed before the acknowledgment commit, and every commit was preceded by an explicit passing preflight (no commit chained after a failing check). Recovery-script assertion and quoting defects, a replay campaign accidentally started under Windows PowerShell 5.1 (UTF-16 logs; stopped, task containers and network removed, restarted under PowerShell 7), one tool-refused combined command, the N04 BUILD_ERROR and the N05e harness defect are all listed in `incidents` of the receipts. No result from an aborted run is used.
- Persistent application containers (`ccma-app-1`, `ccma-db-1`, `ccma-nginx-1`) were never inspected, entered or used. No `.env`, config or credential was read.

## NOT RUN and limits

Race detector NOT RUN (CGO disabled in the test image). Full engine and full backend suites NOT RUN this round (targeted engine ownership selection only; the original full run is the recovered final grouped run). Real provider, channel, live Telegram or email, config, external network, persistent or customer database, application start: NOT RUN and prohibited. The in-process transport observes only `http.DefaultTransport`; raw sockets and other clients are not observed. The engine finalizer and fallback log lines that print driver error text (pre-existing, engine out of scope) remain an open separate concern. No push, merge, deployment, FREEZE, REVIEW_PASS or CVF AI governance or hosted-readiness claim. Facebook/Zalo OA accounts remain parked; R043 and all prior dispositions are unchanged.
