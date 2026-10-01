# CCMAI-RUNTIME-022 — F02-A Facebook sync window coverage

Status: REVIEW_PENDING / FREEZE_OPEN. Issued 2026-10-01 by Codex (ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR). Planning base/dispatcher seed: `3f101ca58ecb7f9f3c56c4d46addf5ff74a7a39e`. Risk ceiling R2. Authority: [SPEC](../specs/RUNTIME_FACEBOOK_SYNC_COVERAGE_F02A_2026-10-01.md), [F02 source finding](../reviews/CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md), [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), `CVF_SESSION/authority/CCMAI-RUNTIME-022.json`. R021 F01-B is REVIEW_PASS / FREEZE_OPEN; R022 does not FREEZE or deploy F01.

## Assignment and phase gate

Claude is IMPLEMENTATION_WORKER and COMMIT_STEWARD for one local BUILD commit; Codex independently reviews. Rehydrate current manifest/policy, state/memory/handoff, status/index, this SPEC/order, dispatcher seed and relevant R013–R016 sync contracts; run workspace doctor. Record `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` and R022 acknowledgment in the active handoff **before** implementation. Return exact local SHA, changed paths, test/negative evidence, claim limits and `REVIEW_PENDING`. The seed belongs to Codex and is read-only for Claude.

## Allowed scope

- `backend/engine/sync.go`: choose exhaustive mode for Facebook and preserve other adapter limits, then preserve error/checkpoint/after-sync admission behavior.
- `backend/channels/facebook.go`: enumerate the `since` window across pages; validate page content and next URL; fail closed on incomplete coverage and sanitize credential-bearing request errors. Add only a narrow test seam if needed.
- Focused tests `backend/channels/facebook*_test.go` and `backend/engine/sync*_test.go`: synthetic Graph transport and disposable-MySQL engine/retry evidence. Existing sync tests may be edited only when their Facebook-limit expectation must reflect the SPEC; do not weaken ownership, lease or partial-failure assertions.
- One BUILD evidence record under `docs/reviews/`, this order/SPEC for necessary clarification, roadmap/status/continuity and catalog/index if registration changes. `.cvf/`, dispatcher seed, CVF core, non-Facebook adapters, API, UI, DB schema, Compose, secrets and workflows are read-only.

## Build and proof sequence

1. Trace `SyncReservedChannel` from reservation through adapter fetch, per-conversation writes and `recordSyncStatus`. Confirm the exact prior-checkpoint behavior on fetch error and the current Facebook `limit=100`/older-row shortcuts. Record the proposed changed set and complete accepted dependencies before editing.
2. Implement exhaustive Facebook enumeration within explicit finite safety bounds. Every page including an empty page with `paging.next` must be handled according to the SPEC; a cap, cycle, unsafe URL, malformed row or page failure returns a bounded error. Keep token/URL/body out of errors and logs. The engine must not treat an error accompanied by partial rows as success.
3. Add non-vacuous adapter tests for >250 rows, interleaved old/new and boundary-equal timestamps, malformed/cyclic/unsafe pagination and safe error text. Add disposable-MySQL engine tests for >100 stored rows, failed later page preserving checkpoint/no after-sync, retry/upsert idempotence, and non-Facebook limit unchanged. Prove the old implementation fails at least one new coverage assertion before repair, then restore all temporary edits.
4. Run focused tests and R013–R016 regressions, full backend suite on disposable MySQL (`TEST_DB_DSN`, `-count=1 -p 1`, JSON log), R019 DB sentinel gate, `go build ./...`, downstream preflight, catalog `-Check`, doctor and `git diff --check`. Retain exact command/result counts and cleanup evidence; never use the persistent Compose database.
5. Commit only authorized files locally once, return `REVIEW_PENDING`, and leave PR #1 at its existing remote head. If a source dependency requires changing another adapter, interface, schema or credential contract, return `BUILD_BLOCKED` with evidence for Codex to amend the order; do not broaden scope silently. A small same-scope reviewer repair may be done by Codex under the existing CVF rule without a new tranche.

## Review boundary

R022 may pass independent REVIEW for Facebook coverage while F02 remains OPEN for Pancake/Zalo. Passing synthetic pages and disposable MySQL do not prove live Facebook completeness. No real channel/provider call, Alibaba key use, persistent DB, push, merge, deployment, parent-CVF work or FREEZE under this order.
