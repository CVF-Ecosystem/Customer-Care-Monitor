# Work order CCMAI-RUNTIME-009 — job-dispatch configuration admission

**State:** `WORK_ORDER_READY / BUILD_PENDING` · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER` → `SESSION_SYNC_STEWARD` → `COMMIT_STEWARD`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** owner “next”, [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [SPEC](../specs/RUNTIME_JOB_CONFIG_ADMISSION_S1_2026-09-28.md), and R008 independent REVIEW PASS.

## Entry and role route

This tranche inherits the reviewed R008 dispatch-admission pattern and enters BUILD after this SPEC/WORK_ORDER. Claude must rehydrate current CVF continuity, declare context and acknowledge `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff before editing source. Return one local BUILD/evidence commit as `REVIEW_PENDING` for independent Codex R2 REVIEW. No self-approval or FREEZE.

## Objective and allowed scope

Admit `POST /jobs/:jobId/trigger` and `POST /jobs/:jobId/test-run` only after successful configuration validation, and pass that same configuration to their workers.

Allowed implementation paths: `backend/api/handlers/jobs.go` and focused test file(s) under `backend/api/handlers/`. A small private loader/launcher seam in `jobs.go` is allowed for deterministic tests. No edit to `backend/config`, `backend/engine`, models/migrations, channels, agents, adapters, scheduler, provider clients, permissions, frontend, notifications or CVF core. If another path or effect is necessary, return a bounded change request before editing.

Allowed accompanying paths: new `docs/reviews/RUNTIME_JOB_CONFIG_ADMISSION_S1_BUILD_2026-09-28.md`, this work order, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, and the S1 roadmap status line. Keep unrelated worktree files and generated `knowledge/_index.json` out of the commit.

## BUILD and evidence

1. Trace tenant-scoped lookup, both response paths, goroutine launch, config load, analyzer construction and cancel registration. Keep lookup before config validation. If config load fails or yields nil, return generic `500 {"error":"job_start_failed"}`; do not launch a goroutine, create a run, register cancellation, log raw validation detail or expose secrets.
2. Load configuration once per accepted request and pass the same pointer into the worker; remove both ignored in-worker loads. Preserve existing 202 bodies, test-run limit 3, trigger mode/date/limit behavior, timeout, cancellation and analyzer call. Do not alter job execution outcomes.
3. Use synthetic fixtures and disposable MySQL. Test both endpoints for forced error, nil config, wrong tenant, valid config identity and launch count. Prove no run/cancel registration occurs on rejection. Exercise actual `config.Load` with synthetic environment values. Stub launchers so tests never call a real adapter or provider. Restore all package-global seams and do not use `t.Parallel` with them.
4. Run focused and full backend tests, `go build ./...`, `go vet ./...`, catalog `-Check`, workspace doctor and `git diff --check`. Record exact commands/results, cleanup, any untested branch and no-provider boundary in BUILD evidence. If a test fails, return `BUILD_BLOCKED` with evidence.

## Exit and boundary

After checks pass, route `IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD`, synchronize continuity/status, and create one local commit without push. Return `REVIEW_PENDING` to Codex. The 202 response means dispatch accepted, not analysis completed.

**External-effect ceiling:** local source/docs/tests and disposable MySQL only. No real AI provider, real channel sync, credential use, customer data, persistent Compose DB change, deployment, push, S1 closure, S2/S3/S5 implementation or FREEZE.
