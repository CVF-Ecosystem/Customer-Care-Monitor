# CCMAI-UX-014 independent review — Settings, Users, Login, Setup

**Date:** 2026-09-28 · **Reviewer:** Codex (`REVIEWER`) · **Build commit:** `23ec432` · **Disposition:** `CHANGES_REQUIRED` (UX014-R1); no `REVIEW_PASS` or `FREEZE`.

The changed implementation paths match the UX-014 work order. Source and focused tests confirm that a failed Settings load hides its forms, the saved-key test is labelled accurately, user password rules match the server, and Login preserves its auth flow. Focused UX-014 tests pass 6/6; the shared frontend suite passes 151/151; forced `vue-tsc`, production build, and the commit whitespace check pass. The inspected desktop storage capture renders cleanly. Tests/captures use mocked or synthetic data; no provider or S3 test was pressed.

## R1-1 — S3-off dialog provides a command that cannot target the current app service

`frontend/src/views/Settings.vue` still presents `docker exec cqa-app /app/cqa-server migrate-files -down -apply`. The current `docker-compose.yml` defines an `app` service with no `container_name`; `docker compose config --services` confirms `app`, and the local project uses `ccma-app-1`. Thus a user copying the dialog's command cannot reach the app container. This string predates UX-014, but UX-014 preserves and presents it as an actionable command in the redesigned S3-off dialog. It matters because the dialog says to copy files back **before** turning S3 off.

**Repair acceptance:** Use the Compose service form from the project directory, `docker compose exec app /app/cqa-server migrate-files -down -apply`, or another command verified against the current deployment instructions. Keep the `-down -apply` operation and the existing S3-off confirmation behavior. Add a focused assertion for the displayed command. The stale command in `docs/guide/s3-storage.md` is a separate documentation correction; do not expand this UI repair into backend/storage behavior.

## R1-2 — Required Setup and S3-off visual states are absent

SPEC §6 requires disposable captures of Setup across desktop/mobile and light/dark. BUILD evidence explicitly says Setup was not captured because its disposable environment had already completed setup. The captured storage page shows only local storage; it does not show the S3-off dialog whose command is being reviewed.

**Repair acceptance:** Capture Setup in a fresh disposable pre-setup state or an isolated rendering harness at desktop/mobile × light/dark, and capture the S3-off dialog with synthetic S3 configuration at least on desktop and mobile. Record JS errors, overflow, and outbound-request results. Keep the persistent `ccma` stack and real credentials untouched.

**Next governed move:** After completing UX-015's independent BUILD commit, Claude may perform UX014-R1 within the UX-014 work order (`Settings.vue`, focused test, scoped evidence/captures and continuity), then return `REVIEW_PENDING` for Codex re-review. No provider test, S3 request, push, deployment, or FREEZE is authorized.
