# CCMAI-RUNTIME-022 / F02-A BUILD evidence — Facebook sync window coverage

**Date:** 2026-10-01 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 · **Role:** IMPLEMENTATION_WORKER + COMMIT_STEWARD (Claude); independent REVIEWER: Codex
**Authority:** [SPEC](../specs/RUNTIME_FACEBOOK_SYNC_COVERAGE_F02A_2026-10-01.md), [work order](../work_orders/CCMAI_RUNTIME_022.md), dispatcher seed `CVF_SESSION/authority/CCMAI-RUNTIME-022.json` (commit `3f101ca`, unedited). Base `8cd4a7e`.
Synthetic Graph transport and disposable MySQL only; no real Facebook/channel call, provider/Alibaba key, persistent DB, secret, push or FREEZE. The pre-existing untracked `knowledge/_index.json` is not part of the commit.

## Source trace before the change

- `SyncReservedChannel` (`engine/sync.go`): reservation → decrypt/adapter → `since` = `last_sync_at − 1h` (or now − 7d) → `FetchRecentConversations(ctx, since, 100)` → per-conversation upsert/messages → `recordSyncStatus`. On a fetch error it calls `finish("error", …)` and returns **before any conversation is processed**; `buildSyncStatusUpdates` sets `last_sync_at` **only for status `success`**, so an error/partial run keeps the previous checkpoint and `triggerAfterSync` is reached only after a recorded success. A partial slice returned together with an error is therefore never processed — no engine change was needed for that.
- Facebook adapter (`channels/facebook.go`): broke at `limit` and when `data` was empty or `paging.next` absent; returned early on the **first row older than `since`** (assumed newest-first); skipped malformed rows silently; followed any `paging.next` URL; and wrapped the `net/http` error, whose text embeds the full request URL **including `access_token`** (the engine stores that text in `last_sync_error` and logs it).
- Result: with >100 conversations in the window the old run could report `success`, advance `last_sync_at` and leave unvisited conversations behind the next run's 1-hour overlap.

## Implementation

`backend/channels/facebook.go`
- `FetchRecentConversations` follows every page to a terminal page (a page with no `paging.next`); an **empty page that still has `next` continues**. Every row is examined: eligible means `updated_time >= since` (inclusive), older rows are skipped (never a stop signal), IDs are de-duplicated across pages.
- Incomplete coverage returns `ErrFacebookCoverageIncomplete` (rows seen so far are returned for diagnostics only) for: missing/non-array `data`, non-object row, missing/empty `id`, missing/unparseable `updated_time` (also for rows older than `since`), non-string `next`, **unsafe next URL** (must be `https`, host exactly `graph.facebook.com`, port empty/443, no userinfo/fragment, path exactly `/v21.0/<page_id>/conversations`), a **repeated cursor** (query compared without the token, so a token-only change cannot hide a cycle), a cancelled context, the **page budget** (`fbMaxConversationPages = 500`) and a positive `limit` that the window exceeds (returns exactly `limit` rows plus the error, never a quiet truncation). `limit <= 0` means exhaustive. Page/read/decode/API failures keep returning their own error.
- `withoutRequestURL` strips every `*url.Error` layer from request-creation and `client.Do` errors, so no URL/token reaches the error text (the underlying cause, e.g. `connection reset by peer`, is kept). `doRequest` and `FetchMessages` share it. Messages enumeration is otherwise unchanged.

`backend/engine/sync.go`: `conversationFetchLimit(channelType)` returns 0 (exhaustive) for `facebook` and the previous 100 for everything else; the call site uses it. Nothing else in the run (ownership, lease, R013–R016 fencing, per-conversation partial status, one-hour overlap, after-sync admission) changed. No other adapter, interface, schema, API or UI file was touched.

## Tests

`channels/facebook_coverage_test.go` — a recording `RoundTripper` serves scripted Graph pages (no network):
- 270 eligible rows over 4 pages with an old row **before** newer rows, a row exactly at `since`, a duplicate across a page boundary and a terminal page: every eligible ID exactly once, the old row absent, 4 requests (positive control), only `graph.facebook.com` contacted.
- Limit 100 over that window → error + exactly 100 rows; limit 100 over an exactly-100-row window succeeds.
- Empty page with `next` then rows then a terminal empty page; empty terminal first page.
- Malformed pages (11 variants incl. `null` data, malformed old row) each → incomplete coverage.
- Later-page failures (Graph error object, non-JSON body containing the token, transport error wrapped in `*url.Error`): error, the 5 rows seen kept for diagnostics, and no `TOKEN`, `access_token`, host or body text in the error; the real cause survives.
- Unsafe/repeated pagination (13 URL variants + two cycle cases): error after exactly one request, the second page is never requested, only the Graph host is contacted.
- 500-page budget with an endless fresh-cursor chain: error, exactly 500 requests/rows. Cancellation mid-run and before the run: error.

`engine/sync_facebook_coverage_test.go` — the **real** `FacebookAdapter` (fake Graph installed as `http.DefaultTransport`, restored on cleanup) feeding the real engine on disposable MySQL:
- 130 eligible conversations over 3 pages with the checkpoint 3 days back: 130 conversations + 130 messages stored, status `success`, checkpoint advanced, 3 page + 130 message requests, after-sync fired once.
- Page 2 fails (Graph error object, and transport error): run returns an error, status `error`, `last_sync_at` **unchanged**, **0 conversations/messages stored**, after-sync fired 0 times, stored error text free of token/URL, exactly 2 page requests and 0 message fetches. Retry from the same checkpoint: 130/130 stored, `success`, checkpoint advanced, after-sync once; a further replay leaves 130/130 (idempotent).
- A page without a `data` array (malformed) → error, checkpoint unchanged, no after-sync.
- Limit selection: a probing adapter receives 0 for `facebook` and 100 for `pancake` and `zalo_oa`.

## Non-vacuity

- **Pre-BUILD source** (`git show HEAD:` versions of `facebook.go`/`sync.go`, plus a temporary shim for the two new symbols, restored byte-for-byte): all 8 new adapter tests and all 4 engine tests (`StoresEveryConversation…`, both `PageFailure…` modes, `MalformedPage…`, `ConversationFetchLimit…`) **failed**.
- **Mutations of the new adapter code** (temporary, restored — `restored True`; `go test ./channels -run Facebook`): host check removed, path check removed, scheme check removed, cycle check removed, page budget removed, boundary made exclusive, stop on first old row, limit guard removed, token not sanitized, missing `data` ends quietly, bad row skipped — **11/11 detected**. No mutation was run on `conversationFetchLimit` itself; it is covered by the limit-selection test and the >100 end-to-end test (the old limit 100 failed the latter).

## Gates (repo root)

| Check | Result |
|---|---|
| Adapter tests `go test ./channels -count=1` | PASS |
| Focused engine: `scripts/test-backend.ps1 -Packages ./engine -Run 'Facebook\|ConversationFetchLimit' -VerboseTests` | 4 test functions (+2 subtests) PASS |
| Full: `go test ./... -json -count=1 -p 1` on disposable MySQL (`TEST_DB_DSN`, `golang:1.26-alpine`, GOPROXY=off) — includes the R013–R016 sync suites | exit 0; 509 pass / 0 fail / 2 optional skips (was 495 at R021) |
| R019 gate on the full JSON log | `GATE PASSED`, 0 invalid records, five sentinels PASS, 0 DB-unavailable skips; optional skips `pricing.TestFetchThatTuNguonNgoai`, `storage.TestNoiCatS3` |
| `go build ./...`, `go vet ./channels ./engine` | OK |
| `git diff --check` | clean (CRLF warnings only) |
| Cleanup | disposable MySQL containers/networks removed after each run (`docker ps -a` / `docker network ls`: none left); persistent Compose DB never used |

The wrapper's two non-JSON lines ("Waiting for…", "Removed…") were stripped before replaying the Go JSON through the R019 gate (as in R021); the committed script is unchanged. Downstream preflight, catalog `-Check` and doctor are run on the committed range after the commit and reported in the hand-over message.

## Behavior changes and limits for the reviewer

1. A Facebook run that cannot prove a complete window now ends `error` and keeps the old checkpoint — more error statuses than before when Graph pages are flaky or exceed 500 pages (~50k conversations at 100/page). No resumable cursor was added (SPEC: separate tranche).
2. Rows with an invalid `updated_time`/`id` previously were silently treated as time zero/empty; they now fail the run.
3. The Facebook page must be reachable under the exact path `/v21.0/<page_id>/conversations`; a next link under another Graph API version would be rejected as unsafe (fails closed).
4. **Not covered:** Pancake and Zalo still receive limit 100 (F02 stays OPEN for them). `FetchMessages` is unchanged (it still returns on the first message older than `since` and returns partial messages with an error, which the engine records as a per-conversation failure). Synthetic pages and disposable MySQL prove application sync semantics only — not live Facebook completeness, rate-limit behavior or CVF AI governance.

## Changed set

`backend/channels/facebook.go`, `backend/engine/sync.go`, new `backend/channels/facebook_coverage_test.go`, new `backend/engine/sync_facebook_coverage_test.go`, this evidence, work-order status line, roadmap F02 row, handoff, active state, session memory, tranche record, implementation status.

## Disposition

`REVIEW_PENDING` for Codex independent REVIEW. Claude does not self-approve; no push, merge, deployment or FREEZE.
