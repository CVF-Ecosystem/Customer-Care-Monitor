# CCMAI-RUNTIME-031 — F02-E Facebook message coverage: BUILD evidence

Status: REVIEW_PENDING / FREEZE_OPEN. Date: 2026-10-02. Role: IMPLEMENTATION_WORKER / COMMIT_STEWARD (Claude); independent REVIEWER: Codex (pending). Authority: [SPEC](../specs/RUNTIME_FACEBOOK_MESSAGE_COVERAGE_F02E_2026-10-02.md), [work order](../work_orders/CCMAI_RUNTIME_031.md), dispatcher seed (not edited). The exact BUILD SHA is recorded in the active handoff after the local commit (never self-referential here). Every result below is worker-reported and not yet independently reproduced. Meta Messenger endpoint documentation was NOT verified; this is the local safety contract only.

## Scope and changed set

- `backend/channels/facebook.go`: message-only changes. New `ErrFacebookMessageCoverageIncomplete`, `fbMaxMessagePages` (500), `fbMaxMessageResponseBytes` (8 MiB), message-only request path `doMessageRequest`, `messageEndpoint`, `validMessageNext`, rewritten `FetchMessages`, and `toSyncedMessage` (mapping moved verbatim from the old loop). `doRequest`, `HealthCheck`, `validNextURL`, `FetchRecentConversations`, credentials, Graph version (v21.0) and the shared client are untouched. `sortMessagesBySentAt` from pancake.go is reused, not edited.
- `backend/channels/facebook_messages_test.go` (new); `backend/engine/sync_facebook_messages_test.go` (new); `backend/engine/sync_facebook_coverage_test.go` (the fake gained `msgFn`/`msgLog`/`badTokens`; defaults unchanged). No engine source, Pancake or Zalo edit. The engine tests reuse storage/dispatch helpers from `sync_pancake_messages_test.go` without editing that file.
- Records: state/marker/handoff/status/order/SPEC/tranche/roadmap. Excluded pre-existing untracked: `knowledge/_index.json`, two Python bytecode files.

## Implementation summary

Initial GET `https://graph.facebook.com/v21.0/{escaped id}/messages` with the existing fields and limit 100; a blank conversation ID sends nothing. Every outbound URL is rebuilt from validated parts (host, escaped path, non-auth query) and the configured token replaces any provider link token. Next links must be absolute HTTPS on `graph.facebook.com` (no userinfo, fragment, opaque form or non-443 port) with the decoded path exactly this conversation's v21.0 messages path and a well-formed query; they are validated before any request. Cycle identity is the canonical path + query minus the access token (so order, percent-encoding and rotated tokens cannot evade it), seeded with the initial page. The 500th page may be terminal; a 500th page with another next fails with no request 501. The message-only client is a shallow copy of the injected client with redirects blocked (`ErrUseLastResponse`), so Location is never followed; non-2xx fails; bodies are bounded to 8 MiB and closed; no retry added. Rows are validated (object, non-blank string `id`, parseable non-zero `created_time`) before dedupe or `since` filtering; absent paging or paging without `next` is terminal, while null/non-object paging or a null/non-string/empty `next` fails even on an empty or old page; empty or duplicate-only pages with a safe new next continue. Output: first-seen mapping, inclusive `since`, ascending time, stable ties.

## F02E-01..09 matrix

| ID | Evidence |
|---|---|
| F02E-01 | `TestFacebookMessageRequestInventory`: GET, https, host, escaped path `t_1%2Fx%20y`, fields/limit on page 1, configured token exactly once on every request, provider token never in any outbound URL, non-auth parameter `__paging_token` and `after` preserved; `TestFacebookBlankConversationIDSendsNothing` (0 requests); `TestFacebookUnsafeNextLinksAreRejectedBeforeAnyRequest` (21 unsafe forms: other host, suffix/userinfo tricks, http/ftp/protocol-relative/relative/opaque, port 444, other conversation, conversation-list, v22.0/no version, traversal and encoded traversal, extra segment, fragment, malformed query/URL, uppercase host) each rejected with exactly 1 request total and 0 to any other host; positive controls `:443` and an encoded-equal path. |
| F02E-02 | `…MalformedPagesFailClosed` (14 shapes incl. missing/null/object/string data, null/string/array rows, malformed JSON, top-level array/null, Graph error with and without data; message identity distinct from R022's; 1 request; bodies closed); `…InvalidRowsAreNeverFilteredAway` (11 invalid-row shapes; invalid REPEAT of a seen ID beside a new valid row on a page without next; invalid old row beside a fresh row; both assert the validation class so the nonprogress/other guards cannot satisfy them). |
| F02E-03 | `…EmptyAndShortPagesWithNextContinue` (empty, short, empty, nonempty terminal = 4 requests, IDs `e1,e2`); `…TerminalForms` (5 terminal forms succeed in 1 request; 10 malformed paging/next forms on empty, old and nonempty pages fail with no second request). |
| F02E-04 | `…SinceFiltersOutputWithoutEarlyStop` (exact `boundary,newB,newA` over 3 requests; zero since returns 5); `…DeduplicateOrderAndMapping` (order `agent,photo,stk,dup,c2b,c3`, first-seen `FIRST` mapping, agent/attachment/sticker/RawData mapping unchanged). |
| F02E-05 | `…CyclesFailClosed` (direct, alternating, initial identity with rotated token, reordered query, percent-encoded query, token-only difference; requests 2/3/1/2/2/2 and no more); `…DistinctCursorsContinueDespiteEmptyOrDuplicatePages` (4 requests); `…PageBudgetBoundary` (terminal page 500: 500 rows/500 requests; page 500 with next: exactly 500 requests, 500 diagnostic rows, no request 501). |
| F02E-06 | `…RedirectsAreNeverFollowed`: 4 locations x 4 codes (301/302/307/308) plus a later-page redirect, each with the trap body a lenient client would trust; total requests 1 (2 for the late case) and 0 to `evil.example`; `…HTTPFailuresAndLateErrorsAreNeverSuccess` (9 failure kinds on page 2 and again on page 1: HTTP 500 with a valid array, 503/429 non-JSON, 404 JSON, Graph error, malformed JSON, missing data, oversize 8 MiB+, connection error; all safe, 2 diagnostic rows, bodies opened = closed); `…BodyReadFailureIsNotSuccess`; `…CancellationIsNeverSuccess` (before request: 0 requests; terminal page; between pages: 1 request, hook-ordered, no sleeps). Errors checked for token, provider token, `access_token`, marker, host and conversation ID. |
| F02E-07 | Engine `TestFacebookSyncStoresExactEligibleMessageIDsAcrossPages` (disposable MySQL, actual adapter, closed synthetic transport): stored `boundary,e1,e2,e3,e4` (empty intermediate page with next, overlap, boundary, two ineligible old rows) and `two_a,two_b`; counts 2/7; log `t_0000?-,t_0000?p1,t_0000?p2,t_0001?-`; 0 requests with a non-configured token, 0 other hosts; status success, checkpoint advanced past previous, after-sync seam 1 and 1 completion activity; sibling channel/tenant 0 messages; replay of the identical window unchanged, no duplicate rows. |
| F02E-08 | Engine `TestFacebookSyncMessageFailureIsPartialAndRetryCompletes`, 6 subtests after valid page 1: no data array, malformed row, unsafe next link, actual RoundTripper connection error, HTTP 500 with a valid array (labelled separately), redirect. Observed in each: status `partial`, checkpoint equal to previous, after-sync seam 0 and 0 completion activity, failed conversation stored 0 messages, peer stored exactly `ok_1`, log `t_0000?-,t_0000?p1,t_0001?-` (no extra request, `evil.example` never contacted), stored error free of tokens/host/body marker/message IDs; retry success stores `bad_new,bad_old,bad_older`, after-sync 1, replay unchanged, no duplicates. Existing R022 engine tests and R013–R016 ownership/cancellation tests ran in the full engine run below. |
| F02E-09 | Below. |

## Original-source and mutation evidence

Original adapter (`git show HEAD:backend/channels/facebook.go` plus a temporary declaration-only shim for the new symbols, removed afterwards): 14 tests FAIL — `BlankConversationIDSendsNothing`, `MessageCyclesFailClosed`, `MessageDeduplicateOrderAndMapping`, `MessageDistinctCursorsContinueDespiteEmptyOrDuplicatePages`, `MessageEmptyAndShortPagesWithNextContinue`, `MessageHTTPFailuresAndLateErrorsAreNeverSuccess`, `MessageInvalidRowsAreNeverFilteredAway`, `MessageMalformedPagesFailClosed`, `MessagePageBudgetBoundary`, `MessageRedirectsAreNeverFollowed`, `MessageRequestInventory`, `MessageSinceFiltersOutputWithoutEarlyStop`, `MessageTerminalForms`, `UnsafeNextLinksAreRejectedBeforeAnyRequest`; the cancellation, body-read and R022 tests pass (positive controls). Engine tests against the original: both new engine tests FAIL (stored `[e3 e4]` instead of five IDs; success instead of partial for contract-failure modes; the unsafe-link and connection-error modes also trip the host/token assertions because the original forwards the provider token and follows links). Note: the connection-error subtest fails on the original through the token/host assertion, not through status, so it is regression coverage for the connection-error path rather than an independent detector of it.

First original-source attempt: the first version of the budget probe had an unbounded chain, so the original (which has no budget) never terminated; I killed the stuck test process (this also made the in-flight mutation M1 report `INCONCLUSIVE`: `fork/exec … Access is denied`, i.e. a harness error, not a kill). Repair (test only): the chain transport errors after request 502, so a runaway fails instead of hanging. The original-source run and M1 were then repeated.

Mutations (baseline `facebook.go` SHA256 `8341b19347fda089e643a55246b8562d1cc816eba4906bb953c347d463c1aa23`, 18861 bytes, LF; exact match counts asserted, bytes verified changed, restored in `finally` and identity re-verified; hash re-checked after all runs):

| Mutation | Matches | Result | Failing tests |
|---|---|---|---|
| M1 old row returns early | 1 | first run INCONCLUSIVE (harness: Access denied, see above); rerun KILLED | `SinceFiltersOutputWithoutEarlyStop`, `TerminalForms` |
| M2 empty page returns success | 1 | KILLED | `DistinctCursorsContinue…`, `EmptyAndShortPagesWithNextContinue`, `TerminalForms` |
| M3 unsafe next not rejected | 1 | KILLED | `UnsafeNextLinksAreRejectedBeforeAnyRequest` |
| M4 terminal success after budget | 1 | KILLED | `PageBudgetBoundary` |
| M5 validation after dedupe | 1+1 | KILLED | `InvalidRowsAreNeverFilteredAway` |
| M6 redirects followed (policy removed) | 1 | KILLED | `RedirectsAreNeverFollowed` |
| M7 token in cycle identity | 1 | KILLED | `CyclesFailClosed` |
| M8 HTTP status unchecked | 1 | KILLED | `HTTPFailuresAndLateErrors…`, `RedirectsAreNeverFollowed` |

M1–M5 are the five required by F02E-09; M6–M8 are extras. No survivor occurred. M5 was killed on the first applied run because the invalid-repeat probe was built with the competing-guard rule from the R030 learning.

## Regression and gate results

| Check | Result |
|---|---|
| `go -C backend test ./channels -count=1 -v` (baseline harness) | 77 PASS lines (incl. subtests), 0 FAIL |
| `go -C backend build ./...`, `go vet ./...` | exit 0 / exit 0 |
| `gofmt -l` on touched Go files | clean |
| Full backend `./...` on disposable MySQL (`scripts/test-backend.ps1`), first run | all packages `ok` (959 PASS lines, 0 FAIL, 2 SKIP `TestFetchThatTuNguonNgoai`/`TestNoiCatS3`, unrelated) EXCEPT `./engine`: `panic: test timed out after 10m0s` (Go default; the engine package took 433 s in the previous tranche on this host, host was slower this time; the test running at the alarm was a Zalo sync test). Unexplained as a slowdown beyond the host being slower; no engine test failed. |
| Engine package rerun in two disjoint halves (`-Run '^Test[A-Ma-m]'`, `-Run '^Test[^A-Ma-m]'`; 94 + 115 = 209 top-level tests, equal to the 209 in the timed-out run's test table) | both `ok`: part A 239 PASS lines / 0 FAIL / 0 SKIP (197.5 s), part B 205 PASS lines / 0 FAIL / 0 SKIP (420.7 s); zero DB skips. R013–R016, R022, R023, R030 and R019 error-sentinel tests are inside these runs. A single un-partitioned engine run did not complete inside the Go default timeout; tooling was not changed. |
| Race detector | NOT RUN: `go test -race` needs cgo; CGO_ENABLED=0 and no C compiler on this host |
| Docker inventory after the runs | no `ccma-test-*` container or network; persistent `ccma-*` stack untouched |

Docs build, catalog, doctor, preflights and gate unit tests are recorded in the active handoff verification section.

## Fixture isolation and limits

Only the repository's disposable-database script was used (own network/container, removed by the script; verified absent). No real channel, provider, credential or customer data; tokens are synthetic sentinels; transports are closed `RoundTripper`s, and the engine fake records any host other than `graph.facebook.com` (none observed). Not proven: Messenger endpoint compatibility, live permissions/retention/ordering/cursor stability, Zalo messages, global F02, provider/governance or hosted CI. Nil-error means the local validated next-link contract completed under fixture assumptions. The attachment-download surface and shared `doRequest` remain outside this repair and are not certified. All mutation/original-source evidence is worker-run and awaits independent reproduction.
