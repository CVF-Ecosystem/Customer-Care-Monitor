# CCMAI-RUNTIME-018 — independent R2 review

**Date:** 2026-09-29 · **Reviewer:** Codex (`REVIEWER`, independent of Claude BUILD) · **Base/BUILD:** `7582064..f4bc93c` · **Disposition:** `CHANGES_REQUIRED / REVIEW_OPEN` · **FREEZE:** open.

## Scope and verification

- Rehydrated the current CVF state and acknowledged `COMMIT_STEWARD (Claude) → REVIEWER (Codex)` in the active handoff before review checks. The pinned public core matches `origin/main`; workspace doctor passed 25/25.
- Compared the complete BUILD diff with the R018 SPEC and work order. Source changes stay within the authorized paths. The server-owned marker, shared conditional reservation predicate, scheduler and lease recovery exclusions, agent `sync_all` skip, and manual 409 path are present. The positive real-channel tests are meaningful.
- Independently ran `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Run 'DemoFixture|DemoChannel'` (exit 0; this regex selected the DB fixture tests) and `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages './...'` (exit 0; all tested packages, including handlers, db and engine). Each run used its own disposable MySQL and removed its container/network. No persistent Compose database, real channel or provider was contacted. These tests are not live CVF governance proof.

## Blocking finding — R018-R1: legacy match and cleanup are case-insensitive

`backend/db/mysql.go` uses `=` for `channel_type` and `external_id`, and `=`/`LIKE` for `last_sync_status` and `last_sync_error`, without a binary or case-sensitive collation. In the test's actual `channels` schema, all four text columns use `utf8mb4_0900_ai_ci`. On the same disposable MySQL, `SELECT 'Demo-Zalo-OA' = 'demo-zalo-oa', 'Decrypt failed: x' LIKE 'decrypt failed:%'` returned `1, 1`; the corresponding `BINARY` comparisons returned `0, 0`.

Consequences: a channel whose type or external ID differs in case from the historical fixture identity can be marked when the other predicates match. A marked fixture with differently cased `error` status or `decrypt failed:` prefix can have its status/error cleared despite not meeting the SPEC's exact cleanup condition. The existing backfill matrix tests only the canonical casing, so M8–M11 do not detect this. This breaches the exact legacy identity and narrowly bounded cleanup requirements. The false marker matters because it blocks all later sync admissions for that row.

**Required repair:** Make the type/external-ID identity and error status/prefix comparisons case-sensitive in the backfill SQL without changing the database's global collation or credential format. Add disposable-MySQL cases for wrong-case type, wrong-case external ID, wrong-case status and wrong-case decrypt prefix; each must remain untouched while canonical rows still mark/clear and a second migration stays idempotent. Include a mutation or equivalent detector showing these new cases fail if the case-sensitive predicate is removed. Keep source changes in `backend/db/mysql.go` and focused `backend/db/` tests, then return `REVIEW_PENDING` for another independent review.

## Boundaries

The full backend suite passed, but that cannot override the demonstrated SPEC mismatch. No source repair, push, deployment, live governance claim or FREEZE occurred in this review. Mixed-version rollout remains unsupported; Zalo/legacy crash recovery and UX-016 F1 remain separate.
