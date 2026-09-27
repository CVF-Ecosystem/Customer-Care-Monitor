# Independent REVIEW: CCMAI-RUNTIME-005 confidence truth

**Reviewer:** Codex (`REVIEWER`) · **Date:** 2026-09-27 · **Target:** local BUILD commit `9a6ccc1` · **Disposition:** `PASS` for R2 REVIEW. S1 remains IN_PROGRESS; FREEZE remains open.

## Source and contract assessment

- The approved scope extension is recorded in the active handoff after Claude's `BUILD_BLOCKED`: four demo writer literals, a compile-only Results fixture, and a focused demo test. The commit stays within the original work order plus that extension. No provider, prompt, adapter, sync, snapshot or aggregate Results production source was changed.
- `JobResult` persists nullable `confidence` and `confidence_basis` separately from the JSON fields. `AfterFind` exposes a number only for a classification tag with basis `model_reported_uncalibrated` and value in `[0,1]`. Other rows, including legacy rows with a NULL basis, serialize `null` / `unavailable`; their stored numbers remain intact after `Save`.
- Analyzer QC and evaluation writers now store NULL/unavailable, while validated classification tags retain a labeled model estimate. Demo rows store no invented estimate. Notifications omit unknown numbers and label the one model estimate as uncalibrated. The frontend type accepts `number | null`.
- The BUILD evidence documents fresh and existing-table `AutoMigrate` twice, preserved legacy rows, and the rollback limit: an old binary reads newly NULL confidence as `0`. This limit does not contradict the reviewed forward contract; rollback to that binary would require an explicit compatibility decision.

## Independent verification

Codex ran the focused R005 engine, handler and notification tests against a newly created disposable `mysql:8.0` `CCMA` instance from a read-only source mount in `golang:1.26-alpine`, with `GOPROXY=off`, `GOFLAGS=-mod=readonly` and an isolated Docker network. The command selected `TestQCResultsStoreNoFabricatedConfidence`, classification tag/SKIP/invalid-confidence cases, legacy/mislabeled rows, both actual job-result API handlers, demo import and notification formatting. All selected tests passed; the two API subtests both executed and passed. SQL readiness was confirmed before the run. The disposable MySQL container and network were removed; persistent Compose was untouched.

Claude's separate BUILD evidence records all 13 backend packages passing, frontend build, go build/vet, the migration/rollback probe and mutation regression. Codex inspected those claims against the changed source and test assertions. Workspace doctor passed 25/25 and the catalog check passed independently. No provider API was called; this is persistence/API honesty evidence, not live CVF governance or calibrated model quality proof.

## Boundary and next move

`CCMAI-RUNTIME-005` passes independent REVIEW. R001–R005 FREEZE decisions and S1 closure remain open. The next governed move is an ORCHESTRATOR decision for a bounded S1 continuation or a separate CLOSER/FREEZE evaluation. No provider call, real channel sync, customer data, persistent database reset, deployment, push, S2/S3/S5 implementation or FREEZE follows from this review.
