# Work order CCMAI-RUNTIME-006 — S1 job-result source integrity

**State:** `REVIEW_PASS / FREEZE_OPEN` after independent repair-round-1 re-review of `682d88b` · **Risk:** R2 · **Assignee:** Claude (`REPAIR_WORKER` -> `COMMIT_STEWARD`) · **Independent reviewer:** Codex (`REVIEWER`, [PASS](../reviews/CCMAI_RUNTIME_006_REPAIR_R1_REREVIEW_2026-09-27.md)) · **Authority:** owner “next”, [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), [SPEC](../specs/RUNTIME_JOB_RESULT_SOURCE_INTEGRITY_S1_2026-09-27.md), and accepted R004/R005 reviews.

## Entry and role route

This tranche inherits R004's independently reviewed status semantics and R005's confidence semantics. It enters BUILD after this SPEC/WORK_ORDER. Claude must rehydrate manifest, policy, active state, memory, handoff, implementation status and docs index; declare its CVF context; append `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` acknowledgment to the active handoff before BUILD; then return one local evidence commit as `REVIEW_PENDING` for independent Codex R2 REVIEW. No self-approval or FREEZE.

## Objective and allowed paths

Expose R004's local source-change signal on job-specific result lists, exports and Job Detail presentation, preserving tenant boundaries and R005 confidence semantics.

Allowed implementation paths: `backend/api/handlers/jobs.go`, `backend/api/handlers/results.go`, one small shared helper and focused tests in `backend/api/handlers/`; `frontend/src/views/Jobs/JobDetail.vue`, `frontend/src/stores/jobs.ts`, `frontend/src/i18n/vi.ts`, `frontend/src/i18n/en.ts`, and the directly relevant frontend test file if present. Reuse `backend/engine`'s existing verification/comparison functions without editing engine source. No DB model/migration, analyzer/provider/prompt, adapter/sync, notifications, scheduler, permissions, other frontend view or CVF core change. If a new path/effect is necessary, stop and return a bounded change request.

Allowed accompanying paths: new `docs/reviews/RUNTIME_JOB_RESULT_SOURCE_INTEGRITY_S1_BUILD_2026-09-27.md`, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, and roadmap/SPEC/work-order status lines. Keep unrelated `.gitignore` and `docs/references/` out of the commit.

## Required BUILD and evidence

1. Trace `ListJobResults`, `ListAllJobResults`, `ExportJobResults`, R004's `attachSourceIntegrity`, and `JobDetail.vue` grouping before edits. Record all grouping/export paths and the current-chat-at-export distinction. Extract one shared tenant-scoped batched evaluator; preserve R004's exact status/error behavior and its tests.
2. Add per-result status to both job JSON endpoints and distinct status representation to QC/classification CSV/XLSX. Check DB query errors before emitting any file bytes or download headers. Process large exports in bounded batches without one query per result; preserve existing selection, sort and other columns. A row-local corrupt/cross-tenant link is unavailable, never unchanged.
3. Show status in the job table, card and dialog with bilingual text and the local-only caveat; do not hide changed status when grouped results have mixed statuses. Preserve R005 confidence fields and existing RBAC. Do not persist or trigger any result/sync/notification change.
4. Add meaningful disposable-MySQL endpoint/export tests, including mixed snapshots, tenant isolation and forced query failure for both formats. Rerun R004 aggregate tests to catch shared-helper regressions. Verify visible grouping/labels by a focused frontend test or component assertion plus build. Use synthetic rows only; no provider call or customer data.
5. Run focused and full backend tests on disposable MySQL, `go build ./...`, `go vet ./...`, frontend build, catalog `-Check`, workspace doctor and `git diff --check`. Record exact commands/results, cleanup, query cost, claim limits and remaining untested branches in BUILD evidence.

## Exit and boundary

After passing checks, transition `IMPLEMENTATION_WORKER -> SESSION_SYNC_STEWARD -> COMMIT_STEWARD`, synchronize continuity/status and create one local commit without push. Return `REVIEW_PENDING` to Codex. If any required check fails or an out-of-scope change is necessary, return `BUILD_BLOCKED` with exact evidence.

**External-effect ceiling:** local source/docs/tests and disposable MySQL `CCMA` only. No provider API or credential, real channel sync, customer data, persistent Compose database change, deployment, push, S2/S3/S5 implementation, S1 closure or FREEZE.

## Repair round 1 — R006-R1 local-only caveat visibility

**Entry:** Codex independent REVIEW of `61a775a` accepted the backend/source status and export checks but found the local-only `results_source_note` appears only after opening the Job Detail dialog. The primary table/card view lacks the qualification required by the SPEC. Finding and evidence: `docs/reviews/CCMAI_RUNTIME_006_INDEPENDENT_REVIEW_2026-09-27.md`.

Claude must rehydrate current continuity and acknowledge `REVIEWER (Codex) -> REPAIR_WORKER (Claude)` in the active handoff before repair. Allowed implementation file: `frontend/src/views/Jobs/JobDetail.vue` only. Allowed accompanying files: existing R006 BUILD evidence, active state/handoff, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, and this work-order status line. Place the existing bilingual `results_source_note` visibly in the results tab above the table/card presentation when results are shown; preserve badges and the dialog note. No backend, status logic, i18n key, frontend store, notification, DB, provider or CVF core edit.

Verify placement by source inspection, run `npm run build`, the existing `i18n.spec.ts` test, catalog `-Check`, workspace doctor and `git diff --check`. Append exact commands/results and the no-provider boundary to R006 BUILD evidence. Then synchronize continuity/status, make one local commit without push and return `REVIEW_PENDING` for Codex re-review. If a new path/effect is needed or a check fails, report `BUILD_BLOCKED` with evidence. No self-approval or FREEZE.
