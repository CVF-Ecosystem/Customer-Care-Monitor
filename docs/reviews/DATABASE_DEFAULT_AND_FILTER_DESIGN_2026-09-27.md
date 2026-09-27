# Bằng chứng BUILD: database mặc định và thiết kế filter pipeline

**Work order:** `CCMAI-DATABASE-001` · **Ngày:** 2026-09-27 · **Kết quả:** BUILD_PASS / OWNER_REVIEW_ACCEPTED.

## Changed behavior

- Fresh install defaults `DB_NAME`/`MYSQL_DATABASE`/backend fallback to `CCMA`.
- Explicit `DB_NAME`/`DB_USER` values continue to override the defaults for separately configured environments.
- No prior CQA data was imported or migrated. The new persistent `CCMA` database was initialized with the application's current tables.
- The MySQL → Go snapshot/gate → provider flow and selected `pg-jev` lessons are specified, not implemented as a runtime gate.

## Source review

`Blackbird081/pg-jev` uses PostgreSQL row functions backed by TypeSafe Jev. Its documented useful patterns are cheap predicates before semantic evaluation, narrow row projection, measured batch sizing, content cache, row/character spend caps, concurrency/timeout/cancellation, and request/token/cost statistics. Its PostgreSQL 14–17, `plpython3u`, superuser and third-party row disclosure requirements do not fit this application's current MySQL/provider-boundary decision. Source: <https://github.com/Blackbird081/pg-jev>.

## Validation

- `C:\Program Files\Go\bin\go.exe test ./config`: PASS.
- `C:\Program Files\Go\bin\go.exe build ./...`: PASS.
- `docker compose config --quiet` using a temporary ignored `.env` copied from `.env.example` and validation-only environment values: PASS. The temporary `.env` was removed.
- Live fresh-volume Compose validation using isolated project `ccmai-db-validation-20260927a`: PASS. MySQL 8 reached `healthy`; `information_schema` returned exact schema `CCMA` with `utf8mb4` / `utf8mb4_unicode_ci`, and returned no application schema named `cqa`.
- Application user validation: PASS. User `cqa` connected to database `CCMA`, observed the expected charset/collation, then created, inserted, selected and dropped `ccma_validation_probe`. The final table-existence count was `0`.
- Isolation cleanup: PASS. The validation container, network and project-scoped volume were removed with Compose; label-based follow-up found no remaining project container or volume. The ignored `.env` containing validation-only values was removed.
- Persistent development setup: PASS. Compose project `ccma` retains `ccma_mysql_data`; exact database `CCMA` and application user `ccma` authenticate successfully. The git-ignored local `.env` uses generated secrets.
- Authenticated readiness: PASS. The Compose healthcheck now logs in as the application user and executes `SELECT 1` against the configured database, preventing the temporary initialization server from being treated as ready.
- Application initialization: PASS. The backend image built and AutoMigrate created 16 tables in `CCMA`. The app and database containers remain running; the database volume was deliberately retained.
- Restart idempotence: PASS after repair. The first restart exposed duplicate unique-index DDL in the inherited manual migration. `addUniqueConstraints` now skips indexes already present and returns unexpected DDL errors. Containerized Go 1.26 tests for `./config ./db` passed; rebuild/restart logs show a successful migration with no duplicate-index or connection error, and all three expected unique indexes remain present.
- Outbound-default correction: the first backend start automatically fetched the public LiteLLM pricing dataset and logged 395 models. This was a metadata HTTP fetch without provider credential or customer content, not an LLM call. Because it was unnecessary for database initialization, `PRICING_SYNC_ENABLED` now defaults to `false` in config and `.env.example`; the retained local `.env` also disables it. Final restart logs confirm the static table path and no pricing fetch.
- `npm run docs:build` with the local Node path: PASS. Initial build exposed links from public docs to governance folders excluded by VitePress; those references were changed to source-repo paths and the rerun passed.
- Governed catalog `-Check`: PASS.
- Local Markdown link resolution for changed product/governance pages: PASS.
- `git diff --check`: PASS at pre-commit validation.

The live checks prove fresh initialization and restart behavior on this workspace's retained development volume. They do not test migration of an existing external database. No AI/LLM provider API was called; the initial public pricing metadata fetch is disclosed above and was disabled by default afterward. This artifact makes no CVF runtime-governance claim.

## Disposition

The owner clarified that this workspace is a new product development environment with no CQA database content to preserve. The persistent `CCMA` setup and database identity are accepted; no standalone database review or CQA migration remains. Any future import of an external database, production deployment, machine-gate runtime or provider behavior is a separate governed scope.
