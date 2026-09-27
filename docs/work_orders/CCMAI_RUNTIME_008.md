# Work order CCMAI-RUNTIME-008 — manual sync configuration admission

**State:** `REVIEW_PENDING` after BUILD (local commit; no path addition) · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER` -> `COMMIT_STEWARD`) · **Independent reviewer:** Codex (`REVIEWER`, next) · **Authority:** owner “tiếp”, [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [SPEC](../specs/RUNTIME_MANUAL_SYNC_CONFIG_ADMISSION_S1_2026-09-27.md), and accepted R007 review.

## Entry and role route

This tranche inherits R007's checked manual start-state write and enters BUILD after this SPEC/WORK_ORDER. Claude must rehydrate current CVF continuity, declare context and acknowledge `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` in the active handoff before BUILD. Return one local BUILD/evidence commit as `REVIEW_PENDING` for independent Codex R2 REVIEW. No self-approval or FREEZE.

## Objective and allowed scope

Make successful configuration validation a prerequisite for manual sync `syncing` persistence and HTTP 202, then pass that validated config to the worker without a second load.

Allowed implementation paths: `backend/api/handlers/channels.go`, `backend/api/handlers/sync_start_test.go`, and one additional focused test file under `backend/api/handlers/` if needed. A small private config-loader seam is allowed for deterministic tests; restore any changed package-global seam in test cleanup and do not use `t.Parallel`. No edit to `backend/config`, `backend/engine`, models/migrations, adapters, scheduler, agents, provider, permissions, frontend, notifications or CVF core. If a new path or effect is necessary, return a bounded change request before editing.

Allowed accompanying paths: new `docs/reviews/RUNTIME_MANUAL_SYNC_CONFIG_ADMISSION_S1_BUILD_2026-09-27.md`, this work order, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json` and roadmap status line. Exclude unrelated `.gitignore`, `docs/references/` and generated `knowledge/_index.json`.

## BUILD and evidence

1. Trace manual lookup, R007 start write, launcher, worker config load, and `config.Load` validation. Keep wrong-tenant 404 ahead of config validation. On validation failure, respond with a generic non-2xx error; make no status/checkpoint/error write and launch no worker. Do not put raw config errors or secrets in the API body or logs.
2. Pass the validated `*config.Config` through the launcher into `runManualSync`; remove its second load. Keep its timeout, `SyncEngine` call, R007 panic recovery, and response shape. Preserve all R007 error/zero-row behavior.
3. Add focused tests for forced config failure, real `config.Load` validation with synthetic environment, passed-config identity and R007 regressions. Use a stubbed launcher, synthetic channel rows and disposable MySQL; no adapter/provider call or customer data. Prove a config failure leaves the previous status visible and cannot return 202.
4. Run focused and full backend tests on disposable MySQL, `go build ./...`, `go vet ./...`, catalog `-Check`, workspace doctor and `git diff --check`. Record exact commands/results, temporary-resource cleanup, no-provider boundary and untested branches in BUILD evidence.

## Exit and boundary

After checks pass, route `IMPLEMENTATION_WORKER -> SESSION_SYNC_STEWARD -> COMMIT_STEWARD`, synchronize continuity/status and create one local commit without push. Return `REVIEW_PENDING` to Codex. On failed checks or out-of-scope need, return `BUILD_BLOCKED` with evidence.

**External-effect ceiling:** local source/docs/tests and disposable MySQL only. No real channel sync, real credential/provider use, customer data, persistent Compose DB change, deployment, push, S1 closure, S2/S3/S5 implementation or FREEZE.
