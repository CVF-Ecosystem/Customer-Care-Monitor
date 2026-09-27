# Work order CCMAI-RUNTIME-004 — S1 result source freshness

**State:** `REVIEW_PENDING` after repair round 1 (R004-R1/R004-R2/R004-R3 complete, local commit made) · **Risk:** R2 · **Assignee:** Claude (`REPAIR_WORKER` -> `COMMIT_STEWARD`) · **Independent reviewer:** Codex (`REVIEWER`, re-review of repair round 1 next) · **Authority:** owner “next”, `docs/roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md`, `docs/specs/RUNTIME_RESULT_SOURCE_FRESHNESS_S1_2026-09-27.md`, and `docs/reviews/CCMAI_RUNTIME_004_INDEPENDENT_REVIEW_2026-09-27.md`.

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

## Repair round 1 — R004-R1/R004-R2/R004-R3

**Entry:** Codex independent REVIEW of BUILD commit `a9559e3` found the three same-scope findings at `docs/reviews/CCMAI_RUNTIME_004_INDEPENDENT_REVIEW_2026-09-27.md`. Claude must rehydrate current continuity and append `REVIEWER (Codex) -> REPAIR_WORKER (Claude)` acknowledgment to the active handoff before repair. This round keeps the existing R2/local-disposable-MySQL/no-provider/no-push boundary; no new product authority is granted.

Allowed implementation files: `backend/engine/snapshot.go`, `backend/engine/source_integrity_test.go`, `backend/api/handlers/results.go`, and `backend/api/handlers/results_test.go` only. Allowed accompanying files: existing R004 BUILD evidence, active handoff/state, session memory, implementation status, and roadmap/spec status if source truth changes. Do not change DB models/migrations, frontend, provider, analyzer, adapters, sync dispatch, job-specific/conversation APIs, notifications or CVF core.

1. Validate linked snapshot provenance and integrity before reporting `bound_currentness_unverified`: exact manifest SHA-256 digest, supported schema, manifest/row/result tenant and conversation, run link, and manifest message count. Invalid row-local evidence returns `verification_unavailable`. Replace the valid-test fixture's `deadbeef` digest and add corrupt-digest, wrong conversation/run and cross-tenant regressions.
2. Correct the added-message comparison for messages earlier than the manifest's earliest `sent_at` when `omitted_earlier_messages=0`; for windowed history, compare the recorded omitted count with the current earlier-message count and flag proven count changes. Where equal counts still leave earlier identities unknowable, retain the explicitly limited `bound_currentness_unverified` status. Add zero/nonzero omitted-count regressions. Preserve the no-positive-freshness claim.
3. Execute handler-level `ListResults` and `ExportResults` tests, including CSV and XLSX content/status equivalence, tenant isolation for the new data reads, and observable failures from snapshot/message batch-query errors. Verify no partial success and preserve pagination, filters, count, sort and export cap.
4. Re-run focused tests and `go test ./... -count=1` on disposable MySQL `CCMA` (set `log_bin_trust_function_creators=1` for existing trigger tests), `go build ./...`, `go vet ./...`, catalog `-Check`, workspace doctor and `git diff --check`. Append exact commands/results, fixture cleanup and any remaining limits to the R004 BUILD evidence. Frontend build is needed only if its source changes (which is outside this repair scope).

After passing checks, synchronize continuity/status, create one local commit without push, and return `REVIEW_PENDING` to Codex. If a requirement needs a path or effect outside this repair class, or a required check fails, report the boundary/failure instead of claiming completion. No self-approval or FREEZE.

## Repair round 2 — R004-R3-T1 test/evidence completion

**Entry:** Codex independent re-review of repair commit `a074870` accepted R004-R1 and R004-R2 but found the original R004-R3 endpoint evidence incomplete at `docs/reviews/CCMAI_RUNTIME_004_REPAIR_R1_REREVIEW_2026-09-27.md`. Claude must rehydrate current continuity and append `REVIEWER (Codex) -> REPAIR_WORKER (Claude)` acknowledgment to the active handoff before repair. This is the same R2 objective and external-effect class, repair round 2; no review-cost escalation is due.

Allowed implementation file: `backend/api/handlers/results_test.go` only. Allowed accompanying files: existing R004 BUILD evidence, active handoff/state, session memory and implementation status. Do not change production source, DB models/migrations, frontend, provider, adapter, analyzer, scheduler or CVF core. If a product-source change becomes necessary, stop and return a bounded change request to Codex.

1. Extend the actual `ExportResults` CSV and XLSX handler test so it checks both `bound_currentness_unverified` and `changed_since_analysis` after a source edit. Assert the status label in the exported data for each format; retain the existing page and export cap/filter tests.
2. Exercise `ExportResults` with a forced snapshot/message batch-query failure and assert an observable non-2xx error, no success download header and no partial file body. Keep `TestListResultsSnapshotBatchQueryFailureIsObservable` passing. Restore any renamed disposable table in cleanup even on failure.
3. Run the focused result tests and `go test ./... -count=1 -p 1` on disposable MySQL `CCMA` with `log_bin_trust_function_creators=1`, then catalog `-Check`, workspace doctor and `git diff --check`. Append exact commands/results and cleanup to `docs/reviews/RUNTIME_RESULT_SOURCE_FRESHNESS_S1_BUILD_2026-09-27.md`. No frontend build is needed because frontend source is outside this repair scope.

After passing checks, synchronize continuity/status, create one local commit without push, and return `REVIEW_PENDING` to Codex. No self-approval or FREEZE.
