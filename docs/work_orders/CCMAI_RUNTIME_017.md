# CCMAI-RUNTIME-017 — additive Dashboard QC violation count

**Date:** 2026-09-29 · **State:** REVIEW_PASS / FREEZE_OPEN · **Risk ceiling:** R2 · **Authority:** [SPEC](../specs/RUNTIME_DASHBOARD_QC_VIOLATION_COUNT_S1_2026-09-29.md), [S1 roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [R016-R1 re-review](../reviews/CCMAI_RUNTIME_016_R1_INDEPENDENT_REREVIEW_2026-09-29.md), [R017 independent review](../reviews/CCMAI_RUNTIME_017_INDEPENDENT_REVIEW_2026-09-29.md).

## Route and scope

Codex: `ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR`. Claude: `IMPLEMENTATION_WORKER → SESSION_SYNC_STEWARD → COMMIT_STEWARD`, then one local `REVIEW_PENDING` commit for Codex's independent R2 review. Before editing source, rehydrate current CVF continuity and record `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in the active handoff.

Implement only the additive `qc_violation_count` field in `GET /dashboard` per the SPEC. The existing `issues` field must continue to count all job results in the same effective date interval. Allowed source: `backend/api/handlers/dashboard.go`; new/focused tests in `backend/api/handlers/`. Allowed records: this SPEC/work order, new BUILD evidence under `docs/reviews/`, active state/handoff, memory, `IMPLEMENTATION_STATUS.json`, S1 roadmap and catalog only if required.

No frontend, `frontend/src/api/`, stores, router, other handler or model changes; no schema/write change, provider/channel call, persistent Compose DB access, deployment, push or FREEZE. Do not touch CVF core or unrelated worktree files. If the bounded handler/test paths cannot meet the contract, return `BUILD_BLOCKED` with a precise reason instead of broadening scope.

## Acceptance and evidence

1. Query tenant-owned `job_results` with exact `result_type = 'qc_violation'` and the existing effective `created_at BETWEEN from AND to` interval. Return numeric zero when empty. Preserve all existing JSON fields and `issues` meaning. A new-query DB error returns generic HTTP 500 without SQL or internals.
2. Use disposable MySQL and call the actual handler. Test mixed result types, tenant isolation, two findings in one conversation, date boundaries, empty match, DB error, and both response fields. The accepted path must demonstrate nonzero output; a mutation removing the type or tenant predicate must fail focused tests. Do not use mock output as governance proof.
3. Run focused and full backend tests, `go build ./...`, `go vet ./...`, catalog `-Check`, workspace doctor and `git diff --check`. Record exact commands/results, MySQL cleanup, changed set, failure modes and claim limits in BUILD evidence. No real provider/channel call is needed because this is a database API metric, not a CVF governance claim.
4. Synchronize state, handoff, memory and implementation status; verify exact changed set; create one local BUILD/evidence commit. Return `REVIEW_PENDING` to Codex. Claude must not self-PASS or FREEZE.
