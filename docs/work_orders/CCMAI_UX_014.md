# Work order CCMAI-UX-014 — Settings, Users, Login and Setup (frontend only)

**State:** `BUILD` authorized under this work order; returns `REVIEW_PENDING` · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** owner direction 2026-09-28 (UI series UX-012..015) and [SPEC](../specs/SETTINGS_USERS_AUTH_UX014_2026-09-28.md) with canvas version `1790607141-5a73`.

## Role route

`ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` (Claude), each acknowledged in the active handoff. One local commit, `REVIEW_PENDING`; no self-PASS, FREEZE or push.

## Allowed paths (exhaustive)

- `frontend/src/views/Settings.vue`, `Users.vue`, `Login.vue`, `Setup.vue`;
- `frontend/src/i18n/vi.ts`, `en.ts` — additive `st_*` / `us_*` / `au_*` keys only;
- new `frontend/src/__tests__/settings-users-auth.spec.ts`;
- evidence `docs/reviews/SETTINGS_USERS_AUTH_UX014_BUILD_2026-09-28.md`, `docs/reviews/assets/ux-014-2026-09-28/**`;
- this work order, SPEC status, roadmap row, continuity files.

Not allowed: layouts (`AuthLayout.vue`, `DefaultLayout.vue`), shared components (imported only), stores, router, API module, backend, permission values, repository scripts, CVF core. Evidence must not press "Kiểm tra key đã lưu", refresh the provider model list, test S3 against a real endpoint, or display secrets.

## Requirements, evidence, ceiling

SPEC §2, §4 and §6. Local source/tests/docs and a disposable screenshot environment (removed afterwards); synthetic data only; no provider call, no external storage call, no persistent `ccma` change, deployment, push or FREEZE. A gate that cannot pass within scope → `BUILD_BLOCKED`.

## Independent review UX014-R1 (Codex, 2026-09-28)

`CHANGES_REQUIRED`: [R1-1 and R1-2](../reviews/CCMAI_UX_014_INDEPENDENT_REVIEW_2026-09-28.md). Correct the S3-off dialog's stale `docker exec cqa-app` command within `Settings.vue`, assert the displayed command in the focused test, and complete the missing Setup and S3-off-dialog visual evidence with synthetic disposable state. Keep the existing S3 confirmation and request behavior. Repair stays within the allowed paths and returns `REVIEW_PENDING`; no live provider or storage test.

**R1 disposition:** [Codex independent re-review](../reviews/CCMAI_UX_014_R1_INDEPENDENT_REREVIEW_2026-09-28.md) is `REVIEW_PASS`; both findings resolved. FREEZE remains open. The stale-token `/setup` hang and the old command in `docs/guide/s3-storage.md` are separate follow-ups.

**Repair R1 result (Claude, 2026-09-28):** done within the allowed paths; `REVIEW_PENDING` for Codex re-review ([evidence addendum](../reviews/SETTINGS_USERS_AUTH_UX014_BUILD_2026-09-28.md#repair-r1--ux014-r1-claude-repair_worker-2026-09-28)).
