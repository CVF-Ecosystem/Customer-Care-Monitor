# Work order CCMAI-UX-015 — Activity logs, cost logs, notification history and MCP connections (frontend only)

**State:** `BUILD` authorized under this work order; returns `REVIEW_PENDING` · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** owner direction 2026-09-28 (UI series UX-012..015) and [SPEC](../specs/LOGS_COST_NOTIFY_MCP_UX015_2026-09-28.md) with canvas version `1790608303-0cb6`.

## Role route

`ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR → IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` (Claude), each acknowledged in the active handoff. One local commit, `REVIEW_PENDING`; no self-PASS, FREEZE or push.

## Allowed paths (exhaustive)

- `frontend/src/views/ActivityLogs.vue`, `CostLogs.vue`, `NotificationLogs.vue`, `MCPConnections.vue`;
- `frontend/src/i18n/vi.ts`, `en.ts` — additive `lg_*` / `mc_*` keys only;
- new `frontend/src/__tests__/logs-mcp.spec.ts`;
- evidence `docs/reviews/LOGS_COST_NOTIFY_MCP_UX015_BUILD_2026-09-28.md`, `docs/reviews/assets/ux-015-2026-09-28/**`;
- this work order, SPEC status, roadmap row, continuity files.

Not allowed: layouts, shared components (imported only), stores, router, API module, `utils/format.ts`, backend, permission values, repository scripts, CVF core. Evidence must not display a real MCP secret or credential; any MCP client created for captures lives only in the disposable environment.

## Requirements, evidence, ceiling

SPEC §2, §4 and §6. Local source/tests/docs and a disposable screenshot environment (removed afterwards); synthetic data only; no provider call, no notification send, no persistent `ccma` change, deployment, push or FREEZE. A gate that cannot pass within scope → `BUILD_BLOCKED`.

## Independent review disposition (Codex, 2026-09-28)

[UX-015 review](../reviews/CCMAI_UX_015_INDEPENDENT_REVIEW_2026-09-28.md): `REVIEW_PASS`, FREEZE open. The filtered cost total, activity-detail language, notification permission and per-company MCP scope remain separate contracts; this review does not authorize those changes.
