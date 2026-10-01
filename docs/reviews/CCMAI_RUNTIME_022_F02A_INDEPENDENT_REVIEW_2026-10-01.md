# CCMAI-RUNTIME-022 / F02-A — independent Facebook sync review

Date: 2026-10-01. Reviewer: Codex, independent of Claude IMPLEMENTATION_WORKER. Exact BUILD commit `583c51c8cc87d350b2b98eeef5f71e526def9ce7`, parent `8cd4a7efe74ba183a78004aed35efea674a8ce95`. Disposition: **REVIEW_PASS / FREEZE_OPEN** for the Facebook slice, including the small reviewer repair recorded below. F02 as a whole remains OPEN for Pancake/Zalo.

## Continuity and scope

At INTAKE, state, handoff header, front marker, tranche record and order said `REVIEW_PENDING`, but the current memory prose still described R022 as `DISPATCH_READY`. I reported `BLOCKED_CONTINUITY_DRIFT`, corrected that pointer and reran the downstream gate before source review. During final review sync, I also corrected stale `DISPATCH_READY` prose in implementation status. The BUILD diff contains only `backend/channels/facebook.go`, `backend/engine/sync.go`, focused tests, evidence, and authorized continuity/order/roadmap/status files. The dispatcher seed at `3f101ca` is unchanged. The pre-existing untracked `knowledge/_index.json` remains outside commits.

## Independent source review

- Engine `conversationFetchLimit` sends `0` (exhaustive) to Facebook and retains `100` for Pancake/Zalo. On a fetch error it processes no returned partial rows, records non-success and keeps the previous `last_sync_at`; after-sync runs only after success. Ownership, lease and the one-hour overlap code were not changed by BUILD.
- The Facebook adapter now visits every page until a terminal page, including across an empty page with `next`; checks every row without assuming timestamp order; includes `updated_time == since`; and de-duplicates IDs. A malformed `id` or `updated_time`, even on an older row, fails the run. This stricter validation is consistent with the accepted fail-closed SPEC and changes prior silent-skip behavior.
- `next` must use HTTPS, `graph.facebook.com`, the expected port and exact `/v21.0/<page_id>/conversations` path. A different Graph API version/path fails closed. This is an intentional fixed-version contract; synthetic tests prove rejection, while live Facebook URL shape is not established by this review. Cycle, positive-limit and 500-page budget errors prevent a truncated success.
- `FetchMessages` still stops at the first message older than `since`; its paging/selection logic was not changed. Shared request-error handling changed so failures carry less diagnostic text. Message-window completeness is outside R022's Facebook conversation-pagination acceptance and should not be inferred from this review.

## Same-scope reviewer repair

Focused negative probes failed on the exact BUILD: `paging` present as `null`/non-object and `next` present as empty/null were treated as terminal success; a Graph error message or transport inner cause that echoed the token-bearing URL leaked it into the returned error. The first failure could advance a checkpoint without proof of terminal pagination; the latter could persist a credential in `last_sync_error`.

I changed only `backend/channels/facebook.go` and its focused test: absent `paging` or an object without `next` remains a terminal page; malformed explicit `paging`/`next` now returns `ErrFacebookCoverageIncomplete`. Graph error text and arbitrary request/read causes are omitted from returned errors, retaining the numeric Graph code and context cancellation where safe. This loses some operational error detail to enforce the SPEC's no-token/no-raw-URL boundary. New tests first failed on BUILD and passed after repair; `FetchMessages` selection logic is unchanged. CVF's same-scope minor-reviewer-repair rule was used without opening another tranche.

## Evidence and limits

- Claude's BUILD record reports 270 unique eligible conversations across four synthetic pages, interleaved old/new rows, inclusive boundary, empty-page continuation, URL/cycle/budget/cancellation negatives, 11/11 mutations detected, and disposable-MySQL end-to-end 130 conversations/messages, later-page error with unchanged checkpoint/no after-sync, retry/replay idempotence. Its full backend run reports 509 pass / 0 fail / 2 optional skips and R019 five DB sentinels PASS. I checked the exact changed source and test assertions; I did not independently replay the BUILD's full JSON log or mutations.
- Independently, the added negative tests failed on BUILD and the complete `channels` package passed after repair; `go build ./...` passed. The focused `engine` package ran against the disposable-MySQL script after repair, passed, and removed its isolated DB/network. Downstream preflight passed 7/7, catalog `-Check` passed, workspace doctor passed 25/25, docs build passed and `git diff --check` found no whitespace error.
- This evidence establishes local application semantics with a synthetic Graph transport and disposable MySQL only. It does not prove the real Facebook API's page shape, rate-limit behavior or complete live delivery, and makes no CVF AI-governance claim. PR #1's green GitHub Actions are for older remote head `3e0b37e`, not this BUILD or reviewer repair. No live channel/provider call, persistent DB, push, merge, deployment or FREEZE occurred.

## Disposition

R022 passes independent REVIEW for Facebook conversation-window coverage after the small repair. F02 remains OPEN for Pancake/Zalo. A later PR run is needed before claiming hosted CI for this local revision; FREEZE and deployed behavior remain open.
