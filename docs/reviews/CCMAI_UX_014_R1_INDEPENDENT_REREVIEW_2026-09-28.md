# CCMAI-UX-014 R1 independent re-review

**Date:** 2026-09-28 · **Reviewer:** Codex (`REVIEWER`) · **Repair commit:** `05d3c11` · **Disposition:** `REVIEW_PASS`; findings R1-1 and R1-2 resolved. `FREEZE` remains open.

R1-1: the S3-off dialog now displays `docker compose exec app /app/cqa-server migrate-files -down -apply` with a note to run it where `docker-compose.yml` resides. The Compose file defines service `app` without a fixed container name; the Dockerfile installs `/app/cqa-server`, and the CLI accepts `-down -apply`. The focused test verifies the exact command and that opening the dialog makes no PUT or POST. The confirmation flow is unchanged.

R1-2: the repair adds desktop/mobile × light/dark captures of Setup before account creation and the S3-off dialog with synthetic S3 settings. I inspected a mobile dark Setup capture and a desktop light S3-off capture. The evidence records 12 captures with no JS errors, overflow, failed steps, or external requests; no S3 operation was requested.

Independent checks: focused UX-014 tests passed 7/7; the complete frontend suite passed 160/160; forced `vue-tsc`, production build, repair diff whitespace check, and workspace doctor 25/25 passed. These are UI checks only; no provider or CVF governance claim is made.

Separate follow-ups: `docs/guide/s3-storage.md` still contains the old `cqa-app` command. A stale invalid browser token can hang `/setup` on an unconfigured installation; this was observed during capture and belongs to a separate auth/router tranche. Neither item changes the disposition of this bounded UX-014 repair. No push, deployment, or FREEZE.
