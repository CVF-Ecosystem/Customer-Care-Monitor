# Work order CCMAI-UX-001a — display data fixes

**State:** `WORK_ORDER` (BUILD authorized) · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** owner instruction of 2026-09-28, [SPEC](../specs/DISPLAY_DATA_FIXES_UX001A_2026-09-28.md), [roadmap](../roadmaps/UI_UX_REDESIGN_ROADMAP_2026-09-27.md) Phase 1. The roadmap deferred this SPEC until R008 finished; R008 and R009 are REVIEW PASS, and the demo-brand tranche is REVIEW PASS, so no open tranche edits the same files.

## Entry and role route

Stacked on `CCMAI-UX-000` (uses its `format.ts`); declared in the SPEC. Role route `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` (Claude), acknowledged in the handoff. One local BUILD commit, `REVIEW_PENDING`; no self-approval or FREEZE.

## Allowed scope

- `frontend/src/views/Dashboard.vue` (relative time; stats card label/hint), `frontend/src/views/Jobs/JobDetail.vue` (trend grouping; classification count label);
- `frontend/src/utils/format.ts` (add `vnDateKey`), new `frontend/src/utils/trend.ts` if the trend grouping is extracted for testing;
- `frontend/src/i18n/vi.ts`, `en.ts` (additive keys only);
- `backend/api/handlers/demo.go` (timestamp placement only, plus a package-level clock seam), `backend/api/handlers/demo_test.go` (new test);
- new frontend tests; BUILD evidence `docs/reviews/DISPLAY_DATA_FIXES_UX001A_BUILD_2026-09-28.md` + assets; continuity files.

Not allowed: `dashboard.go` or any API response change, models/migrations, stores, other views, running database, CVF core.

## Evidence and ceiling

SPEC acceptance gates plus `git diff --check`, catalog `-Check`, doctor. Disposable MySQL and the disposable screenshot environment only. No provider call, real channel sync, customer data, persistent `ccma` change, deployment, push or FREEZE. Existing demo rows in any database keep their old timestamps until someone resets and re-imports demo data; this tranche does not do that.
