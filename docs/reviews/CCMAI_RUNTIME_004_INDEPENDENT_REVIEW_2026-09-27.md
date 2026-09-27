# Independent REVIEW: CCMAI-RUNTIME-004 BUILD

**Reviewer:** Codex (`REVIEWER`) · **Date:** 2026-09-27 · **Target:** local commit `a9559e3` · **Disposition:** `CHANGES_REQUIRED` (first R2 review round). No FREEZE.

## Accepted portions and independent checks

- The changed set stays within the implementation/evidence paths in `CCMAI_RUNTIME_004.md`. The shared Results `fetchRows` path now supplies the same status to page and export; queries batch by tenant, and the UI has Vietnamese/English labels in table, card and detail. The snapshot canonicalization extraction appears digest-neutral. No provider call or governance-runtime claim was made.
- Fresh disposable MySQL 8 `CCMA`, validation-only credentials, `GOFLAGS=-mod=readonly`: `go test ./engine ./api/handlers -run 'TestCompareSnapshot|TestSourceIntegrity' -count=1` PASS. After setting `log_bin_trust_function_creators=1` for pre-existing trigger tests, `go test ./... -count=1 -p 1` PASS across all tested backend packages. The initial full-suite attempt failed only in existing trigger tests because that disposable MySQL setting was absent; it passed after the setting was applied. The database container was removed afterward. Workspace doctor PASS 25/25. These tests do not resolve the findings below.

## Blocking findings

### R004-R1 — a linked but invalid snapshot can be presented as locally matching

`attachSourceIntegrity` loads snapshots by tenant/ID but never checks that `AnalysisSnapshot.ConversationID` and `JobRunID` match the result, that the manifest's tenant/conversation match its row, that `MessageCount` matches the manifest, or that SHA-256 of the stored manifest bytes equals `AnalysisSnapshot.Digest` (`backend/api/handlers/results.go`, lines 310–366). `CompareSnapshotToCurrentMessages` only parses JSON and checks the schema string (`backend/engine/snapshot.go`, lines 278–289). The new DB fixture actually stores digest `deadbeef` and the unchanged test still expects `bound_currentness_unverified` (`results_test.go`, fixture around line 390 and test around line 417). Thus a tampered or mislinked record can be reported as locally matching even though the stored evidence itself fails its integrity/provenance contract. This violates SPEC contract 2/3's stored-manifest validation and the tenant-boundary acceptance.

**Repair acceptance:** Before comparison, validate the stored snapshot digest against its exact manifest bytes and check tenant, conversation, run, schema and message-count consistency against the result/manifest. Return `verification_unavailable` for a row-local mismatch; never return the locally matching status. Use a real digest in valid fixtures. Add regressions for corrupt digest and wrong conversation/run/tenant provenance, including a cross-tenant link, while preserving the batched tenant-scoped query and no-write behavior.

### R004-R2 — newly backfilled messages can be silently skipped

The comparison drops every current message with `sent_at` before the earliest message in the stored manifest (`backend/engine/snapshot.go`, lines 302–308), regardless of `omitted_earlier_messages`. Counterexample: a snapshot recorded one message at 10:00 with `omitted_earlier_messages=0`; after analysis, a second source message is inserted at 09:00. It changes the observed conversation set, but the function skips it and returns `bound_currentness_unverified`. The original analysis cutoff could be before 09:00, so this is not necessarily outside the analyzed window. The existing test exercises an artificial `omitted_earlier_messages=3` case only. This violates SPEC contract 2 and the added-message acceptance.

**Repair acceptance:** Detect an added earlier message when the snapshot omitted no earlier history. For windowed snapshots, compare the recorded omitted count with the current earlier-message count and flag a proven count change; if counts match but individual earlier messages cannot be identified from the stored manifest, retain `bound_currentness_unverified` with that explicit limitation. Add focused tests for an added message before the first manifest timestamp with zero and nonzero omitted counts. Preserve the no-positive-freshness claim.

### R004-R3 — required API/export and isolation evidence is incomplete

The work order explicitly requires a feature-specific tenant-isolation test, CSV/XLSX export equivalence, and page/export error handling. The new DB tests call `fetchRows` and `sourceIntegrityLabel` only (`backend/api/handlers/results_test.go`, lines 417–550). They do not execute `ListResults`, `ExportResults`, either file writer, a cross-tenant linked snapshot case, or a batch-query failure. BUILD evidence calls these omissions out for tenant isolation but still claims BUILD complete. Passing helper tests cannot prove the endpoint/export acceptance.

**Repair acceptance:** Add disposable-MySQL or handler tests that exercise the actual page response and both CSV/XLSX exports for a changed and a nonchanged row, prove tenant isolation for the newly joined snapshot/message data, and force a snapshot/message batch-query error to verify page/export return an observable failure rather than a partial success. Keep existing filters, count, order and export cap tests passing. If a specific error injection cannot fit the allowed test path, document the exact blocker before claiming completion.

## Boundary and next move

These are same-objective, same R2/path/effect findings, so one bounded repair round may proceed under the existing authority. Claude returns as `REPAIR_WORKER` after continuity rehydration and handoff acknowledgment, edits only `backend/engine/snapshot.go`, focused engine tests, `backend/api/handlers/results.go` and `results_test.go` plus existing BUILD evidence/continuity. No DB model/migration or frontend change is required by these findings. Run focused/full backend tests on disposable MySQL, frontend build only if frontend changes, catalog check, workspace doctor and `git diff --check`; return one local commit without push for Codex re-review. No provider, real channel sync, customer data, deployment, S2/S3/S5 or FREEZE authority is added.
