# Work order CCMAI-UX-012 — Messages screen redesign (frontend only)

**State:** `BUILD` authorized under this work order; returns `REVIEW_PENDING` · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** owner direction 2026-09-28 (UI series UX-012..015) and [SPEC](../specs/MESSAGES_SCREEN_UX012_2026-09-28.md) with canvas version `1790601057-3d67`.

## Role route

`ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` (Claude), each acknowledged in the active handoff. One local commit, `REVIEW_PENDING`; no self-PASS, FREEZE or push.

## Allowed paths (exhaustive)

- `frontend/src/views/Messages.vue`;
- `frontend/src/i18n/vi.ts`, `en.ts` — additive `msgs_*` keys only;
- new `frontend/src/__tests__/messages-screen.spec.ts`;
- evidence `docs/reviews/MESSAGES_SCREEN_UX012_BUILD_2026-09-28.md`, `docs/reviews/assets/ux-012-2026-09-28/**`;
- this work order, the SPEC status line, the roadmap row, continuity files (`CVF_SESSION/**`, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`).

Not allowed: shared components (`frontend/src/components/**`), stores, composables, utils, API module, other views, backend, repository scripts, CVF core, provider calls.

## Requirements

SPEC §2 and §4. Request parameters for list, evaluated map, evaluations, messages, page lookup and export stay the same. Held items in SPEC §5 are not built.

## Evidence and ceiling

As SPEC §6. Local source/tests/docs plus a disposable screenshot environment (removed afterwards); synthetic data; no provider call, real sync, customer data, persistent `ccma` change, deployment, push or FREEZE. A failing gate that cannot be fixed within scope → `BUILD_BLOCKED`.

## Independent review R1 (Codex, 2026-09-28)

`CHANGES_REQUIRED`: [finding R1-1](../reviews/CCMAI_UX_012_INDEPENDENT_REVIEW_2026-09-28.md). When `GET /conversations/evaluated` fails or is pending, the list must not display “Chưa phân tích” for conversations whose status is unknown. Keep the chip mapping after a successful response and add the two focused regression cases in the existing UX-012 test file. Repair stays within the paths above. Complete the concurrent UX-013 commit before editing shared i18n files; return `REVIEW_PENDING` for independent re-review.
