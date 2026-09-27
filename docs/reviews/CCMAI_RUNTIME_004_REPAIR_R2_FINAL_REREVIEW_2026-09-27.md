# Independent final re-review: CCMAI-RUNTIME-004

**Reviewer:** Codex (`REVIEWER`) · **Date:** 2026-09-27 · **Target:** local repair-round-2 commit `e8ab021` · **Disposition:** `PASS` for R2 REVIEW. S1 remains in progress; FREEZE remains open.

## Review result

- R004-R1 snapshot digest/provenance validation and R004-R2 earlier-message count handling were accepted in the previous independent re-review of commit `a074870`.
- R004-R3-T1 is now complete. `TestExportResultsCSVAndXLSXIncludeSourceIntegrityColumn` executes the real `ExportResults` handler for both CSV and XLSX with unchanged and edited source, parses each file and asserts the exact last-column status label. `TestExportResultsSnapshotBatchQueryFailureIsObservable` forces the batched snapshot read to fail for both export formats and asserts a 4xx/5xx JSON `query_failed` response, no download header and no partial CSV/XLSX prefix. The page failure test still uses the shared restore-safe fixture. The commit changes only `backend/api/handlers/results_test.go` plus authorized evidence/continuity; no production source changed.

## Independent verification

On a fresh disposable MySQL 8 `CCMA` instance, with validation-only credentials and `GOFLAGS=-mod=readonly`, Codex reran:

```text
go test ./api/handlers -run '^Test(ExportResultsCSVAndXLSXIncludeSourceIntegrityColumn|ExportResultsSnapshotBatchQueryFailureIsObservable|ListResultsSnapshotBatchQueryFailureIsObservable)$' -count=1 -v
```

The final verbose run confirmed `PASS` for both export status subtests, both export failure subtests and the page failure test. Codex verified that the `analysis_snapshots` table was restored and removed the disposable container. An initial run occurred while MySQL was still initializing and was discarded; only the run after SQL readiness is used as independent evidence. Claude's BUILD addendum reports the full backend suite, vet, catalog check and workspace doctor passing on a separate disposable MySQL instance. Codex independently reran the workspace doctor (25/25) and catalog check. No provider API, real channel sync or customer data was used; these tests are not live CVF governance proof.

## Boundary and next move

`CCMAI-RUNTIME-004` passes independent REVIEW. `CCMAI-RUNTIME-001/002/003/004` remain FREEZE open and S1 remains IN_PROGRESS. The status is a local, informational comparison on the aggregate Results page/export only; job-specific/conversation APIs, notifications, automatic re-analysis and upstream history outside adapter fetch windows remain separate scope. The next governed move is an ORCHESTRATOR decision for a bounded S1 continuation or a separate CLOSER/FREEZE evaluation. This review does not authorize provider calls, persistent database reset, deployment, push, S2/S3/S5 implementation or FREEZE.
