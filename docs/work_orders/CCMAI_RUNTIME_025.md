# CCMAI-RUNTIME-025 — F03 analyzer incremental coverage

Status: REVIEW_PENDING. Issued 2026-10-01 by Codex (ORCHESTRATOR → SPEC_AUTHOR → WORK_ORDER_AUTHOR). Risk ceiling R2. Dispatcher seed/base `517406f1bfafbfe48b2cef31b558a76b8e67d8c0`.
Authority: [SPEC](../specs/RUNTIME_ANALYZER_INCREMENTAL_COVERAGE_F03_2026-10-01.md), [roadmap](../roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md), `CVF_SESSION/authority/CCMAI-RUNTIME-025.json`. R024 REVIEW_PASS / FREEZE_OPEN; F02 remains OPEN for live/message claims. F03 is not repaired by planning.

## Assignment and entry gate

Claude is IMPLEMENTATION_WORKER and COMMIT_STEWARD for one local BUILD commit, returned REVIEW_PENDING; Codex independently reviews. Rehydrate manifest/policy, bootstrap or active state, memory/current prose, handoff, status/index, SPEC/order, immutable seed and tranche record. Run doctor and acknowledge `WORK_ORDER_AUTHOR (Codex) → IMPLEMENTATION_WORKER (Claude)` in handoff before BUILD. Read R002/R004 snapshot/provenance and current analyzer reanalysis contracts. Seed is read-only. Existing untracked `knowledge/_index.json` stays outside commit.

## Authorized scope

- `backend/engine/analyzer.go`, optional focused `analyzer_incremental.go`, `analyzer*_test.go`, `reanalyze_test.go`: explicit ordinary-unlimited mode, shared source-version eligibility using existing snapshots, full-local-message preparation for that mode, scan-start checkpoint and checked ordinary terminal persistence. No snapshot-builder/schema or explicit-mode redesign.
- `backend/engine/scheduler.go`, `scheduler*_test.go`: narrow private test seam/entry-path proof; preserve real cron/after-sync lookup/routing/lifecycle behavior. No scheduling/cancellation registry/single-flight redesign.
- SPEC/order/roadmap, one BUILD evidence in `docs/reviews/`, tranche/status/continuity and catalog/index as needed. Other product paths, dispatcher seed, .cvf, workflows and CVF core are read-only.

## Build and proof sequence

1. Trace source at planning HEAD `1e74d18`: candidate event-time predicate, result-time suppression, snapshot sent-time window, single/batch preparation, evaluation/snapshot commit and job finalization, plus both scheduler entry paths. Confirm why merely moving last_run_at to start leaves late timestamps and already analyzed conversations unhandled.
2. Implement SPEC for ordinary unlimited mode only. Compare the exact full snapshot sent/saved against the latest tenant/job-bound evaluation receipt; unchanged skips, changed/new/legacy schedules, unverifiable links/DB errors fail. Check final run/checkpoint persistence. Keep F05/F06 behavior outside this contract and document their residuals. If schema/snapshot format or API changes are needed, stop BUILD_BLOCKED; do not widen authority.
3. Add SPEC's deterministic two/three-run DB proofs in single/batch plus real cron/after-sync paths. Assert transcripts, message/evaluation IDs, snapshots, calls and checkpoints; test a backdated late insert, late message/edit without newer last_message_at, unchanged replay, source mutation after preparation and tenant/job isolation. Show old-source failure and meaningful mutations, including checkpoint write failures. Restore all probes. Synthetic provider proves local scheduling/bookkeeping only; no governance claim.
4. Run focused analyzer/snapshot/reanalysis/scheduler regressions, full backend on disposable MySQL (`TEST_DB_DSN`, `go test ./... -json -count=1 -p 1`), R019 five-sentinel gate and `go build ./...`; record commands/counts/optional skips and cleanup. Run catalog/doctor/diff checks and preflight over every intended changed path. Default worktree preflight is not PASS while unrelated `knowledge/_index.json` remains; disclose it and use explicit `--files`, never omit a BUILD file. Set `PYTHONDONTWRITEBYTECODE=1` for gate tests. No persistent Compose DB, real credentials or real provider/channel calls.
5. Synchronize **current prose as well as machine fields** to REVIEW_PENDING in the BUILD commit: memory front paragraph/marker, state, handoff header/result, status phase/limitations, work-order Status, tranche and roadmap. Check them together before commit; prior R024 drift passed the marker gate. Return exact SHA, changed files, evidence path, uncovered assumptions and REVIEW_PENDING. Do not self-approve or push. Codex may repair a small same-scope defect locally under the CVF rule.

## Evidence and effects

Use `docs/reviews/RUNTIME_ANALYZER_INCREMENTAL_COVERAGE_F03_BUILD_2026-10-01.md`. Ordinary runs will consider historical never-analyzed local conversations and prepare full source instead of only a last-run message suffix; record that behavior explicitly. Alibaba authorization persists, but R025 needs **zero** real calls and grants none. Test doubles assert no CVF AI governance/provider-quality behavior. No notification delivery, push, merge, deployment, F05/F06 redesign, parent-CVF work or FREEZE. PR #1 stays draft at older remote head `3e0b37e`.
