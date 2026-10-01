# CCMAI-RUNTIME-023 / F02-B BUILD evidence — Pancake conversation-window coverage

**Date:** 2026-10-01 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 · **Role:** IMPLEMENTATION_WORKER + COMMIT_STEWARD (Claude); independent REVIEWER: Codex
**Authority:** [SPEC](../specs/RUNTIME_PANCAKE_SYNC_COVERAGE_F02B_2026-10-01.md), [work order](../work_orders/CCMAI_RUNTIME_023.md), dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-023.json` (commit `5b785d5`, unedited; identical allowed paths/prohibited effects to the tranche record). Base `0bfbbc0`.
Local httptest / synthetic `http.DefaultTransport` and disposable MySQL only; no real Pancake endpoint or token, provider/Alibaba key, persistent DB, secret, push or FREEZE. The pre-existing untracked `knowledge/_index.json` is not part of the commit.

## Source trace before the change

- Engine (`engine/sync.go`): `since = last_sync_at − 1h` (or now − 7d) → `FetchRecentConversations(ctx, since, conversationFetchLimit(type))`, which returned **100 for pancake** → on a fetch error `finish("error")` before any conversation is processed → per-conversation writes → `recordSyncStatus`, whose `buildSyncStatusUpdates` sets `last_sync_at = now` (the **completion** time) only for `success`.
- Adapter (`channels/pancake.go`) returned **success** when: the eligible count reached `limit` (100 from the engine); a page was shorter than 60; the next cursor equalled the previous one; the 200-page loop ran out; or `conversations` was missing/null/not decodable as rows (decoded as an empty slice → terminal). `until` was recomputed (`time.Now()`) on **every** page. Rows with an empty/invalid `updated_at` were treated as eligible (zero time). `scrubURLError` removed only the outermost `*url.Error`, and request-creation errors and provider `message` text were not scrubbed.
- Consequence: a run could record `success` and move `last_sync_at` to its finish time after incomplete coverage, and updates arriving during a long run could fall behind the next run's one-hour overlap.

**OpenAPI reconciliation.** I did not re-fetch the OpenAPI in this BUILD (no external call); I relied on the SPEC's record of the official document (v2 conversations, ≤60 per call, `last_conversation_id` = last row of the previous call, `since`/`until` timestamp filters, no promise that a short page is the last). Assumptions carried over unchanged from the existing adapter and fixtures, not proven by synthetic tests: `since`/`until` are Unix seconds; `type=INBOX` and `order_by=updated_at` are accepted; `updated_at` without a zone is UTC; an HTTP 200 with `success:false` is an API error.

## Implementation

`backend/channels/pancake.go`
- `FetchRecentConversations` captures `since`/`until` (Unix seconds) **once** before page 1 and sends the same pair on every page (only when `since` is non-zero, as before). It follows `last_conversation_id` = the **last physical row** of the previous page (including filtered COMMENT rows); a non-empty page of any length continues; **only an explicit empty `conversations` array** ends with success.
- Every row is decoded and validated (non-empty string `id`, parseable `updated_at`) **before** the type/`since` filters, so a malformed COMMENT or old row also fails. Eligible = INBOX (or untyped) with `updated_at >= since`; no ordering assumption; duplicate IDs keep the first occurrence. Metadata reduction and message mapping are unchanged.
- `ErrPancakeCoverageIncomplete` (rows returned only for diagnostics) for: missing/null/non-array `conversations`, malformed array or row, missing/invalid `id`/`updated_at`, a repeated or non-progressing cursor, cancelled context, a positive `limit` the window exceeds (returns exactly `limit` rows + error), and the 200-page budget. Request/API/429-exhausted/HTTP/read/decode failures keep returning their own error.
- `doRequest`: `scrubURLError` now unwraps **every** `*url.Error` layer (keeping `Op` and the root cause) and is also applied to request-creation errors; the provider `message` in `success:false` errors has the page token replaced by `[redacted]` and is capped at 200 runes. Pacing (`waitTurn`), 429 retry/backoff and `FetchMessages` are unchanged.

`backend/engine/sync.go`
- `conversationFetchLimit`: 0 (exhaustive) for `facebook` **and `pancake`**, 100 otherwise (Zalo unchanged).
- `boundsCheckpointByFetchStart(type)` is true only for `pancake`. For such a run the engine captures `syncNow().Truncate(time.Second)` immediately before the conversation fetch; on `success`, `recordSyncStatusAt` writes that value as `last_sync_at` while `updated_at` still records the completion time. Error/partial/ownership-lost paths are unchanged and never touch `last_sync_at`. `recordSyncStatus` keeps its signature (delegates with no checkpoint), so Facebook/Zalo and all fenced writes behave as before.
- `syncNow` (`= time.Now`) is the narrow clock seam allowed by the order, used for the status-write time and the fetch-start checkpoint.

## Tests

`channels/pancake_coverage_test.go` (local httptest server recording every query):
- 143 eligible rows over three non-empty pages (60 with a COMMENT last row, **45 short**, 41) and an empty terminal page, with an older row ahead of newer rows, a row exactly at `since` and a duplicate ID: each eligible ID once, old/COMMENT rows absent, cursor chain `"" → comment-1 → boundary → a0003`, identical `since` and fixed `until` (≥ fetch start) and `type=INBOX` on every request.
- Short pages continue (3 requests for 1+1+empty); limit 100 over 143 → error + 100 rows; limit equal to the window succeeds.
- 15 malformed cases (missing/null/object/string `conversations`, non-object/null row, empty/numeric id, missing/invalid `updated_at`, bad time on a COMMENT row and on an old row, good-then-broken row, `success:false` with the token in its message, undecodable body) → error, no token/URL/body text.
- Later-page failures (429 exhausted after 4 attempts, HTTP 500, API error echoing the token, undecodable, connection dropped, truncated body) → safe error + the 5 rows seen; positive control on the same server succeeds in 2 requests. Nested `*url.Error` from a transport is scrubbed but keeps `connection refused`. Non-progressing and cyclic cursors → error. 200-page budget → error after exactly 200 requests. Cancellation → error.
- `pancake_test.go`: the existing limit assertion was changed from "limit 10 returns 10 rows" to "limit 10 over 61 rows returns `ErrPancakeCoverageIncomplete` with 10 diagnostic rows" (the old expectation is the truncated success the SPEC forbids). The rest of that file is untouched and passes.

`engine/sync_pancake_coverage_test.go` — the **real** `PancakeAdapter` (fake pages.fm transport honouring `since`/`until`, the cursor and 60-row pages, installed as `http.DefaultTransport` and restored) and the real engine on disposable MySQL:
- 105 conversations (60 + 45 + empty): 105 conversations and 105 messages stored, `success`, checkpoint ≥ fetch start and ≤ the fixed `until`, identical `until` on all 3 page requests, 105 message fetches, after-sync once.
- Page 2 fails (HTTP 500 / missing `conversations`) with 70 conversations: run error, status `error`, `last_sync_at` unchanged, 0 rows stored, 0 after-sync, stored error free of token/`page_access_token`/host; retry stores 70/70, advances the checkpoint, after-sync once; replay stays 70/70.
- Delayed run: while the first run processes messages the clock seam jumps two hours and the provider receives an update after the run's fixed `until`. The first run's checkpoint lies in `[fetch start, until]` while `updated_at` is ≥ 119 minutes later, and the late conversation is not stored; the next run stores it. (With a completion-time checkpoint the next window would start an hour after that update and miss it — see the `checkpoint-unbounded` mutation.)
- Selection table: limit 0/0/100/100 and bounded checkpoint false/true/false/false for facebook/pancake/zalo_oa/unknown. The R022 limit test (renamed `TestConversationFetchLimitIsExhaustiveForFacebookAndPancake`) now expects pancake 0; Facebook 0 and Zalo 100 are asserted through a real sync run.

## Non-vacuity

- **Pre-BUILD source** (`git show HEAD:` for `pancake.go` and `sync.go`, with a temporary shim for `ErrPancakeCoverageIncomplete`, `syncNow` and `boundsCheckpointByFetchStart`; removed and byte-compared afterwards): **8 of 9** new adapter tests failed (the cancellation test passes on old code too — the old `doRequest` already returned the context error), the updated `TestPancakeFetchRecentConversationsPaginates` failed, and **all 4 new Pancake engine tests** plus the renamed limit test failed (the `http500` subtest passes on old code — an HTTP error was already an `error`; the `malformed` subtest fails).
- **Engine mutations** of the new code (disposable MySQL): `checkpoint-unbounded` (pancake not bounded) → `CheckpointIsBoundedByFetchStart`, `StoresEveryConversation…`, `LimitAndCheckpointSelection` failed; `pancake-limit-100` → the limit test, `StoresEveryConversation…` and the selection test failed.
- **Adapter mutations** (12): short page terminal, repeated cursor success, page budget success, missing array terminal, `until` recomputed per page, cursor = last eligible row, limit truncation success, bad row skipped, exclusive boundary, no dedupe, one-layer URL scrub, raw provider message — **12/12 detected** (the `until` mutation was re-run once after my first variant did not compile). All temporary edits were restored and byte-compared (`restored-identical`).

## Gates (repo root)

| Check | Result |
|---|---|
| Adapter `go test ./channels -count=1` | PASS |
| Focused engine: `scripts/test-backend.ps1 -Packages ./engine -Run 'Pancake\|ConversationFetchLimit\|Facebook' -VerboseTests` | 8 test functions (+4 subtests) PASS (Pancake and R022 Facebook) |
| Full: `go test ./... -json -count=1 -p 1` on disposable MySQL (`TEST_DB_DSN`, `golang:1.26-alpine`, GOPROXY=off) — includes the R013–R016 and R022 sync suites | exit 0; 524 pass / 0 fail / 2 optional skips (509 at the R022 BUILD) |
| R019 gate on the full JSON log | `GATE PASSED`, 0 invalid records (271370 events), five sentinels PASS, 0 DB-unavailable skips; optional skips `pricing.TestFetchThatTuNguonNgoai`, `storage.TestNoiCatS3` |
| `go build ./...`, `go vet ./channels ./engine` | OK |
| `git diff --check` | clean (CRLF warnings only) |
| Cleanup | disposable MySQL containers/networks removed after every run; persistent Compose DB never used |

As in R021/R022, the wrapper's two non-JSON lines ("Waiting for…", "Removed…") were stripped before replaying the Go JSON through the R019 gate; the committed script is unchanged. Downstream preflight, catalog `-Check` and doctor run on the committed range after the commit and are reported in the hand-over.

## Behavior changes and limits for the reviewer

1. A Pancake run now ends `error` (keeping the previous checkpoint) whenever the cursor chain does not reach an empty page within 200 pages (~12k conversations at 60 per page), or any row lacks a valid `id`/`updated_at` — including COMMENT and old rows that used to be ignored. No resumable cursor was added.
2. Every successful Pancake window costs one extra request (the empty terminal page), and short pages no longer stop early, under the existing 250 ms pacing.
3. The Pancake success checkpoint is now the fetch start (≤ the fixed `until`), not the completion time, so the next window re-reads anything updated during the run; upserts keep this idempotent. `updated_at` remains the completion time.
4. **Not covered:** Zalo still receives limit 100 (F02 stays OPEN for F02-C). `FetchMessages` selection/pagination is unchanged (only its shared request helper got the token scrubbing). Synthetic pages and disposable MySQL prove application semantics only — not live Pancake completeness, ordering under concurrent updates, rate limits or CVF AI governance.

## Changed set

`backend/channels/pancake.go`, `backend/channels/pancake_test.go` (one assertion), new `backend/channels/pancake_coverage_test.go`, `backend/engine/sync.go`, `backend/engine/sync_facebook_coverage_test.go` (limit expectation and name), new `backend/engine/sync_pancake_coverage_test.go`, this evidence, work-order status line, roadmap F02 row, handoff, active state, session memory, tranche record, implementation status.

## Disposition

`REVIEW_PENDING` for Codex independent REVIEW. Claude does not self-approve; no push, merge, deployment or FREEZE.
