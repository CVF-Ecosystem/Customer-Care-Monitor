# CCMAI-REPO-MIGRATE-001 — move the active project to the new repository

**Date:** 2026-09-29 · **Authority:** owner confirmed the three-person contributor display and directed migration to `CVF-Ecosystem/Customer-Care-Monitor` · **Risk ceiling:** R2 · **Status:** BUILD complete / REVIEW_PENDING.

## INTAKE, DESIGN and SPEC

The new repository already contains current `main`; the owner will remove the old repository separately. Make the new repository the canonical development and documentation target. Keep the product name Customer Care Monitor AI, the MIT copyright and permission notices, and the historical import commit unchanged. The import commit records provenance, not current product identity.

Acceptance: Go builds under the new module path; active clone/source/docs links and VitePress base use the new repository; local `origin` tracks the new repository; only `main` is pushed; docs and application builds pass; GitHub Actions and Pages are checked and any remaining Pages setting is reported accurately. Preserve historical evidence URLs as records of the old repository; do not pretend old PR/run IDs exist in the new one.

## Bounded BUILD

- Change only the exact Go module/import prefix in `backend`, the update-check release endpoint, current README/install/introduction and application documentation links, VitePress base/social link, and the one escaped Markdown placeholder that blocks docs build. Include necessary documentation-link fixes if the docs build exposes dead links; keep embedded screenshot images local.
- Update this work order, migration evidence, and continuity/catalog records. Switch local `origin` to the named new repository only after checking its `main` tip. Push only `main`, without force, tags, archived CQA history, or secrets.
- Verify Go tests/build, frontend build, docs build, catalog/doctor and diff. Check the remote SHA and Pages/Actions status after push.
- Do not delete or alter the old repository, rewrite existing commit messages, change `LICENSE`, copy CQA history into the new repository, use provider credentials, or claim CVF runtime governance behavior.

The screenshot supplied by the owner resolves the CREDIT-002 rendered-sidebar observation: exactly Blackbird081, claude and codex are visible. CREDIT-002 remains separately reviewable; this migration does not silently FREEZE it.

Role route: `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` (Codex). An independent R2 REVIEW follows BUILD; no self-approval or FREEZE.
