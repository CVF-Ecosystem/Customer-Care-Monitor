# Independent re-review: CCMAI-RUNTIME-004 repair round 1

**Reviewer:** Codex (`REVIEWER`) · **Date:** 2026-09-27 · **Target:** local commit `a074870` · **Disposition:** `CHANGES_REQUIRED_ROUND_2` for R004-R3 test/evidence completion only. No FREEZE.

## Accepted repairs

- **R004-R1 accepted:** `VerifySnapshotProvenance` now checks the linked row's schema, tenant/conversation/run identity, exact manifest SHA-256 digest and message count before local source comparison. `attachSourceIntegrity` reports `verification_unavailable` on mismatch. Valid DB fixtures now use a real digest; focused unit and MySQL tests cover corrupt digest, wrong links and cross-tenant lookup.
- **R004-R2 accepted:** current messages before the manifest's earliest included time are counted against `omitted_earlier_messages`. A zero-omitted backfill and a nonzero-omitted count change now report `changed_since_analysis`; equal counts retain the explicitly limited `bound_currentness_unverified`. The new focused regressions exercise both cases.
- The repair stays within the four authorized source/test files plus evidence/continuity. No provider, channel, customer data, DB model/migration, frontend, deployment or CVF-core change was made.

## Independent verification

Fresh disposable MySQL 8 `CCMA`, validation-only credentials, `GOFLAGS=-mod=readonly`:

```text
go test ./engine ./api/handlers -run 'TestCompareSnapshot|TestVerifySnapshot|TestSourceIntegrity|TestListResults|TestExportResults' -count=1   PASS
go test ./... -count=1 -p 1                                                                                                           PASS (all tested backend packages)
```

The initial full-suite attempt on this new container failed in existing trigger tests because `log_bin_trust_function_creators` was still OFF. Setting it to ON resolved those failures; the complete rerun passed. The container was removed after verification. Workspace doctor passed 25/25. These local tests do not prove live CVF governance or upstream source completeness.

## Remaining blocking item: R004-R3-T1

The R004 repair addendum specifically requires actual CSV **and** XLSX exports for both a changed and a nonchanged result, and an observable failure from both the page and export handlers when a batched snapshot/message query fails. New `TestExportResultsCSVAndXLSXIncludeSourceIntegrityColumn` (`backend/api/handlers/results_test.go`, around line 730) checks only `bound_currentness_unverified`; no changed-source export is exercised. New `TestListResultsSnapshotBatchQueryFailureIsObservable` (around line 789) calls `ListResults` only; no forced failure reaches `ExportResults`. The shared `fetchRows` path and readable production branches support the intended behavior, but the required endpoint evidence is still absent. The remaining work is a test/evidence completion within the same root cause, not a new source defect.

**Repair acceptance:** In `backend/api/handlers/results_test.go`, exercise `ExportResults` to CSV and XLSX after a source edit and assert the `changed_since_analysis` label in the exported data, alongside the existing nonchanged case. Force a snapshot or message batch-query failure while calling `ExportResults` and assert a non-2xx response with no file-success headers/body; preserve the existing `ListResults` failure test. Reuse the current disposable fixture and cleanup; no product-source edit is needed. Append exact commands/results to the existing R004 BUILD evidence, synchronize continuity, and return one local test/evidence commit for Codex re-review.

## Boundary

This is repair round 2 under the same R2 objective/path/effect/commit-owner ceiling. `REVIEW_COST_ESCALATION_REQUIRED` does not apply yet; it is triggered at repair round three without an independent new root cause. No provider call, real channel sync, customer data, persistent database reset, deployment, push, S2/S3/S5 implementation or FREEZE is authorized.
