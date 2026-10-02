# CCMAI-RUNTIME-030 — F02-D Pancake message coverage: BUILD evidence

Status: REVIEW_PENDING / FREEZE_OPEN. Date: 2026-10-02. Role: IMPLEMENTATION_WORKER / COMMIT_STEWARD (Claude); independent REVIEWER: Codex (pending). Authority: [SPEC](../specs/RUNTIME_PANCAKE_MESSAGE_COVERAGE_F02D_2026-10-02.md), [work order](../work_orders/CCMAI_RUNTIME_030.md), dispatcher seed (not edited). Exact BUILD SHA is supplied in the active handoff after the local commit (never self-referential here). All results below are worker-reported, not yet independently reproduced.

## Scope and changed set

- `backend/channels/pancake.go`: message-only `FetchMessages` rewrite, new `ErrPancakeMessageCoverageIncomplete`, helper `sortMessagesBySentAt`; the unused `flattenOldestFirst` is removed. Conversation enumeration, `doRequest` (retry/throttle/8 MiB bound), mapping/redaction are untouched.
- `backend/channels/pancake_messages_test.go` (new), `backend/channels/pancake_test.go` (one legacy expectation adapted, see below).
- `backend/engine/sync_pancake_messages_test.go` (new), `backend/engine/sync_pancake_coverage_test.go` (fake gains `msgFn`/`msgLog`, position-aware default). No engine source edit.
- Session/status/order/tranche/roadmap records. Excluded pre-existing untracked: `knowledge/_index.json`, two Python bytecode files.

## Implementation summary

Omit `current_count` on the first request; later positions equal cumulative physical rows consumed (duplicates and old rows included; retries reuse the position). Success only on a valid explicit empty `messages` array after a final context check. Every row is decoded and validated (non-blank string `id`, parseable non-zero `inserted_at`) before dedupe/`since` filtering. A non-empty page with no new IDs fails; the 200-page budget allows no request 201. Output is deduplicated by first-seen ID and stably sorted by `SentAt`. Errors use fixed classes and page numbers; cancellation stays visible through `errors.Is` (`%w: %w`).

Adapted legacy expectations (disclosed, not deleted): `TestPancakeFetchMessagesPaginatesUntilSince` previously asserted one request when `since` fell in the first batch; it now asserts the exact IDs m50..m69 and four requests (three pages + explicit empty page). The engine fake's message handler now returns an empty page for positions after the first (the old fake replayed the same two rows, which the nonprogress rule correctly rejects); `msgReqs` for 105 conversations changed 105 → 210 (page + explicit empty page each).

## F02D-01..09 matrix (tests are in the two new files unless noted)

| ID | Evidence |
|---|---|
| F02D-01 | `TestPancakeMessageRequestInventory`: GET, escaped path `page%2F1%20conv%3Fx`, token present, first request omits `current_count`, positions `,3,3,5` (3 physical rows incl. old row; 429 retry repeats 3; 5 after a duplicate). Conversation tests (R023) pass unchanged. |
| F02D-02 | `TestPancakeMessagesEmptyFirstPageIsSuccess`, `…MalformedPagesFailClosed` (15 shapes: missing/null/object/string/number array, top-level null/array/string, null/non-object rows, malformed JSON, malformed `success`, explicit `success:false`, undecodable; each 1 request, safe error, message identity not the conversation identity), `…AbsentSuccessFieldIsCompatible`, cancellation test (terminal page). |
| F02D-03 | `TestPancakeMessagesInvalidRowsAreNeverFilteredAway`: 13 invalid-row shapes incl. blank ID on an old row, plus invalid repeat of a seen ID beside a new valid row (message must name the validation class), plus invalid row after a filtered one. |
| F02D-04 | `…SinceFiltersOutputWithoutEarlyStop`: exact IDs `boundary,newB,newA` over positions `,2,4,5` (old rows before/after eligible ones, equality included); zero since returns all five. |
| F02D-05 | `…DeduplicateAndOrderChronologically` (first-seen mapping `FIRST`, tie order tieA,tieB, order `c0,tieA,tieB,dup,c2b,c3`), `…RealShapedPayloadKeepsMappingAndRedaction` (7 fixture messages, sticker/file/agent mapping, no email/phone in raw data). |
| F02D-06 | `…NonprogressFailsClosed` (duplicate-only: 2 requests; alternating replay: 3; duplicate within page), `…OverlapWithNewIDsContinues`, `…PageBudgetBoundary` (199 rows + empty page 200 succeeds with 200 requests; 205-row chain fails with exactly 200 requests/200 diagnostic rows), `…LaterPageFailuresAreNeverSuccess` (8 failure kinds, 429 = 1+4 requests), `…TransportCauseCannotEchoRequestURL`, `…CancellationIsNeverSuccess` (before request: 0 requests; terminal page; between pages: 1 request; during retry backoff with a 1 h backoff so only cancellation can end it). Ordering is fixed by in-transport hooks, not sleeps. |
| F02D-07 | Engine `TestPancakeSyncMessageFailureIsPartialAndRetryCompletes` (disposable MySQL, actual adapter, closed synthetic transport), 3 modes: duplicate-only later page, missing array, HTTP 500 after valid messages. Observed: status `partial`; checkpoint unchanged; `triggerAfterSync` seam called 0 times and 0 `sync.completed` activity rows; failed conversation stored 0 messages; peer conversation stored exactly `ok_1`; request log `mf_bad@0,mf_bad@2,mf_ok@0,mf_ok@1` (no extra request); stored error free of token/URL/body marker/message ID; no unexpected hosts. |
| F02D-08 | Engine `TestPancakeSyncStoresExactEligibleMessageIDsAcrossPages`: stored IDs exactly `boundary,e1,e2,e3,e4` (since-boundary equality, overlap, two ineligible old rows) and `two_a,two_b`; counts 2/7; log `mc_0001@0,@3,@6,@8,mc_0002@0,@2`; success, checkpoint advanced, after-sync 1, other channel/tenant 0 messages; replay over the same window leaves IDs/counts, no duplicate rows. The failure test above then retries successfully (3 stored IDs for the formerly failed conversation), after-sync 1, replay idempotent, no duplicates. |
| F02D-09 | Below. |

## Original-source and mutation evidence

Original adapter (`git show HEAD:backend/channels/pancake.go`, plus a temporary shim declaring the new error symbol so tests compile; shim removed): 12 tests FAIL — `TestPancakeFetchMessagesPaginatesUntilSince`, `…RequestInventory`, `…CancellationIsNeverSuccess`, `…DeduplicateAndOrderChronologically`, `…InvalidRowsAreNeverFilteredAway`, `…LaterPageFailuresAreNeverSuccess`, `…MalformedPagesFailClosed`, `…NonprogressFailsClosed`, `…OverlapWithNewIDsContinues`, `…PageBudgetBoundary`, `…RealShapedPayloadKeepsMappingAndRedaction`, `…SinceFiltersOutputWithoutEarlyStop`. The empty-first-page, absent-success and transport-scrub tests pass on the original (positive controls). Engine tests against the original: `…StoresExactEligibleMessageIDsAcrossPages` FAIL (stored `[e3 e4]`), failure test FAIL for duplicate-only page and missing array (status `success`); the HTTP-500 mode PASSES on the original (existing engine semantics already treat a transport error as partial) and is retained as a regression, not as a detector.

Mutations (baseline SHA256 `4f7c613e8e1df876789059f0849b655f52c55c8b960e6b9ee8ebfc6ed0a935d3`, 20622 bytes, LF; exact match counts asserted, byte change verified, harness `finally` restores and verifies identity):

| Mutation | Matches | Result | Failing tests |
|---|---|---|---|
| M1 budget exhaustion returns success | 1 | KILLED | `PageBudgetBoundary` |
| M2 stop once an older-than-since row is seen | 1 | KILLED | `PaginatesUntilSince`, `RequestInventory`, `SinceFiltersOutputWithoutEarlyStop` |
| M3 missing array treated as success | 1 | KILLED | `LaterPageFailuresAreNeverSuccess`, `MalformedPagesFailClosed` |
| M4 validate after dedupe | 1+1 | first run **SURVIVED** (rc 0); test repaired; rerun KILLED | `InvalidRowsAreNeverFilteredAway` |

The M4 survivor: my invalid-repeat probe put the invalid repeat alone on its page, so dedupe-first code still failed on the nonprogress rule and the identity-only assertion passed. Repair (test only): place the invalid repeat beside a new valid row and require the validation message. The first-run survivor is retained here. After every harness run the source was byte-identical to baseline and the shim absent; baseline then 61 PASS, 0 FAIL.

## Regression and gate results

| Check | Result |
|---|---|
| `go test ./channels -count=1 -v` | 61 PASS lines (incl. subtests), 0 FAIL |
| Disposable MySQL (`scripts/test-backend.ps1`, mysql:8.0 on its own network), `./engine -Run Pancake` | PASS (219 s, before the final test edit); new engine tests also PASS in the full run |
| Full backend `./...` on disposable MySQL, final tree | all packages `ok`; 937 PASS lines, 0 FAIL, 2 SKIP (`TestFetchThatTuNguonNgoai`, `TestNoiCatS3`: external-network/S3, unrelated to Pancake or DB); zero DB skips. engine 433 s |
| `go build ./...`, `go vet ./...` | PASS |
| `gofmt -l` on touched Go files | clean (pre-existing `channels/adapter.go` listed, untouched) |
| Race detector | NOT RUN: `go test -race` requires cgo; CGO_ENABLED=0 and no C compiler on this host |

R013–R016/R023 focused engine regressions were exercised as part of the full engine package run (no separate focused subset was run). Docs build, catalog, doctor, preflight and gate-unit results are recorded in the active handoff verification section for this tranche.

## Fixture isolation, cleanup, limits

Only the repository's disposable-database script was used: its own Docker network and container, no host port or data, removed afterwards by the script; the persistent Compose database (`ccma-db-1`) and its app were not touched. No existing credentials, channel or provider were used; tokens in tests are synthetic; the synthetic transports permit only `pages.fm`/loopback and the engine fake records any other host (none observed). Cleanup relied on the script's own teardown; residual disposable containers/networks were checked in the handoff verification section.

Limits (NOT proven): live Pancake offset-snapshot stability or insertion/deletion during traversal; real provider ordering/retention; Facebook/Zalo messages; global F02; channel credentials; CVF runtime governance or provider readiness; hosted CI. Explicit-empty completion establishes only the local traversal contract under the fixture's assumptions. Mutation and original-source evidence is worker-run and awaits independent reproduction.
