# Bằng chứng BUILD: database mặc định và thiết kế filter pipeline

**Work order:** `CCMAI-DATABASE-001` · **Ngày:** 2026-09-27 · **Kết quả:** BUILD_PASS / REVIEW_PENDING.

## Changed behavior

- Fresh install defaults `DB_NAME`/`MYSQL_DATABASE`/backend fallback to `CCMA`.
- An explicit `DB_NAME`, including legacy `cqa`, continues to override the default.
- No schema/table/data migration or database-engine change was performed.
- The MySQL → Go snapshot/gate → provider flow and selected `pg-jev` lessons are specified, not implemented as a runtime gate.

## Source review

`Blackbird081/pg-jev` uses PostgreSQL row functions backed by TypeSafe Jev. Its documented useful patterns are cheap predicates before semantic evaluation, narrow row projection, measured batch sizing, content cache, row/character spend caps, concurrency/timeout/cancellation, and request/token/cost statistics. Its PostgreSQL 14–17, `plpython3u`, superuser and third-party row disclosure requirements do not fit this application's current MySQL/provider-boundary decision. Source: <https://github.com/Blackbird081/pg-jev>.

## Validation

- `C:\Program Files\Go\bin\go.exe test ./config`: PASS.
- `C:\Program Files\Go\bin\go.exe build ./...`: PASS.
- `docker compose config --quiet` using a temporary ignored `.env` copied from `.env.example` and validation-only environment values: PASS. The temporary `.env` was removed.
- `npm run docs:build` with the local Node path: PASS. Initial build exposed links from public docs to governance folders excluded by VitePress; those references were changed to source-repo paths and the rerun passed.
- Governed catalog `-Check`: PASS.
- Local Markdown link resolution for changed product/governance pages: PASS.
- `git diff --check`: PASS at pre-commit validation.

Docker Desktop was not running, so no MySQL container or data migration test was attempted. Compose rendering does not prove a fresh MySQL volume was initialized. No provider API was called; this artifact makes no CVF runtime-governance claim.

## Review focus

Independent R2 review should verify exact-case `CCMA` portability, the legacy `DB_NAME=cqa` upgrade warning, absence of implicit migration, and the boundary between patterns learned from `pg-jev` and dependencies actually adopted.
