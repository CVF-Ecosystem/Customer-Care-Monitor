# CCMAI-RUNTIME-012 — OAuth credential persistence truth

**Date:** 2026-09-29 · **State:** REVIEW_PASS / FREEZE_OPEN after [independent review](../reviews/CCMAI_RUNTIME_012_INDEPENDENT_REVIEW_2026-09-29.md) · **Risk ceiling:** R2 · **Authority:** [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [SPEC](../specs/RUNTIME_OAUTH_CREDENTIAL_PERSISTENCE_S1_2026-09-29.md), and R011 independent REVIEW PASS.

## Route and independence

Codex: `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR`. Claude: `IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD` for one local BUILD/evidence commit. Codex independently reviews R2 BUILD. Before source edit, Claude rehydrates CVF state/handoff and records `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff. Prior R001–R011 remain REVIEW PASS / FREEZE open; S1 is IN_PROGRESS.

## Objective and allowed scope

Stop Zalo and Facebook OAuth callbacks from reporting success unless the final tenant-scoped credential update succeeds on exactly one row. Allowed source: `backend/api/handlers/channels.go` only. Allowed tests: focused files under `backend/api/handlers/`, including extension of `channel_config_admission_test.go` if useful. A narrow private persistence/test seam in `channels.go` is allowed only if deterministic MySQL error and zero-row tests cannot be written with existing seams. Allowed records: this work order, SPEC, new BUILD evidence under `docs/reviews/`, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json` and the S1 roadmap status line.

Do not edit config, engine, adapters, scheduler, agents, jobs, router, models/migrations, frontend, provider clients, CVF core or unrelated worktree files. If a new path or external effect is necessary, return a bounded change request before editing.

## BUILD requirements

1. Trace both callbacks from verified state and tenant-scoped channel lookup through exchange, encryption, final update and redirect. Keep the existing preceding behavior. Change the final `Updates` predicate to `id AND tenant_id`; check `.Error` and `RowsAffected == 1`. On either failure, log only a bounded failure class and return the existing generic tenant-scoped `Authorization failed` redirect. Never report success for 0 rows or a write error; never print SQL detail, token, credential, code or state.
2. Keep the successful Zalo/Facebook 302 Location and update field sets unchanged. Do not add a retry, external compensation, token refresh, new status field, transaction spanning network, or stronger claim that the external exchange is reversible.
3. Use only disposable MySQL and synthetic OAuth states, credentials and recorded HTTP replies. For each callback, force an update error and a zero-row update *after* token exchange; assert exact generic failure Location, no success, no token persistence, tenant isolation and no secret/DB detail in response/log. The zero-row case must catch ID-only writes, for example by reassigning the channel to another fixture tenant after its authorized lookup through a controlled transport hook. Assert expected outbound calls occurred before persistence. Run accepted callbacks and verify exactly one row receives the correct synthetic tokens. Clean triggers/hooks and restore package-global transport/seams. No real endpoint call.
4. Run focused and full backend tests, `go build ./...`, `go vet ./...`, catalog `-Check`, workspace doctor and `git diff --check`. Record exact commands/results, disposal of test containers, claim limits and any untested branch. A failed gate returns `BUILD_BLOCKED` rather than REVIEW_PENDING.

## Exit and effect boundary

Synchronize continuity/status and create one **local** BUILD/evidence commit, without push. Return `REVIEW_PENDING` to Codex. No saved Alibaba key, real provider/channel call, customer data, persistent Compose DB mutation, deployment, S1 closure or FREEZE is authorized. Synthetic tests prove callback persistence behavior only, not live CVF governance.
