# CCMAI-RUNTIME-032 — F02-F Zalo message coverage: BUILD evidence

Status: REVIEW_PENDING evidence (worker-reported; not independently reviewed, not FREEZE). Date 2026-10-03. Worker: Claude IMPLEMENTATION_WORKER / BUILD COMMIT_STEWARD. Authority: [SPEC](../specs/RUNTIME_ZALO_MESSAGE_COVERAGE_F02F_2026-10-03.md), [work order](../work_orders/CCMAI_RUNTIME_032.md), immutable seed `CVF_SESSION/authority/CCMAI-RUNTIME-032.json` (committed at `6983a891b58272eb74e6ae0b793a71a55d1d8f8e`, not edited). Exact BUILD SHA is supplied in the following documentation commit to avoid self-reference.

Every statement here is bound to the local application contract exercised with synthetic transports and a disposable MySQL. Nothing here is live Zalo behaviour, a provider-governance receipt, hosted CI or a global F02 claim.

## Scope and changed set

| Path | Change |
|---|---|
| `backend/channels/zalo_oa.go` | `FetchMessages` rewritten as the message-only traversal; new `parseZaloMessagePage`, `doMessageRequest`, `ErrZaloMessageCoverageIncomplete`, `zaloMessagePageSize`, `zaloMaxMessagePages`. `extractZaloDataArray` (no remaining caller) removed. `FetchRecentConversations`, `doRequest`, `doRequestRaw`, `refreshToken`, `HealthCheck`, credentials, endpoint and OAuth protocol unchanged. |
| `backend/channels/zalo_messages_test.go` | New F02F-01..06 probes (adapter, synthetic RoundTripper). |
| `backend/channels/zalo_coverage_test.go` | One retained R024 assertion superseded by the new contract (see "Retained-assertion changes"). |
| `backend/engine/sync_zalo_messages_test.go` | New F02F-07/08 actual adapter + SyncEngine + disposable MySQL acceptance. |
| `backend/engine/sync_zalo_coverage_test.go` | R024 fixture extension: the fake serves message histories by physical offset with an explicit empty page and supports failure/expiry hooks; one retained request-count assertion updated (see below). |
| session / status / order / SPEC / roadmap / tranche / catalog records | BUILD, then REVIEW_PENDING synchronization. |

No engine product source, shared request/refresh/conversation/OAuth behaviour, other platform, UI, schema, workflow, tooling or dependency was changed. No real channel, provider or credential was used; no persistent database was touched.

## Implementation summary

`FetchMessages` requests `GET /v2.0/oa/conversation` with the unchanged JSON `data` query `{"count":10,"offset":N,"user_id":"<string>"}` and the `access_token` header. Offsets advance by the physical row count of each page (duplicates included); only a successful explicit empty array (`data: []` or `data: {data: []}`) ends the traversal, so short and duplicate-only pages continue. `since` is deliberately not applied (full history, as before).

- **Validation** (every physical row, before deduplication): numeric integral `error` 0; explicit array; object rows; nonblank string `message_id` (exact string kept, never float-converted); integral `src` 0/1; positive integral millisecond `time` up to year 9999. Present optional consumed fields (`message`, `type`, `from_display_name`, `url`, `thumb`) must be strings (null counts as mistyped); `links`, when present, must be an array of objects and the first link's `url`/`name` strings. An absent or empty `links` is valid.
- **Mapping** (inherited): text/default content type (a missing or empty `type` is text), `src` 0 → agent `OA`, `src` 1 → customer with `from_display_name`; attachment URL is `url`, else `thumb`, overridden by the first link's `url`, name from the first link else `<type>-<message_id>`; unknown nonempty types keep their attachments; `RawData` is the exact decoded row (numbers as `json.Number`).
- **Output**: one message per `message_id`, ascending by time, ties in first-seen order. Exact repeated rows collapse; a conflicting row with the same ID, and any repeated canonical page (sha256 of length-prefixed canonical rows, envelope text excluded), fail.
- **Budget and context**: at most 500 requested pages including the terminal empty page (499 nonempty + empty passes; 500 nonempty fails with no 501st request). Context is checked before each request and after each response, including the terminal empty page.
- **Request path**: `doMessageRequest` uses a copy of the adapter client with `CheckRedirect` returning `http.ErrUseLastResponse` (no destination request), the inherited 8 MiB bounded read (body closed), status/envelope handling and the inherited single `-216` refresh/retry through the unchanged `refreshToken`. Errors carry fixed safe text with `%w` causes (`errors.Is` for context, transport and callback causes); the log line stays `[zalo] API conversation: status=… len=…`.
- **Errors**: `ErrZaloMessageCoverageIncomplete` (distinct from the conversation sentinel) with the page number; rows returned with an error are diagnostic only, which the engine discards.

## F02F-01..09 matrix

| ID | Evidence |
|---|---|
| F02F-01 | `TestZaloMessagesTraverseWholeHistoryToExplicitEmpty` (146 messages incl. a short nonterminal page of 7, a short final page of 9, alternating data shapes, exact large string IDs, a far-old message; exact request inventory with count 10, string `user_id`, token, `data` JSON; offsets equal cumulative physical rows plus the empty page; nonzero `since` returns the identical history); `…ShortAndDuplicateOnlyPagesContinue` (offsets `[0 3 4 5]` over a duplicate-only page); `…ExplicitEmptyTerminalShapes` (three terminal forms). |
| F02F-02 | `…RejectMalformedSuccess`: 64 negative subtests (envelope/error/data/row/id/src/time/optional-type/link shapes, trailing data, three invalid-duplicate rows each with a valid peer so only row validation can yield the named message), each asserting the message sentinel, the named cause, one request and safe text; `…AcceptBoundaryAndMinimalRows` (time 1, year-9999 bound, large exact ID, empty `links`, empty/unknown type, padded ID kept exactly); `…LatePageFailureKeepsDiagnosticRowsOnly`. |
| F02F-03 | `…MapEveryKind` (text, photo, gif/thumb, voice, file with link precedence, link without name, sticker url-over-thumb, location without URL, no type; sender/name/content/type/attachments/RawData); `…OrderDeduplicateAndConflict` (out-of-order times, ties in first-seen order across pages, exact duplicate collapse, conflicting same-ID rows → sentinel with the earlier rows diagnostic). |
| F02F-04 | `…RepeatedPageFails` (A-B-A cycle; same rows with changed envelope text/shape); `…PageBudgetCountsTerminalPage` (499+empty passes in exactly 500 requests; 500 nonempty fails in exactly 500, no page 501, distinct IDs per page so only the budget can stop it, finite 700-request ceiling); `…CancellationBarriers` (before first request, at the terminal empty response, after a nonempty response; `errors.Is` context.Canceled + sentinel, safe text). |
| F02F-05 | `…BlockRedirects` (301/302/307/308 × first/later page: exactly the expected request count, zero destination requests, `http 30x`); `…LaterFailuresAreSafe` (transport, read, invalid JSON, HTTP 500, API error, 8 MiB+ body: class, page number, `errors.Is` cause where one exists, body closed once); `…RefreshPersistsThenRetriesSameOffset` (sequence GET0, GET2 `-216`, POST refresh, GET2 new token, GET3; offsets `[0 2 2 3]`); `…RefreshFailuresStop` (second expiry, failed persistence with callback cause, incomplete pair, refused refresh: at most one refresh, no extra retry, credentials replaced only after a persisted complete pair). |
| F02F-06 | `…BlankConversationIDMakesNoRequest`; `…LaterFailuresAreSafe` / `…RefreshFailuresStop` / `…RejectMalformedSuccess` assert no token, AppSecret, URL, host, query or provider text in errors and logs (synthetic markers); `…LogsStaySafe` (summary line only, no URL/content/conversation ID). |
| F02F-07 | Engine `TestZaloSyncStoresFullMessageHistoryAcrossPages` (actual adapter + SyncEngine + disposable MySQL): 143-row conversation stored as exactly the 143 IDs (old history 400 days before any window, large exact string ID, photo/file attachments decoded and compared, sender/content/time mapping), peer conversation stored, request offsets `0,10,…,140,143`/`0,3`, success checkpoint within `[fetch start, …]` and after the previous one, after-sync once, completion activity once, sibling channel/tenant empty, replay unchanged, changed content updated in place with unchanged counts. |
| F02F-08 | Engine `TestZaloSyncMessageFailureIsPartialAndRetryCompletes`: late malformed page, actual connection error and HTTP 500 are separate subtests after a valid first page of 25 rows; each: status `partial`, checkpoint equals the previous one, after-sync/completion 0, failed conversation stores 0 rows (diagnostic page discarded), peer stored, request logs `0,10` / `0,1`, stored error names the conversation without tokens/provider text; retry stores 25 once, after-sync 1, replay unchanged. `TestZaloSyncMessageTokenRefreshPersistsAndRetriesSameOffset`: mid-traversal `-216` refreshed once, rotated pair persisted to the channel's encrypted credentials by the owning run, offsets `0,10,10,20,23`, full history stored. |
| F02F-09 | Below. |

## Original-source and mutation evidence (F02F-09)

Harness: worker-side script (kept outside the repository; tracked tooling unchanged). Each run asserts the exact match count, writes the changed `zalo_oa.go`, runs `go -C backend test ./channels/ -run Zalo -count=1 -timeout 150s`, always restores the original bytes in `finally` and verifies byte equality, then reruns the baseline (PASS 3.8 s). KILLED means a named behavioural assertion failed; build errors, timeouts and no-ops would be INCONCLUSIVE / NOT_APPLIED.

**Original source** (`git HEAD` `zalo_oa.go` plus a temporary sentinel shim, removed afterwards): KILLED by 4+ tests, first named assertion: `TestZaloFetchMessagesMappingUnchanged` "a short page must not end paging … got 1 calls"; `TestZaloMessagesTraverseWholeHistoryToExplicitEmpty` "57 messages, want 146"; blank ID, redirect and boundary-row probes also fail. Engine acceptance on the original adapter (disposable MySQL): the four new engine cases and the retained traffic count fail (offsets stop at 140 without the empty page; a late malformed page, a connection error and an HTTP 500 end `success` instead of `partial`; the mid-traversal refresh test stops at `0,10,10,20`).

| Mutation (matches) | Result | First failing assertion |
|---|---|---|
| M1 short page ends success (1) | KILLED | `…MappingUnchanged` expects page then explicit empty (also redirect/cancel/late-failure probes) |
| M2 malformed/missing data → empty success (1) | KILLED | `RejectMalformedSuccess/data_missing` |
| M3 offset advances by 10 not physical rows (1) | KILLED | `TraverseWholeHistory…` unscripted offset (request fails at page 7) |
| M4 skip invalid duplicate before validation (2) | KILLED | `RejectMalformedSuccess/invalid_duplicate_src` |
| M5a budget guard removed (1) | KILLED | `PageBudget…` 500 nonempty reaches page 701 (finite ceiling) |
| M5b terminal context guard removed (1) | KILLED | `CancellationBarriers/at_the_terminal_empty_response` |
| M5c pre-request context guard removed (1) | KILLED | `CancellationBarriers/before_the_first_request` |
| M6 redirects followed (1) | KILLED | `BlockRedirects/301_later=false` |
| M7 link URL no longer overrides (1) | KILLED | `MapEveryKind` f1 attachment URL |
| M8 conflicting duplicate ignored (1) | KILLED | `OrderDeduplicateAndConflict` conflicting duplicate returned nil |
| M9 no chronological sort (1) | KILLED | `TraverseWholeHistory…` order |
| M10 count 20 (1) | KILLED | `TraverseWholeHistory…` request inventory |
| M11 blank-ID guard removed (1) | KILLED | `BlankConversationIDMakesNoRequest` made a request |
| M12 second refresh retry allowed (2) | KILLED | `RefreshFailuresStop/second_expiry` |
| M13 repeated-page guard disabled (1) | initial worker kill NOT certified (see R1) | original fixture panicked (index out of range) when the guard was disabled; after R1: KILLED by `RepeatedPageFails/cycle_A-B-A` named assertion |
| M14 blank message ID accepted (1) | KILLED | `RejectMalformedSuccess/message_id_""` |
| M15 cause flattened with `%v` (1) | KILLED | `LaterFailuresAreSafe/transport_error` |
| M16 tie order by ID desc (1) | KILLED | `OrderDeduplicateAndConflict` ids `a,d,c,b` |

Retained history: the first campaign run reported M7 and M9 as INCONCLUSIVE (the mutations failed to compile: unused variable/import), retained here, then corrected to the compilable forms above; the table is from a full clean rerun of all 18 plus the original source. M4's competing guards (conflict, repeated page, non-progress) are excluded by a valid peer row, and the assertion names the validation cause.

## Retained-assertion changes (not weakened)

- `TestZaloFetchMessagesMappingUnchanged` asserted that a short page ends paging (1 call). That is the behaviour F02F-01 replaces; it now serves the two rows then the explicit empty page and asserts 2 calls. Mapping assertions are unchanged.
- `TestZaloSyncStoresEveryConversationBeyondTheOldLimit`: message requests 105 → 210 (each conversation now needs its explicit empty page). All other assertions unchanged.
- The R024 `zlFake` conversation endpoint ignored offsets and would repeat one page forever under the new contract; it now serves rows by offset (default: one row at offset 0, empty after).

## Regression and gate results

| Check | Result |
|---|---|
| `go -C backend test ./channels -count=1 -v` | exit 0; 94 top-level / 192 PASS lines, 0 FAIL, 0 SKIP (35 Zalo top-level) |
| `go -C backend build ./...`, `go vet ./...` | exit 0 / exit 0; `gofmt -l` clean on touched Go files (`adapter.go`, `integration_test.go`, `sync_demo_fixture_test.go` pre-existing, untouched) |
| Engine `-Run Zalo` on disposable MySQL via `scripts/test-backend.ps1` | `ok`, new and R024 Zalo tests PASS, no `ccma-test-*` residue |
| Full backend `./...` uncached, disposable MySQL, `-p 1 -timeout 40m -json`, `max_connections=1000` (worker-side recipe identical to `scripts/test-backend.ps1` plus those flags; the override is isolated to the throwaway container) | exit 0, 14 min; 1074 PASS, 0 FAIL, 2 SKIP (`ai/pricing.TestFetchThatTuNguonNgoai`, `storage.TestNoiCatS3`, external-service); 0 DB-unavailable skips; all packages ok |
| R019 `scripts/ci_db_test_gate.py` on that JSON | GATE PASSED, five sentinels PASS, 601745 events, 0 invalid records |
| Focused R012–R016 / R024 / replay / lease regressions | inside the full run (PASS); not separately repeated |
| Race detector | NOT RUN: `CGO_ENABLED=0`, no C compiler on this host |
| Docker inventory | no `ccma-test-*` container or network remains; persistent `ccma-*` stack untouched |

Docs build, catalog, doctor (25/25), preflights and gate unit tests are recorded in the active handoff verification section.

## Fixture isolation and limits

Synthetic RoundTrippers only; engine tests install one as `http.DefaultTransport` and fail on any unexpected host. `sync_files` stays off, so no media is fetched. MySQL ran in throwaway containers on a private network with no host port, removed afterwards. Synthetic markers stand in for credentials; none are printed or committed. GORM/SQL logging sinks are outside certification.

Not claimed: live Zalo offset snapshot stability, terminal behaviour, retention, permissions or string-link compatibility (the official reference documents string links; the inherited object-link mapping is kept and unverified live); global F02; provider governance; hosted CI; FREEZE. Strictness choices that could reject live data: `null` for an optional consumed string field and a missing `error` key are treated as malformed. Independent Codex REVIEW is required (seed author/timing, integrated source, all nine requirements, mutation sensitivity, regressions, claim limits).

## R1 repair (R032-R1-01..02), 2026-10-03

Independent review (CHANGES_REQUIRED for exact BUILD `b31b974`) found that the committed cycle fixture indexed a three-element slice with the request number, so disabling `if seenPages[fp]` panicked instead of failing a semantic assertion; the worker M13 kill above is therefore not certified. Repair (test only): `TestZaloMessagesRepeatedPageFails` serves valid recurring pages up to a synthetic ceiling at request 7 and asserts the repeated-page failure plus the exact request count (A-B-A: 3, envelope-change: 2). Applied M13 (1 match, applied sha `14230cef4ddfb85613d40d8f05389cce928b23534be7451f3b1c4db6f38b44b6`): KILLED, first assertion `want the repeated-page failure, got zalo message coverage incomplete: page 7: zalo api request failed` (7 requests reached the ceiling); no panic, timeout or build error; `zalo_oa.go` restored to `213a3f7b3a962b8fed3fb643f4a2212b2bfb363f27323089ec493811d880dc8e` (equal to the reviewed BUILD) and baseline PASS. The full original-source + 18-mutation campaign was rerun against the repaired tests: all KILLED, all restored; the first-run M7/M9 build INCONCLUSIVE results remain recorded above. Current implementation prose (SPEC, order, roadmap, memory, status, handoff) was synchronized; planning-era sentences are marked historical locally. No product source change. Race NOT RUN; full-backend (1074 PASS) and R019 results above are from the BUILD source, which the repair does not change; the repaired test was rerun with the whole channels package (below).
