# CCMAI-UX-017 — Channels sync-status presentation

**Date:** 2026-09-30 · **Phase:** WORK_ORDER · **Risk ceiling:** R2 · **Authority:** [UX-017 SPEC](../specs/CHANNELS_SYNC_STATUS_TRUTH_UX017_2026-09-30.md), [UI roadmap](../roadmaps/UI_UX_REDESIGN_ROADMAP_2026-09-27.md), [R018 independent re-review](../reviews/CCMAI_RUNTIME_018_R1_INDEPENDENT_REREVIEW_2026-09-29.md).

## Roles and objective

Codex is `ORCHESTRATOR -> SPEC_AUTHOR -> WORK_ORDER_AUTHOR`, then independent `REVIEWER`. Claude is `IMPLEMENTATION_WORKER -> SESSION_SYNC_STEWARD -> COMMIT_STEWARD` for one local BUILD/evidence commit. Claude must rehydrate canonical CVF state and record `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` in the active handoff before source edit. Implement the UX-017 SPEC on the Channels list and detail so status, last successful sync and unresolved polling are shown with their actual meanings.

## Allowed paths

- Source: `frontend/src/views/Channels.vue`, `frontend/src/views/Channels/ChannelDetail.vue`, `frontend/src/i18n/vi.ts`, `frontend/src/i18n/en.ts`.
- Focused frontend tests under `frontend/src/__tests__/` for these two screens. `SyncStatusChip.vue`, `frontend/src/utils/review.ts`, channel store/API and router are read-only unless a failing acceptance case proves a necessary change; in that case return `BUILD_BLOCKED` with the smallest proposed boundary amendment.
- Evidence: new `docs/reviews/CHANNELS_SYNC_STATUS_TRUTH_UX017_BUILD_2026-09-30.md`. Continuity/status: active handoff/state, session memory, `IMPLEMENTATION_STATUS.json`, and UI roadmap only if needed to record BUILD truth. Do not edit unrelated worktree files.

No backend, API response or DB schema change; no demo reset/backfill, sync engine/adaptor/OAuth change, provider/channel request, persistent Compose database, deployment, push, CVF core edit or FREEZE.

## BUILD acceptance and gates

1. Use the existing shared sync-status presentation on list and detail. Preserve all six states including unknown. Rename the timestamp to last **successful** sync on both screens; show a clear no-success value for null, and keep an older successful timestamp visible beside a later partial/error/syncing status. Do not infer a last-attempt time or demo marker.
2. Preserve manual 202/start wording. Detail polling timeout or fetch failure must say completion is unconfirmed, while observed success/partial/error retain their separate wording. A generic sync error must not make the optional reauth control look required or credential-specific; retain its existing permission/channel-type boundary and route.
3. Add mounted synthetic UI tests for the SPEC's status/time matrix, start-versus-finish, polling uncertainty, reauth affordance and vi/en labels. Include at least one negative detector that would fail if empty status returned to `—` or partial/error displayed the old success time as the latest attempt. Mock responses here are UI-only evidence.
4. Run focused and full frontend Vitest, `npx vue-tsc -b --force`, `npm run build`, catalog `-Check`, doctor and `git diff --check`. Record exact commands/results, test data, claim limits and the changed set in BUILD evidence. If a required screen state cannot be tested or the UI needs data absent from the current API, return `BUILD_BLOCKED` with evidence; do not expand scope silently.
5. Synchronize handoff/state/memory/status, stage only allowed files, then make one local `REVIEW_PENDING` BUILD/evidence commit for Codex. Claude must not self-PASS, push, deploy or FREEZE.

## Independent review

Codex will inspect the exact diff against this SPEC/work order and independently rerun focused UI tests. Passing synthetic UI tests does not establish real-channel completeness, provider behavior or CVF runtime governance. Zalo/legacy/mixed-version recovery and S1 closure remain open.

## Repair addendum UX017-R1 — 2026-09-30

[Codex independent review](../reviews/CCMAI_UX_017_INDEPENDENT_REVIEW_2026-09-30.md) returned `CHANGES_REQUIRED / REVIEW_OPEN`: detail polling treats every non-`syncing` response as terminal and can report an empty/null/unknown status as confirmed failure. Claude may repair within the existing R2 objective and effect boundary after rehydrating and recording `REVIEWER (Codex) -> REPAIR_WORKER (Claude)` in the active handoff.

Allowed implementation is `frontend/src/views/Channels/ChannelDetail.vue` and focused `frontend/src/__tests__/channels-sync-status.spec.ts` only. Correct the terminal-status predicate: `success`, `partial`, `error` are observed terminal outcomes; empty/null/`never`/unknown after an accepted request yield the existing unconfirmed warning and stop polling without a false failure. Add deterministic mounted cases for empty and unknown (and null if the fixture supports it); prove the test fails if the broad `!== 'syncing'` shortcut is restored. Preserve timeout, refresh-failure, rejected-start, observed terminal, shared chip, timestamp and reauth behavior. BUILD evidence may be amended; synchronize state/handoff/memory/status and the UI roadmap as needed. Correct the BUILD evidence's final changed set to match the actual commit (it did not include `IMPLEMENTATION_STATUS.json`).

Rerun focused/full frontend tests, forced typecheck, build, catalog `-Check`, doctor and diff check. Make one local repair/evidence commit and return `REVIEW_PENDING` for Codex independent re-review. No backend/API/store/router/shared-chip/i18n change, provider/channel call, persistent Compose DB, push, deployment, self-PASS or FREEZE. If the fix requires a path beyond this boundary, return `BUILD_BLOCKED` with the reason.
