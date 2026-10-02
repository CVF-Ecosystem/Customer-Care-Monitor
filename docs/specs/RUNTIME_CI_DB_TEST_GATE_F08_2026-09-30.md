# CCMAI-RUNTIME-019 / F08 — Backend CI phải thực thi DB tests

Status: SPEC accepted for bounded WORK_ORDER; product BUILD pending Claude. Date: 2026-09-30. Risk: R2 because this is a release-quality test gate; the implementation touches CI only and uses disposable test data. Source finding: [F08 local review](../reviews/CCMAI_F01_F08_LOCAL_SOURCE_REVIEW_2026-09-30.md).

## Observed state

`.github/workflows/backend.yml` runs `go test ./...` and `go build ./...` without MySQL or `TEST_DB_DSN`. DB helpers such as `backend/engine/snapshot_db_test.go:95-105` call `t.Skip` when the DSN is absent or unreachable. Codex reproduced `TestSingleAndBatchShareSnapshotContract` as `SKIP` with package `PASS` under an empty DSN. `scripts/test-backend.ps1` already proves a separate local pattern: disposable MySQL 8, schema `CCMA`, application user `ccma`, `log_bin_trust_function_creators=1`, no persistent Compose DB.

## Intended contract

1. For backend-related PRs and pushes, the backend workflow provisions an ephemeral MySQL 8 test database, verifies readiness through the application user, and supplies `TEST_DB_DSN` only to the test job. It runs the complete backend Go suite with cache disabled (`-count=1`) and bounded package parallelism (`-p 1`), then builds. The workflow must also trigger if its CI gate helper changes.
2. A green workflow means the Go test command exited successfully **and** required DB-backed sentinel tests completed with `PASS`, not `SKIP` or no result. At minimum observe one named DB test from each of `db`, `api/handlers`, `engine`, `cli`, and `storagecfg`; use package-qualified test identities to avoid name collisions. The required list must be explicit and source-grounded.
3. Missing/unreachable DSN, a failed MySQL readiness check, a skipped required test, absent/truncated test output, or a failed Go test/build makes the job fail. Report the required tests and the count/reasons of skips. Optional S3 or opt-in live-fetch tests may continue to skip; the gate must not mistake those for DB coverage.
4. The service uses only disposable schema/data and a CI-only credential. No app Compose volume, real channel, Alibaba/provider API, external storage, local `.env` or production dataset is touched. The DSN/password must not be printed or uploaded in artifacts; no API key is committed.
5. Keep product/runtime behavior, DB test assertions and migration logic unchanged. CI configuration and a small test-result gate helper may change; any previously failing DB test discovered under real MySQL is reported as `BUILD_BLOCKED` for a separate scoped repair, not silently skipped or weakened.

## Acceptance evidence

- Positive: full suite and build on disposable MySQL; a machine-readable or verbose log identifies each required package/test as PASS and gives the skip summary. No DB-unavailable skip is accepted.
- Negative: prove the gate returns nonzero for a required test marked SKIP, a missing required result and a failed test; prove missing/unreachable DB stops the workflow before a green outcome. Synthetic test-output fixtures are valid for checking the gate parser; they are not a substitute for the positive DB run.
- Source check: workflow is triggered by changes to its own file, backend code and any new gate helper. `git diff --check`, catalog `-Check` and workspace doctor pass.
- Independent Codex REVIEW checks the exact diff, positive and negative evidence, secret boundaries and claim wording. A local simulation cannot establish a historical GitHub Actions run; public CI success is claimed only after an actual workflow run, under separate push authority.

## Claim boundary

This tranche fixes F08's ability for DB tests to skip unnoticed in backend CI. It does not fix F01–F07, prove real-channel completeness, or prove CVF governs AI. Alibaba key permission is recorded for later relevant tests, but F08 needs zero provider calls.
