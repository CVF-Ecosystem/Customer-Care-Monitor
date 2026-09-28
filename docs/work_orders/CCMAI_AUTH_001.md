# Work order CCMAI-AUTH-001 — stale browser token must not hang first-time Setup

**Phase:** WORK_ORDER → BUILD authorized; return `REVIEW_PENDING` · **Risk ceiling:** R2 · **Implementation:** Claude (`IMPLEMENTATION_WORKER`) · **Independent review:** Codex (`REVIEWER`) · **Authority:** owner-directed continuation on 2026-09-28, [SPEC](../specs/SETUP_STALE_TOKEN_RECOVERY_AUTH001_2026-09-28.md), and observed UX-014 R1 evidence.

## Role route and bounded scope

Codex completed `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR`. Claude rehydrates canonical continuity, records `WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER` in the active handoff **before BUILD**, then performs `IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD`. One local commit returns `REVIEW_PENDING` to Codex. No self-PASS, FREEZE or push.

Allowed application paths: `frontend/src/router/index.ts`, `frontend/src/stores/auth.ts`, `frontend/src/App.vue`, `frontend/src/api/index.ts`, and one focused new test file under `frontend/src/__tests__/`. Change only the files needed for the SPEC; explain why each changed file is needed. Allowed documentation: this work order, the SPEC status, `docs/reviews/SETUP_STALE_TOKEN_RECOVERY_AUTH001_BUILD_2026-09-28.md`, `CVF_SESSION_MEMORY.md`, `CVF_SESSION/ACTIVE_SESSION_STATE.json`, the active handoff, `IMPLEMENTATION_STATUS.json`, and generated catalog/index files if the catalog tool requires them.

Do not edit the existing UX-014 files/tests, backend, storage guide, provider settings, deployment files, CVF core or persistent `ccma` stack. A newly discovered necessary path or backend contract change is a boundary change: record it and return `BUILD_BLOCKED` for Codex routing.

## Required implementation and evidence

Meet SPEC §§1–2. First add an executable failing regression for the real guard's stale-token case, then make the smallest cohesive fix. Guard against both the `/setup ↔ /` redirect and the stale-token profile/refresh path. Preserve configured-installation auth behavior and successful Setup's new token. Use synthetic tokens and mocked API responses; do not use saved provider credentials, send notifications or claim live governance proof.

BUILD evidence must state the exact source cause, changed paths, before/after regression result, complete acceptance matrix, commands/results, and any remaining limitation. Run focused and full frontend tests, forced `vue-tsc`, production build, `git diff --check`, catalog `-Check`, and project doctor. If a disposable browser environment is used, remove it and record cleanup. Review the changed set before local commit. No deployment, push or FREEZE.
