# CCMAI-UX-016-F1 BUILD evidence — Dashboard stale-response regression

**Date:** 2026-09-30 · **Phase:** BUILD → REVIEW_PENDING · **Risk:** R2 (inherited) · **Role:** IMPLEMENTATION_WORKER (Claude); independent REVIEWER: Codex
**Authority:** [work order](../work_orders/CCMAI_UX_016_F1.md), [UX-016 independent review F1](CCMAI_UX_016_INDEPENDENT_REVIEW_2026-09-29.md).
Synthetic API responses prove UI behavior only, not CVF governance. No provider/channel call.

## Change

`frontend/src/__tests__/dashboard-qc-card.spec.ts` only: new test *keeps the newer date selection when an older response resolves last*.

1. Mounts the real `Dashboard.vue` on a settled baseline (QC `4`), then swaps the `api.get` seam for deferred promises recorded with their `params`.
2. Two date selections (`2026-03-10`, `2026-03-11`) issue two unresolved `/dashboard` requests; the test asserts exactly two pending calls with distinct `from` params and a `—` card while loading.
3. Resolves the newer request first (`qc_violation_count = 7`, `issues = 13`): card shows `7`. Then resolves the older request (`qc_violation_count = 99`, `issues = 55`) and flushes: card still shows `7` and does not contain `99`.
4. No sleeps, no network; ordering is controlled by the recorded resolvers.

`Dashboard.vue` was not changed (verified by `git diff` after the mutation restore).

## Non-vacuity mutation (disposable, not committed)

Disposable edit: commented out `if (requestId !== dashboardRequestId) return` after the `api.get` await in `loadDashboard`.

- Focused run: **1 failed / 3 passed** — `AssertionError: expected '99' to be '7'` (Expected `"7"`, Received `"99"`).
- Source restored from a byte copy; `git diff -- frontend/src/views/Dashboard.vue` empty; focused rerun: 4 passed / 0 failed.

## Gates

| Command | Result |
|---|---|
| `npx vitest run src/__tests__/dashboard-qc-card.spec.ts` (frontend) | 4/4 PASS (after restore) |
| `npx vitest run` (frontend, full) | 17 files, 171 tests PASS |
| `npx vue-tsc -b --force` | exit 0 |
| `npm run build` | built successfully |
| `scripts/manage_cvf_downstream_catalog.ps1 -Check` | PASS |
| `check_cvf_workspace_agent_enforcement.ps1 -ProjectPath <root>` | PASS 25/25 |
| `git diff --check` | exit 0 (CRLF warnings only) |

## Final changed set

`frontend/src/__tests__/dashboard-qc-card.spec.ts`, this evidence file, `CVF_SESSION/handoffs/AGENT_HANDOFF_V1_2026-09-26.md`, `CVF_SESSION/ACTIVE_SESSION_STATE.json`, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`.

## Disposition

`REVIEW_PENDING` for Codex. Claude does not self-approve F1 and does not freeze UX-016/S1. Zalo/legacy/mixed-version recovery and the rest of S1 remain open. No push.
