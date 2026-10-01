# CCMAI-RUNTIME-023 — F02-B Pancake sync window coverage

Status: DISPATCH_READY / FREEZE_OPEN. Issued 2026-10-01 by Codex (ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR). Planning base/dispatcher seed: `5b785d593d20c5eb05cd54da1af5f068c8bed0f2`. Risk ceiling R2. Authority: [SPEC](../specs/RUNTIME_PANCAKE_SYNC_COVERAGE_F02B_2026-10-01.md), [F02 source finding](../reviews/CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md), [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), `CVF_SESSION/authority/CCMAI-RUNTIME-023.json`. R022 Facebook is REVIEW_PASS / FREEZE_OPEN; F02 remains OPEN for Zalo.

## Assignment and phase gate

Claude is IMPLEMENTATION_WORKER and COMMIT_STEWARD for one local BUILD commit; Codex independently reviews. Rehydrate manifest/policy, compact bootstrap or state, memory/handoff, status/index, this SPEC/order, dispatcher seed and R013–R016/R022 contracts; run workspace doctor. Record `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` and R023 acknowledgment in the active handoff **before** BUILD. Return exact local SHA, changed paths, test/negative evidence, claim limits and `REVIEW_PENDING`. The dispatcher seed is read-only for Claude.

## Allowed scope

- `backend/channels/pancake.go` and focused `backend/channels/pancake*_test.go`: exhaustive, fail-closed cursor traversal with a fixed upper time bound, safe errors and preserved message mapping/PII reduction.
- `backend/engine/sync.go` and focused `backend/engine/sync*_test.go`: Pancake exhaustive limit and success checkpoint bounded by fetch start, without changing Facebook/Zalo behavior, ownership or status fencing. A narrow clock test seam is allowed if required for an exact boundary assertion.
- One BUILD evidence record under `docs/reviews/`, this order/SPEC for necessary clarification, roadmap/status/continuity and catalog/index if registration changes. `.cvf/`, dispatcher seed, CVF core, Facebook/Zalo adapters, shared adapter interface, schema, API, UI, Compose, secrets and workflows are read-only.

## Build and proof sequence

1. Trace the real Pancake page response through `doRequest`, `FetchRecentConversations`, engine fetch, per-conversation writes and `recordSyncStatus`. Confirm current 100-row success, short-page/repeated-cursor/page-budget success, moving `until`, and finish-time checkpoint. Reconcile the official [Pancake OpenAPI](https://developer.pancake.biz/openapi/openapi.yaml) fields with the local fixtures; record any undocumented assumption explicitly before editing.
2. Implement the SPEC with finite page/request memory bounds. A nonempty short page continues; only a valid empty array terminates. Every failure returns an error, and any partial slice is diagnostic only. Preserve `type=INBOX`, last physical row cursor, metadata redaction, request pacing/retry contract and `FetchMessages` behavior. If a shared request helper must change to prevent credential leakage, keep the change narrow and cover positive and negative paths.
3. Add non-vacuous synthetic adapter tests for >130 rows, fixed `until`, boundary/interleaving, short page, duplicate and every listed failure class. Prove old source fails at least one new coverage assertion before repair using a reversible probe or recorded mutation, then restore all temporary edits. Add disposable-MySQL engine tests for >100 stored conversations/messages, failure checkpoint/no after-sync, retry/replay, and checkpoint ≤ fetch start on a delayed successful run. The tests must observe stored rows/checkpoint, not only call counts.
4. Run focused channel/engine and R013–R016/R022 regressions, full backend suite on disposable MySQL (`TEST_DB_DSN`, `-count=1 -p 1`, JSON log), R019 DB sentinel gate, `go build ./...`, downstream preflight, catalog `-Check`, doctor and `git diff --check`. Retain exact commands, results and cleanup evidence; never use the persistent Compose DB.
5. Commit only authorized files locally once, return `REVIEW_PENDING`, and leave PR #1 at its current remote head. If a real dependency requires interface/schema, Zalo/Facebook adapter, credential/OAuth, or product workflow changes, return `BUILD_BLOCKED` with evidence for Codex to revise scope; do not widen the seed. A small same-scope reviewer repair may be done by Codex under the existing CVF rule without a new tranche.

## Review boundary

R023 may pass independent REVIEW for Pancake conversation coverage while F02 remains OPEN for Zalo. Synthetic pages and disposable MySQL do not prove live Pancake completeness, actual response ordering under concurrent updates or CVF AI governance. No real channel/provider call, Alibaba key, persistent DB, push, merge, deployment, parent-CVF work or FREEZE under this order.
