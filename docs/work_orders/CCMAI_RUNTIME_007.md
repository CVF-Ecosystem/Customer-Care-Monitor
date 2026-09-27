# Work order CCMAI-RUNTIME-007 — manual sync start truth

**State:** `WORK_ORDER_READY / BUILD_PENDING` · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** owner “next”, [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [SPEC](../specs/RUNTIME_MANUAL_SYNC_START_TRUTH_S1_2026-09-27.md), and R001–R006 accepted review evidence.

## Entry and role route

This tranche inherits the reviewed S1 sync checkpoint and status semantics and enters BUILD after this SPEC/WORK_ORDER. Claude must rehydrate manifest, policy, continuity, handoff, implementation truth and docs index; declare CVF context; and record `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` in the active handoff before BUILD. Return one local evidence commit as `REVIEW_PENDING` for independent Codex R2 REVIEW. No self-approval or FREEZE.

## Objective and allowed scope

Make `POST /channels/:channelId/sync` acknowledge a manual sync only after its `syncing` state is persisted, and make panic status handling tenant-scoped and observable.

Allowed implementation paths: `backend/api/handlers/channels.go` and focused test file(s) under `backend/api/handlers/` only. A small private helper in `channels.go` is allowed for deterministic testing. No `backend/engine`, model/migration, adapter, scheduler, agent, provider, permission, frontend, notification or CVF core changes. If reliable no-worker-launch testing or the acceptance contract requires another path, return a bounded change request before editing it.

Allowed accompanying paths: new `docs/reviews/RUNTIME_MANUAL_SYNC_START_TRUTH_S1_BUILD_2026-09-27.md`, this work order, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json` and roadmap status line. Keep unrelated `.gitignore`, `docs/references/` and generated `knowledge/_index.json` out of the commit.

## BUILD and evidence

1. Trace the tenant-scoped lookup, start-state update, goroutine launch and panic recovery. Preserve the existing 202 body and asynchronous completion semantics. Check write error and affected-row count with the tenant-and-channel predicate; start no goroutine on failure. Return a generic non-2xx response without DB or credential detail.
2. Make panic recovery write tenant scoped, check its result, and use a bounded safe status message. Keep failure logging useful without printing raw panic content or credentials. Do not change normal `SyncEngine` behavior.
3. Use synthetic fixtures and a disposable MySQL `CCMA` database. Force a DB update error and a zero-row outcome; prove neither produces 202 nor launches a worker. Prove successful start-state persistence before 202. Test recovery helper with another tenant/channel and an injected write failure. Tests must not invoke a real adapter or provider, and must not mistake mocked output for CVF governance proof.
4. Run focused and full backend tests against disposable MySQL, `go build ./...`, `go vet ./...`, catalog `-Check`, workspace doctor and `git diff --check`. Record exact commands/results, cleanup, pre-existing limitations and any untested branch in BUILD evidence.

## Exit and boundary

After passing checks, route `IMPLEMENTATION_WORKER -> SESSION_SYNC_STEWARD -> COMMIT_STEWARD`, synchronize continuity/status and create one local commit without push. Return `REVIEW_PENDING` to Codex. On failure or out-of-scope need, return `BUILD_BLOCKED` with evidence.

**External-effect ceiling:** local source/docs/tests and disposable MySQL only. No real channel sync, credential use, customer data, provider API, persistent Compose DB change, deployment, push, S1 closure, S2/S3/S5 implementation or FREEZE.
