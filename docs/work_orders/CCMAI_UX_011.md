# Work order CCMAI-UX-011 — Results screen redesign (frontend only)

**State:** `BUILD` authorized under this work order; returns `REVIEW_PENDING` · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** [INTAKE](../specs/RESULTS_SCREEN_UX011_INTAKE_2026-09-28.md) (Codex, ORCHESTRATOR) and [SPEC](../specs/RESULTS_SCREEN_UX011_2026-09-28.md) with canvas version `1790597255-fb44`. Dependencies UX-000, UX-002, UX-001a and UX-010 are REVIEW PASS.

## Entry and role route

`ORCHESTRATOR (Codex) → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` (Claude). Each transition is acknowledged in the active handoff before acting. One local BUILD/evidence/continuity commit, `REVIEW_PENDING`; no self-PASS, FREEZE, push or deployment.

## Allowed paths (exhaustive)

- `frontend/src/views/Results.vue` — template/script/style rewrite within SPEC §2;
- `frontend/src/components/ui/SourceStatusPanel.vue` — one additive optional prop `scope?: string` (rendered after the title only when set); no other shared-component change unless strictly additive and backward compatible, recorded in the evidence;
- `frontend/src/i18n/vi.ts`, `frontend/src/i18n/en.ts` — additive `results_*` keys only; existing values unchanged;
- `frontend/src/__tests__/results-*.spec.ts` (new) and `frontend/src/__tests__/results-source-note.spec.ts` (selector updates only if the structure moves; its three assertions keep their meaning); `ui-components.spec.ts` for the new prop;
- evidence `docs/reviews/RESULTS_SCREEN_UX011_BUILD_2026-09-28.md` and `docs/reviews/assets/ux-011-2026-09-28/**`;
- this work order, the SPEC status line, the UI roadmap status row, `CVF_SESSION/**`, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, `docs/catalog/**`, `docs/INDEX.md`.

Not allowed: `backend/**`, API contracts, `frontend/src/api/**`, `frontend/src/stores/**`, `frontend/src/composables/**`, `frontend/src/utils/**`, other views (Dashboard, Channels, Messages, Jobs, Job Detail), `scripts/**` in the repository (ad-hoc capture helpers live in the scratchpad), database, CVF core, provider calls. A need outside these paths becomes a `BLOCKED_API_CONTRACT` or scope note, not an edit.

## BUILD requirements

1. SPEC §2 items 1–11 and invariants §4.
2. Request parameters of `GET /results` and `GET /results/export` are byte-for-byte the same function of the filter state as today (`job_type`, `verdict`, `date_field`, `sort`, `q`, `job_ids`, `channel_ids`, `tags`, `from`, `to`, `score_min`, `score_max`, `page`, `page_size=25`, `format`). Debounce, page reset and tab reset rules unchanged.
3. Source panel counts use only the rows of the current page and say so; no global source number anywhere.
4. No confidence on this screen; no "Cần xem lại" filter; no quote highlighting (SPEC §5).
5. The transcript moves from the card expansion into the dialog using the existing `useChatTranscript` calls (same endpoint, loaded when the dialog opens).
6. Colors via theme tokens / UX-000 components; no new hard-coded hex in the view.

## Evidence

`npx vue-tsc -b --force`, `npm run build`, full `npm test` with the focused tests of SPEC §7 (UI-only mocks; no governance claim); i18n parity; disposable `scripts/ui-screenshots.ps1 -Mode app` capture of `/results` desktop/mobile × light/dark with 0 JS errors/overflow/external requests, plus dialog and classification captures where the disposable environment allows; `git diff --check`; catalog `-Check`; workspace doctor. A gate that cannot pass within scope → `BUILD_BLOCKED` with the reason.

## External-effect ceiling

Local source/tests/docs and the disposable screenshot environment (removed afterwards). Synthetic disposable data only. No provider call, real channel sync, customer data, persistent `ccma` change, deployment, push or FREEZE.
