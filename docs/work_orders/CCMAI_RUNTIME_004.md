# Work order CCMAI-RUNTIME-004 — S1 result source freshness

**State:** `READY_FOR_ASSIGNEE_ACK` · **Risk:** R2 · **Assignee:** Claude (`IMPLEMENTATION_WORKER`) · **Independent reviewer:** Codex (`REVIEWER`) · **Authority:** owner “next”, `docs/roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md`, and `docs/specs/RUNTIME_RESULT_SOURCE_FRESHNESS_S1_2026-09-27.md`.

## Entry and role route

This tranche inherits the accepted S1 snapshot and replay evidence from `CCMAI-RUNTIME-002/003`, enters at BUILD after this SPEC/WORK_ORDER, and does not close either prior FREEZE. Claude must rehydrate manifest, policy, active state, memory, handoff, implementation status and docs index; declare the CVF context; then append `WORK_ORDER_AUTHOR (Codex) -> IMPLEMENTATION_WORKER (Claude)` acknowledgment to the active handoff before BUILD. Return one local evidence commit for independent Codex R2 REVIEW; no self-approval or FREEZE.

## Objective and scope

On the aggregate Results page and its CSV/XLSX export, distinguish a result whose stored snapshot conflicts with current local source from a result merely bound to a snapshot. Keep status honest when verification is unavailable and preserve tenant isolation, paging and read-only behavior.

Allowed implementation paths: `backend/engine/snapshot.go` plus focused engine tests if a shared canonical comparison helper is needed; `backend/api/handlers/results.go` and `backend/api/handlers/results_test.go`; `frontend/src/views/Results.vue`; `frontend/src/i18n/vi.ts`, `frontend/src/i18n/en.ts`; only the existing frontend test/config file directly needed to verify those labels. A small new helper or focused test file in those same backend directories is allowed. Do not change DB models/migrations, analyzer/provider/adapter/sync dispatch, job-specific or conversation APIs, notification code, scheduler, CVF core or other projects. If a path outside this class is necessary, stop and return a bounded change request.

Allowed evidence/continuity paths: new `docs/reviews/RUNTIME_RESULT_SOURCE_FRESHNESS_S1_BUILD_2026-09-27.md`, active handoff/state, `CVF_SESSION_MEMORY.md`, `IMPLEMENTATION_STATUS.json`, and roadmap/spec/work-order status lines. Keep existing unrelated `.gitignore` and `docs/references/` worktree entries outside the commit.

## Required implementation and evidence

1. Audit the actual Results read/export query and `ccma.snapshot.v1` fields before editing. Reuse the snapshot's canonical digest-relevant message representation; do not duplicate an inconsistent attachment fingerprint. Define how `omitted_earlier_messages` and newly added/removed rows affect the comparison. Use `bound_currentness_unverified` rather than a positive freshness claim when the local comparison cannot prove upstream completeness.
2. Return `source_integrity_status` for each paged aggregate result and bounded export row with the four SPEC values. Batch tenant-scoped snapshot/message reads; handle corrupt/missing snapshots and DB errors without false `unchanged`. Do not mutate result or source rows.
3. Show the status on Results table, card and detail dialog with Vietnamese/English strings, including an explicit explanation of the local-only comparison. Put the same semantic field in CSV and XLSX. Preserve existing RBAC, filters, count, sort, pagination and export cap.
4. Add disposable-MySQL tests for unchanged, edited content/role/attachment, missing/new message, legacy, corrupt/missing snapshot, tenant isolation and export equivalence. Seed synthetic rows directly; no mock provider output or real AI/provider call. Prove the key stale case fails against pre-change behavior. Test page and export error handling.
5. Run focused tests, full backend tests/build/vet, frontend build, catalog `-Check`, workspace doctor and `git diff --check`. Record exact commands/results, status semantics, fixture/cleanup, source limitations and any untested branch in BUILD evidence. Do not present local MySQL or UI tests as live CVF governance proof.

## Exit and boundary

On success, Claude transitions `IMPLEMENTATION_WORKER -> SESSION_SYNC_STEWARD -> COMMIT_STEWARD`, synchronizes continuity/status, creates one local commit without push, and returns `REVIEW_PENDING` to Codex. If a required check fails, a source signal is ambiguous, or a new path/effect is needed, report `BUILD_BLOCKED` with exact evidence instead of claiming completion.

**External-effect ceiling:** local source/docs/tests and disposable MySQL `CCMA` only. No provider API or credential, real channel sync, customer data, persistent Compose database reset, deployment, push, S2/S3/S5 implementation or FREEZE.
