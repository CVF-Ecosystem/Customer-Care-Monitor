# CCMAI-RUNTIME-023 / F02-B — independent Pancake sync review

Date: 2026-10-01. Reviewer: Codex, independent of Claude IMPLEMENTATION_WORKER. Exact BUILD `1fa8e14774149e68df10176f52804d80c11bb5bf`, parent `0bfbbc0cec2d99218d76f86f789b9aab40a3f3bd`. Disposition: **REVIEW_PASS / FREEZE_OPEN** for Pancake conversation coverage, including the same-scope reviewer repair below. F02 remains OPEN for Zalo/F02-C and live-channel claims.

## Intake and scope

State, handoff header, front marker, tranche record and work-order status correctly said `REVIEW_PENDING`; current memory prose and implementation-status limitations still said `DISPATCH_READY`. Codex reported `BLOCKED_CONTINUITY_DRIFT`, aligned both stale pointers at INTAKE, re-read them and reran preflight before source review. Doctor passed 25/25. The later observation is added to the existing [parent learning intake](learnings/CCMAI_TO_CVF_DOWNSTREAM_GATE_LEARNING_INTAKE_2026-10-01.md); parent implementation/testing remains with its assigned agent.

Exact BUILD changed only Pancake source/tests, engine sync source/tests, BUILD evidence and authorized order/roadmap/continuity/status files. `git diff --exit-code 0bfbbc0 1fa8e14 -- CVF_SESSION/authority/CCMAI-RUNTIME-023.json backend/channels/facebook.go backend/channels/zalo_oa.go backend/channels/adapter.go .github docker-compose.yml` returned 0. Dispatcher seed `5b785d5` predates BUILD and is unedited. The pre-existing untracked `knowledge/_index.json` remains outside commits.

## Source and reviewer-attention decisions

- The two old test expectation changes are accepted: limit 10 over 61 eligible rows now demands `ErrPancakeCoverageIncomplete` and 10 diagnostic rows; the R022 engine probe now requires Pancake limit 0, alongside Facebook 0 and Zalo 100. The original positive mapping, metadata-redaction, actual sync-run observations and other channel expectations remain asserted. These changes implement the new contract rather than hiding a failed invariant.
- Conversation paging follows the last physical row, including COMMENT rows, until a valid empty array. Nonempty short pages continue; the 200-page budget and cursor cycles fail. Missing/null/non-array pages and malformed ID/time fail before filtering, including COMMENT or old rows. Inclusive `since`, deterministic first-ID deduplication, INBOX filtering and metadata reduction match the SPEC. An extra empty-page request and more error statuses are intentional behavior changes.
- Only Pancake receives the fetch-start success checkpoint, rounded down to seconds. `recordSyncStatusAt` changes `last_sync_at` only on success with an explicit checkpoint; `updated_at` still uses completion. The same fenced tenant/channel/status/run-ID update clears ownership/lease, and fetch errors process no partial rows. Facebook/Zalo keep their checkpoint behavior. The delayed-run test observes stored rows/checkpoint and a later update caught in a second run, rather than merely comparing mock call counts.
- `FetchMessages` selection, conversion and pagination logic is unchanged. Shared request failures now expose bounded classes/codes and the response reader has a size budget, as recorded below. Zalo remains outside R023.

## Independent OpenAPI reconciliation

Codex retrieved the public [Pancake OpenAPI](https://developer.pancake.biz/openapi/openapi.yaml) on 2026-10-01, HTTP 200, 175390 bytes; SHA256 of the UTF-8 document string was `1692045af3c4c58b71471f6b52403c1ad748e7fd77cd26834d17bf8160650c71`. The v2 conversation operation confirms pagination by `last_conversation_id`, up to 60 rows, integer `since`/`until` in seconds and the `order_by` enum including `updated_at`. Its response schema identifies `conversations` as an array. It does not establish a short-page terminal guarantee or timezone semantics for a zone-less `updated_at`; the Conversation schema calls it a date-time field without that interpretation. Treating zone-less values as UTC remains an inherited local assumption, not a newly proved live fact. No authenticated channel endpoint or real token was used.

## Reviewer-local repair

`reviewerLocalRepairBoundary`: objective, authorized source paths, R2 ceiling, prohibited effects and local-only scope remain unchanged. The owner explicitly authorized minor reviewer repairs earlier in this session. Claude remains the BUILD commit steward; Codex owns this disclosed review/repair commit. No new tranche or worker return was required.

`reviewerLocalRepairBasis`: four added focused probes failed on BUILD. A transport inner cause could echo the complete request URL/token; a provider message with an encoded token/URL/body marker survived literal-token replacement; a direct `since=0` fetch omitted the required fixed `until`; and cancellation while receiving an empty terminal page returned success. Codex repaired these in `pancake.go`: generic request/read errors (safe context cancellation retained), numeric API codes without provider text, fixed `until` even with zero `since`, and a post-request cancellation check before accepting page content. The legacy HealthCheck test now asserts code 102 instead of raw provider text; the nested-URL test no longer requires preserving arbitrary transport cause text.

The order also requires finite request memory bounds. The unbounded response reader was capped at 8 MiB plus one byte for overflow detection; exceeding the cap returns an error before decoding. A new negative test proves even a syntactically terminal response over the budget cannot yield success. This shared helper affects message-request resource/error handling too; message selection/pagination is unchanged. Operational error detail is reduced, and valid responses larger than 8 MiB now fail safely. This resource threshold and live payload compatibility are not established by synthetic tests.

## Verification and limits

- Independent `go test ./channels -count=1` passed after repair, including all old adapter tests and the new negative probes. `go build ./...` passed. The disposable-MySQL engine run used `scripts/test-backend.ps1 -Packages ./engine -Run 'Pancake|Facebook|ConversationFetchLimit|SyncStatus'`: exit 0, package PASS in 115.285s, DB/container network removed. Source inspection confirms the checkpoint/ownership behavior and accepted test changes.
- Claude's BUILD record reports 143 eligible synthetic adapter rows, 105 conversations/messages stored end to end, later-page error/checkpoint/no-after-sync, retry/replay and delayed overlap; 12/12 adapter and 2/2 engine mutations detected; full backend 524 pass / 0 fail / 2 optional skips with R019 five-sentinel PASS. Codex did not independently repeat that full suite, its mutations, or replay its unretained JSON log. The worker disclosed stripping the wrapper's two non-JSON status lines before R019 replay.
- Final preflight covering all BUILD paths and reviewer changes passed 7/7; catalog `-Check`, doctor 25/25, docs build and diff check passed. Synthetic pages and disposable MySQL establish local application semantics; real Pancake completeness, concurrent provider ordering, timezone behavior and rate limits remain unproved. PR #1 green Actions belongs to older remote SHA `3e0b37e`, not this local BUILD/review. No provider/key use, persistent DB, push, merge, deployment, CVF parent work or FREEZE occurred.

## Disposition

R023 passes independent REVIEW after the disclosed small repair. Facebook R022 remains reviewed; Zalo F02-C is the next bounded planning step. F02 closure, live-channel claims, hosted CI for this later local revision and FREEZE remain open.
