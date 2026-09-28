# Work order CCMAI-UX-013 — AI job list, create and edit (frontend only)

**State:** `BUILD` authorized under this work order; returns `REVIEW_PENDING` · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** owner direction 2026-09-28 (UI series UX-012..015) and [SPEC](../specs/JOBS_SCREENS_UX013_2026-09-28.md) with canvas version `1790603381-76c3`.

## Role route

`ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` (Claude), each acknowledged in the active handoff. One local commit, `REVIEW_PENDING`; no self-PASS, FREEZE or push.

## Allowed paths (exhaustive)

- `frontend/src/views/Jobs/JobList.vue`, `JobCreate.vue`, `JobEdit.vue`;
- `frontend/src/components/JobWizard/*.vue` and `frontend/src/components/CronPicker.vue` (used only by JobCreate/JobEdit; `StepAnalysisSchedule.vue` is unused and left as is);
- new `frontend/src/views/Jobs/job-list/**` (pure helpers);
- `frontend/src/i18n/vi.ts`, `en.ts` — additive `jl_*` / `jw_*` keys only;
- new `frontend/src/__tests__/jobs-screens.spec.ts`;
- evidence `docs/reviews/JOBS_SCREENS_UX013_BUILD_2026-09-28.md`, `docs/reviews/assets/ux-013-2026-09-28/**`;
- this work order, SPEC status, roadmap row, continuity files.

Not allowed: `JobDetail.vue` and `job-detail/**` (imported read-only), UX-000 shared components (imported only), stores, API module, composables, utils, other views, backend, repository scripts, CVF core, provider calls. Telegram test sends are not performed in evidence (they would call an external service).

## Requirements, evidence, ceiling

SPEC §2, §4 and §6. Local source/tests/docs plus a disposable screenshot environment (removed afterwards); synthetic data; no provider call, no test-output send, no real sync, no customer data, no persistent `ccma` change, deployment, push or FREEZE. A gate that cannot pass within scope → `BUILD_BLOCKED`.
