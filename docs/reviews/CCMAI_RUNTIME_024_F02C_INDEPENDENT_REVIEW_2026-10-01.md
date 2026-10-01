# CCMAI-RUNTIME-024 / F02-C — independent review

Date: 2026-10-01. Reviewer: Codex, independent of Claude's BUILD.
Target BUILD: `561bfaebdf9c14bbf420a8e48101742c16b24047`; parent `97a65ecb8524ded9c7d4b8f3705e5b665bbf5432`; dispatcher seed `b145d71091d3117c12d4479415b8e88b67e0b3cb` committed by Codex before BUILD and unchanged. Intake correction: `024b9f2` (memory pointer only). Before intake the worktree had only pre-existing untracked `knowledge/_index.json`; it remains outside commits. Authority: [SPEC](../specs/RUNTIME_ZALO_SYNC_COVERAGE_F02C_2026-10-01.md), [order](../work_orders/CCMAI_RUNTIME_024.md), [worker evidence](RUNTIME_ZALO_SYNC_COVERAGE_F02C_BUILD_2026-10-01.md).

## Consolidated source and claim review

| Item | Conclusion and evidence |
|---|---|
| Changed set and independent roles | BUILD's 14 files fit seed paths plus permitted evidence/continuity. No seed, adapter interface, schema, OAuth protocol, lease eligibility, non-Zalo adapter or workflow change. Claude's acknowledgment predates BUILD; Git uses one human identity, so role attribution rests on recorded dispatch/worker evidence, not Git identity alone. |
| Two existing tests changed | Accepted under SPEC clauses 1/6: the R022 test asserts all known adapters receive limit 0 through real sync; the R023 table now expects Zalo limit 0 and fetch-start checkpoint. Facebook/Pancake and unknown-type expectations remain. No DB assertion removed. |
| Conversation traversal | `FetchRecentConversations` validates every physical row, preserves string IDs and exact JSON numbers, advances offset by physical rows, includes since equality, dedupes newest rows with first-seen tie, and requires an explicit empty array. Short pages continue; malformed old/duplicate rows fail. 500-page budget, cycle, positive-limit overflow and pre/post-response cancellation fail with identifiable incomplete-coverage error. Engine discards partial results on error. |
| Shared helper behavior | Non-2xx/body bound and malformed-code checks apply to messages/health too; allowed by SPEC. `FetchMessages` and `extractZaloDataArray` mapping/pagination/selection are unchanged. Strict conversation parsing is separate. Missing error remains compatible for OAuth success with a valid token pair; conversation pages separately require numeric zero. |
| Refresh and error safety | Numeric `-216` triggers at most one refresh/retry for the same request. Token-pair validation precedes callback; persistence precedes memory replacement; failure stops retry and preserves `errors.Is`. Fixed error text omits URLs/bodies/provider messages/credentials. A numeric-string admission defect was independently found and repaired below. |
| Checkpoint and lease | Engine now requests exhaustive Zalo and selects the existing second-truncated fetch-start checkpoint; completion remains in updated_at. Ownership/error/partial/after-sync gates unchanged. Zalo remains excluded from lease recovery. |
| Claim limits | Synthetic proofs establish application semantics. Real offset reordering, provider retention/access limits and complete message coverage are unproved. F02 remains OPEN; old hosted PR evidence at `3e0b37e` does not prove these later local changes. |

## Independent finding and bounded reviewer repair

The new `zaloEnvelopeError` decoded a raw code directly into `json.Number`. An independent synthetic probe demonstrated that numeric **strings** were admitted:

- `{"error":"-216"}` caused GET → refresh POST → retry (3 requests), with persistence callback called, instead of rejecting the malformed code before refresh.
- Refresh `{"error":"0",...valid pair...}` returned nil and called persistence, instead of rejecting the malformed code.

Both executable probes failed against BUILD. This is one local type-validation root cause within SPEC's malformed-envelope contract. Reviewer-local repair assessment: same objective/path class/R2/effects, no real API/credential use, no new OAuth protocol or interface, a six-line helper correction and focused regressions; no REWORK tranche needed under the CVF small-repair rule. Codex acts as REVIEWER performing bounded repair, then SESSION_SYNC_STEWARD/COMMIT_STEWARD for the review result; Claude's BUILD attribution remains intact.

Repair decodes with UseNumber into an interface and requires the dynamic value to be `json.Number` before Int64. Numeric-string codes are now rejected before refresh/persistence. `backend/channels/zalo_review_probe_test.go` retains the two failing probes, messages/health numeric-string negatives with numeric-zero positive controls, and refresh positive controls for numeric zero or absent error. Existing persistence-failure and safe error tests remain unchanged. No unrelated product repair was made.

## Executed evidence and disposition

Independent adapter suite and `go build ./...` passed on BUILD before repair. Two new probes failed as described, then the complete channels suite and build passed after repair (`go test ./channels -count=1`, 2.191 s). Synthetic transports prevent real channel/provider calls.

Independent engine command: `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages ./engine -Run 'Zalo|Facebook|Pancake|ConversationFetchLimit|SyncStatus|Refresh|Lease' -VerboseTests` passed, exit 0, 128.101 s, with the real engine/adapters and synthetic transport. This run compiled before the envelope type repair; engine source was unchanged by that repair, and the complete post-repair channels suite separately verifies the revised helper and all refresh regressions. It covered >100 stored rows, failed-page checkpoint/no after-sync, retry/replay, token persistence, delayed-run checkpoint and known-channel selection/lease cases. Wrapper removed disposable DB `ccma-test-db-66628e15` and network `ccma-test-net-66628e15`; subsequent Docker name queries returned neither. Persistent Compose DB was not used.

Repository checks: docs build PASS; catalog check PASS; doctor 25/25 PASS; diff check PASS; explicit preflight covering all 11 review changed files PASS 7/7 (catalog included). Explicit changed-file preflight is required because the unrelated pre-existing untracked knowledge index keeps default whole-worktree scope preflight non-green. Worker-reported full suite (546 pass / 0 fail / 2 optional skips), R019 five-sentinel gate and 18 adapter/2 engine mutations were inspected as BUILD evidence, not independently rerun or represented as hosted proof.

Disposition: **REVIEW_PASS / FREEZE_OPEN** for BUILD plus the bounded reviewer type repair and retained negative/positive regressions. R024's original BUILD did not satisfy the numeric-code contract until that repair. F02 remains OPEN for real offset stability/provider limits and complete message coverage; the next roadmap move is bounded F03 planning. No push, merge, deployment, parent-CVF implementation/test, real channel/provider call or FREEZE. The review commit contains the helper repair, four regression test functions, this review, learning intake and synchronized continuity; only the knowledge index remains untracked outside it.
